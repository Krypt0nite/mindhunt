// Daily challenge in two browsers: one solves, one gives up.
// Usage: node daily.mjs [baseURL] [screenshotDir]
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
  const daily = []; // bodies of /api/daily responses
  page.on("response", async (r) => {
    if (r.url().includes("/api/daily") && !r.url().includes("/match")) {
      try {
        daily.push(await r.text());
      } catch {}
    }
  });
  return { label, page, daily };
}

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
const shot = (p, name) => p.page.screenshot({ path: `${OUT}/${name}.png`, fullPage: true });
const signIn = async (p, name) => {
  await p.page.goto(BASE + "/daily/");
  await p.page.waitForSelector("#display-name");
  await p.page.type("#display-name", name);
  await clickText(p, "Play today's challenge");
  await findText(p, "Daily #");
};

const A = await player("solver", { width: 1280, height: 900 });
const B = await player("quitter", { width: 390, height: 844, isMobile: true, hasTouch: true });

try {
  await signIn(A, "Asha");
  await shot(A, "d1-daily-start");
  // Ask one question, then reload: progress must survive.
  await clickText(A, "Country");
  const chips = await A.page.$$eval('[aria-label="Country value"] button', (els) => els.map((e) => e.textContent.trim()));
  await clickText(A, chips[0]);
  await clickText(A, "Ask (1 move)");
  await A.page.waitForFunction(() => document.querySelector("[aria-live=polite].tabular")?.textContent.trim() === "1");
  await A.page.reload();
  await findText(A, "Daily #");
  const moves = await A.page.$eval("[aria-live=polite].tabular", (e) => e.textContent.trim());
  check("progress survives a reload", moves === "1", moves);

  // Solve by guessing names from the board until correct.
  const names = await A.page.$$eval('ul[aria-label="Candidates"] button.tile .tile-name', (els) => els.map((e) => e.textContent.trim()));
  let solved = false;
  for (const n of names) {
    await clickText(A, n);
    await clickText(A, `Guess ${n}`);
    await new Promise((r) => setTimeout(r, 250)); // stay under the rate limit
    solved = await A.page.evaluate(() => document.body.innerText.includes("Solved in"));
    if (solved) break;
  }
  check("solver sees the reveal", solved);
  const before = A.daily.slice(0, -1);
  check("no answer or image sent before finishing", before.every((b) => !b.includes('"result"') && !b.includes('"image"')), `${before.length} responses`);
  await findText(A, "finished today");
  await shot(A, "d2-daily-solved");

  // Quitter on a phone gives up and sees the answer and the chart.
  await signIn(B, "Bilal");
  await clickText(B, "Give up");
  await clickText(B, "Give up today");
  await findText(B, "Not today");
  await findText(B, "1 finished today", 3000).catch(() => {});
  const text = await B.page.evaluate(() => document.body.innerText);
  check("give-up reveals the answer and stats", text.includes("Today's secret") && /\d+ finished today, [1-9]\d* gave up/.test(text), "");
  await shot(B, "d3-daily-gave-up-mobile");

  // Home shows the day's result.
  await A.page.goto(BASE + "/");
  await findText(A, "See today's result");
  check("home page shows today's result", true);
} catch (e) {
  check("flow completed", false, e.message);
  await shot(A, "dz-solver-failure").catch(() => {});
  await shot(B, "dz-quitter-failure").catch(() => {});
} finally {
  await Promise.all(browsers.map((b) => b.close()));
}
const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
process.exit(failed.length ? 1 : 0);
