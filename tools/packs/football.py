SPEC = {
    "id": "football-players",
    "version": 3,  # v2: Henry captained France; v3: checked 2026-10-18 (tools/evidence/football.py)
    "draft": True,  # stronger foot not yet sourced for 24 players
    "as_of": "2026-10-18",
    "category": "Sports",
    "title": "Footballers",
    "noun": "footballer",
    "description": "Legends and modern stars of men's international football, plus India's greats.",
    "letters": "Does their name start with “{value}”?",
    "attributes": [
        {"key": "country", "label": "Country", "type": "enum",
         "values": ["Argentina", "Brazil", "France", "Germany", "Italy", "Spain", "Portugal", "Netherlands", "England",
                    "Croatia", "Poland", "Sweden", "Norway", "Egypt", "Ivory Coast", "Liberia", "India", "South Korea"],
         "templates": {"eq": "Have they played for {value}?"},
         "aliases": {"Argentina": ["argentine", "argentinian"], "Brazil": ["brazilian"], "France": ["french"],
                     "Germany": ["german"], "Italy": ["italian"], "Spain": ["spanish"], "Portugal": ["portuguese"],
                     "Netherlands": ["dutch", "holland"], "England": ["english"], "India": ["indian"],
                     "South Korea": ["korean", "korea"], "Egypt": ["egyptian"], "Croatia": ["croatian"]}},
        {"key": "continent", "label": "National team's continent", "type": "enum",
         "values": ["Europe", "South America", "Africa", "Asia"],
         "templates": {"eq": "Is their national team from {value}?"},
         "aliases": {"Europe": ["european"], "South America": ["south american"], "Africa": ["african"], "Asia": ["asian"]}},
        {"key": "position", "label": "Position", "type": "enum", "values": ["Goalkeeper", "Defender", "Midfielder", "Forward"],
         "help": "Main position over their career (first listed in their Wikipedia profile). Unknown where it is split between attacking midfield and forward (Maradona, Ronaldinho, Neymar).",
         "templates": {"eq:Goalkeeper": "Are they a goalkeeper?", "eq:Defender": "Are they a defender?",
                       "eq:Midfielder": "Are they a midfielder?", "eq:Forward": "Are they a forward?"},
         "aliases": {"Goalkeeper": ["keeper", "goalie", "gk"], "Defender": ["centre back", "full back", "defence"],
                     "Midfielder": ["midfield", "playmaker"], "Forward": ["striker", "winger", "attacker"]}},
        {"key": "foot", "label": "Stronger foot", "type": "enum", "values": ["Left", "Right"],
         "templates": {"eq": "Are they {value_lc}-footed?"},
         "aliases": {"Left": ["left footed", "left foot"], "Right": ["right footed", "right foot"]}},
        {"key": "debut", "label": "International debut decade", "type": "int", "min": 1950, "max": 2020, "step": 10,
         "templates": {"eq": "Did they make their international debut in the {value}s?",
                       "gt": "Did they debut after the {value}s?", "lt": "Did they debut before the {value}s?",
                       "between": "Did they debut between the {value}s and the {value2}s?"}},
        {"key": "world_cup", "label": "World Cup winner", "type": "bool",
         "help": "Member of a squad that won the men's FIFA World Cup.",
         "templates": {"is": "Have they won the World Cup?", "is_false": "Have they never won the World Cup?"}},
        {"key": "ballon_dor", "label": "Ballon d'Or winner", "type": "bool",
         "help": "Won the Ballon d'Or at least once (honorary awards don't count).",
         "templates": {"is": "Have they won the Ballon d'Or?", "is_false": "Have they never won the Ballon d'Or?"}},
        {"key": "captain", "label": "Captained their country", "type": "bool",
         "help": "Regular national team captain. Unknown where it was only occasional (Ronaldinho) or isn't sourced (Weah).",
         "templates": {"is": "Have they captained their country?", "is_false": "Have they never captained their country?"}},
    ],
    "columns": ["name", "country", "continent", "position", "foot", "debut", "world_cup", "ballon_dor", "captain"],
}

