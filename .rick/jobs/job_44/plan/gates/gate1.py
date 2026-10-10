#!/usr/bin/env python3
"""gate1.py — job_44 第 1 层门禁（task1: human-loop v4 协议重构）

模块级集成测试（human 确认，agent 不得修改）：
  A. Go 构建与包测试：go build ./... + go test internal/{prompt,env,builder,handler}
  B. 文件契约：teach.md 在 / think.md、exporter.md 消失（templates + skills 两处）
  C. 全链 exporter 痕迹清除（prompt/env/builder/handler 非测试源码）
  D. 模板内容契约：teach 含 SENSE 五关注面；不含已删方法论（系统论描述符/三维打分/回流/重试）
  E. 渲染契约（dry-run）：主提示词含 teach/research 与学习材料路径约定；无 exporter；
     无阶段机程序列；无未替换变量
  F. 版本号：cmd/rick/main.go VERSION >= 5.0.14
  G. 生产零触碰：生产仓库 tracked 文件无改动

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import re
import subprocess
import sys
import tempfile


def get_project_root():
    # .rick/jobs/job_44/plan/gates/gate1.py → 6×dirname 到仓库根
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
ERRORS = []


def run(cmd, cwd=ROOT, timeout=900, env=None):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True,
                       timeout=timeout, env=env)
    if r.returncode != 0:
        ERRORS.append(f"$ {' '.join(cmd)}\n{(r.stdout + r.stderr).strip()[-2000:]}")
    return r


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def grep_all(pattern, subdirs, skip_tests=False):
    """返回 [(file, line, text)]；仅扫非二进制文本文件。"""
    hits = []
    rx = re.compile(pattern)
    for sub in subdirs:
        base = os.path.join(ROOT, sub)
        for dirpath, dirnames, filenames in os.walk(base):
            for fn in filenames:
                if not fn.endswith((".go", ".md", ".ts", ".tsx")):
                    continue
                if skip_tests and ("_test.go" in fn or fn.endswith(".test.ts")):
                    continue
                fp = os.path.join(dirpath, fn)
                try:
                    with open(fp, encoding="utf-8", errors="ignore") as f:
                        for i, line in enumerate(f, 1):
                            if rx.search(line):
                                hits.append((os.path.relpath(fp, ROOT), i, line.strip()[:160]))
                except OSError:
                    pass
    return hits


def main():
    # ---- A. Go build + 包测试 ----
    run(["go", "build", "./..."])
    run(["go", "test", "./internal/prompt/", "./internal/env/", "./internal/builder/", "./internal/handler/"])

    # ---- B. 文件契约 ----
    for p in ("internal/prompt/templates/teach.md",
              "internal/prompt/templates/skills/teach.md",
              "internal/prompt/templates/research.md",
              "internal/prompt/templates/skills/research.md",
              "internal/prompt/templates/sense_loop.md"):
        if not exists(p):
            ERRORS.append(f"[B] 必需模板文件缺失：{p}")
    for p in ("internal/prompt/templates/think.md",
              "internal/prompt/templates/skills/think.md",
              "internal/prompt/templates/exporter.md",
              "internal/prompt/templates/skills/exporter.md"):
        if exists(p):
            ERRORS.append(f"[B] 应删除的文件仍存在：{p}")

    # ---- C. exporter 痕迹清除（源码 + 模板，排除测试）----
    hits = grep_all(r"exporter", ["internal/prompt", "internal/env", "internal/builder", "internal/handler"],
                    skip_tests=True)
    if hits:
        ERRORS.append("[C] 非测试源码仍含 exporter 痕迹：\n" +
                      "\n".join(f"  {f}:{i}: {t}" for f, i, t in hits[:10]))

    # ---- D. teach 模板内容契约 ----
    teach_text = ""
    for p in ("internal/prompt/templates/teach.md", "internal/prompt/templates/skills/teach.md"):
        if exists(p):
            teach_text += read(p)
    for aspect in ("还原事实", "呈现视角", "洞察矛盾", "启发逆转", "良质批判"):
        if aspect not in teach_text:
            ERRORS.append(f"[D] teach 模板缺 SENSE 关注面：{aspect}")
    # 概念本质（human 要求：讲本质、不讲方法论）——至少有一处「本质」表述
    if "本质" not in teach_text:
        ERRORS.append("[D] teach 模板未见「本质」表述（human 要求只讲概念本质）")
    for forbidden, label in ((r"三维打分", "三维打分"),
                             (r"系统论描述符", "系统论描述符"),
                             (r"4\s*维打分|4 维打分", "4 维打分"),
                             (r"think_top_n", "think_top_n 配置引用"),
                             (r"max_backflows|反向回流", "反向回流机制"),
                             (r"node/input/output/inner/edge", "系统论五要素描述符")):
        if re.search(forbidden, teach_text):
            ERRORS.append(f"[D] teach 模板仍含已删方法论：{label}")
    # 产出契约：lessons/teach_{N}.md
    if not re.search(r"teach_\{?[Nn]|\{N\}", teach_text) and "lessons/" not in teach_text:
        ERRORS.append("[D] teach 模板未见 lessons/teach_{N}.md 产出契约")

    # research 模板：research_{N}.md 产出契约
    res_text = ""
    for p in ("internal/prompt/templates/research.md", "internal/prompt/templates/skills/research.md"):
        if exists(p):
            res_text += read(p)
    if "research_" not in res_text:
        ERRORS.append("[D] research 模板未见 research_{N}.md 落盘契约")

    # ---- E. 渲染契约（dry-run）----
    binp = os.path.join(tempfile.gettempdir(), f"rick-gate1-{os.getpid()}")
    r = subprocess.run(["go", "build", "-o", binp, "./cmd/rick"], cwd=ROOT,
                       capture_output=True, text=True, timeout=900)
    if r.returncode != 0:
        ERRORS.append("[E] 构建 gate1 自检二进制失败：\n" + (r.stdout + r.stderr)[-1500:])
    else:
        with tempfile.TemporaryDirectory(prefix="rick-gate1-ws-") as ws:
            os.makedirs(os.path.join(ws, ".rick"), exist_ok=True)
            env = os.environ.copy()
            env["HOME"] = ws  # 隔离真实 ~/.rick（domain 已知坑：子进程继承 HOME）
            r2 = subprocess.run([binp, "human-loop", "--dry-run", "gate1 自检主题"],
                                cwd=ws, capture_output=True, text=True, timeout=300, env=env)
            out = r2.stdout + r2.stderr
            if r2.returncode != 0:
                ERRORS.append(f"[E] human-loop --dry-run 退出码 {r2.returncode}：\n{out[-1500:]}")
            else:
                low = out.lower()
                for need, label in (("teach", "teach agent"), ("research", "research agent"),
                                    ("mission", "mission.md"),
                                    ("lessons", "lessons/ 教学节目录"),
                                    ("judgment", "judgment.md")):
                    if need not in low:
                        ERRORS.append(f"[E] 渲染后的主提示词缺：{label}")
                if "exporter" in low:
                    ERRORS.append("[E] 渲染后的主提示词仍含 exporter")
                for bad in ("阶段 1:S", "N1:", "反向回流", "三维打分", "系统论描述符", "{{"):
                    if bad in out:
                        ERRORS.append(f"[E] 渲染后的主提示词仍含：{bad}")
        try:
            os.remove(binp)
        except OSError:
            pass

    # ---- F. 版本号 ----
    try:
        ver = re.search(r'VERSION\s*=\s*"([0-9.]+)"', read("cmd/rick/main.go"))
        if not ver:
            ERRORS.append("[F] cmd/rick/main.go 未见 VERSION 常量")
        else:
            parts = [int(x) for x in ver.group(1).split(".")]
            if parts < [5, 0, 14]:
                ERRORS.append(f"[F] VERSION 未提升（当前 {ver.group(1)}，要求 >= 5.0.14）")
    except FileNotFoundError:
        ERRORS.append("[F] cmd/rick/main.go 缺失")

    # ---- G. 生产零触碰 ----
    wt = subprocess.run(["git", "worktree", "list", "--porcelain"], cwd=ROOT,
                        capture_output=True, text=True)
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = subprocess.run(["git", "status", "--porcelain", "-uno"], cwd=prod,
                           capture_output=True, text=True)
        if r.stdout.strip():
            ERRORS.append(f"[G][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[G][RSI] 未能定位生产仓库（worktree list 解析失败）")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
