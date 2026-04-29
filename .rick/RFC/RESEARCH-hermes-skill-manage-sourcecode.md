# Hermes Agent skill_manage 源码深度调研报告

> 调研日期：2026-04-14
> 调研方式：直接读取 /tmp/hermes-agent 源码（git clone --depth=1）
> 核心文件：
> - tools/skill_manager_tool.py（skill_manage 实现）
> - tools/skills_tool.py（skills_list / skill_view 实现）
> - tools/fuzzy_match.py（模糊匹配算法）
> - tools/skills_guard.py（安全扫描）
> - agent/prompt_builder.py（SKILLS_GUIDANCE + 缓存机制）

---

## 一、skill_manage 工具定义

**文件**：`tools/skill_manager_tool.py`（第588-598行）

```python
def skill_manage(
    action: str,
    name: str,
    content: str = None,
    category: str = None,
    file_path: str = None,
    file_content: str = None,
    old_string: str = None,
    new_string: str = None,
    replace_all: bool = False,
) -> str:
```

支持的 actions（第678-679行）：
```python
"enum": ["create", "patch", "edit", "delete", "write_file", "remove_file"]
```

---

## 二、create action 完整实现

函数：`_create_skill(name, content, category=None)`（第292-346行）

### 完整步骤

1. **验证名称**（第295-297行）
   - 正则：`r'^[a-z0-9][a-z0-9._-]*$'`
   - 最长 64 字符

2. **验证分类**（第299-301行）
   - 可选，单层目录名，同名称规则

3. **验证 frontmatter**（第304-306行）
   - 必须以 `---` 开头，有闭合 `---`
   - 必须包含 `name` 和 `description` 字段
   - 使用 `yaml.safe_load()` 解析
   - 必须有非空 body

4. **验证内容大小**（第308-310行）
   - 最大 100,000 字符

5. **检查名称冲突**（第313-318行）
   - 跨所有技能目录搜索（本地 + 外部）

6. **创建目录**（第321-322行）
   - 路径：`~/.hermes/skills/[category]/name/`

7. **原子化写入 SKILL.md**（第325-326行）
   - 临时文件 + `os.replace()` 防写入中断

8. **安全扫描**（第329-332行）
   - 失败则回滚整个目录 `shutil.rmtree(skill_dir)`

9. **清除系统提示缓存**（第639-644行）
   ```python
   clear_skills_system_prompt_cache(clear_snapshot=True)
   ```

---

## 三、patch action 完整实现

函数：`_patch_skill(name, old_string, new_string, file_path=None, replace_all=False)`（第382-467行）

### fuzzy_find_and_replace 算法

源文件：`tools/fuzzy_match.py`（第50-101行）

**9步策略链**（按顺序尝试，命中即停）：

| 步骤 | 策略名 | 说明 |
|------|--------|------|
| 1 | exact | 直接字符串匹配 |
| 2 | line_trimmed | 每行去首尾空白 |
| 3 | whitespace_normalized | 多个空白压缩为单个空格 |
| 4 | indentation_flexible | 忽略缩进差异 |
| 5 | escape_normalized | `\\n` 字面量转换为实际换行 |
| 6 | trimmed_boundary | 仅修剪第一行和最后一行 |
| 7 | unicode_normalized | 智能双引号、破折号等 Unicode 标准化 |
| 8 | block_anchor | 按首尾行锚定，中间部分用相似度匹配（阈值 0.50/0.70） |
| 9 | context_aware | 行级相似度 ≥80%，至少50%行满足 |

### patch 完整步骤

1. 验证输入
2. 查找 skill
3. 确定目标文件（默认 SKILL.md，可指定 file_path）
4. 读取原内容
5. 调用 `fuzzy_find_and_replace()`
6. 检查匹配结果：
   - 失败 → 返回错误 + 文件前500字符预览（供 LLM 自我纠正）
   - 多个匹配且 `replace_all=False` → 报错
7. 验证结果大小和 frontmatter 完整性
8. 原子化写入新内容
9. 安全扫描 + 可选回滚

---

## 四、edit action 完整实现

函数：`_edit_skill(name, content)`（第349-379行）

**与 patch 的核心区别**：

| 维度 | patch | edit |
|------|-------|------|
| 操作方式 | 目标替换（old → new） | 全量重写整个 SKILL.md |
| 匹配算法 | fuzzy_find_and_replace | 无需匹配 |
| Token 消耗 | 低（只传变化部分） | 高（传完整内容） |
| 风险 | 低（只改变化部分） | 高（可能破坏正常部分） |
| 推荐场景 | 日常维护 | 结构性重写 |

