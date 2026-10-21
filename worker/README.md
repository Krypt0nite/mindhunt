# MindHunt on Cloudflare (free plan)

Live: https://mindhunt.combio-worker.workers.dev

Everything runs on one Cloudflare Worker, with no credit card and no server to keep running:

```
browser ──► Worker "mindhunt"
             ├─ static files (web/out, assets/)   Workers Static Assets: free, unmetered
             └─ /api/*, /ws ──► Durable Object "hub"
                                 Go game server compiled to WebAssembly
                                 (server/cmd/mindhunt-wasm)
                                 rooms: in memory
                                 guests, daily plays, player packs: SQLite storage
```

- The game code is the same Go code as the normal server (`server/internal/...`). Only the shell
  is different: the Durable Object owns HTTP and WebSockets and calls into Go, and Go keeps its data
  in the Durable Object's SQLite storage instead of Postgres (`cmd/mindhunt-wasm/sqlstore.go`).
- There is one hub for all rooms, like the single Go process. One object running all day uses
  about 11,000 of the free plan's 13,000 GB-seconds a day, so it can never go over.
- Same limits as the Go server, plus the free plan's: 100,000 Worker requests a day
  (WebSocket messages count 1 per 20), 100,000 storage row writes a day. Match history is not
  recorded on Cloudflare (it is only ever read from Postgres) to save writes.

## Deploy

Needs Go 1.24+ and Node 22+. `wrangler` must be logged in (`npx wrangler login`).

```bash
cd worker
npm install
npm run deploy          # builds web/out, the wasm and public/, then wrangler deploy
node smoke.mjs https://mindhunt.combio-worker.workers.dev   # end-to-end check (adds a few test guests)
```

Local run on the real Workers runtime: `npm run dev`, then open http://127.0.0.1:8787 and
`node smoke.mjs` in another terminal.

### Settings

Plain settings go in `vars` in `wrangler.jsonc` (same names as the README's Configuration
table: `APP_NAME`, `DEFAULT_PACK`, `DAILY_TZ`, `WRONG_GUESS_COST`, ...). Secrets:

```bash
npx wrangler secret put DAILY_KEY     # already set to a random value; changing it changes today's answer
npx wrangler secret put ADMIN_TOKEN   # turns on /api/admin/* (reported packs)
```

`PORT`, `DATABASE_URL`, `WEB_DIR`, `ASSETS_DIR` and `PACKS_DIR` do not apply here.

### Auto-deploy on push (optional)

`.github/workflows/deploy.yml` tests and deploys on every push to `main` once the repository has
two secrets (Settings → Secrets and variables → Actions):

- `CLOUDFLARE_API_TOKEN`: dash.cloudflare.com → My Profile → API Tokens → Create Token →
  "Edit Cloudflare Workers" template.
- `CLOUDFLARE_ACCOUNT_ID`: shown by `npx wrangler whoami`.

Without them the workflow only runs the tests.

## Things to know

- A deploy restarts the hub, which ends rounds that are being played (the same as restarting the
  Go server). Guests, daily streaks and player packs are kept.
- The hub can be evicted after a few minutes with nobody connected; rooms nobody is in are lost,
  stored data is not.
- Images: put licensed files in `assets/<pack>/<id>.webp` as before; the build copies them and tells
  the server which ones exist.
