#!/usr/bin/env python3
"""gate12：RSI 去特殊化后的标准机制（task24/25 的修正版 + task29）。

RSI 不再是特殊会话类型（human 裁决 2026-09-22）：它只是一个普通 loop，
由 rick 标准机制发现——easy/plan 会话提示词里的「可用的项目 Loops」目录
（LoadLoopsContext：name+trigger）。本门禁断言：
① loop 载体合规（loops_check）
② 标准发现：easy 会话的提示词里必须出现 rick-rsi-loop 目录条目
③ 特殊类型已删：type=rsi 返回 400；`rick rsi` 子命令不存在
④ 前端无 rsi 类型残留；tsc/build 通过
⑤ 后端单测 + 构建 + vet 全绿
⑥ 真实生产只读回归（指纹 + 健康）
"""
import hashlib, json, os, shutil, signal, socket, subprocess, sys, tempfile, time, urllib.request, urllib.error

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
PROD_HOME = "/home/hadoop-recsys"
errors, notes = [], []
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

def run(cmd, cwd=ROOT, env=None, timeout=900):
    e = dict(os.environ); e.update(env or {})
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=e)

def md5f(p):
    try: return hashlib.md5(open(p, "rb").read()).hexdigest()
    except OSError: return "missing"

def free_port():
    s = socket.socket(); s.bind(("127.0.0.1", 0)); p = s.getsockname()[1]; s.close(); return p

def http(method, url, token=None, body=None, timeout=10):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if token: req.add_header("Authorization", "Bearer " + token)
    if data: req.add_header("Content-Type", "application/json")
    try:
        with opener.open(req, timeout=timeout) as r: return r.status, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e: return e.code, e.read().decode("utf-8", "replace")
    except Exception as e: return 0, str(e)

base = (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
        md5f(os.path.join(PROD_HOME, ".rick", "web.json")))

# ① loop 载体合规（含 rick-rsi-loop 的五要素与 frontmatter）
r = run(["go", "run", "./cmd/rick", "tools", "loops_check", "--dir", ".rick", "--json"], timeout=600)
if r.returncode != 0:
    errors.append(f"① loops_check 未通过:\n{((r.stdout or '')+(r.stderr or ''))[-900:]}")
else:
    try:
        v = json.loads(r.stdout.strip().splitlines()[-1])
        if not v.get("pass"):
            errors.append(f"① loops_check 判定 fail: {json.dumps(v, ensure_ascii=False)[:300]}")
        notes.append(f"① loops_check loops={v.get('loops')}")
    except Exception as e:
        errors.append(f"① loops_check 未输出 JSON: {r.stdout[-200:]} ({e})")

# 造「普通 rick 源码工作区」（含 loop）——与其它工作区无任何差别
tmp = tempfile.mkdtemp(prefix="gate12-")
ws = os.path.join(tmp, "ws-rick")
os.makedirs(os.path.join(ws, "cmd", "rick"), exist_ok=True)
os.makedirs(os.path.join(ws, "internal", "web"), exist_ok=True)
os.makedirs(os.path.join(ws, ".rick", "loops"), exist_ok=True)
open(os.path.join(ws, "cmd", "rick", "main.go"), "w").write("package main\n")
open(os.path.join(ws, "internal", "web", "web.go"), "w").write("package web\n")
shutil.copy(os.path.join(ROOT, ".rick", "loops", "rick-rsi-loop.md"),
            os.path.join(ws, ".rick", "loops", "rick-rsi-loop.md"))

BIN = os.path.join(tmp, "rick")
r = run(["go", "build", "-o", BIN, "./cmd/rick"])
if r.returncode != 0:
    print(json.dumps({"pass": False, "errors": [f"构建失败:\n{r.stderr[-1000:]}"]}, ensure_ascii=False, indent=1)); sys.exit(1)

