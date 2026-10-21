#!/usr/bin/env python3
"""Find, check and download licensed portraits for a pack, from Wikimedia Commons.

Run on a machine that can reach en.wikipedia.org, commons.wikimedia.org and
upload.wikimedia.org (the build sandbox could not). No API key is needed.

  # 1. find candidates, check licences, download and decode into assets/
  MINDHUNT_CONTACT=you@example.com python3 tools/fetch_portraits.py fetch
  # 2. look at every downloaded picture (assets/cricket-men-intl/*.webp), then
  #    confirm the ones that really show the named player:
  python3 tools/fetch_portraits.py confirm c_44c51f c_12ae0c ...
  # 3. rebuild the pack; only confirmed entries are attached
  python3 tools/gen_cricket_pack.py > server/packs/cricket-men-intl.v3.json
  python3 tools/verify_cricket.py

Rules (from the cricket brief):
  * A picture is only "Verified" when its Commons file page answered, the
    licence is on the allow-list, the author is known, the file decoded, and a
    person confirmed the identity. Any failure leaves it Missing/Rejected/
    Needs review with the reason. Nothing is marked verified by default.
  * Leads come from tools/portraits/cricket-candidates.json (file titles you
    or earlier research suggested) and the player's Wikipedia lead image. A
    lead image hosted only on Wikipedia (fair use) is rejected.
  * One image may not be used for two players.

Writes tools/portraits/cricket-manifest.json (the attribution manifest).
"""
import argparse
import html
import io
import json
import os
import pathlib
import re
import runpy
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parent.parent
PACK_ID = "cricket-men-intl"
# The newest version of the pack.
PACK = max((ROOT / "server" / "packs").glob(f"{PACK_ID}.v*.json"), key=lambda p: int(p.stem.rsplit(".v", 1)[1]))
DIR = ROOT / "tools" / "portraits"
CANDIDATES = DIR / "cricket-candidates.json"
MANIFEST = DIR / "cricket-manifest.json"
ASSETS = ROOT / "assets" / PACK_ID
EVIDENCE = ROOT / "tools" / "cricket_evidence.py"

WIKI_API = "https://en.wikipedia.org/w/api.php"
COMMONS_API = "https://commons.wikimedia.org/w/api.php"
UA = "MindHuntPortraitFetcher/1.0 (educational quiz game; contact: {})"
MIN_SIDE = 240
OUT_SIDE = 512

# Licences that allow reuse in a commercial or non-commercial game, with
# attribution where required. NC, ND and anything unrecognised are refused.
ALLOWED = [
    (re.compile(r"^cc0( 1\.0)?$", re.I), False),
    (re.compile(r"^public domain$|^pd\b|^pd-", re.I), False),
    (re.compile(r"^cc by(-sa)? (1\.0|2\.0|2\.5|3\.0|4\.0)( [a-z]{2,})?$", re.I), True),
    # Government Open Data License - India: free reuse with attribution (accepted on Commons).
    (re.compile(r"^godl-india$", re.I), True),
]
LICENCE_URLS = {"godl-india": "https://www.data.gov.in/Godl"}
REFUSED = re.compile(r"\b(nc|nd|non-?commercial|no ?deriv|fair use|non-free|all rights reserved)\b", re.I)


class Net:
    """Small HTTP client: descriptive User-Agent, timeouts, retries, pacing."""

    def __init__(self, contact, delay=float(os.environ.get("FETCH_DELAY", "1.5")), retries=6, timeout=20):
        self.ua = UA.format(contact)
        self.delay, self.retries, self.timeout = delay, retries, timeout
        self.last = 0.0

    def get(self, url, params=None):
        if params:
            url += "?" + urllib.parse.urlencode(params)
        err = None
        for attempt in range(self.retries):
            wait = self.delay - (time.monotonic() - self.last)
            if wait > 0:
                time.sleep(wait)
            self.last = time.monotonic()
            try:
                req = urllib.request.Request(url, headers={"User-Agent": self.ua})
                with urllib.request.urlopen(req, timeout=self.timeout) as r:
                    return r.read()
            except urllib.error.HTTPError as e:
                err = f"HTTP {e.code}"
                if e.code not in (429, 500, 502, 503, 504):
                    break
                if e.code == 429:  # Wikimedia asks us to wait: do exactly that
                    try:
                        wait = int(e.headers.get("Retry-After", "30"))
                    except ValueError:
                        wait = 30
                    time.sleep(min(max(wait, 5), 120) + attempt * 5)
                    continue
            except (urllib.error.URLError, TimeoutError, OSError) as e:
                err = f"network: {getattr(e, 'reason', e)}"
            time.sleep(2 ** attempt)
        raise IOError(f"{url.split('?')[0]}: {err}")

    def json(self, url, params):
        return json.loads(self.get(url, {**params, "format": "json", "formatversion": "2"}))


