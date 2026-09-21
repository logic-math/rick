#!/usr/bin/env python3
"""gate8：待实现流水线**第 2 层**（task17 dev-web 闭环 + task19 挂起/人工恢复/报告）。
全程临时目录 + 临时端口；生产 `~/.rick` 与 8413 只读（末尾指纹+健康复检）。"""
import hashlib, json, os, signal, socket, subprocess, sys, tempfile, time, urllib.request, urllib.error

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

def http(method, url, token=None, body=None, timeout=8):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if token: req.add_header("Authorization", "Bearer " + token)
    if data: req.add_header("Content-Type", "application/json")
    try:
        with opener.open(req, timeout=timeout) as r: return r.status, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e: return e.code, e.read().decode("utf-8", "replace")
    except Exception as e: return 0, str(e)

prod_sessions = os.path.join(PROD_HOME, ".rick", "web", "sessions.json")
prod_webjson = os.path.join(PROD_HOME, ".rick", "web.json")
base = (md5f(prod_sessions), md5f(prod_webjson))

# ① dev-web 子命令契约
r = run(["go", "run", "./cmd/rick", "tools", "dev-web", "--help"], timeout=300)
out = (r.stdout or "") + (r.stderr or "")
if r.returncode != 0:
    errors.append(f"① `rick tools dev-web --help` 失败:\n{out[-700:]}")
else:
    for sub in ("init", "build", "up", "restart", "status", "down"):
        if sub not in out:
            errors.append(f"① dev-web 缺少子命令 {sub}")

# ② devweb/cmd 单测
rr = run(["go", "test", "./internal/env/devweb/", "./internal/cmd/", "-timeout", "900s", "-count=1"])
if rr.returncode != 0:
    errors.append(f"② devweb/cmd 测试失败:\n{(rr.stderr or rr.stdout)[-1400:]}")

tmp = tempfile.mkdtemp(prefix="gate8-")
BIN = os.path.join(tmp, "rick")
r = run(["go", "build", "-ldflags", "-X github.com/sunquan/rick/internal/cmd.BuildID=gate8-260101000000", "-o", BIN, "./cmd/rick"])
if r.returncode != 0:
    print(json.dumps({"pass": False, "errors": [f"构建失败:\n{r.stderr[-1200:]}"]}, ensure_ascii=False, indent=1)); sys.exit(1)

home, state, agent = os.path.join(tmp, "home"), os.path.join(tmp, "state"), os.path.join(tmp, "agent")
for d in (home, state, agent, os.path.join(state, "web")): os.makedirs(d, exist_ok=True)
token, port = "gate8tok", free_port()
env = {**os.environ, "HOME": home, "RICK_PI_AGENT_DIR": agent}

