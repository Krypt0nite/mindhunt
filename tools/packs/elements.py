SPEC = {
    "id": "chemical-elements",
    "version": 2,  # v2: checked 2026-10-18 (tools/evidence/elements.py)
    "draft": False,
    "as_of": "2026-10-18",
    "category": "Science",
    "title": "Chemical Elements",
    "noun": "element",
    "description": "Fifty-two elements from hydrogen to plutonium.",
    "letters": "Does the element's name start with “{value}”?",
    "attributes": [
        {"key": "atomic_number", "label": "Atomic number", "type": "int", "min": 1, "max": 118, "step": 1,
         "templates": {"eq": "Is its atomic number {value}?", "gt": "Is its atomic number more than {value}?",
                       "lt": "Is its atomic number less than {value}?",
                       "between": "Is its atomic number between {value} and {value2}?"}},
        {"key": "period", "label": "Period", "type": "int", "min": 1, "max": 7, "step": 1,
         "help": "The row of the periodic table.",
         "templates": {"eq": "Is it in period {value}?", "gt": "Is it below period {value}?",
                       "lt": "Is it above period {value}?", "between": "Is it in periods {value} to {value2}?"}},
        {"key": "block", "label": "Block", "type": "enum", "values": ["s", "p", "d", "f"],
         "templates": {"eq": "Is it in the {value}-block?"}},
        {"key": "category", "label": "Category", "type": "enum",
         "values": ["Alkali metal", "Alkaline earth metal", "Transition metal", "Post-transition metal",
                    "Metalloid", "Other nonmetal", "Halogen", "Noble gas", "Lanthanide", "Actinide"],
         "help": "Common textbook grouping. Zinc and mercury count as transition metals (IUPAC 2011). 'Other nonmetal' means hydrogen, carbon, nitrogen, oxygen, phosphorus and sulfur. Polonium is Unknown because sources disagree.",
         "templates": {"eq": "Is it a {value_lc}?",
                       "eq:Other nonmetal": "Is it a nonmetal that isn't a halogen or noble gas?"},
         "aliases": {"Noble gas": ["inert gas"], "Transition metal": ["transition element"],
                     "Other nonmetal": ["nonmetal", "non metal", "reactive nonmetal"], "Alkali metal": ["alkali"],
                     "Alkaline earth metal": ["alkaline earth"]}},
        {"key": "state", "label": "State at room temperature", "type": "enum", "values": ["Solid", "Liquid", "Gas"],
         "help": "At 25 °C and normal pressure.",
         "templates": {"eq": "Is it a {value_lc} at room temperature?"},
         "aliases": {"Gas": ["gaseous"]}},
        {"key": "radioactive", "label": "Radioactive", "type": "bool",
         "help": "Has no stable isotopes.",
         "templates": {"is": "Is it radioactive?", "is_false": "Does it have a stable isotope?"}},
        {"key": "ancient", "label": "Known since ancient times", "type": "bool",
         "help": "The nine classical elements: carbon, sulfur, iron, copper, silver, tin, gold, mercury, lead. Antimony, zinc, arsenic and platinum were used in antiquity but recognised as elements later, so they answer Unknown.",
         "templates": {"is": "Was it known in ancient times?", "is_false": "Was it discovered in modern times?"}},
        {"key": "odd_symbol", "label": "Symbol not from its English name", "type": "bool",
         "help": "Symbols from Latin or German names, like Fe for iron or W for tungsten.",
         "templates": {"is": "Does its symbol come from a non-English name?", "is_false": "Does its symbol come from its English name?"}},
    ],
    "columns": ["name", "atomic_number", "period", "block", "category", "state", "radioactive", "ancient", "odd_symbol"],
}

