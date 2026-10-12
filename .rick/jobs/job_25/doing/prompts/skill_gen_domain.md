---
name: gen-domain
description: 将项目事实性知识整理为 Domain 文件，存储在 .rick/domain/ 目录中
---

# skill:gen-domain（创建领域知识文件）

将 job 执行中发现的**事实性信息**整理为结构化 Domain 文件。

## Domain 是什么

Domain 存储项目特定的**客观事实**（Facts），是不依赖流程的静态知识：

| 类别 | 示例 |
|------|------|
| 环境配置 | 端口号、服务地址、版本要求 |
| 接口事实 | API 路径、参数格式、返回结构 |
| 已知问题与解法 | bug 根因 + 精确命令修复方案 |
| 业务规则 | 不可修改的外部约束、数据格式规范 |
| 构建事实 | 编译命令、测试命令、部署步骤 |

**与 Loop/Skill 的区别**：
- Loop = 执行流程（How to iterate）
- Skill = 能力模块（What to do when triggered）
- **Domain = 事实知识（What is true about this project）**

## 文件结构

```
.rick/domain/
├── env.md          # 环境配置事实（端口、路径、版本）
├── api.md          # 接口与服务事实
├── bugs.md         # 已知问题与精确解决方案
├── build.md        # 构建/测试/部署事实
└── {topic}.md      # 按主题的其他事实
```

## 文件格式

```markdown
# {主题} Domain

**最后更新**: {date}  **来源 Job**: {job_id}

## 事实列表

### {事实分类}

- **{事实名称}**: {具体内容}
  - 验证命令: `{exact command}`
  - 来源: {job_id} / {commit_hash}
  - 状态: ✅ 已确认

## 已知问题与解决方案

### {问题描述}

**根因**: {根本原因}

**精确解决步骤**:
```bash
{command_1}
{command_2}
```

**首次发现**: {job_id}  **验证状态**: ✅ 已修复
```

## 提取协议

```
1. 识别事实信号：哪些信息是客观、可验证的事实（非流程、非方法论）？
2. 分类：环境 / 接口 / 已知问题 / 构建 / 业务规则
3. 检查 .rick/domain/ 是否已有相关文件：
   - 有 → 追加新事实（不重复已有内容，更新状态）
   - 无 → 创建新的主题文件
4. 写入 .rick/domain/{topic}.md
```

## 质量标准

- 每条事实必须有来源（job_id 或 commit hash）
- 有命令的事实必须精确到完整命令（`go test ./internal/...` 而非"运行测试"）
- 只记录客观可验证的事实，不记录主观判断
- 已知问题的解决方案必须包含精确的可复制命令
