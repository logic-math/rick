"""experiment_record.py — 实验记录（KR1 / 迭代8：跨 agent 恒常上下文）

一个「实验」= 在一个特征格上的一次迭代。实验记录是这次迭代的**完整上下文文件**：
  - builder 建（六段：背景与价值 / 实验假设 / 验证逻辑 / 具体实验步骤 / 观测现象 / 执行过程与结论（占位））
  - coding 补（执行过程与结论：实现要点 / TDD 红绿 / 门禁结果）
  - debug 补（卡点 / 根因 / 解决）
  - adjust 全局实验分析读「区间内全部实验记录」（recent）

落盘位置（J-i8-2 裁决）：`<iter_dir>/experiment_record.md`，iter_dir = `<ws>/codes/iter{N}-{uuid>/`。

设计契约：
  - **恒常上下文**：三个执行 agent 的 ctx 都注入本文件（read/digest），保证「实验意图」不丢
  - **幂等追加**：同段名再次写入 = 替换该段内容（agent 重试/多轮修复不产生重复段）
  - **stdlib-only，py ≥3.9**（禁 PEP604 联合类型注解）

对外 API：
    path_for(iter_dir) -> str
    create(iter_dir, meta: dict) -> str           # 六段模板；返回落盘路径
    append_section(iter_dir, title, body) -> bool # 幂等（同名段替换）；False=无可写文件
    read(iter_dir) -> str                         # 缺文件 → ""
    digest(iter_dir, limit=1200) -> str           # 摘要（第 1-3 段 + 结论段）
    recent(ws, since_iter, until_iter, limit_chars=4000) -> list[dict]
"""
import os
import re
from typing import Any, Dict, List, Optional

RECORD_NAME = "experiment_record.md"

# 六段标准标题（顺序即模板顺序；第 6 段由 coding/debug 补）
SECTION_TITLES = (
    "1. 背景与价值",
    "2. 实验假设",
    "3. 验证逻辑",
    "4. 具体实验步骤",
    "5. 观测现象",
    "6. 执行过程与结论",
)

_PLACEHOLDER = "（待 %s 填写）"

# iter 目录名：iter{N}-{uuid}（也兼容 iter{N}）
_ITER_DIR_RE = re.compile(r"^iter(\d+)(?:-|$)")


def path_for(iter_dir: str) -> str:
    """实验记录路径：<iter_dir>/experiment_record.md。"""
    return os.path.join(iter_dir, RECORD_NAME)


def _meta_block(meta: Dict[str, Any]) -> str:
    it = meta.get("iteration_id")
    cell = meta.get("cell_id") or ""
    lines = ["- 迭代：%s" % (it if it is not None else "?")]
    if cell:
        lines.append("- 特征格：%s" % cell)
    parents = meta.get("parents")
    if isinstance(parents, (list, tuple)) and parents:
        lines.append("- 父本：%s" % ", ".join(str(p) for p in parents))
    return "\n".join(lines)


def create(iter_dir: str, meta: Dict[str, Any]) -> str:
    """按六段模板创建实验记录（builder 建）。返回落盘路径。

    meta 约定：{iteration_id, cell_id, idea, parents}
      - idea 写进第 1 段「背景与价值」（为什么做这个实验）
      - 其余段留占位（builder 在实验中填，coding/debug 补第 6 段）
    已存在 → 覆盖重建（新迭代目录天然不冲突；重试场景以最新意图为准）。
    """
    if not isinstance(meta, dict):
        raise ValueError("meta 必须是 dict，收到 %r" % (meta,))
    it = meta.get("iteration_id")
    cell = meta.get("cell_id") or ""
    title = "# 实验记录 iter%s · %s" % (it if it is not None else "?", cell or "?")
    idea = str(meta.get("idea") or "").strip() or "（待 builder 填写）"
    parts = [
        title,
        "",
        _meta_block(meta),
        "",
        "## %s" % SECTION_TITLES[0],
        "",
        "%s" % idea,
        "",
    ]
    fillers = ("builder", "builder", "builder", "builder", "builder", "coding/debug")
    for i, sec in enumerate(SECTION_TITLES[1:], start=1):
        parts.append("## %s" % sec)
        parts.append("")
        parts.append(_PLACEHOLDER % fillers[i])
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
    """供 ctx 注入的摘要：前 3 段（背景/假设/验证逻辑）+ 结论段（执行过程与结论）。

    缺文件 → ""；段缺失跳过；总长截断到 limit（超长在尾部标注省略）。
    """
    text = read(iter_dir)
    if not text:
        return ""
    header, sections = _split_sections(text)
    want = (SECTION_TITLES[0], SECTION_TITLES[1], SECTION_TITLES[2], SECTION_TITLES[5])
    picked = []
    for title, body in sections:
        if title in want:
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
    """从标题行 `# 实验记录 iter{N} · {cell_id}` 提取格 id。"""
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
