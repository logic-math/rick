#!/usr/bin/env python3
"""gate7：待实现流水线**第 1 层**（task16 隔离底座 + task18 构建指纹/前端参数化）。
全部操作在临时目录/临时端口上；生产 `~/.rick` 与 8413 只读（末尾做指纹+健康复检）。"""
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

def http(url, token=None, timeout=6):
    req = urllib.request.Request(url)
    if token: req.add_header("Authorization", "Bearer " + token)
    try:
        with opener.open(req, timeout=timeout) as r: return r.status, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e: return e.code, e.read().decode("utf-8", "replace")
    except Exception as e: return 0, str(e)

def wait_up(port, tries=60):
    for _ in range(tries):
        time.sleep(0.25)
        if http(f"http://127.0.0.1:{port}/api/health")[0] == 200: return True
    return False

prod_sessions = os.path.join(PROD_HOME, ".rick", "web", "sessions.json")
prod_webjson = os.path.join(PROD_HOME, ".rick", "web.json")
base = (md5f(prod_sessions), md5f(prod_webjson))

tmp = tempfile.mkdtemp(prefix="gate7-")
BIN = os.path.join(tmp, "rick")
fake_id = "gate7-260101000000"
r = run(["go", "build", "-ldflags", f"-X github.com/sunquan/rick/internal/cmd.BuildID={fake_id}", "-o", BIN, "./cmd/rick"])
if r.returncode != 0:
    print(json.dumps({"pass": False, "errors": [f"构建失败:\n{r.stderr[-1200:]}"]}, ensure_ascii=False, indent=1)); sys.exit(1)

dev_home, state_dir, agent_dir = os.path.join(tmp, "home"), os.path.join(tmp, "state"), os.path.join(tmp, "agent")
for d in (dev_home, state_dir, agent_dir): os.makedirs(d, exist_ok=True)
env = {**os.environ, "HOME": dev_home, "RICK_PI_AGENT_DIR": agent_dir}
token = "gate7tok"

