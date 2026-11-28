#!/usr/bin/env node
// pi-daemon-shim: keeps one `pi --mode rpc` process alive behind a unix
// socket so the rick web server can restart without interrupting the pi
// session (in-flight tool calls / subagents keep running).
//
// argv: node daemon_shim.mjs <socket-path> -- <pi-binary> [pi-args...]
//
// Contract (rick internal/runtime/daemon.go is the client):
// - Listens on <socket-path> (unlink stale first). One active client; a new
//   connection REPLACES the previous one (server restart reconnects cleanly).
// - pi stdout lines are forwarded to the attached client; while detached they
//   accumulate in an in-memory ring (BUFFER_CAP lines, oldest dropped with a
//   counter). On (re)connect the shim first sends
//   {"type":"shim_attached","buffered":N,"dropped":M} then the buffered lines.
// - Client bytes are forwarded verbatim to pi stdin. No client → pi stdin
//   simply stays open (never EOF — pi --mode rpc exits on stdin EOF).
// - Writes <socket-path>.meta.json {version, shim_pid, pi_pid, pgid} so the
//   Go side can kill the whole process group (shim+pi+grandchildren).
// - pi exit → unlink socket+meta, exit with pi's code.
// - SIGTERM/SIGINT → exit (cleanup via exit handler). The Go side signals the
//   whole process group, so pi receives TERM too.
import { spawn } from "node:child_process";
import { createServer } from "node:net";
import { writeFileSync, unlinkSync } from "node:fs";

const argv = process.argv.slice(2);
const dash = argv.indexOf("--");
if (argv.length < 3 || dash !== 1) {
  console.error("usage: daemon_shim.mjs <socket-path> -- <pi-binary> [pi-args...]");
  process.exit(2);
}
const socketPath = argv[0];
const [piBin, ...piArgs] = argv.slice(dash + 1);

const BUFFER_CAP = 2000;
let buffered = [];
let dropped = 0;
let client = null;
let closed = false;

const pi = spawn(piBin, piArgs, { stdio: ["pipe", "pipe", "inherit"] });

function cleanup() {
  if (closed) return;
  closed = true;
  try { unlinkSync(socketPath); } catch {}
  try { unlinkSync(socketPath + ".meta.json"); } catch {}
}
process.on("exit", cleanup);
process.on("SIGTERM", () => process.exit(0));
process.on("SIGINT", () => process.exit(0));

// pgid: Go spawns this shim with Setpgid (syscall.SysProcAttr), so the
// shim IS the process-group leader — its pid equals the pgid by
// construction (pi and every tool call it spawns inherit the group, so
// kill(-pgid) reaches the whole tree). Node ≥26 removed process.getpgid,
// so the identity is used directly.
writeFileSync(socketPath + ".meta.json", JSON.stringify({
  version: 1, shim_pid: process.pid, pi_pid: pi.pid, pgid: process.pid,
}));

function attach(conn) {
  if (client) { try { client.destroy(); } catch {} }
  client = conn;
  conn.write(JSON.stringify({ type: "shim_attached", buffered: buffered.length, dropped: dropped }) + "\n");
  for (const line of buffered) conn.write(line);
  buffered = [];
  dropped = 0;
  conn.on("data", (chunk) => { if (pi.stdin.writable) pi.stdin.write(chunk); });
  conn.on("error", () => {});
  conn.on("close", () => { if (client === conn) client = null; });
}

const server = createServer(attach);
try { unlinkSync(socketPath); } catch {}
server.listen(socketPath, () => {
  console.error("[pi-daemon-shim] listening " + socketPath + " pi_pid=" + pi.pid);
});

pi.stdout.setEncoding("utf8");
let tail = "";
pi.stdout.on("data", (chunk) => {
  tail += chunk;
  let idx;
  while ((idx = tail.indexOf("\n")) >= 0) {
    const line = tail.slice(0, idx + 1);
    tail = tail.slice(idx + 1);
    if (client) client.write(line);
    else {
      buffered.push(line);
      if (buffered.length > BUFFER_CAP) { buffered.shift(); dropped++; }
    }
  }
});
pi.stdout.on("end", () => {
  if (tail) {
    const line = tail + "\n";
    if (client) client.write(line); else buffered.push(line);
  }
});
pi.on("exit", (code) => { server.close(); cleanup(); process.exit(code ?? 0); });
pi.on("error", (err) => { console.error("[pi-daemon-shim] spawn failed:", err); process.exit(3); });
