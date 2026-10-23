#!/usr/bin/env python3
"""research.py — 检索指引（第四类 env 资产：检索 skill）。

**定位：产出「检索指引」，不执行检索。** 真正的检索动作由 agent 自带的 web_search
工具完成；本脚本的价值是把 human 定义的「检索三要素」按 agent 角色固化下来，
让 agent 复用前人认知、缩小探索空间——语义空间进化容易困在局部最优（同一格
以小角度反复微调出不来），检索是唯一能注入外部新原理的通道。

检索三要素（每个角色一组）：
  1. **搜索方法**（method）  —— 怎么搜（文档站内 / GitHub 代码搜索 / arXiv）
  2. **搜索信源**（sources） —— 去哪搜（本域最可信来源清单）
  3. **关注点**（focus）     —— 关注什么；与特征格绑定（cell 入参时注入）

角色分工（human 裁决）：
  - coding / debug  → 使用手册类：官方文档、代码样例、已知坑（不查论文）
  - builder         → 论文类：实验细节 / 数据 / 方法（benchmark 配置、规模影响）
  - adjust          → 论文类：创新点 / 研究趋势（尚未覆盖的方法族，供 expand_cell）

输出协议：stdout 单行 JSON
    {"role": "...", "cell": "...|null", "method": "...", "sources": ["..."],
     "focus": "...", "suggested_queries": ["..."], "discipline": "..."}
错误路径（缺/未知 role）：合法 JSON 带 "error" 字段，退出码 0（不阻断 agent）。

env.json 覆盖：`{"research": {"<role>": {"method","sources","focus",
"discipline","query_template","extra_queries"}}}`——按角色字段级覆盖默认值
（gen_env 引导 human 为本域定制）。

用法：
    python3 skills/research.py --role builder [--cell "<cell_id>"] [--env-dir <env>]
"""
import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_ENV_DIR = os.path.dirname(HERE)

ROLES = ("adjust", "builder", "coding", "debug")

DOC_DISCIPLINE = ("≤3 个来源（官方文档/代码样例/FAQ）；必须落到具体 API 用法与限制"
                  "（参数约束/已知坑）；记录「来源 + 要点」到实现注释或 notes")
PAPER_DISCIPLINE = ("≤3 篇论文；必须落到可实现机制；记录 arxiv_id / title / "
                    "insight（可实现机制）/ applicability（本格怎么用）")

# 四角色默认三元组（matmul 域；env.json research.<role> 可字段级覆盖）。
ROLE_PROFILES = {
    "coding": {
        "method": "产品文档与代码库检索（官方文档站内搜索 + GitHub 代码搜索）",
        "sources": ["numpy 官方文档", "BLAS/Accelerate 文档",
                    "GitHub 优秀实现", "Stack Overflow"],
        "focus": "该格实现细节：API 用法与限制（np.dot out= 的行切片约束 / "
                 "dtype 转换成本 / 已知坑）",
        "discipline": DOC_DISCIPLINE,
        "query_template": "{strategy_kw} numpy documentation out parameter constraints",
        "extra_queries": [
            "numpy dot out parameter row slice constraints",
            "{layout_kw} contiguous memory layout numpy performance",
        ],
    },
    "debug": {
        "method": "产品文档与代码样例检索（官方文档站内搜索 + GitHub issue/代码搜索）",
        "sources": ["numpy 官方文档", "GitHub issues/源实现",
                    "BLAS/Accelerate 文档", "Stack Overflow"],
        "focus": "该格失败模式的已知根因与正确用法：报错语义 / API 参数边界 / "
                 "dtype 与内存布局陷阱",
        "discipline": DOC_DISCIPLINE,
        "query_template": "{strategy_kw} numpy error correct usage documentation",
        "extra_queries": [
            "numpy dot out parameter invalid row slice fix",
            "{layout_kw} numpy memory layout pitfall",
        ],
    },
    "builder": {
        "method": "论文检索（arXiv 站内检索 / Google Scholar）",
        "sources": ["arxiv.org", "paperswithcode.com"],
        "focus": "实验细节与数据：benchmark 配置 / 规模影响 / 硬件差异对比 / 可复现方法",
        "discipline": PAPER_DISCIPLINE,
        "query_template": "{strategy_kw} {layout_kw} GEMM performance benchmark",
        "extra_queries": [
            "{strategy_kw} matrix multiplication implementation paper",
            "GEMM optimization benchmark methodology arxiv CPU",
        ],
    },
    "adjust": {
        "method": "论文检索（arXiv / OpenReview）",
        "sources": ["arxiv.org", "openreview.net"],
        "focus": "创新点与研究趋势：尚未覆盖的方法族 / 近期新方向（用于 expand_cell 判断）",
        "discipline": PAPER_DISCIPLINE,
        "query_template": "{strategy_kw} matrix multiplication novel approach survey",
        "extra_queries": [
            "GEMM optimization new methods 2024 2025 arxiv",
            "matrix multiplication algorithm family taxonomy",
        ],
    },
}

# 检索关键词词表（特征轴取值 → 英文检索词；未收录的取值回落为下划线转空格）。
LEXICON = {
    "impl_strategy": {
        "blocked_loops": "blocked loops cache blocking",
        "rowwise_vectorized": "row-wise vectorized dot",
        "full_blas_dispatch": "full BLAS dispatch sgemm",
    },
    "memory_layout": {
        "standard": "standard C order row-major",
        "fused_transpose": "fused transpose memory layout",
    },
}

