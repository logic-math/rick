#!/usr/bin/env python3
"""gate9：待实现流水线第 3 层（task20 前端挂起 UI + task21 `rick tools release`）——提升能力与前端契约（prod 只读）。"""
import hashlib, json, os, signal, socket, subprocess, sys, tempfile, time, urllib.request, urllib.error

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
PROD_HOME = "/home/hadoop-recsys"
errors, notes = [], []
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

def run(cmd, cwd=ROOT, env=None, timeout=1200):
    e = dict(os.environ); e.update(env or {})
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=e)

def md5f(p):
    try: return hashlib.md5(open(p, "rb").read()).hexdigest()
    except OSError: return "missing"

def http(url, timeout=5):
    try:
        with opener.open(url, timeout=timeout) as r: return r.status, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e: return e.code, ""
    except Exception: return 0, ""

prod_sessions = os.path.join(PROD_HOME, ".rick", "web", "sessions.json")
prod_repo_bin = os.path.join(PROD_HOME, "..")  # 占位，避免误用
base = (md5f(prod_sessions), md5f(os.path.join(PROD_HOME, ".rick", "web.json")))

# ① release 包单测（全部在 t.TempDir 造「假生产」上跑：换链/回滚/门禁失败/健康失败自愈）
r = run(["go", "test", "./internal/env/release/", "-timeout", "900s", "-count=1"])
if r.returncode != 0:
    errors.append(f"① release 包测试失败:\n{(r.stderr or r.stdout)[-1500:]}")

# ② release CLI 契约：子命令存在 + flag 齐备（--yes/--rollback/--dry-run）
r = run(["go", "run", "./cmd/rick", "tools", "release", "--help"], timeout=300)
out = (r.stdout or "") + (r.stderr or "")
if r.returncode != 0:
    errors.append(f"② `rick tools release --help` 失败:\n{out[-600:]}")
for flag in ("--yes", "--rollback", "--dry-run"):
    if flag not in out:
        errors.append(f"② release 缺少 {flag}")

# ③ 前端：挂起态 UI 存在且类型/构建通过
need = [
    ("web/src/types.ts", '"suspended"'),
    ("web/src/api/client.ts", "continueSession"),
    ("web/src/components/recovery/RecoveryBanner.tsx", "recovery"),
]
for rel, needle in need:
    p = os.path.join(ROOT, rel)
    if not os.path.exists(p):
        errors.append(f"③ 缺少 {rel}")
    elif needle not in open(p).read():
        errors.append(f"③ {rel} 未包含 {needle!r}")
for cmd, label in [(["npx", "--prefix", "web", "tsc", "--noEmit", "-p", "web"], "③ tsc"),
                   (["npm", "--prefix", "web", "run", "build"], "③ web build")]:
    rr = run(cmd)
    if rr.returncode != 0:
        errors.append(f"{label} 失败:\n{(rr.stderr or rr.stdout)[-1000:]}")

# ④ 前端文案不得把挂起说成「已中断（agent 进程不在）」
steer = os.path.join(ROOT, "web/src/components/chat/SteerBar.tsx")
if os.path.exists(steer):
    s = open(steer).read()
    if "suspended" not in s:
        errors.append("④ SteerBar 未处理 suspended 分支（挂起应显示专属文案与恢复按钮）")

# ⑤ release 不得在代码里写死真实端口/生产路径（防误触）
rel_src_dir = os.path.join(ROOT, "internal/env/release")
hard = []
for dirpath, _, files in os.walk(rel_src_dir):
    for f in files:
        if not f.endswith(".go"): continue
        body = open(os.path.join(dirpath, f)).read()
        if "8413" in body and "Test" not in f:
            hard.append(f"{f}: 出现字面量 8413")
if hard:
    errors.append(f"⑤ release 实现里出现生产端口字面量（应参数化）: {hard}")

if (md5f(prod_sessions), md5f(os.path.join(PROD_HOME, ".rick", "web.json"))) != base:
    errors.append("❌ PROD 状态被改动")
if http("http://127.0.0.1:8413/api/health")[0] != 200:
    errors.append("❌ PROD 健康检查异常")
notes.append("release 测试均在模拟生产（临时目录/临时端口）上执行；prod 零触碰校验完成")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
