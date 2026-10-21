SPEC = {
    "id": "historical-figures",
    "version": 2,  # v2: checked 2026-10-18 (tools/evidence/history.py)
    "draft": False,
    "as_of": "2026-10-18",
    "category": "History",
    "title": "Historical Figures",
    "noun": "historical figure",
    "description": "Rulers, reformers, explorers and freedom fighters from Ashoka to Mandela.",
    "letters": "Does their name start with “{value}”?",
    "attributes": [
        {"key": "region", "label": "Region", "type": "enum",
         "values": ["South Asia", "East Asia", "Middle East", "Europe", "Africa", "Americas"],
         "help": "Where they lived and did their main work (Mother Teresa counts as South Asia; explorers count where they sailed from).",
         "templates": {"eq": "Did they make their name in {value}?"},
         "aliases": {"South Asia": ["india", "indian", "south asian"], "East Asia": ["china", "chinese", "east asian"],
                     "Europe": ["european"], "Africa": ["african"], "Americas": ["america", "american"],
                     "Middle East": ["middle eastern"]}},
        {"key": "born", "label": "Born", "type": "enum",
         "values": ["Before 500 CE", "500 to 1499", "1500 to 1799", "1800s", "1900s"],
         "templates": {"eq:Before 500 CE": "Were they born before 500 CE?",
                       "eq:500 to 1499": "Were they born between 500 and 1499?",
                       "eq:1500 to 1799": "Were they born between 1500 and 1799?",
                       "eq:1800s": "Were they born in the 1800s?",
                       "eq:1900s": "Were they born in the 1900s?"},
         "aliases": {"Before 500 CE": ["ancient", "bce", "bc"], "500 to 1499": ["medieval", "middle ages"]}},
        {"key": "role", "label": "Best known as", "type": "set",
         "help": "What they are best known for, following Wikipedia's description of them. Someone can be best known for more than one thing.",
         "values": ["Ruler", "Military leader", "Political leader", "Revolutionary or freedom fighter",
                    "Religious figure", "Philosopher", "Reformer", "Explorer", "Writer"],
         "templates": {"contains:Ruler": "Are they best known as a ruler (king, queen or emperor)?",
                       "contains:Military leader": "Are they best known as a military leader?",
                       "contains:Political leader": "Are they best known as a political leader?",
                       "contains:Revolutionary or freedom fighter": "Are they best known as a revolutionary or freedom fighter?",
                       "contains:Religious figure": "Are they best known as a religious figure?",
                       "contains:Philosopher": "Are they best known as a philosopher?",
                       "contains:Reformer": "Are they best known as a social reformer?",
                       "contains:Explorer": "Are they best known as an explorer?",
                       "contains:Writer": "Are they best known as a writer?"},
         "aliases": {"Ruler": ["king", "queen", "emperor", "empress", "monarch", "sultan"],
                     "Military leader": ["general", "warrior", "conqueror"],
                     "Political leader": ["prime minister", "president", "politician"],
                     "Revolutionary or freedom fighter": ["freedom fighter", "revolutionary"],
                     "Religious figure": ["religious", "saint", "spiritual"], "Explorer": ["sailor", "navigator"],
                     "Writer": ["author", "poet"], "Reformer": ["reformist", "activist"]}},
        {"key": "woman", "label": "Woman", "type": "bool",
         "templates": {"is": "Is this person a woman?", "is_false": "Is this person a man?"}},
        {"key": "nobel_peace", "label": "Nobel Peace Prize", "type": "bool",
         "templates": {"is": "Did they win the Nobel Peace Prize?", "is_false": "Did they never win the Nobel Peace Prize?"}},
    ],
    "columns": ["name", "region", "born", "role", "woman", "nobel_peace"],
}

