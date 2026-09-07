#!/usr/bin/env python3
"""gate6：第 6 层（task15 终局）——E2E：真实服务器进程启动/认证/静态/API/SSE/优雅退出 + dist + 文档。"""
import json, os, signal, socket, subprocess, sys, tempfile, time, urllib.request, urllib.error

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
errors = []

# 绕过本机 http_proxy 对 127.0.0.1 的拦截（实证：公司代理返回 503）
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def http(method, url, token=None, timeout=10):
    req = urllib.request.Request(url, method=method)
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with opener.open(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")

# ① dist 构建产物就绪
dist = os.path.join(ROOT, "web/dist")
if not os.path.exists(os.path.join(dist, "index.html")):
    errors.append("web/dist/index.html 不存在（dist 未构建/未提交）")
assets = os.path.join(dist, "assets")
if not os.path.isdir(assets) or not os.listdir(assets):
    errors.append("web/dist/assets/ 为空")

# ② 文档
if not os.path.exists(os.path.join(ROOT, "wiki/web-ui.md")):
    errors.append("wiki/web-ui.md 不存在")

# ③ 受影响包全回归
for pkg in ["./internal/cmd/", "./internal/env/", "./internal/web/...", "./internal/runtime/", "./internal/handler/..."]:
    r = run(["go", "test", pkg, "-timeout", "600s"])
    if r.returncode != 0:
        errors.append(f"{pkg} 测试失败:\n{r.stderr[-1500:]}")

# ④ 构建 + 启动真实服务 E2E
r = run(["go", "build", "-o", "bin/rick", "./cmd/rick"])
if r.returncode != 0:
    errors.append(f"构建失败:\n{r.stderr[-1500:]}")
    print(json.dumps({"pass": False, "errors": errors}, ensure_ascii=False, indent=1)); sys.exit(1)

port = 16374
with socket.socket() as s:
    s.settimeout(1)
    if s.connect_ex(("127.0.0.1", port)) == 0:
        port = 16375

e2e_home = tempfile.mkdtemp(prefix="rick-web-e2e-")
env = {**os.environ, "HOME": e2e_home, "no_proxy": "127.0.0.1,localhost",
       "NO_PROXY": "127.0.0.1,localhost"}
proc = subprocess.Popen([os.path.join(ROOT, "bin/rick"), "web", "--port", str(port),
                         "--listen", "127.0.0.1", "--token", "e2etest-token"],
                        cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
base = f"http://127.0.0.1:{port}"
try:
    ok = False
    for _ in range(50):
        time.sleep(0.2)
        try:
            st, _ = http("GET", f"{base}/api/health", timeout=2)
            if st == 200:
                ok = True; break
        except Exception:
            pass
    if not ok:
        out = proc.stdout.read(2000) if proc.poll() is not None else "(still running)"
        errors.append(f"服务 10s 未就绪:\n{out}")
    else:
        st, _ = http("GET", f"{base}/api/config")
        if st != 401: errors.append(f"/api/config 无 token 应 401，实际 {st}")
        st, body = http("GET", f"{base}/api/config", token="e2etest-token")
        if st != 200 or "rick_version" not in body:
            errors.append(f"/api/config 带 token 异常 {st}: {body[:200]}")
        st, body = http("GET", f"{base}/", timeout=5)
        if st != 200 or "root" not in body:
            errors.append(f"GET / 异常 {st}")
        st, _ = http("GET", f"{base}/some/route", timeout=5)
        if st != 200:
            errors.append(f"SPA fallback 异常 {st}")
        # 工作区注册往返 + jobs 真实数据
        req = urllib.request.Request(f"{base}/api/workspaces", method="POST",
              data=json.dumps({"path": ROOT}).encode(),
              headers={"Authorization": "Bearer e2etest-token", "Content-Type": "application/json"})
        try:
            with opener.open(req, timeout=10) as resp:
                if resp.status not in (200, 201):
                    errors.append(f"注册工作区异常 {resp.status}")
        except urllib.error.HTTPError as e:
            errors.append(f"注册工作区失败 {e.code}: {e.read().decode()[:300]}")
        st, body = http("GET", f"{base}/api/workspaces", token="e2etest-token")
        if st != 200 or ROOT not in body:
            errors.append(f"工作区列表异常: {body[:300]}")
        else:
            ws = json.loads(body)[0]["id"]
            # 语义升级（job_36 迭代需求：已完成 job 自动归档 done）：默认列表只含
            # 进行中 job——fixture 仓库 job 全完成后默认列表可为空；include_archived
            # 必须仍能取到全部 job（数据真实可达）。
            st, body = http("GET", f"{base}/api/workspaces/{ws}/jobs", token="e2etest-token")
            if st != 200:
                errors.append(f"jobs 列表异常 {st}: {body[:300]}")
            st, body = http("GET", f"{base}/api/workspaces/{ws}/jobs?include_archived=true", token="e2etest-token")
            if st != 200 or "job_" not in body:
                errors.append(f"jobs include_archived 列表异常 {st}: {body[:300]}")
        # SSE：行读 + 截止时间（流式 read(4096) 会阻塞到超时）
        try:
            req = urllib.request.Request(f"{base}/api/events?token=e2etest-token")
            with opener.open(req, timeout=5) as resp:
                deadline = time.time() + 3
                data = ""
                while time.time() < deadline and "server_info" not in data:
                    line = resp.fp.readline()
                    if not line:
                        break
                    data += line.decode("utf-8", "replace")
                if "server_info" not in data:
                    errors.append(f"SSE 未收到 server_info: {data[:200]}")
        except Exception as e:
            errors.append(f"SSE 连接失败: {e}")
finally:
    proc.send_signal(signal.SIGINT)
    try:
        proc.wait(timeout=10)
    except subprocess.TimeoutExpired:
        proc.kill()

if os.path.exists(os.path.join(e2e_home, ".rick", "web.pid")):
    errors.append("web.pid 残留未清理")

print(json.dumps({"pass": len(errors) == 0, "errors": errors}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
