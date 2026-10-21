#!/usr/bin/env python3
"""Generate packs/cricket-men-intl.v3.json.

Only attributes that do not drift over time are used (no career stats or
current teams). Every value was checked on 2026-10-18; the evidence is in
tools/cricket_evidence.py and the audit in mindhunt-cricket-players-verified.xlsx
(tools/verify_cricket.py rebuilds both and fails if they disagree with ROWS).

Debut decade is the decade of the first ODI for any country; "-" means the
player never played an ODI and the game answers Unknown.

Portraits are attached only from tools/portraits/cricket-manifest.json entries
whose image_status is "Verified" (licence checked on Commons and the person
confirmed by a human). See tools/fetch_portraits.py.

Run: python3 tools/gen_cricket_pack.py > server/packs/cricket-men-intl.v3.json
"""
import hashlib
import json
import pathlib

ROWS = """
Sachin Tendulkar|Tendulkar|India|Batter|R|N|-|1980|Y|Y
Virat Kohli|Kohli|India|Batter|R|N|-|2000|Y|Y
Rohit Sharma|Sharma|India|Batter|R|N|-|2000|Y|N
MS Dhoni|Dhoni|India|Wicketkeeper|R|N|-|2000|Y|Y
Sourav Ganguly|Ganguly|India|Batter|L|N|-|1990|Y|N
Rahul Dravid|Dravid|India|Batter|R|N|-|1990|Y|N
Anil Kumble|Kumble|India|Bowler|R|S|R|1990|Y|N
Kapil Dev|Dev|India|All-rounder|R|P|R|1970|Y|Y
Sunil Gavaskar|Gavaskar|India|Batter|R|N|-|1970|Y|Y
Jasprit Bumrah|Bumrah|India|Bowler|R|P|R|2010|Y|N
Ravindra Jadeja|Jadeja|India|All-rounder|L|S|L|2000|N|N
Ravichandran Ashwin|Ashwin|India|All-rounder|R|S|R|2010|N|Y
Yuvraj Singh|Singh|India|All-rounder|L|S|L|2000|N|Y
Virender Sehwag|Sehwag|India|Batter|R|N|-|1990|Y|Y
Zaheer Khan|Khan|India|Bowler|R|P|L|2000|N|Y
Harbhajan Singh|Singh|India|Bowler|R|S|R|1990|N|Y
Shikhar Dhawan|Dhawan|India|Batter|L|N|-|2010|Y|N
Shubman Gill|Gill|India|Batter|R|N|-|2010|Y|N
Hardik Pandya|Pandya|India|All-rounder|R|P|R|2010|Y|N
VVS Laxman|Laxman|India|Batter|R|N|-|1990|N|N
Mohammed Shami|Shami|India|Bowler|R|P|R|2010|N|N
Ricky Ponting|Ponting|Australia|Batter|R|N|-|1990|Y|Y
Shane Warne|Warne|Australia|Bowler|R|S|R|1990|Y|Y
Glenn McGrath|McGrath|Australia|Bowler|R|P|R|1990|N|Y
Adam Gilchrist|Gilchrist|Australia|Wicketkeeper|L|N|-|1990|Y|Y
Steve Waugh|Waugh|Australia|Batter|R|N|-|1980|Y|Y
Don Bradman|Bradman|Australia|Batter|R|N|-|-|Y|N
Steve Smith|Smith|Australia|Batter|R|N|-|2010|Y|Y
Pat Cummins|Cummins|Australia|Bowler|R|P|R|2010|Y|Y
Mitchell Starc|Starc|Australia|Bowler|L|P|L|2010|N|Y
Matthew Hayden|Hayden|Australia|Batter|L|N|-|1990|N|Y
Allan Border|Border|Australia|Batter|L|N|-|1970|Y|Y
Dennis Lillee|Lillee|Australia|Bowler|R|P|R|1970|N|N
Joe Root|Root|England|Batter|R|N|-|2010|Y|Y
Ben Stokes|Stokes|England|All-rounder|L|P|R|2010|Y|Y
James Anderson|Anderson|England|Bowler|L|P|R|2000|N|N
Stuart Broad|Broad|England|Bowler|L|P|R|2000|Y|N
Alastair Cook|Cook|England|Batter|L|N|-|2000|Y|N
Kevin Pietersen|Pietersen|England|Batter|R|N|-|2000|Y|N
Ian Botham|Botham|England|All-rounder|R|P|R|1970|Y|N
Eoin Morgan|Morgan|England|Batter|L|N|-|2000|Y|Y
Jos Buttler|Buttler|England|Wicketkeeper|R|N|-|2010|Y|Y
Andrew Flintoff|Flintoff|England|All-rounder|R|P|R|1990|Y|N
Imran Khan|Khan|Pakistan|All-rounder|R|P|R|1970|Y|Y
Wasim Akram|Akram|Pakistan|Bowler|L|P|L|1980|Y|Y
Waqar Younis|Younis|Pakistan|Bowler|R|P|R|1980|Y|N
Babar Azam|Azam|Pakistan|Batter|R|N|-|2010|Y|N
Shahid Afridi|Afridi|Pakistan|All-rounder|R|S|R|1990|Y|N
Inzamam-ul-Haq|Inzamam-ul-Haq|Pakistan|Batter|R|N|-|1990|Y|Y
Shoaib Akhtar|Akhtar|Pakistan|Bowler|R|P|R|1990|N|N
Muttiah Muralitharan|Muralitharan|Sri Lanka|Bowler|R|S|R|1990|N|Y
Kumar Sangakkara|Sangakkara|Sri Lanka|Wicketkeeper|L|N|-|2000|Y|N
Mahela Jayawardene|Jayawardene|Sri Lanka|Batter|R|N|-|1990|Y|N
Sanath Jayasuriya|Jayasuriya|Sri Lanka|All-rounder|L|S|L|1980|Y|Y
Lasith Malinga|Malinga|Sri Lanka|Bowler|R|P|R|2000|Y|N
AB de Villiers|de Villiers|South Africa|Batter|R|N|-|2000|Y|N
Dale Steyn|Steyn|South Africa|Bowler|R|P|R|2000|N|N
Graeme Smith|Smith|South Africa|Batter|L|N|-|2000|Y|N
Hashim Amla|Amla|South Africa|Batter|R|N|-|2000|Y|N
Kagiso Rabada|Rabada|South Africa|Bowler|L|P|R|2010|N|N
Brian Lara|Lara|West Indies|Batter|L|N|-|1990|Y|N
Viv Richards|Richards|West Indies|Batter|R|N|-|1970|Y|Y
Chris Gayle|Gayle|West Indies|All-rounder|L|S|R|1990|Y|N
Clive Lloyd|Lloyd|West Indies|Batter|L|N|-|1970|Y|Y
Curtly Ambrose|Ambrose|West Indies|Bowler|L|P|R|1980|N|N
Courtney Walsh|Walsh|West Indies|Bowler|R|P|R|1980|Y|N
Kane Williamson|Williamson|New Zealand|Batter|R|N|-|2010|Y|N
Trent Boult|Boult|New Zealand|Bowler|R|P|L|2010|N|N
Ross Taylor|Taylor|New Zealand|Batter|R|N|-|2000|Y|N
Daniel Vettori|Vettori|New Zealand|Bowler|L|S|L|1990|Y|N
Shakib Al Hasan|Al Hasan|Bangladesh|All-rounder|L|S|L|2000|Y|N
Rashid Khan|Khan|Afghanistan|Bowler|R|S|R|2010|Y|N
"""

