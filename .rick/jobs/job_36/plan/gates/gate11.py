#!/usr/bin/env python3
"""gate11：待实现流水线第 1 层（task23）——rick-rsi-loop 制度载体 + loops_check + README 对齐。prod 只读。"""
import hashlib, json, os, subprocess, sys, tempfile, urllib.request, urllib.error

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
PROD_HOME = "/home/hadoop-recsys"
errors, notes = [], []
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

def run(cmd, cwd=ROOT, timeout=900):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def md5f(p):
    try: return hashlib.md5(open(p, "rb").read()).hexdigest()
    except OSError: return "missing"

base = (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
        md5f(os.path.join(PROD_HOME, ".rick", "web.json")))

# ① rick-rsi-loop 存在且 frontmatter + 五要素齐备
loop = os.path.join(ROOT, ".rick", "loops", "rick-rsi-loop.md")
if not os.path.exists(loop):
    errors.append("① .rick/loops/rick-rsi-loop.md 不存在")
else:
    body = open(loop).read()
    for key in ("name: rick-rsi-loop", "trigger:", "scope:"):
        if key not in body:
            errors.append(f"① loop frontmatter 缺 {key!r}")
    for sec in ("## 目标", "## 上下文管理", "## 可调用工具", "## 产出评估", "## 停止标准"):
        if sec not in body:
            errors.append(f"① loop 缺五要素小节 {sec}")
    # loop 必须把机制写成可执行命令（否则只是愿望）
    for cmd in ("dev-web", "release", "rsi_check", "门禁"):
        if cmd not in body:
            errors.append(f"① loop 未引用关键机制 {cmd!r}")
    if "dev 工作区" not in body and "dev 工作树" not in body:
        errors.append("① loop 未写「必须在 dev 工作区运行」的硬约束")

# ② README 目录与实况对齐（现存条目漏了 go-refactor-migration-loop）
readme = os.path.join(ROOT, ".rick", "loops", "README.md")
if not os.path.exists(readme):
    errors.append("② .rick/loops/README.md 不存在")
else:
    r = open(readme).read()
    for name in ("rick-rsi-loop", "go-refactor-migration-loop", "tdd-red-green-refactor-loop"):
        if name not in r:
            errors.append(f"② README 未列 {name}")

# ③ loops_check 子命令可用且对真实 .rick 通过
r = run(["go", "run", "./cmd/rick", "tools", "loops_check", "--dir", ".rick", "--json"], timeout=600)
out = (r.stdout or "") + (r.stderr or "")
if r.returncode != 0:
    errors.append(f"③ rick tools loops_check 未通过（exit={r.returncode}）:\n{out[-1200:]}")
else:
    try:
        verdict = json.loads(r.stdout.strip().splitlines()[-1])
        if not verdict.get("pass"):
            errors.append(f"③ loops_check 判定 fail: {json.dumps(verdict, ensure_ascii=False)[:400]}")
    except Exception as e:
        errors.append(f"③ loops_check 未输出 JSON 结论: {r.stdout[-300:]} ({e})")

# ④ 单测 + 构建
for cmd, label in [(["go", "test", "./internal/cmd/", "-timeout", "600s"], "④ cmd 测试"),
                   (["go", "build", "./..."], "④ build"), (["go", "vet", "./internal/cmd/"], "④ vet")]:
    rr = run(cmd)
    if rr.returncode != 0:
        errors.append(f"{label} 失败:\n{(rr.stderr or rr.stdout)[-900:]}")

if (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
    md5f(os.path.join(PROD_HOME, ".rick", "web.json"))) != base:
    errors.append("❌ PROD 状态被改动")
try:
    with opener.open("http://127.0.0.1:8413/api/health", timeout=5) as rr:
        if rr.status != 200: errors.append(f"❌ PROD 健康异常 {rr.status}")
except Exception as e:
    errors.append(f"❌ PROD 健康检查失败: {e}")
notes.append("prod 零触碰校验完成")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
