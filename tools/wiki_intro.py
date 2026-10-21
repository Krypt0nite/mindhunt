#!/usr/bin/env python3
"""Fetch the lead paragraph and short description of Wikipedia articles, in
batches, cached under data/cache/wiki/ so evidence can be re-read offline.

  python3 tools/wiki_intro.py "Ashoka" "Akbar" ...      (prints JSON)

Used while checking packs; the cache records what each article said on the
day it was read (with the revision id).
"""
import json
import os
import pathlib
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parent.parent
CACHE = ROOT / "data" / "cache" / "wiki"
API = "https://en.wikipedia.org/w/api.php"
UA = "MindHuntFactCheck/1.0 (educational quiz game; low-volume fact check)"


def _get(params):
    url = API + "?" + urllib.parse.urlencode({**params, "format": "json", "formatversion": "2"})
    for attempt in range(8):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": UA})
            with urllib.request.urlopen(req, timeout=30) as r:
                return json.loads(r.read())
        except urllib.error.HTTPError as e:
            if e.code == 429:
                wait = int(e.headers.get("Retry-After", "30") or 30)
                time.sleep(min(max(wait, 5), 120) + attempt * 5)
                continue
            raise
        except (urllib.error.URLError, TimeoutError):
            time.sleep(2 ** attempt)
    raise IOError("Wikipedia API kept refusing")


def infobox(wikitext):
    """Top-level 'key = value' lines of the first infobox, lightly cleaned."""
    import re
    out = {}
    start = wikitext.find("{{Infobox")
    if start < 0:
        start = wikitext.find("{{infobox")
    if start < 0:
        return out
    depth, i = 0, start
    while i < len(wikitext):
        if wikitext.startswith("{{", i):
            depth += 1
            i += 2
        elif wikitext.startswith("}}", i):
            depth -= 1
            i += 2
            if depth == 0:
                break
        else:
            i += 1
    body = wikitext[start:i]
    for m in re.finditer(r"\n\s*\|\s*([a-z0-9_ ]+?)\s*=\s*(.*)", body):
        v = re.sub(r"<ref[^>]*/>|<ref[^>]*>.*?</ref>|<!--.*?-->", "", m.group(2))
        v = re.sub(r"\[\[(?:[^|\]]*\|)?([^\]]*)\]\]", r"\1", v)
        out.setdefault(m.group(1).strip(), v.strip()[:300])
    return out


def intros(titles):
    CACHE.mkdir(parents=True, exist_ok=True)
    out, todo = {}, []
    for t in titles:
        f = CACHE / (urllib.parse.quote(t.replace(" ", "_"), safe="") + ".json")
        if f.exists():
            out[t] = json.loads(f.read_text())
        else:
            todo.append(t)
    for i in range(0, len(todo), 20):
        batch = todo[i:i + 20]
        d = _get({"action": "query", "titles": "|".join(batch), "redirects": 1, "prop": "extracts|description|revisions|pageprops",
                  "exintro": 1, "explaintext": 1, "exlimit": 20, "rvprop": "ids|timestamp|content", "rvslots": "main",
                  "rvsection": 0, "ppprop": "disambiguation"})
        q = d.get("query", {})
        back = {}
        for n in q.get("normalized", []) + q.get("redirects", []):
            back[n["to"]] = back.get(n["from"], n["from"])
        for p in q.get("pages", []):
            asked = back.get(p["title"], p["title"])
            while asked in back:
                asked = back[asked]
            rec = {"asked": asked, "title": p["title"], "missing": bool(p.get("missing")),
                   "disambiguation": "disambiguation" in p.get("pageprops", {}),
                   "description": p.get("description", ""), "extract": p.get("extract", ""),
                   "revid": (p.get("revisions") or [{}])[0].get("revid"),
                   "infobox": infobox(((p.get("revisions") or [{}])[0].get("slots", {}).get("main", {}) or {}).get("content", "")),
                   "url": "https://en.wikipedia.org/wiki/" + urllib.parse.quote(p["title"].replace(" ", "_")),
                   "fetched": time.strftime("%Y-%m-%d")}
            out[asked] = rec
            (CACHE / (urllib.parse.quote(asked.replace(" ", "_"), safe="") + ".json")).write_text(json.dumps(rec, ensure_ascii=False, indent=1))
        time.sleep(float(os.environ.get("FETCH_DELAY", "2")))
    return out


if __name__ == "__main__":
    print(json.dumps(intros(sys.argv[1:]), ensure_ascii=False, indent=1))