MAX_QUERIES = 4


def _emit(payload):
    """输出单行 JSON（stdout 最后一行，调用方协议）。"""
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()


def _parse_cell(cell):
    """cell_id → descriptor dict（形如 'a=v1|b=v2'；空/坏串 → {}）。"""
    desc = {}
    if not isinstance(cell, str) or not cell:
        return desc
    for part in cell.split("|"):
        part = part.strip()
        if not part or "=" not in part:
            continue
        key, _, value = part.partition("=")
        key, value = key.strip(), value.strip()
        if key:
            desc[key] = value
    return desc


def _kw(axis, value):
    """轴取值 → 检索关键词（词表优先；未知回落）。"""
    if not value:
        return ""
    lexicon = LEXICON.get(axis) or {}
    return lexicon.get(value, value.replace("_", " "))


def _load_env_config(env_dir):
    """读 env.json（缺文件/坏 JSON → {}，不阻断）。"""
    try:
        with open(os.path.join(env_dir, "env.json"), encoding="utf-8") as f:
            cfg = json.load(f)
        return cfg if isinstance(cfg, dict) else {}
    except (OSError, ValueError):
        return {}


def _role_override(cfg, role):
    """env.json research.<role> 覆盖（字段级；类型不合法忽略）。"""
    research = cfg.get("research")
    if not isinstance(research, dict):
        return {}
    ov = research.get(role)
    if not isinstance(ov, dict):
        return {}
    out = {}
    for key in ("method", "focus", "discipline", "query_template"):
        v = ov.get(key)
        if isinstance(v, str) and v.strip():
            out[key] = v.strip()
    sources = ov.get("sources")
    if isinstance(sources, list):
        cleaned = [s for s in sources if isinstance(s, str) and s.strip()]
        if cleaned:
            out["sources"] = cleaned
    extra = ov.get("extra_queries")
    if isinstance(extra, list):
        cleaned = [q for q in extra if isinstance(q, str) and q.strip()]
        if cleaned:
            out["extra_queries"] = cleaned
    return out


def _build_queries(profile, descriptor):
    """按 role 模板 + 格描述组合检索词（去空去重，≤MAX_QUERIES）。"""
    parts = {
        "strategy_kw": _kw("impl_strategy", descriptor.get("impl_strategy")),
        "layout_kw": _kw("memory_layout", descriptor.get("memory_layout")),
    }
    raw = [profile.get("query_template") or ""]
    raw.extend(profile.get("extra_queries") or [])
    out = []
    for tpl in raw:
        if not isinstance(tpl, str) or not tpl.strip():
            continue
        try:
            q = tpl.format(**parts)
        except (KeyError, IndexError, ValueError):
            q = tpl
        q = " ".join(q.split())
        if q and q not in out:
            out.append(q)
    return out[:MAX_QUERIES]


def _focus_with_cell(base_focus, cell):
    """把关注点绑到特征格（cell 入参时注入；规格要求 focus 含格信息）。"""
    if not cell:
        return base_focus
    return "%s（目标格：%s）" % (base_focus, cell)


def build_payload(role, cell, env_dir):
    """组装检索指引 payload（不执行任何网络请求）。"""
    profile = dict(ROLE_PROFILES[role])
    cfg = _load_env_config(env_dir)
    profile.update(_role_override(cfg, role))
    descriptor = _parse_cell(cell)
    return {
        "role": role,
        "cell": cell or None,
        "method": profile["method"],
        "sources": list(profile["sources"]),
        "focus": _focus_with_cell(profile["focus"], cell),
        "suggested_queries": _build_queries(profile, descriptor),
        "discipline": profile["discipline"],
    }


def main():
    parser = argparse.ArgumentParser(
        description="检索指引（三要素：方法/信源/关注点；不执行检索）")
    parser.add_argument("--role", default=None,
                        help="agent 角色：%s" % "/".join(ROLES))
    parser.add_argument("--cell", default=None,
                        help="目标特征格（形如 'impl_strategy=blocked_loops|memory_layout=standard'）")
    parser.add_argument("--env-dir", default=None,
                        help="env 目录（默认取脚本上一级；用于读取 research.<role> 覆盖）")
    args = parser.parse_args()

    role = (args.role or "").strip()
    if not role:
        # 缺 role：合法 JSON + 退出码 0（不阻断 agent）
        _emit({"error": "missing --role",
               "available_roles": list(ROLES),
               "usage": "python3 skills/research.py --role <role> [--cell <cell_id>] [--env-dir <env>]"})
        sys.exit(0)
    if role not in ROLE_PROFILES:
        _emit({"role": role, "error": "unknown role: %s" % role,
               "available_roles": list(ROLES)})
        sys.exit(0)

    env_dir = os.path.abspath(args.env_dir) if args.env_dir else DEFAULT_ENV_DIR
    try:
        payload = build_payload(role, (args.cell or "").strip(), env_dir)
    except Exception as e:  # 检索指引生成失败也不阻断（合法 JSON）
        payload = {"role": role, "cell": args.cell or None,
                   "error": "research payload build failed: %s" % e}
    _emit(payload)
    sys.exit(0)


if __name__ == "__main__":
    main()