#             Z  per blk category                state  rad anc odd
ROWS = """
Hydrogen   |  1 | 1 | s | Other nonmetal        | Gas    | N | N | N
Helium     |  2 | 1 | s | Noble gas             | Gas    | N | N | N
Lithium    |  3 | 2 | s | Alkali metal          | Solid  | N | N | N
Beryllium  |  4 | 2 | s | Alkaline earth metal  | Solid  | N | N | N
Boron      |  5 | 2 | p | Metalloid             | Solid  | N | N | N
Carbon     |  6 | 2 | p | Other nonmetal        | Solid  | N | Y | N
Nitrogen   |  7 | 2 | p | Other nonmetal        | Gas    | N | N | N
Oxygen     |  8 | 2 | p | Other nonmetal        | Gas    | N | N | N
Fluorine   |  9 | 2 | p | Halogen               | Gas    | N | N | N
Neon       | 10 | 2 | p | Noble gas             | Gas    | N | N | N
Sodium     | 11 | 3 | s | Alkali metal          | Solid  | N | N | Y
Magnesium  | 12 | 3 | s | Alkaline earth metal  | Solid  | N | N | N
Aluminium  | 13 | 3 | p | Post-transition metal | Solid  | N | N | N
Silicon    | 14 | 3 | p | Metalloid             | Solid  | N | N | N
Phosphorus | 15 | 3 | p | Other nonmetal        | Solid  | N | N | N
Sulfur     | 16 | 3 | p | Other nonmetal        | Solid  | N | Y | N
Chlorine   | 17 | 3 | p | Halogen               | Gas    | N | N | N
Argon      | 18 | 3 | p | Noble gas             | Gas    | N | N | N
Potassium  | 19 | 4 | s | Alkali metal          | Solid  | N | N | Y
Calcium    | 20 | 4 | s | Alkaline earth metal  | Solid  | N | N | N
Titanium   | 22 | 4 | d | Transition metal      | Solid  | N | N | N
Chromium   | 24 | 4 | d | Transition metal      | Solid  | N | N | N
Manganese  | 25 | 4 | d | Transition metal      | Solid  | N | N | N
Iron       | 26 | 4 | d | Transition metal      | Solid  | N | Y | Y
Cobalt     | 27 | 4 | d | Transition metal      | Solid  | N | N | N
Nickel     | 28 | 4 | d | Transition metal      | Solid  | N | N | N
Copper     | 29 | 4 | d | Transition metal      | Solid  | N | Y | Y
Zinc       | 30 | 4 | d | Transition metal      | Solid  | N | - | N
Gallium    | 31 | 4 | p | Post-transition metal | Solid  | N | N | N
Arsenic    | 33 | 4 | p | Metalloid             | Solid  | N | - | N
Bromine    | 35 | 4 | p | Halogen               | Liquid | N | N | N
Krypton    | 36 | 4 | p | Noble gas             | Gas    | N | N | N
Rubidium   | 37 | 5 | s | Alkali metal          | Solid  | N | N | N
Strontium  | 38 | 5 | s | Alkaline earth metal  | Solid  | N | N | N
Silver     | 47 | 5 | d | Transition metal      | Solid  | N | Y | Y
Tin        | 50 | 5 | p | Post-transition metal | Solid  | N | Y | Y
Antimony   | 51 | 5 | p | Metalloid             | Solid  | N | - | Y
Iodine     | 53 | 5 | p | Halogen               | Solid  | N | N | N
Xenon      | 54 | 5 | p | Noble gas             | Gas    | N | N | N
Caesium    | 55 | 6 | s | Alkali metal          | Solid  | N | N | N
Barium     | 56 | 6 | s | Alkaline earth metal  | Solid  | N | N | N
Neodymium  | 60 | 6 | f | Lanthanide            | Solid  | N | N | N
Tungsten   | 74 | 6 | d | Transition metal      | Solid  | N | N | Y
Platinum   | 78 | 6 | d | Transition metal      | Solid  | N | - | N
Gold       | 79 | 6 | d | Transition metal      | Solid  | N | Y | Y
Mercury    | 80 | 6 | d | Transition metal      | Liquid | N | Y | Y
Lead       | 82 | 6 | p | Post-transition metal | Solid  | N | Y | Y
Polonium   | 84 | 6 | p | -                     | Solid  | Y | N | N
Radon      | 86 | 6 | p | Noble gas             | Gas    | Y | N | N
Radium     | 88 | 7 | s | Alkaline earth metal  | Solid  | Y | N | N
Uranium    | 92 | 7 | f | Actinide              | Solid  | Y | N | N
Plutonium  | 94 | 7 | f | Actinide              | Solid  | Y | N | N
"""
