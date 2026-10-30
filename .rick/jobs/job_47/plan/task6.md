# task6：prompts.py 渲染器 + 4-agent prompt 模板

## 任务目标

实现 prompt 预渲染层：把「角色模板 + env 上下文 + 格/精英元信息 + 实验计划」渲染成每迭代的 agent 系统提示词，落盘 codes/{uuid}/prompt_{role}.md（调度器派发时给 pi --append-system-prompt）。openevolve TemplateManager 目录覆盖机制（借鉴清单 #2）：模板可被工作区 prompts/ 覆盖、再被 env 覆盖。

## 关键结果

- 新建 `super/prompts/adjust.md`、`builder.md`、`coding.md`、`debug.md`（默认模板，中文，每模板含）：
  - **角色使命**（一句话）+ **输入材料说明**（调度器会注入哪些上下文段）
  - **adjust.md**：解空间调整者——读特征空间现状（轴/格/visit/权重/近期事件摘要）→ 产出**提案 JSON**：{proposals: [{type: soft_weight|hard_delete_cell|hard_modify_cell|hard_modify_gate, target_cell, action, evidence, rule_id}]}；纪律：软动作（weight 调控 0-1）自主、硬动作（删格/改格定义/修门）只提案；**不删格原则表述**（功能性灭绝=weight 0）；正交性是第一优先（新轴须与既有轴正交）
  - **builder.md**：实验设计者——读目标格元信息 + 邻格精英（跨格重组灵感）+ 历史失败摘要 → 产出 {experiment_plan: {idea, target_cell, parents: [code_path], plan_steps, expected_behavior}}；一迭代一实验；父本选择策略（同格精英优先 + 跨格灵感注入概率）
  - **coding.md**：实现者——读 experiment_plan + 父本代码 + env skills 自检协议 → 在 workdir 实现代码 → 跑 skills/self_check.py 自检 → 产出 {diff_path, self_check: {passed, stage, metrics}} + 六字段；纪律：**自检仅参考，框架终验为准**；禁止作弊模式声明（预存结果/篡改计时/降精度——env exploit 注册表会拒）
  - **debug.md**：修复者——读失败上下文（failure_stage + 诊断包 stderr_tail + 自检结果）→ 同 worktree 修复 → 产出 {fix_report: {attempts, root_cause, changes}, final_self_check}；上限 M=3 次由调度器控制，模板声明「这是第 {attempt}/3 次尝试」占位符
- 新建 `super/superopt/prompts.py`（stdlib-only）：
  - `def render(role: str, template_dir: str, ctx: dict) -> str`——模板查找链：`<ws>/prompts/<role>.md` → `<ws>/env/prompts/<role>.md`（env 级覆盖）→ 仓内 `super/prompts/<role>.md`（默认）；占位符 `{{key}}` 简单替换（str.format 防注入改用自定义 replace）；ctx 键：task/perf_desc/feature_axes/cell 元信息（descriptor/elite/visit/weight）/experiment_plan/attempt/失败上下文/事件摘要
  - `def write_prompt(workdir: str, role: str, content: str) -> str`——落盘 codes/{uuid}/prompt_{role}.md 返回路径
- 新建 `super/tests/test_prompts.py`：四角色渲染（占位符全替换——残留 `{{` 断言失败）；覆盖链（ws 覆盖 > env 覆盖 > 默认）；缺占位数据 → 保留原样不炸（宽松替换）

# 写域

- super/prompts/adjust.md、builder.md、coding.md、debug.md（新建）
- super/superopt/prompts.py（新建）
- super/tests/test_prompts.py（新建）

## 依赖关系

- task1/task2/task3/task4（Kahn L2；无 py import 依赖——render 是纯字符串层，ctx 由 task8 loop 组装）

## 实现要点

- 模板占位符与 ctx 键名对齐设计树 §3.4/§4.3；debug 模板的 attempt 占位符由调度器填
- 自定义 `{{key}}` 替换（不用 str.format——避免 JSON 大括号冲突）
- 模板文件加「输出契约」段：final message 必须含可解析 JSON 块（与 task5 的六字段提取对齐）

## 测试方法

`python3 -m unittest super.tests.test_prompts -v` 全绿 + 四模板文件存在且含各自角色关键词（proposals/experiment_plan/diff_path/fix_report——gate2 断言）

## 禁止事项

- 不得在模板写死 matmul 细节（env 上下文注入——模板通用）；不得让 render 依赖网络/外部命令
