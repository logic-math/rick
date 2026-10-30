# task14：e2e 全链验收 + 双回归 harness + 污染演练 + realpi 冒烟

## 任务目标

KR 验收门：mock e2e 全链（gen_env 产出→Go start→调度→入格→kill 重启→事件重放→北极星）+ 双回归 harness + 污染注入演练（KR4/KR6）+ realpi 冒烟 ≥3 真迭代（J8）。

## 关键结果

- 新建 `super/tests/e2e_test.py`（stdlib unittest，**全链 mock**，分钟级）：
  1. **env 生成**：以 super/env_templates/matmul/ 为源现场生成测试 ws 的 env/（拷贝模板；numpy 依赖：e2e 的 data 脚本用 venv——CI 环境 helper：若 env/.venv 缺失则 `python3 -m venv + pip install numpy`（网络可用；离线 fallback：bench 用纯 python 微缩矩阵 32×32——模板 bench 支持 `MATMUL_SIZE` env 缩放，模板侧不改，e2e 传参））
  2. **门先证红**：Scheduler --preverify-red → 通过（exploit 样本：模板 README 提示的 3 个假解——预存结果/篡改计时/降精度——e2e 内联构造）+ 金样例（参考实现）通过
  3. **Go 启动**：构建 `go build -o <tmp>/rick ./cmd/rick` → `rick super start --name e2e --env <env> --daemon`（真二进制、真 PySupervisor、pi 用 mock_pi 包装——RICK_SUPER_PI_PATH 环境变量? **取 super.json 的 pi_path 字段**：e2e 写 ws/super.json pi_path=<mock 路径>）
  4. **N≥5 迭代**：mock_pi 场景编排（builder/coding ok + 一次 coding 失败→debug 成功 + 一次 gate 拒绝）→ status --json 断言：iterations≥5/精英格≥1/token 聚合>0
  5. **监督树 kill 重启**：kill -9 py 调度器 → PySupervisor OnDead 自动重启（游标恢复）→ events seq 连续无断档 → archive 与事件流投影一致
  6. **事件重放一致性**：重建 Scheduler._recover 于快照 events → 投影状态 == archive 实态
  7. **北极星计算**：从 events 聚合（总迭代/入格数/各格最优 perf/token 成本/墙钟）→ status --json 一致
  8. **stop 优雅停**：super.json state=stopped + session_ended 事件存在
- 新建 `super/gates/elite_replay.py` + `exploit_regression.py`（双回归 harness，运行于 ws/gates/ 数据上）：
  - elite_replay.py：读 archive 全部精英 → 逐个跑当前门（CascadeGate.verify）→ **拒历史合法解=误杀** → 输出 {replayed, killed: []}；killed 空=过
  - exploit_regression.py：读 exploit_registry 全样本 → 逐个 verify → **放行已知坏解=放水** → 输出 {checked, leaked: []}；leaked 空=过
  - 两者 CLI：`python3 super/gates/elite_replay.py --workdir <ws>` → exit 0/1 + JSON（adjudicator 修门提案的证据协议消费方）
- **污染注入演练**（e2e_test.py 内一用例）：构造假阳性解（gate bug 场景：mock 门禁短暂放水一个坏解入格）→ 后续迭代软降权（gate 失败路径）→ pending 复测 → invalidation 出档 → resurrection 复活语义全链走一例（KR5 验收）
- **realpi 冒烟**（`super/tests/realpi_smoke.py` 或 go test `-tags realpi`——**取 py 脚本**）：读 RICK_REAL_PI_ARGS（pi_path + bootstrap 提示）→ 生成 env → `rick super start`（真 pi、max-iterations=3）→ 断言 ≥3 迭代完成 + 每迭代 4-agent 事件齐 + token 聚合非零 + 至少一次 gate_result 事件（真实验证跑了）；**RICK_REAL_PI_ARGS 未设 → 整用例 skipTest**（照 rick realpi 冒烟先例：环境门控不阻塞 CI）
- 新建 `super/tests/run_all.sh`：unittest discover + e2e + 双回归 一键（gate4 消费）

# 写域

- super/tests/e2e_test.py（新建）
- super/tests/realpi_smoke.py（新建）
- super/tests/run_all.sh（新建）
- super/gates/elite_replay.py、exploit_regression.py（新建）

## 依赖关系

- task8/task9（调度器全链）+ task11（Go 命令面）+ task12/task13（资产/env 模板）——Kahn L4 收口，依赖全部

## 实现要点

- e2e 用 mock_pi 的 MOCK_SCENARIO 编排（task5 的六场景组合）——**不 mock 的部分**：Go supervisor 真杀真启、events/archive 真读写、gates 真跑（fixture env subprocess）
- go build 在 e2e setup 一次（复用）；每次 e2e 独立 tmp ws（不污染 ~/.rick/super）
- 污染演练的「假阳性」注入：直接往 archive.json 写一个坏精英（模拟门曾放水）→ 跑重评估路径 → 断言出档 + 事件链
- run_all.sh 幂等可重复（清理 tmp）

## 测试方法

- `bash super/tests/run_all.sh` 全绿（单测+e2e+双回归）
- `RICK_REAL_PI_ARGS='<pi_path>' python3 super/tests/realpi_smoke.py` → ≥3 真迭代（**人工跑一次并留输出**——gate4 对 realpi 段只验证脚本存在与协议，真跑证据贴 doing/rsi/gates.md）
- Go 侧回归：`go test ./internal/super/... ./internal/handler/... -timeout 120s` 仍绿

## 禁止事项

- 不得为过 e2e 放宽任何断言（e2e 是验收门）；realpi 不可用不得标红阻塞（环境门控 skip）；不得污染用户 ~/.rick/super 真工作区
