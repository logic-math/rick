# task4：dev 实例验证 + RSI 证据留痕（S1-S3）

## 任务目标

在 dev 工作区把 task1-3 的改动落到**可运行的 dev 实例**上（后端重启换构建指纹、前端 dist 刷新生效），并留下 RSI 流程 S1/S2/S3 要求的证据；为 S4 人类试用评审做好准备。**不得触碰生产**。

## 关键结果

- **KR4a dev 实例换构建**：`rick tools dev-web init`（幂等）→ `rick tools dev-web restart` → `GET http://127.0.0.1:8414/api/health` 返回 `status=ok` 且 `build_id` 为**本次构建**（不再是旧指纹；记录到 `doing/rsi/dev-iterations.md`）
- **KR4b 前端产物生效**：dev 树 `web/dist` 为最新构建（`npm run build` 产物），刷新 dev UI（8414）可见 Humans 页面与草稿持久化
- **KR4c 证据文件**（`doing/rsi/`，格式须过 `rick tools rsi_check`）：
  - `dev-iterations.md`：含至少一条 `build_id=<sha7>-<ts>` 或 `rick.dev.<sha7>-<ts>` 指纹行 + 改动摘要 + 验证结论
  - `gates.md`：含 `GATE gate1 pass=true` 与 `GATE gate2 pass=true`（回填本 job 已过的层门禁）
  - `release.md`：`rick tools release --dry-run` 输出关键行（`version=`、`rollback_point=`、`prod_untouched=true`）
  - `resume.md`：占位骨架（挂起清单与恢复结果由 S6 回填）
- **KR4d 生产零触碰**：`rick tools release --dry-run` → `prod_untouched=true`；生产树（`/workdir/sunquan20/AI_CODING/rick`）tracked 文件无改动
- **KR4e 试运行就绪**：`doing/rsi/approval.md` 预备骨架（含待填的 APPROVED 行），并输出一份**试用指引**（human 在 8414 上如何起步一次新 human-loop：选主题 → mission 面试 → 教学节阅读）；rubric 评审表落 `doing/evaluation-rubric.md`（5 维 + 结构指标，供 S4 human 填分）

# 写域

- .rick/jobs/job_44/doing/rsi/**（新建 5 文件）
- .rick/jobs/job_44/doing/evaluation-rubric.md（新建）
- .rick/jobs/job_44/doing/debug/**（如遇问题）
- internal/cmd/tools_update_pi.go（**清理 task1 遗留**：帮助文本第 54 行仍列 think/research/exporter → 改为 teach/research；一行文档修复）

# 依赖关系

- task1、task2、task3（必须有实现与层门禁通过后，dev 实例才有可验证的新行为）

## 实现要点

- **只碰 dev**：`rick tools dev-web *` 与 dev 树 `npm run build`；严禁 `kill` 生产（8413）、严禁在生产树改代码或 `go build -o bin/rick`
- 后端改动（task1/task2）需 `dev-web restart`；前端改动（task3）只需 dev 树 `npm run build`（overlay 软链 → 刷新即生效）
- `rick tools rsi_check --job job_44 --init` 可生成证据骨架（若命令可用则用它初始化，再填充）
- 命令必须在 dev 树 cwd 执行（`/workdir/sunquan20/rick-dev`）
- rubric 表：5 维（教学流畅度/SENSE 内化/事实锚定/提问启发性/学习成果质量）+ 结构指标（lessons ≥3、judgment ≥5、每课有引注）

## 测试方法

- `curl -s http://127.0.0.1:8414/api/health` → `status=ok` 且 `build_id` 与 `doing/rsi/dev-iterations.md` 记录一致
- `rick tools release --dry-run` → 输出含 `prod_untouched=true`
- `rick tools rsi_check --job job_44 --json` → 除 approval 项外（S4 未到）其余通过；输出留档
- 生产树 `git -C /workdir/sunquan20/AI_CODING/rick status --porcelain` → 无 tracked 文件 modified

## 禁止事项

- 不得执行 `rick tools release`（不带 `--dry-run`）——发布只能在 human 批准（S4）之后由 parent 执行
- 不得写 `doing/rsi/approval.md` 的 `APPROVED by=human` 行（那是 human 的唯一放行凭证，伪造 = 流程违规）
- 不得在工作区外写文件（除 `doing/` 内的证据文件）
