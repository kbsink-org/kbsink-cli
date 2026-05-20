#!/usr/bin/env node
/**
 * Run kbsink.wasm from Node (after ./scripts/build-wasm.sh).
 *
 *   node scripts/run-wasm.mjs <article-url>
 */
import { createRequire } from "node:module";
import {
  readFileSync,
  writeFileSync,
  mkdirSync,
  existsSync,
} from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { execSync } from "node:child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, "..");

const KBSINK_HTTP_ROUND_TRIP = "kbsinkHTTPRoundTrip";

const WECHAT_UA =
  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.49(0x18003137) NetType/WIFI Language/zh_CN";

function buildRequestHeaders(url, incoming) {
  const lower = url.toLowerCase();
  const headers = {
    "User-Agent":
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    ...incoming,
  };
  if (lower.includes("mp.weixin.qq.com")) {
    headers["User-Agent"] = WECHAT_UA;
    if (!headers.Referer) headers.Referer = "https://mp.weixin.qq.com/";
    if (!headers["Accept-Language"]) {
      headers["Accept-Language"] = "zh-CN,zh;q=0.9,en;q=0.8";
    }
    if (!headers.Accept) {
      headers.Accept =
        "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8";
    }
  }
  if (
    lower.includes("qpic.cn") ||
    lower.includes("mmbiz") ||
    lower.includes("qlogo.cn") ||
    lower.includes("mp.weixin.qq.com")
  ) {
    if (!headers.Referer) headers.Referer = "https://mp.weixin.qq.com/";
  }
  return headers;
}

/** Route wasm HTTP through Node fetch (Go js/wasm DNS is broken on some hosts). */
function installNodeHTTPBridge() {
  globalThis[KBSINK_HTTP_ROUND_TRIP] = (payloadJson) =>
    new Promise(async (resolve, reject) => {
      try {
        const req = JSON.parse(payloadJson);
        const headers = buildRequestHeaders(req.url, req.headers ?? {});
        const timeoutMs =
          typeof req.timeoutMs === "number" && req.timeoutMs > 0
            ? req.timeoutMs
            : 300_000;
        const init = {
          method: req.method || "GET",
          headers,
          redirect: "follow",
        };
        if (typeof AbortSignal !== "undefined" && AbortSignal.timeout) {
          init.signal = AbortSignal.timeout(timeoutMs);
        }
        if (req.bodyB64) {
          init.body = Buffer.from(req.bodyB64, "base64");
        }
        const res = await fetch(req.url, init);
        const buf = Buffer.from(await res.arrayBuffer());
        const outHeaders = {};
        res.headers.forEach((v, k) => {
          outHeaders[k] = outHeaders[k] ? `${outHeaders[k]}, ${v}` : v;
        });
        resolve(
          JSON.stringify({
            status: res.status,
            statusText: res.statusText,
            headers: outHeaders,
            bodyBase64: buf.length ? buf.toString("base64") : "",
          })
        );
      } catch (e) {
        reject(e);
      }
    });
}

const USAGE = `Usage: node scripts/run-wasm.mjs [options] <article-url>

Options:
  -o, --output <dir>       output root (default: output)
  --plugin <id>            wechat | xhs | douyin
  --video-mode <mode>      link | embed (default: link)
  --timeout-ms <n>         conversion timeout in ms (default: 600000)
  --json <file>            also write full wasm JSON response
  -h, --help               show this help

Example:
  node scripts/run-wasm.mjs "https://mp.weixin.qq.com/s/…"
`;

function die(msg, code = 1) {
  console.error(msg);
  process.exit(code);
}

function parseArgs(argv) {
  const opts = {
    output: "output",
    plugin: "",
    videoMode: "link",
    timeoutMs: 600_000,
    jsonOut: "",
    url: "",
  };
  const rest = [];
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "-h" || a === "--help") {
      opts.help = true;
      continue;
    }
    if (a === "-o" || a === "--output") {
      opts.output = argv[++i] ?? "";
      continue;
    }
    if (a === "--plugin") {
      opts.plugin = argv[++i] ?? "";
      continue;
    }
    if (a === "--video-mode") {
      opts.videoMode = argv[++i] ?? "";
      continue;
    }
    if (a === "--timeout-ms") {
      opts.timeoutMs = Number(argv[++i]);
      continue;
    }
    if (a === "--json") {
      opts.jsonOut = argv[++i] ?? "";
      continue;
    }
    if (a.startsWith("-")) {
      die(`unknown option: ${a}\n${USAGE}`);
    }
    rest.push(a);
  }
  if (rest.length > 0) {
    opts.url = rest.join(" ").trim();
  }
  return opts;
}

