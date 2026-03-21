# 依赖关系
task3, task4

# 任务名称
实现 rick tools merge 和 skills 注入到 doing 提示词

# 任务目标
实现两个紧密相关的功能：
1. `rick tools merge job_N`：**由 learning 阶段的 AI agent 在交互对话中自主调用**，将 learning 产出通过 git branch + cp 覆盖合并到主上下文，完成后向 AI agent 输出结构化摘要和 git 操作指引，AI agent 再将这些信息转述给人类
2. doing 阶段自动加载 `.rick/skills/` 下的所有 Python 技能脚本，注入到提示词中，使 AI Agent 可感知并在合适时机调用

**设计原则**：`rick tools` 的所有子命令都是给 AI agent 使用的元技能，命令的输出格式要对 AI 友好——结构清晰、信息完整、便于 AI 理解和转述。

# 关键结果
1. 完成 internal/cmd/tools_merge.go：merge 子命令，执行流程如下：

   **前置检查**：
   - 检查 `learning/SUMMARY.md` 第一行包含 `APPROVED: true`（人工审核标记），否则拒绝并提示用户在 SUMMARY.md 顶部添加 `APPROVED: true` 后重试

   **创建 git 分支**：
   - 基于当前 HEAD 创建分支 `learning/job_N`
   - 切换到该分支

   **处理 wiki**（给人类的知识）：
   - 如存在 `learning/wiki/*.md`：
     * 将整个 `learning/wiki/` 目录 cp 到 `.rick/wiki/`（直接覆盖，同名文件被替换，新文件被添加）
     * 重新生成 `.rick/wiki/README.md`：扫描所有 .md 文件，提取每个文件第一个 `#` 标题和第一段摘要，生成 Markdown 索引表格

   **处理 skills**（给 AI 的技能）：
   - 如存在 `learning/skills/*.py`：
     * 将整个 `learning/skills/` 目录 cp 到 `.rick/skills/`（直接覆盖，同名文件被替换，新文件被添加）
     * 重新生成 `.rick/skills/README.md`：扫描所有 .py 文件，提取每个文件第一行的 `# Description:` 注释，生成 Markdown 索引表格

   **处理 OKR/SPEC**（完整版本覆盖）：
   - 如存在 `learning/OKR.md`：直接 cp 覆盖 `.rick/OKR.md`
   - 如存在 `learning/SPEC.md`：直接 cp 覆盖 `.rick/SPEC.md`
   - 不处理旧格式的 `OKR_UPDATE.md` / `SPEC_UPDATE.md`

   **完成 git 操作**：
   - 仅 git add 实际存在且有变更的路径（条件判断，避免 add 不存在的路径报错）：
     存在则 add：`.rick/wiki/`、`.rick/skills/`、`.rick/OKR.md`、`.rick/SPEC.md`
   - `git commit -m "learning: merge job_N knowledge"`
   - 切换回原分支（不合并）
   - 打印对 AI agent 友好的结构化摘要：
     ```
     ✅ Knowledge merge completed for job_N.

     Branch created: learning/job_N
     Current branch: <original_branch>

     Changes applied to .rick/:
       wiki:   N files  (new: X, updated: Y)
       skills: N files  (new: X, updated: Y)
       OKR:    [updated | no change]
       SPEC:   [updated | no change]

     Next: show diff to human for review.
       git diff <original_branch>..learning/job_N
     ```
   - **merge 命令只负责到此为止**：创建分支、cp 覆盖、commit、切回原分支、输出摘要。
   - 后续流程由 AI agent 在对话中继续完成：展示 diff → 人类审查循环 → 确认后 `git merge --no-ff` 合并回主分支 → `git branch -D learning/job_N` 删除分支

2. 完成 internal/workspace/skills.go：
   - `LoadSkillsList(rickDir string) ([]SkillInfo, error)`：扫描 `.rick/skills/*.py`，读取每个文件第一行的 `# Description:` 注释，返回 `[]SkillInfo{Path, Name, Description}`
   - `GenerateSkillsREADME(rickDir string) error`：根据 skills 列表生成 README.md 索引

3. 完成 internal/prompt/doing_prompt.go 修改：
   - 在 `GenerateDoingPromptFile` 中调用 `workspace.LoadSkillsList(rickDir)`
   - 如果 skills 列表非空，在 doing 提示词末尾追加以下章节：
     ```markdown
     ## 可用的项目 Skills

     以下 Python 脚本封装了本项目的可复用技能，在需要时可直接调用：

     | 脚本 | 用途 |
     |------|------|
     | `python3 .rick/skills/check_go_build.py` | 检查 Go 项目编译 |
     | `python3 .rick/skills/check_task_format.py` | 检查任务文件格式 |

     调用方式：`python3 {脚本路径} [--help]`，输出 JSON `{"pass": bool, "result": ..., "errors": []}`
     ```
   - 如果 `.rick/skills/` 不存在或为空，不添加此章节（不影响正常执行）

# 测试方法
1. 先运行 `go build -o bin/rick ./cmd/rick/` 构建最新二进制
2. 模拟 AI agent 调用场景：在 `.rick/jobs/job_1/learning/` 下创建以下测试文件，在 SUMMARY.md 第一行添加 `APPROVED: true`，运行 `./bin/rick tools merge job_1`（模拟 AI agent 在对话中执行此命令），验证：
   - `learning/wiki/test_wiki.md`（含 `# 测试 Wiki` 标题）
   - `learning/skills/test_skill.py`（含 `# Description: 测试技能`）
   - `learning/OKR.md`（完整 OKR 文件，顶部含变更注释）
   - `learning/SPEC.md`（完整 SPEC 文件，顶部含变更注释）
   - 执行后：当前分支切换到 `learning/job_1`
   - `.rick/wiki/test_wiki.md` 存在，`.rick/wiki/README.md` 包含该 wiki 条目
   - `.rick/skills/test_skill.py` 存在，`.rick/skills/README.md` 包含 test_skill 描述
   - `.rick/OKR.md` 内容已被覆盖为 `learning/OKR.md` 的内容
   - `.rick/SPEC.md` 内容已被覆盖为 `learning/SPEC.md` 的内容
   - `git log --oneline` 显示新的 commit
   - 命令最后打印变更摘要（wiki/skills/OKR/SPEC 各自状态）和 git diff/merge 指引
   - 当前分支切换回原分支
3. 不添加 `APPROVED: true` 直接运行 `./bin/rick tools merge job_1`，验证拒绝执行并打印提示信息
4. 在 `.rick/skills/` 下放一个含 `# Description: 检查 Go 项目编译` 的 check_go_build.py，运行 `./bin/rick doing job_1 --dry-run`（或检查生成的临时提示词文件），验证提示词包含 `## 可用的项目 Skills` 章节和对应表格
5. `.rick/skills/` 为空时运行 `./bin/rick doing job_1 --dry-run`，验证提示词不包含 skills 章节（不报错）
6. 运行 `go test ./internal/workspace/... ./internal/prompt/... ./internal/cmd/...`，验证测试通过
