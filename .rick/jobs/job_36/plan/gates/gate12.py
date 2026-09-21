#!/usr/bin/env python3
"""gate12：第 2 层（task24 rsi 类型后端 + task25 前端入口）——loop 必然注入 + workspace 硬校验。prod 只读。"""
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

tmp = tempfile.mkdtemp(prefix="gate12-")
BIN = os.path.join(tmp, "rick")
r = run(["go", "build", "-o", BIN, "./cmd/rick"])
if r.returncode != 0:
    print(json.dumps({"pass": False, "errors": [f"构建失败:\n{r.stderr[-1000:]}"]}, ensure_ascii=False, indent=1)); sys.exit(1)

# 造「假 dev 工作区」：rick 源码树必备文件 + loop 文件
ws = os.path.join(tmp, "ws-dev"); os.makedirs(os.path.join(ws, "cmd", "rick"), exist_ok=True)
os.makedirs(os.path.join(ws, "internal", "web"), exist_ok=True)
os.makedirs(os.path.join(ws, ".rick", "loops"), exist_ok=True)
open(os.path.join(ws, "cmd", "rick", "main.go"), "w").write("package main\n")
open(os.path.join(ws, "internal", "web", "web.go"), "w").write("package web\n")
shutil.copy(os.path.join(ROOT, ".rick", "loops", "rick-rsi-loop.md"), os.path.join(ws, ".rick", "loops", "rick-rsi-loop.md"))
os.makedirs(os.path.join(ws, ".rick"), exist_ok=True)

# 缺 loop 的源码树（用于 400 断言）
ws_noloop = os.path.join(tmp, "ws-noloop")
os.makedirs(os.path.join(ws_noloop, "cmd", "rick"), exist_ok=True)
os.makedirs(os.path.join(ws_noloop, "internal", "web"), exist_ok=True)
open(os.path.join(ws_noloop, "cmd", "rick", "main.go"), "w").write("package main\n")

# 非源码树（用于 400 断言）
ws_notree = os.path.join(tmp, "ws-notree"); os.makedirs(os.path.join(ws_notree, ".rick"), exist_ok=True)

state, home, agent = os.path.join(tmp, "state"), os.path.join(tmp, "home"), os.path.join(tmp, "agent")
for d in (state, home, agent): os.makedirs(d, exist_ok=True)
token, port = "gate12tok", free_port()
env = {**os.environ, "HOME": home, "RICK_PI_AGENT_DIR": agent, "RICK_RSI_ALLOW_PROD_TREE": "1"}
p = subprocess.Popen([BIN, "web", "--listen", "127.0.0.1", "--port", str(port), "--token", token, "--state-dir", state],
                     cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
up = False
for _ in range(80):
    time.sleep(0.25)
    if http("GET", f"http://127.0.0.1:{port}/api/health")[0] == 200: up = True; break
if not up:
    errors.append("实例未起来")
else:
    # 注册工作区
    for path, name in ((ws, "gate12-dev"), (ws_noloop, "gate12-noloop"), (ws_notree, "gate12-notree")):
        http("POST", f"http://127.0.0.1:{port}/api/workspaces", token, {"path": path, "name": name})

    # ① 合规 workspace → 创建 rsi 会话成功，且提示词/方法文件含 loop 全文
    # 从列表取 workspace id
    code_list, lbody = http("GET", f"http://127.0.0.1:{port}/api/workspaces", token)
    wsid = None
    try:
        for w in json.loads(lbody):
            if w.get("name") == "gate12-dev": wsid = w["id"]
    except Exception: pass
    if not wsid:
        errors.append("① 无法解析 dev 工作区 id")
    else:
        code, body = http("POST", f"http://127.0.0.1:{port}/api/sessions", token, {"workspace_id": wsid, "type": "rsi"})
        if code not in (200, 201):
            errors.append(f"① 创建 rsi 会话失败 HTTP {code}: {body[:250]}")
        else:
            sid = json.loads(body).get("id")
            notes.append(f"① rsi 会话已创建 id={sid}")
            # loop 全文注入：查 prompt（GET /api/sessions/{id}/prompt 返回方法/提示词文件内容）
            pc, pbody = http("GET", f"http://127.0.0.1:{port}/api/sessions/{sid}/prompt", token)
            blob = pbody if pc == 200 else ""
            if "rick-rsi-loop" not in blob:
                errors.append(f"① 会话提示词未包含 loop 标记（HTTP {pc}）: {blob[:200]}")
            for cmd in ("dev-web", "release"):
                if cmd not in blob:
                    errors.append(f"① 注入的 loop 未含机制命令 {cmd!r}")

    # ② workspace 缺 loop → 400
    code_nl, nlid = None, None
    for w in json.loads(http("GET", f"http://127.0.0.1:{port}/api/workspaces", token)[1]):
        if w.get("name") == "gate12-noloop": nlid = w["id"]
    if nlid:
        code_nl, b2 = http("POST", f"http://127.0.0.1:{port}/api/sessions", token, {"workspace_id": nlid, "type": "rsi"})
        if code_nl != 400:
            errors.append(f"② 缺 loop 的 workspace 未被拒（HTTP {code_nl}）: {b2[:200]}")

    # ③ 非 rick 源码树 → 400
    ntid = None
    for w in json.loads(http("GET", f"http://127.0.0.1:{port}/api/workspaces", token)[1]):
        if w.get("name") == "gate12-notree": ntid = w["id"]
    if ntid:
        code_nt, b3 = http("POST", f"http://127.0.0.1:{port}/api/sessions", token, {"workspace_id": ntid, "type": "rsi"})
        if code_nt != 400:
            errors.append(f"③ 非源码树未被拒（HTTP {code_nt}）: {b3[:200]}")
    p.send_signal(signal.SIGTERM)
    try: p.wait(timeout=25)
    except Exception: p.kill()

# ④ CLI 入口存在
r = run(["go", "run", "./cmd/rick", "rsi", "--help"], timeout=600)
if r.returncode != 0:
    errors.append(f"④ `rick rsi --help` 失败: {((r.stdout or '')+(r.stderr or ''))[-400:]}")

# ⑤ 前端类型与入口
need = [("web/src/types.ts", '"rsi"'), ("web/src/components/sessions/NewSessionModal.tsx", "rsi")]
for rel, needle in need:
    pth = os.path.join(ROOT, rel)
    if not os.path.exists(pth) or needle not in open(pth).read():
        errors.append(f"⑤ {rel} 未包含 {needle!r}")
rr = run(["npx", "--prefix", "web", "tsc", "--noEmit", "-p", "web"])
if rr.returncode != 0:
    errors.append(f"⑤ tsc 失败:\n{(rr.stderr or rr.stdout)[-800:]}")

# ⑥ 单测 + 构建
for cmd, label in [(["go", "test", "./internal/web/", "./internal/prompt/", "./internal/cmd/", "-timeout", "900s"], "⑥ 单测"),
                   (["go", "build", "./..."], "⑥ build")]:
    rr = run(cmd)
    if rr.returncode != 0:
        errors.append(f"{label} 失败:\n{(rr.stderr or rr.stdout)[-1000:]}")

if (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
    md5f(os.path.join(PROD_HOME, ".rick", "web.json"))) != base:
    errors.append("❌ PROD 状态被改动")
if http("GET", "http://127.0.0.1:8413/api/health")[0] != 200:
    errors.append("❌ PROD 健康异常")
notes.append("prod 零触碰校验完成")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
