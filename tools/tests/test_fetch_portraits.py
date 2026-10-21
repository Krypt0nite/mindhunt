"""Offline tests for tools/fetch_portraits.py (no network; a fake client).

Run: python3 -m unittest discover -s tools/tests
"""
import importlib.util
import io
import json
import pathlib
import tempfile
import unittest
from types import SimpleNamespace

ROOT = pathlib.Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("fetch_portraits", ROOT / "tools" / "fetch_portraits.py")
fp = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fp)


def meta(**kw):
    return {k: {"value": v} for k, v in kw.items()}


def jpeg(w=600, h=800):
    from PIL import Image

    b = io.BytesIO()
    Image.new("RGB", (w, h), (120, 90, 60)).save(b, "JPEG")
    return b.getvalue()


class FakeNet:
    """Answers like the Wikipedia/Commons APIs for a scripted set of files."""

    def __init__(self, files, lead=None, down=False):
        self.files, self.lead, self.down = files, lead or {}, down

    def json(self, url, params):
        if self.down:
            raise IOError("network: blocked")
        if url == fp.WIKI_API:
            img = self.lead.get(params["titles"])
            return {"query": {"pages": [{"title": params["titles"], **({"pageimage": img} if img else {})}]}}
        f = self.files.get(params["titles"])
        if f is None:
            return {"query": {"pages": [{"title": params["titles"], "missing": True}]}}
        return {"query": {"pages": [{"title": params["titles"], "imageinfo": [f]}]}}

    def get(self, url, params=None):
        if url.endswith("broken.jpg"):
            return b"not an image"
        return jpeg()


def info(name, lic="CC BY-SA 4.0", url="https://creativecommons.org/licenses/by-sa/4.0", artist="<a>Jane Doe</a>", sha1=None, mime="image/jpeg"):
    return {"url": f"https://upload.wikimedia.org/{name}", "thumburl": f"https://upload.wikimedia.org/thumb/{name}",
            "descriptionurl": f"https://commons.wikimedia.org/wiki/File:{name}", "mime": mime, "sha1": sha1 or name,
            "extmetadata": meta(LicenseShortName=lic, LicenseUrl=url, Artist=artist)}


class LicenceTests(unittest.TestCase):
    def test_allowed(self):
        for lic, url in [("CC BY 2.0", "https://creativecommons.org/licenses/by/2.0"),
                         ("CC BY-SA 3.0", "https://creativecommons.org/licenses/by-sa/3.0"),
                         ("CC0", ""), ("Public domain", ""), ("GODL-India", "")]:
            ok, *_ = fp.classify_licence(meta(LicenseShortName=lic, LicenseUrl=url))
            self.assertTrue(ok, lic)

    def test_refused(self):
        for m in [meta(LicenseShortName="CC BY-NC 2.0", LicenseUrl="x"),
                  meta(LicenseShortName="CC BY-ND 4.0", LicenseUrl="x"),
                  meta(LicenseShortName="Fair use"),
                  meta(LicenseShortName=""),
                  meta(LicenseShortName="CC BY 4.0"),  # attribution licence without its URL
                  meta(LicenseShortName="Copyrighted free use"),  # not on the allow-list
                  meta(LicenseShortName="CC BY-SA 4.0", LicenseUrl="x", NonFree="true"),
                  meta(LicenseShortName="GODL")]:
            ok, *_ = fp.classify_licence(m)
            self.assertFalse(ok, m)


