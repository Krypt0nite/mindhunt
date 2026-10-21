# MindHunt

A two-player deduction game. Each player secretly picks a candidate from a topic pack, then both hunt the other's secret at the same time by asking yes/no questions. Fewer moves wins.

The working title comes from `APP_NAME`; change it there.

**Play:** https://mindhunt.combio-worker.workers.dev. It runs free on Cloudflare: one Worker serves the web client, and a Durable Object runs this Go server compiled to WebAssembly. See [worker/README.md](worker/README.md) to deploy it.

## Run it

Requirements: Go 1.24+, Node 22+, optionally PostgreSQL 14+.

```bash
make build        # builds web/out and bin/mindhunt
make run          # serves everything on http://localhost:8080 (memory only)

# with persistence
DATABASE_URL="postgres://user:pass@localhost:5432/mindhunt?sslmode=disable" make run
```

Open http://localhost:8080 in two different browsers (or one normal and one private window) to play yourself.

Docker: `docker compose up --build` starts Postgres and the server on port 8080.

### Development

```bash
cd server && go run ./cmd/mindhunt                          # API on :8080
cd web && NEXT_PUBLIC_API_BASE=http://localhost:8080 npm run dev   # UI on :3000
# the server must allow the dev origin:
ALLOWED_ORIGINS=localhost:3000 go run ./cmd/mindhunt
```

### Tests

```bash
make test                     # Go unit + room actor + WebSocket end-to-end tests, tsc
make test-db                  # also the Postgres store test (needs MINDHUNT_TEST_DATABASE_URL)
make e2e                      # two real browsers against a running server (see tools/e2e)
make verify-data              # cricket evidence vs pack, verified workbook, portrait pipeline tests
```

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `APP_NAME` | `MindHunt` | Name shown in the UI |
| `DATABASE_URL` | (empty) | Postgres URL; empty means guests and results live in memory |
| `WEB_DIR` | `../web/out` | Built web client |
| `ASSETS_DIR` | `../assets` | Licensed images served at `/assets/` (results only) |
| `ADMIN_TOKEN` | (off) | Enables `/api/admin/*` (reported packs) with header `X-Admin-Token` |
| `BLOCKED_WORDS` | | Extra comma-separated words refused in player-made packs |
| `PACKS_DIR` | (embedded) | Load packs from a folder instead of the built-in ones |
| `PUBLIC_URL` | request host | Base for invite links behind a proxy |
| `ALLOWED_ORIGINS` | same host only | Extra WebSocket origins, comma separated |
| `WRONG_GUESS_COST` | `3` | Moves charged for a wrong guess |
| `RECONNECT_GRACE_S` | `90` | Seconds before the other player can claim the win |
| `ROOMS_PER_IP_HOUR` | `20` | Room creation limit per IP |
| `DAILY_KEY` | dev key (warns) | Secret that picks each day's answer. **Set this in production.** |
| `DAILY_TZ` | `Asia/Kolkata` | When the daily rolls over (local midnight) |
| `DEFAULT_PACK` | `cricket-men-intl` | Pack new rooms start with |
| `DAILY_PACK` | `rotate` | A pack id, or `rotate` to use a different pack each day (default pack first) |
| `DAILY_EPOCH` | `2026-10-18` | Date of Daily #1 |
| `DAILY_POOL` | `48` | Candidates in each daily |

## How it is built

```
server/                 Go: one binary for REST, WebSocket and static files
  internal/game         pure rules: packs, predicates, Eval, scoring (no I/O)
  internal/room         one goroutine per room; per-player views; timers; bot seat
  internal/bot          practice opponent (minimises expected remaining candidates)
  internal/transport    REST, WebSocket, static client, rate limits
  internal/store        guests + match log; Postgres or memory
  packs/                topic packs (JSON, embedded)
web/                    Next.js static export, TypeScript, Tailwind
tools/                  pack generator, two-browser e2e script
```

- **Questions are predicates** (`{attr, op, value}`) answered by the server from the pack.
- **Typed and spoken questions** go through a text matcher (`internal/game/matcher.go`). If the words match a known question exactly, the data answers it. A looser match is only offered as a suggestion the player confirms. With **free text** on (host setting, not in practice), anything else is sent to the opponent, who answers Yes, No or Not sure (3 Not sure per round). The asker's move timer pauses while waiting.
- **Audit**: at the reveal, manual answers the matcher can read are checked against the data; two or more contradictions hand the win to the other player.
- **Daily challenge** (`internal/daily`, page `/daily`): one secret per day for everyone, picked by an HMAC of the date under `DAILY_KEY`, so it is stable across restarts but not predictable from the code. Progress is stored as a move log and rebuilt by replay; results show the score distribution, percentile and streak.
- **Voice**: push-to-talk through the browser's speech recognition (Chrome, Edge, Safari; not Firefox). The transcript only fills the question box; nothing is sent until the player presses a button.
- **Privacy**: every state message is built by `viewFor(seat)`. The opponent's secret, questions, move count and image never leave the server before the round ends. Tested in `room_test.go` and in the browser run.
- **Idempotency**: each message carries an increasing `cseq` per connection; repeats are dropped.
- **Reconnects**: the server sends a full snapshot after every change and on reconnect, so there is no event replay to get wrong.

