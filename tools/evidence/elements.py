"""Evidence for the Chemical Elements pack, checked 2026-10-18."""

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"

# name | Z | period | block | phase at STP | origin   (List of chemical elements)
TABLE = """
Hydrogen 1 1 s Gas P|Helium 2 1 s Gas P|Lithium 3 2 s Solid P|Beryllium 4 2 s Solid P|Boron 5 2 p Solid P
Carbon 6 2 p Solid P|Nitrogen 7 2 p Gas P|Oxygen 8 2 p Gas P|Fluorine 9 2 p Gas P|Neon 10 2 p Gas P
Sodium 11 3 s Solid P|Magnesium 12 3 s Solid P|Aluminium 13 3 p Solid P|Silicon 14 3 p Solid P
Phosphorus 15 3 p Solid P|Sulfur 16 3 p Solid P|Chlorine 17 3 p Gas P|Argon 18 3 p Gas P|Potassium 19 4 s Solid P
Calcium 20 4 s Solid P|Titanium 22 4 d Solid P|Chromium 24 4 d Solid P|Manganese 25 4 d Solid P|Iron 26 4 d Solid P
Cobalt 27 4 d Solid P|Nickel 28 4 d Solid P|Copper 29 4 d Solid P|Zinc 30 4 d Solid P|Gallium 31 4 p Solid P
Arsenic 33 4 p Solid P|Bromine 35 4 p Liquid P|Krypton 36 4 p Gas P|Rubidium 37 5 s Solid P|Strontium 38 5 s Solid P
Silver 47 5 d Solid P|Tin 50 5 p Solid P|Antimony 51 5 p Solid P|Iodine 53 5 p Solid P|Xenon 54 5 p Gas P
Caesium 55 6 s Solid P|Barium 56 6 s Solid P|Neodymium 60 6 f Solid P|Tungsten 74 6 d Solid P|Platinum 78 6 d Solid P
Gold 79 6 d Solid P|Mercury 80 6 d Liquid P|Lead 82 6 p Solid P|Polonium 84 6 p Solid D|Radon 86 6 p Gas D
Radium 88 7 s Solid D|Uranium 92 7 f Solid P|Plutonium 94 7 f Solid D
"""
ROWS = [r.split() for r in TABLE.replace("\n", "|").split("|") if r.strip()]
NAMES = [r[0] for r in ROWS]


def yes(*names):
    return {n: ("Y" if n in names else "N") for n in NAMES}


CATEGORY = {}
for names, cat in [
    ("Lithium Sodium Potassium Rubidium Caesium", "Alkali metal"),
    ("Beryllium Magnesium Calcium Strontium Barium Radium", "Alkaline earth metal"),
    ("Titanium Chromium Manganese Iron Cobalt Nickel Copper Zinc Silver Tungsten Platinum Gold Mercury", "Transition metal"),
    ("Aluminium Gallium Tin Lead", "Post-transition metal"),
    ("Boron Silicon Arsenic Antimony", "Metalloid"),
    ("Hydrogen Carbon Nitrogen Oxygen Phosphorus Sulfur", "Other nonmetal"),
    ("Fluorine Chlorine Bromine Iodine", "Halogen"),
    ("Helium Neon Argon Krypton Xenon Radon", "Noble gas"),
    ("Neodymium", "Lanthanide"), ("Uranium Plutonium", "Actinide"),
]:
    for n in names.split():
        CATEGORY[n] = cat
CATEGORY["Polonium"] = "-"  # post-transition metal on one page, sometimes a metalloid on another

EXPECT = {
    "atomic_number": {r[0]: r[1] for r in ROWS},
    "period": {r[0]: r[2] for r in ROWS},
    "block": {r[0]: r[3] for r in ROWS},
    "state": {r[0]: r[4] for r in ROWS},
    "category": CATEGORY,
    "radioactive": yes("Polonium", "Radon", "Radium", "Uranium", "Plutonium"),
    "ancient": {**yes("Carbon", "Sulfur", "Iron", "Copper", "Silver", "Tin", "Gold", "Mercury", "Lead"),
                **{n: "-" for n in ("Antimony", "Zinc", "Arsenic", "Platinum")}},
    "odd_symbol": yes("Sodium", "Potassium", "Iron", "Copper", "Silver", "Tin", "Antimony", "Tungsten", "Gold",
                      "Mercury", "Lead"),
}

SOURCES = {
    "atomic_number": [W + "List_of_chemical_elements"], "period": [W + "List_of_chemical_elements"],
    "block": [W + "List_of_chemical_elements"], "state": [W + "List_of_chemical_elements"],
    "category": [W + "Nonmetal", W + "Transition_metal"],
    "radioactive": [W + "List_of_elements_by_stability_of_isotopes"],
    "ancient": [W + "Timeline_of_chemical_element_discoveries"],
    "odd_symbol": [W + "List_of_chemical_element_name_etymologies"],
}

NOTES = {
    ("Zinc", "category"): "Group 12: a transition metal under IUPAC's 2011 definition (groups 3-12); some books exclude group 12.",
    ("Mercury", "category"): "Group 12: a transition metal under IUPAC's 2011 definition (groups 3-12); some books exclude group 12.",
    ("Polonium", "category"): "Post-transition metal per its Wikipedia page, sometimes counted as a metalloid: Unknown.",
    **{(n, "category"): "Wikipedia groups H, C, N, O, P, S (and Se) as 'unclassified nonmetals'; renamed from 'Reactive nonmetal', which elsewhere includes the halogens."
       for n in ("Hydrogen", "Carbon", "Nitrogen", "Oxygen", "Phosphorus", "Sulfur")},
    **{(n, "ancient"): "Used in antiquity (Wikipedia's 'Antiquity to 1600' table) but not one of the nine classical elements: Unknown."
       for n in ("Antimony", "Zinc", "Arsenic", "Platinum")},
    **{(n, "category"): "Metalloid (one of the six commonly recognised) per the Metalloid article." for n in ("Boron", "Silicon", "Arsenic", "Antimony")},
}

RULES = {
    "category": "Common textbook groups. Group 12 counted as transition metals (IUPAC 2011). Polonium Unknown.",
    "ancient": "The nine elements known in classical antiquity. Elements used in antiquity but recognised later (antimony, zinc, arsenic, platinum) answer Unknown.",
    "radioactive": "No stable isotopes.",
    "state": "At standard temperature and pressure.",
}
