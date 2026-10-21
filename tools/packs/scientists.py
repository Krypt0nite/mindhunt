SPEC = {
    "id": "famous-scientists",
    "version": 2,  # v2: checked 2026-10-18 (tools/evidence/scientists.py)
    "draft": False,
    "as_of": "2026-10-18",
    "category": "Science",
    "title": "Scientists",
    "noun": "scientist",
    "description": "Physicists, chemists, biologists and mathematicians who changed how we see the world.",
    "letters": "Does their name start with “{value}”?",
    "attributes": [
        {"key": "field", "label": "Field", "type": "set",
         "values": ["Physics", "Chemistry", "Biology", "Mathematics", "Astronomy", "Computer science", "Medicine", "Engineering"],
         "help": "The fields they are best known for; some have more than one.",
         "templates": {"contains": "Are they best known for {value_lc}?"},
         "aliases": {"Physics": ["physicist"], "Chemistry": ["chemist"], "Biology": ["biologist"],
                     "Mathematics": ["mathematician", "maths", "math"], "Astronomy": ["astronomer", "astrophysicist"],
                     "Computer science": ["computer scientist", "computing", "computers"], "Medicine": ["doctor", "medical"],
                     "Engineering": ["engineer", "inventor"]}},
        {"key": "born_in", "label": "Country of birth", "type": "enum",
         "values": ["United Kingdom", "Germany", "France", "Italy", "Poland", "Austria", "Switzerland", "Denmark",
                    "Czech Republic", "Croatia", "Russia", "India", "China", "United States", "New Zealand"],
         "help": "Where they were born, using today's borders.",
         "templates": {"eq": "Were they born in what is now {value}?"},
         "aliases": {"United Kingdom": ["british", "english", "scottish", "uk", "england", "britain"],
                     "Germany": ["german"], "France": ["french"], "Italy": ["italian"], "Poland": ["polish"],
                     "Austria": ["austrian"], "Russia": ["russian"], "India": ["indian"], "China": ["chinese"],
                     "United States": ["american", "usa", "us"], "Denmark": ["danish"], "Switzerland": ["swiss"]}},
        {"key": "century", "label": "Born in the", "type": "int", "min": 1400, "max": 1900, "step": 100,
         "help": "1800 means born between 1800 and 1899.",
         "templates": {"eq": "Were they born in the {value}s?", "gt": "Were they born after the {value}s?",
                       "lt": "Were they born before the {value}s?", "between": "Were they born between the {value}s and the {value2}s?"}},
        {"key": "nobel", "label": "Nobel Prize winner", "type": "bool",
         "templates": {"is": "Did they win a Nobel Prize?", "is_false": "Did they never win a Nobel Prize?"}},
        {"key": "woman", "label": "Woman", "type": "bool",
         "templates": {"is": "Is this scientist a woman?", "is_false": "Is this scientist a man?"}},
    ],
    "columns": ["name", "field", "born_in", "century", "nobel", "woman"],
}

ROWS = """
Nicolaus Copernicus     | Astronomy                     | Poland         | 1400 | N | N
Galileo Galilei         | Physics;Astronomy             | Italy          | 1500 | N | N
Johannes Kepler         | Astronomy;Mathematics         | Germany        | 1500 | N | N
Isaac Newton            | Physics;Mathematics;Astronomy | United Kingdom | 1600 | N | N
Leonhard Euler          | Mathematics                   | Switzerland    | 1700 | N | N
Antoine Lavoisier       | Chemistry                     | France         | 1700 | N | N
Carl Friedrich Gauss    | Mathematics                   | Germany        | 1700 | N | N
Michael Faraday         | Physics;Chemistry             | United Kingdom | 1700 | N | N
Ada Lovelace            | Mathematics;Computer science  | United Kingdom | 1800 | N | Y
Charles Darwin          | Biology                       | United Kingdom | 1800 | N | N
Gregor Mendel           | Biology                       | Czech Republic | 1800 | N | N
Louis Pasteur           | Chemistry;Biology             | France         | 1800 | N | N
James Clerk Maxwell     | Physics                       | United Kingdom | 1800 | N | N
Dmitri Mendeleev        | Chemistry                     | Russia         | 1800 | N | N
Nikola Tesla            | Engineering                   | Croatia        | 1800 | N | N
Max Planck              | Physics                       | Germany        | 1800 | Y | N
Marie Curie             | Physics;Chemistry             | Poland         | 1800 | Y | Y
Ernest Rutherford       | Physics;Chemistry             | New Zealand    | 1800 | Y | N
Albert Einstein         | Physics                       | Germany        | 1800 | Y | N
Alexander Fleming       | Biology;Medicine              | United Kingdom | 1800 | Y | N
Niels Bohr              | Physics                       | Denmark        | 1800 | Y | N
Erwin Schrödinger       | Physics                       | Austria        | 1800 | Y | N
Srinivasa Ramanujan     | Mathematics                   | India          | 1800 | N | N
C. V. Raman             | Physics                       | India          | 1800 | Y | N
Edwin Hubble            | Astronomy                     | United States  | 1800 | N | N
Satyendra Nath Bose     | Physics                       | India          | 1800 | N | N
Enrico Fermi            | Physics                       | Italy          | 1900 | Y | N
Werner Heisenberg       | Physics                       | Germany        | 1900 | Y | N
Linus Pauling           | Chemistry                     | United States  | 1900 | Y | N
Paul Dirac              | Physics                       | United Kingdom | 1900 | Y | N
Barbara McClintock      | Biology                       | United States  | 1900 | Y | Y
Robert Oppenheimer      | Physics                       | United States  | 1900 | N | N
Homi Bhabha             | Physics                       | India          | 1900 | N | N
Alan Turing             | Mathematics;Computer science  | United Kingdom | 1900 | N | N
Richard Feynman         | Physics                       | United States  | 1900 | Y | N
Katherine Johnson       | Mathematics                   | United States  | 1900 | N | Y
Vikram Sarabhai         | Physics;Astronomy             | India          | 1900 | N | N
Rosalind Franklin       | Chemistry                     | United Kingdom | 1900 | N | Y
Tu Youyou               | Medicine;Chemistry            | China          | 1900 | Y | Y
Jane Goodall            | Biology                       | United Kingdom | 1900 | N | Y
Carl Sagan              | Astronomy                     | United States  | 1900 | N | N
Stephen Hawking         | Physics                       | United Kingdom | 1900 | N | N
Venkatraman Ramakrishnan| Biology;Chemistry             | India          | 1900 | Y | N
"""
