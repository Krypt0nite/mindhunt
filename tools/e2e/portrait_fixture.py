#!/usr/bin/env python3
"""Build a throwaway PACKS_DIR + ASSETS_DIR to test the portrait reveal.

  python3 tools/e2e/portrait_fixture.py OUTDIR valid|missing|corrupt

Every cricket candidate gets an image reference to a synthetic test card
(grey square labelled TEST, credited "MindHunt test fixture", CC0) - never a
real photo. "missing" ships no files (server must drop the references);
"corrupt" ships undecodable files (browser must fall back to initials).
"""
import json
import pathlib
import sys

from PIL import Image, ImageDraw

ROOT = pathlib.Path(__file__).resolve().parents[2]
out, mode = pathlib.Path(sys.argv[1]), sys.argv[2]
packs, assets = out / "packs", out / "assets" / "cricket-men-intl"
packs.mkdir(parents=True, exist_ok=True)
assets.mkdir(parents=True, exist_ok=True)
pack = json.loads((ROOT / "server/packs/cricket-men-intl.v3.json").read_text())
card = Image.new("RGB", (320, 400), (200, 200, 200))
ImageDraw.Draw(card).text((130, 190), "TEST", fill=(40, 40, 40))
for c in pack["candidates"]:
    c["image"] = {"asset": f"cricket-men-intl/{c['id']}.webp", "license": "CC0",
                  "license_url": "https://creativecommons.org/publicdomain/zero/1.0/",
                  "credit": "MindHunt test fixture", "source_url": "https://example.org/mindhunt-test-fixture"}
    f = assets / f"{c['id']}.webp"
    if mode == "valid":
        card.save(f, "WEBP")
    elif mode == "corrupt":
        f.write_bytes(b"RIFF0000WEBPnot really an image")
(packs / "cricket-men-intl.v3.json").write_text(json.dumps(pack))
print(out)
