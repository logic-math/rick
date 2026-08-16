# Rick CLI Wiki 贡献指南

欢迎为 Rick CLI Wiki 做贡献！

## 贡献方式

1. **报告问题**：在仓库创建 Issue，标签 `documentation`，描述问题所在文档与位置
2. **改进文档**：修正拼写、过时内容、补充缺失章节
3. **新增模块文档**：在 `wiki/modules/` 下新增，并更新 `wiki/README.md` 索引

## 文档规范

### 事实优先

每一个具体陈述都必须有代码出处。文档事实来源是 `.rick/domain/`（**只读**）与源码。凭记忆写文档是幻觉的主要来源。

### 事实调查命令

```bash
grep -n "AddCommand" internal/cmd/root.go
grep -n "Use:\|Long:\|Flags()" internal/cmd/<cmd>.go
cat internal/config/config.go
ls .rick/domain/ .rick/loops/ .rick/skills/
git log --oneline -20
```

### 文档层次

| 文档 | 路径 | 面向 | 权限 |
|------|------|------|------|
| README.md | 项目根 | 用户（GitHub 门面） | 可读写 |
| wiki/ | 项目根 `wiki/` | 用户（深入学习） | 可读写 |
| .rick/domain/ | `.rick/domain/` | agent（内部知识库） | **只读，禁止修改** |

### Mermaid 图表规范

GitHub 原生渲染 Mermaid，无需本地安装。图表节点 ID 用大写字母或下划线命名，避免中文/空格。

### 代码示例规范

- 命令示例用 ```bash 代码块
- Go 代码示例用 ```go 代码块
- 标注输出用注释 `# 应为 0`

## 提交流程

1. 修改文档后运行一致性检查（命令/flag/目录与代码事实一致）
2. `git add README.md wiki/`（**禁止** `git add .rick/domain/`）
3. `git commit -m "docs: <概述>"`