def strip_html(s):
    return re.sub(r"\s+", " ", html.unescape(re.sub(r"<[^>]+>", "", s or ""))).strip()


def classify_licence(meta):
    """(ok, licence, licence_url, attribution_required, reason) from Commons extmetadata."""
    v = lambda k: strip_html((meta.get(k) or {}).get("value", ""))
    name = v("LicenseShortName") or v("License")
    url = (meta.get("LicenseUrl") or {}).get("value", "").strip()
    if not name:
        return False, "", url, True, "no licence on the file page"
    if REFUSED.search(name) or REFUSED.search(v("UsageTerms")):
        return False, name, url, True, f"licence {name!r} does not allow this use"
    if v("NonFree").lower() in ("true", "1"):
        return False, name, url, True, "file is marked non-free"
    for rx, attribution in ALLOWED:
        if rx.search(name):
            url = url or LICENCE_URLS.get(name.lower(), "")
            if attribution and not url:
                return False, name, url, True, "licence URL missing"
            return True, name, url, attribution, ""
    return False, name, url, True, f"licence {name!r} is not on the allow-list"


def wiki_lead_image(net, title):
    """File title of the article's lead image, or (None, reason)."""
    d = net.json(WIKI_API, {"action": "query", "titles": title.replace("_", " "), "redirects": 1,
                             "prop": "pageimages|pageprops", "piprop": "name", "ppprop": "disambiguation"})
    pages = d.get("query", {}).get("pages", [])
    if not pages or pages[0].get("missing"):
        return None, "Wikipedia article not found"
    p = pages[0]
    if "disambiguation" in p.get("pageprops", {}):
        return None, "Wikipedia title is a disambiguation page"
    if not p.get("pageimage"):
        return None, "article has no lead image"
    return "File:" + p["pageimage"], ""


def commons_search(net, name, limit=4):
    """Commons file search for '<name> cricket'; only files whose title names the player."""
    d = net.json(COMMONS_API, {"action": "query", "list": "search", "srnamespace": 6, "srlimit": limit * 3,
                                "srsearch": f'"{name}" cricket'})
    surname = name.split()[-1].lower()
    out = []
    for r in d.get("query", {}).get("search", []):
        t = r["title"]
        if surname in t.lower() and re.search(r"\.(jpe?g|png|webp)$", t, re.I):
            out.append(t)
    return out[:limit]


def commons_info(net, file_title):
    d = net.json(COMMONS_API, {"action": "query", "titles": file_title, "redirects": 1, "prop": "imageinfo",
                                "iiprop": "url|extmetadata|mime|size|sha1", "iiurlwidth": OUT_SIDE * 2})
    pages = d.get("query", {}).get("pages", [])
    if not pages:
        return None, "no response for file"
    p = pages[0]
    if p.get("missing") or p.get("invalid"):
        return None, "not on Wikimedia Commons (may be a local, non-free Wikipedia file)"
    ii = p.get("imageinfo") or []
    if not ii:
        return None, "Commons returned no imageinfo"
    info = ii[0]
    info["title"] = p.get("title", file_title)
    return info, ""


