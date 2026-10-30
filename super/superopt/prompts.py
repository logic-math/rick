"""prompts.py — agent prompt 预渲染层（job_47/task6）

职责：把「角色模板 + env 上下文 + 格/精英元信息 + 实验计划」渲染成每迭代的
agent 系统提示词，落盘 `codes/{uuid}/prompt_{role}.md`（调度器派发时给
`pi --append-system-prompt`）。

模板查找链（openevolve TemplateManager 目录覆盖机制，优先级从高到低）：
  1. `<ws>/prompts/<role>.md`      工作区级覆盖（用户自定义）
  2. `<ws>/env/prompts/<role>.md`  env 级覆盖（gen_env 产出可随 env 携带）
  3. `super/prompts/<role>.md`     仓内默认模板（经 __file__ 定位，不依赖 cwd）

占位符语义（宽松替换）：
  - `{{key}}` 为占位符；key 在 ctx 中且值非 None → 替换；
    key 缺失或值为 None → **保留原样**（缺占位数据不炸，调用方负责补全）。
  - 值格式化：str 原样；dict/list/bool/数字 → json.dumps（ensure_ascii=False）。
  - 不用 str.format——模板中的 JSON 示例大括号会与之冲突。

ctx 键约定（调度器 loop 组装，设计树 §3.4/§4.3）：
  task / perf_desc / perf_type           任务名与性能方向（env.json）
  feature_axes                           特征空间轴定义
  iteration_id                           当前迭代号
  cell_id / cell_descriptor / cell_elite / cell_visit / cell_weight
                                         目标格元信息（builder）
  neighbor_elites / history_failures     邻格精英（跨格灵感）/ 历史失败摘要（builder）
  cells_status / recent_events           格子现状 / 近期事件摘要（adjust）
  experiment_plan                        实验计划（coding/debug 输入）
  workdir / env_dir / self_check_cmd     工作目录 / env 目录 / 自检命令（coding/debug）
  attempt / max_attempts                 修复尝试序号与上限（debug）
  failure_stage / stderr_tail / self_check_result
                                         失败上下文（debug）

stdlib-only，Python >=3.9 兼容（禁 PEP604 联合类型注解）。
"""
import json
import os
import re
from typing import Any, Dict, List, Optional

__all__ = [
    "ROLES",
    "DEFAULT_TEMPLATE_DIR",
    "template_candidates",
    "find_template",
    "extract_placeholders",
    "apply_placeholders",
    "render",
    "write_prompt",
]

# 四个默认角色（模板查找不限于这四个——env/工作区可自带新角色模板）。
ROLES = ("adjust", "builder", "coding", "debug")

# 仓内默认模板目录：super/prompts/（模块位置推导，不依赖 cwd）。
_SUPEROPT_DIR = os.path.dirname(os.path.abspath(__file__))
_SUPER_DIR = os.path.dirname(_SUPEROPT_DIR)
DEFAULT_TEMPLATE_DIR = os.path.join(_SUPER_DIR, "prompts")

# 占位符：{{key}}，键名 [A-Za-z_][A-Za-z0-9_]*，两侧允许可选空白。
_PLACEHOLDER_RE = re.compile(r"\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}")

# 角色名规范（防路径注入：角色名直接拼进文件路径）。
_ROLE_NAME_RE = re.compile(r"^[a-z][a-z0-9_]*$")


def _validate_role(role: str) -> None:
    """角色名必须匹配 ^[a-z][a-z0-9_]*$（否则拼路径有注入风险）。"""
    if not isinstance(role, str) or not _ROLE_NAME_RE.match(role):
        raise ValueError(
            "role 必须匹配 ^[a-z][a-z0-9_]*$（小写字母开头，仅小写/数字/下划线），"
            "got %r" % (role,))


def template_candidates(role: str, template_dir: str) -> List[str]:
    """三级查找链的路径列表（优先级从高到低）。

    template_dir = 工作区根目录（<ws>）：查找 <ws>/prompts 与 <ws>/env/prompts，
    最后落到仓内默认 super/prompts。
    """
    _validate_role(role)
    return [
        os.path.join(template_dir, "prompts", role + ".md"),
        os.path.join(template_dir, "env", "prompts", role + ".md"),
        os.path.join(DEFAULT_TEMPLATE_DIR, role + ".md"),
    ]


def find_template(role: str, template_dir: str) -> str:
    """返回第一个存在的模板路径；三级链全空 → FileNotFoundError（列出全部路径）。"""
    candidates = template_candidates(role, template_dir)
    for path in candidates:
        if os.path.isfile(path):
            return path
    raise FileNotFoundError(
        "角色 %r 的模板在三级查找链中均不存在（依次检查过）：\n  %s"
        % (role, "\n  ".join(candidates)))


def extract_placeholders(text: str) -> List[str]:
    """提取模板中的占位符键名（首次出现顺序，去重）。"""
    keys = []  # type: List[str]
    for m in _PLACEHOLDER_RE.finditer(text):
        key = m.group(1)
        if key not in keys:
            keys.append(key)
    return keys


def _format_value(value: Any) -> str:
    """占位符值的渲染：str 原样；其余（dict/list/bool/数字）JSON 化。"""
    if isinstance(value, str):
        return value
    return json.dumps(value, ensure_ascii=False)


def apply_placeholders(template: str, ctx: Optional[Dict[str, Any]]) -> str:
    """宽松替换：key 在 ctx 且值非 None → 替换；否则保留 {{key}} 原样。

    自定义替换而非 str.format——模板中的 JSON 示例大括号不参与替换。
    """
    ctx = ctx or {}

    def _sub(m):
        key = m.group(1)
        if key in ctx and ctx[key] is not None:
            return _format_value(ctx[key])
        return m.group(0)

    return _PLACEHOLDER_RE.sub(_sub, template)


def render(role: str, template_dir: str, ctx: Optional[Dict[str, Any]]) -> str:
    """渲染角色 prompt：查找链定位模板 → 宽松占位符替换。

    - template_dir：工作区根目录（<ws>），见 template_candidates。
    - ctx：占位符数据（见模块 docstring 的键约定）；None 视为空。
    - 返回渲染后的 prompt 全文（不落盘——落盘用 write_prompt）。
    """
    path = find_template(role, template_dir)
    with open(path, "r", encoding="utf-8") as f:
        template = f.read()
    return apply_placeholders(template, ctx)


def write_prompt(workdir: str, role: str, content: str) -> str:
    """落盘 `codes/{uuid}/prompt_{role}.md`（workdir 即 codes/{uuid} 目录）。

    - 目录不存在则创建（幂等）；同路径重复写 = 覆写。
    - 返回写入的绝对文件路径（调用方直接传给 pi --append-system-prompt）。
    """
    _validate_role(role)
    os.makedirs(workdir, exist_ok=True)
    path = os.path.join(workdir, "prompt_%s.md" % role)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    return path
