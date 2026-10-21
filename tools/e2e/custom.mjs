// Player-made packs end to end: build a pack in the browser, share it, play
// it with a friend on a phone, edit it, report it.
// Usage: node custom.mjs [baseURL] [screenshotDir]
import chromium from "@sparticuz/chromium";
import puppeteer from "puppeteer-core";
import fs from "node:fs";

const BASE = process.argv[2] ?? "http://localhost:8080";
const OUT = process.argv[3] ?? "./shots";
fs.mkdirSync(OUT, { recursive: true });
const exe = process.env.CHROME_PATH ?? (await chromium.executablePath());
const browsers = [];
const results = [];
const check = (name, ok, detail = "") => {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
};

async function player(label, viewport) {
  const browser = await puppeteer.launch({ executablePath: exe, args: chromium.args, headless: true });
  browsers.push(browser);
  const page = (await browser.pages())[0] ?? (await browser.newPage());
  await page.setViewport(viewport);
  page.on("pageerror", (e) => console.log(`[${label}] page error:`, e.message));
  page.on("dialog", (d) => d.accept());
  return { label, page };
}
const shot = (p, name) => p.page.screenshot({ path: `${OUT}/${name}.png`, fullPage: true });
const findText = (p, text, timeout = 8000) =>
  p.page.waitForFunction(
    (t) => {
      const all = [...document.querySelectorAll("button, a, h1, h2, h3, p, span, li, label, summary, dt, dd, th")];
      const hits = all.filter((el) => el.innerText && el.innerText.includes(t) && el.offsetParent !== null);
      const deepest = hits.filter((el) => !hits.some((o) => o !== el && el.contains(o)));
      return deepest.find((el) => el.closest("button, a, label, summary")) ?? deepest[0] ?? null;
    },
    { timeout },
    text,
  );
const clickText = async (p, text) => {
  const h = await findText(p, text);
  const el = await h.evaluateHandle((el) => el.closest("button, a, label, summary") ?? el);
  await el.click();
};
const signIn = async (p, name) => {
  await p.page.goto(BASE + "/");
  await p.page.waitForSelector("#display-name");
  await p.page.type("#display-name", name);
  await clickText(p, "Continue");
  await findText(p, "Play with a friend");
};
const typeInto = async (p, sel, text) => {
  await p.page.focus(sel);
  await p.page.$eval(sel, (el) => el.select());
  await p.page.keyboard.press("Backspace");
  if (text) await p.page.type(sel, text);
};
const noSideScroll = (p) => p.page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);

const A = await player("maker", { width: 1280, height: 900 });
const B = await player("friend", { width: 390, height: 844, isMobile: true, hasTouch: true });