ROWS = """
Gautama Buddha        | South Asia  | Before 500 CE | Religious figure;Philosopher                                  | N | N
Chandragupta Maurya   | South Asia  | Before 500 CE | Ruler;Military leader                                         | N | N
Ashoka                | South Asia  | Before 500 CE | Ruler                                                         | N | N
Guru Nanak            | South Asia  | 500 to 1499   | Religious figure                                              | N | N
Akbar                 | South Asia  | 1500 to 1799  | Ruler                                                         | N | N
Shivaji               | South Asia  | 1500 to 1799  | Ruler;Military leader                                         | N | N
Tipu Sultan           | South Asia  | 1500 to 1799  | Ruler;Military leader                                         | N | N
Rani Lakshmibai       | South Asia  | 1800s         | Ruler;Military leader;Revolutionary or freedom fighter        | Y | N
Rabindranath Tagore   | South Asia  | 1800s         | Writer;Philosopher;Reformer                                   | N | N
Swami Vivekananda     | South Asia  | 1800s         | Religious figure;Philosopher;Reformer                         | N | N
Mahatma Gandhi        | South Asia  | 1800s         | Political leader;Revolutionary or freedom fighter;Reformer    | N | N
Jawaharlal Nehru      | South Asia  | 1800s         | Political leader;Revolutionary or freedom fighter             | N | N
B. R. Ambedkar        | South Asia  | 1800s         | Reformer;Political leader                                     | N | N
Subhas Chandra Bose   | South Asia  | 1800s         | Revolutionary or freedom fighter;Military leader;Political leader | N | N
Bhagat Singh          | South Asia  | 1900s         | Revolutionary or freedom fighter                              | N | N
Mother Teresa         | South Asia  | 1900s         | Religious figure                                              | Y | Y
Indira Gandhi         | South Asia  | 1900s         | Political leader                                              | Y | N
Confucius             | East Asia   | Before 500 CE | Philosopher                                                   | N | N
Genghis Khan          | East Asia   | 500 to 1499   | Ruler;Military leader                                         | N | N
Sun Yat-sen           | East Asia   | 1800s         | Political leader;Revolutionary or freedom fighter             | N | N
Mao Zedong            | East Asia   | 1800s         | Political leader;Military leader;Revolutionary or freedom fighter | N | N
Hammurabi             | Middle East | Before 500 CE | Ruler                                                         | N | N
Saladin               | Middle East | 500 to 1499   | Ruler;Military leader                                         | N | N
Socrates              | Europe      | Before 500 CE | Philosopher                                                   | N | N
Alexander the Great   | Europe      | Before 500 CE | Ruler;Military leader                                         | N | N
Julius Caesar         | Europe      | Before 500 CE | Ruler;Military leader                                         | N | N
Marco Polo            | Europe      | 500 to 1499   | Explorer                                                      | N | N
Joan of Arc           | Europe      | 500 to 1499   | Military leader;Religious figure                              | Y | N
Christopher Columbus  | Europe      | 500 to 1499   | Explorer                                                      | N | N
Vasco da Gama         | Europe      | 500 to 1499   | Explorer                                                      | N | N
Martin Luther         | Europe      | 500 to 1499   | Religious figure;Reformer                                     | N | N
Elizabeth I           | Europe      | 1500 to 1799  | Ruler                                                         | Y | N
Napoleon Bonaparte    | Europe      | 1500 to 1799  | Ruler;Military leader                                         | N | N
Queen Victoria        | Europe      | 1800s         | Ruler                                                         | Y | N
Karl Marx             | Europe      | 1800s         | Philosopher;Writer                                            | N | N
Winston Churchill     | Europe      | 1800s         | Political leader;Writer                                       | N | N
Cleopatra             | Africa      | Before 500 CE | Ruler                                                         | Y | N
Mansa Musa            | Africa      | 500 to 1499   | Ruler                                                         | N | N
Kwame Nkrumah         | Africa      | 1900s         | Political leader;Revolutionary or freedom fighter             | N | N
Nelson Mandela        | Africa      | 1900s         | Political leader;Revolutionary or freedom fighter             | N | Y
George Washington     | Americas    | 1500 to 1799  | Political leader;Military leader                              | N | N
Simón Bolívar         | Americas    | 1500 to 1799  | Military leader;Political leader;Revolutionary or freedom fighter | N | N
Abraham Lincoln       | Americas    | 1800s         | Political leader                                              | N | N
Martin Luther King Jr.| Americas    | 1900s         | Reformer;Religious figure                                     | N | Y
"""
