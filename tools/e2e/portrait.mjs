// Portrait reveal on the daily challenge, against a server started with
// tools/e2e/portrait_fixture.py data. Usage: node portrait.mjs baseURL valid|missing|corrupt [shots]
import chromium from "@sparticuz/chromium";
import puppeteer from "puppeteer-core";
import fs from "node:fs";

const BASE = process.argv[2];
const MODE = process.argv[3];
const OUT = process.argv[4] ?? "./shots";
fs.mkdirSync(OUT, { recursive: true });
const results = [];
const check = (name, ok, detail = "") => {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"}  [${MODE}] ${name}${detail ? "  — " + detail : ""}`);
};
const browser = await puppeteer.launch({ executablePath: await chromium.executablePath(), args: chromium.args, headless: true });
const page = (await browser.pages())[0] ?? (await browser.newPage());
await page.setViewport({ width: 390, height: 844, isMobile: true, hasTouch: true });
const requests = [];
const bodies = [];
let revealed = false;
page.on("request", (r) => requests.push({ url: r.url(), revealed }));
page.on("response", async (r) => {
  if (r.url().includes("/api/daily")) bodies.push({ revealed, text: await r.text().catch(() => "") });
});
const click = async (text) => {
  const h = await page.waitForFunction(
    (t) => [...document.querySelectorAll("button")].find((b) => b.innerText.includes(t) && b.offsetParent !== null),
    { timeout: 8000 },
    text,
  );
  await h.click();
};

try {
  await page.goto(BASE + "/daily/");
  await page.waitForSelector("#display-name");
  await page.type("#display-name", "Portrait tester");
  await click("Play today's challenge");
  await page.waitForFunction(() => document.body.innerText.includes("Daily #"));
  check("no portrait on the board while hunting", (await page.$("[data-portrait]")) === null);
  await click("Give up");
  revealed = true; // requests from here on may legitimately carry the image
  await click("Give up today");
  await page.waitForFunction(() => document.body.innerText.includes("Today's secret"));
  await new Promise((r) => setTimeout(r, 800));

  const early = requests.filter((r) => !r.revealed && r.url.includes("/assets/"));
  check("no image fetched before the reveal", early.length === 0, early.map((r) => r.url).join(", "));
  const leaked = bodies.filter((b) => !b.revealed && b.text.includes('"image"'));
  check("no image reference in API responses before the reveal", leaked.length === 0, `${leaked.length} leaking`);
  const remote = requests.filter((r) => /upload\.wikimedia|commons\.wikimedia/.test(r.url));
  check("no remote image hosts contacted", remote.length === 0);

  const img = await page.$("[data-portrait]");
  const state = await page.evaluate(() => {
    const i = document.querySelector("[data-portrait]");
    const credit = document.querySelector("[data-credit]");
    const article = [...document.querySelectorAll("article")].find((a) => a.innerText.includes("Today's secret"));
    return {
      loaded: i ? i.complete && i.naturalWidth > 0 : false,
      credit: credit?.innerText ?? "",
      initials: article?.querySelector("[aria-hidden].rounded-full")?.textContent ?? "",
      overflow: document.documentElement.scrollWidth > window.innerWidth,
    };
  });
  if (MODE === "valid") {
    check("portrait shown after the reveal", !!img && state.loaded);
    check("credit and licence shown with it", state.credit.includes("MindHunt test fixture") && state.credit.includes("CC0") && state.credit.includes("Source"), state.credit);
  } else {
    check("initials shown instead of a picture", !img && /^[A-Z]{1,2}$/.test(state.initials), `initials=${state.initials}`);
    check("no credit line without a picture", state.credit === "");
  }
  check("no horizontal scroll on a phone", !state.overflow);
  await page.screenshot({ path: `${OUT}/p-${MODE}.png`, fullPage: true });
} catch (e) {
  check("flow completed", false, e.message);
  await page.screenshot({ path: `${OUT}/p-${MODE}-failure.png`, fullPage: true }).catch(() => {});
} finally {
  await browser.close();
}
const failed = results.filter((r) => !r.ok);
console.log(`${results.length - failed.length}/${results.length} checks passed`);
process.exit(failed.length ? 1 : 0);
