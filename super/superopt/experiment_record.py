"""experiment_record.py — 实验记录（KR1 / 迭代8：跨 agent 恒常上下文；job_56 task4：四节结构）

一个「实验」= 在一个特征格上的一次迭代。实验记录是这次迭代的**完整事实记录**（全部事实：设计/过程/结论/反思）：
  - builder 建（frontmatter + 四节：实验设计[背景/假设/验证逻辑/步骤/观测预期/风险预演]占位主体）
  - coding 补（第 2 节执行过程：实现与自验；第 3 节实验结论）
  - debug 补（第 4 节反思：根因/教训/信息增量；失败时补全全部四节）
  - adjust 全局实验分析读「区间内全部实验记录」（recent）

落盘位置：`<iter_dir>/experiment_record.md`，iter_dir = `<ws>/codes/<experiment_id>/`
（experiment_id = `<iteration>-<YYYYMMDDHHMM>`，job_56 task1 新编号 = 工作目录名）。

设计契约：
  - **恒常上下文**：三个执行 agent 的 ctx 都注入本文件（read/digest），保证「实验意图」不丢
  - **四节结构**（job_56 task4 KR-C）：实验设计 / 执行过程 / 实验结论 / 反思——
    完整事实记录，格式过 pi hook 完整性检查（四节标题校验）
  - **frontmatter**：experiment_id / cell_id / hypothesis / status / score（机器可读）
  - **幂等追加**：同段名再次写入 = 替换该段内容（agent 重试/多轮修复不产生重复段）
  - **stdlib-only，py ≥3.9**（禁 PEP604 联合类型注解）

对外 API：
    path_for(iter_dir) -> str
    create(iter_dir, meta: dict) -> str           # 四节模板；返回落盘路径
    append_section(iter_dir, title, body) -> bool # 幂等（同名段替换）；False=无可写文件
    read(iter_dir) -> str                         # 缺文件 → ""
    digest(iter_dir, limit=1200) -> str           # 摘要（第 1 节 + 第 3 节）
    recent(ws, since_iter, until_iter, limit_chars=4000) -> list[dict]
"""
import os
import re
from typing import Any, Dict, List, Optional

RECORD_NAME = "experiment_record.md"

# 四节标准标题（顺序即模板顺序；job_56 task4：完整事实记录——设计/过程/结论/反思）
SECTION_TITLES = (
    "1. 实验设计",
    "2. 执行过程",
    "3. 实验结论",
    "4. 反思",
)

_PLACEHOLDER = "（待 %s 填写）"

# iter 目录名：`<N>-<YYYYMMDDHHMM>`（job_56 task1 新编号 = 工作目录名）；
# 兼容旧形态 `iter{N}` / `iter{N}-{uuid}`（历史工作区归档直读）
_ITER_DIR_RE = re.compile(r"^(?:iter)?(\d+)(?:-|$)")


def path_for(iter_dir: str) -> str:
    """实验记录路径：<iter_dir>/experiment_record.md。"""
    return os.path.join(iter_dir, RECORD_NAME)


def _frontmatter_block(meta: Dict[str, Any]) -> str:
    """frontmatter（机器可读：experiment_id/cell_id/hypothesis/status/score）。"""
    def _v(key: str) -> str:
        v = meta.get(key)
        if v is None or v == "":
            return ""
        return str(v).replace("\n", " ").replace('"', "'")

    def _kv(key: str) -> str:
        v = _v(key)
        return "%s: \"%s\"" % (key, v) if v else "%s: " % key

    return "\n".join([
        "---",
        _kv("experiment_id"),
        _kv("cell_id"),
        _kv("hypothesis"),
        "status: running",
        "score: ",
        "---",
    ])


def _meta_block(meta: Dict[str, Any]) -> str:
    it = meta.get("iteration_id")
    cell = meta.get("cell_id") or ""
    lines = ["- 迭代：%s" % (it if it is not None else "?"),
             "- 实验 ID：%s" % (meta.get("experiment_id") or "?")]
    if cell:
        lines.append("- 特征格：%s" % cell)
    parents = meta.get("parents")
    if isinstance(parents, (list, tuple)) and parents:
        lines.append("- 父本：%s" % ", ".join(str(p) for p in parents))
    return "\n".join(lines)


