# research-L1 简报：rick 功能面全景 + 改进候选（job_43 根层）

> 阶段：L1 根层调研 | 主题：功能级改进候选全景 | 基准：2026-09-24 · v5.0.11 | 方法：4 叶子扇出 + 主调研交叉抽验

## 事实性结论（12 条）

1. **CLI 面完整**：9 命令 + tools 9 子命令（init-pi/update-pi/theme/learning_check/dream_check/loops_check/dev-web/release/rsi_check）；dry-run 全覆盖；--resume 覆盖 plan/doing/easy/human-loop，learning/dream/ctrl 无。[代码+运行时，高]
2. **无 job 列表/状态/归档 CLI 命令**——进度只能 ls+cat tasks.json，归档仅 web 有（soft-archive）。[代码+运行时，高]
3. **skills 格式校验空转（P0）**：checkMarkdownDir 跳过目录（tools_loops_skills_check.go:45-49），skills 全是 *_skill/ 目录 → loops_check 实测输出 `skills 0`；且校验器要求英文章节与实际中文三段式（触发场景/预期效果/核心内容）冲突。[代码+运行时实测，高]
4. **skill 生命周期未机器化**：README 称"3 次 dream 未引用→deprecated"但无实现（grep 零命中）；README 索引 9 条 vs 实际 18 目录。[代码，高]
5. **dream 积压 14 个全成 job**：job_24-37 全 success 且 dreamed=0（实测盘点），仅早期 job_5/6/9/22 被 dream 过；dream 无 --job 定向、无积压视图。[运行时实测，高]
6. **web 功能面完整度高**：7 种会话类型全覆盖、多 workspace/多设备/SSE/挂起恢复（仅人工 /continue）/daemon/compact 全链路。[代码，高]

7. **Dreams 页占位**：dream 日志浏览"待后端接口支持"（Dreams.tsx:6,28），KnowledgeTree 只扫 domain/loops/skills。[代码，高]
8. **learning 产物不可 web 浏览**：ReadJobFile 白名单仅 plan/**+doing/**（jobs.go:385-397），learning/SUMMARY.md 取不到。[代码，高]
9. **domain 文档漂移 3 处**：commands.md 仍载 plan_check/doing_check（实际未注册）；architecture.md internal/ 包清单失真（列 executor/agent/actpath 等，实际为 builder/cmd/config/env/handler/prompt/runtime/web/workspace）；skills README 索引滞后。[代码+运行时，高]
10. **job_36 遗留 3 项**：F6 level_complete 写回晚于提交（SUMMARY 明确"下次迭代候选"）；dev 沙盒 catpaw 扩展不随 init 更新；pi 上游缺陷未报社区。[文档，中高]
11. **代码真 TODO 仅 2 处**：easy_prompt.go:310/344 debug.md 兼容回退（待迁移全量后清理）；web/src 零 TODO。[代码，高]
12. **bugs.md 无未修复项**，但有 ≥10 条"已实施待实测/待验证"。[文档，中]

## R7 上报项（L1 无法澄清，6 项）

1. F6 技术根因细节（写回时序代码路径）未深挖——若选为候选需下层调研
2. dream 积压成因（纯未主动运行 or 有阻塞）未考证
3. learning --resume 的 pi session 语义未考证（恢复面是否成立）
4. bugs.md 待实测项逐条验证状态未展开
5. gen-loop/gen-skill 是否 CLI 化属产品判断（现仅 easy prompt 注入模板）
6. rick tools 管理操作是否需要 web 入口属产品判断

## 改进候选清单（11 项，供 human L3 裁决；均为功能级，不触架构）

| # | 候选 | 面 | 量 | 价值 | 风险 |
|---|---|---|---|---|---|
| C1 | 修 skills 校验空转：checkMarkdownDir 支持 *_skill/ 目录 + 章节对齐中文三段式，令三个 check 的 skills 部分真正生效 | tools | 小 | P0 校验体系诚实性（当前 0 个被校验） | 需先统一 skill.md 规范（校验器/README/现状三套不一致） |
| C2 | skill 生命周期机器化：dream 引用统计 + 3 次未引用提示淘汰 | loop/dream | 中 | skill 进化闭环 | 引用判定语义需设计 |
| C3 | `rick jobs` 只读命令：列表/状态/过滤/积压标记 | CLI | 小-中 | CLI 核心体验（当前 ls+cat） | 低 |
| C4 | dream 定向+积压可见：--job <id> / --list | CLI | 小-中 | 消化 14 job 积压 | 低 |
| C5 | learning --resume 对齐其他会话 | CLI | 小 | 长会话恢复 | session 语义需考证 |
| C6 | Dreams 页 dream 日志浏览：后端扩 roots/加端点 + 前端填占位 | web | 小 | dream 产物闭环可视化 | 低 |
| C7 | job 文件白名单加 learning/**（前端同补入口） | web | 极小 | learning/SUMMARY 可 web 浏览 | 白名单语义勿泛化 |
| C8 | domain 文档-代码同步修复（commands/architecture/skills README）+ rel-state-004 空目录清理 | docs | 小 | agent 依 domain 决策不失真 | 低 |
| C9 | easy_prompt debug.md 回退清理（先盘点 debug/ 迁移完成度） | code | 小 | 减技术债 | 需全 job 盘点 |
| C10 | F6 修复：level_complete 写回时序 | code | 中 | 消 merge 冲突根因 | 中：涉 rick-gates 扩展时序 |
| C11 | bugs.md 待实测项集中验证轮（≥10 条） | QA | 小-中 | 知识库诚实性 | 低 |

**流程约束（非候选）**：job_43 本身即 RSI 迭代，执行须走 rick-rsi-loop S0-S7（dev 树改动 + 层门禁 + 人类审批 + release 提升），候选清单不改变该流程。

## 叶子文件（详证据）

- research-L1-leaf-a.md（CLI 面）/ -leaf-b.md（web 面）/ -leaf-c.md（loop/skill/tools 面）/ -leaf-d.md（近期交付与遗留）