state, home, agent = os.path.join(tmp, "state"), os.path.join(tmp, "home"), os.path.join(tmp, "agent")
for d in (state, home, agent): os.makedirs(d, exist_ok=True)
token, port = "gate12tok", free_port()
p = subprocess.Popen([BIN, "web", "--listen", "127.0.0.1", "--port", str(port), "--token", token, "--state-dir", state],
                     cwd=ROOT, env={**os.environ, "HOME": home, "RICK_PI_AGENT_DIR": agent},
                     stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
up = False
for _ in range(80):
    time.sleep(0.25)
    if http("GET", f"http://127.0.0.1:{port}/api/health")[0] == 200: up = True; break
if not up:
    errors.append("实例未起来")
else:
    code, _ = http("POST", f"http://127.0.0.1:{port}/api/workspaces", token, {"path": ws, "name": "gate12-rick"})
    wsid = None
    _, lbody = http("GET", f"http://127.0.0.1:{port}/api/workspaces", token)
    try:
        for w in json.loads(lbody):
            if w.get("name") == "gate12-rick": wsid = w["id"]
    except Exception: pass
    if not wsid:
        errors.append("② 无法解析工作区 id")
    else:
        # ② 标准发现：easy 会话的提示词必须含 rick-rsi-loop 目录条目（LoadLoopsContext）
        code, body = http("POST", f"http://127.0.0.1:{port}/api/sessions", token,
                          {"workspace_id": wsid, "type": "easy", "params": {"requirement": "改进 rick 自身：加一个测试功能"}})
        if code not in (200, 201):
            errors.append(f"② 创建 easy 会话失败 HTTP {code}: {body[:250]}")
        else:
            sid = json.loads(body).get("id")
            pc, pbody = http("GET", f"http://127.0.0.1:{port}/api/sessions/{sid}/prompt", token)
            blob = pbody if pc == 200 else ""
            if "rick-rsi-loop" not in blob:
                errors.append(f"② easy 会话提示词未出现 rick-rsi-loop 目录条目（标准加载机制断裂，HTTP {pc}）: {blob[:200]}")
            else:
                notes.append("② easy 提示词含 rick-rsi-loop 目录条目 ✓")
            if "可用的项目 Loops" not in blob:
                errors.append("② easy 提示词缺少「可用的项目 Loops」目录节")
        # ③ 特殊类型已删：type=rsi → 400；CLI 子命令不存在
        code, body = http("POST", f"http://127.0.0.1:{port}/api/sessions", token,
                          {"workspace_id": wsid, "type": "rsi"})
        if code == 200 or code == 201:
            errors.append(f"③ type=rsi 竟然创建成功（HTTP {code}）——特殊类型应已删除")
        else:
            notes.append(f"③ type=rsi → HTTP {code}（未知类型，符合预期）")
    p.send_signal(signal.SIGTERM)
    try: p.wait(timeout=25)
    except Exception: p.kill()

r = run(["go", "run", "./cmd/rick", "rsi", "--help"], timeout=300)
if r.returncode == 0:
    errors.append("③ `rick rsi` 子命令仍存在（应已删除）")

# ④ 前端无 rsi 类型残留 + 构建
types_ts = os.path.join(ROOT, "web", "src", "types.ts")
src = open(types_ts).read() if os.path.exists(types_ts) else ""
if '"rsi"' in src:
    errors.append("④ web/src/types.ts 仍含 \"rsi\" 类型")
if os.path.exists(os.path.join(ROOT, "web", "src", "lib", "rsi.ts")):
    errors.append("④ web/src/lib/rsi.ts 仍存在（应已删除）")
for cmd, label in [(["npx", "--prefix", "web", "tsc", "--noEmit", "-p", "web"], "④ tsc"),
                   (["npm", "--prefix", "web", "run", "build"], "④ web build")]:
    rr = run(cmd)
    if rr.returncode != 0:
        errors.append(f"{label} 失败:\n{(rr.stderr or rr.stdout)[-800:]}")

# ⑤ 后端全绿
for cmd, label in [(["go", "test", "./internal/web/", "./internal/prompt/", "./internal/cmd/", "-timeout", "900s"], "⑤ 单测"),
                   (["go", "build", "./..."], "⑤ build"),
                   (["go", "vet", "./internal/web/", "./internal/prompt/", "./internal/cmd/"], "⑤ vet")]:
    rr = run(cmd)
    if rr.returncode != 0:
        errors.append(f"{label} 失败:\n{(rr.stderr or rr.stdout)[-900:]}")

# ⑥ 生产只读回归
if (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
    md5f(os.path.join(PROD_HOME, ".rick", "web.json"))) != base:
    errors.append("❌ PROD 状态被改动")
if http("GET", "http://127.0.0.1:8413/api/health")[0] != 200:
    errors.append("❌ PROD 健康异常")
notes.append("prod 零触碰校验完成")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
