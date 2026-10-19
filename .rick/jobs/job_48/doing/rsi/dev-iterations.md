# RSI 证据：dev-iterations

通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。
示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## 迭代 1（2026-10-18，job_48 起点）
- 任务：web super 中控会话（super 会话类型 + 严格加载 gen_env skill + 工作区模型迁移 .rick/super_env）
- 裁决：J1=A 纯会话 / J2=env 在 <ws>/.rick/super_env 生成 / J3=A CLI / J4=A 无管理 UI / L2-Q1=A CLI 一并切换
- plan：4 task（task1 CLI 解析切换 → task2 web 类型 → task3 前端 ∥ task4 提示词+e2e），pipeline_gate 通过