# ① --state-dir 显式 + agent dir → 启动、锁就位、状态落位
port1 = free_port()
p1 = subprocess.Popen([BIN, "web", "--listen", "127.0.0.1", "--port", str(port1), "--token", token, "--state-dir", state_dir],
                      cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
if not wait_up(port1):
    errors.append("① --state-dir 实例未起来（health 非 200）")
else:
    if not os.path.exists(os.path.join(state_dir, "web.lock")):
        errors.append("① state-dir 下无 web.lock（flock 未生效）")
    # 状态落位断言：注册表文件是「首次写入才落盘」，所以先注册一个临时工作区再断言
    ok_ws = os.path.join(tmp, "okws"); os.makedirs(os.path.join(ok_ws, ".rick"), exist_ok=True)
    try:
        req = urllib.request.Request(f"http://127.0.0.1:{port1}/api/workspaces",
              data=json.dumps({"path": ok_ws, "name": "gate7-ok"}).encode(), method="POST",
              headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"})
        with opener.open(req, timeout=10) as rr:
            if rr.status not in (200, 201):
                errors.append(f"① 注册临时工作区失败 HTTP {rr.status}")
    except Exception as e:
        errors.append(f"① 注册临时工作区异常: {e}")
    if not os.path.exists(os.path.join(state_dir, "web.json")):
        errors.append("① 注册后 web.json 仍未落在 --state-dir（状态目录未生效）")
    # 强隔离断言：dev 实例读的是自己的会话注册表 —— 绝不能看到生产那批会话
    try:
        with opener.open(urllib.request.Request(f"http://127.0.0.1:{port1}/api/sessions",
                headers={"Authorization": "Bearer " + token}), timeout=10) as rr:
            dev_sessions = json.loads(rr.read().decode("utf-8", "replace") or "[]")
    except Exception as e:
        dev_sessions = None; errors.append(f"① 读取 dev 会话列表失败: {e}")
    if dev_sessions is not None and len(dev_sessions) != 0:
        errors.append(f"① dev 实例看到了 {len(dev_sessions)} 条会话（应为 0）→ 状态目录未真正隔离")

# ② 同 state-dir 第二实例 → flock 拒绝
port2 = free_port()
p2 = subprocess.run([BIN, "web", "--listen", "127.0.0.1", "--port", str(port2), "--token", token, "--state-dir", state_dir],
                    cwd=ROOT, env=env, capture_output=True, text=True, timeout=90)
if p2.returncode == 0:
    errors.append("② 同 state-dir 第二实例启动成功（singleton 未加固）")
else:
    msg = (p2.stderr + p2.stdout).lower()
    if "lock" not in msg and "占用" not in msg:
        errors.append(f"② 被拒但无锁提示: {(p2.stderr+p2.stdout)[-300:]}")

# ③ --state-dir 未显式给 RICK_PI_AGENT_DIR → fail-fast
p3 = subprocess.run([BIN, "web", "--listen", "127.0.0.1", "--port", str(free_port()), "--token", token,
                     "--state-dir", state_dir + "-guard"], cwd=ROOT,
                    env={**os.environ, "HOME": dev_home}, capture_output=True, text=True, timeout=90)
if p3.returncode == 0:
    errors.append("③ dev 模式未给 RICK_PI_AGENT_DIR 却被放行（隔离守卫缺失）")

# ④ dev 实例注册生产已注册的工作区 → 拒绝
if p1.poll() is None:
    try:
        prodws = json.load(open(prod_webjson))["workspaces"]
    except Exception:
        prodws = []
    if prodws:
        victim = prodws[0]["path"]
        req = urllib.request.Request(f"http://127.0.0.1:{port1}/api/workspaces",
              data=json.dumps({"path": victim, "name": "should-be-rejected"}).encode(), method="POST",
              headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"})
        try:
            with opener.open(req, timeout=10) as rr: code = rr.status
        except urllib.error.HTTPError as e: code = e.code
        except Exception: code = 0
        if code not in (400, 403, 409):
            errors.append(f"④ dev 允许注册生产工作区 {victim} → HTTP {code}（应 4xx 拒绝）")

# ⑤ 构建指纹：/api/health 免认证携带 build_id，/api/config 也带；status 语义不变
if p1.poll() is None:
    code, body = http(f"http://127.0.0.1:{port1}/api/health")
    try:
        hj = json.loads(body)
        if hj.get("build_id") != fake_id: errors.append(f"⑤ health.build_id={hj.get('build_id')!r} ≠ {fake_id!r}")
        if hj.get("status") != "ok": errors.append("⑤ health.status 语义被破坏（兼容性回归）")
    except Exception as e:
        errors.append(f"⑤ health 非 JSON/缺 build_id: {body[:200]} ({e})")
    _, cbody = http(f"http://127.0.0.1:{port1}/api/config", token)
    try:
        if json.loads(cbody).get("build_id") != fake_id:
            errors.append(f"⑤ /api/config 未携带 build_id: {cbody[:200]}")
    except Exception as e:
        errors.append(f"⑤ /api/config 解析失败: {e}")

# ⑥ 默认兼容：无 --state-dir → 状态落 $HOME/.rick
home2 = os.path.join(tmp, "home2"); os.makedirs(home2, exist_ok=True)
port6 = free_port()
p6 = subprocess.Popen([BIN, "web", "--listen", "127.0.0.1", "--port", str(port6), "--token", token],
                      cwd=ROOT, env={**os.environ, "HOME": home2}, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
if not wait_up(port6):
    errors.append("⑥ 默认实例未起来（兼容性回归）")
elif not os.path.isdir(os.path.join(home2, ".rick")):
    errors.append("⑥ 默认状态未落 $HOME/.rick（兼容性回归）")

# ⑦ 前端参数化：vite proxy 不得硬编码 6137，需支持 RICK_DEV_API
vt = os.path.join(ROOT, "web", "vite.config.ts")
src = open(vt).read() if os.path.exists(vt) else ""
if "RICK_DEV_API" not in src:
    errors.append("⑦ vite.config.ts 未参数化 dev proxy（缺 RICK_DEV_API）")
if "6137" in src:
    errors.append("⑦ vite.config.ts 仍含 6137 死值")

# ⑧ 受影响包回归
for pkg in ["./internal/web/", "./internal/handler/", "./internal/cmd/"]:
    rr = run(["go", "test", pkg, "-timeout", "600s"])
    if rr.returncode != 0:
        errors.append(f"⑧ {pkg} 测试失败:\n{rr.stderr[-1200:]}")

for p in (p1, p6):
    try: p.send_signal(signal.SIGTERM); p.wait(timeout=20)
    except Exception:
        try: p.kill()
        except Exception: pass

if (md5f(prod_sessions), md5f(prod_webjson)) != base:
    errors.append("❌ PROD 状态被改动")
if http("http://127.0.0.1:8413/api/health")[0] != 200:
    errors.append("❌ PROD 健康检查异常")
notes.append("prod 零触碰校验完成（指纹 + 健康）")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