def create(iter_dir: str, meta: Dict[str, Any]) -> str:
    """按四节模板创建实验记录（builder 建）。返回落盘路径。

    meta 约定：{iteration_id, experiment_id, cell_id, idea, hypothesis, parents}
      - idea 写进第 1 节「实验设计」（背景小节：为什么做这个实验）
      - hypothesis 写进第 1 节「实验假设」小节 + frontmatter
      - 第 2/3/4 节留占位（coding 补 2/3，debug 补 4）
    已存在 → 覆盖重建（新迭代目录天然不冲突；重试场景以最新意图为准）。
    """
    if not isinstance(meta, dict):
        raise ValueError("meta 必须是 dict，收到 %r" % (meta,))
    it = meta.get("iteration_id")
    cell = meta.get("cell_id") or ""
    exp = meta.get("experiment_id") or ""
    title = "# 实验记录 %s · %s" % (exp or ("iter%s" % it if it is not None else "?"),
                                 cell or "?")
    idea = str(meta.get("idea") or "").strip() or _PLACEHOLDER % "builder"
    hyp = str(meta.get("hypothesis") or "").strip()
    if not hyp:
        hyp = _PLACEHOLDER % "builder"
    # 第 1 节「实验设计」的完整子结构（设计即事实：背景/假设/验证逻辑/步骤/观测/风险预演）
    design_body = "\n".join([
        "### 背景（为什么现在做这个实验：来自现状轨迹/台账/research 的证据）",
        idea,
        "",
        "### 实验假设（一句话，可证伪）",
        hyp,
        "",
        "### 验证逻辑（什么结果支持假设、什么结果否定它）",
        _PLACEHOLDER % "builder",
        "",
        "### 具体实验步骤（可复现：改哪、怎么改、跑什么命令）",
        _PLACEHOLDER % "builder",
        "",
        "### 观测预期（预期指标与量级，与 gate_defs 同尺度）",
        _PLACEHOLDER % "builder",
        "",
        "### 仿真预演（风险点推演：执行过程可能遇到的风险 + 每个风险的预防/解决措施）",
        _PLACEHOLDER % "builder",
        "",
        "### 证伪性论证（无论实验成功还是失败，各能获得什么信息增量）",
        _PLACEHOLDER % "builder",
    ])
    parts = [
        title,
        "",
        _frontmatter_block(dict(meta, hypothesis=hyp
                                if hyp != _PLACEHOLDER % "builder" else "")),
        "",
        _meta_block(meta),
        "",
        "## %s" % SECTION_TITLES[0],
        "",
        design_body,
        "",
    ]
    fillers = ("coding", "coding/debug", "debug")
    for i, sec in enumerate(SECTION_TITLES[1:], start=1):
        parts.append("## %s" % sec)
        parts.append("")
        parts.append(_PLACEHOLDER % fillers[i - 1])
        parts.append("")
    os.makedirs(iter_dir, exist_ok=True)
    path = path_for(iter_dir)
    with open(path, "w", encoding="utf-8") as f:
        f.write("\n".join(parts).rstrip() + "\n")
    return path


def read(iter_dir: str) -> str:
    """读实验记录全文；缺文件 → ""（早迭代无记录属正常）。"""
    path = path_for(iter_dir)
    if not os.path.isfile(path):
        return ""
    with open(path, encoding="utf-8") as f:
        return f.read()


def _split_sections(text: str):
    """→ (header_lines, [(title, body)])；标题行形如 `## X`。"""
    header: List[str] = []
    sections: List[Any] = []
    cur_title = None
    cur_body: List[str] = []
    for line in text.splitlines():
        if line.startswith("## "):
            if cur_title is not None:
                sections.append((cur_title, "\n".join(cur_body).strip()))
            cur_title = line[3:].strip()
            cur_body = []
        elif cur_title is None:
            header.append(line)
        else:
            cur_body.append(line)
    if cur_title is not None:
        sections.append((cur_title, "\n".join(cur_body).strip()))
    return header, sections


