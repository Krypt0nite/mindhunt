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

const A = await player("host", { width: 1280, height: 900 });

try {
  await A.page.goto(BASE + "/");
  await A.page.waitForSelector("#display-name");
  await A.page.type("#display-name", "Asha");
  await clickText(A, "Continue");
  await findText(A, "Historical Figures");
  check("home lists packs by category", (await A.page.evaluate(() => document.body.innerText)).includes("Historical Figures"));
  await clickText(A, "Practice against the bot");
  await A.page.waitForSelector('input[aria-label="Search topic packs"]');
  await A.page.type('input[aria-label="Search topic packs"]', "elements");
  await clickText(A, "Fifty-two elements");
  await findText(A, "Chemical Elements in Science");
  check("search and pick a pack", true);
  await shot(A, "p1-lobby-pack-picker");
  await clickText(A, "Start the round");
  await findText(A, "Pick your secret");
  await clickText(A, "Choose at random");
  await clickText(A, "Lock in");
  await findText(A, "Your moves");
  await clickText(A, "State at room temperature");
  await clickText(A, "Gas");
  await clickText(A, "Ask (1 move)");
  await findText(A, "Is it a gas at room temperature?");
  check("pack-specific question text", true);
  await clickText(A, "Type or say");
  await A.page.type("#text-question", "is it a noble gas");
  await findText(A, "Answered from the data");
  check("typed question matched in a new pack", true);
  await shot(A, "p2-elements-hunt");
  await findText(A, "Found your secret", 45000);
  check("bot solves a non-cricket pack", true);
  // Surprise me changes the pack.
  await A.page.goto(BASE + "/");
  await clickText(A, "Practice against the bot");
  await A.page.waitForSelector('input[aria-label="Search topic packs"]');
  const before = await A.page.$eval("fieldset p", (e) => e.textContent);
  await clickText(A, "Surprise me");
  await A.page.waitForFunction((b) => document.querySelector("fieldset p")?.textContent !== b, { timeout: 5000 }, before);
  check("surprise me picks another pack", true);
} catch (e) {
  check("flow completed", false, e.message);
  await shot(A, "pz-failure").catch(() => {});
} finally {
  await Promise.all(browsers.map((b) => b.close()));
}
const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
process.exit(failed.length ? 1 : 0);
