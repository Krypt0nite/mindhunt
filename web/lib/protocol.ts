// Wire types. Mirrors server/internal/room/protocol.go and game types —
// keep the two in sync when either changes.

export type Phase = "lobby" | "picking" | "hunting" | "results" | "void";
export type SeatStatus = "picking" | "picked" | "hunting" | "solved" | "gave_up";
export type Answer = "yes" | "no" | "unknown" | "correct" | "wrong";
export type AttrType = "enum" | "bool" | "int" | "set" | "string";
export type MoveKind = "ask" | "ask_free" | "guess" | "give_up" | "timeout";

export interface Predicate {
  attr: string;
  op: string;
  value: string | number | boolean | [number, number];
}

export interface Move {
  n: number;
  kind: MoveKind;
  predicate?: Predicate;
  text: string;
  guess_id?: string;
  answer?: Answer;
  cost: number;
  total: number;
  at: string;
  removed?: number;
  /** Answered by the opponent by hand, not from the data. */
  manual?: boolean;
  /** The player's own wording when the question was typed or spoken. */
  asked?: string;
}

export interface Pending {
  id: number;
  text: string;
  at: string;
}

export interface Flag {
  n: number;
  text: string;
  answered: Answer;
  data: Answer;
  read_as: string;
}

export interface MatchView {
  text?: string;
  predicate?: Predicate;
  exact: boolean;
}

export interface Config {
  pack_id: string;
  difficulty: "easy" | "normal" | "hard";
  timer_s: number;
  free_text: boolean;
}

export interface ConfigView extends Config {
  pack_title: string;
  pack_version: number;
  pack_draft: boolean;
  pack_author?: string;
  pool_size: number;
  wrong_guess_cost: number;
}

export interface PlayerView {
  seat: number;
  name: string;
  host: boolean;
  bot: boolean;
  connected: boolean;
  ready: boolean;
  status?: SeatStatus;
  claimable_at?: string;
}

export interface HuntView {
  your_secret_id?: string;
  your_status: SeatStatus;
  your_moves: number;
  your_log: Move[];
  remaining_ids: string[];
  deadline?: string;
  /** Your typed question waiting for the opponent. */
  pending?: Pending;
  /** The opponent's typed question you must answer. */
  incoming?: Pending;
  unknowns_left: number;
}

export interface AttrLine {
  label: string;
  value: string;
}

export interface ImageView {
  url: string;
  license: string;
  license_url?: string;
  credit: string;
  source_url: string;
}

export interface CandidateCard {
  id: string;
  name: string;
  attrs: AttrLine[];
  facts?: string[];
  image?: ImageView;
}

export interface ResultPlayer {
  seat: number;
  name: string;
  bot: boolean;
  secret: CandidateCard;
  status: SeatStatus;
  moves: number;
  score: number;
  active_ms: number;
  log: Move[];
  cracked_by?: Move;
  grid: string;
  answer_flags?: Flag[];
}

export type Reason = "fewer_moves" | "faster" | "tie" | "both_gave_up" | "abandoned" | "contradictions";

export interface ResultView {
  winner: number;
  reason: Reason;
  players: [ResultPlayer, ResultPlayer];
  ended_at: string;
}

export interface StateView {
  room: string;
  phase: Phase;
  practice: boolean;
  round: number;
  you: PlayerView;
  opponent: PlayerView | null;
  config: ConfigView;
  hunt?: HuntView;
  result?: ResultView;
  rematch_you: boolean;
  rematch_opponent: boolean;
  server_time: string;
  grace_s: number;
}

export interface AttrView {
  key: string;
  label: string;
  type: AttrType;
  values?: string[];
  min?: number;
  max?: number;
  step?: number;
  help?: string;
  ops: string[];
}

export interface PoolCandidate {
  id: string;
  name: string;
  attrs: Record<string, string>;
}

export interface PoolView {
  round: number;
  pack_id: string;
  pack_title: string;
  version: number;
  as_of: string;
  attributes: AttrView[];
  candidates: PoolCandidate[];
}

export interface GameError {
  code: string;
  message: string;
}

export interface ServerMsg<T = unknown> {
  type: "state" | "pool" | "answer" | "guess_result" | "error" | "pong" | "kicked" | "asked" | "match";
  seq: number;
  reply_to?: number;
  data: T;
}

export type ClientMsgType =
  | "set_config"
  | "ready"
  | "start"
  | "pick_secret"
  | "ask"
  | "guess"
  | "give_up"
  | "claim_win"
  | "rematch"
  | "to_lobby"
  | "leave"
  | "ping"
  | "ask_text"
  | "answer_text"
  | "cancel_text"
  | "match_text";

export interface PackSummary {
  id: string;
  version: number;
  as_of: string;
  category: string;
  title: string;
  noun: string;
  description: string;
  draft: boolean;
  candidates: number;
}

export interface Guest {
  player_id: string;
  token?: string;
  display_name: string;
}

export interface PublicConfig {
  app_name: string;
  wrong_guess_cost: number;
  reconnect_grace_s: number;
  pool_sizes: Record<string, number>;
  timers: number[];
}

/** WebSocket close codes sent by the server. */
export const CloseCodes = {
  roomFull: 4001,
  replaced: 4002,
  unauthorized: 4003,
  roomNotFound: 4004,
} as const;

export interface DailyStats {
  finished: number;
  solved: number;
  /** moves -> number of players who solved in that many */
  scores: Record<string, number>;
}

export interface DailyResult {
  secret: CandidateCard;
  score: number;
  grid: string;
  cracked_by?: Move;
  stats: DailyStats;
  better_than: number;
}

export interface Streak {
  current: number;
  best: number;
  played: number;
  solved: number;
}

export interface DailyView {
  date: string;
  number: number;
  next: string;
  pack_id: string;
  pack_title: string;
  pack_noun: string;
  category: string;
  pool?: PoolView;
  status: SeatStatus;
  moves: number;
  log: Move[];
  remaining_ids: string[];
  wrong_guess_cost: number;
  last?: Move;
  result?: DailyResult;
  streak: Streak;
}

export interface DailyAction {
  predicate?: Predicate;
  text?: string;
  candidate_id?: string;
  give_up?: boolean;
}

// ---- Player-made packs ----

export type AttrKind = "yesno" | "choice" | "number";

export interface DraftAttr {
  label: string;
  kind: AttrKind;
  choices?: string[];
  question?: string;
}

export interface DraftItem {
  name: string;
  values: string[];
}

export interface PackDraft {
  title: string;
  noun: string;
  description: string;
  attributes: DraftAttr[];
  items: DraftItem[];
}

export interface DraftIssue {
  path: string;
  message: string;
  severity: "error" | "warning";
}

export interface DraftStats {
  items: number;
  attributes: number;
  unique: number;
  avg_moves_est: number;
}

export interface DraftCheck {
  ok: boolean;
  issues: DraftIssue[];
  stats: DraftStats;
}

export interface SavedPack {
  id: string;
  version: number;
  issues: DraftIssue[];
  stats: DraftStats;
  share_url: string;
}

export interface CustomPackView extends PackSummary {
  author: string;
  attributes: string[];
  items: string[];
  is_owner: boolean;
  hidden: boolean;
  reports?: number;
  edit_draft?: PackDraft;
  share_url: string;
}

export interface MyPack {
  id: string;
  title: string;
  author: string;
  version: number;
  hidden: boolean;
  reports: number;
  updated_at: string;
}
