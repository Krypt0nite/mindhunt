"""Evidence for the Footballers pack, checked 2026-10-18.

Position and senior international debut from each player's infobox; captaincy
and stronger foot from the article text (cached in data/cache/wiki_full/);
World Cup and Ballon d'Or from the winners' lists.
"""
import json
import pathlib
import urllib.parse

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"
TITLE = {"Ronaldo Nazário": "Ronaldo (Brazilian footballer)", "Xavi": "Xavi (footballer, born 1980)"}

# name: (country, continent, position, first senior cap year, world cup, ballon d'or, captain)
P = {
    "Pelé": ("Brazil", "South America", "Forward", 1957, "Y", "N", "N"),
    "Johan Cruyff": ("Netherlands", "Europe", "Forward", 1966, "N", "Y", "Y"),
    "Franz Beckenbauer": ("Germany", "Europe", "Defender", 1965, "Y", "Y", "Y"),
    "Michel Platini": ("France", "Europe", "Midfielder", 1976, "N", "Y", "Y"),
    "Diego Maradona": ("Argentina", "South America", "-", 1977, "Y", "N", "Y"),
    "Lothar Matthäus": ("Germany", "Europe", "Midfielder", 1980, "Y", "Y", "Y"),
    "Paolo Maldini": ("Italy", "Europe", "Defender", 1988, "N", "N", "Y"),
    "George Weah": ("Liberia", "Africa", "Forward", 1986, "N", "Y", "-"),
    "Zinedine Zidane": ("France", "Europe", "Midfielder", 1994, "Y", "Y", "Y"),
    "Ronaldo Nazário": ("Brazil", "South America", "Forward", 1994, "Y", "Y", "N"),
    "Ronaldinho": ("Brazil", "South America", "-", 1999, "Y", "Y", "-"),
    "David Beckham": ("England", "Europe", "Midfielder", 1996, "N", "N", "Y"),
    "Thierry Henry": ("France", "Europe", "Forward", 1997, "Y", "N", "Y"),
    "Fabio Cannavaro": ("Italy", "Europe", "Defender", 1997, "Y", "Y", "Y"),
    "Gianluigi Buffon": ("Italy", "Europe", "Goalkeeper", 1997, "Y", "N", "Y"),
    "Bhaichung Bhutia": ("India", "Asia", "Forward", 1995, "N", "N", "Y"),
    "Iker Casillas": ("Spain", "Europe", "Goalkeeper", 2000, "Y", "N", "Y"),
    "Xavi": ("Spain", "Europe", "Midfielder", 2000, "Y", "N", "N"),
    "Zlatan Ibrahimović": ("Sweden", "Europe", "Forward", 2001, "N", "N", "Y"),
    "Didier Drogba": ("Ivory Coast", "Africa", "Forward", 2002, "N", "N", "Y"),
    "Kaká": ("Brazil", "South America", "Midfielder", 2002, "Y", "Y", "N"),
    "Cristiano Ronaldo": ("Portugal", "Europe", "Forward", 2003, "N", "Y", "Y"),
    "Wayne Rooney": ("England", "Europe", "Forward", 2003, "N", "N", "Y"),
    "Lionel Messi": ("Argentina", "South America", "Forward", 2005, "Y", "Y", "Y"),
    "Sergio Ramos": ("Spain", "Europe", "Defender", 2005, "Y", "N", "Y"),
    "Sunil Chhetri": ("India", "Asia", "Forward", 2005, "N", "N", "Y"),
    "Luka Modrić": ("Croatia", "Europe", "Midfielder", 2006, "N", "Y", "Y"),
    "Andrés Iniesta": ("Spain", "Europe", "Midfielder", 2006, "Y", "N", "N"),
    "Robert Lewandowski": ("Poland", "Europe", "Forward", 2008, "N", "N", "Y"),
    "Manuel Neuer": ("Germany", "Europe", "Goalkeeper", 2009, "Y", "N", "Y"),
    "Neymar": ("Brazil", "South America", "-", 2010, "N", "N", "Y"),
    "Son Heung-min": ("South Korea", "Asia", "Forward", 2010, "N", "N", "Y"),
    "Mohamed Salah": ("Egypt", "Africa", "Forward", 2011, "N", "N", "Y"),
    "Harry Kane": ("England", "Europe", "Forward", 2015, "N", "N", "Y"),
    "Virgil van Dijk": ("Netherlands", "Europe", "Defender", 2015, "N", "N", "Y"),
    "Kylian Mbappé": ("France", "Europe", "Forward", 2017, "Y", "N", "Y"),
    "Erling Haaland": ("Norway", "Europe", "Forward", 2019, "N", "N", "N"),
}
# Stronger foot, only where the article says so ("left-footed", "right foot", ...).
FOOT = {"Michel Platini": "Right", "Diego Maradona": "Left", "Paolo Maldini": "Right", "Zinedine Zidane": "Right",
        "Ronaldo Nazário": "Right", "Ronaldinho": "Right", "David Beckham": "Right", "Zlatan Ibrahimović": "Right",
        "Kaká": "Right", "Cristiano Ronaldo": "Right", "Lionel Messi": "Left", "Mohamed Salah": "Left",
        "Virgil van Dijk": "Right"}


