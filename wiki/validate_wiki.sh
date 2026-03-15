#!/bin/bash
# Wiki 文档验证脚本

set -e

WIKI_DIR="wiki"
ERRORS=0
WARNINGS=0

echo "==================================="
echo "Rick CLI Wiki 验证报告"
echo "==================================="
echo ""

# 1. 文档统计
echo "📊 文档统计"
echo "-----------------------------------"
DOC_COUNT=$(find "$WIKI_DIR" -name "*.md" | wc -l | tr -d ' ')
LINE_COUNT=$(find "$WIKI_DIR" -name "*.md" -exec wc -l {} + | tail -1 | awk '{print $1}')
MERMAID_COUNT=$(grep -r '```mermaid' "$WIKI_DIR" | wc -l | tr -d ' ')

echo "文档总数: $DOC_COUNT"
echo "总行数: $LINE_COUNT"
echo "Mermaid 图表数: $MERMAID_COUNT"
echo ""

# 2. 检查必需文档是否存在
echo "📁 文档完整性检查"
echo "-----------------------------------"
REQUIRED_DOCS=(
    "README.md"
    "architecture.md"
    "runtime-flow.md"
    "dag-execution.md"
    "prompt-system.md"
    "testing.md"
    "installation.md"
    "modules/cmd.md"
    "modules/workspace.md"
    "modules/parser.md"
    "modules/executor.md"
    "modules/prompt.md"
    "modules/git.md"
    "modules/config.md"
)

for doc in "${REQUIRED_DOCS[@]}"; do
    if [ -f "$WIKI_DIR/$doc" ]; then
        echo "✅ $doc"
    else
        echo "❌ $doc (缺失)"
        ERRORS=$((ERRORS + 1))
    fi
done
echo ""

# 3. 检查内部链接
echo "🔗 内部链接检查"
echo "-----------------------------------"
BROKEN_LINKS=0
while IFS= read -r file; do
    # 提取 Markdown 链接 [text](path)
    grep -oE '\[([^]]+)\]\(([^)]+)\)' "$file" | while IFS= read -r link; do
        # 提取路径部分
        path=$(echo "$link" | sed -E 's/.*\(([^)]+)\).*/\1/')

        # 跳过外部链接和锚点
        if [[ "$path" =~ ^https?:// ]] || [[ "$path" =~ ^# ]]; then
            continue
        fi

        # 计算绝对路径
        dir=$(dirname "$file")
        target="$dir/$path"

        # 检查文件是否存在
        if [ ! -f "$target" ] && [ ! -d "$target" ]; then
            echo "⚠️  断链: $file -> $path"
            WARNINGS=$((WARNINGS + 1))
            BROKEN_LINKS=$((BROKEN_LINKS + 1))
        fi
    done
done < <(find "$WIKI_DIR" -name "*.md")

if [ $BROKEN_LINKS -eq 0 ]; then
    echo "✅ 所有内部链接正常"
else
    echo "⚠️  发现 $BROKEN_LINKS 个可能的断链"
fi
echo ""

# 4. 检查 Mermaid 语法
echo "📊 Mermaid 图表语法检查"
echo "-----------------------------------"
MERMAID_ERRORS=0
while IFS= read -r file; do
    # 查找 mermaid 代码块
    awk '/```mermaid/,/```/' "$file" | while IFS= read -r line; do
        # 检查常见语法错误
        if [[ "$line" =~ ^[[:space:]]*graph[[:space:]]+(TB|TD|BT|RL|LR) ]]; then
            echo "✅ $file: 发现有效的 graph 定义"
        elif [[ "$line" =~ ^[[:space:]]*flowchart[[:space:]]+(TB|TD|BT|RL|LR) ]]; then
            echo "✅ $file: 发现有效的 flowchart 定义"
        elif [[ "$line" =~ ^[[:space:]]*sequenceDiagram ]]; then
            echo "✅ $file: 发现有效的 sequenceDiagram 定义"
        elif [[ "$line" =~ ^[[:space:]]*classDiagram ]]; then
            echo "✅ $file: 发现有效的 classDiagram 定义"
        fi
    done
done < <(grep -l '```mermaid' "$WIKI_DIR"/*.md "$WIKI_DIR"/**/*.md 2>/dev/null || true)

echo "✅ Mermaid 图表基本语法检查完成"
echo ""

# 5. 检查代码引用
echo "💻 代码示例检查"
echo "-----------------------------------"
CODE_REFS=0
while IFS= read -r file; do
    # 统计 Go 代码块
    go_blocks=$(grep -c '```go' "$file" 2>/dev/null || echo "0")
    bash_blocks=$(grep -c '```bash' "$file" 2>/dev/null || echo "0")
    json_blocks=$(grep -c '```json' "$file" 2>/dev/null || echo "0")

    # 确保变量是数字
    go_blocks=${go_blocks:-0}
    bash_blocks=${bash_blocks:-0}
    json_blocks=${json_blocks:-0}

    total=$((go_blocks + bash_blocks + json_blocks))
    if [ $total -gt 0 ]; then
        echo "✅ $file: $total 个代码示例 (Go: $go_blocks, Bash: $bash_blocks, JSON: $json_blocks)"
        CODE_REFS=$((CODE_REFS + total))
    fi
done < <(find "$WIKI_DIR" -name "*.md")

echo ""
echo "总计: $CODE_REFS 个代码示例"
echo ""

# 6. 文档风格检查
echo "📝 文档风格检查"
echo "-----------------------------------"
STYLE_ISSUES=0
while IFS= read -r file; do
    # 检查标题层级（不应该跳级）
    prev_level=0
    while IFS= read -r line; do
        if [[ "$line" =~ ^(#+)[[:space:]] ]]; then
            level=${#BASH_REMATCH[1]}
            if [ $prev_level -ne 0 ] && [ $level -gt $((prev_level + 1)) ]; then
                echo "⚠️  $file: 标题层级跳跃 (从 h$prev_level 到 h$level)"
                WARNINGS=$((WARNINGS + 1))
                STYLE_ISSUES=$((STYLE_ISSUES + 1))
            fi
            prev_level=$level
        fi
    done < "$file"
done < <(find "$WIKI_DIR" -name "*.md")

if [ $STYLE_ISSUES -eq 0 ]; then
    echo "✅ 文档风格检查通过"
else
    echo "⚠️  发现 $STYLE_ISSUES 个风格问题"
fi
echo ""

# 7. 总结
echo "==================================="
echo "验证总结"
echo "==================================="
echo "文档数量: $DOC_COUNT"
echo "总行数: $LINE_COUNT"
echo "Mermaid 图表: $MERMAID_COUNT"
echo "代码示例: $CODE_REFS"
echo "错误: $ERRORS"
echo "警告: $WARNINGS"
echo ""

if [ $ERRORS -eq 0 ]; then
    echo "✅ Wiki 文档验证通过！"
    exit 0
else
    echo "❌ Wiki 文档验证失败，请修复错误"
    exit 1
fi
