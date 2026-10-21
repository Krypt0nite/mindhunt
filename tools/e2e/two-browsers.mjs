// Two-browser end-to-end run against a live server.
// Usage: node two-browsers.mjs [baseURL] [screenshotDir]
// Needs: npm i puppeteer-core @sparticuz/chromium (or set CHROME_PATH).
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
  results.push({ name, ok, detail });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
};

async function player(label, viewport) {
  // One browser per player: separate storage, cookies and sockets.
  const browser = await puppeteer.launch({ executablePath: exe, args: chromium.args, headless: true });
  browsers.push(browser);
  const page = (await browser.pages())[0] ?? (await browser.newPage());
  await page.setViewport(viewport);
  page.on("pageerror", (e) => console.log(`[${label}] page error:`, e.message));
  const frames = [];
  const cdp = await page.createCDPSession();
  await cdp.send("Network.enable");
  cdp.on("Network.webSocketFrameReceived", (e) => frames.push(e.response.payloadData));
  return { label, page, frames };
}

const shot = (p, name) => p.page.screenshot({ path: `${OUT}/${name}.png`, fullPage: true });
// Find the deepest visible element whose text includes `text`.
const findText = (p, text, timeout = 8000) =>
  p.page.waitForFunction(
    (t) => {
      const all = [...document.querySelectorAll("button, a, h1, h2, h3, p, span, li, label, summary")];
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
const waitText = (p, text, timeout = 8000) => findText(p, text, timeout);

const A = await player("host", { width: 1280, height: 900 });
const B = await player("guest", { width: 390, height: 844, isMobile: true, hasTouch: true });

try {
  // Host signs in and creates a room.
  await A.page.goto(BASE + "/");
  await A.page.waitForSelector("#display-name");
  await shot(A, "01-home-new");
  await A.page.type("#display-name", "Asha");
  await clickText(A, "Continue");
  await clickText(A, "Play with a friend");
  await A.page.waitForFunction(() => location.pathname.startsWith("/room"));
  const code = new URL(A.page.url()).searchParams.get("code");
  check("room created", /^[A-Z2-9]{6}$/.test(code ?? ""), code);
  await waitText(A, "Open seat");

  // Guest opens the invite link on a phone.
  await B.page.goto(`${BASE}/room/?code=${code}`);
  await waitText(B, "You're invited");
  await shot(B, "02-invite-mobile");
  await B.page.type("#display-name", "Bilal");
  await clickText(B, "Join room");
  await waitText(B, "I'm ready");
  await waitText(A, "Bilal");
  check("both players see each other", true);

  // Host picks Easy; guest readies; host starts.
  await clickText(A, "Easy");
  await clickText(B, "I'm ready");
  await A.page.waitForFunction(() => !document.querySelector("button.btn-ink[disabled]")?.textContent?.includes("Start"), { timeout: 5000 });
  await shot(A, "03-lobby-host");
  await shot(B, "04-lobby-guest-mobile");
  await clickText(A, "Start the round");

  // Picking.
  await waitText(A, "Pick your secret");
  await waitText(B, "Pick your secret");
  const names = await A.page.$$eval('ul[aria-label="Candidates"] button span.font-semibold', (els) => els.map((e) => e.textContent.trim()));
  check("easy pool has 24 candidates", names.length === 24, String(names.length));
  const secretA = names[0];
  const secretB = names[5];
  await clickText(A, secretA);
  await clickText(A, `Lock in ${secretA}`);
  await waitText(B, "has picked");
  await shot(B, "05-picker-mobile");
  await clickText(B, secretB);
  await clickText(B, `Lock in ${secretB}`);

  // Hunting.
  await waitText(A, "Your moves");
  await waitText(B, "Your moves");
  await clickText(A, "Country");
  const countryChips = await A.page.$$eval('[aria-label="Country value"] button', (els) => els.map((e) => e.textContent.trim()));
  await clickText(A, countryChips[0]);
  await clickText(A, "Ask (1 move)");
  await A.page.waitForSelector('[role="status"].panel');
  await clickText(A, "Bowling");
  await clickText(A, "Spin");
  await new Promise((r) => setTimeout(r, 300));
  await shot(A, "06-hunt-host-desktop");

  // Guest asks one question on mobile, then opens the board.
  await clickText(B, "ODI debut decade");
  await clickText(B, "Ask (1 move)");
  await B.page.waitForSelector('[role="status"].panel');
  await shot(B, "07-hunt-guest-mobile");
  await clickText(B, "Board,");
  await shot(B, "08-board-mobile");

  // A guesses B's secret from the board.
  await clickText(A, secretB);
  await shot(A, "09-detail-host");
  await clickText(A, `Guess ${secretB}`);
  await waitText(A, "You found it in");
  await waitText(B, "Found your secret");
  check("guest sees host solved without seeing count", true);

  // Leak check: before results, no frame to B may contain A's secret
  // except the pool list and B's own remaining-candidate list.
  const poolA = JSON.parse(A.frames.find((f) => f.includes('"type":"pool"'))).data.candidates;
  const idA = poolA.find((c) => c.name === secretA).id;
  const leaks = B.frames.filter((f) => {
    const m = JSON.parse(f);
    if (m.type === "pool") return false;
    if (m.data?.hunt) delete m.data.hunt.remaining_ids;
    return JSON.stringify(m).includes(idA) || JSON.stringify(m).includes('"image"');
  });
  check("guest never received host's secret before results", leaks.length === 0, `${leaks.length} leaking frames`);

  // B solves too (guessing straight away).
  await clickText(B, secretA);
  await clickText(B, `Guess ${secretA}`);
  await waitText(A, "Rematch");
  await waitText(B, "Rematch");
  const hA = await A.page.$eval("h1", (e) => e.textContent);
  const hB = await B.page.$eval("h1", (e) => e.textContent);
  check("results agree", (hA === "You won" && hB === "You lost") || (hA === "You lost" && hB === "You won") || (hA === "It's a draw" && hB === "It's a draw"), `${hA} / ${hB}`);
  await shot(A, "10-results-host");
  await shot(B, "11-results-guest-mobile");

  // Rematch.
  await clickText(A, "Rematch");
  await waitText(B, "wants a rematch");
  await clickText(B, "wants a rematch");
  await waitText(A, "Pick your secret");
  await waitText(B, "Pick your secret");
  check("rematch starts a new round in the same room", new URL(B.page.url()).searchParams.get("code") === code);

  // Reload mid-round keeps the seat.
  await B.page.reload();
  await waitText(B, "Pick your secret");
  check("guest reload rejoins the same seat", true);

  // Practice round against the bot, from the home page.
  await A.page.goto(BASE + "/");
  await clickText(A, "Practice against the bot");
  await waitText(A, "Practice round");
  await waitText(A, "Computer opponent");
  await clickText(A, "Start the round");
  await waitText(A, "Pick your secret");
  await clickText(A, "Choose at random");
  await clickText(A, "Lock in");
  await waitText(A, "Your moves");
  await waitText(A, "Found your secret", 45000);
  check("practice bot finds the secret", true);
  await shot(A, "12-practice-bot-solved");

  // Typed questions: host allows them; a known question is answered by the
  // data, an unknown one goes to the guest, who answers by hand.
  await A.page.goto(BASE + "/");
  await clickText(A, "Play with a friend");
  await A.page.waitForFunction(() => location.pathname.startsWith("/room"));
  const code2 = new URL(A.page.url()).searchParams.get("code");
  await B.page.goto(`${BASE}/room/?code=${code2}`);
  await waitText(B, "I'm ready");
  await clickText(A, "Allowed");
  await waitText(B, "go to the other player");
  await clickText(B, "I'm ready");
  await waitText(A, "Ready");
  await clickText(A, "Start the round");
  await waitText(A, "Pick your secret");
  await clickText(A, "Choose at random");
  await clickText(A, "Lock in");
  await waitText(B, "Pick your secret");
  await clickText(B, "Choose at random");
  await clickText(B, "Lock in");
  await waitText(A, "Your moves");
  await clickText(A, "Type or say");
  await A.page.type("#text-question", "did they ever represent India");
  await waitText(A, "Answered from the data");
  check("typed known question is recognised", true);
  await clickText(A, "Ask (1 move)");
  await waitText(A, "Have they represented India?");
  await A.page.type("#text-question", "Is he married?");
  await waitText(A, "Send to Bilal");
  await shot(A, "13-typed-question");
  await clickText(A, "Send to Bilal");
  await waitText(A, "Waiting for Bilal to answer");
  await waitText(B, "Asha asks about your secret");
  await shot(B, "14-incoming-question-mobile");
  await clickText(B, "Yes");
  await waitText(A, "Answered by Bilal");
  check("manual answer reaches the asker", true);
  const movesA = await A.page.$eval("[aria-live=polite].tabular", (e) => e.textContent.trim());
  check("both typed questions cost one move each", movesA === "2", movesA);
  await shot(A, "15-manual-answer");
} catch (e) {
  check("flow completed", false, e.message);
  await shot(A, "zz-host-failure").catch(() => {});
  await shot(B, "zz-guest-failure").catch(() => {});
} finally {
  await Promise.all(browsers.map((b) => b.close()));
}

const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
process.exit(failed.length ? 1 : 0);
