# 依赖关系
task2

# 任务名称
编写核心模块文档

# 任务目标
创建 `wiki/modules/` 目录，并为每个核心模块创建详细文档。包含 7 个模块：cmd（命令处理器）、workspace（工作空间管理）、parser（内容解析）、executor（任务执行引擎）、prompt（提示词管理）、git（Git 操作）、config（配置管理）。每个文档包含模块职责、核心类型、关键函数、类图和使用示例。

# 关键结果
1. 完成 `wiki/modules/` 目录创建
2. 完成 `wiki/modules/cmd.md` - 命令处理器文档
3. 完成 `wiki/modules/workspace.md` - 工作空间管理文档
4. 完成 `wiki/modules/parser.md` - 内容解析文档
5. 完成 `wiki/modules/executor.md` - 任务执行引擎文档
6. 完成 `wiki/modules/prompt.md` - 提示词管理文档
7. 完成 `wiki/modules/git.md` - Git 操作文档
8. 完成 `wiki/modules/config.md` - 配置管理文档
9. 每个文档包含：模块职责、核心类型、关键函数、类图（Mermaid）、使用示例

# 测试方法
1. 验证 modules 目录已创建：`test -d wiki/modules && echo "PASS" || echo "FAIL"`
2. 验证所有 7 个模块文档已创建：`test -f wiki/modules/cmd.md && test -f wiki/modules/workspace.md && test -f wiki/modules/parser.md && test -f wiki/modules/executor.md && test -f wiki/modules/prompt.md && test -f wiki/modules/git.md && test -f wiki/modules/config.md && echo "PASS" || echo "FAIL"`
3. 检查每个文档包含必要章节（以 cmd.md 为例）：`grep -q "## 模块职责\|## 核心类型\|## 关键函数\|## 类图\|## 使用示例" wiki/modules/cmd.md && echo "PASS" || echo "FAIL"`
4. 验证至少一个文档包含 Mermaid 类图：`grep -q '```mermaid' wiki/modules/executor.md && echo "PASS" || echo "FAIL"`
5. 验证文档总行数（所有模块文档总和至少 500 行）：`wc -l wiki/modules/*.md | tail -1 | awk '{if($1>=500) print "PASS"; else print "FAIL"}'`
