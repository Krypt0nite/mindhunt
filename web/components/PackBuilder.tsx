"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { ApiError, checkPack, rejectedIssues, savePack } from "@/lib/api";
import { useApp } from "@/lib/app";
import {
  EXAMPLES,
  MAX_ATTRS,
  MAX_ITEMS,
  addAttribute,
  emptyDraft,
  fieldId,
  forSaving,
  isBlank,
  loadLocalDraft,
  normalise,
  removeAttribute,
  rowsFromPaste,
  saveLocalDraft,
  setAttribute,
  setCell,
} from "@/lib/packDraft";
import type { AttrKind, DraftAttr, DraftCheck, DraftIssue, PackDraft } from "@/lib/protocol";

const KINDS: { id: AttrKind; label: string; hint: string }[] = [
  { id: "yesno", label: "Yes / No", hint: "Wears glasses, Has a pet" },
  { id: "choice", label: "Choice", hint: "Team: Sales, Tech, Design" },
  { id: "number", label: "Number", hint: "Years here, Age" },
];

function defaultQuestion(a: DraftAttr): string {
  const l = a.label.trim().toLowerCase() || "this";
  if (a.kind === "yesno") return `${a.label.trim() || "This"}?`;
  if (a.kind === "number") return `Is their ${l} more than 5? (and other comparisons)`;
  return `Is their ${l} ${a.choices?.[0] ?? "…"}?`;
}

/**
 * The pack builder: basics, attributes (the questions players can ask) and a
 * table of items. The server checks the draft as you type and has the final
 * word when you save.
 */
