# Learning Step 3 产出：可复用 Skills 提取（job_43）

> 依据：gen-skill 协议（相似 skill 优先升级，不建重复文件）| 提取源：debug/bug1+bug2、rsi/{dev-iterations,gates,release}.md、6 轮迭代全程

## 清单（4 新建 + 1 升级）

| # | Skill | 类型 | 一句话 |
|---|-------|------|--------|
| 1 | `virtualization_component_protocol_skill` | 新建 | 虚拟化库自定义 components 必须全量转发 props（`...rest`）+ 引用稳定 + 间距走 Footer——防 task7 P0（Scroller 丢 props 全空白） |
| 2 | `render_layer_acceptance_skill` | 新建 | 渲染层结构性改动 gate 绿≠可用——release 前必须 dev UI 真渲染走主路径（静态断言测不到运行时协议破坏） |
| 3 | `acl_permission_recovery_skill` | 新建 | 环境（休眠/快照）重置 ACL mask/属主后的恢复法：诊断→父目录可写则 rename+git checkout 重建（零丢失）→ 沙盒直接重建 |
| 4 | `rick_release_ops_skill` | 新建 | 本机 release 实操：RICK_RELEASES_DIR 重定向（bin/releases 属主坑）、版本号=构建时刻非演练时刻、detach 跨休眠、生产接管 |
| 5 | `subprocess_env_isolation_skill` | **升级**（追加 Step 5） | RICK_* 显式 env 穿透 t.Setenv(HOME) 隔离（RICK_STATE_DIR 优先级高于 HOME 派生）——gate 跑 go test 前剥离 RICK_* 前缀全家 |

## 未提取（评估后排除）

- **bug1（ResumeEasyIn 覆盖任务史）**：单一代码 bug 已留痕遗留下轮修，非可泛化模式（其捕获机制「生产零触碰断言」已是 rick-rsi-loop 既有纪律）
- **白名单建了又拆（迭代5→6）**：human 产品裁决变更，非执行缺陷——其教训（「功能默认值尽量简单、避免为单一用户堆防御逻辑」）属产品决策范畴，记入 SUMMARY 不入 skill
- **dev-web/工具箱等 web 功能知识**：已在 task 文件+代码注释中，无跨 job 复用价值

## 文件清单

```
.rick/skills/virtualization_component_protocol_skill/skill.md   (新建, 1157B)
.rick/skills/render_layer_acceptance_skill/skill.md             (新建, 879B)
.rick/skills/acl_permission_recovery_skill/skill.md              (新建, 1219B)
.rick/skills/rick_release_ops_skill/skill.md                    (新建, 1185B)
.rick/skills/subprocess_env_isolation_skill/skill.md            (升级, 追加 Step 5)
.rick/skills/README.md                                          (索引+4 行)
```
