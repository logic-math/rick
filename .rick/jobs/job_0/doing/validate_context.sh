#!/bin/bash

# Rick 项目全局上下文完整性验证脚本
# 生成时间: 2026-03-14

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 统计变量
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
}

check_file_exists() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    local file=$1
    local desc=$2

    if [ -f "$file" ]; then
        log_pass "文件存在: $desc ($file)"
        return 0
    else
        log_error "文件缺失: $desc ($file)"
        return 1
    fi
}

check_file_not_empty() {
    local file=$1
    local desc=$2

    if [ ! -f "$file" ]; then
        return 1
    fi

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    local size=$(wc -c < "$file" | tr -d ' ')

    if [ "$size" -gt 100 ]; then
        log_pass "文件非空: $desc ($size bytes)"
        return 0
    else
        log_error "文件过小: $desc ($size bytes)"
        return 1
    fi
}

check_markdown_structure() {
    local file=$1
    local desc=$2

    if [ ! -f "$file" ]; then
        return 1
    fi

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    # 检查是否包含标题
    if grep -q "^#" "$file"; then
        log_pass "Markdown 结构: $desc 包含标题"
        return 0
    else
        log_warning "Markdown 结构: $desc 缺少标题"
        return 1
    fi
}

check_links() {
    local file=$1
    local desc=$2

    if [ ! -f "$file" ]; then
        return 1
    fi

    # 提取所有 Markdown 链接
    local links=$(grep -o '\[.*\](.*)' "$file" | grep -o '(.*)'  | tr -d '()' | grep -v '^http' | grep -v '^#')

    if [ -z "$links" ]; then
        return 0
    fi

    local broken=0
    while IFS= read -r link; do
        if [ -n "$link" ]; then
            local base_dir=$(dirname "$file")
            local full_path="$base_dir/$link"

            if [ ! -f "$full_path" ] && [ ! -d "$full_path" ]; then
                log_warning "断链: $desc -> $link"
                broken=$((broken + 1))
            fi
        fi
    done <<< "$links"

    if [ $broken -eq 0 ]; then
        log_pass "链接检查: $desc 所有链接有效"
    fi
}

echo "======================================"
echo "Rick 项目全局上下文完整性验证"
echo "======================================"
echo ""

# 1. 检查核心文档文件
log_info "1. 检查核心文档文件..."
check_file_exists ".rick/OKR.md" "OKR 文档"
check_file_exists ".rick/SPEC.md" "SPEC 文档"
check_file_exists ".rick/skills/index.md" "Skills 索引"
check_file_exists ".rick/wiki/index.md" "Wiki 索引"
check_file_exists "README.md" "项目 README"
echo ""

# 2. 检查文件完整性
log_info "2. 检查文件完整性（非空）..."
check_file_not_empty ".rick/OKR.md" "OKR 文档"
check_file_not_empty ".rick/SPEC.md" "SPEC 文档"
check_file_not_empty ".rick/skills/index.md" "Skills 索引"
check_file_not_empty ".rick/wiki/index.md" "Wiki 索引"
echo ""

# 3. 检查 Markdown 结构
log_info "3. 检查 Markdown 结构..."
check_markdown_structure ".rick/OKR.md" "OKR 文档"
check_markdown_structure ".rick/SPEC.md" "SPEC 文档"
check_markdown_structure ".rick/skills/index.md" "Skills 索引"
check_markdown_structure ".rick/wiki/index.md" "Wiki 索引"
echo ""

# 4. 检查 Skills 目录
log_info "4. 检查 Skills 目录..."
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
skill_count=$(find .rick/skills -type d -mindepth 1 -maxdepth 1 | wc -l | tr -d ' ')
if [ "$skill_count" -ge 5 ]; then
    log_pass "Skills 数量: $skill_count 个技能"
else
    log_warning "Skills 数量: 仅 $skill_count 个技能（预期 >= 5）"
fi

