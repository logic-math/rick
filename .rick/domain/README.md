# Domain 知识文档

项目级领域知识，来源于 SPEC.md（v2.9.0 迁移），补充 loops/skills 中未覆盖的命令规范、架构设计和编码约定。

| 文件 | 内容 |
|------|------|
| `architecture.md` | 技术栈、模块划分、DIP 组合根模式、act-path/agent/dream 模块设计 |
| `commands.md` | 所有 rick 命令规范（doing/plan/learning/dream/ctrl/human-loop）+ NDJSON 解析规范 |
| `go-patterns.md` | Go 编码规范（Cobra flag/variadic/embed.FS/包内共享/接口签名/配置污染防护） |
| `testing-conventions.md` | 测试约定（go test 范围/mock命名/Mock Agent同步/JSON输出/断言精确性） |
| `project-conventions.md` | 路径约定、工程实践、构建/发布流程、新架构说明 |

## 与 skills/loops 的关系

- **domain/**：描述性知识，"系统是什么、规范是什么"，供人类和 agent 查阅参考
- **skills/**：操作性知识，"当 X 时执行 Y"，agent 在触发条件下直接加载执行
- **loops/**：迭代控制流，"重复执行直到收敛"，带停止条件的工作流
