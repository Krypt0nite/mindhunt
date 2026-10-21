SPEC = {
    "id": "world-countries",
    "version": 2,  # v2: every fact checked 2026-10-18 (tools/evidence/countries.py); Egypt spans Asia, Indonesia mostly south
    "draft": False,
    "as_of": "2026-10-18",
    "category": "Geography",
    "title": "Countries",
    "noun": "country",
    "description": "Countries from every continent, with an India-first selection.",
    "letters": "Does the country's name start with “{value}”?",
    "attributes": [
        {"key": "continent", "label": "Continent", "type": "set",
         "values": ["Asia", "Europe", "Africa", "North America", "South America", "Oceania"],
         "help": "Countries whose land spans two continents (Russia, Turkey, Kazakhstan, Egypt) count for both; overseas islands don't.",
         "templates": {"contains": "Is it (at least partly) in {value}?"},
         "aliases": {"Asia": ["asian"], "Europe": ["european"], "Africa": ["african"],
                     "North America": ["north american", "caribbean"], "South America": ["south american", "latin america"],
                     "Oceania": ["pacific", "australasia"]}},
        {"key": "landlocked", "label": "Landlocked", "type": "bool",
         "templates": {"is": "Is it landlocked?", "is_false": "Does it have a coastline?"}},
        {"key": "island", "label": "Island nation", "type": "bool",
         "help": "The whole country is on islands. Australia counts as a continent, not an island nation.",
         "templates": {"is": "Is it an island nation?", "is_false": "Is it on a mainland?"}},
        {"key": "drives_left", "label": "Drives on the left", "type": "bool",
         "templates": {"is": "Do they drive on the left?", "is_false": "Do they drive on the right?"}},
        {"key": "g20", "label": "G20 member", "type": "bool",
         "help": "A member country of the G20 in its own right (the EU and African Union don't count here).",
         "templates": {"is": "Is it a G20 member?", "is_false": "Is it outside the G20?"}},
        {"key": "borders_india", "label": "Borders India", "type": "bool",
         "help": "Shares a land border with India. India itself is No.",
         "templates": {"is": "Does it share a land border with India?", "is_false": "Does it not border India?"}},
        {"key": "summer_olympics", "label": "Hosted the Summer Olympics", "type": "bool",
         "help": "Games held up to 2024. Moscow 1980 counts for Russia, Munich 1972 for Germany.",
         "templates": {"is": "Has it hosted the Summer Olympics?", "is_false": "Has it never hosted the Summer Olympics?"}},
        {"key": "fifa_world_cup", "label": "Hosted the FIFA World Cup", "type": "bool",
         "help": "Men's FIFA World Cups held so far, including co-hosting (2002, 2026). England 1966 counts for the UK; future hosts (2030, 2034) don't yet.",
         "templates": {"is": "Has it hosted the men's FIFA World Cup?", "is_false": "Has it never hosted the FIFA World Cup?"}},
        {"key": "pop_100m", "label": "Over 100 million people", "type": "bool",
         "templates": {"is": "Does it have more than 100 million people?", "is_false": "Does it have fewer than 100 million people?"}},
        {"key": "monarchy", "label": "Monarch as head of state", "type": "bool",
         "help": "Includes Commonwealth realms whose head of state is the British monarch (Canada, Australia, New Zealand, Jamaica).",
         "templates": {"is": "Is its head of state a monarch?", "is_false": "Is it a republic?"}},
        {"key": "south", "label": "Mostly south of the equator", "type": "bool",
         "help": "Wholly or mostly south of the equator. Unknown for countries the equator splits roughly in half (Kenya).",
         "templates": {"is": "Is it mostly in the Southern Hemisphere?", "is_false": "Is it mostly in the Northern Hemisphere?"}},
    ],
    "columns": ["name", "continent", "landlocked", "island", "drives_left", "g20", "borders_india",
                "summer_olympics", "fifa_world_cup", "pop_100m", "monarchy", "south"],
}

