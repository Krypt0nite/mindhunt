# Continuing on your computer (VS Code)

State on 18 Oct 2026: all 8 packs checked against sources (audits in `data/<pack>/`),
cricket fully verified, custom packs built, all tests passing. What's left needs a
machine that can reach Wikimedia, run Docker, or both.

## 1. Set up (once)

Needs Go 1.24+, Node 22+, Python 3.11+, PostgreSQL 16 (or Docker).

```bash
cd web && npm ci && npx next build && cd ..
cd server && go test ./... && cd ..
pip install openpyxl pillow
python3 -m unittest discover -s tools/tests
```

Run it: `cd server && WEB_DIR=../web/out go run ./cmd/mindhunt` and open http://localhost:8080
(add `DATABASE_URL=postgres://…` to keep guests, results and custom packs).

## 2. Cricket player photos (about 15 minutes)

The licence check ran for all 72 players in the cloud: 62 photos passed and are on the review page. Nine Indian players (Kohli, Rohit, Dravid, Bumrah, Jadeja, Ashwin, Harbhajan, Hardik, Shami) were checked before the Indian government licence (GODL-India) was allowed, and Waqar Younis has no Wikipedia photo; the re-check below covers them.
Wikimedia blocks image downloads from cloud servers, so finish here:

```bash
export MINDHUNT_CONTACT="you@example.com"     # Wikimedia asks for a contact in the User-Agent
python3 tools/fetch_portraits.py fetch --search --no-download   # re-check; picks up the nine GODL-India photos and searches Commons for the rest
python3 tools/fetch_portraits.py review       # writes data/cricket/portrait-review.html
```

Open `data/cricket/portrait-review.html` in your browser. Tick only photos that clearly
show the named player on his own, press "Copy confirm command" and run it. Then:

```bash
python3 tools/fetch_portraits.py download     # downloads, checks the file opens, saves assets/cricket-men-intl/*.webp
# bump "version" in tools/gen_cricket_pack.py (3 -> 4) and rename the JSON, then:
python3 tools/gen_cricket_pack.py > server/packs/cricket-men-intl.v4.json && git rm server/packs/cricket-men-intl.v3.json
python3 tools/verify_cricket.py               # fails if any image breaks the rules
```

Unticked players keep their initials. Never tick a group photo or a doubtful one.

## 3. Footballers: stronger foot (24 players)

The articles don't say which foot for 24 players, so the pack stays marked draft.
Add them to `FOOT` in `tools/evidence/football.py` with a source, then
`python3 tools/verify_pack.py football --baseline d569dca` and, when it shows 0 not
checked, set `"draft": False` in `tools/packs/football.py` and run `python3 tools/build_packs.py`.

## 4. Docker (never tested)

```bash
DAILY_KEY=$(openssl rand -hex 32) docker compose up --build
```

Then play a round on two phones, and try voice input with a real microphone.

## 5. Before going public

- Hosting is done: Cloudflare Workers + Durable Objects, free (see `worker/README.md`). `DAILY_KEY` is set as a secret; set `ADMIN_TOKEN` the same way when you need moderation.
- Decide on accounts (Google sign-in, phone OTP or email link) so streaks and custom packs follow players across devices.

## Re-checking data

```bash
python3 tools/verify_cricket.py
for p in countries elements history scientists economists languages; do python3 tools/verify_pack.py $p; done
python3 tools/verify_pack.py football --baseline d569dca
```

Each writes `data/<pack>/…-verified.xlsx` (Pack, Audit, Methodology) and fails if the
pack and its evidence (`tools/evidence/`, `tools/cricket_evidence.py`) disagree.
