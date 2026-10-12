# 依赖关系
task1, task2

# 任务名称
修改 learning.md 模板 - 四类产出规范（Wiki/Skills/OKR/SPEC）

# 任务目标
修改 learning.md 提示词模板，明确 learning 阶段必须按需产出四类知识，全部输出到本 job 的 `learning/` 目录，由 `rick tools merge` 统一 cp 覆盖合并，产生可审查的 git diff：
- **Wiki**（给人类）：系统运行原理与控制方法，让人理解这个 AI 执行系统如何工作、如何控制
- **Skills**（给 AI）：可执行的 Python 技能脚本，让 AI Agent 在后续 doing 阶段直接调用
- **OKR.md**（完整新版本）：如果本次 job 执行中目标发生变化，产出完整的新版 OKR 文件覆盖旧版
- **SPEC.md**（完整新版本）：如果发现需要沉淀到规范中的信息，产出完整的新版 SPEC 文件覆盖旧版

# 关键结果
1. 完成 internal/prompt/templates/learning.md 修改，明确四类产出结构：

   ```
   learning/
   ├── SUMMARY.md       # 执行总结（必需）
   ├── wiki/            # 给人类阅读的知识（按需）
   │   └── *.md
   ├── skills/          # 给 AI 调用的技能脚本（按需）
   │   └── *.py
   ├── OKR.md           # 完整新版 OKR（按需，有目标变化时产出）
   └── SPEC.md          # 完整新版 SPEC（按需，有规范更新时产出）
   ```

   **注意**：OKR.md 和 SPEC.md 是完整版本文件（不是 diff/patch），merge 时直接 cp 覆盖 `.rick/OKR.md` 和 `.rick/SPEC.md`，产生 git diff 供人类审查。

2. 完成 learning.md 中 **Wiki 产出规范**：

   **Wiki 的定位**：
   - 受众：人类（开发者、使用者）
   - 内容：系统运行原理、控制方法、决策逻辑、架构理解
   - 触发条件：本次 job 执行中涉及了系统的某个运行机制，或发现了值得记录的控制模式

   **Wiki 写作要求**：
   - 每篇 wiki 聚焦一个主题（如：DAG 执行原理、重试机制控制、Skills 注入机制）
   - 必须包含：概述、工作原理、如何控制/使用、示例
   - 使用 Mermaid 图表辅助说明运行流程
   - 文件命名：小写下划线（如 `dag_execution.md`、`skills_injection.md`）
   - 输出到 `learning/wiki/`，不直接修改 `.rick/wiki/`

3. 完成 learning.md 中 **Skills 产出规范**：

   **技能形式**：
   - skills 必须是可执行的 Python 脚本（`.py`），不是 Markdown 文档
   - 输出目录：`learning/skills/*.py`
   - 标准格式：文件头 `# Description: 一句话描述`、argparse 支持 `--help`、返回 JSON `{"pass": bool, "result": ..., "errors": []}`、`if __name__ == "__main__"` 入口

   **技能进化流程**（AI 必须按此顺序执行）：
   1. **定义目标**：明确这个技能要解决什么问题，与 OKR 的关联
   2. **GitHub 搜索**：用 WebSearch 搜索 GitHub 上是否有现成的工具/脚本实现该功能，优先复用
   3. **组合评估**：检查 `.rick/skills/` 下的现有脚本，判断能否通过组合现有技能实现目标
   4. **实现决策**：
      - 找到 GitHub 实现 → 适配为标准格式的 .py 脚本
      - 可组合现有技能 → 编写组装脚本，import 或 subprocess 调用现有 skills
      - 需更新现有技能 → 使用同名文件覆盖（merge 时产生 diff）
      - 全新技能 → 创建新的原子化脚本

   **技能质量要求**：
   - 原子化：一个脚本只做一件事
   - 与 OKR 相关：能提升后续任务成功率
   - 可独立调用：`python3 skill.py [args]` 直接运行
   - 有自测：脚本支持 `--test` 参数执行内置验证

   **目录说明**：
   - 无论新建还是更新技能，都在本 job 的 `learning/skills/` 下创建完整文件
   - 同名文件在 merge 时覆盖 `.rick/skills/` 旧版本，产生可审查的 git diff
   - 不直接修改 `.rick/skills/`

4. 完成 learning.md 中 **OKR/SPEC 更新规范**：

   **触发条件**：
   - 产出 `learning/OKR.md`：本次 job 执行中发现目标需要调整（新增目标、修改 KR 指标、删除过时目标）
   - 产出 `learning/SPEC.md`：本次 job 执行中发现需要沉淀到规范的信息（新的技术约束、工程实践、路径规范等）

   **产出要求**：
   - 必须是**完整版本**的 OKR.md / SPEC.md，包含所有内容（不是只写变更部分）
   - 在文件顶部用注释说明本次变更的内容和原因：
     ```markdown
     <!-- 更新说明：基于 job_N 执行经验，新增 KR2.5（xxx），修改 KR3.1（原因：xxx） -->
     ```
   - 格式必须与 `.rick/OKR.md` / `.rick/SPEC.md` 保持一致（merge 时直接覆盖）
   - 不产出 `OKR_UPDATE.md` / `SPEC_UPDATE.md`（旧格式废弃）

