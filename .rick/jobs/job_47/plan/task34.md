# task34：四模板检索纪律（按 brief 三元组检索 + 关注点对齐）

# 任务目标
把检索写进四角色纪律：按 research_brief 的三元组检索（不自由发挥），关注点必须与目标格绑定，产出必须落到可实现动作。
**ctx 键契约**（task33）：`research_cmd`、`research_brief`（四角色均有）。

# 关键结果
- `super/prompts/builder.md`：
  - 检索段升级：从「强制 arXiv 检索」升级为**「按 research_brief 三元组检索」**——method（怎么搜）、sources（去哪搜）、focus（该格关注什么，如实验细节/数据/方法）、suggested_queries（起点）；可用 `{{research_cmd}}` 换角色/格再查
  - 纪律：≤3 篇；每篇落到 `{arxiv_id, title, insight(可实现机制), applicability(本格怎么用)}`；**关注点必须与本格 descriptor 对齐**（不是泛读论文堆砌）；无法检索则 `papers: []` 说明原因
- `super/prompts/coding.md`：增「检索纪律（文档/代码样例）」——按 research_brief（role=coding）查官方文档/代码样例，解决 API 用法与限制问题；**不查论文**（角色分工）；检索结论写进实现注释或 notes
- `super/prompts/debug.md`：同 coding（role=debug），失败定位时优先查「该 API/模式的已知坑与正确用法」
- `super/prompts/adjust.md`：按 research_brief（role=adjust）查**创新点/趋势**（不是实验细节）——用于判断「是否还有未覆盖的方法族」（expand_cell 依据）；检索结论写进 proposal 的 evidence
- 全线使用 `{{...}}` 占位符；未提供时「跳过检索」兜底

# 写域
- super/prompts/builder.md
- super/prompts/coding.md
- super/prompts/debug.md
- super/prompts/adjust.md

# 依赖关系
- task33（ctx 键契约；键名固定）

# 测试方法
四模板含 research_brief/research_cmd；`grep 无 /Users/`；test_prompts 绿

# 禁止事项
- 不改框架；不改 env_9 的 env/prompts
