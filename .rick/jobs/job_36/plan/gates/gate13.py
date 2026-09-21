#!/usr/bin/env python3
"""gate13：第 3 层（task26 rsi_check + task27 release --merge-source）——产出可机器校验 + 源码合并冲突即中止。prod 只读。"""
import hashlib, json, os, subprocess, sys, tempfile, urllib.request, urllib.error

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

base = (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
        md5f(os.path.join(PROD_HOME, ".rick", "web.json")))

# 先构建一次二进制：临时 cwd 下 `go run ./cmd/rick` 会因找不到 go.mod 而失败
# （门禁自身 bug，实测暴露 —— 断言必须用构建好的二进制，不能用 go run）
BIN = os.path.join(tempfile.mkdtemp(prefix="gate13-bin-"), "rick")
_r = run(["go", "build", "-o", BIN, "./cmd/rick"])
if _r.returncode != 0:
    print(json.dumps({"pass": False, "errors": [f"构建失败:\n{_r.stderr[-800:]}"]}, ensure_ascii=False, indent=1)); sys.exit(1)

# ① rsi_check 子命令存在 + 对缺证据的 job 给出 fail 与中文指引
r = run([BIN, "tools", "rsi_check", "--job", "job_36", "--json"], timeout=600)
out = (r.stdout or "") + (r.stderr or "")
if "rsi_check" not in out and r.returncode > 1:
    errors.append(f"① rsi_check 命令不可用: {out[-400:]}")
else:
    if r.returncode == 0:
        # 若 job_36 已具备全部证据也可接受；否则必须是 fail
        notes.append("① rsi_check 对 job_36 返回 pass（说明证据已齐备）")
    else:
        if "rsi" not in out:
            errors.append(f"① rsi_check 失败但输出未提及 rsi 证据路径: {out[-300:]}")
        notes.append("① rsi_check 对缺证据 job 正确返回非零")

# ② --init 生成五项证据骨架（幂等），且**骨架状态不得通过校验**（防自欺）
with tempfile.TemporaryDirectory(prefix="gate13-job-") as jd:
    os.makedirs(os.path.join(jd, ".rick", "jobs", "job_99"), exist_ok=True)
    r = run([BIN, "tools", "rsi_check", "--job", "job_99", "--init", "--json"], cwd=jd, timeout=600)
    d = os.path.join(jd, ".rick", "jobs", "job_99", "doing", "rsi")
    want_files = ["dev-iterations.md", "gates.md", "approval.md", "release.md", "resume.md"]
    if not os.path.isdir(d):
        errors.append(f"② --init 未生成 {d}: {((r.stdout or '')+(r.stderr or ''))[-300:]}")
    else:
        missing = [f for f in want_files if not os.path.exists(os.path.join(d, f))]
        if missing:
            errors.append(f"② --init 骨架缺文件: {missing}")
        # 骨架（含 TODO 占位标记）必须被判「未填写」→ 非零退出
        r2 = run([BIN, "tools", "rsi_check", "--job", "job_99"], cwd=jd, timeout=600)
        if r2.returncode == 0:
            errors.append("② 未填写的骨架竟然通过了校验（契约自欺）")
        # 幂等：二次 --init 不覆盖已填内容
        appr = os.path.join(d, "approval.md")
        with open(appr, "a") as fh:
            fh.write("APPROVED by=human at=2026-09-21T21:00:00+08:00\n")
        r3 = run([BIN, "tools", "rsi_check", "--job", "job_99", "--init", "--json"], cwd=jd, timeout=600)
        if "APPROVED by=human" not in open(appr).read():
            errors.append("② 二次 --init 覆盖了已填内容（幂等被破坏）")
        # job 自动解析：不给 --job 时应解析到唯一 job_99
        r4 = run([BIN, "tools", "rsi_check", "--json"], cwd=jd, timeout=600)
        if '"job":"job_99"' not in (r4.stdout or "").replace(" ", ""):
            errors.append(f"② 未指定 --job 时未自动解析到 job_99: {(r4.stdout or '')[:200]}")

# ③ merge-source：模拟生产（两个真实 git 仓库）——无冲突合并成功
def git(cwd, *args):
    return subprocess.run(["git", "-C", cwd, *args], capture_output=True, text=True)

with tempfile.TemporaryDirectory(prefix="gate13-git-") as g:
    prod = os.path.join(g, "prod"); dev = os.path.join(g, "dev")
    os.makedirs(prod); git(prod, "init", "-q", "-b", "main"); git(prod, "config", "user.email", "t@t"); git(prod, "config", "user.name", "t")
    open(os.path.join(prod, "f.txt"), "w").write("base\n")
    git(prod, "add", "."); git(prod, "commit", "-qm", "base")
    subprocess.run(["git", "-C", prod, "worktree", "add", "-q", "-b", "dev/self-evolve", dev], capture_output=True, text=True)
    open(os.path.join(dev, "f.txt"), "w").write("dev-change\n")
    git(dev, "add", "."); git(dev, "commit", "-qm", "dev change")
    # 调用合并（用 release 的实现：通过 rick tools release --merge-source --dry-run? 不可行）
    # → 用 Go 测试覆盖：断言 merge 包单测全绿（真实冲突/中止/恢复）
    rr = run(["go", "test", "./internal/env/release/", "-timeout", "900s", "-run", "Merge", "-count=1"])
    if rr.returncode != 0:
        errors.append(f"③ merge 单测失败:\n{(rr.stderr or rr.stdout)[-1200:]}")
    else:
        notes.append("③ merge 单测（含冲突中止/恢复）全绿")

# ④ release CLI flag 契约
r = run(["go", "run", "./cmd/rick", "tools", "release", "--help"], timeout=600)
out = (r.stdout or "") + (r.stderr or "")
if r.returncode != 0:
    errors.append(f"④ release --help 失败: {out[-400:]}")
for flag in ("--merge-source", "--no-merge-source"):
    if flag not in out:
        errors.append(f"④ release 缺 {flag}")

# ⑤ 单测 + 构建
for cmd, label in [(["go", "test", "./internal/cmd/", "./internal/env/release/", "-timeout", "900s"], "⑤ 单测"),
                   (["go", "build", "./..."], "⑤ build"), (["go", "vet", "./internal/cmd/", "./internal/env/release/"], "⑤ vet")]:
    rr = run(cmd)
    if rr.returncode != 0:
        errors.append(f"{label} 失败:\n{(rr.stderr or rr.stdout)[-1000:]}")

if (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
    md5f(os.path.join(PROD_HOME, ".rick", "web.json"))) != base:
    errors.append("❌ PROD 状态被改动")
if subprocess.run(["ls", "-d", os.path.join(PROD_HOME, "..", "AI_CODING", "rick", "bin", "releases")],
                  capture_output=True).returncode == 0:
    errors.append("❌ 生产树出现 bin/releases（gate 不得写生产树）")
if opener.open("http://127.0.0.1:8413/api/health", timeout=5).status != 200:
    errors.append("❌ PROD 健康异常")
notes.append("prod 零触碰校验完成")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
