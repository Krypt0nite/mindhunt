"""Evidence for the Economists pack, checked 2026-10-18.

Birth year and place from each article's infobox; school from the infobox's
'school or tradition' and the lead (plus the Classical economics and
Monetarism articles); Nobel from Wikipedia's list of laureates.
"""
import json
import pathlib
import urllib.parse

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"

P = {  # name: (birth year, birthplace -> today's country, school or "-", nobel year or 0, woman)
    "Adam Smith": (1723, "United Kingdom", "Classical", 0, "N"),               # Kirkcaldy, Scotland
    "Thomas Malthus": (1766, "United Kingdom", "Classical", 0, "N"),
    "David Ricardo": (1772, "United Kingdom", "Classical", 0, "N"),
    "John Stuart Mill": (1806, "United Kingdom", "Classical", 0, "N"),
    "Karl Marx": (1818, "Germany", "Marxian", 0, "N"),                          # Trier, Prussia
    "Alfred Marshall": (1842, "United Kingdom", "Neoclassical", 0, "N"),
    "John Maynard Keynes": (1883, "United Kingdom", "Keynesian", 0, "N"),
    "Friedrich Hayek": (1899, "Austria", "Austrian", 1974, "N"),
    "John Kenneth Galbraith": (1908, "Canada", "-", 0, "N"),
    "Milton Friedman": (1912, "United States", "Monetarist", 1976, "N"),
    "Paul Samuelson": (1915, "United States", "Keynesian", 1970, "N"),
    "Amartya Sen": (1933, "India", "Development", 1998, "N"),
    "Elinor Ostrom": (1933, "United States", "Institutional", 2009, "Y"),
    "Joseph Stiglitz": (1943, "United States", "Keynesian", 2001, "N"),
    "Richard Thaler": (1945, "United States", "Behavioural", 2017, "N"),
    "Paul Krugman": (1953, "United States", "Keynesian", 2008, "N"),
    "Abhijit Banerjee": (1961, "India", "Development", 2019, "N"),
    "Thomas Piketty": (1971, "France", "-", 0, "N"),
    "Esther Duflo": (1972, "France", "Development", 2019, "Y"),
    "Raghuram Rajan": (1963, "India", "-", 0, "N"),
}


def _url(name):
    f = pathlib.Path(__file__).resolve().parents[2] / "data" / "cache" / "wiki" / (urllib.parse.quote(name.replace(" ", "_"), safe="") + ".json")
    return json.loads(f.read_text())["url"] if f.exists() else W + urllib.parse.quote(name.replace(" ", "_"))


ITEM_URL = {n: _url(n) for n in P}
PER_ITEM = ("born_in", "century", "school", "woman")
EXPECT = {
    "born_in": {n: v[1] for n, v in P.items()},
    "century": {n: str(v[0] // 100 * 100) for n, v in P.items()},
    "school": {n: v[2] for n, v in P.items()},
    "nobel": {n: ("Y" if v[3] else "N") for n, v in P.items()},
    "woman": {n: v[4] for n, v in P.items()},
}
SOURCES = {"nobel": [W + "List_of_Nobel_Memorial_Prize_laureates_in_Economic_Sciences"],
           "school": [W + "Classical_economics", W + "Monetarism"]}
NOTES = {
    ("John Kenneth Galbraith", "school"): "Was Keynesian; the infobox says Institutional and the lead says 'post-Keynesian from an institutionalist perspective', so neither single school fits: Unknown.",
    ("Milton Friedman", "school"): "The Monetarism article: 'mainly associated with the work of Milton Friedman'. (His infobox also lists the Chicago school.)",
    ("Paul Krugman", "school"): "Article: 'Krugman identifies as a Keynesian and a saltwater economist'.",
    ("Paul Samuelson", "school"): "Infobox: Neo-Keynesian economics.",
    ("Joseph Stiglitz", "school"): "Infobox: Neo-Keynesian economics.",
    ("Elinor Ostrom", "school"): "Infobox: New institutional economics.",
    ("Amartya Sen", "school"): "Known for welfare and development economics (capability approach).",
    **{(n, "school"): "The Classical economics article names Smith, Malthus, Ricardo and Mill among its main thinkers."
       for n in ("Adam Smith", "Thomas Malthus", "David Ricardo", "John Stuart Mill")},
    ("Friedrich Hayek", "century"): "Born 8 May 1899: the 1800s.",
}
RULES = {
    "school": "One school per economist where sources agree; where they don't (Galbraith, Piketty, Rajan) the game answers Unknown.",
    "nobel": "Nobel Memorial Prize in Economic Sciences (from 1969).",
    "born_in": "Birthplace from the infobox, mapped to today's country.",
}