def append_section(iter_dir: str, title: str, body: str) -> bool:
    """幂等追加段：同名段（`## {title}`）存在 → 替换其内容；否则文末追加。

    title 允许带/不带序号前缀（按原样匹配标题文本）。返回 False 表示没有记录文件
    （调用方应先 ensure/create）。
    """
    path = path_for(iter_dir)
    if not os.path.isfile(path):
        return False
    text = read(iter_dir)
    header, sections = _split_sections(text)
    body = (body or "").strip()
    replaced = False
    out_sections = []
    for sec_title, sec_body in sections:
        if sec_title == title or sec_title.startswith(title + " ") \
                or sec_title == title.rstrip():
            out_sections.append((sec_title, body))
            replaced = True
        else:
            out_sections.append((sec_title, sec_body))
    if not replaced:
        out_sections.append((title, body))
    parts = [ln for ln in header]
    while parts and not parts[-1].strip():
        parts.pop()
    parts.append("")
    for sec_title, sec_body in out_sections:
        parts.append("## %s" % sec_title)
        parts.append("")
        if sec_body:
            parts.append(sec_body)
        parts.append("")
    with open(path, "w", encoding="utf-8") as f:
        f.write("\n".join(parts).rstrip() + "\n")
    return True


def digest(iter_dir: str, limit: int = 1200) -> str:
    """供 ctx 注入的摘要：第 1 节（实验设计）+ 第 3 节（实验结论）。

    缺文件 → ""；段缺失跳过；总长截断到 limit（超长在尾部标注省略）。
    """
    text = read(iter_dir)
    if not text:
        return ""
    header, sections = _split_sections(text)
    want = (SECTION_TITLES[0], SECTION_TITLES[2])
    picked = []
    for title, body in sections:
        if title in want:
            picked.append("## %s\n%s" % (title, body))
    if not picked:
        # 旧六段记录（历史工作区）：取前 3 段 + 第 6 段
        legacy = ("1. 背景与价值", "2. 实验假设", "3. 验证逻辑",
                  "6. 执行过程与结论")
        for title, body in sections:
            if title in legacy:
                picked.append("## %s\n%s" % (title, body))
    if not picked:
        picked = [text]
    out = "\n\n".join(picked)
    if limit and len(out) > limit:
        out = out[:limit] + "\n…（截断）"
    return out


def _iter_of(dirname: str) -> Optional[int]:
    m = _ITER_DIR_RE.match(dirname)
    if not m:
        return None
    try:
        return int(m.group(1))
    except ValueError:
        return None


def _cell_of(text: str) -> str:
    """格 id：frontmatter cell_id 优先；旧记录从标题行 `# 实验记录 · {cell_id}` 提取。"""
    fm = re.search(r"^cell_id:\s*\"?([^\"\n]*)\"?", text, re.M)
    if fm and fm.group(1).strip():
        return fm.group(1).strip()
    for line in text.splitlines():
        if line.startswith("# 实验记录"):
            if "·" in line:
                return line.split("·", 1)[1].strip()
            return ""
    return ""


def recent(ws: str, since_iter: int, until_iter: int,
           limit_chars: int = 4000) -> List[Dict[str, Any]]:
    """区间内的全部实验记录摘要（adjust 全局实验分析用）。

    扫描 `<ws>/codes/iter{N}*/`，取 since_iter ≤ N ≤ until_iter 的记录：
      [{iteration_id, cell_id, path, digest}]
    按迭代号升序；累计字符超 limit_chars 时**不丢条目**（区间全景是 adjust 的
    核心输入），按比例压缩每条 digest（下限 200 字符）。
    """
    codes_dir = os.path.join(ws, "codes")
    if not os.path.isdir(codes_dir):
        return []
    found: List[Dict[str, Any]] = []
    for name in sorted(os.listdir(codes_dir)):
        n = _iter_of(name)
        if n is None or n < since_iter or n > until_iter:
            continue
        iter_dir = os.path.join(codes_dir, name)
        path = path_for(iter_dir)
        if not os.path.isfile(path):
            continue
        text = read(iter_dir)
        found.append({
            "iteration_id": n,
            "cell_id": _cell_of(text),
            "path": path,
            "digest": digest(iter_dir, limit=1200),
        })
    if limit_chars and limit_chars > 0 and found:
        total = sum(len(it["digest"]) + len(it["path"]) + 40 for it in found)
        if total > limit_chars:
            # 超预算：**不丢条目**（区间全景是 adjust 的核心输入），按比例压缩每条
            # digest（下限 200 字符，保住「假设 + 结论」的可读性）
            overhead = sum(len(it["path"]) + 40 for it in found)
            per = max(80, (limit_chars - overhead) // len(found))
            for it in found:
                d = os.path.dirname(it["path"])
                it["digest"] = digest(d, limit=per)
    return found
