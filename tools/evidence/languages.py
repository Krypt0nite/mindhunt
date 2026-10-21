"""Evidence for the Programming Languages pack, checked 2026-10-18.

First release, developer/designer, typing and paradigm from each language's
English Wikipedia infobox and lead (cached in data/cache/wiki/); origins from
the history sections where the lead doesn't say.
"""
import json
import pathlib
import urllib.parse

CHECKED = "2026-10-18"
W = "https://en.wikipedia.org/wiki/"
TITLE = {"Fortran": "Fortran", "COBOL": "COBOL", "Lisp": "Lisp (programming language)", "C": "C (programming language)",
         "Pascal": "Pascal (programming language)", "Ada": "Ada (programming language)", "C++": "C++",
         "Objective-C": "Objective-C", "Erlang": "Erlang (programming language)", "Perl": "Perl", "MATLAB": "MATLAB",
         "Bash": "Bash (Unix shell)", "Haskell": "Haskell", "Python": "Python (programming language)",
         "Lua": "Lua (programming language)", "R": "R (programming language)", "Java": "Java (programming language)",
         "JavaScript": "JavaScript", "Ruby": "Ruby (programming language)", "PHP": "PHP",
         "C#": "C Sharp (programming language)", "Scala": "Scala (programming language)", "Clojure": "Clojure",
         "Go": "Go (programming language)", "Rust": "Rust (programming language)", "Kotlin": "Kotlin (programming language)",
         "TypeScript": "TypeScript", "Elixir": "Elixir (programming language)", "Julia": "Julia (programming language)",
         "Swift": "Swift (programming language)", "Zig": "Zig (programming language)"}

# name: (typing, runs, first released, gc, mainly functional, created at a company)
L = {
    "Fortran": ("Static", "Compiled to machine code", 1957, "N", "N", "Y"),     # John Backus and IBM
    "COBOL": ("Static", "Compiled to machine code", 1960, "N", "N", "N"),       # CODASYL committee; design began 1959
    "Lisp": ("Dynamic", "-", 1960, "Y", "Y", "N"),                              # McCarthy, MIT
    "C": ("Static", "Compiled to machine code", 1972, "N", "N", "Y"),           # Bell Labs
    "Pascal": ("Static", "Compiled to machine code", 1970, "N", "N", "N"),      # Wirth, ETH Zurich
    "Ada": ("Static", "Compiled to machine code", 1980, "N", "N", "-"),
    "C++": ("Static", "Compiled to machine code", 1985, "N", "N", "Y"),         # Stroustrup, Bell Labs
    "Objective-C": ("Static", "Compiled to machine code", 1984, "-", "N", "Y"),
    "Erlang": ("Dynamic", "On a virtual machine", 1986, "Y", "Y", "Y"),         # Ericsson
    "Perl": ("Dynamic", "Interpreted or JIT", 1987, "Y", "N", "N"),             # Larry Wall
    "MATLAB": ("Dynamic", "Interpreted or JIT", 1979, "Y", "N", "N"),
    "Bash": ("Dynamic", "Interpreted or JIT", 1989, "-", "N", "N"),             # GNU Project
    "Haskell": ("Static", "Compiled to machine code", 1990, "Y", "Y", "N"),     # committee; GHC compiles natively
    "Python": ("Dynamic", "Interpreted or JIT", 1991, "Y", "N", "N"),
    "Lua": ("Dynamic", "Interpreted or JIT", 1993, "Y", "N", "N"),
    "R": ("Dynamic", "Interpreted or JIT", 1993, "Y", "N", "N"),
    "Java": ("Static", "On a virtual machine", 1995, "Y", "N", "Y"),            # Sun Microsystems
    "JavaScript": ("Dynamic", "Interpreted or JIT", 1995, "Y", "N", "Y"),       # Netscape
    "Ruby": ("Dynamic", "Interpreted or JIT", 1995, "Y", "N", "N"),
    "PHP": ("Dynamic", "Interpreted or JIT", 1995, "Y", "N", "N"),
    "C#": ("Static", "On a virtual machine", 2000, "Y", "N", "Y"),             # Microsoft
    "Scala": ("Static", "On a virtual machine", 2004, "Y", "N", "N"),           # EPFL
    "Clojure": ("Dynamic", "On a virtual machine", 2007, "Y", "Y", "N"),
    "Go": ("Static", "Compiled to machine code", 2009, "Y", "N", "Y"),          # Google
    "Rust": ("Static", "Compiled to machine code", 2012, "N", "N", "Y"),
    "Kotlin": ("Static", "On a virtual machine", 2011, "Y", "N", "Y"),          # JetBrains
    "TypeScript": ("Static", "-", 2012, "Y", "N", "Y"),                         # Microsoft
    "Elixir": ("Dynamic", "On a virtual machine", 2012, "Y", "Y", "Y"),
    "Julia": ("Dynamic", "Interpreted or JIT", 2012, "Y", "N", "N"),
    "Swift": ("Static", "Compiled to machine code", 2014, "Y", "N", "Y"),       # Apple
    "Zig": ("Static", "Compiled to machine code", 2016, "N", "N", "N"),
}


def _url(name):
    t = TITLE[name]
    f = pathlib.Path(__file__).resolve().parents[2] / "data" / "cache" / "wiki" / (urllib.parse.quote(t.replace(" ", "_"), safe="") + ".json")
    return json.loads(f.read_text())["url"] if f.exists() else W + urllib.parse.quote(t.replace(" ", "_"))


ITEM_URL = {n: _url(n) for n in L}
PER_ITEM = ("typing", "runs", "decade", "gc", "functional", "company")
EXPECT = {k: {n: (str(v[i] // 10 * 10) if k == "decade" else v[i]) for n, v in L.items()}
          for i, k in enumerate(["typing", "runs", "decade", "gc", "functional", "company"])}
NOTES = {
    ("COBOL", "decade"): "Design began in 1959 (CODASYL); the infobox gives 1960 as first appeared.",
    ("Lisp", "decade"): "Designed by McCarthy in 1958; the infobox gives 1960 as first appeared.",
    ("Lisp", "runs"): "A family of languages; major implementations both interpret and compile to native code: Unknown.",
    ("MATLAB", "decade"): "First version written by Cleve Moler at the University of New Mexico; first publicly shown and distributed free in 1979. The commercial MathWorks product came in 1984.",
    ("MATLAB", "company"): "First built at the University of New Mexico (MathWorks was founded in 1984 to sell it).",
    ("Ada", "company"): "Designed by a Honeywell team under contract to the US Department of Defense: a company built it, the government commissioned it. Unknown.",
    ("Objective-C", "company"): "Created by Brad Cox and Tom Love 'at their company Productivity Products International (PPI)'.",
    ("Elixir", "company"): "'José Valim created the Elixir programming language as a research and development project at Plataformatec' (a company).",
    ("Rust", "company"): "'Graydon Hoare created Rust in 2006 while working at Mozilla, which officially sponsored the project in 2009.'",
    ("Perl", "company"): "Developed by Larry Wall (the infobox lists him as developer).",
    ("Haskell", "runs"): "Main implementation is the Glasgow Haskell Compiler, which compiles to native code.",
}
RULES = {
    "decade": "Infobox 'first appeared' (or first public release where the infobox gives only the commercial date).",
    "company": "Where it was first built. Mixed cases (Ada) are Unknown.",
    "runs": "Usual implementation; language families with both (Lisp) are Unknown.",
}