ROWS = """
Pelé                 | Brazil      | South America | Forward    | Right | 1950 | Y | N | N
Johan Cruyff         | Netherlands | Europe        | Forward    | Right | 1960 | N | Y | Y
Franz Beckenbauer    | Germany     | Europe        | Defender   | Right | 1960 | Y | Y | Y
Michel Platini       | France      | Europe        | Midfielder | Right | 1970 | N | Y | Y
Diego Maradona       | Argentina   | South America | -          | Left  | 1970 | Y | N | Y
Lothar Matthäus      | Germany     | Europe        | Midfielder | Right | 1980 | Y | Y | Y
Paolo Maldini        | Italy       | Europe        | Defender   | Right | 1980 | N | N | Y
George Weah          | Liberia     | Africa        | Forward    | Right | 1980 | N | Y | -
Zinedine Zidane      | France      | Europe        | Midfielder | Right | 1990 | Y | Y | Y
Ronaldo Nazário      | Brazil      | South America | Forward    | Right | 1990 | Y | Y | N
Ronaldinho           | Brazil      | South America | -          | Right | 1990 | Y | Y | -
David Beckham        | England     | Europe        | Midfielder | Right | 1990 | N | N | Y
Thierry Henry        | France      | Europe        | Forward    | Right | 1990 | Y | N | Y
Fabio Cannavaro      | Italy       | Europe        | Defender   | Right | 1990 | Y | Y | Y
Gianluigi Buffon     | Italy       | Europe        | Goalkeeper | Right | 1990 | Y | N | Y
Bhaichung Bhutia     | India       | Asia          | Forward    | Right | 1990 | N | N | Y
Iker Casillas        | Spain       | Europe        | Goalkeeper | Right | 2000 | Y | N | Y
Xavi                 | Spain       | Europe        | Midfielder | Right | 2000 | Y | N | N
Zlatan Ibrahimović   | Sweden      | Europe        | Forward    | Right | 2000 | N | N | Y
Didier Drogba        | Ivory Coast | Africa        | Forward    | Right | 2000 | N | N | Y
Kaká                 | Brazil      | South America | Midfielder | Right | 2000 | Y | Y | N
Cristiano Ronaldo    | Portugal    | Europe        | Forward    | Right | 2000 | N | Y | Y
Wayne Rooney         | England     | Europe        | Forward    | Right | 2000 | N | N | Y
Lionel Messi         | Argentina   | South America | Forward    | Left  | 2000 | Y | Y | Y
Sergio Ramos         | Spain       | Europe        | Defender   | Right | 2000 | Y | N | Y
Sunil Chhetri        | India       | Asia          | Forward    | Right | 2000 | N | N | Y
Luka Modrić          | Croatia     | Europe        | Midfielder | Right | 2000 | N | Y | Y
Andrés Iniesta       | Spain       | Europe        | Midfielder | Right | 2000 | Y | N | N
Robert Lewandowski   | Poland      | Europe        | Forward    | Right | 2000 | N | N | Y
Manuel Neuer         | Germany     | Europe        | Goalkeeper | Right | 2000 | Y | N | Y
Neymar               | Brazil      | South America | -          | Right | 2010 | N | N | Y
Son Heung-min        | South Korea | Asia          | Forward    | -     | 2010 | N | N | Y
Mohamed Salah        | Egypt       | Africa        | Forward    | Left  | 2010 | N | N | Y
Harry Kane           | England     | Europe        | Forward    | Right | 2010 | N | N | Y
Virgil van Dijk      | Netherlands | Europe        | Defender   | Right | 2010 | N | N | Y
Kylian Mbappé        | France      | Europe        | Forward    | Right | 2010 | Y | N | Y
Erling Haaland       | Norway      | Europe        | Forward    | Left  | 2010 | N | N | N
"""
