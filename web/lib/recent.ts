// Recent results kept in the browser for the home screen. No server storage.

export interface RecentResult {
  room: string;
  round: number;
  pack: string;
  opponent: string;
  outcome: "won" | "lost" | "draw";
  moves: number;
  opponentMoves: number;
  practice: boolean;
  at: string;
}

const KEY = "mindhunt.recent";

export function loadRecent(): RecentResult[] {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? "[]") as RecentResult[];
  } catch {
    return [];
  }
}

export function saveRecent(r: RecentResult) {
  try {
    const list = loadRecent().filter((x) => !(x.room === r.room && x.round === r.round && x.at === r.at));
    if (list.some((x) => x.room === r.room && x.round === r.round)) return; // already saved this round
    localStorage.setItem(KEY, JSON.stringify([r, ...list].slice(0, 20)));
  } catch {
    /* storage unavailable */
  }
}

/** Current run of consecutive wins against people (practice excluded). */
export function winStreak(list: RecentResult[]): number {
  let n = 0;
  for (const r of list) {
    if (r.practice) continue;
    if (r.outcome !== "won") break;
    n++;
  }
  return n;
}
