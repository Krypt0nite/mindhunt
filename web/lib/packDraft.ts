import type { AttrKind, DraftAttr, PackDraft } from "./protocol";

export const MAX_ATTRS = 10;
export const MIN_ITEMS = 8;
export const MAX_ITEMS = 150;

export function emptyDraft(): PackDraft {
  return {
    title: "",
    noun: "",
    description: "",
    attributes: [{ label: "", kind: "yesno" }],
    items: Array.from({ length: MIN_ITEMS }, () => ({ name: "", values: [""] })),
  };
}

/** DOM id for a field addressed by a server issue path ("items.3.values.1"). */
export function fieldId(path: string): string {
  return "f-" + path.replace(/\./g, "-");
}

/** Keeps every row's values aligned with the attribute list. */
export function normalise(d: PackDraft): PackDraft {
  const n = d.attributes.length;
  return {
    ...d,
    items: d.items.map((it) => ({
      ...it,
      values: Array.from({ length: n }, (_, i) => it.values[i] ?? ""),
    })),
  };
}

export function addAttribute(d: PackDraft, kind: AttrKind = "yesno"): PackDraft {
  if (d.attributes.length >= MAX_ATTRS) return d;
  return {
    ...d,
    attributes: [...d.attributes, { label: "", kind, ...(kind === "choice" ? { choices: [] } : {}) }],
    items: d.items.map((it) => ({ ...it, values: [...it.values, ""] })),
  };
}

export function removeAttribute(d: PackDraft, index: number): PackDraft {
  return {
    ...d,
    attributes: d.attributes.filter((_, i) => i !== index),
    items: d.items.map((it) => ({ ...it, values: it.values.filter((_, i) => i !== index) })),
  };
}

export function setAttribute(d: PackDraft, index: number, patch: Partial<DraftAttr>): PackDraft {
  const attributes = d.attributes.map((a, i) => {
    if (i !== index) return a;
    const next = { ...a, ...patch };
    return next.kind === "choice" && !next.choices ? { ...next, choices: [] } : next;
  });
  let items = d.items;
  // Changing the kind clears that column: yes/no answers don't fit a choice.
  if (patch.kind && patch.kind !== d.attributes[index]?.kind) {
    items = d.items.map((it) => ({ ...it, values: it.values.map((v, i) => (i === index ? "" : v)) }));
  }
  return { ...d, attributes, items };
}

export function setCell(d: PackDraft, row: number, col: number | "name", value: string): PackDraft {
  return {
    ...d,
    items: d.items.map((it, i) => {
      if (i !== row) return it;
      if (col === "name") return { ...it, name: value };
      const values = [...it.values];
      values[col] = value;
      return { ...it, values };
    }),
  };
}

/** Rows ready to save: blank rows at the end are dropped. */
export function forSaving(d: PackDraft): PackDraft {
  const items = d.items.filter((it) => it.name.trim() !== "" || it.values.some((v) => v.trim() !== ""));
  return {
    ...d,
    attributes: d.attributes.map((a) => (a.kind === "choice" ? a : { ...a, choices: undefined })),
    items,
  };
}

/** Splits pasted spreadsheet text: tabs if present, otherwise commas (with "quoted, values"). */
export function parseTable(text: string): string[][] {
  const lines = text.replace(/\r\n?/g, "\n").split("\n").filter((l) => l.trim() !== "");
  const tabs = lines.some((l) => l.includes("\t"));
  return lines.map((l) => (tabs ? l.split("\t") : splitCSV(l)).map((c) => c.trim()));
}

function splitCSV(line: string): string[] {
  const out: string[] = [];
  let cur = "";
  let quoted = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (quoted) {
      if (ch === '"' && line[i + 1] === '"') {
        cur += '"';
        i++;
      } else if (ch === '"') quoted = false;
      else cur += ch;
    } else if (ch === '"') quoted = true;
    else if (ch === ",") {
      out.push(cur);
      cur = "";
    } else cur += ch;
  }
  out.push(cur);
  return out;
}

/**
 * Turns pasted rows into items. The first column is the name, the rest follow
 * the attribute order. A first row that repeats the attribute names is a
 * header and is skipped.
 */