### Topic packs

| Category | Pack | Candidates |
| --- | --- | --- |
| Sports | International Cricketers | 72 |
| Sports | Footballers | 37 |
| Science | Chemical Elements | 52 |
| Science | Scientists | 43 |
| Technology | Programming Languages | 31 |
| History | Historical Figures | 44 |
| Geography | Countries | 53 |
| Economics | Economists | 20 |

All packs are marked draft until every row is verified. Not yet covered: Entertainment (Bollywood, films, music), where facts about living people change often; it fits the same format once verified data is supplied.

### Adding or verifying a pack

- New packs are compact tables in `tools/packs/<name>.py`; `python3 tools/build_packs.py` compiles them to `server/packs/*.json`.
- `python3 tools/pack_sheet.py export tools/packs/<name>.py sheet.xlsx` makes a verification spreadsheet; `import` checks an edited sheet row by row and writes it back into the pack source.
- Cricket uses `tools/gen_cricket_pack.py` with `tools/export_cricket_xlsx.py` / `import_cricket_xlsx.py`. Its evidence lives in `tools/cricket_evidence.py`; `python3 tools/verify_cricket.py` checks the pack against it and writes `data/cricket/` (verified workbook, audit CSV/JSON, flat dataset, image attribution). See `data/cricket/VERIFICATION_REPORT.md`.
- Portraits: `tools/fetch_portraits.py fetch` (needs access to Wikipedia/Commons; set `MINDHUNT_CONTACT`) finds candidates, checks licences, downloads and decodes them to `assets/`, and records everything in `tools/portraits/cricket-manifest.json`. A person then runs `confirm <id>` for pictures that really show the player; only those reach the pack.
- `go test ./internal/game/ ./internal/bot/` checks every pack: no two candidates identical, every attribute useful, every question text recognised, and the bot solving each pack in a sensible number of moves.

### Player-made packs

Anyone can build a pack at `/create/`: a title, up to 10 facts (yes/no, choice from a list, or a number) with optional question wording, and 8–150 rows, typed in or pasted from a spreadsheet. The server checks the draft as it is typed (lengths, links, emails, phone numbers, a word list, values that don't match the facts, look-alike rows) and compiles it into a normal pack, adding a "first letters of the name" question. Packs are unlisted: they are shared by link (`/pack/?id=u-…`), played with a friend or the bot, and never appear in the public list or the daily.

- Editing saves a new version under the same link; item ids are kept, and rounds already running keep their version.
- Limits: 30 packs per player, 20 saves an hour.
- Moderation: any player can report a pack; three different reporters hide it until an admin restores it (`GET /api/admin/reports`, `POST /api/admin/custom/{id}` with `{"hidden": false}`). Owners can delete their packs.
- Rooms never wait on the database: a pack is loaded when its link is opened and kept in memory (most recent 500).

### Adding a category

Write a pack JSON (see `server/packs/cricket-men-intl.v3.json` and `tools/gen_cricket_pack.py`): define attributes (`enum`, `bool`, `int`, `set`, `string`), optional question templates, and candidates. No code changes. Bump `version` when data changes; running matches keep the version they started with.

## Status: what is real, mocked or missing

Working and tested: typed questions (auto-answered when recognised), free-text questions with manual answers, the Not sure cap and reveal-time audit, voice input UI, private rooms with invite links, guest identities, lobby with host settings, secret picking, simultaneous hunting with server-answered questions, 3-move wrong-guess penalty, give up, optional move timer, elimination board, scoring with time tie-break and draws, disconnect grace + claim win, void when both leave, rematch, practice bot, results with fact cards and spoiler-free share text, responsive layouts, Postgres persistence.

Needs attention before a public launch:

- **Cricket data: 64 of 72 players fully verified, 8 role calls await a decision** (see `data/cricket/VERIFICATION_REPORT.md`). The pack stays marked draft until they are signed off. Most facts rest on Wikipedia because ESPNcricinfo profiles could not be opened from the build environment.
- **No portraits yet.** Serving, attribution, reveal-only delivery and the initials fallback are built and tested, but Wikimedia Commons was unreachable here, so no image was verified. Run `tools/fetch_portraits.py` where Commons is reachable.
- **Live rooms are in memory.** A server restart ends running rounds; guests and finished results survive in Postgres.
- **Voice input is untested with a real microphone** (headless browsers have none). Recognition quality for Indian accents depends on the browser; server-side speech-to-text would need an API key.
- **Daily streaks belong to a guest identity** (one browser). Clearing site data or switching devices starts a new streak, and a new guest could replay the day; accounts would fix both.
- **Not built yet:** accounts and rankings. Player-made packs belong to a guest identity (one browser), like daily streaks; there is no admin screen yet, only the two admin endpoints.
- Protocol types are mirrored by hand in `web/lib/protocol.ts`; keep them in sync with `server/internal/room/protocol.go`.
- The Dockerfile and compose file were written but not built here (no Docker daemon in the build environment).
