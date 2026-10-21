"""Evidence for the Historical Figures pack, checked 2026-10-18.

Birth years, region and roles come from each person's English Wikipedia
article (lead paragraph and short description, cached in data/cache/wiki/).
Nobel Peace Prize from the list of laureates.
"""
import json
import pathlib
import urllib.parse

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"

# name: (birth year as the article gives it; negative = BCE, region, roles)
PEOPLE = {
    "Gautama Buddha": (-563, "South Asia", "Religious figure;Philosopher"),  # c. 6th-5th century BCE
    "Chandragupta Maurya": (-350, "South Asia", "Ruler;Military leader"),
    "Ashoka": (-304, "South Asia", "Ruler"),
    "Guru Nanak": (1469, "South Asia", "Religious figure"),
    "Akbar": (1542, "South Asia", "Ruler"),
    "Shivaji": (1630, "South Asia", "Ruler;Military leader"),
    "Tipu Sultan": (1751, "South Asia", "Ruler;Military leader"),
    "Rani Lakshmibai": (1828, "South Asia", "Ruler;Military leader;Revolutionary or freedom fighter"),  # 1827-30 or 1835
    "Rabindranath Tagore": (1861, "South Asia", "Writer;Philosopher;Reformer"),
    "Swami Vivekananda": (1863, "South Asia", "Religious figure;Philosopher;Reformer"),
    "Mahatma Gandhi": (1869, "South Asia", "Political leader;Revolutionary or freedom fighter;Reformer"),
    "Jawaharlal Nehru": (1889, "South Asia", "Political leader;Revolutionary or freedom fighter"),
    "B. R. Ambedkar": (1891, "South Asia", "Reformer;Political leader"),
    "Subhas Chandra Bose": (1897, "South Asia", "Revolutionary or freedom fighter;Military leader;Political leader"),
    "Bhagat Singh": (1907, "South Asia", "Revolutionary or freedom fighter"),
    "Mother Teresa": (1910, "South Asia", "Religious figure"),
    "Indira Gandhi": (1917, "South Asia", "Political leader"),
    "Confucius": (-551, "East Asia", "Philosopher"),
    "Genghis Khan": (1162, "East Asia", "Ruler;Military leader"),
    "Sun Yat-sen": (1866, "East Asia", "Political leader;Revolutionary or freedom fighter"),
    "Mao Zedong": (1893, "East Asia", "Political leader;Military leader;Revolutionary or freedom fighter"),
    "Hammurabi": (-1810, "Middle East", "Ruler"),
    "Saladin": (1137, "Middle East", "Ruler;Military leader"),
    "Socrates": (-470, "Europe", "Philosopher"),
    "Alexander the Great": (-356, "Europe", "Ruler;Military leader"),
    "Julius Caesar": (-100, "Europe", "Ruler;Military leader"),
    "Marco Polo": (1254, "Europe", "Explorer"),
    "Joan of Arc": (1412, "Europe", "Military leader;Religious figure"),
    "Christopher Columbus": (1451, "Europe", "Explorer"),
    "Vasco da Gama": (1460, "Europe", "Explorer"),
    "Martin Luther": (1483, "Europe", "Religious figure;Reformer"),
    "Elizabeth I": (1533, "Europe", "Ruler"),
    "Napoleon Bonaparte": (1769, "Europe", "Ruler;Military leader"),
    "Queen Victoria": (1819, "Europe", "Ruler"),
    "Karl Marx": (1818, "Europe", "Philosopher;Writer"),
    "Winston Churchill": (1874, "Europe", "Political leader;Writer"),
    "Cleopatra": (-69, "Africa", "Ruler"),
    "Mansa Musa": (1280, "Africa", "Ruler"),
    "Kwame Nkrumah": (1909, "Africa", "Political leader;Revolutionary or freedom fighter"),
    "Nelson Mandela": (1918, "Africa", "Political leader;Revolutionary or freedom fighter"),
    "George Washington": (1732, "Americas", "Political leader;Military leader"),
    "Simón Bolívar": (1783, "Americas", "Military leader;Political leader;Revolutionary or freedom fighter"),
    "Abraham Lincoln": (1809, "Americas", "Political leader"),
    "Martin Luther King Jr.": (1929, "Americas", "Reformer;Religious figure"),
}
WOMEN = {"Rani Lakshmibai", "Mother Teresa", "Indira Gandhi", "Joan of Arc", "Elizabeth I", "Queen Victoria", "Cleopatra"}
NOBEL_PEACE = {"Mother Teresa": 1979, "Nelson Mandela": 1993, "Martin Luther King Jr.": 1964}


def bucket(y):
    if y < 500:
        return "Before 500 CE"
    if y < 1500:
        return "500 to 1499"
    if y < 1800:
        return "1500 to 1799"
    return "1800s" if y < 1900 else "1900s"


def _url(name):
    f = pathlib.Path(__file__).resolve().parents[2] / "data" / "cache" / "wiki" / (urllib.parse.quote(name.replace(" ", "_"), safe="") + ".json")
    return json.loads(f.read_text())["url"] if f.exists() else W + urllib.parse.quote(name.replace(" ", "_"))


ITEM_URL = {n: _url(n) for n in PEOPLE}
PER_ITEM = ("born", "region", "role", "woman")

EXPECT = {
    "born": {n: bucket(y) for n, (y, _, _) in PEOPLE.items()},
    "region": {n: r for n, (_, r, _) in PEOPLE.items()},
    "role": {n: roles for n, (_, _, roles) in PEOPLE.items()},
    "woman": {n: ("Y" if n in WOMEN else "N") for n in PEOPLE},
    "nobel_peace": {n: ("Y" if n in NOBEL_PEACE else "N") for n in PEOPLE},
}
SOURCES = {"nobel_peace": [W + "List_of_Nobel_Peace_Prize_laureates",
                           "https://www.nobelprize.org/prizes/facts/facts-on-the-nobel-peace-prize/"]}
NOTES = {
    ("Rabindranath Tagore", "role"): "Added Reformer: the article's lead calls him a social reformer.",
    ("Swami Vivekananda", "role"): "Added Philosopher: the article's lead calls him a philosopher.",
    ("Mao Zedong", "role"): "Added Revolutionary: the article's lead calls him a communist revolutionary.",
    ("Mahatma Gandhi", "nobel_peace"): "Never awarded; the Nobel committee says a posthumous award wasn't allowed (no prize in 1948).",
    ("Rani Lakshmibai", "born"): "Born 1827-30 or 1835; 1800s either way.",
    ("Gautama Buddha", "born"): "c. 6th to 5th century BCE.",
    ("Vasco da Gama", "born"): "c. 1460s.",
}
RULES = {
    "born": "Birth year from the article (approximate dates fall in the same bucket either way).",
    "role": "What the article's lead and short description say they are known for; questions ask 'best known as', so a missing role is the pack's judgement, not a claim they never did it.",
    "region": "Where they lived and did their main work.",
    "nobel_peace": "Nobel Peace Prize (Churchill's Nobel was in Literature).",
}
