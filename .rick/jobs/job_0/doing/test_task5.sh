#!/bin/bash
# Test script for task5: 提取可复用技能并创建 Skills 库

set -e

echo "=== 测试任务5: 提取可复用技能并创建 Skills 库 ==="

# 测试1: 验证 Skills 目录结构
echo "✓ 测试1: 验证 Skills 目录结构"
if [ ! -d ".rick/skills" ]; then
    echo "✗ 失败: .rick/skills 目录不存在"
    exit 1
fi
echo "  ✓ .rick/skills/ 目录存在"

# 测试2: 验证 index.md 存在
echo "✓ 测试2: 验证 index.md"
if [ ! -f ".rick/skills/index.md" ]; then
    echo "✗ 失败: index.md 不存在"
    exit 1
fi

# 检查 index.md 内容
if ! grep -q "技能分类" ".rick/skills/index.md"; then
    echo "✗ 失败: index.md 缺少技能分类"
    exit 1
fi

if ! grep -q "技能索引" ".rick/skills/index.md"; then
    echo "✗ 失败: index.md 缺少技能索引"
    exit 1
fi

echo "  ✓ index.md 包含完整导航"

# 测试3: 验证至少包含 5 个技能目录
echo "✓ 测试3: 验证技能数量"
skill_count=$(find .rick/skills -maxdepth 1 -type d ! -name skills | wc -l | tr -d ' ')
# 减去1是因为 find 会包含 .rick/skills 自身
skill_count=$((skill_count - 1))

if [ "$skill_count" -lt 5 ]; then
    echo "✗ 失败: 技能数量不足 ($skill_count < 5)"
    exit 1
fi
echo "  ✓ 包含 $skill_count 个技能"

# 测试4: 验证每个技能的文档结构
echo "✓ 测试4: 验证技能文档结构"
failed_skills=()

for skill_dir in .rick/skills/*/; do
    skill=$(basename "$skill_dir")

    # 跳过非目录文件
    if [ ! -d "$skill_dir" ]; then
        continue
    fi

    echo "  检查技能: $skill"

    # 检查 description.md
    if [ ! -f "$skill_dir/description.md" ]; then
        echo "    ✗ 缺少 description.md"
        failed_skills+=("$skill")
        continue
    fi

    # 检查 description.md 长度（至少200字，约50行）
    desc_lines=$(wc -l < "$skill_dir/description.md" | tr -d ' ')
    if [ "$desc_lines" -lt 50 ]; then
        echo "    ✗ description.md 内容不足 ($desc_lines 行)"
        failed_skills+=("$skill")
        continue
    fi

    # 检查 implementation.md
    if [ ! -f "$skill_dir/implementation.md" ]; then
        echo "    ✗ 缺少 implementation.md"
        failed_skills+=("$skill")
        continue
    fi

    # 检查 implementation.md 包含代码示例
    if ! grep -q '```' "$skill_dir/implementation.md"; then
        echo "    ✗ implementation.md 缺少代码示例"
        failed_skills+=("$skill")
        continue
    fi

    # 检查 examples 目录
    if [ ! -d "$skill_dir/examples" ]; then
        echo "    ✗ 缺少 examples 目录"
        failed_skills+=("$skill")
        continue
    fi

    # 检查至少有1个案例
    example_count=$(find "$skill_dir/examples" -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$example_count" -lt 1 ]; then
        echo "    ✗ 缺少实际应用案例"
        failed_skills+=("$skill")
        continue
    fi

    echo "    ✓ 文档结构完整"
done

if [ ${#failed_skills[@]} -gt 0 ]; then
    echo "✗ 失败: 以下技能文档不完整:"
    for skill in "${failed_skills[@]}"; do
        echo "  - $skill"
    done
    exit 1
fi

echo "  ✓ 所有技能文档结构完整"

# 测试5: 验证技能与项目代码一致
echo "✓ 测试5: 验证技能与项目代码一致"

# 检查 DAG 拓扑排序技能是否引用了实际代码
if ! grep -q "internal/executor/topological.go" ".rick/skills/dag-topological-sort/description.md"; then
    echo "✗ 失败: DAG 技能未引用实际代码"
    exit 1
fi

# 检查重试模式是否引用了实际代码
if ! grep -q "internal/executor/retry.go" ".rick/skills/retry-pattern/description.md"; then
    echo "✗ 失败: 重试模式未引用实际代码"
    exit 1
fi

echo "  ✓ 技能文档与项目代码一致"

# 测试6: 验证技能具有可复用性
echo "✓ 测试6: 验证技能可复用性"

# 检查 description.md 是否包含使用场景
for skill_dir in .rick/skills/*/; do
    if [ ! -f "$skill_dir/description.md" ]; then
        continue
    fi

    if ! grep -q "使用场景" "$skill_dir/description.md"; then
        echo "✗ 失败: $(basename "$skill_dir") 缺少使用场景"
        exit 1
    fi
done

echo "  ✓ 所有技能包含使用场景"

# 测试7: 验证技能具有通用性
echo "✓ 测试7: 验证技能通用性"

# 检查是否包含优缺点分析
for skill_dir in .rick/skills/*/; do
    if [ ! -f "$skill_dir/description.md" ]; then
        continue
    fi

    if ! grep -q "优点\|优势" "$skill_dir/description.md"; then
        echo "✗ 失败: $(basename "$skill_dir") 缺少优点分析"
        exit 1
    fi
done

echo "  ✓ 所有技能包含优缺点分析"

# 最终统计
echo ""
echo "=== 测试总结 ==="
echo "✓ Skills 目录结构: 已创建"
echo "✓ 技能数量: $skill_count 个"
echo "✓ 文档结构: 完整"
echo "✓ 代码一致性: 通过"
echo "✓ 可复用性: 通过"
echo "✓ 通用性: 通过"
echo ""
echo "🎉 所有测试通过!"
exit 0
