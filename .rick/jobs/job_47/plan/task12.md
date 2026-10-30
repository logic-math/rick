# task12：super-loop + gen_env skill（系统级知识资产）

## 任务目标

新建两个系统级知识资产：`super-loop`（super 会话中控 loop）与 `gen_env_skill`（env 生成器 skill）。这是 J3 变更裁决的核心交付：env 建设与升级在 super 中控会话内完成，门禁系统设计引导归 gen_env（L3-Q2 架构修正）。

## 关键结果

- 新建 `.rick/loops/super-loop.md`（照 loops/ 目录同构格式：frontmatter name/trigger/scope + 正文）：
  - frontmatter：`name: super-loop`；`trigger: "当启动或操作 rick super 模式（env 建设/升级、启停监控后台进化、干预队列）时触发"`；`scope: super 会话（全局）`
  - 正文（Step 结构，中控中心会话的工作法）：
    - Step 0 环境自检：`rick super status` / 工作区存在性 / pi 可用性（pi_path 探测）
    - Step 1 env 建设引导：env 缺失 → 加载 gen_env skill 现场生成（matmul 走 env_templates/matmul 模板起步）；已有 → 跳 Step 2
    - Step 2 启动：`rick super start --name <n> --env <path> [--daemon]`——启动前门先证红确认（exploit 拒识 100% 才放行——读 status 输出的 gate 状态）；--daemon 后台运行
    - Step 3 监控：`rick super status`（心跳/迭代游标/预算/北极星）+ `rick super event --tail`（事件流跟踪）
    - Step 4 干预：`rick super event --type <t> --payload '<json>'`（写干预队列；**不直写 events.jsonl**——单写者架构，干预经队列由调度器摄取入事件）
    - Step 5 env 升级（热更新）：改 env 定义（门禁/评分/特征轴）→ 调度器每迭代 subprocess 直调 pi（agent 上下文每迭代重建，env 变更下迭代自动生效）；改门禁必须重跑门先证红（gate preverify_red）
    - Step 6 停止与收尾：`rick super stop`（优雅：SIGTERM 让调度器落完当前事件）；长跑预算墙 (--max-wall-time)
    - 附：super 工作区目录结构速查 + events 事件类型速查（引用 super/README.md）
- 新建 `.rick/skills/gen_env_skill/skill.md`（目录形式，三段式：触发场景/预期效果/核心内容）：
  - **触发场景**：super 会话中 env 缺失或需升级（版本演进/门禁加强/特征轴调整）
  - **预期效果**：产出可运行 env——目录协议：`env/{env.json, feature_space.json, code/, skills/, data/, domain_knowledge.md}` + 可选 `.venv/`（Python 依赖隔离，stdlib 之外依赖装这里，gates 用 `.venv/bin/python` 优先）
  - **核心内容**（访谈式流程，逐环节引导用户决策）：
    1. 任务定义（任务名/输入输出/参考实现——参考实现放 code/，是正确性锚点与 naive 起点）
    2. 评分方向（perf_desc：高优=越大越好 / 低优=越小越好；perf_type 单位如 runtime_ms/score）
    3. **门禁系统设计引导**（L3-Q2 核心）：正确性门（data/test_correctness.py：输出 JSON {passed, max_error}，容差 ε 由用户定）/ 性能门（data/bench.py：输出 {runtime_ms, score}，多计时 K=3 中位数由框架做）/ 环境健康自检（skills/ 自检脚本协议）/ exploit 注册表初始化（gates/exploit_registry.json：3-5 个已知作弊/坏解样本——引导用户想「这个域里 agent 会怎么作弊」）/ 金样例（golden_examples/：参考实现即首个金样例）
    4. 初始特征空间（feature_space.json：2-4 轴起步，每轴 name/type/取值域；轴要正交——不同轴刻画解的不同维度）
    5. 初始解（code/ 的 naive 起点——进化从它起步）
    6. 验收自检清单：参考实现过正确性门 ✓ bench 出分 ✓ exploit 样本被正确性门拒识 ✓（门先证红的地基）
- 两文件均遵循 gen-loop/gen-skill skill 的格式规范（读 .rick/skills/gen_loop_skill、gen_skill_skill 参照）

# 写域

- .rick/loops/super-loop.md（新建）
- .rick/skills/gen_env_skill/skill.md（新建）

## 依赖关系

- 无（Kahn L1；纯文档资产。内容引用 CLI 子命令名/目录协议——与 task11/task13 的实现契约对齐（以设计树 §4.3 为共同事实源），无文件依赖）

## 实现要点

- loops 目录现有成员格式参照（frontmatter 三键 + 正文 Step 结构）
- skill.md 三段式参照现有 skill（如 fake-binary-script）
- super-loop 的 Step 2「启动前门先证红」与 task7 的 preverify_red、task11 的 start 流程三方对齐（表述一致）

## 测试方法

- 两文件存在 + frontmatter 可解析（yaml 头 name/trigger/scope 三键齐）
- 内容契约：super-loop 含 start/stop/status/event 四子命令引用 + 干预走队列表述；gen_env skill 含「门禁系统设计引导」环节 + 目录协议完整
- grep 交叉引用一致：两文件引用的目录协议与 task13 的 matmul 模板实际目录一致

## 禁止事项

- 不得把门禁的具体阈值写死进 loop/skill（那是 env 范畴——只写引导方法）
- 不得改动其它既有 loop/skill 文件
