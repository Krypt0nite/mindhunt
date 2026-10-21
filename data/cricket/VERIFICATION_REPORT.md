# MindHunt Cricket: verification report

Checked on 18 October 2026. The pack is now `cricket-men-intl` version 2. All 72 player IDs and the roster order are unchanged.

## Summary

| # | Item | Result |
|---|------|--------|
| 1 | Player records checked | 72 of 72, 10 fields each (720 audit records) |
| 2 | Player records fully verified | 72 (the 8 role calls were decided by the pack owner on 18 Oct 2026). |
| 3 | Individual facts corrected | 8 values across 5 players (listed below) |
| 4 | Image candidates examined | 5 Commons file pages tried (the leads in the brief, covering 4 players). None could be opened. |
| 5 | Images verified and integrated | 0 |
| 6 | Players still missing a licensed image | 72 (68 Missing, plus 4 Needs review whose lead pages could not be opened) |
| 7 | Unresolved cricket-data claims | None. |
| 8 | Tests | All pass. See below. |
| 9 | Docker build | **Untested.** There is a Docker client but no daemon (`/var/run/docker.sock` does not exist). |
| 10 | Deliverables | Listed below |
| 11 | Public-launch blockers | Listed below |

## Verified results

### Corrections (the original values are kept in the Audit sheet)

| Player (ID) | Field | Was | Now | Evidence |
|---|---|---|---|---|
| Don Bradman (c_12ae0c) | ODI debut decade | 1920 | N/A | He never played an ODI: his career was 1928–48 and ODIs began in 1971. The game now answers Unknown to this question for him. |
| Clive Lloyd (c_dbbd1b) | ODI debut decade | 1960 | 1970 | Test debut 1966, first ODI 5 Sep 1973 |
| Muttiah Muralitharan (c_44c51f) | Name | Muttiah Muralidaran | Muttiah Muralitharan | Matches the roster and the Wikipedia article. The ID is kept from the old spelling. |
| Muttiah Muralitharan | Surname | Muralidaran | Muralitharan | Follows the corrected name |

### Checked against the definitions and unchanged

- **ODI debut decade.** Now means the decade of the first ODI, for any country. Every other player's value already matched this definition, including:
  - Eoin Morgan: first ODI for Ireland in 2006, so 2000s.
  - Dale Steyn: first ODI for the Africa XI in 2005, so 2000s.
- **Captained country.** All 72 values match the country captain lists. The formats are recorded per player on the Players sheet.
- **ODI World Cup winner.** All 72 values match the winning squads. These were checked against:
  - the list of finals;
  - the squad pages;
  - the official BCCI announcement (2011) and ICC announcements (2019 and 2023).
  - Shane Warne counts for 1999 only, because he withdrew before the 2003 tournament.
  - Waqar Younis was not in the 1992 squad.
- **Batting hand, bowling type and bowling arm.** All match the player infoboxes. For specialist batters and keepers, the workbook rule ("Not a regular bowler") still applies.

## Role decisions (18 Oct 2026)

The pack owner decided the 8 role calls: Yuvraj Singh All-rounder, Ravichandran Ashwin All-rounder (he said "bowling all-rounder"; the pack's nearest class), Wasim Akram Bowler, Rashid Khan Bowler, Daniel Vettori Bowler, AB de Villiers Batter, Chris Gayle All-rounder (so his bowling is now Spin, right arm, from the infobox's off break), Viv Richards Batter. That changed Ashwin's role and Gayle's role, bowling and arm: 4 more corrected values.

## Previously unresolved items (now decided)

**Role classification (8 players).** In each case Wikipedia's wording differs from the pack's four classes. The pack keeps its current value until you decide.

| Player | Pack | Wikipedia infobox |
|---|---|---|
| Yuvraj Singh | All-rounder | Middle-order batsman |
| Ravichandran Ashwin | Bowler | Bowling all-rounder |
| Wasim Akram | Bowler | Bowling all-rounder |
| Rashid Khan | Bowler | Bowling all-rounder |
| Daniel Vettori | Bowler | All-rounder |
| AB de Villiers | Batter | Wicket-keeper-batter |
| Chris Gayle | Batter | All-rounder |
| Viv Richards | Batter | Batting all-rounder |

For AB de Villiers, Gayle and Richards, the "Bowling" record also depends on the decision, because of the batter rule.

**Other limitations:**

- **2015 World Cup.** The 2015 squad page did not render. Smith, Cummins and Starc are recorded as "Yes" because of the 2023 squad, which was confirmed by the ICC. Their 2015 titles are noted but not counted as checked.
- **Source tier.** The brief puts ICC and ESPNcricinfo first. ESPNcricinfo player profiles, Wikidata and Wikimedia Commons could not be opened from this environment, so most facts are sourced from English Wikipedia (secondary).
  - Where an official page was reachable (BCCI 2011, ICC 2019 and 2023), it agreed with Wikipedia.
  - Each audit row cites the page the fact came from.
  - Wikipedia article URLs are recorded in their canonical form.

## Images

No image is marked verified. All five candidate file pages from the brief failed to open, along with the Commons and Wikipedia APIs:
- `Glenn_McGrath_in_final_test_series.jpg`
- `Glenn_mcgrath_(6279982367).jpg`
- `STEVE_WAUGH_(6631163989).jpg`
- `ALASTAIR_COOK_(2876041389).jpg`
- `Andrew_Flintoff_(1).jpg`