def _url(name):
    t = TITLE.get(name, name)
    f = pathlib.Path(__file__).resolve().parents[2] / "data" / "cache" / "wiki_full" / (urllib.parse.quote(t.replace(" ", "_"), safe="") + ".json")
    return json.loads(f.read_text())["url"] if f.exists() else W + urllib.parse.quote(t.replace(" ", "_"))


ITEM_URL = {n: _url(n) for n in P}
PER_ITEM = ("country", "continent", "position", "foot", "debut", "captain")
EXPECT = {
    "country": {n: v[0] for n, v in P.items()},
    "continent": {n: v[1] for n, v in P.items()},
    "position": {n: v[2] for n, v in P.items()},
    "debut": {n: str(v[3] // 10 * 10) for n, v in P.items()},
    "world_cup": {n: v[4] for n, v in P.items()},
    "ballon_dor": {n: v[5] for n, v in P.items()},
    "captain": {n: v[6] for n, v in P.items()},
    "foot": FOOT,
}
SOURCES = {"world_cup": [W + "List_of_FIFA_World_Cup_winning_players"],
           "ballon_dor": ["https://uefa.com/insideuefa/news/0287-195e642735da-0594342b9554-1000--history-of-the-ballon-d-or-all-the-winners"]}
NOTES = {
    ("Ronaldinho", "position"): "Infobox: 'Attacking midfielder, left winger' (midfield first, forward second): Unknown.",
    ("Neymar", "position"): "Article: 'plays as an attacking midfielder or a forward': Unknown.",
    ("Diego Maradona", "position"): "Infobox: 'Attacking midfielder, second striker': Unknown.",
    ("Ronaldinho", "captain"): "Captained Brazil at the 2005 Confederations Cup (and the 2008 Olympic team) but was never the regular captain: occasional, so Unknown.",
    ("Thierry Henry", "captain"): "AFP, 6 Sep 2008: 'New France captain Thierry Henry', with the armband while Vieira was injured (https://today.thefinancialexpress.com.bd/print/henry-happy-to-play-captains-part).",
    ("Johan Cruyff", "captain"): "Captained the Netherlands in the 1974 World Cup final (https://www.11v11.com/matches/germany-v-netherlands-07-july-1974-232910/).",
    ("George Weah", "captain"): "Liberia's star and later player-coach; no source found calling him the national team's captain (SI, 2001: 'technical director and coach as well as its star player'). Unknown.",
    ("Zinedine Zidane", "captain"): "Article: 'Zidane came out of retirement and was immediately reinstated as team captain' (France, 2005).",
    ("Bhaichung Bhutia", "captain"): "Article: 'The next successful SAFF Championship was in 2005 where Bhutia was captain'.",
    ("Neymar", "captain"): "Article: Dunga 'decided to make Neymar captain' (2015).",
    ("Xavi", "captain"): "Club captain at Barcelona and Al Sadd; no national captaincy in the article.",
    ("Andrés Iniesta", "captain"): "Only Spain U21 captaincy in the article.",
    ("Erling Haaland", "captain"): "Club captaincy only in the article.",
    ("Diego Maradona", "foot"): "Article: 'Maradona was dominantly left-footed'.",
}
RULES = {
    "position": "First position category in the infobox; where it leads with attacking midfield and adds a forward role, Unknown.",
    "debut": "First senior cap year from the infobox (youth teams excluded).",
    "captain": "Regular senior national team captain; occasional or unsourced captaincy is Unknown.",
    "foot": "Only checked where the article states it; the rest are marked not checked.",
}
