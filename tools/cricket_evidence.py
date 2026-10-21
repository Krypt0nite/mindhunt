"""Evidence behind every fact in the cricket pack, checked 2026-10-18.

Each entry records what the sources said, so the pack, the verified workbook
and the audit can all be rebuilt from here (tools/verify_cricket.py).

wiki      English Wikipedia article title for the player (infobox: batting,
          bowling, role, ODI debut). Secondary source.
odi       first ODI date as given in the infobox; None = never played an ODI.
odi_team  set only when the first ODI was not for the pack country.
capt      formats captained (T = Test, O = ODI, T20 = T20I) as listed in the
          country's Wikipedia captains list; [] = not listed.
wc        ODI World Cup titles won as a squad member.
wc_unchecked titles believed won but whose squad page could not be opened.
wiki_role role wording in the infobox, when it differs from the pack class.
"""

ACCESSED = "2026-10-18"
WIKI = "https://en.wikipedia.org/wiki/"

CAPTAIN_LISTS = {
    "India": "List_of_India_national_cricket_captains",
    "Australia": "List_of_Australia_national_cricket_captains",
    "England": "List_of_England_cricket_captains",
    "Pakistan": "List_of_Pakistan_national_cricket_captains",
    "Sri Lanka": "List_of_Sri_Lanka_national_cricket_captains",
    "South Africa": "List_of_South_Africa_national_cricket_captains",
    "West Indies": "List_of_West_Indies_cricket_captains",
    "New Zealand": "List_of_New_Zealand_national_cricket_captains",
    "Bangladesh": "List_of_Bangladesh_national_cricket_captains",
    "Afghanistan": "List_of_Afghanistan_national_cricket_captains",
}

ODI_LISTS = {c: f"List_of_{c.replace(' ', '_')}_ODI_cricketers" for c in CAPTAIN_LISTS}

# Primary source first where an official one was reachable.
WC_SOURCES = {
    1975: [WIKI + "1975_Cricket_World_Cup_squads"],
    1979: [WIKI + "1979_Cricket_World_Cup_squads"],
    1983: [WIKI + "1983_Cricket_World_Cup_squads"],
    1987: [WIKI + "1987_Cricket_World_Cup_squads"],
    1992: [WIKI + "1992_Cricket_World_Cup_squads"],
    1996: [WIKI + "1996_Cricket_World_Cup_squads"],
    1999: [WIKI + "1999_Cricket_World_Cup_squads"],
    2003: [WIKI + "2003_Cricket_World_Cup_squads"],
    2007: [WIKI + "2007_Cricket_World_Cup_squads"],
    2011: ["https://www.bcci.tv/news/article/bcci-names-15-member-team-for-wc",
           WIKI + "2011_Cricket_World_Cup_squads"],
    2019: ["https://www.icc-cricket.com/tournaments/cricketworldcup/news/icc-mens-cricket-world-cup-2019-full-teams-and-squads",
           WIKI + "2019_Cricket_World_Cup_squads"],
    2023: ["https://www.icc-cricket.com/tournaments/cricketworldcup/news/india-bound-australia-lock-in-squad-for-2023-world-cup",
           WIKI + "2023_Cricket_World_Cup_squads"],
}

T, O, T20 = "Test", "ODI", "T20I"

