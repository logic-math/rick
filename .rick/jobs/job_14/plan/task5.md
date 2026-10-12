# 依赖关系

task3

# 任务名称

实现 dream cmd 基础版本，显式引用 sense 和 evolve-skills skill

# 任务目标

新增 `internal/cmd/dream.go` 和 `internal/prompt/templates/dream.md`，实现 `rick dream` 命令。dream 是人机交互阶段（使用 `callClaudeCodeCLI` 交互模式），在对应步骤显式引用 core-skills。

**注意**：dream 只需 task3（LoadCoreSkills），不依赖 task1/task2 的 agent 接口——dream 使用交互式 `callClaudeCodeCLI`，不生成 act-path。

**dream SOP core-skill 映射：**

| 步骤 | skill |
|------|-------|
| c（SENSE 反思） | sense |
| f（skills 进化） | evolve-skills |
| 其余步骤 | 无 |

# 关键结果

1. **`workspace/workspace.go` 新增**：
   - `paths.go` 中追加常量 `DreamDirName = "dream"`
   - `EnsureDirectories()` 切片中追加 `filepath.Join(w.rickDir, DreamDirName)`

2. **新建 `internal/cmd/dream.go`**：
   - `NewDreamCmd()`，支持 `--dry-run` 标志
   - `dreamWorkflow()`：读取 `.rick/dream/readme.md` → 取最多 5 个待处理 job → 生成提示词文件 → 调用同包 `callClaudeCodeCLI(cfg, promptFile)`
   - `readme.md` 不存在时自动创建：`## 已处理 Jobs\n（空）\n\n## 待处理 Jobs\n`

3. **新建 `internal/prompt/templates/dream.md`**，完整 SOP（a-h 步）：
   - 步骤 c 承诺：`YOU MUST declare: "I will use skill:sense for reflection." Before analyzing each job.`
   - 步骤 e：整理 wiki；步骤 f 精简 SPEC.md，**SPEC.md ≤ 500 行**（删除已过时/低频触发条目）
   - 步骤 f 承诺：`YOU MUST declare: "I will use skill:evolve-skills." Before modifying any skill.`
   - 变更约束：仅允许修改 `wiki/`、`tools/`、`SPEC.md`，**严禁修改业务代码**
   - 仅含 `sense` 和 `evolve-skills`（不含 tdd/debug/gen-skill/tc）

4. **新建 `internal/prompt/dream_prompt.go`**：
   - `GenerateDreamPromptFile(jobIDs []string, rickDir string) (string, error)`
   - 读取各 job 的 `doing/tasks/*/act-path.md` 内容和 `.rick/dream/run_log_*.md` 注入模板
   - 调用 `LoadCoreSkills([]string{"sense", "evolve-skills"})` 注入 `{{core_skills}}`

5. **`root.go` 注册**：`rootCmd.AddCommand(NewDreamCmd())`

# 测试方法

1. 编译：`python3 tools/build_and_get_rick_bin.py`，rick 二进制含 dream 命令
2. **KR3 dry-run 全提示词验证**（OKR KR3）：
   ```bash
   rick_bin=$(python3 tools/build_and_get_rick_bin.py)
   $rick_bin dream --dry-run 2>&1 | tee /tmp/dream_dryrun.txt
   # 验证输出包含 SOP 各步骤关键词
   python3 -c "
   c = open('/tmp/dream_dryrun.txt').read()
   assert 'skill:sense' in c, 'missing sense'
   assert 'skill:evolve-skills' in c, 'missing evolve-skills'
   assert 'readme.md' in c, 'missing readme.md reference'
   assert 'I will use skill' in c, 'missing commitment declaration'
   print('dream dry-run OK')
   "
   ```
3. **SPEC 行数约束验证**：`python3 tools/check_prompt_variables.py --phase dream --keywords "500"` 或直接检查 dream.md 包含行数限制说明：`grep "500" internal/prompt/templates/dream.md`，输出非空
4. help：`<rick_bin> dream --help`，输出含 `--dry-run` 标志说明
4. sense skill：`python3 tools/check_prompt_variables.py --phase dream --keywords "skill:sense"`，`{"pass": true}`
5. evolve-skills：`python3 tools/check_prompt_variables.py --phase dream --keywords "skill:evolve-skills"`，`{"pass": true}`
6. 无 tdd 污染：`python3 tools/check_prompt_variables.py --phase dream --keywords "skill:tdd"` → "关键词未找到"
7. workspace 目录：`go test ./internal/workspace/... -v -run TestDreamDir`（此测试在本 task 中新增），验证 `.rick/dream/` 被创建
8. 完整测试：`go test ./...`，无新增失败
