# 依赖关系
task1

# 任务名称
编写架构概览文档

# 任务目标
创建 `wiki/architecture.md`，全面介绍 Rick CLI 的架构设计。包含核心理论（AICoding 公式、Context Loop vs Agent Loop）、三维信息组织（环境、验证、反馈）、整体架构图、工作空间结构说明、核心设计原则和版本管理机制。使用 Mermaid 图表增强可读性。

# 关键结果
1. 完成 `wiki/architecture.md` 文档创建
2. 包含 AICoding 核心理论说明（AICoding = Humans + Agents 公式）
3. 详细说明 Context Loop（Plan → Doing → Learning）和三维信息组织
4. 绘制整体架构图（使用 Mermaid 图表）
5. 说明工作空间结构（.rick/ 目录组织）
6. 对比 Rick vs Morty 的简化设计原则
7. 说明版本管理机制（生产版 + 开发版）

# 测试方法
1. 验证文件已创建：`test -f wiki/architecture.md && echo "PASS" || echo "FAIL"`
2. 检查包含核心章节：`grep -q "## 核心理论\|## Context Loop\|## 架构设计\|## 工作空间结构\|## 设计原则\|## 版本管理" wiki/architecture.md && echo "PASS" || echo "FAIL"`
3. 验证包含 Mermaid 图表：`grep -q '```mermaid' wiki/architecture.md && echo "PASS" || echo "FAIL"`
4. 验证文档长度（至少 200 行）：`wc -l wiki/architecture.md | awk '{if($1>=200) print "PASS"; else print "FAIL"}'`
5. 检查包含关键术语：`grep -q "AICoding\|Context Loop\|Agent Loop\|三维信息\|拓扑排序" wiki/architecture.md && echo "PASS" || echo "FAIL"`
