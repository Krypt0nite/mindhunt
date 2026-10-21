// End-to-end check of a running deployment (local `wrangler dev` or live),
// following server/internal/transport/e2e_test.go: REST, a full two-player
// round over WebSockets, close codes, a practice round against the bot, and
// the daily challenge.
//
// Usage: node smoke.mjs [base URL]   (default http://127.0.0.1:8787)
const base = (process.argv[2] || "http://127.0.0.1:8787").replace(/\/$/, "");

function check(ok, what) {
  if (!ok) throw new Error("FAILED: " + what);
  console.log("ok   " + what);
}

async function api(method, path, token, body) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = "Bearer " + token;
  const res = await fetch(base + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  const data = await res.json().catch(() => ({}));
  return { status: res.status, data };
}

function dial(code, token) {
  const url = base.replace(/^http/, "ws") + "/ws?" + new URLSearchParams({ room: code, token });
  const ws = new WebSocket(url);
  const c = { ws, cseq: 0, queue: [], waiters: [], closeCode: null, closed: null };
  c.closed = new Promise((resolve) => ws.addEventListener("close", (e) => resolve((c.closeCode = e.code))));
  ws.addEventListener("message", (e) => {
    c.queue.push(e.data);
    c.waiters.forEach((w) => w());
  });
  c.opened = new Promise((resolve, reject) => {
    ws.addEventListener("open", resolve);
    ws.addEventListener("error", reject);
  });
  c.send = (type, data) => ws.send(JSON.stringify({ type, cseq: ++c.cseq, data: data ?? null }));
  // next waits for a message of this type whose raw JSON contains every needle.
  c.next = (type, ...needles) =>
    new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`timed out waiting for ${type} ${needles.join(" ")}`)), 90_000); // the bot takes ~3 s a move
      const look = () => {
        while (c.queue.length) {
          const raw = c.queue.shift();
          const m = JSON.parse(raw);
          if (m.type === type && needles.every((n) => raw.includes(n))) {
            clearTimeout(timer);
            c.waiters = c.waiters.filter((w) => w !== look);
            resolve(raw);
            return;
          }
        }
      };
      c.waiters.push(look);
      look();
    });
  return c;
}

const t0 = Date.now();
const home = await fetch(base + "/");
check(home.status === 200 && (await home.text()).includes("<html"), "home page");
check((await fetch(base + "/room/?code=AAAAAA")).status === 200, "room page");
const health = await api("GET", "/api/health");
check(health.status === 200 && health.data.ok, "health");
const cfg = await api("GET", "/api/config");
check(cfg.data.app_name === "MindHunt", "config");
check((await api("POST", "/api/guest", "", { display_name: "" })).status === 400, "empty name refused");

const ga = (await api("POST", "/api/guest", "", { display_name: "Asha" })).data;
const gb = (await api("POST", "/api/guest", "", { display_name: "Bilal" })).data;
const gc = (await api("POST", "/api/guest", "", { display_name: "Chen" })).data;
check(ga.token && ga.player_id, "guest created");
const me = await api("GET", "/api/me", ga.token);
check(me.status === 200 && me.data.display_name === "Asha" && !me.data.token, "me");
check((await api("PATCH", "/api/me", gc.token, { display_name: "Chen Li" })).data.display_name === "Chen Li", "rename");
check((await api("POST", "/api/rooms")).status === 401, "room needs a token");
const packs = (await api("GET", "/api/packs")).data;
check(packs.length >= 8 && packs.some((p) => p.id === "cricket-men-intl"), `packs (${packs.length})`);

const created = (await api("POST", "/api/rooms", ga.token, { practice: false })).data;
check(created.code?.length === 6 && created.invite_url.includes("/room/?code=" + created.code), "room created");

const a = dial(created.code, ga.token);
await a.opened;
const b = dial(created.code.toLowerCase(), gb.token);
await b.opened;
await a.next("state", '"name":"Bilal"');
check(true, "both players in the room");

const c3 = dial(created.code, gc.token);
check((await c3.closed) === 4001, "third player refused (4001)");
const bad = dial(created.code, "nope");
check((await bad.closed) === 4003, "bad token refused (4003)");
const none = dial("ZZZZZZ", ga.token);
check((await none.closed) === 4004, "unknown room refused (4004)");
const info = (await api("GET", "/api/rooms/" + created.code)).data;
check(info.full && info.phase === "lobby", "room info");