def decode_and_save(data, dest):
    from PIL import Image, ImageOps

    im = Image.open(io.BytesIO(data))
    im.load()
    im = ImageOps.exif_transpose(im).convert("RGB")
    if min(im.size) < MIN_SIDE:
        raise ValueError(f"too small ({im.size[0]}x{im.size[1]})")
    im.thumbnail((OUT_SIDE, OUT_SIDE * 4 // 3))
    dest.parent.mkdir(parents=True, exist_ok=True)
    im.save(dest, "WEBP", quality=82, method=6)
    Image.open(dest).verify()  # what we wrote must decode too
    return im.size


def blank_entry(c, wiki):
    return {
        "id": c["id"], "name": c["name"], "wikipedia": wiki,
        "image_status": "Missing", "reason": "not searched yet",
        "file_page": "", "file_url": "", "thumb_url": "", "asset": "", "sha1": "",
        "credit": "", "license": "", "license_url": "", "attribution_required": None,
        "attribution_line": "", "source_url": "",
        "licence_status": "Unverified", "identity_confirmed": False, "download_status": "Not downloaded",
        "candidates_examined": [], "checked": "",
    }


def load_manifest(pack):
    ev = runpy.run_path(str(EVIDENCE))["P"]
    old = {e["id"]: e for e in json.loads(MANIFEST.read_text())} if MANIFEST.exists() else {}
    return [old.get(c["id"]) or blank_entry(c, ev[c["name"]]["wiki"]) for c in pack["candidates"]]


def fetch(args):
    contact = os.environ.get("MINDHUNT_CONTACT", "").strip()
    if not contact:
        sys.exit("Set MINDHUNT_CONTACT to an email or URL for the User-Agent (Wikimedia policy).")
    net = Net(contact)
    pack = json.loads(PACK.read_text())
    leads = json.loads(CANDIDATES.read_text()) if CANDIDATES.exists() else {}
    manifest = load_manifest(pack)
    used = {e["sha1"]: e["id"] for e in manifest if e["sha1"] and e["image_status"] in ("Verified", "Needs review")}
    for e in manifest:
        if args.only and e["id"] not in args.only:
            continue
        if e["image_status"] == "Verified" and not args.recheck:
            continue
        e.update(blank_entry({"id": e["id"], "name": e["name"]}, e["wikipedia"]))
        e["checked"] = time.strftime("%Y-%m-%d")
        titles = list(leads.get(e["id"], {}).get("files", []))
        try:
            lead, why = wiki_lead_image(net, e["wikipedia"])
            if lead:
                titles.append(lead)
            else:
                e["candidates_examined"].append({"file": "(Wikipedia lead image)", "result": why})
        except IOError as ex:
            e["candidates_examined"].append({"file": "(Wikipedia lead image)", "result": f"request failed: {ex}"})
        if args.search:
            try:
                titles += commons_search(net, e["name"])
            except IOError as ex:
                e["candidates_examined"].append({"file": "(Commons search)", "result": f"request failed: {ex}"})
        for t in dict.fromkeys(titles):
            rec = {"file": t, "result": ""}
            e["candidates_examined"].append(rec)
            try:
                info, why = commons_info(net, t)
            except IOError as ex:
                rec["result"] = f"file page unreachable: {ex}"
                continue
            if not info:
                rec["result"] = why
                continue
            meta = info.get("extmetadata", {})
            ok, lic, lic_url, attr_req, why = classify_licence(meta)
            author = strip_html((meta.get("Artist") or {}).get("value", ""))
            if not ok:
                rec["result"] = "rejected: " + why
                continue
            if attr_req and not author:
                rec["result"] = "rejected: author missing"
                continue
            if info.get("sha1") in used and used[info["sha1"]] != e["id"]:
                rec["result"] = f"rejected: same file already used for {used[info['sha1']]}"
                continue
            if not str(info.get("mime", "")).startswith("image/"):
                rec["result"] = f"rejected: not a photo ({info.get('mime')})"
                continue
            thumb = info.get("thumburl") or info.get("url")
            dest = ASSETS / f"{e['id']}.webp"
            size = None
            if not args.no_download:
                try:
                    size = decode_and_save(net.get(thumb), dest)
                except (IOError, ValueError, OSError) as ex:
                    rec["result"] = f"rejected: download/decode failed: {ex}"
                    continue
            page = info.get("descriptionurl") or "https://commons.wikimedia.org/wiki/" + urllib.parse.quote(info["title"].replace(" ", "_"))
            e.update({
                "image_status": "Needs review", "reason": "licence verified on Commons; identity needs a human check",
                "file_page": page, "file_url": info.get("url", ""), "thumb_url": thumb, "sha1": info.get("sha1", ""),
                "asset": f"{PACK_ID}/{e['id']}.webp" if size else "", "credit": author or "Unknown author (public domain)",
                "license": lic, "license_url": lic_url, "attribution_required": attr_req,
                "attribution_line": f"Photo: {author or 'unknown author'}, {lic}" + (f" ({lic_url})" if lic_url else "") + f", via Wikimedia Commons ({page}). Resized.",
                "source_url": page, "licence_status": "Verified",
                "download_status": f"Decoded and saved {size[0]}x{size[1]} WebP" if size else "Not downloaded yet (run: download)",
            })
            rec["result"] = "selected"
            used[e["sha1"]] = e["id"]
            break
        else:
            if e["image_status"] == "Missing":
                e["reason"] = "no usable candidate: " + "; ".join(f"{r['file']}: {r['result']}" for r in e["candidates_examined"]) if e["candidates_examined"] else "no candidates found"
                if any(r["result"].startswith("rejected") for r in e["candidates_examined"]):
                    e["image_status"] = "Rejected"
        print(f"{e['name']:<24} {e['image_status']:<13} {e['reason'][:90]}", flush=True)
        save(manifest, quiet=True)  # keep progress if the run is interrupted
    save(manifest)


def download(args):
    """Download and decode every licence-checked photo (run where Wikimedia images are reachable)."""
    contact = os.environ.get("MINDHUNT_CONTACT", "").strip()
    if not contact:
        sys.exit("Set MINDHUNT_CONTACT to an email or URL for the User-Agent (Wikimedia policy).")
    net = Net(contact)
    manifest = json.loads(MANIFEST.read_text())
    for e in manifest:
        if e["licence_status"] != "Verified" or not e.get("thumb_url"):
            continue
        if e.get("asset") and (ASSETS / f"{e['id']}.webp").exists() and not args.again:
            continue
        try:
            size = decode_and_save(net.get(e["thumb_url"]), ASSETS / f"{e['id']}.webp")
        except (IOError, ValueError, OSError) as ex:
            e["download_status"] = f"Failed: {ex}"
            print(f"{e['name']:<24} FAILED {ex}", flush=True)
            continue
        e["asset"] = f"{PACK_ID}/{e['id']}.webp"
        e["download_status"] = f"Decoded and saved {size[0]}x{size[1]} WebP"
        if e.get("identity_confirmed"):
            e["image_status"] = "Verified"
            e["reason"] = "licence checked on Commons, identity confirmed by a person, file decoded"
        print(f"{e['name']:<24} saved {size[0]}x{size[1]}", flush=True)
        save(manifest, quiet=True)
    save(manifest)


REVIEW = ROOT / "data" / "cricket" / "portrait-review.html"


def review(args):
    """An HTML page to check each photo shows the right player (loads images from Wikimedia)."""
    manifest = [e for e in json.loads(MANIFEST.read_text()) if e["licence_status"] == "Verified"]
    esc = html.escape
    cards = []
    for e in manifest:
        cards.append(f"""<label class="card"><input type="checkbox" value="{esc(e['id'])}" {'checked' if e.get('identity_confirmed') else ''}>
<img src="{esc(e['thumb_url'])}" alt="Candidate photo for {esc(e['name'])}" loading="lazy">
<span class="name">{esc(e['name'])}</span>
<span class="meta">{esc(e['license'])} · {esc(e['credit'][:60])}</span>
<a href="{esc(e['file_page'])}" target="_blank" rel="noreferrer">File page</a></label>""")
    REVIEW.write_text(f"""<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>Portrait review</title>
<style>
:root{{--bg:#eef1f4;--card:#f8fafb;--ink:#1b2a4a;--soft:#4a5873;--rule:#b9c4d3;--yes:#1f7a4d}}
@media (prefers-color-scheme:dark){{:root{{--bg:#11161f;--card:#1a2130;--ink:#e8edf5;--soft:#9aa6ba;--rule:#334055;--yes:#4fbf86}}}}
body{{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 system-ui,sans-serif;padding:16px}}
h1{{margin:.2em 0}} p{{color:var(--soft);max-width:70ch}}
.grid{{display:grid;grid-template-columns:repeat(auto-fill,minmax(170px,1fr));gap:12px;margin:16px 0 96px}}
.card{{background:var(--card);border:2px solid var(--rule);border-radius:12px;padding:8px;display:flex;flex-direction:column;gap:4px;cursor:pointer}}
.card:has(input:checked){{border-color:var(--yes)}}
.card img{{width:100%;aspect-ratio:3/4;object-fit:cover;object-position:top;border-radius:8px;background:var(--rule)}}
.name{{font-weight:700}} .meta{{font-size:12px;color:var(--soft)}} .card a{{font-size:12px;color:var(--soft)}}
.bar{{position:fixed;left:0;right:0;bottom:0;background:var(--card);border-top:1px solid var(--rule);padding:12px 16px;display:flex;gap:12px;align-items:center;flex-wrap:wrap}}
button{{font:inherit;font-weight:600;padding:10px 18px;border-radius:999px;border:0;background:var(--ink);color:var(--bg);cursor:pointer}}
code{{word-break:break-all;font-size:12px;color:var(--soft)}}
</style></head><body>
<h1>Does each photo show the named player?</h1>
<p>Tick the photos that clearly show the right person on their own (not with someone else, not the wrong player).
Leave the rest unticked; those players keep their initials. Licences were already checked. Then copy the command at the bottom.</p>
<div class="grid">{''.join(cards)}</div>
<div class="bar"><button id="copy">Copy confirm command</button><span id="n"></span><code id="cmd"></code></div>
<script>
const boxes=[...document.querySelectorAll('input[type=checkbox]')];
function cmd(){{const ids=boxes.filter(b=>b.checked).map(b=>b.value);
document.getElementById('n').textContent=ids.length+' of '+boxes.length+' ticked';
return 'python3 tools/fetch_portraits.py confirm '+ids.join(' ');}}
boxes.forEach(b=>b.addEventListener('change',()=>document.getElementById('cmd').textContent=cmd()));
document.getElementById('cmd').textContent=cmd();
document.getElementById('copy').onclick=async()=>{{const c=cmd();try{{await navigator.clipboard.writeText(c)}}catch(e){{prompt('Copy this',c)}}}};
</script></body></html>""")
    print(f"wrote {REVIEW} ({len(manifest)} photos)")


def confirm(args):
    manifest = json.loads(MANIFEST.read_text())
    by = {e["id"]: e for e in manifest}
    for cid in args.ids:
        e = by.get(cid)
        if not e:
            sys.exit(f"unknown id {cid}")
        problems = []
        if e["licence_status"] != "Verified":
            problems.append("licence not verified")
        if not e["credit"] or not e["file_page"]:
            problems.append("attribution incomplete")
        if problems:
            sys.exit(f"{cid} ({e['name']}): cannot confirm: {', '.join(problems)}")
        e["identity_confirmed"] = True
        if e.get("asset") and (ASSETS / f"{cid}.webp").exists():
            e["image_status"] = "Verified"
            e["reason"] = f"identity confirmed by a person on {time.strftime('%Y-%m-%d')}; file decoded"
        else:
            e["reason"] = f"identity confirmed by a person on {time.strftime('%Y-%m-%d')}; run download to fetch the file"
        print(f"confirmed {e['name']}")
    save(manifest)


def reject(args):
    manifest = json.loads(MANIFEST.read_text())
    for e in manifest:
        if e["id"] in args.ids:
            e.update({"image_status": "Rejected", "identity_confirmed": False, "asset": "",
                      "reason": "rejected by reviewer: " + args.why})
            (ASSETS / f"{e['id']}.webp").unlink(missing_ok=True)
    save(manifest)


def save(manifest, quiet=False):
    MANIFEST.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n")
    if quiet:
        return
    counts = {}
    for e in manifest:
        counts[e["image_status"]] = counts.get(e["image_status"], 0) + 1
    print("manifest:", ", ".join(f"{k} {v}" for k, v in sorted(counts.items())))


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    f = sub.add_parser("fetch")
    f.add_argument("--only", nargs="*", help="limit to these candidate ids")
    f.add_argument("--recheck", action="store_true", help="re-check entries already verified")
    f.add_argument("--no-download", action="store_true", help="check licences only; download later")
    f.add_argument("--search", action="store_true", help="also try Commons search results that name the player")
    f.set_defaults(fn=fetch)
    d = sub.add_parser("download")
    d.add_argument("--again", action="store_true", help="re-download files already saved")
    d.set_defaults(fn=download)
    v = sub.add_parser("review")
    v.set_defaults(fn=review)
    c = sub.add_parser("confirm")
    c.add_argument("ids", nargs="+")
    c.set_defaults(fn=confirm)
    r = sub.add_parser("reject")
    r.add_argument("ids", nargs="+")
    r.add_argument("--why", required=True)
    r.set_defaults(fn=reject)
    a = ap.parse_args()
    a.fn(a)
