// Builds everything the Worker deploys:
//   public/              exported web client + licensed images (static assets)
//   build/mindhunt.wasm  Go server for the Durable Object
//   build/wasm_exec.js   Go's JavaScript glue (must match the Go version)
//   build/assets.json    image files shipped, so packs keep only those
//
// Usage: node build.mjs [--skip-web]
import { execSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "..");
const web = join(root, "web");
const pub = join(here, "public");
const build = join(here, "build");
const skipWeb = process.argv.includes("--skip-web");

const run = (cmd, cwd, env = {}) => execSync(cmd, { cwd, stdio: "inherit", env: { ...process.env, ...env } });

// 1. Web client (same-origin API, so NEXT_PUBLIC_API_BASE stays empty).
if (!skipWeb || !existsSync(join(web, "out"))) {
  if (!existsSync(join(web, "node_modules"))) run("npm ci", web);
  run("npx next build", web, { NEXT_TELEMETRY_DISABLED: "1", NEXT_PUBLIC_API_BASE: "" });
}
rmSync(pub, { recursive: true, force: true });
cpSync(join(web, "out"), pub, { recursive: true });

// 2. Licensed images: only image files, at /assets/<pack>/<id>.<ext>.
const assetsDir = join(root, "assets");
const shipped = [];
const walk = (dir) => {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) walk(full);
    else if (/\.(webp|jpe?g|png)$/i.test(name)) shipped.push(relative(assetsDir, full).split("\\").join("/"));
  }
};
if (existsSync(assetsDir)) walk(assetsDir);
for (const rel of shipped) {
  mkdirSync(dirname(join(pub, "assets", rel)), { recursive: true });
  cpSync(join(assetsDir, rel), join(pub, "assets", rel));
}

writeFileSync(
  join(pub, "_headers"),
  `/*
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
  X-Frame-Options: DENY

/_next/static/*
  Cache-Control: public, max-age=31536000, immutable

/assets/*
  Cache-Control: public, max-age=604800
`,
);

// 3. Go server as WebAssembly.
mkdirSync(build, { recursive: true });
run(`go build -trimpath -ldflags="-s -w" -o "${join(build, "mindhunt.wasm")}" ./cmd/mindhunt-wasm`, join(root, "server"), {
  GOOS: "js",
  GOARCH: "wasm",
  CGO_ENABLED: "0",
});
const goroot = execSync("go env GOROOT", { encoding: "utf8" }).trim();
const glue = [join(goroot, "lib", "wasm", "wasm_exec.js"), join(goroot, "misc", "wasm", "wasm_exec.js")].find(existsSync);
if (!glue) throw new Error("wasm_exec.js not found in GOROOT");
cpSync(glue, join(build, "wasm_exec.js"));
writeFileSync(join(build, "assets.json"), JSON.stringify(shipped));

const kb = (f) => Math.round(statSync(f).size / 1024);
console.log(`built: public/ (${shipped.length} images), build/mindhunt.wasm (${kb(join(build, "mindhunt.wasm"))} KB)`);
