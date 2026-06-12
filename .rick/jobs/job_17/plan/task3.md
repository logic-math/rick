# 依赖关系
task2

# 任务名称
重构 easy.go 消除内部重复，复用已有 callClaudeCodeCLI

# 任务目标
`rick doing --easy` 功能完整保留，通过复用已有函数消除 `easy.go` 内的重复实现：
- `callClaudeCodeCLIEasy`（传 `--session-id`）和 `callClaudeCodeCLIResume`（传 `--resume`）与 `callClaudeCodeCLI`（plan.go）逻辑高度重复
- 按 SPEC variadic 改造模式，给 `callClaudeCodeCLI` 加 `extraArgs ...string` 参数，使三者统一
- 删除 `callClaudeCodeCLIEasy` 和 `callClaudeCodeCLIResume` 两个重复函数，调用处改为 `callClaudeCodeCLI(cfg, promptFile, "--session-id", sessionID)` 等形式
- `easy.go` 文件本身保留（easy 模式专用逻辑仍在此文件中），只消除重复代码

# 关键结果
1. `internal/cmd/plan.go` 中 `callClaudeCodeCLI(cfg, promptFile string)` 签名改为 `callClaudeCodeCLI(cfg *config.Config, promptFile string, extraArgs ...string)`，内部构造 args 时：**先追加 extraArgs，再追加 promptFile（当 promptFile 非空时才追加）**，即：
   ```go
   args := append([]string{}, extraArgs...)
   if promptFile != "" {
       args = append(args, promptFile)
   }
   cmd := exec.Command(claudePath, args...)
   ```
   所有现有调用方（plan.go、ctrl.go、human_loop.go）传的 promptFile 均非空，无需修改（variadic 兼容）
2. `internal/cmd/easy.go` 中 `callClaudeCodeCLIEasy` 和 `callClaudeCodeCLIResume` 两个函数删除，调用处替换：
   - `callClaudeCodeCLIEasy(cfg, sessionID, promptFile)` → `callClaudeCodeCLI(cfg, promptFile, "--session-id", sessionID)`（promptFile 非空，正常追加）
   - `callClaudeCodeCLIResume(cfg, sessionID)` → `callClaudeCodeCLI(cfg, "", "--resume", sessionID)`（promptFile 为空，不追加，最终命令为 `claude --resume <sessionID>`，与原行为完全一致）
3. `go build ./...` 通过，无未使用 import、无未定义符号
4. `go test ./internal/cmd/...` 通过

# 测试方法
1. **重复函数已删除**
   - 前置条件：重构完成后
   - 操作：`grep -n "func callClaudeCodeCLIEasy\|func callClaudeCodeCLIResume" internal/cmd/easy.go`
   - 预期输出：无输出（0 匹配）

2. **参数顺序校验（防止 sessionID 与 promptFile 互换）**
   - 操作：`grep -n "callClaudeCodeCLI(cfg" internal/cmd/easy.go`
   - 预期输出：调用行中 promptFile/mainFile 变量紧跟 cfg 之后（第 2 个参数），sessionID 在其后作为 extraArgs；不得出现 `callClaudeCodeCLI(cfg, sessionID, ...` 形式

3. **callClaudeCodeCLI 已支持 extraArgs**
   - 操作：`grep -n "extraArgs" internal/cmd/plan.go`
   - 预期输出：至少 2 行匹配（函数签名行 + 使用行）

4. **构建通过且现有测试无断裂（含 argv 行为验证）**
   - 操作 A：`go build ./...`；预期：exit code 0
   - 操作 B：在 `internal/cmd/plan_test.go`（或新建 `internal/cmd/callcli_test.go`）中**必须新增** `TestCallClaudeCodeCLI_MockBinary` 两个子用例（使用 mock binary 捕获 `os.Args` 验证 argv，不能只断言 exit code）：
     - (a) `promptFile_nonempty`：`callClaudeCodeCLI(cfg, "test.md", "--session-id", "abc")` → argv 为 `["--session-id", "abc", "test.md"]`，不含空字符串
     - (b) `promptFile_empty`（resume 场景）：`callClaudeCodeCLI(cfg, "", "--resume", "xyz")` → argv 为 `["--resume", "xyz"]`，不含 `""`
   - 操作 C：`go test ./internal/cmd/... -run TestCallClaudeCodeCLI -v`；预期：两个子用例均 PASS

5. **`rick doing --easy` flag 保留**
   - 前置条件：先执行 `python3 .rick/tools/build_and_get_rick_bin.py` 获取 `bin_path`
   - 操作：`$bin_path doing --help`
   - 预期输出：stdout 仍含 `--easy` 和 `--ctx` flag

6. **`rick doing --dry-run` 正常路径不受影响**
   - 前置条件：使用测试 5 中的 `bin_path`；job_1 的 plan 目录存在
   - 操作：`$bin_path doing --dry-run --job job_1`
   - 预期输出：exit code 0，stdout 包含 `[DRY-RUN]` 字样，无 `panic`

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