try {
  await signIn(A, "Asha");
  await clickText(A, "Start a pack");
  await A.page.waitForFunction(() => location.pathname.startsWith("/create"));
  await findText(A, "Make a pack");
  await clickText(A, "Office colleagues");
  await findText(A, "Ready to save", 8000);
  check("example loads and passes the live check", true);
  await shot(A, "c1-builder-example");

  // Break a cell: the check names the row and the field turns red.
  await typeInto(A, "#f-items-1-name", "asha");
  await findText(A, "is also row 1", 8000);
  const red = await A.page.$eval("#f-items-1-name", (e) => e.className.includes("border-no"));
  check("duplicate name flagged on the row", red);
  await shot(A, "c2-builder-error");
  const saveDisabled = await A.page.$eval("aside button.btn-ink", (b) => b.disabled);
  check("can't save with errors", saveDisabled);
  await typeInto(A, "#f-items-1-name", "Bilal");

  // Paste two rows from a spreadsheet.
  await clickText(A, "Paste from a spreadsheet");
  await A.page.type("#paste", "Name\tTeam\tWears glasses\tDrinks coffee\tYears here\nKavya\tOps\tyes\tno\t2\nLeo\tDesign\tno\tyes\t9");
  await findText(A, "2 rows found, header skipped");
  await clickText(A, "Add these rows");
  const rows = await A.page.$$eval("table tbody tr", (r) => r.length);
  check("pasted rows added", rows === 12, `${rows} rows`);
  await findText(A, "Ready to save", 8000);

  await clickText(A, "Save and get the link");
  await A.page.waitForFunction(() => location.pathname.startsWith("/pack"), { timeout: 8000 });
  await findText(A, "Saved. Share the link");
  const packURL = A.page.url().replace("&saved=1", "");
  const packId = new URL(packURL).searchParams.get("id");
  check("saved with a share link", /^u-[a-z2-9]{10}$/.test(packId ?? ""), packId);
  await shot(A, "c3-pack-page-owner");

  // Friend opens the link on a phone.
  await B.page.goto(packURL);
  await findText(B, "What should we call you?");
  await B.page.type("#display-name", "Bilal");
  await clickText(B, "Continue");
  await findText(B, "Play this pack");
  const text = await B.page.evaluate(() => document.body.innerText);
  check("friend sees the pack, author and questions", text.includes("Our office") && text.includes("by Asha") && text.includes("Wears glasses") && !text.includes("Edit"));
  check("pack page fits a phone", await noSideScroll(B));
  await shot(B, "c4-pack-page-friend-mobile");

  // Maker starts a room from the pack page; friend joins by invite.
  await clickText(A, "Play with a friend");
  await A.page.waitForFunction(() => location.pathname.startsWith("/room"));
  const code = new URL(A.page.url()).searchParams.get("code");
  await findText(A, "player-made by Asha");
  check("room starts on the custom pack", true, code);
  await B.page.goto(`${BASE}/room/?code=${code}`);
  await clickText(B, "I'm ready");
  await findText(A, "Ready");
  await shot(A, "c5-lobby-custom");
  await clickText(A, "Start the round");
  await findText(A, "Pick your secret");
  await findText(B, "Pick your secret");
  const pool = await A.page.$$eval('ul[aria-label="Candidates"] button span.font-semibold', (els) => els.map((e) => e.textContent.trim()));
  check("pool is the pack's 12 colleagues", pool.length === 12 && pool.includes("Kavya"), pool.join(", "));
  await clickText(A, "Hari");
  await clickText(A, "Lock in Hari");
  await clickText(B, "Leo");
  await clickText(B, "Lock in Leo");
  await findText(B, "Your moves");

  // Friend asks with the maker's wording.
  await clickText(B, "Team");
  await clickText(B, "Sales");
  await clickText(B, "Ask (1 move)");
  await findText(B, "Do they work in Sales?");
  check("question uses the maker's wording", true);
  await clickText(B, "Wears glasses");
  await clickText(B, "Ask (1 move)");
  await findText(B, "Do they wear glasses?");
  await shot(B, "c6-hunt-custom-mobile");
  await clickText(B, "Board,");
  await clickText(B, "Hari");
  await clickText(B, "Guess Hari");
  await clickText(A, "Leo");
  await clickText(A, "Guess Leo");
  await findText(A, "Rematch", 8000);
  await findText(B, "Rematch", 8000);
  check("round on a player-made pack finishes", true);
  await shot(B, "c7-results-custom-mobile");

  // Maker edits: a new version; the lobby's picker lists "Your packs".
  await A.page.goto(`${BASE}/create/?id=${packId}`);
  await findText(A, "Edit your pack");
  await typeInto(A, "#f-title", "Our office, 3rd floor");
  await findText(A, "Ready to save", 8000);
  await clickText(A, "Save changes");
  await findText(A, "version 2");
  check("edit saves version 2 at the same link", new URL(A.page.url()).searchParams.get("id") === packId);
  await A.page.goto(BASE + "/");
  await findText(A, "Our office, 3rd floor");
  await clickText(A, "Practice against the bot");
  await findText(A, "Your packs");
  await clickText(A, "Our office, 3rd floor");
  await findText(A, "player-made by Asha");
  check("host can switch the room to their pack", true);

  // Friend reports it.
  await B.page.goto(packURL);
  await clickText(B, "Report this pack");
  await B.page.type("#report-reason", "testing the report button");
  await clickText(B, "Send report");
  await findText(B, "We'll review this pack");
  check("report sent", true);

  // Builder fits a phone: the table scrolls inside its box, not the page.
  await B.page.goto(BASE + "/create/");
  await clickText(B, "Fruit");
  await findText(B, "Ready to save", 8000);
  check("builder fits a phone", await noSideScroll(B));
  await shot(B, "c8-builder-mobile");
} catch (e) {
  check("flow completed", false, e.message);
  await shot(A, "cz-maker-failure").catch(() => {});
  await shot(B, "cz-friend-failure").catch(() => {});
} finally {
  await Promise.all(browsers.map((b) => b.close()));
}
const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
process.exit(failed.length ? 1 : 0);
