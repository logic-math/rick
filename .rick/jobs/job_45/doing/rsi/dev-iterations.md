# RSI 证据：dev-iterations

通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。
示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## 迭代 1（2026-10-10）
- 改动：human-loop 教师会话 v5——teach/research 交付契约改写（正文入会话 + domain-survey/syllabus 任务模式）、sense_loop parent 新增「总」阶段（大纲确认后才开课）、human_loop_prompt.go 注入 syllabus_path、测试补断言。commit 7fbc77a
- 构建指纹：build_id=ccf5dca-261010180406（dev 实例 8414 已重启，旧 57f599c-261010152340）
- 验证：go build ./... OK；go test internal/{prompt,web,handler,builder,env} 全绿；dry-run 渲染含 syllabus 路径与「总」阶段
- 待 human：新建 human-loop 会话验证（总分结构 + 正文入会话）
