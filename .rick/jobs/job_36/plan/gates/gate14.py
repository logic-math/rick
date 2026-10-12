#!/usr/bin/env python3
"""gate14：第 4 层（task28）——RSI 端到端（模拟生产）+ 文档 + 生产回归（只读）。"""
import hashlib, json, os, subprocess, sys, urllib.request, urllib.error

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
PROD_HOME = "/home/hadoop-recsys"
errors, notes = [], []
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

def md5f(p):
    try: return hashlib.md5(open(p, "rb").read()).hexdigest()
    except OSError: return "missing"

def http(url, token=None, timeout=8):
    req = urllib.request.Request(url)
    if token: req.add_header("Authorization", "Bearer " + token)
    try:
        with opener.open(req, timeout=timeout) as r: return r.status, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e: return e.code, e.read().decode("utf-8", "replace")
    except Exception as e: return 0, str(e)

base = (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
        md5f(os.path.join(PROD_HOME, ".rick", "web.json")))

# ① RSI 端到端脚本
e2e = os.path.join(ROOT, "scripts", "rsi-loop-e2e.sh")
if not os.path.exists(e2e):
    errors.append("① scripts/rsi-loop-e2e.sh 不存在")
else:
    r = subprocess.run(["bash", e2e], cwd=ROOT, capture_output=True, text=True, timeout=3000, env=dict(os.environ))
    out = (r.stdout or "") + (r.stderr or "")
    if r.returncode != 0:
        errors.append(f"① E2E 退出码 {r.returncode}:\n{out[-2000:]}")
    verdict = None
    for line in reversed(out.strip().splitlines()):
        line = line.strip()
        if line.startswith("{") and line.endswith("}"):
            try: verdict = json.loads(line); break
            except Exception: continue
    if verdict is None:
        errors.append("① E2E 未输出结构化结论（末行应为 {pass,steps,prod_touched}）")
    else:
        if not verdict.get("pass"):
            errors.append(f"① E2E pass=false: {json.dumps(verdict, ensure_ascii=False)[:800]}")
        if verdict.get("prod_touched") is not False:
            errors.append("① E2E 报告 prod_touched 非 false")

# ② 运维手册增补章节
doc = os.path.join(ROOT, "wiki", "self-evolve.md")
if not os.path.exists(doc):
    errors.append("② wiki/self-evolve.md 不存在")
else:
    body = open(doc).read()
    for key in ("RSI", "rsi-loop", "merge-source", "冲突", "rick-dev"):
        if key not in body:
            errors.append(f"② wiki/self-evolve.md 缺少 RSI 章节关键词: {key}")

# ③ 验收报告对照 KR1-KR4（本增量）
rep = os.path.join(ROOT, ".rick", "jobs", "job_36", "plan", "sim-report.md")
if not os.path.exists(rep):
    errors.append("③ plan/sim-report.md 不存在")
else:
    body = open(rep).read()
    for kr in ("KR1", "KR2", "KR3", "KR4"):
        if kr not in body:
            errors.append(f"③ sim-report.md 未对照 {kr}")

# ④ 生产回归（只读）
code, body = http("http://127.0.0.1:8413/api/health")
if code != 200:
    errors.append(f"④ PROD /api/health = {code}")
else:
    try:
        if json.loads(body).get("status") != "ok": errors.append("④ PROD health status != ok")
    except Exception: errors.append(f"④ PROD health 非 JSON: {body[:120]}")
tok = ""
try:
    tok = json.load(open(os.path.join(PROD_HOME, ".rick", "config.json"))).get("web_token", "")
except Exception: pass
code, body = http("http://127.0.0.1:8413/api/sessions", tok)
if code != 200:
    errors.append(f"④ PROD /api/sessions = {code}")
else:
    try: notes.append(f"prod 会话数 = {len(json.loads(body))}")
    except Exception as e: errors.append(f"④ sessions 解析失败: {e}")
if (md5f(os.path.join(PROD_HOME, ".rick", "web", "sessions.json")),
    md5f(os.path.join(PROD_HOME, ".rick", "web.json"))) != base:
    errors.append("❌ PROD 状态文件被 gate 改动（应为只读）")
notes.append("生产回归断言完成（只读）")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
