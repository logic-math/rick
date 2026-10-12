# Job OKR: 代码库死代码清理与重复逻辑消除

## 目标 (Objective)
清除 RFC-refactor-2 和 RFC-refactor-go-codebase 中记录的代码欠债：删除无用的 skill 文件、消除 frontmatter 解析重复、移除 `--easy` 模式及其所有相关代码、清理 `tools merge` 残留文档引用，使代码库更简洁、维护成本更低。（注：RFC-refactor-go-codebase §1 workspace 死代码已在之前 job 中清理完毕，本 job 不重复处理；§2 tools merge 选择删除文档引用而非实现；§3 RED verification 本 job 跳过；RFC-refactor-2 P2 TODO 2026-08 本 job 跳过，后续单独建 RFC；easy.go 保留文件本身，因已通过 --easy flag 集成，不属于死代码）

## 关键结果 (Key Results)
- KR1: `internal/prompt/templates/skills/` 中的 3 个死代码文件（`tc.md`、`tdd.md`、`tdd/testing-anti-patterns.md`）被删除，`tc.md` 内容无损合并进 `tdd-zh.md`，`go test ./internal/prompt/...` 仍通过
- KR2: `internal/parser/frontmatter.go` 提取公共 frontmatter 解析函数，`debug_dir.go` 和 `easy_prompt.go`（删除前）均改为调用它，消除重复实现
- KR3: `callClaudeCodeCLI` 支持 `extraArgs ...string`，`easy.go` 中的 `callClaudeCodeCLIEasy`/`callClaudeCodeCLIResume` 两个重复函数删除；`rick doing --easy` 功能完整保留；`go build ./...` 通过
- KR4: SPEC.md、wiki/ 中所有 `tools merge`、`easy` 模式引用被更新或删除，文档与实现一致
