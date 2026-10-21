"""Evidence for the Countries pack, checked 2026-10-18.

Each attribute was checked against one list page (or an official page), so
EXPECT holds what that page says for every country in the pack. Where the
pack's rule needs interpretation, RULES says how it was applied.
"""

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"

ALL = """India Pakistan Bangladesh Nepal Bhutan Sri_Lanka China Myanmar Japan South_Korea Mongolia Indonesia
Thailand Malaysia Singapore Philippines Saudi_Arabia Iran Kazakhstan Turkey Russia United_Kingdom Ireland Iceland
France Germany Italy Spain Portugal Netherlands Belgium Switzerland Sweden Norway Finland Greece Egypt Morocco
Nigeria Ethiopia Kenya South_Africa United_States Canada Mexico Cuba Jamaica Brazil Argentina Chile Peru
Australia New_Zealand""".replace("_", "\0").split()
ALL = [n.replace("\0", " ") for n in ALL]


def yes(*names):
    return {n: ("Y" if n in names else "N") for n in ALL}


EXPECT = {
    "drives_left": yes("India", "Pakistan", "Bangladesh", "Nepal", "Bhutan", "Sri Lanka", "Japan", "Indonesia", "Thailand",
                       "Malaysia", "Singapore", "United Kingdom", "Ireland", "Kenya", "South Africa", "Jamaica",
                       "Australia", "New Zealand"),
    "landlocked": yes("Bhutan", "Ethiopia", "Mongolia", "Nepal", "Switzerland", "Kazakhstan"),
    "island": yes("Sri Lanka", "Japan", "Indonesia", "Singapore", "Philippines", "United Kingdom", "Ireland", "Iceland",
                  "Cuba", "Jamaica", "New Zealand"),
    "g20": yes("Argentina", "Australia", "Brazil", "Canada", "China", "France", "Germany", "India", "Indonesia", "Italy",
               "Japan", "Mexico", "Russia", "Saudi Arabia", "South Africa", "South Korea", "Turkey", "United Kingdom",
               "United States"),
    "borders_india": yes("Bangladesh", "Bhutan", "China", "Myanmar", "Nepal", "Pakistan"),
    # Held up to 2024 (Moscow 1980 counts for Russia, Munich 1972 and Berlin 1936 for Germany).
    "summer_olympics": yes("Greece", "France", "United States", "United Kingdom", "Sweden", "Belgium", "Netherlands",
                           "Germany", "Finland", "Australia", "Italy", "Japan", "Mexico", "Canada", "Russia",
                           "South Korea", "Spain", "China", "Brazil"),
    # Held up to 2026 (England 1966 counts for the UK, West Germany 1974 for Germany). 2030/2034 not yet held.
    "fifa_world_cup": yes("Italy", "France", "Brazil", "Switzerland", "Sweden", "Chile", "United Kingdom", "Mexico",
                          "Germany", "Argentina", "Spain", "United States", "South Korea", "Japan", "South Africa",
                          "Russia", "Canada"),
    "pop_100m": yes("India", "China", "United States", "Indonesia", "Pakistan", "Nigeria", "Brazil", "Bangladesh",
                    "Russia", "Mexico", "Japan", "Philippines", "Ethiopia", "Egypt"),
    "monarchy": yes("Bhutan", "Japan", "Thailand", "Malaysia", "Saudi Arabia", "United Kingdom", "Spain", "Netherlands",
                    "Belgium", "Sweden", "Norway", "Morocco", "Canada", "Jamaica", "Australia", "New Zealand"),
    # Wholly or mostly south per the page; straddling countries it doesn't settle stay Unknown.
    "south": {**yes("Indonesia", "Australia", "New Zealand", "Brazil", "Argentina", "Chile", "Peru", "South Africa"),
              "Kenya": "-"},
}

CONTINENT = {n: "Asia" for n in ALL[:18]}
CONTINENT.update({"Kazakhstan": "Asia;Europe", "Turkey": "Asia;Europe", "Russia": "Asia;Europe"})
CONTINENT.update({n: "Europe" for n in ["United Kingdom", "Ireland", "Iceland", "France", "Germany", "Italy", "Spain",
                                        "Portugal", "Netherlands", "Belgium", "Switzerland", "Sweden", "Norway",
                                        "Finland", "Greece"]})
CONTINENT.update({n: "Africa" for n in ["Morocco", "Nigeria", "Ethiopia", "Kenya", "South Africa"]})
CONTINENT["Egypt"] = "Africa;Asia"  # Sinai: contiguous transcontinental, like Turkey and Kazakhstan
CONTINENT.update({n: "North America" for n in ["United States", "Canada", "Mexico", "Cuba", "Jamaica"]})
CONTINENT.update({n: "South America" for n in ["Brazil", "Argentina", "Chile", "Peru"]})
CONTINENT.update({n: "Oceania" for n in ["Australia", "New Zealand"]})
EXPECT["continent"] = CONTINENT

SOURCES = {
    "drives_left": [W + "Left-_and_right-hand_traffic"],
    "landlocked": [W + "Landlocked_country"],
    "island": [W + "List_of_island_countries"],
    "g20": ["https://www.bundeswirtschaftsministerium.de/Redaktion/EN/Artikel/Foreign-Trade/world-economic-summit-02.html"],
    "borders_india": [W + "Borders_of_India"],
    "summer_olympics": [W + "List_of_Olympic_Games_host_cities"],
    "fifa_world_cup": [W + "List_of_FIFA_World_Cup_hosts"],
    "pop_100m": [W + "List_of_countries_and_dependencies_by_population"],
    "monarchy": [W + "List_of_current_monarchies"],
    "south": [W + "Southern_Hemisphere"],
    "continent": [W + "List_of_transcontinental_countries"],
}

SOURCE_TIER = {"g20": "Official (German Federal Ministry for Economic Affairs)"}

RULES = {
    "continent": "Contiguous transcontinental countries count for both continents (Russia, Turkey, Kazakhstan, Egypt). "
                 "Overseas territories and islands don't (France, Spain, USA, Indonesia stay single).",
    "landlocked": "Kazakhstan's Caspian coast doesn't count (the Caspian has no access to the ocean).",
    "island": "Whole country on islands; Australia counts as a continent. UK and Ireland share an island but are island countries.",
    "borders_india": "Recognised land borders; India's claimed border with Afghanistan (in Pakistan-administered territory) is not counted.",
    "summer_olympics": "Games held up to 2024. Predecessor states count (Moscow 1980 for Russia, Munich 1972 for Germany). 2028/2032 not yet held.",
    "fifa_world_cup": "Men's World Cups held up to 2026, including co-hosts. England 1966 counts for the UK; West Germany 1974 for Germany. 2030 and 2034 hosts (Morocco, Portugal, Saudi Arabia) not yet.",
    "pop_100m": "Latest figures on the page (2022–2026).",
    "monarchy": "Includes Commonwealth realms (Canada, Jamaica, Australia, New Zealand).",
    "south": "Wholly or mostly south of the equator as the page states; Kenya straddles the equator and stays Unknown.",
    "g20": "Member countries; the EU and African Union are not countries.",
}
