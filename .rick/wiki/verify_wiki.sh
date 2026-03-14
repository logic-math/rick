#!/bin/bash

# Wiki 知识库验证脚本

echo "=== Rick Wiki 知识库验证 ==="
echo ""

# 1. 验证目录结构
echo "1. 验证目录结构..."
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

# 2. 验证核心文档
echo "2. 验证核心文档..."
core_docs=("index.md" "architecture.md" "core-concepts.md")
for doc in "${core_docs[@]}"; do
    if [ ! -f ".rick/wiki/$doc" ]; then
        echo "❌ $doc 不存在"
        exit 1
    fi
    echo "✅ $doc 存在"
done
echo ""

# 3. 验证模块文档
echo "3. 验证模块文档..."
module_docs=(
    "infrastructure.md"
    "parser.md"
    "dag_executor.md"
    "prompt_manager.md"
    "cli_commands.md"
    "git.md"
    "callcli.md"
    "workspace.md"
)
for doc in "${module_docs[@]}"; do
    if [ ! -f ".rick/wiki/modules/$doc" ]; then
        echo "❌ modules/$doc 不存在"
        exit 1
    fi
    echo "✅ modules/$doc 存在"
done
echo ""

# 4. 验证文档内容
echo "4. 验证文档内容..."

# index.md 应包含导航链接
if ! grep -q "文档导航" .rick/wiki/index.md; then
    echo "❌ index.md 缺少文档导航"
    exit 1
fi
echo "✅ index.md 包含文档导航"

# architecture.md 应包含系统架构
if ! grep -q "系统架构" .rick/wiki/architecture.md; then
    echo "❌ architecture.md 缺少系统架构"
    exit 1
fi
echo "✅ architecture.md 包含系统架构"

# core-concepts.md 应包含 Context Loop
if ! grep -q "Context Loop" .rick/wiki/core-concepts.md; then
    echo "❌ core-concepts.md 缺少 Context Loop"
    exit 1
fi
echo "✅ core-concepts.md 包含 Context Loop"

# core-concepts.md 应包含 DAG
if ! grep -q "DAG" .rick/wiki/core-concepts.md; then
    echo "❌ core-concepts.md 缺少 DAG 解释"
    exit 1
fi
echo "✅ core-concepts.md 包含 DAG 解释"

echo ""

# 5. 统计信息
echo "5. 统计信息..."
total_lines=$(wc -l .rick/wiki/*.md .rick/wiki/modules/*.md 2>/dev/null | tail -1 | awk '{print $1}')
echo "📊 总行数: $total_lines"

total_files=$(find .rick/wiki -name "*.md" | wc -l)
echo "📊 总文件数: $total_files"

echo ""
echo "=== 验证完成 ✅ ==="
echo ""
echo "Wiki 知识库已成功创建，包含："
echo "  - 核心文档: 3 个"
echo "  - 模块文档: 8 个"
echo "  - 总行数: $total_lines"
echo ""
echo "访问 .rick/wiki/index.md 开始浏览"
