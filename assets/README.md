# Licensed images

Served by the game at `/assets/<pack>/<candidate id>.webp` from `ASSETS_DIR`
(default `../assets` relative to the server, `/assets` in Docker). Images are
shown only on result screens, with credit and licence.

Only files written by `tools/fetch_portraits.py` and confirmed by a person
(`fetch_portraits.py confirm <id>`) are referenced by a pack. A referenced file
that is missing at startup is dropped from the pack with a warning, and the
player's initials are shown instead.

No cricket portraits are included yet: Wikimedia Commons could not be reached
from the environment that built this release. See
`data/cricket/VERIFICATION_REPORT.md`.
