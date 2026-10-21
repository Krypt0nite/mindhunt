#!/usr/bin/env python3
"""Build topic packs from the specs in tools/packs/*.py.

Each spec module defines:
  SPEC  - pack metadata, attributes and the column order of ROWS
  ROWS  - one candidate per line, values separated by "|"
          bool: Y/N   set: a;b;c   missing: -   int: digits

If SPEC has "letters", a string attribute "Name" is added so players can ask
"Does the name start with ...?".

Run: python3 tools/build_packs.py            (writes server/packs/<id>.v<version>.json)
"""
import hashlib
import json
import pathlib
import runpy
import sys

ROOT = pathlib.Path(__file__).resolve().parent
OUT = ROOT.parent / "server" / "packs"


def cid(pack_id: str, name: str) -> str:
    return "c_" + hashlib.sha256(f"mindhunt-{pack_id}:{name}".encode()).hexdigest()[:6]


def parse(attr, raw, where):
    raw = raw.strip()
    if raw in ("", "-"):
        return None
    t = attr["type"]
    if t == "bool":
        if raw not in ("Y", "N"):
            sys.exit(f"{where}: {attr['key']} must be Y or N, got {raw!r}")
        return raw == "Y"
    if t == "int":
        return int(raw)
    if t == "set":
        items = [x.strip() for x in raw.split(";") if x.strip()]
        for x in items:
            if x not in attr["values"]:
                sys.exit(f"{where}: {attr['key']} value {x!r} not in {attr['values']}")
        return items
    if t == "enum" and raw not in attr["values"]:
        sys.exit(f"{where}: {attr['key']} value {raw!r} not in {attr['values']}")
    return raw


def build(path: pathlib.Path) -> dict:
    mod = runpy.run_path(str(path))
    spec, rows = dict(mod["SPEC"]), mod["ROWS"]
    attrs = [dict(a) for a in spec.pop("attributes")]
    columns = spec.pop("columns")
    letters = spec.pop("letters", None)
    by_key = {a["key"]: a for a in attrs}
    if columns[0] != "name" or any(c not in by_key for c in columns[1:]):
        sys.exit(f"{path.name}: columns must be name + attribute keys")
    if letters:
        attrs.append({"key": "letters", "label": "Name", "type": "string",
                      "help": "Ask about the first 1 to 3 letters (spaces and hyphens ignored).",
                      "templates": {"starts_with": letters}})
    cands = []
    for n, line in enumerate(rows.strip().splitlines(), start=1):
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        cells = [c.strip() for c in line.split("|")]
        if len(cells) != len(columns):
            sys.exit(f"{path.name} row {n} ({cells[0]}): {len(cells)} cells, expected {len(columns)}")
        name = cells[0]
        values = {}
        for key, raw in zip(columns[1:], cells[1:]):
            v = parse(by_key[key], raw, f"{path.name} row {n} ({name})")
            if v is not None:
                values[key] = v
        if letters:
            values["letters"] = name
        cands.append({"id": cid(spec["id"], name), "name": name, "attrs": values})
    pack = {"id": spec["id"], "version": spec.get("version", 1), "as_of": spec["as_of"],
            "category": spec["category"], "title": spec["title"], "noun": spec.get("noun", ""),
            "description": spec["description"], "draft": spec.get("draft", True),
            "notes": spec.get("notes", "Hand-entered; verify every row before a public release."),
            "attributes": attrs, "candidates": cands}
    return pack


def main():
    specs = sorted(p for p in (ROOT / "packs").glob("*.py") if not p.name.startswith("_"))
    for p in specs:
        pack = build(p)
        out = OUT / f"{pack['id']}.v{pack['version']}.json"
        out.write_text(json.dumps(pack, indent=2, ensure_ascii=False) + "\n")
        print(f"{out.name}: {len(pack['candidates'])} candidates, {len(pack['attributes'])} attributes")


if __name__ == "__main__":
    main()
