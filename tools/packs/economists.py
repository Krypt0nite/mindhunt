SPEC = {
    "id": "economists",
    "version": 2,  # v2: checked 2026-10-18 (tools/evidence/economists.py)
    "draft": False,
    "as_of": "2026-10-18",
    "category": "Economics",
    "title": "Economists",
    "noun": "economist",
    "description": "From Adam Smith to Amartya Sen: the thinkers behind how we understand markets.",
    "letters": "Does their name start with “{value}”?",
    "attributes": [
        {"key": "born_in", "label": "Country of birth", "type": "enum",
         "values": ["United Kingdom", "Germany", "Austria", "France", "United States", "Canada", "India"],
         "help": "Using today's borders.",
         "templates": {"eq": "Were they born in what is now {value}?"},
         "aliases": {"United Kingdom": ["british", "english", "scottish", "uk", "britain"], "Germany": ["german"],
                     "Austria": ["austrian"], "France": ["french"], "United States": ["american", "usa", "us"],
                     "Canada": ["canadian"], "India": ["indian"]}},
        {"key": "century", "label": "Born in the", "type": "int", "min": 1700, "max": 1900, "step": 100,
         "templates": {"eq": "Were they born in the {value}s?", "gt": "Were they born after the {value}s?",
                       "lt": "Were they born before the {value}s?"}},
        {"key": "school", "label": "School of thought", "type": "enum",
         "values": ["Classical", "Marxian", "Neoclassical", "Keynesian", "Austrian", "Monetarist", "Development", "Behavioural", "Institutional"],
         "help": "The school they are usually grouped with. Where sources disagree (Galbraith) or none fits (Piketty, Rajan), the answer is Unknown.",
         "templates": {"eq": "Are they usually grouped with the {value} school?"},
         "aliases": {"Keynesian": ["keynes"], "Marxian": ["marxist", "communist"], "Behavioural": ["behavioral"],
                     "Development": ["development economics", "poverty"]}},
        {"key": "nobel", "label": "Nobel Prize in Economics", "type": "bool",
         "help": "Awarded since 1969.",
         "templates": {"is": "Did they win the Nobel Prize in Economics?", "is_false": "Did they never win the Nobel Prize in Economics?"}},
        {"key": "woman", "label": "Woman", "type": "bool",
         "templates": {"is": "Is this economist a woman?", "is_false": "Is this economist a man?"}},
    ],
    "columns": ["name", "born_in", "century", "school", "nobel", "woman"],
}

ROWS = """
Adam Smith              | United Kingdom | 1700 | Classical     | N | N
Thomas Malthus          | United Kingdom | 1700 | Classical     | N | N
David Ricardo           | United Kingdom | 1700 | Classical     | N | N
John Stuart Mill        | United Kingdom | 1800 | Classical     | N | N
Karl Marx               | Germany        | 1800 | Marxian       | N | N
Alfred Marshall         | United Kingdom | 1800 | Neoclassical  | N | N
John Maynard Keynes     | United Kingdom | 1800 | Keynesian     | N | N
Friedrich Hayek         | Austria        | 1800 | Austrian      | Y | N
John Kenneth Galbraith  | Canada         | 1900 | -             | N | N
Milton Friedman         | United States  | 1900 | Monetarist    | Y | N
Paul Samuelson          | United States  | 1900 | Keynesian     | Y | N
Amartya Sen             | India          | 1900 | Development   | Y | N
Elinor Ostrom           | United States  | 1900 | Institutional | Y | Y
Joseph Stiglitz         | United States  | 1900 | Keynesian     | Y | N
Richard Thaler          | United States  | 1900 | Behavioural   | Y | N
Paul Krugman            | United States  | 1900 | Keynesian     | Y | N
Abhijit Banerjee        | India          | 1900 | Development   | Y | N
Thomas Piketty          | France         | 1900 | -             | N | N
Esther Duflo            | France         | 1900 | Development   | Y | Y
Raghuram Rajan          | India          | 1900 | -             | N | N
"""