def start():
    p = subprocess.Popen([BIN, "web", "--listen", "127.0.0.1", "--port", str(port), "--token", token, "--state-dir", state],
                         cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    for _ in range(60):
        time.sleep(0.25)
        if http("GET", f"http://127.0.0.1:{port}/api/health")[0] == 200: return p
    return None

def stop(p):
    if not p: return
    p.send_signal(signal.SIGTERM)
    try: p.wait(timeout=25)
    except Exception: p.kill()

# ③ overlay 热更：替换 state/dist/index.html → GET / 立即反映，进程 pid 不变
ws = os.path.join(tmp, "ws"); os.makedirs(os.path.join(ws, ".rick"), exist_ok=True)
json.dump({"version": 1, "workspaces": [{"id": "wsgate8", "path": ws, "name": "gate8ws",
          "added_at": "2026-01-01T00:00:00+08:00"}]}, open(os.path.join(state, "web.json"), "w"))
p1 = start()
if not p1:
    errors.append("③ 实例未起来")
else:
    ov = os.path.join(state, "dist"); os.makedirs(ov, exist_ok=True)
    m1 = "gate8-overlay-A"
    open(os.path.join(ov, "index.html"), "w").write(f"<html><body>{m1}</body></html>")
    code, body = http("GET", f"http://127.0.0.1:{port}/")
    if m1 not in body: errors.append("③ overlay 首版未生效（GET / 未返回覆盖层）")
    m2 = "gate8-overlay-B"
    open(os.path.join(ov, "index.html"), "w").write(f"<html><body>{m2}</body></html>")
    code, body = http("GET", f"http://127.0.0.1:{port}/")
    if m2 not in body: errors.append("③ overlay 更新未即时生效（热更失败）")
    if p1.poll() is not None: errors.append("③ 覆盖层更新期间 dev 进程退出了（不应重启）")
    stop(p1)

# ④ 挂起语义：active → suspended（不是 error），且**不自动恢复**
json.dump({"version": 1, "sessions": [
  {"id": "s-active", "workspace_id": "wsgate8", "type": "easy", "title": "was running", "params": {"job": "job_1"},
   "status": "active", "pi_session_id": "pi-active", "created_at": "2026-01-01T00:00:00+08:00"},
  {"id": "s-susp", "workspace_id": "wsgate8", "type": "easy", "title": "already suspended", "params": {"job": "job_2"},
   "status": "suspended", "pi_session_id": "pi-susp", "created_at": "2026-01-01T00:00:00+08:00"},
]}, open(os.path.join(state, "web", "sessions.json"), "w"))
p2 = start()
if not p2:
    errors.append("④ 第二轮实例未起来")
else:
    code, body = http("GET", f"http://127.0.0.1:{port}/api/sessions", token)
    try:
        items = {s["id"]: s["status"] for s in json.loads(body)}
    except Exception as e:
        items = {}; errors.append(f"④ 会话列表解析失败: {body[:200]} ({e})")
    if items.get("s-active") != "suspended":
        errors.append(f"④ 重启后 active 会话 = {items.get('s-active')!r}，期望 suspended（不是 error）")
    if items.get("s-susp") != "suspended":
        errors.append(f"④ 既有挂起会话被改动 = {items.get('s-susp')!r}")
    code, body = http("GET", f"http://127.0.0.1:{port}/api/sessions/s-active", token)
    try:
        if json.loads(body).get("busy") is True:
            errors.append("④ 挂起会话 busy=true（不应自动跑）")
    except Exception: pass
    # 恢复报告
    code, body = http("GET", f"http://127.0.0.1:{port}/api/recovery", token)
    if code != 200:
        errors.append(f"④ GET /api/recovery = {code}（应 200）")
    else:
        try:
            rep = json.loads(body)
            for k in ("at", "suspended", "recovered", "failed"):
                if k not in rep: errors.append(f"④ 恢复报告缺字段 {k}")
        except Exception as e:
            errors.append(f"④ 恢复报告非 JSON: {e}")
    # /continue 接受 suspended（不得 404/409）；不存在 → 404
    code, body = http("POST", f"http://127.0.0.1:{port}/api/sessions/s-susp/continue", token)
    if code in (404, 409):
        errors.append(f"④ POST /continue 对 suspended 返回 {code}（应尝试恢复）: {body[:160]}")
    code, _ = http("POST", f"http://127.0.0.1:{port}/api/sessions/nope/continue", token)
    if code != 404:
        errors.append(f"④ /continue 对不存在会话返回 {code}，期望 404")
    stop(p2)
    # ⑤ 优雅关停：active → suspended 且写 suspend.json
    time.sleep(0.5)
    try:
        after = {s["id"]: s["status"] for s in json.load(open(os.path.join(state, "web", "sessions.json")))["sessions"]}
        if after.get("s-active") != "suspended":
            errors.append(f"⑤ 优雅关停后状态 = {after.get('s-active')!r}，期望 suspended")
    except Exception as e:
        errors.append(f"⑤ 关停后注册表读取失败: {e}")
    if not os.path.exists(os.path.join(state, "suspend.json")):
        errors.append("⑤ 关停未写 suspend.json（intent 持久化缺失）")

# ⑥ 单测回归 + 反向断言（不得存在自动恢复）
rr = run(["go", "test", "./internal/web/", "./internal/handler/", "-timeout", "900s"])
if rr.returncode != 0:
    errors.append(f"⑥ 单测失败:\n{rr.stderr[-1400:]}")
rec = os.path.join(ROOT, "internal", "web", "recovery.go")
if os.path.exists(rec):
    body = open(rec).read()
    if "AutoResume" in body or "autoResume" in body:
        errors.append("⑥ recovery.go 出现 AutoResume —— 自动恢复被 human 明确否决（J-L6-6）")

if (md5f(prod_sessions), md5f(prod_webjson)) != base:
    errors.append("❌ PROD 状态被改动")
if http("GET", "http://127.0.0.1:8413/api/health")[0] != 200:
    errors.append("❌ PROD 健康检查异常")
notes.append("prod 零触碰校验完成；挂起语义与人工恢复路径均已断言")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