**实现关键**（第365-373行）：
```python
original_content = skill_md.read_text(encoding="utf-8") if skill_md.exists() else None
_atomic_write_text(skill_md, content)

scan_error = _security_scan_skill(existing["path"])
if scan_error:
    if original_content is not None:
        _atomic_write_text(skill_md, original_content)  # 回滚
    return {"success": False, "error": scan_error}
```

---

## 五、delete / write_file / remove_file

### delete（第470-487行）
```python
shutil.rmtree(skill_dir)
# 清理空的 category 目录
parent = skill_dir.parent
if parent != SKILLS_DIR and parent.exists() and not any(parent.iterdir()):
    parent.rmdir()
```

### write_file（第490-539行）
- 允许写入的子目录：`references/`、`templates/`、`scripts/`、`assets/`
- 单文件大小限制：1 MiB
- 内容字符限制：100,000 字符
- 写入后安全扫描，失败回滚

### remove_file（第542-581行）
- 验证 file_path 安全性
- 删除后清理空目录
- 失败时列出可用文件供参考

---

## 六、安全扫描机制

源文件：`tools/skills_guard.py`

### 扫描步骤（第595-639行）

1. **结构检查**（第616-624行）
   - 最多 50 个文件
   - 总大小最多 1 MB
   - 单文件最多 256 KB
   - 检测可疑二进制扩展名、符号链接

2. **82+ 威胁模式正则匹配**（第82-484行）
   - 外渗：curl 泄漏密钥、读取 SSH/AWS/GPG 目录
   - 注入：忽略指令、角色劫持、系统提示覆盖
   - 破坏：`rm -rf /`、格式化文件系统
   - 持久化：crontab、shell rc、SSH 后门
   - 网络：反向 shell、隧道服务
   - 代码混淆：base64 管道、eval/exec
   - 供应链：curl 管道 shell、未锚定依赖
   - 提权：sudo、SUID/SGID、NOPASSWD
   - 硬编码密钥：API 密钥、GitHub token

3. **不可见 Unicode 字符检测**（第577-590行）
   - 零宽空间、零宽非连接符等

4. **判定 verdict**："safe" / "caution" / "dangerous"

### 信任政策（第41-49行）

| 来源 | safe | caution | dangerous |
|------|------|---------|-----------|
| builtin | allow | allow | allow |
| trusted | allow | allow | block |
| community | allow | block | block |
| **agent-created** | **allow** | **allow** | **ask** |

agent-created 的 skill：safe/caution 直接允许，dangerous 返回警告但不阻止。

---

## 七、skills_list / skill_view 渐进式加载

源文件：`tools/skills_tool.py`

### skills_list（第633-698行）—— Level 0

```python
def skills_list(category=None, task_id=None) -> str:
    # 返回所有 skill 的 name + description，不含完整内容
```

返回格式：
```json
{
  "success": true,
  "skills": [
    {"name": "axolotl", "description": "Fine-tune LLMs", "category": "ml"}
  ],
  "count": 42,
  "hint": "Use skill_view(name) to see full content"
}
```

### skill_view（第701-1164行）—— Level 1/2

```python
def skill_view(name, file_path=None, task_id=None) -> str:
    # file_path=None：返回完整 SKILL.md（Level 1）
    # file_path="scripts/train.py"：返回具体文件（Level 2）
```

返回完整元数据：
```json
{
  "name": "axolotl",
  "description": "...",
  "content": "# Axolotl\n...",
  "tags": ["fine-tuning", "llm"],
  "linked_files": {
    "references": ["api.md"],
    "scripts": ["train.py"]
  },
  "required_environment_variables": [...],
  "missing_required_environment_variables": ["HUGGINGFACE_TOKEN"],
  "readiness_status": "setup_needed"
}
```

---

## 八、SKILLS_GUIDANCE 完整内容

源文件：`agent/prompt_builder.py`（第164-171行）

```python
SKILLS_GUIDANCE = (
    "After completing a complex task (5+ tool calls), fixing a tricky error, "
    "or discovering a non-trivial workflow, save the approach as a "
    "skill with skill_manage so you can reuse it next time.\n"
    "When using a skill and finding it outdated, incomplete, or wrong, "
    "patch it immediately with skill_manage(action='patch') — don't wait to be asked. "
    "Skills that aren't maintained become liabilities."
)
```

