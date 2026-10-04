# Learning Step 4 产出：Loop 模式提取（job_43）

> 依据：gen-loop 协议（相似 loop 优先升级；相似=trigger 80% 重叠或步骤序列一致）

## 相似性判定（先查后写）

对照 `.rick/loops/` 现有 7 个 loop（agent-runtime-bootstrap / go-refactor-migration / protocol-redesign / readme-wiki-sync / rick-rsi / tdd-red-green-refactor / deprecated）：

- **rick-rsi-loop**：trigger「修改 rick 自身并生效到生产」与本候选约 40-50% 重叠（反馈迭代最终也走 rsi 通道），但层次不同——rsi-loop 定义**单轮交付机制**（S0-S7），本模式定义**反馈轮次的编排**（分级/轻流程/验收节奏/呈批纪律）。job_43 实测两者叠加使用 5 轮（迭代 2-6）。**不构成 80% 重叠 → 新建而非升级**，且显式声明「执行机制沿用 rsi-loop 原文」避免重复定义。
- 其余 5 个 loop：场景无关。

## 产出

**新建** `.rick/loops/web-feedback-iteration-loop.md`（5.4KB，loops_check ✅ 7 loops）

核心内容（从 6 轮迭代实测提炼）：
1. **反馈三级分级**（F1 单点 bug 走快速通道跳过 grilling / F2 根因不明先 research / F3 产品范式裁决必须两案对比上呈）——F3 纪律直接来自白名单建了又拆（L2/R1）的教训
2. **gate 断言惯例**：存在+否定断言、生产零触碰、RICK_* 环境净化（gate7 教训）、RED→GREEN 判别力验证
3. **dev UI 真渲染验收为强制步骤**（P0 教训，引用 render_layer_acceptance_skill）
4. **release.md 头部三行保护**（两次误删教训）
5. 引用 4 个 Step 3 产出/相关 skill：acl_permission_recovery / render_layer_acceptance / virtualization_component_protocol / rick_release_ops（+ subprocess_env_isolation）

**升级** `.rick/loops/README.md`（目录索引 +1 行）

## 未提取（评估后排除）

- 「grilling 三轮追问模式」：已被 skill_grilling（easy/plan 提示词）覆盖，且非 job 内反复出现的循环
- 「P0 热修复流程」：单次事件，其可泛化部分（真渲染验收）已进 render_layer_acceptance_skill 与本 loop 步骤 5

## 验证

- `rick tools loops_check` → `✅ loops_check passed: /workdir/sunquan20/rick-dev/.rick (loops 7 / skills 0)`
- skills 0 为已知问题（checkMarkdownDir 跳过目录，research-L1 C1 记载的 P0 校验空转），非本次引入