P = {
    # India
    "Sachin Tendulkar": dict(wiki="Sachin_Tendulkar", odi="1989-12-18", capt=[T, O], wc=[2011], bowl_note="leg/off break; classed as a batter"),
    "Virat Kohli": dict(wiki="Virat_Kohli", odi="2008-08-18", capt=[T, O, T20], wc=[2011], bowl_note="right-arm medium, occasional"),
    "Rohit Sharma": dict(wiki="Rohit_Sharma", odi="2007-06-23", capt=[T, O, T20], wc=[], bowl_note="off break, occasional"),
    "MS Dhoni": dict(wiki="MS_Dhoni", odi="2004-12-23", capt=[T, O, T20], wc=[2011], note="Infobox fields did not render; batting hand, keeper role and ODI debut (23 Dec 2004 v Bangladesh) taken from the article text."),
    "Sourav Ganguly": dict(wiki="Sourav_Ganguly", odi="1992-01-11", capt=[T, O], wc=[], bowl_note="right-arm medium, part-time"),
    "Rahul Dravid": dict(wiki="Rahul_Dravid", odi="1996-04-03", capt=[T, O], wc=[]),
    "Anil Kumble": dict(wiki="Anil_Kumble", odi="1990-04-25", capt=[T, O], wc=[]),
    "Kapil Dev": dict(wiki="Kapil_Dev", odi="1978-10-01", capt=[T, O], wc=[1983]),
    "Sunil Gavaskar": dict(wiki="Sunil_Gavaskar", odi="1974-07-13", capt=[T, O], wc=[1983]),
    "Jasprit Bumrah": dict(wiki="Jasprit_Bumrah", odi="2016-01-23", capt=[T, T20], wc=[]),
    "Ravindra Jadeja": dict(wiki="Ravindra_Jadeja", odi="2009-02-08", capt=[], wc=[]),
    "Ravichandran Ashwin": dict(wiki="Ravichandran_Ashwin", odi="2010-06-05", capt=[], wc=[2011], wiki_role="Bowling all-rounder"),
    "Yuvraj Singh": dict(wiki="Yuvraj_Singh", odi="2000-10-03", capt=[], wc=[2011], wiki_role="Middle-order batsman"),
    "Virender Sehwag": dict(wiki="Virender_Sehwag", odi="1999-04-01", capt=[T, O, T20], wc=[2011]),
    "Zaheer Khan": dict(wiki="Zaheer_Khan", odi="2000-10-03", capt=[], wc=[2011]),
    "Harbhajan Singh": dict(wiki="Harbhajan_Singh", odi="1998-04-17", capt=[], wc=[2011]),
    "Shikhar Dhawan": dict(wiki="Shikhar_Dhawan", odi="2010-10-20", capt=[O, T20], wc=[]),
    "Shubman Gill": dict(wiki="Shubman_Gill", odi="2019-01-31", capt=[T, O, T20], wc=[]),
    "Hardik Pandya": dict(wiki="Hardik_Pandya", odi="2016-10-16", capt=[O, T20], wc=[]),
    "VVS Laxman": dict(wiki="VVS_Laxman", odi="1998-04-09", capt=[], wc=[]),
    "Mohammed Shami": dict(wiki="Mohammed_Shami", odi="2013-01-06", capt=[], wc=[]),
    # Australia
    "Ricky Ponting": dict(wiki="Ricky_Ponting", odi="1995-02-15", capt=[T, O, T20], wc=[1999, 2003, 2007]),
    "Shane Warne": dict(wiki="Shane_Warne", odi="1993-03-24", capt=[O], wc=[1999], note="Withdrew before the 2003 World Cup, so only 1999 counts."),
    "Glenn McGrath": dict(wiki="Glenn_McGrath", odi="1993-12-09", capt=[], wc=[1999, 2003, 2007]),
    "Adam Gilchrist": dict(wiki="Adam_Gilchrist", odi="1996-10-25", capt=[T, O, T20], wc=[1999, 2003, 2007]),
    "Steve Waugh": dict(wiki="Steve_Waugh", odi="1986-01-09", capt=[T, O], wc=[1987, 1999], bowl_note="right-arm medium; classed as a batter"),
    "Don Bradman": dict(wiki="Donald_Bradman", odi=None, capt=[T], wc=[], note="Test career 1928-1948; ODIs began in 1971, so no ODI debut decade (N/A)."),
    "Steve Smith": dict(wiki="Steve_Smith_(cricketer)", odi="2010-02-19", capt=[T, O, T20], wc=[2023], wc_unchecked=[2015], note="2015 squad page did not render, so the 2015 title is not counted here; 2023 is confirmed, which is enough for Yes."),
    "Pat Cummins": dict(wiki="Pat_Cummins", odi="2011-10-19", capt=[T, O], wc=[2023], wc_unchecked=[2015], note="2015 squad not checked (page did not render); 2023 confirmed (captain)."),
    "Mitchell Starc": dict(wiki="Mitchell_Starc", odi="2010-10-20", capt=[], wc=[2023], wc_unchecked=[2015], note="2015 squad not checked (page did not render); 2023 confirmed."),
    "Matthew Hayden": dict(wiki="Matthew_Hayden", odi="1993-05-19", capt=[], wc=[2003, 2007], bowl_note="right-arm medium, rarely bowled"),
    "Allan Border": dict(wiki="Allan_Border", odi="1979-01-13", capt=[T, O], wc=[1987], bowl_note="slow left-arm orthodox, part-time"),
    "Dennis Lillee": dict(wiki="Dennis_Lillee", odi="1972-08-24", capt=[], wc=[]),
    # England
    "Joe Root": dict(wiki="Joe_Root", odi="2013-01-11", capt=[T], wc=[2019], bowl_note="off break, part-time"),
    "Ben Stokes": dict(wiki="Ben_Stokes", odi="2011-08-25", capt=[T, O], wc=[2019]),
    "James Anderson": dict(wiki="James_Anderson_(cricketer)", odi="2002-12-15", capt=[], wc=[]),
    "Stuart Broad": dict(wiki="Stuart_Broad", odi="2006-08-30", capt=[O, T20], wc=[]),
    "Alastair Cook": dict(wiki="Alastair_Cook", odi="2006-06-28", capt=[T, O, T20], wc=[]),
    "Kevin Pietersen": dict(wiki="Kevin_Pietersen", odi="2004-11-28", capt=[T, O], wc=[]),
    "Ian Botham": dict(wiki="Ian_Botham", odi="1976-08-26", capt=[T, O], wc=[]),
    "Eoin Morgan": dict(wiki="Eoin_Morgan", odi="2006-08-05", odi_team="Ireland", capt=[O, T20], wc=[2019], note="First ODI was for Ireland (v Scotland, 2006); first for England in 2009. Brief rule: first ODI for any country."),
    "Jos Buttler": dict(wiki="Jos_Buttler", odi="2012-02-21", capt=[O, T20], wc=[2019]),
    "Andrew Flintoff": dict(wiki="Andrew_Flintoff", odi="1999-04-07", capt=[T, O], wc=[]),
    # Pakistan
    "Imran Khan": dict(wiki="Imran_Khan", odi="1974-08-31", capt=[T, O], wc=[1992]),
    "Wasim Akram": dict(wiki="Wasim_Akram", odi="1984-11-23", capt=[T, O], wc=[1992], wiki_role="Bowling all-rounder"),
    "Waqar Younis": dict(wiki="Waqar_Younis", odi="1989-10-14", capt=[T, O], wc=[], note="Not in the 1992 squad (injured)."),
    "Babar Azam": dict(wiki="Babar_Azam", odi="2015-05-31", capt=[T, O, T20], wc=[]),
    "Shahid Afridi": dict(wiki="Shahid_Afridi", odi="1996-10-02", capt=[T, O, T20], wc=[]),
    "Inzamam-ul-Haq": dict(wiki="Inzamam-ul-Haq", odi="1991-11-23", capt=[T, O, T20], wc=[1992], bowl_note="slow left-arm orthodox, part-time"),
    "Shoaib Akhtar": dict(wiki="Shoaib_Akhtar", odi="1998-03-28", capt=[], wc=[]),
    # Sri Lanka
    "Muttiah Muralitharan": dict(wiki="Muttiah_Muralitharan", odi="1993-08-12", capt=[], wc=[1996], old_name="Muttiah Muralidaran"),
    "Kumar Sangakkara": dict(wiki="Kumar_Sangakkara", odi="2000-07-05", capt=[T, O, T20], wc=[]),
    "Mahela Jayawardene": dict(wiki="Mahela_Jayawardene", odi="1998-01-24", capt=[T, O, T20], wc=[]),
    "Sanath Jayasuriya": dict(wiki="Sanath_Jayasuriya", odi="1989-12-26", capt=[T, O], wc=[1996]),
    "Lasith Malinga": dict(wiki="Lasith_Malinga", odi="2004-07-17", capt=[O, T20], wc=[]),
    # South Africa
    "AB de Villiers": dict(wiki="AB_de_Villiers", odi="2005-02-02", capt=[T, O, T20], wc=[], wiki_role="Wicket-keeper-batter"),
    "Dale Steyn": dict(wiki="Dale_Steyn", odi="2005-08-17", odi_team="Africa XI", capt=[], wc=[], note="First ODI was for the Africa XI v Asia XI (2005); first for South Africa in 2006. Both in the 2000s."),
    "Graeme Smith": dict(wiki="Graeme_Smith", odi="2002-03-30", capt=[T, O, T20], wc=[], bowl_note="off break, part-time"),
    "Hashim Amla": dict(wiki="Hashim_Amla", odi="2008-03-09", capt=[T, O, T20], wc=[]),
    "Kagiso Rabada": dict(wiki="Kagiso_Rabada", odi="2015-07-10", capt=[], wc=[]),
    # West Indies
    "Brian Lara": dict(wiki="Brian_Lara", odi="1990-11-09", capt=[T, O], wc=[]),
    "Viv Richards": dict(wiki="Viv_Richards", odi="1975-06-07", capt=[T, O], wc=[1975, 1979], wiki_role="Batting all-rounder"),
    "Chris Gayle": dict(wiki="Chris_Gayle", odi="1999-09-11", capt=[T, O, T20], wc=[], wiki_role="All-rounder"),
    "Clive Lloyd": dict(wiki="Clive_Lloyd", odi="1973-09-05", capt=[T, O], wc=[1975, 1979], note="Test debut 1966, first ODI 1973."),
    "Curtly Ambrose": dict(wiki="Curtly_Ambrose", odi="1988-03-12", capt=[], wc=[]),
    "Courtney Walsh": dict(wiki="Courtney_Walsh", odi="1985-01-10", capt=[T, O], wc=[]),
    # New Zealand
    "Kane Williamson": dict(wiki="Kane_Williamson", odi="2010-08-10", capt=[T, O, T20], wc=[]),
    "Trent Boult": dict(wiki="Trent_Boult", odi="2012-07-11", capt=[], wc=[]),
    "Ross Taylor": dict(wiki="Ross_Taylor", odi="2006-03-01", capt=[T, O, T20], wc=[]),
    "Daniel Vettori": dict(wiki="Daniel_Vettori", odi="1997-03-25", capt=[T, O, T20], wc=[], wiki_role="All-rounder"),
    # Bangladesh, Afghanistan
    "Shakib Al Hasan": dict(wiki="Shakib_Al_Hasan", odi="2006-08-06", capt=[T, O, T20], wc=[]),
    "Rashid Khan": dict(wiki="Rashid_Khan", odi="2015-10-18", capt=[T, O, T20], wc=[], wiki_role="Bowling all-rounder"),
}