5. 完成 learning.md 中 **AI agent 完整工作流程**（核心）：

   learning 阶段启动的是一个**交互式 AI 对话**，AI agent 在对话中自主完成所有工作，包括最后调用 `rick tools merge` 完成合并。完整流程如下：

   ```
   Step 1: 分析执行数据
     - 读取 debug.md（工作日志）
     - 读取 tasks.json（执行状态）
     - 用 git show <commit_hash> 查看每个任务的代码变更

   Step 2: 产出知识（输出到 learning/ 目录）
     - 生成 SUMMARY.md（必需）
     - 按需生成 wiki/*.md（系统原理与控制方法）
     - 按需生成 skills/*.py（可复用技能脚本，遵循技能进化流程）
     - 按需生成 OKR.md（完整新版本）
     - 按需生成 SPEC.md（完整新版本）

   Step 3: 质量检查
     - 运行 `{rick_bin_path} tools --help` 了解所有可用的元技能（rick_bin_path 由 learning 提示词注入）
     - 运行 `{rick_bin_path} tools learning_check job_N` 验证产出质量
     - 如果 check 失败，根据错误信息修复后重新检查

   Step 4: 执行合并
     - 在 SUMMARY.md 第一行添加 `APPROVED: true`
     - 运行 `{rick_bin_path} tools merge job_N`
       → 命令完成后输出变更摘要和分支名 `learning/job_N`

   Step 5: 展示 diff 并进入人类审查循环
     - 运行 `git diff main..learning/job_N` 展示所有变更
     - 向人类呈现 diff 内容，询问是否确认合并

     [人类确认]
     - 确认当前在主分支（git branch --show-current）
     - 运行 `git merge learning/job_N --no-ff -m "merge: integrate job_N learnings"`
     - 运行 `git branch -D learning/job_N` 删除 learning 分支
     - 向人类报告合并完成，结束

     [人类拒绝并提出改进意见]
     - 切换到 `learning/job_N` 分支
     - 根据人类的改进意见修改 `.rick/` 下的对应文件（wiki/skills/OKR.md/SPEC.md）
     - git add + git commit -m "learning: revise per human feedback"
     - 切换回原分支
     - 重新运行 `git diff main..learning/job_N` 展示更新后的 diff
     - 再次请求人类确认
     - 循环直到人类确认为止
   ```

   **关键约束**：
   - 全程在交互式对话中完成，AI agent 自主执行所有 git 和 rick 命令
   - 人类的角色：审查 diff、确认或提出改进意见，可多轮反馈
   - 每轮修改都追加新的 commit 到 `learning/job_N` 分支，历史完整保留
   - `rick tools merge` 只负责创建分支和 cp 覆盖，后续的审查循环由 AI agent 在对话中完成

6. 完成 `internal/cmd/learning.go` 修改：在构建 learning 提示词时注入 `rick_bin_path` 变量，值为 `filepath.Join(projectRoot, "bin", "rick")`（项目本地构建的二进制路径），使 AI agent 在对话中能用正确路径调用 rick tools 命令

7. 添加示例到 learning.md：
   - Wiki 示例：`skills_injection.md`（描述 Skills 注入到 doing 提示词的工作原理）
   - Skill 示例：`check_go_build.py`（原子技能）、`check_task_format.py`（组合/验证类技能）

# 测试方法
1. 检查 internal/prompt/templates/learning.md，验证包含四类产出目录结构（wiki/、skills/、OKR.md、SPEC.md）
2. 检查文件包含 Wiki 的定位说明（受众是人类、内容是系统运行原理与控制方法）
3. 检查文件包含 Wiki 写作要求（概述/工作原理/如何控制/示例 + Mermaid 图表）
4. 检查文件包含技能进化四步流程（定义目标→GitHub搜索→组合评估→实现决策）
5. 检查文件包含 OKR/SPEC 更新规范（完整版本、顶部注释说明变更）
6. 检查文件不再提及 `OKR_UPDATE.md` / `SPEC_UPDATE.md` 旧格式
7. 检查文件包含 AI agent 五步工作流程（分析→产出→check→merge→审查循环）
8. 检查文件明确说明 Step 5 是循环：人类拒绝时提出改进意见 → AI 修改 → 重新 diff → 再次确认
9. 运行 `go build ./...` 验证 embedded 模板编译正常
