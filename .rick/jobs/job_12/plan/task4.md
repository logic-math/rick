# 依赖关系
task1, task2, task3

# 任务名称
端到端验证 RFC-002 全部 KR 落地

# 任务目标
编写并运行端到端验证脚本 `.rick/jobs/job_12/doing/tests/rfc002_e2e_test.sh`，覆盖 job_12 OKR 的全部 4 个 KR，并运行现有集成测试 `tests/tools_integration_test.sh` 确保无回归。

# 关键结果
1. 创建 `.rick/jobs/job_12/doing/tests/rfc002_e2e_test.sh`，包含 10 个断言，覆盖 KR1~KR4 的所有验收条件
2. 脚本运行结果全部通过（exit 0），每个断言输出 PASS/FAIL
3. `tests/tools_integration_test.sh` 运行全部通过（无回归）

# 测试方法
1. 运行 `bash .rick/jobs/job_12/doing/tests/rfc002_e2e_test.sh` 验证 exit 0 且无 FAIL 行
2. 运行 `bash tests/tools_integration_test.sh` 验证 exit 0 且无 FAIL 行

## rfc002_e2e_test.sh 断言清单

### KR1：tools/ 有 5 个 .py 工具脚本
- 断言1：`ls tools/*.py | wc -l` 等于 5
- 断言2：`python3 tools/build_and_get_rick_bin.py` 返回 JSON 且包含 `rick_bin` 字段

### KR2：.rick/skills/ 只含 .md，触发场景非空
- 断言3：`ls .rick/skills/*.py 2>/dev/null` 为空（无 .py 文件）
- 断言4：`cat .rick/skills/index.md` 包含三列表格（grep `| Skill | 描述 | 触发场景 |`）
- 断言5：`cat .rick/skills/index.md` 不含空触发场景列（grep -v `| |` 验证）

### KR3：dry-run 输出正确
- 断言6：`bin/rick doing job_12 --dry-run` 输出包含 `tools/` 字样（tools section 非空）
- 断言7：`bin/rick doing job_12 --dry-run` 输出包含 `.md` skill 名称（如 `verify_rick_check_commands`）
- 断言8：`bin/rick doing job_12 --dry-run` 输出不含 `.py` 文件列表（skills section 无 `.py` 条目）

### KR4：learning 模板正确区分 tools/skills
- 断言9：`grep "skills/\*\.py" internal/prompt/templates/learning.md` 返回空（无旧格式）
- 断言10：`grep "tools/\*\.py" internal/prompt/templates/learning.md` 非空 且 `grep "skills/\*\.md"` 非空