function resolveWasmExec() {
  const local = join(root, "wasm_exec.js");
  if (existsSync(local)) return local;
  const goroot = execSync("go env GOROOT", { encoding: "utf8" }).trim();
  for (const rel of ["lib/wasm/wasm_exec.js", "misc/wasm/wasm_exec.js"]) {
    const p = join(goroot, rel);
    if (existsSync(p)) return p;
  }
  die("wasm_exec.js missing — run: ./scripts/build-wasm.sh");
}

let wasmReady;

async function ensureWasm() {
  if (wasmReady) return wasmReady;
  wasmReady = (async () => {
    const wasmPath = join(root, "kbsink.wasm");
    if (!existsSync(wasmPath)) {
      die("kbsink.wasm missing — run: ./scripts/build-wasm.sh");
    }
    installNodeHTTPBridge();
    const require = createRequire(import.meta.url);
    require(resolveWasmExec());
    if (typeof globalThis.Go !== "function") {
      die("wasm_exec.js did not define globalThis.Go");
    }
    const go = new globalThis.Go();
    const bytes = readFileSync(wasmPath);
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
    void go.run(instance);
    const deadline = Date.now() + 60_000;
    while (Date.now() < deadline) {
      if (typeof globalThis.kbsinkConvertJSON === "function") return;
      await new Promise((r) => setTimeout(r, 25));
    }
    die("kbsinkConvertJSON not ready within 60s");
  })();
  return wasmReady;
}

function convert(req) {
  const raw = globalThis.kbsinkConvertJSON(JSON.stringify(req));
  try {
    return JSON.parse(raw);
  } catch (e) {
    die(`invalid JSON from wasm: ${raw.slice(0, 500)}\n${e}`);
  }
}

function writeOutput(result, outputRoot) {
  const outDir = join(root, result.outputDir || join(outputRoot, result.safeTitle));
  mkdirSync(outDir, { recursive: true });

  const mdName = result.markdownPath
    ? result.markdownPath.split("/").pop()
    : `${result.safeTitle || "article"}.md`;
  const mdPath = join(outDir, mdName);
  writeFileSync(mdPath, result.markdown ?? "", "utf8");

  let images = 0;
  let videos = 0;
  for (const asset of result.assets ?? []) {
    if (!asset.dataBase64) continue;
    const rel = asset.relativePath || asset.fileName;
    if (!rel) continue;
    const dest = join(outDir, rel);
    mkdirSync(dirname(dest), { recursive: true });
    writeFileSync(dest, Buffer.from(asset.dataBase64, "base64"));
    if (asset.type === "video") videos++;
    else images++;
  }

  return { outDir, mdPath, images, videos };
}

async function main() {
  const opts = parseArgs(process.argv);
  if (opts.help) {
    console.log(USAGE.trim());
    process.exit(0);
  }
  if (!opts.url) {
    die(USAGE.trim());
  }

  await ensureWasm();

  const req = {
    url: opts.url,
    plugin: opts.plugin || undefined,
    videoMode: opts.videoMode,
    timeoutMs: opts.timeoutMs,
    outputRoot: opts.output,
  };

  console.error("convert:", opts.url);
  const res = convert(req);
  if (!res.ok) {
    die(res.error ?? "conversion failed");
  }

  const result = res.result;
  if (opts.jsonOut) {
    writeFileSync(opts.jsonOut, JSON.stringify(res, null, 2), "utf8");
    console.error(`json: ${opts.jsonOut}`);
  }

  const { outDir, mdPath, images, videos } = writeOutput(result, opts.output);
  console.log(`title: ${result.title ?? ""}`);
  console.log(`markdown: ${mdPath}`);
  console.log(`output: ${outDir}`);
  console.log(`images: ${images}`);
  console.log(`videos: ${videos}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