HAND = {"L": "Left", "R": "Right"}
BOWL = {"P": "Pace", "S": "Spin", "N": "Not a regular bowler"}


# IDs are derived from the name a player first had in the pack. A spelling
# fix must not change the ID (saved games, daily logs and stats refer to it).
ID_FROM_NAME = {"Muttiah Muralitharan": "Muttiah Muralidaran"}

MANIFEST = pathlib.Path(__file__).parent / "portraits" / "cricket-manifest.json"


def cid(name: str) -> str:
    name = ID_FROM_NAME.get(name, name)
    return "c_" + hashlib.sha256(("mindhunt-cricket:" + name).encode()).hexdigest()[:6]


def verified_images() -> dict:
    """Images whose licence and identity were both confirmed."""
    if not MANIFEST.exists():
        return {}
    out = {}
    for e in json.loads(MANIFEST.read_text()):
        if e.get("image_status") == "Verified" and e.get("identity_confirmed") is True and e.get("asset"):
            out[e["id"]] = {k: e[k] for k in ("asset", "license", "license_url", "credit", "source_url")}
    return out


def main() -> None:
    countries = []
    candidates = []
    images = verified_images()
    for line in ROWS.strip().splitlines():
        name, surname, country, role, bat, bowl, arm, decade, capt, wc = line.split("|")
        if country not in countries:
            countries.append(country)
        attrs = {
            "country": country,
            "role": role,
            "batting_hand": HAND[bat],
            "bowling": BOWL[bowl],
            "captained_country": capt == "Y",
            "odi_wc_winner": wc == "Y",
            "surname": surname,
        }
        if decade != "-":
            attrs["debut_decade"] = int(decade)
        if arm != "-":
            attrs["bowling_arm"] = HAND[arm]
        c = {"id": cid(name), "name": name, "attrs": attrs}
        if c["id"] in images:
            c["image"] = images[c["id"]]
        candidates.append(c)

    pack = {
        "id": "cricket-men-intl",
        "version": 3,
        "as_of": "2026-10-18",
        "category": "Sports",
        "title": "International Cricketers",
        "noun": "cricketer",
        "description": "Legends and modern stars of men's international cricket.",
        "draft": False,  # every fact verified; role calls decided by the pack owner 2026-10-18
        "notes": "Facts checked 2026-10-18 against ICC/BCCI squad announcements and Wikipedia captain, ODI-cricketer and World Cup squad lists; 8 role classifications decided by the pack owner (see the Audit sheet).",
        "attributes": [
            {"key": "country", "label": "Country", "type": "enum", "values": countries,
             "templates": {"eq": "Have they represented {value}?"},
             "aliases": {k: v for k, v in {
                 "India": ["indian"], "Australia": ["australian", "aussie", "aussies"],
                 "England": ["english"], "Pakistan": ["pakistani", "pak"],
                 "Sri Lanka": ["sri lankan", "lanka", "lankan"], "South Africa": ["south african", "proteas"],
                 "West Indies": ["west indian", "windies"], "New Zealand": ["kiwi", "kiwis", "nz"],
                 "Bangladesh": ["bangladeshi"], "Afghanistan": ["afghan"],
             }.items() if k in countries}},
            {"key": "role", "label": "Role", "type": "enum",
             "values": ["Batter", "Bowler", "All-rounder", "Wicketkeeper"],
             "templates": {"eq:Batter": "Are they mainly a batter?",
                           "eq:Bowler": "Are they mainly a bowler?",
                           "eq:All-rounder": "Are they an all-rounder?",
                           "eq:Wicketkeeper": "Are they a wicketkeeper?"},
             "aliases": {"Batter": ["batsman", "batting specialist"], "Bowler": ["bowling specialist"],
                         "All-rounder": ["allrounder", "all rounder"],
                         "Wicketkeeper": ["keeper", "wicket keeper", "wk", "stumper"]}},
            {"key": "batting_hand", "label": "Batting hand", "type": "enum", "values": ["Left", "Right"],
             "templates": {"eq": "Do they bat {value_lc}-handed?"},
             "aliases": {"Left": ["left handed", "left hander", "left handed batter", "lefty", "southpaw"],
                         "Right": ["right handed", "right hander", "right handed batter"]}},
            {"key": "bowling", "label": "Bowling", "type": "enum", "values": list(BOWL.values()),
             "help": "Batters and wicketkeepers count as 'Not a regular bowler', even if they bowled occasionally.",
             "templates": {"eq:Pace": "Are they a pace bowler?",
                           "eq:Spin": "Are they a spin bowler?",
                           "eq:Not a regular bowler": "Do they not bowl regularly?"},
             "aliases": {"Pace": ["pace bowler", "pacer", "fast", "fast bowler", "seamer", "seam bowler", "quick", "medium pacer"],
                         "Spin": ["spinner", "spin bowler", "leg spinner", "off spinner", "leggie", "finger spinner"]}},
            {"key": "bowling_arm", "label": "Bowling arm", "type": "enum", "values": ["Left", "Right"],
             "help": "Recorded only for bowlers and all-rounders; others answer Unknown.",
             "templates": {"eq": "Do they bowl {value_lc}-arm?"},
             "aliases": {"Left": ["left arm"], "Right": ["right arm"]}},
            {"key": "debut_decade", "label": "ODI debut decade", "type": "int", "min": 1970, "max": 2020, "step": 10,
             "help": "Decade of their first one-day international, for any country. Players who never played an ODI (Bradman) answer Unknown.",
             "templates": {"eq": "Did they make their ODI debut in the {value}s?",
                           "gt": "Did they make their ODI debut after the {value}s?",
                           "lt": "Did they make their ODI debut before the {value}s?",
                           "between": "Did they make their ODI debut between the {value}s and the {value2}s?"}},
            {"key": "captained_country", "label": "Captained their country", "type": "bool",
             "help": "Officially captained in a Test, ODI or T20I, including as stand-in captain.",
             "templates": {"is": "Have they captained their country?",
                           "is_false": "Have they never captained their country?"}},
            {"key": "odi_wc_winner", "label": "ODI World Cup winner", "type": "bool",
             "help": "Member of a squad that won the men's 50-over World Cup.",
             "templates": {"is": "Have they won the 50-over World Cup?",
                           "is_false": "Have they never won the 50-over World Cup?"}},
            {"key": "surname", "label": "Surname", "type": "string",
             "help": "Ask about the first 1 to 3 letters (spaces and hyphens ignored).",
             "templates": {"starts_with": "Does their surname start with “{value}”?"}},
        ],
        "candidates": candidates,
    }
    print(json.dumps(pack, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
