## task3: 更新 learning 提示词模板区分 tools 和 skills

**分析过程 (Analysis)**:
- 读取 `internal/prompt/templates/learning.md`，找到需要修改的位置：
  - "按需产出"列表中的 `skills/*.py` → 需拆分为 `tools/*.py` 和 `skills/*.md`
  - "2. Skills 产出规范" → 需拆分为独立的 Tools 规范和 Skills 规范
  - SUMMARY.md 知识沉淀清单中的 `skills/xxx.py` → 需更新
  - 底部"重要约束"第5条 → 需更新
- 方案：原地修改 learning.md，将 Skills 规范拆分为 Tools（.py）和 Skills（.md）两节，编号顺移

**实现步骤 (Implementation)**:
1. 将"按需产出"中 `skills/*.py` 改为 `tools/*.py`（确定性工具脚本）并新增 `skills/*.md`（组合技能说明书）
2. 将"2. Skills 产出规范"改为"2. Tools 产出规范"（.py 脚本，输出到 `tools/`）
3. 新增"3. Skills 产出规范"（.md 说明书，输出到 `skills/`），包含标准格式和质量要求
4. 原 3/4 节编号顺移为 4/5
5. 更新 SUMMARY.md 知识沉淀清单（新增 `tools/xxx.py` 行，`skills/xxx.py` → `skills/xxx.md`）
6. 更新底部"重要约束"第5条，新增第6条

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`grep -n "skills/\*.py" internal/prompt/templates/learning.md; echo "exit: $?"`
- 测试输出：`exit: 1`（无匹配，✅）
- 测试命令：`grep -c "tools/\*.py" internal/prompt/templates/learning.md && grep -c "skills/\*.md" internal/prompt/templates/learning.md`
- 测试输出：`2` / `2`（均存在，✅）
- 测试命令：`go build ./...`
- 测试输出：无错误，exit 0（✅）
- 结论：✅ 通过

---

## task1: 迁移 `.rick/skills/*.py` 到 `tools/` 目录

**分析过程 (Analysis)**:
- 检查 `.rick/skills/` 目录，发现 5 个 `.py` 文件：`build_and_get_rick_bin.py`、`check_go_build.py`、`check_prompt_variables.py`、`check_variadic_api.py`、`mock_agent_testing.py`
- `rick_tools_check_pattern.py` 在 job_11 的 learning 阶段已删除，无需再次处理
- `tools/` 目录已存在（空目录），由框架预先创建
- `internal/workspace/tools.go` 的 `LoadToolsList` 扫描 `{projectRoot}/tools/*.py`，所有脚本均有 `# Description:` 首行注释
- `internal/workspace/skills.go` 的 `LoadSkillsIndex` 优先读取 `index.md` 内容注入 skills section
- 需要将 `index.md` 从"Python 脚本列表"重构为"Markdown 技能说明书索引"

**实现步骤 (Implementation)**:
1. 将 5 个 `.py` 文件从 `.rick/skills/` 复制到 `tools/`
2. 从 `.rick/skills/` 删除所有 `.py` 文件（使用 `rm -f`）
3. 重写 `.rick/skills/index.md`：改为 Markdown skill 索引，触发场景列非空，调用方式指向 `tools/`

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`ls tools/ && ls .rick/skills/*.py 2>/dev/null || echo "no py files"`
- 测试输出：
  ```
  build_and_get_rick_bin.py
  check_go_build.py
  check_prompt_variables.py
  check_variadic_api.py
  mock_agent_testing.py
  no py files
  ```
- 测试命令：`python3 tools/build_and_get_rick_bin.py`
- 测试输出：
  ```
  {"pass": true, "bin_path": "/Users/sunquan/ai_coding/CODING/rick/bin/rick", "errors": []}
  ```
- 测试命令：`/Users/sunquan/ai_coding/CODING/rick/bin/rick doing job_12 --dry-run | grep -A 20 "可用的项目 Tools\|可用的项目 Skills"`
- 测试输出：tools section 列出 5 个工具，skills section 显示 4 个 Markdown skill 及触发场景
- 结论：✅ 通过