The WebFetch tool returned a fetch error, and the network proxy refused shell connections to the Commons and Wikipedia hosts. The earlier "CC BY 2.0, Jim Dowling" metadata was not copied anywhere.

To finish the images, use a machine that can reach Wikipedia and Commons:

```bash
MINDHUNT_CONTACT=you@example.com python3 tools/fetch_portraits.py fetch   # licence checks, download, decode
# look at assets/cricket-men-intl/*.webp, then for each picture that really shows the player:
python3 tools/fetch_portraits.py confirm c_xxxxxx ...
python3 tools/gen_cricket_pack.py > server/packs/cricket-men-intl.v3.json   # bump "version" first
python3 tools/verify_cricket.py
```

The script:
- allows only CC0, public domain, CC BY and CC BY-SA;
- refuses NC, ND, fair use, non-free and unknown licences;
- requires an author and a licence URL;
- rejects lead images that are not on Commons, and disambiguation pages;
- rejects a file already used for another player;
- retries with backoff;
- never marks anything "Verified" without a person's `confirm`.

## Application changes

- **Pack data.** The cricket pack is now v2 with the corrections and the ODI-based "ODI debut decade" attribute (range 1970–2020).
  - The question wording now says "ODI debut".
  - Version 3 records the owner's role decisions and is no longer marked draft.
- **Image serving.** Images are served locally from `ASSETS_DIR` at `/assets/<pack>/<id>.webp`, so a match never fetches from Wikimedia.
  - Only image files are served. There are no listings and no path traversal.
  - Responses carry cache headers.
- **Pack validation.** Packs reject an image with no licence, no credit, a source that isn't https, or an unsafe asset path.
- **Missing files.** At startup, image references whose file is missing are dropped (with a warning), and initials are shown instead.
- **Reveal only.** Images still reach the client only in results: the multiplayer reveal and the finished daily.
- **Portrait component.** A new shared component shows the photo with a credit line (credit · licence, linked · source). If the file fails to load it falls back to initials. It fits at phone width.

## Tests run

| Suite | Outcome |
|---|---|
| `go vet ./...` | pass |
| `go test -race ./...` (game, room, bot, daily, store, transport) | pass |
| Postgres store test against Postgres 16 | pass |
| New Go tests: 72 players, no duplicate IDs, IDs preserved, roster order, ODI decades 1970–2020, Bradman N/A answering Unknown, Lloyd 1970, image validation, missing image falls back to initials, `/assets` serves only images | pass |
| `tsc --noEmit` and `next build` | pass |
| Portrait pipeline unit tests (offline): licence allow/refuse, Commons-missing lead, NC skipped, duplicate rejected, decode to WebP, no "Verified" without confirm, network failure never counts as success | 4/4 pass |
| `verify_cricket.py`: every pack value equals the evidence, every fact has a source URL, image rules hold | pass |
| Importer round trip of the verified workbook | identical pack, no version change |
| Browser: two players, full round (`two-browsers.mjs`) | 12/12 |
| Browser: daily (`daily.mjs`) | 5/5 |
| Browser: packs (`packs.mjs`) | 6/6 |
| Browser: portrait reveal with test fixtures (`portrait.mjs`), three cases: valid image, file missing, file corrupt | 21/21 |

The portrait reveal test checked that:
- nothing image-related is sent or fetched before the reveal;
- no Wikimedia host is contacted;
- the photo and credit appear after the reveal;
- initials appear when the image is missing or broken;
- there is no horizontal scroll at 390 px.

These tests used a synthetic grey "TEST" card, not real portraits.

Two existing browser tests needed updating:
- `two-browsers.mjs`: the attribute label is now "ODI debut decade".
- `daily.mjs`: the stats line now accepts any counts, because the database keeps earlier plays.

## Deliverables

| File | What |
|---|---|
| `data/cricket/mindhunt-cricket-players-verified.xlsx` | Players (original columns plus Data/Image verification, first ODI, captaincy formats, titles, review note), Audit, Image attribution, Methodology, Lists |
| `data/cricket/original/mindhunt-cricket-players.xlsx` | The original workbook, unmodified |
| `data/cricket/cricket-audit.csv`, `cricket-audit.json` | Source audit: 720 records |
| `data/cricket/cricket-players.json` | Flat app-ready dataset |
| `server/packs/cricket-men-intl.v3.json` | The pack the game loads |
| `tools/portraits/cricket-manifest.json`, `data/cricket/cricket-image-attribution.csv` | Image attribution manifest (72 entries) |
| `tools/cricket_evidence.py` | Evidence per player |
| `tools/verify_cricket.py` | Rebuilds everything above and fails on any disagreement |
| `tools/fetch_portraits.py` | Portrait pipeline |
| `tools/portraits/cricket-candidates.json` | Portrait leads |
| `data/cricket/VERIFICATION_REPORT.md` | This report |

## Public-launch blockers

1. **Portraits.** No player has a licensed portrait. Run the portrait pipeline where Commons is reachable and review identities.
2. ~~Role calls~~ Done 18 Oct 2026 (pack v3).
3. **Official sources.** Optionally re-check against ESPNcricinfo or ICC profiles. Wikipedia is the main source for most facts here.
4. **Docker.** Build and run the Docker image once. It is untested.
5. (Minor) A daily game recorded on cricket v1 that asked about debuts before 1970 replays only up to that move under v2. The server logs this and does not crash. It matters only if v1 was ever public.