a.send("set_config", { pack_id: "cricket-men-intl", difficulty: "easy" });
await b.next("state", '"difficulty":"easy"');
b.send("ready", { ready: true });
await a.next("state", '"ready":true');
a.send("start");
const pool = JSON.parse(await a.next("pool")).data;
await b.next("pool");
check(pool.candidates.length === 24, "easy pool has 24");
const secretA = pool.candidates[0].id;
const secretB = pool.candidates[1].id;
a.send("pick_secret", { candidate_id: secretA });
b.send("pick_secret", { candidate_id: secretB });
await a.next("state", '"phase":"hunting"');
a.send("ask", { predicate: { attr: "country", op: "eq", value: "India" } });
check((await a.next("answer")).includes("Have they represented India?"), "question answered");
a.send("guess", { candidate_id: secretB });
await a.next("guess_result", '"answer":"correct"');
await b.next("state", '"status":"solved"');
b.send("give_up");
const res = await a.next("state", '"phase":"results"');
check(res.includes('"winner":0') && res.includes(secretA), "round finished, winner and reveal");
a.ws.close();
b.ws.close();

const prac = (await api("POST", "/api/rooms", ga.token, { practice: true })).data;
const p = dial(prac.code, ga.token);
await p.opened;
await p.next("state", '"bot":true');
p.send("start");
const ppool = JSON.parse(await p.next("pool")).data;
p.send("pick_secret", { candidate_id: ppool.candidates[3].id });
await p.next("state", '"phase":"hunting"');
await p.next("state", '"bot":true,"connected":true,"ready":true,"status":"solved"');
check(true, "practice bot solved its side");
p.ws.close();

check((await api("GET", "/api/daily")).status === 401, "daily needs a token");
const day = (await api("GET", "/api/daily", gb.token)).data;
check(day.pool?.candidates?.length > 0 && !day.result, `daily #${day.number} (${day.pack_id})`);
let solved = false;
for (const c of day.pool.candidates) {
  let r = await api("POST", "/api/daily/play", gb.token, { candidate_id: c.id });
  while (r.status === 429) {
    await new Promise((s) => setTimeout(s, 300));
    r = await api("POST", "/api/daily/play", gb.token, { candidate_id: c.id });
  }
  if (r.status !== 200) throw new Error("FAILED: daily guess, status " + r.status);
  if (r.data.status === "solved") {
    check(r.data.result?.secret?.id === c.id, "daily solved and revealed");
    solved = true;
    break;
  }
}
check(solved, "daily solvable");
const again = (await api("GET", "/api/daily", gb.token)).data;
check(again.status === "solved", "daily progress persisted");

// Player-made packs (stored in the Durable Object's SQLite).
const fruit = () => ({
  title: "Fruit bowl",
  noun: "fruit",
  attributes: [
    { label: "Colour", kind: "choice", choices: ["Red", "Yellow", "Green", "Orange"], question: "Is it {value}?" },
    { label: "Grows on trees", kind: "yesno", question: "Does it grow on trees?" },
    { label: "Seeds", kind: "number", question: "Does it have {value} seeds?" },
  ],
  items: ["Apple|Red|yes|5", "Banana|Yellow|no|0", "Cherry|Red|yes|1", "Lime|Green|yes|0", "Mango|Orange|yes|1",
    "Orange|Orange|yes|10", "Pear|Green|yes|6", "Kiwi|Green|no|99", "Lemon|Yellow|yes|8"].map((r) => {
    const [name, ...values] = r.split("|");
    return { name, values };
  }),
});
const made = await api("POST", "/api/custom", ga.token, { draft: fruit() });
check(made.status === 201 && made.data.id, "custom pack saved");
const view = (await api("GET", "/api/custom/" + made.data.id, gb.token)).data;
check(view.title === "Fruit bowl" && !view.is_owner, "custom pack visible by link");
const edit = fruit();
edit.items.push({ name: "Grape", values: ["Green", "no", "0"] });
const upd = await api("PUT", "/api/custom/" + made.data.id, ga.token, { draft: edit });
check(upd.status === 200 && upd.data.version === 2, "custom pack edited (version 2)");
check((await api("PUT", "/api/custom/" + made.data.id, gb.token, { draft: edit })).status === 403, "only the owner can edit");
const mine = (await api("GET", "/api/custom", ga.token)).data;
check(JSON.stringify(mine).includes(made.data.id), "my packs list");
const croom = await api("POST", "/api/rooms", gb.token, { practice: true, pack_id: made.data.id });
check(croom.status === 201, "room with a custom pack");
check((await api("DELETE", "/api/custom/" + made.data.id, ga.token)).status < 300, "custom pack deleted");

console.log(`\nall checks passed against ${base} in ${((Date.now() - t0) / 1000).toFixed(1)}s`);
