#!/usr/bin/env python3
"""gate10：待实现流水线第 4 层（task22）——端到端验收 + 文档；并做**生产回归**（prod 必须依旧健康且状态未被改动）。"""
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

sessions_p = os.path.join(PROD_HOME, ".rick", "web", "sessions.json")
webjson_p = os.path.join(PROD_HOME, ".rick", "web.json")
base = (md5f(sessions_p), md5f(webjson_p))

# ① E2E 脚本存在且通过（脚本内部自带 prod_touched=false 断言）
e2e = os.path.join(ROOT, "scripts", "self-evolve-e2e.sh")
if not os.path.exists(e2e):
    errors.append("① scripts/self-evolve-e2e.sh 不存在")
else:
    r = subprocess.run(["bash", e2e], cwd=ROOT, capture_output=True, text=True, timeout=2400, env=dict(os.environ))
    out = (r.stdout or "") + (r.stderr or "")
    if r.returncode != 0:
        errors.append(f"① E2E 脚本退出码 {r.returncode}:\n{out[-2000:]}")
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
            errors.append(f"① E2E pass=false: {json.dumps(verdict, ensure_ascii=False)[:600]}")
        if verdict.get("prod_touched") is not False:
            errors.append("① E2E 报告 prod_touched 非 false —— 触碰了生产")

# ② 运维手册：必备小节
doc = os.path.join(ROOT, "wiki", "self-evolve.md")
if not os.path.exists(doc):
    errors.append("② wiki/self-evolve.md 不存在")
else:
    body = open(doc).read()
    for key in ("dev-web", "release", "rollback", "挂起", "恢复", "已知边界"):
        if key not in body:
            errors.append(f"② wiki/self-evolve.md 缺少小节/关键词: {key}")

# ③ 验收报告：逐条对照 KR1-KR4
rep = os.path.join(ROOT, ".rick", "jobs", "job_36", "plan", "sim-report.md")
if not os.path.exists(rep):
    errors.append("③ plan/sim-report.md 不存在")
else:
    body = open(rep).read()
    for kr in ("KR1", "KR2", "KR3", "KR4"):
        if kr not in body:
            errors.append(f"③ sim-report.md 未逐条对照 {kr}")

# ④ 生产回归（关键：交付完成后 prod 必须原样健康）
code, body = http("http://127.0.0.1:8413/api/health")
if code != 200:
    errors.append(f"④ PROD /api/health = {code}（交付后生产必须可达）")
else:
    try:
        if json.loads(body).get("status") != "ok":
            errors.append("④ PROD health status != ok")
    except Exception:
        errors.append(f"④ PROD health 非 JSON: {body[:120]}")
tok = ""
try:
    tok = json.load(open(os.path.join(PROD_HOME, ".rick", "config.json"))).get("web_token", "")
except Exception: pass
code, body = http("http://127.0.0.1:8413/api/sessions", tok)
if code != 200:
    errors.append(f"④ PROD /api/sessions = {code}")
else:
    try:
        n = len(json.loads(body))
        notes.append(f"prod 会话数 = {n}（升级后仍在）")
    except Exception as e:
        errors.append(f"④ PROD sessions 解析失败: {e}")
if (md5f(sessions_p), md5f(webjson_p)) != base:
    errors.append("❌ PROD 状态文件被 gate 改动（应为只读）")
notes.append("生产回归断言完成（只读）")
print(json.dumps({"pass": not errors, "errors": errors, "notes": notes}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
