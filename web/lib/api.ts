import type {
  CustomPackView,
  DailyAction,
  DailyView,
  DraftCheck,
  DraftIssue,
  DraftStats,
  Guest,
  MatchView,
  MyPack,
  PackDraft,
  PackSummary,
  PublicConfig,
  SavedPack,
} from "./protocol";

const GUEST_KEY = "mindhunt.guest";

/** API origin. Empty means same origin (production, served by the Go server). */
export function apiBase(): string {
  return (process.env.NEXT_PUBLIC_API_BASE ?? "").replace(/\/$/, "");
}

export function wsURL(room: string, token: string): string {
  const base = apiBase() || window.location.origin;
  const u = new URL(base);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  u.pathname = "/ws";
  u.search = new URLSearchParams({ room, token }).toString();
  return u.toString();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    /** Extra fields from the error body (a rejected pack's issues). */
    public body: Record<string, unknown> = {},
  ) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown, token?: string): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers["Authorization"] = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(apiBase() + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "network", "Can't reach the game server. Check your connection and try again.");
  }
  const data = res.status === 204 ? {} : await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, data.code ?? "error", data.message ?? `Request failed (${res.status})`, data);
  }
  return data as T;
}

export function storedGuest(): Guest | null {
  try {
    const raw = localStorage.getItem(GUEST_KEY);
    return raw ? (JSON.parse(raw) as Guest) : null;
  } catch {
    return null;
  }
}

function saveGuest(g: Guest) {
  try {
    localStorage.setItem(GUEST_KEY, JSON.stringify(g));
  } catch {
    /* private mode: guest lasts for this tab only */
  }
}

export function forgetGuest() {
  try {
    localStorage.removeItem(GUEST_KEY);
  } catch {
    /* ignore */
  }
}

export async function createGuest(name: string): Promise<Guest> {
  const g = await request<Guest>("POST", "/api/guest", { display_name: name });
  saveGuest(g);
  return g;
}

/** Returns the stored guest if the server still knows its token. */
export async function verifyGuest(g: Guest): Promise<Guest | null> {
  try {
    const me = await request<Guest>("GET", "/api/me", undefined, g.token);
    const merged = { ...g, display_name: me.display_name };
    saveGuest(merged);
    return merged;
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      forgetGuest();
      return null;
    }
    throw e;
  }
}

export async function renameGuest(g: Guest, name: string): Promise<Guest> {
  const me = await request<Guest>("PATCH", "/api/me", { display_name: name }, g.token);
  const merged = { ...g, display_name: me.display_name };
  saveGuest(merged);
  return merged;
}

export function createRoom(g: Guest, practice: boolean, packId?: string) {
  return request<{ code: string; invite_url: string; practice: boolean }>(
    "POST",
    "/api/rooms",
    packId ? { practice, pack_id: packId } : { practice },
    g.token,
  );
}

export function roomInfo(code: string) {
  return request<{ phase: string; players: number; full: boolean; practice: boolean }>(
    "GET",
    `/api/rooms/${encodeURIComponent(code)}`,
  );
}

export function listPacks() {
  return request<PackSummary[]>("GET", "/api/packs");
}

export function publicConfig() {
  return request<PublicConfig>("GET", "/api/config");
}

export function getDaily(g: Guest) {
  return request<DailyView>("GET", "/api/daily", undefined, g.token);
}

/** Rejects with ApiError (status 422, a game error code) for illegal moves. */
export function playDaily(g: Guest, action: DailyAction) {
  return request<DailyView>("POST", "/api/daily/play", action, g.token);
}

export function matchDaily(text: string) {
  return request<MatchView>("GET", `/api/daily/match?text=${encodeURIComponent(text)}`);
}

// ---- Player-made packs ----

export function checkPack(g: Guest, draft: PackDraft) {
  return request<DraftCheck>("POST", "/api/custom/check", { draft }, g.token);
}

/** A rejected pack throws ApiError with status 422 and body.issues. */
export function savePack(g: Guest, draft: PackDraft, id?: string) {
  return id
    ? request<SavedPack>("PUT", `/api/custom/${encodeURIComponent(id)}`, { draft }, g.token)
    : request<SavedPack>("POST", "/api/custom", { draft }, g.token);
}

export function rejectedIssues(e: unknown): { issues: DraftIssue[]; stats?: DraftStats } | null {
  if (e instanceof ApiError && e.status === 422 && Array.isArray(e.body.issues)) {
    return { issues: e.body.issues as DraftIssue[], stats: e.body.stats as DraftStats | undefined };
  }
  return null;
}

export function getPack(id: string, g?: Guest | null) {
  return request<CustomPackView>("GET", `/api/custom/${encodeURIComponent(id)}`, undefined, g?.token);
}

export function myPacks(g: Guest) {
  return request<MyPack[]>("GET", "/api/custom", undefined, g.token);
}

export function deletePack(g: Guest, id: string) {
  return request<object>("DELETE", `/api/custom/${encodeURIComponent(id)}`, undefined, g.token);
}

export function reportPack(g: Guest, id: string, reason: string) {
  return request<{ reported: boolean; hidden: boolean }>(
    "POST",
    `/api/custom/${encodeURIComponent(id)}/report`,
    { reason },
    g.token,
  );
}

/** Finds a player-made pack id in a pasted link or code. */
export function packIdFrom(text: string): string | null {
  const m = text.trim().match(/u-[a-z2-9]{10}/);
  return m ? m[0] : null;
}
