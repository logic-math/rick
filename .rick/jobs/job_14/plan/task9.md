# 依赖关系

task5, task6, task7, task8

# 任务名称

实现 v2 端到端验收测试，基于 mock 覆盖 plan/doing/learning/dream 全流程

# 任务目标

升级 mock 系统并新建端到端测试脚本，在不启动真实 Claude Code 的情况下验证 rick v2 完整工作流。

## 关键风险预防

**mock NDJSON 格式必须对齐真实输出（实测验证）**：tool_use/tool_result 嵌套在 `message.content[]` 内，不在顶层：
```json
{"type":"system","subtype":"init","session_id":"mock-session-001","model":"mock"}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"go build ./..."}}]},"session_id":"mock-session-001"}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"Build ok","is_error":false}]},"session_id":"mock-session-001"}
{"type":"result","subtype":"success","is_error":false,"duration_ms":5000,"session_id":"mock-session-001"}
```

**executor 容错**：`claudecode/executor.go` 遇非 JSON 行 skip + warn，plain text mock 也能走通（act-path 行为轨迹为空，不报错）

**CWD 隔离**：rick 通过 `os.Getwd()` 找项目根目录，e2e 脚本必须 `os.chdir(tmpdir)` 后调用 rick 二进制

# 关键结果

1. **`tests/mock_agent/mock_agent.py` 升级**：
   - 新增 `--output-format` 参数（choices: `["text", "stream-json"]`，默认 `"text"`，**保持旧行为兼容**）
   - `stream-json` 模式输出符合真实格式的 NDJSON（见上方格式）
   - 新增场景：`doing_v2_success`（stream-json，无报错）、`learning_v2_success`（创建 SUMMARY.md + run_log）、`dream_success`（更新 dream/readme.md）

2. **`tools/mock_agent_testing.py` 同步更新**：新增 3 个 v2 场景描述

3. **新建 `tests/e2e_v2_test.py`**，4 阶段全覆盖，每个 phase 独立 `os.chdir(tmpdir)` 隔离：
   - **Phase 1 Plan**：`MOCK_SCENARIO=plan_success`，验证 `task1.md` 和 `OKR.md` 被创建
   - **Phase 2 Doing**：`MOCK_SCENARIO=doing_v2_success --output-format stream-json --verbose`，验证：
     - `doing/tasks/task1/act-path.md` 含 "## 执行摘要"、"报错次数: 0"、含行号的 raw_session.log 引用链接
     - `doing/tasks/task1/raw_session.log` 存在且每行可 `json.loads`
   - **Phase 3 Learning**：`MOCK_SCENARIO=learning_v2_success`，验证 `learning/SUMMARY.md` 存在且 `.rick/dream/run_log_1.md` 被创建
   - **Phase 4 Dream**：`MOCK_SCENARIO=dream_success`，验证 `.rick/dream/readme.md` 含已处理 job 记录
   - 测试结束后 `shutil.rmtree(tmpdir)`

4. **RED 验证覆盖**：`doing_v2_success` 场景中，testing agent 先输出 `{"pass": false}`（RED），coding agent 后输出 `{"pass": true}`（GREEN），验证 task8 RED 验证逻辑正常（不触发"意外通过"警告）

# 测试方法

1. 编译：`python3 tools/build_and_get_rick_bin.py`
2. mock stream-json 格式：`MOCK_SCENARIO=doing_v2_success python3 tests/mock_agent/mock_agent.py --output-format stream-json /dev/null`，输出 ≥4 行 NDJSON，最后一行含 `session_id`，tool_use 嵌套在 `message.content[]` 内
3. mock 旧场景不回归：`python3 tests/mock_agent/mock_agent.py --self-test`，全部通过
4. 端到端验收：`python3 tests/e2e_v2_test.py`，4 个 phase 全通过，输出 `✅ E2E v2 all phases passed`
5. 完整套件：`go test ./...` + `python3 tools/mock_agent_testing.py`，无新增失败