#            continent            land isl left g20 bInd olym fifa 100m mon south
ROWS = """
India          | Asia                 | N | N | Y | Y | N | N | N | Y | N | N
Pakistan       | Asia                 | N | N | Y | N | Y | N | N | Y | N | N
Bangladesh     | Asia                 | N | N | Y | N | Y | N | N | Y | N | N
Nepal          | Asia                 | Y | N | Y | N | Y | N | N | N | N | N
Bhutan         | Asia                 | Y | N | Y | N | Y | N | N | N | Y | N
Sri Lanka      | Asia                 | N | Y | Y | N | N | N | N | N | N | N
China          | Asia                 | N | N | N | Y | Y | Y | N | Y | N | N
Myanmar        | Asia                 | N | N | N | N | Y | N | N | N | N | N
Japan          | Asia                 | N | Y | Y | Y | N | Y | Y | Y | Y | N
South Korea    | Asia                 | N | N | N | Y | N | Y | Y | N | N | N
Mongolia       | Asia                 | Y | N | N | N | N | N | N | N | N | N
Indonesia      | Asia                 | N | Y | Y | Y | N | N | N | Y | N | Y
Thailand       | Asia                 | N | N | Y | N | N | N | N | N | Y | N
Malaysia       | Asia                 | N | N | Y | N | N | N | N | N | Y | N
Singapore      | Asia                 | N | Y | Y | N | N | N | N | N | N | N
Philippines    | Asia                 | N | Y | N | N | N | N | N | Y | N | N
Saudi Arabia   | Asia                 | N | N | N | Y | N | N | N | N | Y | N
Iran           | Asia                 | N | N | N | N | N | N | N | N | N | N
Kazakhstan     | Asia;Europe          | Y | N | N | N | N | N | N | N | N | N
Turkey         | Asia;Europe          | N | N | N | Y | N | N | N | N | N | N
Russia         | Asia;Europe          | N | N | N | Y | N | Y | Y | Y | N | N
United Kingdom | Europe               | N | Y | Y | Y | N | Y | Y | N | Y | N
Ireland        | Europe               | N | Y | Y | N | N | N | N | N | N | N
Iceland        | Europe               | N | Y | N | N | N | N | N | N | N | N
France         | Europe               | N | N | N | Y | N | Y | Y | N | N | N
Germany        | Europe               | N | N | N | Y | N | Y | Y | N | N | N
Italy          | Europe               | N | N | N | Y | N | Y | Y | N | N | N
Spain          | Europe               | N | N | N | N | N | Y | Y | N | Y | N
Portugal       | Europe               | N | N | N | N | N | N | N | N | N | N
Netherlands    | Europe               | N | N | N | N | N | Y | N | N | Y | N
Belgium        | Europe               | N | N | N | N | N | Y | N | N | Y | N
Switzerland    | Europe               | Y | N | N | N | N | N | Y | N | N | N
Sweden         | Europe               | N | N | N | N | N | Y | Y | N | Y | N
Norway         | Europe               | N | N | N | N | N | N | N | N | Y | N
Finland        | Europe               | N | N | N | N | N | Y | N | N | N | N
Greece         | Europe               | N | N | N | N | N | Y | N | N | N | N
Egypt          | Africa;Asia          | N | N | N | N | N | N | N | Y | N | N
Morocco        | Africa               | N | N | N | N | N | N | N | N | Y | N
Nigeria        | Africa               | N | N | N | N | N | N | N | Y | N | N
Ethiopia       | Africa               | Y | N | N | N | N | N | N | Y | N | N
Kenya          | Africa               | N | N | Y | N | N | N | N | N | N | -
South Africa   | Africa               | N | N | Y | Y | N | N | Y | N | N | Y
United States  | North America        | N | N | N | Y | N | Y | Y | Y | N | N
Canada         | North America        | N | N | N | Y | N | Y | Y | N | Y | N
Mexico         | North America        | N | N | N | Y | N | Y | Y | Y | N | N
Cuba           | North America        | N | Y | N | N | N | N | N | N | N | N
Jamaica        | North America        | N | Y | Y | N | N | N | N | N | Y | N
Brazil         | South America        | N | N | N | Y | N | Y | Y | Y | N | Y
Argentina      | South America        | N | N | N | Y | N | N | Y | N | N | Y
Chile          | South America        | N | N | N | N | N | N | Y | N | N | Y
Peru           | South America        | N | N | N | N | N | N | N | N | N | Y
Australia      | Oceania              | N | N | Y | Y | N | Y | N | N | Y | Y
New Zealand    | Oceania              | N | Y | Y | N | N | N | N | N | Y | Y
"""
