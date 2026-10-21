"""Evidence for the Scientists pack, checked 2026-10-18.

Birth year and birthplace from each article's infobox; fields from the
article's short description and lead; Nobel prizes from Wikipedia's lists
of laureates in Physics, Chemistry and Physiology or Medicine.
"""
import json
import pathlib
import urllib.parse

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"
ARTICLE = {"Homi Bhabha": "Homi J. Bhabha"}

# name: (birth year, birthplace as the infobox gives it -> today's country, fields, nobel, woman)
P = {
    "Nicolaus Copernicus": (1473, "Poland", "Astronomy", "N", "N"),           # Thorn (Toruń)
    "Galileo Galilei": (1564, "Italy", "Physics;Astronomy", "N", "N"),        # Pisa
    "Johannes Kepler": (1571, "Germany", "Astronomy;Mathematics", "N", "N"),  # Weil der Stadt
    "Isaac Newton": (1643, "United Kingdom", "Physics;Mathematics;Astronomy", "N", "N"),  # 1642 Old Style
    "Leonhard Euler": (1707, "Switzerland", "Mathematics", "N", "N"),
    "Antoine Lavoisier": (1743, "France", "Chemistry", "N", "N"),
    "Carl Friedrich Gauss": (1777, "Germany", "Mathematics", "N", "N"),       # Brunswick
    "Michael Faraday": (1791, "United Kingdom", "Physics;Chemistry", "N", "N"),
    "Ada Lovelace": (1815, "United Kingdom", "Mathematics;Computer science", "N", "Y"),
    "Charles Darwin": (1809, "United Kingdom", "Biology", "N", "N"),
    "Gregor Mendel": (1822, "Czech Republic", "Biology", "N", "N"),           # Heinzendorf, Silesia (now Hynčice)
    "Louis Pasteur": (1822, "France", "Chemistry;Biology", "N", "N"),
    "James Clerk Maxwell": (1831, "United Kingdom", "Physics", "N", "N"),
    "Dmitri Mendeleev": (1834, "Russia", "Chemistry", "N", "N"),
    "Nikola Tesla": (1856, "Croatia", "Engineering", "N", "N"),               # Smiljan, now Croatia
    "Max Planck": (1858, "Germany", "Physics", "Y", "N"),
    "Marie Curie": (1867, "Poland", "Physics;Chemistry", "Y", "Y"),           # Warsaw
    "Ernest Rutherford": (1871, "New Zealand", "Physics;Chemistry", "Y", "N"),
    "Albert Einstein": (1879, "Germany", "Physics", "Y", "N"),                # Ulm
    "Alexander Fleming": (1881, "United Kingdom", "Biology;Medicine", "Y", "N"),
    "Niels Bohr": (1885, "Denmark", "Physics", "Y", "N"),
    "Erwin Schrödinger": (1887, "Austria", "Physics", "Y", "N"),              # Vienna
    "Srinivasa Ramanujan": (1887, "India", "Mathematics", "N", "N"),
    "C. V. Raman": (1888, "India", "Physics", "Y", "N"),
    "Edwin Hubble": (1889, "United States", "Astronomy", "N", "N"),
    "Satyendra Nath Bose": (1894, "India", "Physics", "N", "N"),
    "Enrico Fermi": (1901, "Italy", "Physics", "Y", "N"),
    "Werner Heisenberg": (1901, "Germany", "Physics", "Y", "N"),
    "Linus Pauling": (1901, "United States", "Chemistry", "Y", "N"),
    "Paul Dirac": (1902, "United Kingdom", "Physics", "Y", "N"),
    "Barbara McClintock": (1902, "United States", "Biology", "Y", "Y"),
    "Robert Oppenheimer": (1904, "United States", "Physics", "N", "N"),
    "Homi Bhabha": (1909, "India", "Physics", "N", "N"),
    "Alan Turing": (1912, "United Kingdom", "Mathematics;Computer science", "N", "N"),
    "Richard Feynman": (1918, "United States", "Physics", "Y", "N"),
    "Katherine Johnson": (1918, "United States", "Mathematics", "N", "Y"),
    "Vikram Sarabhai": (1919, "India", "Physics;Astronomy", "N", "N"),
    "Rosalind Franklin": (1920, "United Kingdom", "Chemistry", "N", "Y"),
    "Tu Youyou": (1930, "China", "Medicine;Chemistry", "Y", "Y"),
    "Jane Goodall": (1934, "United Kingdom", "Biology", "N", "Y"),
    "Carl Sagan": (1934, "United States", "Astronomy", "N", "N"),
    "Stephen Hawking": (1942, "United Kingdom", "Physics", "N", "N"),
    "Venkatraman Ramakrishnan": (1952, "India", "Biology;Chemistry", "Y", "N"),
}


def _url(name):
    t = ARTICLE.get(name, name)
    f = pathlib.Path(__file__).resolve().parents[2] / "data" / "cache" / "wiki" / (urllib.parse.quote(t.replace(" ", "_"), safe="") + ".json")
    return json.loads(f.read_text())["url"] if f.exists() else W + urllib.parse.quote(t.replace(" ", "_"))


ITEM_URL = {n: _url(n) for n in P}
PER_ITEM = ("field", "born_in", "century", "woman")
EXPECT = {
    "century": {n: str(v[0] // 100 * 100) for n, v in P.items()},
    "born_in": {n: v[1] for n, v in P.items()},
    "field": {n: v[2] for n, v in P.items()},
    "nobel": {n: v[3] for n, v in P.items()},
    "woman": {n: v[4] for n, v in P.items()},
}
SOURCES = {"nobel": [W + "List_of_Nobel_laureates_in_Physics", W + "List_of_Nobel_laureates_in_Chemistry"]}
NOTES = {
    ("Ernest Rutherford", "field"): "Added Chemistry: his Nobel Prize (1908) was in Chemistry; the article calls him a physicist and chemist.",
    ("Tu Youyou", "field"): "Added Chemistry: the article describes her as a pharmaceutical chemist.",
    ("Vikram Sarabhai", "field"): "Added Astronomy: the article describes him as a physicist and astronomer.",
    ("Isaac Newton", "century"): "Born 25 December 1642 (Old Style) / 4 January 1643 (New Style): 1600s either way.",
    ("Gregor Mendel", "born_in"): "Born in Heinzendorf, Austrian Silesia: today Hynčice, Czech Republic.",
    ("Nikola Tesla", "born_in"): "Born in Smiljan, Austrian Empire: today Croatia.",
    ("Marie Curie", "born_in"): "Born in Warsaw (then Russian Empire): today Poland.",
    ("Tu Youyou", "nobel"): "Nobel Prize in Physiology or Medicine 2015 (List of Nobel laureates in Physiology or Medicine).",
    ("Alexander Fleming", "nobel"): "Nobel Prize in Physiology or Medicine 1945.",
    ("Barbara McClintock", "nobel"): "Nobel Prize in Physiology or Medicine 1983.",
    ("Rosalind Franklin", "nobel"): "Not a laureate (died 1958, before the 1962 prize for DNA's structure).",
}
RULES = {
    "field": "What they are best known for, following the article's description; questions ask 'best known for'.",
    "born_in": "Birthplace from the infobox, mapped to today's country.",
    "nobel": "Any Nobel Prize (Physics, Chemistry, Physiology or Medicine; Pauling's second was Peace).",
}