# 检查每个 skill 的文档
for skill_dir in .rick/skills/*/; do
    if [ -d "$skill_dir" ]; then
        skill_name=$(basename "$skill_dir")
        check_file_exists "${skill_dir}description.md" "Skill: $skill_name (description)"
        check_file_exists "${skill_dir}implementation.md" "Skill: $skill_name (implementation)"
        check_file_not_empty "${skill_dir}description.md" "Skill: $skill_name (description)"
    fi
done
echo ""

# 5. 检查 Wiki 模块
log_info "5. 检查 Wiki 模块..."
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
wiki_count=$(find .rick/wiki -type f -name "*.md" | wc -l | tr -d ' ')
if [ "$wiki_count" -ge 8 ]; then
    log_pass "Wiki 文档数量: $wiki_count 个文档"
else
    log_warning "Wiki 文档数量: 仅 $wiki_count 个文档（预期 >= 8）"
fi

# 检查核心模块文档
check_file_exists ".rick/wiki/modules/infrastructure.md" "Wiki: Infrastructure 模块"
check_file_exists ".rick/wiki/modules/parser.md" "Wiki: Parser 模块"
check_file_exists ".rick/wiki/modules/dag_executor.md" "Wiki: DAG Executor 模块"
check_file_exists ".rick/wiki/modules/prompt_manager.md" "Wiki: Prompt Manager 模块"
check_file_exists ".rick/wiki/modules/cli_commands.md" "Wiki: CLI Commands 模块"
check_file_exists ".rick/wiki/modules/workspace.md" "Wiki: Workspace 模块"
echo ""

# 6. 检查链接有效性
log_info "6. 检查文档链接..."
check_links ".rick/OKR.md" "OKR 文档"
check_links ".rick/SPEC.md" "SPEC 文档"
check_links ".rick/skills/index.md" "Skills 索引"
check_links ".rick/wiki/index.md" "Wiki 索引"
echo ""

# 7. 检查代码示例
log_info "7. 检查代码示例..."
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
code_blocks=$(grep -r '```go' .rick/skills/ .rick/wiki/ | wc -l | tr -d ' ')
if [ "$code_blocks" -ge 10 ]; then
    log_pass "代码示例: $code_blocks 个 Go 代码块"
else
    log_warning "代码示例: 仅 $code_blocks 个 Go 代码块（预期 >= 10）"
fi
echo ""

# 8. 检查关键词覆盖
log_info "8. 检查关键词覆盖..."
keywords=("Rick" "Context" "AI Coding" "DAG" "Prompt" "Task" "Workspace" "Parser")

for keyword in "${keywords[@]}"; do
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    count=$(grep -r "$keyword" .rick/*.md .rick/skills/ .rick/wiki/ 2>/dev/null | wc -l | tr -d ' ')
    if [ "$count" -ge 3 ]; then
        log_pass "关键词覆盖: '$keyword' 出现 $count 次"
    else
        log_warning "关键词覆盖: '$keyword' 仅出现 $count 次"
    fi
done
echo ""

# 9. 生成统计报告
echo "======================================"
echo "验证统计"
echo "======================================"
echo "总检查项: $TOTAL_CHECKS"
echo -e "${GREEN}通过: $PASSED_CHECKS${NC}"
echo -e "${RED}失败: $FAILED_CHECKS${NC}"
echo -e "${YELLOW}警告: $WARNINGS${NC}"
echo ""

# 计算成功率
if [ $TOTAL_CHECKS -gt 0 ]; then
    success_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    echo "成功率: $success_rate%"

    if [ $success_rate -ge 90 ]; then
        echo -e "${GREEN}✓ 验证通过！全局上下文完整性良好。${NC}"
        exit 0
    elif [ $success_rate -ge 70 ]; then
        echo -e "${YELLOW}⚠ 验证基本通过，但存在一些问题需要关注。${NC}"
        exit 0
    else
        echo -e "${RED}✗ 验证失败！全局上下文存在严重问题。${NC}"
        exit 1
    fi
fi