export function rowsFromPaste(d: PackDraft, text: string): { rows: PackDraft["items"]; skippedHeader: boolean; extraColumns: boolean } {
  let table = parseTable(text);
  let skippedHeader = false;
  const first = table[0];
  if (first) {
    const head = first.slice(1).map((c) => c.toLowerCase());
    const labels = d.attributes.map((a) => a.label.trim().toLowerCase()).filter(Boolean);
    if (/^name$/i.test(first[0] ?? "") || (labels.length > 0 && labels.every((l, i) => head[i] === l))) {
      table = table.slice(1);
      skippedHeader = true;
    }
  }
  const n = d.attributes.length;
  const extraColumns = table.some((r) => r.length - 1 > n);
  const rows = table.map((r) => ({ name: r[0] ?? "", values: Array.from({ length: n }, (_, i) => r[i + 1] ?? "") }));
  return { rows, skippedHeader, extraColumns };
}

export const EXAMPLES: { id: string; label: string; draft: PackDraft }[] = [
  {
    id: "office",
    label: "Office colleagues",
    draft: {
      title: "Our office",
      noun: "colleague",
      description: "Guess the colleague. Keep it friendly: nothing private or hurtful.",
      attributes: [
        { label: "Team", kind: "choice", choices: ["Sales", "Tech", "Design", "Ops"], question: "Do they work in {value}?" },
        { label: "Wears glasses", kind: "yesno", question: "Do they wear glasses?" },
        { label: "Drinks coffee", kind: "yesno", question: "Do they drink coffee?" },
        { label: "Years here", kind: "number", question: "Have they been here {value} years?" },
      ],
      items: [
        ["Asha", "Sales", "yes", "yes", "3"],
        ["Bilal", "Tech", "no", "yes", "5"],
        ["Chen", "Design", "yes", "no", "1"],
        ["Divya", "Tech", "no", "no", "8"],
        ["Esha", "Ops", "no", "yes", "2"],
        ["Farhan", "Design", "yes", "yes", "6"],
        ["Gita", "Tech", "yes", "no", "4"],
        ["Hari", "Sales", "no", "no", "10"],
        ["Ira", "Ops", "yes", "yes", "7"],
        ["Jai", "Sales", "yes", "no", "1"],
      ].map(([name, ...values]) => ({ name: name ?? "", values })),
    },
  },
  {
    id: "fruit",
    label: "Fruit",
    draft: {
      title: "Fruit bowl",
      noun: "fruit",
      description: "Sixteen fruits, from apple to watermelon.",
      attributes: [
        { label: "Colour", kind: "choice", choices: ["Red", "Yellow", "Green", "Orange", "Purple"], question: "Is it {value} when ripe?" },
        { label: "Grows on trees", kind: "yesno", question: "Does it grow on a tree?" },
        { label: "Has a stone", kind: "yesno", question: "Does it have a single stone?" },
        { label: "Size", kind: "choice", choices: ["Small", "Medium", "Large"], question: "Is it {value}?" },
      ],
      items: [
        "Apple|Red|yes|no|Medium", "Banana|Yellow|no|no|Medium", "Cherry|Red|yes|yes|Small", "Grape|Purple|no|no|Small",
        "Lemon|Yellow|yes|no|Medium", "Lime|Green|yes|no|Small", "Mango|Orange|yes|yes|Large", "Orange|Orange|yes|no|Medium",
        "Peach|Orange|yes|yes|Medium", "Pear|Green|yes|no|Medium", "Plum|Purple|yes|yes|Small", "Strawberry|Red|no|no|Small",
        "Watermelon|Green|no|no|Large", "Kiwi|Green|no|no|Small", "Coconut|Green|yes|no|Large", "Papaya|Orange|yes|no|Large",
      ].map((r) => {
        const [name, ...values] = r.split("|");
        return { name: name ?? "", values };
      }),
    },
  },
];

const LOCAL_KEY = "mindhunt.packDraft";

export function loadLocalDraft(): PackDraft | null {
  try {
    const raw = localStorage.getItem(LOCAL_KEY);
    return raw ? normalise(JSON.parse(raw) as PackDraft) : null;
  } catch {
    return null;
  }
}

export function saveLocalDraft(d: PackDraft | null) {
  try {
    if (d) localStorage.setItem(LOCAL_KEY, JSON.stringify(d));
    else localStorage.removeItem(LOCAL_KEY);
  } catch {
    /* storage unavailable: the draft lives in memory only */
  }
}

export function isBlank(d: PackDraft): boolean {
  return !d.title.trim() && d.items.every((it) => !it.name.trim()) && d.attributes.every((a) => !a.label.trim());
}
