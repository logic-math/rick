#!/usr/bin/env python3
"""gate4.py — job_44 第 4 层门禁（task5: SENSE 范式重构 human-loop v6）

模块级集成测试（human 确认，agent 不得修改）：
  A. Go 构建与包测试：go build ./... + go test internal/{prompt,env,builder,handler}
  B. sense canon 内容契约：五步权威语义齐备；含「范式/形式自由」声明；
     无旧方法论（系统论描述符/三维打分/反向回流/旧阶段标题）
  C. 全仓旧方法论痕迹清除（templates + builder SOP）
  D. research 提示词：第 1 步（递归至常识性置信信息源 + 价值性信息上报 values.md）
  E. teach 提示词：四种任务模式（facts 由 research 承担；teach 的 syllabus/lesson/syllabus-update/assess）
     + 写作要求（非固定模板）+ 还原作者 SENSE 过程七问 + 教学良质自检 + 追问回到学生自己
     + assess=human 当老师/teach 当学生+教练
  F. parent 提示词：价值裁决回合 + 大纲确认 + 大纲动态变更(changelog) + 考核演练 + sense canon 引用
  G. 代码：values_path / sense_skill_path 注入；旧 SOP 文本清理；VERSION >= 5.0.15
  H. 渲染契约（dry-run）：含 values.md/syllabus.md/skill_sense 与五步名；无旧方法论痕迹
  I. dream 兼容：dream.md 仍引用 sense_skill_path（共用 canon）
  J. 生产零触碰

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import re
import subprocess
import sys
import tempfile

OLD_METHOD = r"系统论描述符|node/input/output/inner/edge|反向回流|三维打分|全局性\s*[:：]\s*1\.0"


def get_project_root():
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


def need(content, patterns, ctx):
    """patterns: [(regex, label)]；缺失即记错。"""
    for pat, label in patterns:
        if not re.search(pat, content, re.S):
            ERRORS.append(f"[{ctx}] 未见 {label}（pattern: {pat}）")


def main():
    # ---- A. build + tests ----
    run(["go", "build", "./..."])
    run(["go", "test", "./internal/prompt/", "./internal/env/", "./internal/builder/", "./internal/handler/"])

    # ---- B. sense canon ----
    sense_p = "internal/prompt/templates/skills/sense.md"
    if not exists(sense_p):
        ERRORS.append(f"[B] 缺失 {sense_p}")
        sense = ""
    else:
        sense = read(sense_p)
        need(sense, [
            (r"事实还原", "第一步 事实还原"),
            (r"构建视角", "第二步 构建视角"),
            (r"洞察矛盾", "第三步 洞察矛盾"),
            (r"辩证逆转", "第四步 辩证逆转"),
            (r"良质批判", "第五步 良质批判"),
            (r"常识[性]?[的]?置信信息源|常识.*置信", "递归终止于常识性置信信息源"),
            (r"价值性信息.*上报|上报.*价值性信息", "价值性信息必须上报 human"),
            (r"强相关.*弱相关|弱相关.*强相关", "强相关 / 弱相关 知识领域"),
            (r"收敛性思维", "强相关=收敛性思维"),
            (r"发散性思维", "弱相关=发散性思维"),
            (r"全局性", "全局性成因"),
            (r"根本性", "根本性成因"),
            (r"决定性", "决定性成因"),
            (r"备选", "多个备选供 human 判断"),
            (r"阻碍", "辩证逆转的 X=阻碍"),
            (r"期望", "辩证逆转的 Y=期望状态"),
            (r"前提", "X 是 Y 必然成立的前提"),
            (r"手段", "把阻碍变成达成目标的手段"),
            (r"假设", "良质批判：各假设是否足够置信"),
            (r"置信", "假设的置信度"),
            (r"必然", "能否必然得出目标可达成"),
            (r"范式", "明示这是思考范式"),
            (r"自由|不固定|形式", "形式自由（非流程/非固定模板）声明"),
        ], "B")
        if re.search(OLD_METHOD, sense) or re.search(r"^##\s*S\s*—|^##\s*E\s*—|^##\s*N1", sense, re.M):
            ERRORS.append("[B] sense.md 仍含旧方法论痕迹（系统论描述符/三维打分/回流/旧阶段标题）")

    # ---- C. 全仓旧方法论痕迹 ----
    hits = []
    for sub in ("internal/prompt/templates", "internal/builder"):
        base = os.path.join(ROOT, sub)
        for dirpath, _dirs, files in os.walk(base):
            for fn in files:
                if not fn.endswith((".md", ".go")):
                    continue
                fp = os.path.join(dirpath, fn)
                try:
                    with open(fp, encoding="utf-8", errors="ignore") as f:
                        for i, line in enumerate(f, 1):
                            if re.search(OLD_METHOD, line):
                                hits.append(f"{os.path.relpath(fp, ROOT)}:{i}")
                except OSError:
                    pass
    if hits:
        ERRORS.append("[C] 旧方法论痕迹残留：\n  " + "\n  ".join(hits[:10]))

    # ---- D. research 提示词 ----
    for p, ctx in (("internal/prompt/templates/research.md", "D-research"),
                   ("internal/prompt/templates/skills/research.md", "D-research-skill")):
        if not exists(p):
            ERRORS.append(f"[{ctx}] 缺失 {p}")
            continue
        c = read(p)
        need(c, [(r"递归", "递归调研"), (r"常识|置信信息源", "递归至常识性置信信息源"),
                 (r"价值", "价值性信息识别"), (r"values\.md|values_path", "价值清单落盘 values.md")], ctx)

    # ---- E. teach 提示词 ----
    for p, ctx in (("internal/prompt/templates/teach.md", "E-teach"),
                   ("internal/prompt/templates/skills/teach.md", "E-teach-skill")):
        if not exists(p):
            ERRORS.append(f"[{ctx}] 缺失 {p}")
            continue
        c = read(p)
        need(c, [(r"syllabus", "syllabus 任务模式"), (r"lesson", "lesson 任务模式"),
                 (r"syllabus-update|大纲.*变更|变更.*大纲", "syllabus-update（动态大纲）模式"),
                 (r"assess|考核|教练", "assess（考核演练）模式"),
                 (r"原始问题", "还原作者：原始问题"),
                 (r"价值判断", "还原作者：价值判断"),
                 (r"知识领域", "还原作者：知识领域"),
                 (r"主要矛盾", "还原作者：主要矛盾"),
                 (r"辩证逆转", "还原作者：辩证逆转"),
                 (r"学生自身|学生.*知识体系|知识体系", "教学良质自检：结合学生自身知识体系"),
                 (r"无法理解|不可理解", "教学良质自检：无法理解的信息"),
                 (r"写作要求|不必固定|形式自由|自由叙事", "写作要求（非固定模板）")], ctx)
    tskill = read("internal/prompt/templates/skills/teach.md") if exists("internal/prompt/templates/skills/teach.md") else ""
    need(tskill, [(r"教练", "assess：teach 作为教练"), (r"学生", "assess：teach 作为学生"),
                  (r"老师|教师|讲给", "assess：human 作为老师")], "E-assess")

    # ---- F. parent 提示词 ----
    if not exists("internal/prompt/templates/sense_loop.md"):
        ERRORS.append("[F] 缺失 sense_loop.md")
    else:
        sl = read("internal/prompt/templates/sense_loop.md")
        need(sl, [(r"values\.md|values_path", "价值裁决回合"),
                  (r"syllabus", "教学大纲回合"),
                  (r"changelog|变更记录", "大纲动态变更留痕"),
                  (r"assess|考核", "考核演练回合（终止）"),
                  (r"sense_skill_path|skill_sense\.md", "引用 SENSE canon"),
                  (r"原文|正文", "正文原文入会话")], "F")

    # ---- G. 代码接线 ----
    hl = read("internal/prompt/human_loop_prompt.go") if exists("internal/prompt/human_loop_prompt.go") else ""
    need(hl, [(r"values_path", "注入 values_path"), (r"sense_skill_path", "注入 sense_skill_path")], "G")
    pb = read("internal/builder/pibuilder.go")
    if re.search(r"SENSE 方法（5 阶段）|允许反向回流", pb):
        ERRORS.append("[G] pibuilder.go 仍含旧 human-loop SOP（SENSE 方法（5 阶段）/反向回流）")
    ver = re.search(r'VERSION\s*=\s*"([0-9.]+)"', read("cmd/rick/main.go"))
    if not ver:
        ERRORS.append("[G] 未见 VERSION 常量")
    elif [int(x) for x in ver.group(1).split(".")] < [5, 0, 15]:
        ERRORS.append(f"[G] VERSION 未提升（当前 {ver.group(1)}，要求 >= 5.0.15）")

    # ---- H. 渲染契约 ----
    binp = os.path.join(tempfile.gettempdir(), f"rick-gate4-{os.getpid()}")
    r = subprocess.run(["go", "build", "-o", binp, "./cmd/rick"], cwd=ROOT,
                       capture_output=True, text=True, timeout=900)
    if r.returncode != 0:
        ERRORS.append("[H] 构建 gate4 自检二进制失败")
    else:
        with tempfile.TemporaryDirectory(prefix="rick-gate4-ws-") as ws:
            os.makedirs(os.path.join(ws, ".rick"), exist_ok=True)
            env = os.environ.copy()
            env["HOME"] = ws
            r2 = subprocess.run([binp, "human-loop", "--dry-run", "gate4 自检主题"],
                                cwd=ws, capture_output=True, text=True, timeout=300, env=env)
            out = r2.stdout + r2.stderr
            if r2.returncode != 0:
                ERRORS.append(f"[H] dry-run 退出码 {r2.returncode}：\n{out[-1200:]}")
            else:
                for pat, label in ((r"values\.md", "values.md 路径"), (r"syllabus", "syllabus 路径"),
                                   (r"skill_sense|sense_skill", "SENSE canon 路径"),
                                   (r"事实还原", "五步名（事实还原）"), (r"良质批判", "五步名（良质批判）")):
                    if not re.search(pat, out, re.S):
                        ERRORS.append(f"[H] 渲染后的主提示词缺 {label}")
                for bad in ("反向回流", "系统论描述符", "三维打分"):
                    if bad in out:
                        ERRORS.append(f"[H] 渲染后的主提示词仍含 {bad}")
        try:
            os.remove(binp)
        except OSError:
            pass

    # ---- I. dream 兼容 ----
    if exists("internal/prompt/templates/dream.md"):
        if not re.search(r"sense_skill_path", read("internal/prompt/templates/dream.md")):
            ERRORS.append("[I] dream.md 未再引用 sense_skill_path（共用 canon 断链）")
    else:
        ERRORS.append("[I] dream.md 缺失")

    # ---- J. 生产零触碰 ----
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
            ERRORS.append(f"[J][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[J][RSI] 未能定位生产仓库（worktree list 解析失败）")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
