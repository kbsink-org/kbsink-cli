#!/usr/bin/env node
/**
 * Minimal wasm integration test (after ./scripts/build-wasm.sh):
 *   node scripts/run-wasm.mjs <article-url> [--plugin wechat|xhs|douyin] [-o output]
 */
import { createRequire } from "node:module";
import { readFileSync, writeFileSync, mkdirSync, existsSync, mkdtempSync, rmSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { execSync, spawnSync } from "node:child_process";
import { tmpdir } from "node:os";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

/** Sync curl bridge: wasm export is synchronous, so Promise-based fetch deadlocks. */
function installHTTPBridge() {
  globalThis.kbsinkHTTPRoundTrip = (payloadJson) => {
    const req = JSON.parse(payloadJson);
    const timeoutSec = Math.max(
      1,
      Math.ceil((req.timeoutMs > 0 ? req.timeoutMs : 600_000) / 1000)
    );
    const dir = mkdtempSync(join(tmpdir(), "kbsink-http-"));
    const bodyPath = join(dir, "body");
    try {
      const args = [
        "-sS",
        "-L",
        "--max-time",
        String(timeoutSec),
        "-o",
        bodyPath,
        "-w",
        "%{http_code}\n%{content_type}",
        "-X",
        req.method || "GET",
        ...Object.entries(req.headers ?? {}).flatMap(([k, v]) => ["-H", `${k}: ${v}`]),
        req.url,
      ];
      const out = spawnSync("curl", args, { encoding: "utf8" });
      if (out.error) throw out.error;
      if (out.status !== 0) {
        throw new Error(out.stderr?.trim() || `curl exit ${out.status}`);
      }
      const body = readFileSync(bodyPath);
      const lines = String(out.stdout ?? "").trim().split("\n");
      const status = Number.parseInt(lines[0] ?? "0", 10) || 0;
      const contentType = (lines[1] ?? "").trim();
      const headers = contentType ? { "Content-Type": contentType } : {};
      return JSON.stringify({
        status,
        headers,
        bodyBase64: body.length ? body.toString("base64") : "",
      });
    } finally {
      try {
        rmSync(dir, { recursive: true, force: true });
      } catch {
        /* ignore */
      }
    }
  };
}

function parseArgs(argv) {
  const opts = { output: "output", plugin: "", url: "" };
  const rest = [];
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "-o" || a === "--output") opts.output = argv[++i] ?? "";
    else if (a === "--plugin") opts.plugin = argv[++i] ?? "";
    else if (a.startsWith("-")) throw new Error(`unknown option: ${a}`);
    else rest.push(a);
  }
  opts.url = rest.join(" ").trim();
  return opts;
}

let wasmReady;

async function ensureWasm() {
  if (wasmReady) return wasmReady;
  wasmReady = (async () => {
    const wasmPath = join(root, "kbsink.wasm");
    if (!existsSync(wasmPath)) {
      throw new Error("kbsink.wasm missing — run: ./scripts/build-wasm.sh");
    }
    installHTTPBridge();
    const require = createRequire(import.meta.url);
    const localExec = join(root, "wasm_exec.js");
    if (existsSync(localExec)) {
      require(localExec);
    } else {
      const goroot = execSync("go env GOROOT", { encoding: "utf8" }).trim();
      const fromGo =
        ["lib/wasm/wasm_exec.js", "misc/wasm/wasm_exec.js"]
          .map((rel) => join(goroot, rel))
          .find((p) => existsSync(p));
      if (!fromGo) throw new Error("wasm_exec.js missing — run: ./scripts/build-wasm.sh");
      require(fromGo);
    }
    const go = new globalThis.Go();
    const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
    void go.run(instance);
    for (let t = Date.now() + 60_000; Date.now() < t; ) {
      if (typeof globalThis.kbsinkConvertJSON === "function") return;
      await new Promise((r) => setTimeout(r, 25));
    }
    throw new Error("kbsinkConvertJSON not ready within 60s");
  })();
  return wasmReady;
}

async function main() {
  const opts = parseArgs(process.argv);
  if (!opts.url) {
    console.error("usage: node scripts/run-wasm.mjs <article-url> [--plugin id] [-o dir]");
    process.exit(2);
  }

  await ensureWasm();

  const raw = globalThis.kbsinkConvertJSON(
    JSON.stringify({
      url: opts.url,
      plugin: opts.plugin || undefined,
      outputRoot: opts.output,
    })
  );
  const res = JSON.parse(raw);
  if (!res.ok) {
    console.error(res.error ?? "conversion failed");
    process.exit(1);
  }

  const article = res.result;
  const outDir = join(root, article.outputDir || join(opts.output, article.safeTitle));
  mkdirSync(outDir, { recursive: true });
  const mdPath = join(
    outDir,
    article.markdownPath?.split("/").pop() ?? `${article.safeTitle || "article"}.md`
  );
  writeFileSync(mdPath, article.markdown ?? "", "utf8");

  let images = 0;
  for (const asset of article.assets ?? []) {
    if (!asset.dataBase64) continue;
    const rel = asset.relativePath || asset.fileName;
    if (!rel) continue;
    const dest = join(outDir, rel);
    mkdirSync(dirname(dest), { recursive: true });
    writeFileSync(dest, Buffer.from(asset.dataBase64, "base64"));
    images++;
  }

  console.log(`ok title=${article.title ?? ""}`);
  console.log(`markdown=${mdPath}`);
  console.log(`images=${images}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
