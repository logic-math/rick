# Job_57 Learning Step 5 报告：Domain 事实 + 文档架构信息更新（2026-11-29）

声明：I will use skill:gen-domain。

## 更新文件清单（commit 3323974）

| 文件 | 更新内容 |
|---|---|
| `.rick/domain/bugs.md` | 追加「dev-web 管理命令禁 go run + status fingerprint=MISMATCH 竞态」一条两症状（go run 收割服务进程；health build_id 为真值） |
| `.rick/domain/architecture.md` | human-loop 模块节 v3.1 → v8.3 整体重写（五阶段门禁表/单 agent orig/teacher_polish 递归解释/产物结构/六项关键设计决策；删除 think/research/exporter 四文件架构与阈值配置旧描述） |
| `.rick/domain/commands.md` | human-loop 命令节更新（v8 产物结构 thought_path+orig；无阈值配置；dry-run 验证命令更新） |
| `README.md` | 命令表 + 恢复表两处 human-loop 描述更新（思考门禁会话/orig 概念随机游走/递归解释/thought_path） |
| `wiki/modules/human-loop.md` | 整体重写 v8.3（职责/门禁流程表/架构文件表/产物结构/设计要点） |
| `wiki/README.md` | 目录树 + 模块索引两处同步（SENSE 五阶段思考门禁会话） |

## 核查无需更新（描述仍准确）

- `wiki/architecture.md`（draft 描述「human-loop 思考记录」——thought_path 即思考记录；session 恢复机制 v8 保留）
- `wiki/runtime-flow.md`（命令级统一编排泛指）
- `wiki/prompt-system.md`（无 human-loop 直接引用）
- `.rick/domain/README.md`（commands.md 行泛指）

## 验证

- `bash wiki/validate_wiki.sh`：✅ 通过（文档 20 / 错误 0 / 警告 0 / 文档风格检查通过）
- commit：3323974（6 files, +90/−90）

## 事实性发现（更新中核实的代码事实）

- v8 后 config 的 `HumanLoopConfig` 旧字段（think_top_n/sense_max_backflows/think_min_assumptions/research_source_weights）无消费者（grep internal/ 全量核实）——architecture.md/commands.md 如实记录
- v8 模板文件清单（8 个：sense_loop + sense + orig + teacher_polish + 五阶段）与渲染管线 `human_loop_prompt.go` 部署清单一致
