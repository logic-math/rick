#!/bin/bash

# Task3 测试脚本：验证 Wiki 知识库索引和架构文档

echo "=== Task3 测试：Wiki 知识库索引和架构文档 ==="
echo ""

# 测试1：验证目录结构已创建
echo "测试1：验证目录结构已创建"
if [ ! -d ".rick/wiki" ]; then
    echo "❌ .rick/wiki 目录不存在"
    exit 1
fi

if [ ! -d ".rick/wiki/modules" ]; then
    echo "❌ .rick/wiki/modules 目录不存在"
    exit 1
fi

echo "✅ 目录结构正确"
echo ""

# 测试2：检查 index.md 包含完整的文档导航
echo "测试2：检查 index.md 包含完整的文档导航"
if [ ! -f ".rick/wiki/index.md" ]; then
    echo "❌ index.md 不存在"
    exit 1
fi

if ! grep -q "文档导航" .rick/wiki/index.md; then
    echo "❌ index.md 缺少文档导航"
    exit 1
fi

if ! grep -q "modules/" .rick/wiki/index.md; then
    echo "❌ index.md 缺少模块链接"
    exit 1
fi

# 检查是否包含所有模块链接
modules=("infrastructure" "parser" "dag_executor" "prompt_manager" "cli_commands" "git" "callcli" "workspace")
for module in "${modules[@]}"; do
    if ! grep -q "$module.md" .rick/wiki/index.md; then
        echo "❌ index.md 缺少 $module 模块链接"
        exit 1
    fi
done

echo "✅ index.md 包含完整的文档导航"
echo ""

# 测试3：验证 architecture.md 包含架构图和模块说明
echo "测试3：验证 architecture.md 包含架构图和模块说明"
if [ ! -f ".rick/wiki/architecture.md" ]; then
    echo "❌ architecture.md 不存在"
    exit 1
fi

if ! grep -q "系统架构" .rick/wiki/architecture.md; then
    echo "❌ architecture.md 缺少系统架构"
    exit 1
fi

if ! grep -q "核心模块职责" .rick/wiki/architecture.md; then
    echo "❌ architecture.md 缺少模块职责说明"
    exit 1
fi

if ! grep -q "数据流向" .rick/wiki/architecture.md; then
    echo "❌ architecture.md 缺少数据流向图"
    exit 1
fi

if ! grep -q "技术栈" .rick/wiki/architecture.md; then
    echo "❌ architecture.md 缺少技术栈说明"
    exit 1
fi

# 检查是否包含8个核心模块
module_count=$(grep -c "Module" .rick/wiki/architecture.md)
if [ "$module_count" -lt 8 ]; then
    echo "❌ architecture.md 模块说明不完整（期望8个，实际 $module_count 个）"
    exit 1
fi

echo "✅ architecture.md 包含架构图和模块说明"
echo ""

# 测试4：验证 core-concepts.md 包含核心理论解释
echo "测试4：验证 core-concepts.md 包含核心理论解释"
if [ ! -f ".rick/wiki/core-concepts.md" ]; then
    echo "❌ core-concepts.md 不存在"
    exit 1
fi

concepts=("Context Loop" "Agent Loop" "DAG" "提示词管理" "失败重试")
for concept in "${concepts[@]}"; do
    if ! grep -q "$concept" .rick/wiki/core-concepts.md; then
        echo "❌ core-concepts.md 缺少 $concept 解释"
        exit 1
    fi
done

echo "✅ core-concepts.md 包含核心理论解释"
echo ""

# 测试5：确保所有文档使用 Markdown 格式
echo "测试5：确保所有文档使用 Markdown 格式"
md_files=$(find .rick/wiki -name "*.md" | wc -l)
if [ "$md_files" -lt 11 ]; then
    echo "❌ Markdown 文件数量不足（期望至少11个，实际 $md_files 个）"
    exit 1
fi

echo "✅ 所有文档使用 Markdown 格式"
echo ""

# 测试6：确保文档包含目录和交叉引用
echo "测试6：确保文档包含目录和交叉引用"
if ! grep -q "](\./" .rick/wiki/index.md; then
    echo "❌ index.md 缺少交叉引用"
    exit 1
fi

link_count=$(grep -c "](\./" .rick/wiki/index.md)
if [ "$link_count" -lt 10 ]; then
    echo "❌ index.md 交叉引用数量不足（期望至少10个，实际 $link_count 个）"
    exit 1
fi

echo "✅ 文档包含目录和交叉引用"
echo ""

# 测试7：验证模块详解框架
echo "测试7：验证模块详解框架"
for module in "${modules[@]}"; do
    if [ ! -f ".rick/wiki/modules/$module.md" ]; then
        echo "❌ modules/$module.md 不存在"
        exit 1
    fi

    # 检查模块文档是否包含必需的部分
    if ! grep -q "概述" .rick/wiki/modules/$module.md; then
        echo "❌ modules/$module.md 缺少概述"
        exit 1
    fi

    if ! grep -q "核心功能\|核心命令\|核心组件" .rick/wiki/modules/$module.md; then
        echo "❌ modules/$module.md 缺少核心功能"
        exit 1
    fi
done

echo "✅ 所有模块文档包含完整框架"
echo ""

# 测试8：统计验证
echo "测试8：统计验证"
total_lines=$(wc -l .rick/wiki/*.md .rick/wiki/modules/*.md 2>/dev/null | tail -1 | awk '{print $1}')
echo "📊 总行数: $total_lines"

if [ "$total_lines" -lt 3000 ]; then
    echo "❌ 文档内容不足（期望至少3000行，实际 $total_lines 行）"
    exit 1
fi

echo "✅ 文档内容充实"
echo ""

# 所有测试通过
echo "=== ✅ 所有测试通过 ==="
echo ""
echo "Wiki 知识库索引和架构文档已成功创建！"
echo ""
echo "包含："
echo "  - ✅ Wiki 目录结构"
echo "  - ✅ index.md（完整导航）"
echo "  - ✅ architecture.md（架构图、模块说明、数据流向、技术栈）"
echo "  - ✅ core-concepts.md（Context Loop、DAG、提示词管理）"
echo "  - ✅ 8个模块详解文档"
echo "  - ✅ Markdown 格式、目录、交叉引用"
echo ""
echo "总计："
echo "  - 文档数量: $md_files 个"
echo "  - 总行数: $total_lines 行"
echo ""
exit 0