export function PackBuilder({ initial, packId }: { initial?: PackDraft; packId?: string }) {
  const { guest } = useApp();
  const router = useRouter();
  const [draft, setDraft] = useState<PackDraft>(() => normalise(initial ?? emptyDraft()));
  const [check, setCheck] = useState<DraftCheck | null>(null);
  const [checking, setChecking] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);
  const [pasteOpen, setPasteOpen] = useState(false);
  const restored = useRef(false);

  // A new pack resumes from this browser's last unsaved draft.
  useEffect(() => {
    if (packId || initial || restored.current) return;
    restored.current = true;
    const local = loadLocalDraft();
    if (local && !isBlank(local)) setDraft(local);
  }, [packId, initial]);

  useEffect(() => {
    if (!packId && touched) saveLocalDraft(draft);
  }, [draft, packId, touched]);

  // Live check, debounced.
  useEffect(() => {
    if (!guest || !touched) return;
    const t = setTimeout(async () => {
      setChecking(true);
      try {
        setCheck(await checkPack(guest, forSaving(draft)));
      } catch {
        /* keep the last result; saving reports problems anyway */
      } finally {
        setChecking(false);
      }
    }, 700);
    return () => clearTimeout(t);
  }, [draft, guest, touched]);

  const update = (d: PackDraft) => {
    setDraft(d);
    setTouched(true);
  };

  const issuesAt = useMemo(() => {
    const m = new Map<string, DraftIssue[]>();
    for (const i of check?.issues ?? []) m.set(i.path, [...(m.get(i.path) ?? []), i]);
    return m;
  }, [check]);
  const errAt = (path: string) => issuesAt.get(path)?.some((i) => i.severity === "error");
  const errors = (check?.issues ?? []).filter((i) => i.severity === "error");
  const warnings = (check?.issues ?? []).filter((i) => i.severity === "warning");
  const noun = draft.noun.trim() || "item";
  const filled = draft.items.filter((it) => it.name.trim()).length;

  async function save() {
    if (!guest) return;
    setSaving(true);
    setSaveError(null);
    try {
      const saved = await savePack(guest, forSaving(draft), packId);
      if (!packId) saveLocalDraft(null);
      router.push(`/pack/?id=${saved.id}&saved=1`);
    } catch (e) {
      const rej = rejectedIssues(e);
      if (rej) {
        setCheck({ ok: false, issues: rej.issues, stats: rej.stats ?? check?.stats ?? { items: 0, attributes: 0, unique: 0, avg_moves_est: 0 } });
        setSaveError("Fix the problems listed, then save again.");
      } else {
        setSaveError(e instanceof ApiError ? e.message : "Couldn't save the pack.");
      }
      setSaving(false);
    }
  }

  function focusIssue(path: string) {
    let el: HTMLElement | null = null;
    for (let p = path; p && !el; p = p.split(".").slice(0, -1).join(".")) el = document.getElementById(fieldId(p));
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
    el?.focus({ preventScroll: true });
  }

  return (
    <div className="grid gap-8 pb-24 lg:grid-cols-[minmax(0,1fr)_20rem] lg:gap-10">
      <div className="min-w-0">
        <h1 className="text-4xl font-extrabold sm:text-5xl">{packId ? "Edit your pack" : "Make a pack"}</h1>
        <p className="mt-2 max-w-[60ch] text-ink-soft">
          List the {noun}s to hunt and a few facts about each. Players ask about the facts; the game answers from your
          table. Anyone with the link can play it.
        </p>

        {!packId && isBlank(draft) && (
          <div className="mt-5 flex flex-wrap items-center gap-2">
            <span className="text-sm text-ink-soft">Start from an example:</span>
            {EXAMPLES.map((ex) => (
              <button key={ex.id} type="button" className="chip" onClick={() => update(normalise(structuredClone(ex.draft)))}>
                {ex.label}
              </button>
            ))}
          </div>
        )}

        {/* Basics */}
        <section className="panel mt-6 p-5 sm:p-6" aria-labelledby="basics-h">
          <h2 id="basics-h" className="text-xl font-bold">
            1. The pack
          </h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-[2fr_1fr]">
            <Field label="Title" id={fieldId("title")} error={errAt("title")}>
              <input
                id={fieldId("title")}
                className="field"
                maxLength={60}
                placeholder="Our office"
                value={draft.title}
                onChange={(e) => update({ ...draft, title: e.target.value })}
              />
            </Field>
            <Field label="One of them is a…" id={fieldId("noun")} error={errAt("noun")} hint="Singular: colleague, fruit">
              <input
                id={fieldId("noun")}
                className="field"
                maxLength={30}
                placeholder="colleague"
                value={draft.noun}
                onChange={(e) => update({ ...draft, noun: e.target.value })}
              />
            </Field>
          </div>
          <Field label="Description (optional)" id={fieldId("description")} error={errAt("description")} className="mt-4">
            <input
              id={fieldId("description")}
              className="field"
              maxLength={200}
              placeholder="Guess who from the third floor."
              value={draft.description}
              onChange={(e) => update({ ...draft, description: e.target.value })}
            />
          </Field>
        </section>

        {/* Attributes */}
        <section className="panel mt-6 p-5 sm:p-6" aria-labelledby="attrs-h">
          <h2 id="attrs-h" className="text-xl font-bold">
            2. What players can ask
          </h2>
          <p className="mt-1 text-sm text-ink-soft">
            Each fact becomes a question. Players can always ask about the first letters of the name too.
          </p>
          <ol className="mt-4 flex flex-col gap-4" id={fieldId("attributes")} tabIndex={-1}>
            {draft.attributes.map((a, i) => (
              <AttrEditor
                key={i}
                index={i}
                attr={a}
                errAt={errAt}
                canRemove={draft.attributes.length > 1}
                onChange={(patch) => update(setAttribute(draft, i, patch))}
                onRemove={() => update(removeAttribute(draft, i))}
              />
            ))}
          </ol>
          {draft.attributes.length < MAX_ATTRS && (
            <button type="button" className="btn mt-4" onClick={() => update(addAttribute(draft))}>
              Add a fact
            </button>
          )}
        </section>

        {/* Items */}
        <section className="panel mt-6 p-5 sm:p-6" aria-labelledby="items-h">
          <div className="flex flex-wrap items-baseline justify-between gap-3">
            <h2 id="items-h" className="text-xl font-bold">
              3. The {noun}s <span className="text-base font-normal text-ink-soft tabular">({filled})</span>
            </h2>
            <button type="button" className="btn btn-quiet" onClick={() => setPasteOpen((v) => !v)} aria-expanded={pasteOpen}>
              Paste from a spreadsheet
            </button>
          </div>
          <p className="mt-1 text-sm text-ink-soft">
            At least 8, at most {MAX_ITEMS}. Leave a cell blank if you&apos;re not sure; the game answers &ldquo;Not sure&rdquo;.
          </p>
          {pasteOpen && <PasteBox draft={draft} onApply={(d) => { update(d); setPasteOpen(false); }} />}
          <ItemsTable draft={draft} errAt={errAt} onChange={update} />
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              className="btn"
              disabled={draft.items.length >= MAX_ITEMS}
              onClick={() =>
                update({ ...draft, items: [...draft.items, { name: "", values: draft.attributes.map(() => "") }] })
              }
            >
              Add a row
            </button>
          </div>
        </section>

        <p className="mt-6 text-sm text-ink-soft">
          Keep it kind. Don&apos;t include private details (addresses, phone numbers, health) or anything that would hurt
          someone. Packs that players report are hidden until reviewed.
        </p>
      </div>

      {/* Check and save */}
      <aside className="lg:sticky lg:top-4 lg:self-start" aria-label="Check and save">
        <div className="panel p-5">
          <h2 className="text-lg font-bold">Check</h2>
          {!touched && !check ? (
            <p className="mt-2 text-sm text-ink-soft">Start typing and the pack is checked as you go.</p>
          ) : (
            <>
              <p className="mt-2 text-sm" aria-live="polite">
                {checking
                  ? "Checking…"
                  : errors.length
                    ? `${errors.length} ${errors.length === 1 ? "problem" : "problems"} to fix`
                    : check
                      ? "Ready to save"
                      : ""}
              </p>
              {check && check.stats.items > 0 && (
                <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
                  <dt className="text-ink-soft">{noun.charAt(0).toUpperCase() + noun.slice(1)}s</dt>
                  <dd className="tabular">{check.stats.items}</dd>
                  <dt className="text-ink-soft">Told apart by facts</dt>
                  <dd className="tabular">
                    {check.stats.unique} of {check.stats.items}
                  </dd>
                  <dt className="text-ink-soft">Moves to solve</dt>
                  <dd className="tabular">about {Math.max(2, Math.round(check.stats.avg_moves_est))}</dd>
                </dl>
              )}
              {[...errors, ...warnings].length > 0 && (
                <ul className="mt-3 flex max-h-72 flex-col gap-1.5 overflow-y-auto text-sm">
                  {[...errors, ...warnings].map((i, n) => (
                    <li key={n}>
                      <button
                        type="button"
                        onClick={() => focusIssue(i.path)}
                        className={`text-left underline-offset-2 hover:underline ${i.severity === "error" ? "text-no" : "text-ink-soft"}`}
                      >
                        {i.severity === "warning" && "Tip: "}
                        {describePath(i.path, draft)}
                        {i.message}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
          {!guest ? (
            <p className="mt-4 text-sm text-ink-soft">
              <Link href="/" className="underline">
                Choose a name
              </Link>{" "}
              to save packs.
            </p>
          ) : (
            <button
              type="button"
              className="btn btn-ink mt-4 h-12 w-full"
              disabled={saving || (check !== null && !check.ok) || filled === 0}
              onClick={save}
            >
              {saving ? "Saving…" : packId ? "Save changes" : "Save and get the link"}
            </button>
          )}
          {saveError && (
            <p role="alert" className="mt-2 text-sm text-no">
              {saveError}
            </p>
          )}
          {packId && (
            <p className="mt-3 text-xs text-ink-soft">Saving makes a new version. Rounds already running keep the old one.</p>
          )}
        </div>
      </aside>
    </div>
  );
}

function describePath(path: string, d: PackDraft): string {
  const p = path.split(".");
  if (p[0] === "items" && p.length >= 2) {
    const row = Number(p[1]);
    const name = d.items[row]?.name.trim();
    const col = p[2] === "values" && p[3] !== undefined ? d.attributes[Number(p[3])]?.label.trim() : "";
    return `Row ${row + 1}${name ? ` (${name})` : ""}${col ? `, ${col}` : ""}: `;
  }
  if (p[0] === "attributes" && p.length >= 2) {
    const a = d.attributes[Number(p[1])];
    return `Fact ${Number(p[1]) + 1}${a?.label.trim() ? ` (${a.label.trim()})` : ""}: `;
  }
  return "";
}

function Field({
  label,
  id,
  error,
  hint,
  className = "",
  children,
}: {
  label: string;
  id: string;
  error?: boolean;
  hint?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={`${className} ${error ? "[&_.field]:border-no" : ""}`}>
      <label htmlFor={id} className="mb-1 block text-sm font-semibold">
        {label}
        {hint && <span className="ml-1.5 font-normal text-ink-soft">{hint}</span>}
      </label>
      {children}
    </div>
  );
}

function AttrEditor({
  index,
  attr,
  errAt,
  canRemove,
  onChange,
  onRemove,
}: {
  index: number;
  attr: DraftAttr;
  errAt: (p: string) => boolean | undefined;
  canRemove: boolean;
  onChange: (p: Partial<DraftAttr>) => void;
  onRemove: () => void;
}) {
  const base = `attributes.${index}`;
  const [choicesText, setChoicesText] = useState((attr.choices ?? []).join(", "));
  useEffect(() => {
    setChoicesText((cur) => {
      const parsed = cur.split(",").map((s) => s.trim()).filter(Boolean);
      return parsed.join("|") === (attr.choices ?? []).join("|") ? cur : (attr.choices ?? []).join(", ");
    });
  }, [attr.choices]);

  return (
    <li className="rounded-[10px] border border-rule-soft p-4" id={fieldId(base)} tabIndex={-1}>
      <div className="flex items-start justify-between gap-3">
        <Field label={`Fact ${index + 1}`} id={fieldId(base + ".label")} error={errAt(base + ".label")} className="grow">
          <input
            id={fieldId(base + ".label")}
            className="field"
            maxLength={40}
            placeholder={KINDS.find((k) => k.id === attr.kind)?.hint.split(",")[0]}
            value={attr.label}
            onChange={(e) => onChange({ label: e.target.value })}
          />
        </Field>
        {canRemove && (
          <button type="button" className="btn btn-quiet mt-6 shrink-0" onClick={onRemove} aria-label={`Remove fact ${index + 1}`}>
            Remove
          </button>
        )}
      </div>
      <div className="mt-3 flex flex-wrap gap-2" role="group" aria-label="Kind of answer" id={fieldId(base + ".kind")}>
        {KINDS.map((k) => (
          <button key={k.id} type="button" className="chip" aria-pressed={attr.kind === k.id} onClick={() => onChange({ kind: k.id })}>
            {k.label}
          </button>
        ))}
      </div>
      {attr.kind === "choice" && (
        <Field
          label="Choices"
          hint="separated by commas"
          id={fieldId(base + ".choices")}
          error={errAt(base + ".choices")}
          className="mt-3"
        >
          <input
            id={fieldId(base + ".choices")}
            className="field"
            placeholder="Sales, Tech, Design"
            value={choicesText}
            onChange={(e) => {
              setChoicesText(e.target.value);
              onChange({ choices: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) });
            }}
          />
        </Field>
      )}
      <Field
        label="How it's asked"
        hint={attr.kind === "yesno" ? "optional" : "optional; put {value} where the answer goes"}
        id={fieldId(base + ".question")}
        error={errAt(base + ".question")}
        className="mt-3"
      >
        <input
          id={fieldId(base + ".question")}
          className="field"
          maxLength={120}
          placeholder={
            attr.kind === "yesno" ? "Do they wear glasses?" : attr.kind === "choice" ? "Do they work in {value}?" : "Have they been here {value} years?"
          }
          value={attr.question ?? ""}
          onChange={(e) => onChange({ question: e.target.value })}
        />
      </Field>
      {!attr.question?.trim() && attr.label.trim() && (
        <p className="mt-1 text-xs text-ink-soft">Without one, players see: {defaultQuestion(attr)}</p>
      )}
    </li>
  );
}

function ItemsTable({
  draft,
  errAt,
  onChange,
}: {
  draft: PackDraft;
  errAt: (p: string) => boolean | undefined;
  onChange: (d: PackDraft) => void;
}) {
  const bad = "border-no bg-no/5";
  return (
    <div className="mt-4 overflow-x-auto rounded-[10px] border border-rule-soft" id={fieldId("items")} tabIndex={-1}>
      <table className="w-full min-w-max border-collapse text-sm">
        <thead>
          <tr className="bg-rule-soft/60 text-left">
            <th className="w-10 px-2 py-2 font-semibold text-ink-soft">#</th>
            <th className="min-w-40 px-2 py-2 font-semibold">Name</th>
            {draft.attributes.map((a, i) => (
              <th key={i} className={`px-2 py-2 font-semibold ${a.kind === "number" ? "min-w-20" : "min-w-28"}`}>
                {a.label.trim() || `Fact ${i + 1}`}
              </th>
            ))}
            <th className="w-10" aria-label="Remove" />
          </tr>
        </thead>
        <tbody>
          {draft.items.map((it, r) => (
            <tr key={r} className="border-t border-rule-soft">
              <td className="px-2 text-ink-soft tabular">{r + 1}</td>
              <td className="px-1 py-1">
                <input
                  id={fieldId(`items.${r}.name`)}
                  aria-label={`Row ${r + 1} name`}
                  className={`field min-h-9 px-2 ${errAt(`items.${r}.name`) ? bad : ""}`}
                  maxLength={60}
                  value={it.name}
                  onChange={(e) => onChange(setCell(draft, r, "name", e.target.value))}
                />
              </td>
              {draft.attributes.map((a, c) => {
                const path = `items.${r}.values.${c}`;
                const cls = `field min-h-9 px-2 ${errAt(path) ? bad : ""}`;
                const label = `Row ${r + 1} ${a.label || `fact ${c + 1}`}`;
                const v = it.values[c] ?? "";
                return (
                  <td key={c} className="px-1 py-1">
                    {a.kind === "number" ? (
                      <input
                        id={fieldId(path)}
                        aria-label={label}
                        className={`${cls} tabular`}
                        inputMode="numeric"
                        value={v}
                        onChange={(e) => onChange(setCell(draft, r, c, e.target.value))}
                      />
                    ) : (
                      <select
                        id={fieldId(path)}
                        aria-label={label}
                        className={cls}
                        value={a.kind === "yesno" ? yesNoValue(v) : v}
                        onChange={(e) => onChange(setCell(draft, r, c, e.target.value))}
                      >
                        <option value="">Not sure</option>
                        {(a.kind === "yesno" ? ["yes", "no"] : (a.choices ?? [])).map((o) => (
                          <option key={o} value={o}>
                            {a.kind === "yesno" ? (o === "yes" ? "Yes" : "No") : o}
                          </option>
                        ))}
                        {a.kind === "choice" && v && !(a.choices ?? []).some((o) => o.toLowerCase() === v.toLowerCase()) && (
                          <option value={v}>{v} (not a choice)</option>
                        )}
                      </select>
                    )}
                  </td>
                );
              })}
              <td className="px-1">
                <button
                  type="button"
                  className="grid h-9 w-9 place-items-center rounded-full text-ink-soft hover:bg-rule-soft"
                  aria-label={`Remove row ${r + 1}`}
                  onClick={() => onChange({ ...draft, items: draft.items.filter((_, i) => i !== r) })}
                >
                  ×
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function yesNoValue(v: string): string {
  const s = v.trim().toLowerCase();
  if (["yes", "y", "true", "1"].includes(s)) return "yes";
  if (["no", "n", "false", "0"].includes(s)) return "no";
  return s ? v : "";
}

function PasteBox({ draft, onApply }: { draft: PackDraft; onApply: (d: PackDraft) => void }) {
  const [text, setText] = useState("");
  const parsed = useMemo(() => (text.trim() ? rowsFromPaste(draft, text) : null), [draft, text]);
  const cols = ["Name", ...draft.attributes.map((a, i) => a.label.trim() || `Fact ${i + 1}`)].join(", ");
  const keep = draft.items.filter((it) => it.name.trim() || it.values.some((v) => v.trim()));

  return (
    <div className="mt-4 rounded-[10px] border border-rule-soft p-4">
      <label htmlFor="paste" className="text-sm font-semibold">
        Copy rows from Excel or Google Sheets and paste them here
      </label>
      <p className="text-xs text-ink-soft">Columns in this order: {cols}. A header row is skipped.</p>
      <textarea
        id="paste"
        className="field mt-2 min-h-32 py-2 font-mono text-xs"
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder={"Asha\tSales\tyes\nBilal\tTech\tno"}
      />
      {parsed && (
        <p className="mt-1 text-xs text-ink-soft">
          {parsed.rows.length} rows found{parsed.skippedHeader ? ", header skipped" : ""}.
          {parsed.extraColumns && " Some rows have more columns than facts; the extras are ignored."}
        </p>
      )}
      <div className="mt-2 flex flex-wrap gap-2">
        <button
          type="button"
          className="btn"
          disabled={!parsed?.rows.length}
          onClick={() => parsed && onApply({ ...draft, items: [...keep, ...parsed.rows].slice(0, MAX_ITEMS) })}
        >
          Add these rows
        </button>
        <button
          type="button"
          className="btn btn-quiet"
          disabled={!parsed?.rows.length}
          onClick={() => parsed && onApply({ ...draft, items: parsed.rows.slice(0, MAX_ITEMS) })}
        >
          Replace all rows
        </button>
      </div>
    </div>
  );
}
