SPEC = {
    "id": "programming-languages",
    "version": 2,  # v2: checked 2026-10-18 (tools/evidence/languages.py)
    "draft": False,
    "as_of": "2026-10-18",
    "category": "Technology",
    "title": "Programming Languages",
    "noun": "programming language",
    "description": "From Fortran to Zig: how they run, how they type, who made them.",
    "letters": "Does the language's name start with “{value}”?",
    "attributes": [
        {"key": "typing", "label": "Type checking", "type": "enum", "values": ["Static", "Dynamic"],
         "help": "When types are checked: before the program runs (Static) or while it runs (Dynamic).",
         "templates": {"eq:Static": "Is it statically typed?", "eq:Dynamic": "Is it dynamically typed?"},
         "aliases": {"Static": ["statically typed", "static typing"], "Dynamic": ["dynamically typed", "dynamic typing"]}},
        {"key": "runs", "label": "How it usually runs", "type": "enum",
         "values": ["Compiled to machine code", "On a virtual machine", "Interpreted or JIT"],
         "help": "Its usual implementation. TypeScript is Unknown because it is translated to JavaScript; Lisp because its dialects do both.",
         "templates": {"eq:Compiled to machine code": "Is it usually compiled to machine code?",
                       "eq:On a virtual machine": "Does it usually run on a virtual machine (like the JVM)?",
                       "eq:Interpreted or JIT": "Is it usually interpreted or JIT-compiled?"},
         "aliases": {"Compiled to machine code": ["compiled", "native code", "machine code"],
                     "On a virtual machine": ["virtual machine", "jvm", "bytecode", "vm"],
                     "Interpreted or JIT": ["interpreted", "scripting language", "jit"]}},
        {"key": "decade", "label": "Decade first released", "type": "int", "min": 1950, "max": 2020, "step": 10,
         "templates": {"eq": "Was it first released in the {value}s?", "gt": "Was it first released after the {value}s?",
                       "lt": "Was it first released before the {value}s?",
                       "between": "Was it first released between the {value}s and the {value2}s?"}},
        {"key": "gc", "label": "Garbage collected", "type": "bool",
         "help": "Memory is freed automatically at run time, including reference counting (Swift, Python). Rust and C++ are No.",
         "templates": {"is": "Is it garbage collected?", "is_false": "Do programmers manage memory without a garbage collector?"},
         "aliases": {}},
        {"key": "functional", "label": "Mainly functional", "type": "bool",
         "help": "Designed mainly around functional programming (Haskell, Lisp, Erlang, Elixir, Clojure).",
         "templates": {"is": "Is it mainly a functional language?", "is_false": "Is it not mainly functional?"}},
        {"key": "company", "label": "Created at a company", "type": "bool",
         "help": "First built inside a company (Java at Sun, Go at Google, C at Bell Labs, Elixir at Plataformatec), not a university or individual. Ada (a Honeywell team, commissioned by the US government) is Unknown.",
         "templates": {"is": "Was it created at a company?", "is_false": "Was it created outside a company?"}},
    ],
    "columns": ["name", "typing", "runs", "decade", "gc", "functional", "company"],
}

#              typing  runs                      decade gc fn co
ROWS = """
Fortran     | Static  | Compiled to machine code | 1950 | N | N | Y
COBOL       | Static  | Compiled to machine code | 1960 | N | N | N
Lisp        | Dynamic | -                        | 1960 | Y | Y | N
C           | Static  | Compiled to machine code | 1970 | N | N | Y
Pascal      | Static  | Compiled to machine code | 1970 | N | N | N
Ada         | Static  | Compiled to machine code | 1980 | N | N | -
C++         | Static  | Compiled to machine code | 1980 | N | N | Y
Objective-C | Static  | Compiled to machine code | 1980 | - | N | Y
Erlang      | Dynamic | On a virtual machine     | 1980 | Y | Y | Y
Perl        | Dynamic | Interpreted or JIT       | 1980 | Y | N | N
MATLAB      | Dynamic | Interpreted or JIT       | 1970 | Y | N | N
Bash        | Dynamic | Interpreted or JIT       | 1980 | - | N | N
Haskell     | Static  | Compiled to machine code | 1990 | Y | Y | N
Python      | Dynamic | Interpreted or JIT       | 1990 | Y | N | N
Lua         | Dynamic | Interpreted or JIT       | 1990 | Y | N | N
R           | Dynamic | Interpreted or JIT       | 1990 | Y | N | N
Java        | Static  | On a virtual machine     | 1990 | Y | N | Y
JavaScript  | Dynamic | Interpreted or JIT       | 1990 | Y | N | Y
Ruby        | Dynamic | Interpreted or JIT       | 1990 | Y | N | N
PHP         | Dynamic | Interpreted or JIT       | 1990 | Y | N | N
C#          | Static  | On a virtual machine     | 2000 | Y | N | Y
Scala       | Static  | On a virtual machine     | 2000 | Y | N | N
Clojure     | Dynamic | On a virtual machine     | 2000 | Y | Y | N
Go          | Static  | Compiled to machine code | 2000 | Y | N | Y
Rust        | Static  | Compiled to machine code | 2010 | N | N | Y
Kotlin      | Static  | On a virtual machine     | 2010 | Y | N | Y
TypeScript  | Static  | -                        | 2010 | Y | N | Y
Elixir      | Dynamic | On a virtual machine     | 2010 | Y | Y | Y
Julia       | Dynamic | Interpreted or JIT       | 2010 | Y | N | N
Swift       | Static  | Compiled to machine code | 2010 | Y | N | Y
Zig         | Static  | Compiled to machine code | 2010 | N | N | N
"""