class FetchTests(unittest.TestCase):
    def setUp(self):
        self.tmp = pathlib.Path(tempfile.mkdtemp())
        pack = {"candidates": [{"id": "c_000001", "name": "Player One"}, {"id": "c_000002", "name": "Player Two"},
                               {"id": "c_000003", "name": "Player Three"}, {"id": "c_000004", "name": "Player Four"}]}
        (self.tmp / "pack.json").write_text(json.dumps(pack))
        ev = self.tmp / "ev.py"
        ev.write_text("P = {" + ",".join(f"'{c['name']}': {{'wiki': '{c['name'].replace(' ', '_')}'}}" for c in pack["candidates"]) + "}")
        self.cands = self.tmp / "cands.json"
        self.cands.write_text(json.dumps({"c_000002": {"files": ["File:Nc.jpg", "File:Good2.jpg"]},
                                          "c_000003": {"files": ["File:Good1.jpg"]}}))
        fp.REVIEW = self.tmp / "review.html"
        for k, v in dict(PACK=self.tmp / "pack.json", MANIFEST=self.tmp / "manifest.json", ASSETS=self.tmp / "assets",
                         CANDIDATES=self.cands, EVIDENCE=ev).items():
            setattr(fp, k, v)
        fp.os.environ["MINDHUNT_CONTACT"] = "test@example.org"

    def run_fetch(self, net, no_download=False):
        orig = fp.Net
        fp.Net = lambda contact: net
        try:
            fp.fetch(SimpleNamespace(only=None, recheck=False, no_download=no_download, search=False))
        finally:
            fp.Net = orig
        return {e["id"]: e for e in json.loads(fp.MANIFEST.read_text())}

    def test_pipeline_never_verifies_without_a_person(self):
        net = FakeNet({"File:Good1.jpg": info("Good1.jpg"),
                       "File:Nc.jpg": info("Nc.jpg", lic="CC BY-NC 2.0"),
                       "File:Good2.jpg": info("Good2.jpg", lic="CC BY 2.0", url="https://creativecommons.org/licenses/by/2.0"),
                       "File:Dup.jpg": info("Dup.jpg", sha1="Good1.jpg"),
                       "File:Local.jpg": None},
                      lead={"Player One": "Local.jpg", "Player Four": "Dup.jpg"})
        m = self.run_fetch(net)
        self.assertEqual(m["c_000001"]["image_status"], "Missing")  # lead image not on Commons
        self.assertIn("not on Wikimedia Commons", m["c_000001"]["reason"])
        two = m["c_000002"]
        self.assertEqual(two["image_status"], "Needs review")  # NC skipped, CC BY selected
        self.assertEqual(two["license"], "CC BY 2.0")
        self.assertEqual(two["credit"], "Jane Doe")
        self.assertFalse(two["identity_confirmed"])
        self.assertTrue((fp.ASSETS / "c_000002.webp").exists())
        self.assertEqual(m["c_000003"]["image_status"], "Needs review")
        self.assertEqual(m["c_000004"]["image_status"], "Rejected")  # duplicate of Player Three's file
        self.assertNotIn("Verified", {e["image_status"] for e in m.values()})

        from PIL import Image
        with Image.open(fp.ASSETS / "c_000002.webp") as im:
            self.assertEqual(im.format, "WEBP")
            self.assertLessEqual(max(im.size), 683)

        fp.confirm(SimpleNamespace(ids=["c_000002"]))
        m = {e["id"]: e for e in json.loads(fp.MANIFEST.read_text())}
        self.assertEqual(m["c_000002"]["image_status"], "Verified")
        with self.assertRaises(SystemExit):  # nothing to confirm without a verified licence
            fp.confirm(SimpleNamespace(ids=["c_000001"]))

    def test_check_here_download_later(self):
        net = FakeNet({"File:Good1.jpg": info("Good1.jpg")})
        m = self.run_fetch(net, no_download=True)
        three = m["c_000003"]
        self.assertEqual((three["image_status"], three["asset"]), ("Needs review", ""))
        self.assertTrue(three["thumb_url"].startswith("https://"))
        self.assertFalse((fp.ASSETS / "c_000003.webp").exists())
        fp.review(SimpleNamespace())
        self.assertIn("c_000003", fp.REVIEW.read_text())
        fp.confirm(SimpleNamespace(ids=["c_000003"]))  # identity first...
        m = {e["id"]: e for e in json.loads(fp.MANIFEST.read_text())}
        self.assertEqual(m["c_000003"]["image_status"], "Needs review")  # ...but no file yet
        orig = fp.Net
        fp.Net = lambda contact: net
        try:
            fp.download(SimpleNamespace(again=False))
        finally:
            fp.Net = orig
        m = {e["id"]: e for e in json.loads(fp.MANIFEST.read_text())}
        self.assertEqual(m["c_000003"]["image_status"], "Verified")
        self.assertTrue((fp.ASSETS / "c_000003.webp").exists())

    def test_network_failure_is_not_success(self):
        m = self.run_fetch(FakeNet({}, down=True))
        for e in m.values():
            self.assertIn(e["image_status"], ("Missing", "Rejected"))
            self.assertEqual(e["licence_status"], "Unverified")
            self.assertFalse(e["identity_confirmed"])
        self.assertIn("unreachable", m["c_000002"]["reason"])


if __name__ == "__main__":
    unittest.main()
