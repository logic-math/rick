# task33：loop.py 注入 research（命令 + 预生成 brief）

# 任务目标
把检索三元组注入四角色：框架预调用 research.py 生成 brief 直接进 ctx（省 agent 回合），同时给命令供 agent 自行查询其它格/角色。

# 关键结果
- loop.py：
  - `_research_cmd(role, cell_id)` → `<python> <env>/skills/research.py --role <role> [--cell <cell>]`；文件缺失 → 空串
  - `_research_brief(role, cell_id)` → subprocess 调 research.py 解析 JSON（失败静默返回 None）
  - ctx 注入：
    - **builder**：`research_cmd`（role=builder, cell=目标格）、`research_brief`（预生成）
    - **coding**：`research_cmd`（role=coding）、`research_brief`
    - **debug**：`research_cmd`（role=debug）、`research_brief`
    - **adjust**：`research_cmd`（role=adjust）、`research_brief`（cell 为空=全局视野）
  - legacy env（无 research.py）→ 命令空串 + brief None（模板兜底「未提供则跳过」）
- 测试 test_loop：①四角色 prompt 含 research_cmd/brief 渲染（fixture env 造 research.py）②legacy env 不炸 ③brief 字段（method/sources/focus/suggested_queries）进 prompt

# 写域
- super/superopt/loop.py
- super/tests/test_loop.py

# 依赖关系
- task32（research.py CLI 契约）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + 全量无回归

# 禁止事项
- 不动 gates.py / research.py 本体（task32）/ 模板（task34）
