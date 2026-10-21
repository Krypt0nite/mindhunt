"use client";

import { useMemo, useState } from "react";
import type { AttrView, Move, Predicate } from "@/lib/protocol";

const OP_LABEL: Record<string, string> = {
  eq: "exactly",
  gt: "more than",
  lt: "less than",
  between: "between",
};

/** Same idea as the server's canonical form, for greying out repeats. */
export function predicateKey(p: Predicate): string {
  const v = typeof p.value === "string" ? p.value.toLowerCase() : JSON.stringify(p.value);
  return `${p.attr}|${p.op}|${v}`;
}

function intSteps(a: AttrView): number[] {
  const min = a.min ?? 0;
  const max = a.max ?? 0;
  const step = a.step && a.step > 0 ? a.step : 1;
  const out: number[] = [];
  for (let v = min; v <= max && out.length < 400; v += step) out.push(v);
  return out;
}

export function preview(a: AttrView, p: Predicate): string {
  switch (a.type) {
    case "bool":
      return `${a.label}?`;
    case "int":
      if (p.op === "between" && Array.isArray(p.value)) return `${a.label} between ${p.value[0]} and ${p.value[1]}?`;
      return `${a.label} ${OP_LABEL[p.op] ?? p.op} ${p.value}?`;
    case "string":
      return `${a.label} starts with “${String(p.value).toUpperCase()}”?`;
    case "set":
      return `${a.label} includes ${p.value}?`;
    default:
      return `${a.label}: ${p.value}?`;
  }
}

export function QuestionBuilder({
  attributes,
  log,
  busy,
  onAsk,
}: {
  attributes: AttrView[];
  log: Move[];
  busy: boolean;
  onAsk: (p: Predicate) => void;
}) {
  const [attrKey, setAttrKey] = useState<string | null>(null);
  const [op, setOp] = useState<string>("eq");
  const [value, setValue] = useState<string>("");
  const [value2, setValue2] = useState<string>("");

  const asked = useMemo(() => new Set(log.filter((m) => m.predicate).map((m) => predicateKey(m.predicate!))), [log]);
  const attr = attributes.find((a) => a.key === attrKey) ?? null;

  function pick(a: AttrView) {
    setAttrKey(a.key);
    setOp(a.ops[0] ?? "eq");
    setValue2("");
    if (a.type === "int") {
      const steps = intSteps(a);
      setValue(String(steps[Math.floor(steps.length / 2)] ?? a.min ?? 0));
      setValue2(String(steps[steps.length - 1] ?? a.max ?? 0));
    } else {
      setValue("");
    }
  }

  const predicate: Predicate | null = useMemo(() => {
    if (!attr) return null;
    switch (attr.type) {
      case "bool":
        return { attr: attr.key, op: "is", value: true };
      case "int": {
        const a = Number(value);
        if (op === "between") {
          const b = Number(value2);
          return a <= b ? { attr: attr.key, op, value: [a, b] } : null;
        }
        return { attr: attr.key, op, value: a };
      }
      case "string": {
        const v = value.trim();
        return v && /^[\p{L}\p{N}]{1,3}$/u.test(v) ? { attr: attr.key, op: "starts_with", value: v } : null;
      }
      default:
        return value ? { attr: attr.key, op: attr.ops[0] ?? "eq", value } : null;
    }
  }, [attr, op, value, value2]);

  const repeat = predicate ? asked.has(predicateKey(predicate)) : false;

  function submit() {
    if (!predicate || repeat) return;
    onAsk(predicate);
    if (attr?.type !== "int") setValue("");
  }

  return (
    <div>
      <p className="font-semibold" id="ask-about">
        Ask about
      </p>
      <div className="mt-2 flex flex-wrap gap-2" role="group" aria-labelledby="ask-about">
        {attributes.map((a) => (
          <button key={a.key} type="button" className="chip" aria-pressed={a.key === attrKey} onClick={() => pick(a)}>
            {a.label}
          </button>
        ))}
      </div>

      {attr && (
        <div className="mt-4 border-t border-rule-soft pt-4">
          {attr.help && <p className="mb-3 text-sm text-ink-soft">{attr.help}</p>}

          {(attr.type === "enum" || attr.type === "set") && (
            <div className="flex flex-wrap gap-2" role="group" aria-label={`${attr.label} value`}>
              {attr.values?.map((v) => {
                const done = asked.has(predicateKey({ attr: attr.key, op: attr.ops[0] ?? "eq", value: v }));
                return (
                  <button
                    key={v}
                    type="button"
                    className={`chip ${done ? "line-through opacity-50" : ""}`}
                    aria-pressed={value === v}
                    disabled={done}
                    title={done ? "Already asked" : undefined}
                    onClick={() => setValue(v)}
                  >
                    {v}
                  </button>
                );
              })}
            </div>
          )}

          {attr.type === "int" && (
            <div className="flex flex-wrap items-center gap-2">
              <span>{attr.label} is</span>
              <select className="field w-auto" value={op} onChange={(e) => setOp(e.target.value)} aria-label="Comparison">
                {attr.ops.map((o) => (
                  <option key={o} value={o}>
                    {OP_LABEL[o] ?? o}
                  </option>
                ))}
              </select>
              <select className="field w-auto tabular" value={value} onChange={(e) => setValue(e.target.value)} aria-label="Value">
                {intSteps(attr).map((v) => (
                  <option key={v}>{v}</option>
                ))}
              </select>
              {op === "between" && (
                <>
                  <span>and</span>
                  <select
                    className="field w-auto tabular"
                    value={value2}
                    onChange={(e) => setValue2(e.target.value)}
                    aria-label="Upper value"
                  >
                    {intSteps(attr).map((v) => (
                      <option key={v}>{v}</option>
                    ))}
                  </select>
                </>
              )}
            </div>
          )}

          {attr.type === "string" && (
            <label className="flex flex-wrap items-center gap-2">
              <span>{attr.label} starts with</span>
              <input
                className="field w-28 uppercase tracking-widest"
                maxLength={3}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && submit()}
                placeholder="A"
              />
              <span className="text-sm text-ink-soft">1 to 3 letters</span>
            </label>
          )}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button className="btn btn-ink" disabled={!predicate || repeat || busy} onClick={submit}>
              {busy ? "Asking…" : "Ask (1 move)"}
            </button>
            {predicate && (
              <span className={repeat ? "text-sm text-no" : "text-ink-soft"}>
                {repeat ? "You already asked this." : preview(attr, predicate)}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