### 技能索引注入 prompt（第775-796行）

```
## Skills (mandatory)
Before replying, scan the skills below. If a skill matches or is even partially relevant
to your task, you MUST load it with skill_view(name) and follow its instructions.
Err on the side of loading — it is always better to have context you don't need
than to miss critical steps, pitfalls, or established workflows.
Skills contain specialized knowledge — API endpoints, tool-specific commands,
and proven workflows that outperform general-purpose approaches. Load the skill
even if you think you could handle the task with basic tools like web_search or terminal.
Skills also encode the user's preferred approach, conventions, and quality standards
for tasks like code review, planning, and testing — load them even for tasks you
already know how to do, because the skill defines how it should be done here.
If a skill has issues, fix it with skill_manage(action='patch').
After difficult/iterative tasks, offer to save as a skill.
If a skill you loaded was missing steps, had wrong commands, or needed
pitfalls you discovered, update it before finishing.

<available_skills>
  mlops: Machine learning operations
    - axolotl: Fine-tune LLMs with Axolotl
    - peft: Parameter-efficient fine-tuning
</available_skills>

Only proceed without loading a skill if genuinely none are relevant to the task.
```

---

## 九、两层缓存机制

源文件：`agent/prompt_builder.py`（第605-702行）

### L1：进程内 LRU 缓存

```python
_SKILLS_PROMPT_CACHE_MAX = 8
_SKILLS_PROMPT_CACHE: OrderedDict[tuple, str] = OrderedDict()
_SKILLS_PROMPT_CACHE_LOCK = threading.Lock()

cache_key = (
    str(skills_dir.resolve()),
    tuple(str(d) for d in external_dirs),
    tuple(sorted(str(t) for t in (available_tools or set()))),
    tuple(sorted(str(ts) for ts in (available_toolsets or set()))),
    _platform_hint,
)
```

### L2：磁盘快照缓存

路径：`~/.hermes/.skills_prompt_snapshot.json`

```json
{
  "version": 1,
  "manifest": {
    "mlops/axolotl/SKILL.md": [mtime_ns, size]
  },
  "skills": [...],
  "category_descriptions": {...}
}
```

快照失效条件：任何 SKILL.md 或 DESCRIPTION.md 的 mtime/size 变化。

### 缓存清除时机

```python
# tools/skill_manager_tool.py 第639-644行
if result.get("success"):
    try:
        from agent.prompt_builder import clear_skills_system_prompt_cache
        clear_skills_system_prompt_cache(clear_snapshot=True)
    except Exception:
        pass
```

**任何成功的 skill_manage 操作后都清除 L1 + L2**，下一次对话重新扫描生效。

---

## 十、存储路径解析

```python
HERMES_HOME = ~/.hermes
SKILLS_DIR  = ~/.hermes/skills/
```

目录结构：
```
~/.hermes/skills/
├── [category]/
│   └── skill-name/
│       ├── SKILL.md
│       ├── scripts/
│       ├── references/
│       ├── templates/
│       └── assets/
└── skill-name/          # 无分类时直接在根目录
    └── SKILL.md
```

外部目录配置（config.yaml）：
```yaml
skills:
  external_dirs:
    - ~/shared-skills
    - /opt/org-skills
  disabled: [deprecated-skill]
```

**优先级**：本地 `~/.hermes/skills/` > 外部目录（同名时本地覆盖）

---

## 十一、完整数据流示例

```
用户完成复杂任务（工具调用 ≥5 次）
    ↓
LLM 自主判断触发条件（SKILLS_GUIDANCE 指令）
    ↓
skill_manage(action="create", name="pdf-table-extract", content="---\n...")
    ↓
_validate_name()        → 名称格式检查
_validate_frontmatter() → YAML 解析，检查 name/description 字段
_validate_content_size()→ ≤100k 字符
_find_skill()           → 检查名称冲突
skill_dir.mkdir()       → ~/.hermes/skills/data-extraction/pdf-table-extract/
_atomic_write_text()    → 临时文件 + os.replace()
_security_scan_skill()  → 82+ 威胁模式 → verdict="safe" → allow
clear_skills_system_prompt_cache(clear_snapshot=True)
    ↓
✓ success: true

下一次对话：
build_skills_system_prompt()
    → L1 miss（已清除）
    → L2 miss（快照已删除）
    → 完整扫描 ~/.hermes/skills/
    → 新 skill 出现在 <available_skills> 中
    → LLM 收到任务时先扫描索引，命中则 skill_view() 加载
```