# Infobox values used to check batting hand, bowling type and arm.
# (bat, bowling style as written in the infobox; "" = none listed)
INFOBOX = {
    "Sachin Tendulkar": ("R", "Leg break, off break"), "Virat Kohli": ("R", "Right-arm medium"),
    "Rohit Sharma": ("R", "Off break"), "MS Dhoni": ("R", ""), "Sourav Ganguly": ("L", "Right-arm medium"),
    "Rahul Dravid": ("R", "Off break"), "Anil Kumble": ("R", "Leg break"), "Kapil Dev": ("R", "Right-arm fast-medium"),
    "Sunil Gavaskar": ("R", ""), "Jasprit Bumrah": ("R", "Right-arm fast"), "Ravindra Jadeja": ("L", "Slow left-arm orthodox"),
    "Ravichandran Ashwin": ("R", "Off break"), "Yuvraj Singh": ("L", "Slow left-arm orthodox"), "Virender Sehwag": ("R", "Off break"),
    "Zaheer Khan": ("R", "Left-arm fast-medium"), "Harbhajan Singh": ("R", "Off break"), "Shikhar Dhawan": ("L", "Off break"),
    "Shubman Gill": ("R", "Off break"), "Hardik Pandya": ("R", "Right-arm medium-fast"), "VVS Laxman": ("R", "Off break"),
    "Mohammed Shami": ("R", "Right-arm fast"),
    "Ricky Ponting": ("R", "Right-arm medium"), "Shane Warne": ("R", "Leg break"), "Glenn McGrath": ("R", "Right-arm fast-medium"),
    "Adam Gilchrist": ("L", ""), "Steve Waugh": ("R", "Right-arm medium"), "Don Bradman": ("R", "Leg break"),
    "Steve Smith": ("R", "Leg break"), "Pat Cummins": ("R", "Right-arm fast"), "Mitchell Starc": ("L", "Left-arm fast"),
    "Matthew Hayden": ("L", "Right-arm medium"), "Allan Border": ("L", "Slow left-arm orthodox"), "Dennis Lillee": ("R", "Right-arm fast"),
    "Joe Root": ("R", "Off break"), "Ben Stokes": ("L", "Right-arm fast-medium"), "James Anderson": ("L", "Right-arm fast-medium"),
    "Stuart Broad": ("L", "Right-arm fast-medium"), "Alastair Cook": ("L", "Right-arm slow seam"), "Kevin Pietersen": ("R", "Off break"),
    "Ian Botham": ("R", "Right-arm fast-medium"), "Eoin Morgan": ("L", ""), "Jos Buttler": ("R", ""),
    "Andrew Flintoff": ("R", "Right-arm fast-medium"),
    "Imran Khan": ("R", "Right-arm fast"), "Wasim Akram": ("L", "Left-arm fast"), "Waqar Younis": ("R", "Right-arm fast"),
    "Babar Azam": ("R", "Off break"), "Shahid Afridi": ("R", "Leg break"), "Inzamam-ul-Haq": ("R", "Slow left-arm orthodox"),
    "Shoaib Akhtar": ("R", "Right-arm fast"),
    "Muttiah Muralitharan": ("R", "Off break"), "Kumar Sangakkara": ("L", "Off break"), "Mahela Jayawardene": ("R", "Right-arm medium"),
    "Sanath Jayasuriya": ("L", "Slow left-arm orthodox"), "Lasith Malinga": ("R", "Right-arm fast"),
    "AB de Villiers": ("R", "Right-arm medium"), "Dale Steyn": ("R", "Right-arm fast"), "Graeme Smith": ("L", "Off break"),
    "Hashim Amla": ("R", "Off break"), "Kagiso Rabada": ("L", "Right-arm fast"),
    "Brian Lara": ("L", "Leg break"), "Viv Richards": ("R", "Slow seam, off break"), "Chris Gayle": ("L", "Off break"),
    "Clive Lloyd": ("L", "Right-arm medium"), "Curtly Ambrose": ("L", "Right-arm fast"), "Courtney Walsh": ("R", "Right-arm fast"),
    "Kane Williamson": ("R", "Off break"), "Trent Boult": ("R", "Left-arm fast-medium"), "Ross Taylor": ("R", "Off break"),
    "Daniel Vettori": ("L", "Slow left-arm orthodox"),
    "Shakib Al Hasan": ("L", "Slow left-arm orthodox"), "Rashid Khan": ("R", "Leg break googly"),
}

# Every men's ODI World Cup final (en.wikipedia.org/wiki/List_of_Cricket_World_Cup_finals).
WINNERS = {1975: "West Indies", 1979: "West Indies", 1983: "India", 1987: "Australia", 1992: "Pakistan",
           1996: "Sri Lanka", 1999: "Australia", 2003: "Australia", 2007: "Australia", 2011: "India",
           2015: "Australia", 2019: "England", 2023: "Australia"}
WINNERS_URL = WIKI + "List_of_Cricket_World_Cup_finals"

# Role calls where the infobox wording doesn't map onto the pack's four
# classes, decided by the pack owner on 2026-10-18 ("bowling all-rounder" for
# Ashwin is recorded as All-rounder, the nearest class).
ROLE_DECISIONS = {
    "Yuvraj Singh": "All-rounder",
    "Ravichandran Ashwin": "All-rounder",
    "Wasim Akram": "Bowler",
    "Rashid Khan": "Bowler",
    "Daniel Vettori": "Bowler",
    "AB de Villiers": "Batter",
    "Chris Gayle": "All-rounder",
    "Viv Richards": "Batter",
}
ROLE_DECIDED_ON = "2026-10-18"
