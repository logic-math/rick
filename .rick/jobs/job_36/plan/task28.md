# 依赖关系
task25
task26
task27

# 写域
scripts/rsi-loop-e2e.sh
wiki/self-evolve.md
.rick/jobs/job_36/plan/sim-report.md

# 任务目标
端到端验证「在（模拟）生产 UI 里启动 RSI 会话 → loop 自动加载 → 按 loop 走完自进化流程 → 产出可被 `rsi_check` 校验」，并把新增能力补进运维手册与验收报告。

# 关键结果
1. `scripts/rsi-loop-e2e.sh`（可重放、幂等、**真实生产只读**）：
   - ① 前置：dev 环境就绪（`dev-web init` 幂等）+ `rick tools loops_check --dir .rick` pass=true（loop 载体合规）
   - ② **loop 注入断言**：用**模拟生产**（临时 repo/home/端口）启动实例并在其上 `POST /api/sessions {"workspace_id":"<模拟 dev 工作区>","type":"rsi"}` → 断言返回的会话（或 `GET /api/sessions/{id}/prompt`）**包含 loop 全文标记**（例如 `rick-rsi-loop` 与 `产出评估` 字样）；
   - ③ **守卫断言**：workspace=生产仓库根（模拟）→ 400 且 message 含「dev 工作区」；workspace 缺 `.rick/loops/rick-rsi-loop.md` → 400；
   - ④ **rsi_check 断言**：在临时 job 目录上跑 `rsi_check` → 缺证据时 fail 并给中文指引；`--init` 生成骨架 → 填入合规内容（含 `APPROVED by=human`）→ pass=true；
   - ⑤ **merge-source 断言**（模拟生产两个 git 仓库）：无冲突 → 合并成功 + release 回执含 `RELEASE_MERGE merged=true`；**制造冲突 → release 中止报错且模拟生产工作树恢复干净**（AI 修复后重跑可成功）；
   - ⑥ 收尾单行 JSON `{"pass":...,"steps":[...],"prod_touched":false}`，并断言真实 prod 的 `~/.rick/web/sessions.json` 指纹与 8413 健康未变。
2. `wiki/self-evolve.md` 增补章节：
   - **「RSI 自进化」如何使用**：在 UI 新建会话选类型 `RSI 自进化` → 选 dev 工作区 → loop 自动加载 → 按 loop 走（含每条命令与人类确认点）；
   - **为什么必须走 loop**（制度 vs 机制；产出评估可机器校验）；
   - **源码合并**（`--merge-source` 语义、冲突时怎么办：AI 修复 → 重新 release；`--rollback` 不含源码回滚）；
   - **把 dev 工作区注册到生产 UI 的操作步骤**（`POST /api/workspaces`，additive、可逆）。
3. `plan/sim-report.md` 追加本增量验收章节：逐条对照 KR1-KR4 给证据 → 结论。

# 测试方法
bash scripts/rsi-loop-e2e.sh          # 末行 {"pass":true,...,"prod_touched":false}
python3 .rick/jobs/job_36/plan/gates/gate14.py
curl -s --noproxy '*' http://127.0.0.1:8413/api/health   # 生产回归

# 上下文提示
- 复用 `scripts/self-evolve-e2e.sh` 的「模拟生产」构造手法（临时 start 脚本 + 临时端口 + 临时 state dir）。
- `RICK_RSI_ALLOW_PROD_TREE=1`（task24 提供的覆盖开关）用于在模拟生产里把「模拟 dev 工作区 == 模拟生产树」的边界跑通。
- 真实生产端到端由**人类**在 release 之后手动跑一次（loop 的第一次真实执行）——脚本只需覆盖可在隔离环境验证的部分，并在文档里写明这一步。
