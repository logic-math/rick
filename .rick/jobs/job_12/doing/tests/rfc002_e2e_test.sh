#!/usr/bin/env bash
# E2E verification script for RFC-002 (skills/tools separation)
# Covers all 4 KRs of job_12 OKR

set -euo pipefail

PASS=0
FAIL=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../../.." && pwd)"

cd "$PROJECT_ROOT"

assert_pass() {
    local name="$1"
    PASS=$((PASS + 1))
    echo "PASS: $name"
}

assert_fail() {
    local name="$1"
    local detail="$2"
    FAIL=$((FAIL + 1))
    echo "FAIL: $name — $detail"
}

# KR1: tools/ contains exactly 5 .py files migrated from .rick/skills/
echo "=== KR1: tools/ 目录包含 5 个 .py 工具脚本 ==="

# Assertion 1: tools/*.py count == 5
count=$(ls tools/*.py 2>/dev/null | wc -l | tr -d ' ')
if [ "$count" -eq 5 ]; then
    assert_pass "断言1: tools/*.py 数量 == 5 (actual: $count)"
else
    assert_fail "断言1: tools/*.py 数量 == 5" "actual: $count"
fi

# Assertion 2: tools/build_and_get_rick_bin.py returns JSON with rick_bin field
result=$(python3 tools/build_and_get_rick_bin.py 2>&1)
if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'bin_path' in d or 'rick_bin' in d" 2>/dev/null; then
    assert_pass "断言2: build_and_get_rick_bin.py 返回 JSON 含 bin_path/rick_bin 字段"
elif echo "$result" | grep -q '"bin_path"'; then
    assert_pass "断言2: build_and_get_rick_bin.py 返回 JSON 含 bin_path 字段"
else
    assert_fail "断言2: build_and_get_rick_bin.py 返回 JSON 含 bin_path/rick_bin 字段" "output: $result"
fi

# KR2: .rick/skills/ only contains .md files; index.md has non-empty trigger column
echo ""
echo "=== KR2: .rick/skills/ 只含 .md 文件，index.md 触发场景列非空 ==="

# Assertion 3: no .py files in .rick/skills/
py_files=$(ls .rick/skills/*.py 2>/dev/null || true)
if [ -z "$py_files" ]; then
    assert_pass "断言3: .rick/skills/ 无 .py 文件"
else
    assert_fail "断言3: .rick/skills/ 无 .py nd: $py_files"
fi

# Assertion 4: index.md contains 3-column table header
if grep -q "| Skill | 描述 | 触发场景 |" .rick/skills/index.md; then
    assert_pass "断言4: index.md 含三列表格 (Skill | 描述 | 触发场景)"
else
    assert_fail "断言4: index.md 含三列表格 (Skill | 描述 | 触发场景)" "header not found"
fi

# Assertion 5: no empty trigger column (no "| |" pattern in data rows)
# Extract table rows (lines starting with | and containing skill name)
empty_trigger=$(grep "^|" .rick/skills/index.md | grep -v "^| Skill\|^|---" | grep "| |$" || true)
if [ -z "$empty_trigger" ]; then
    assert_pass "断言5: index.md 无空触发场景列"
else
    assert_fail "断言5: index.md 无空触发场景列" "empty rows: $empty_trigger"
fi

# KR3: rick doing --dry-run output: tools section non-empty, skills section shows .md names
echo ""
echo "=== KR3: dry-run 输出 tools section 非空，skills section 显示 .md skill 名称 ==="

dry_run_output=$(bin/rick doing job_12 --dry-run 2>&1)

# Assertion 6: dry-run output contains tools/ path (tools section non-empty)
if echo "$dry_run_output" | grep -q "tools/"; then
    assert_pass "断言6: dry-run 输出含 tools/ 字样（tools section 非空）"
else
    assert_fail "断言6: dry-run 输出含 tools/ 字样（tools section 非空）" "tools/ not found in output"
fi

# Assertion 7: dry-run output contains .md skill name in skills section
# Extract skills section content
skills_section=$(echo "$dry_run_output" | awk '/^## 可用的项目 Skills/{found=1} found && /^## 可用的项目 Tools/{found=0} found{print}')
if echo "$skills_section" | grep -q "\.md"; then
    assert_pass "断言7: dry-run skills section 含 .md skill 名称"
else
    assert_fail "断言7: dry-run skills section 含 .md skill 名称" "no .md found in skills section"
fi

# Assertion 8: dry-run skills section does not contain .py entries
py_in_skills=$(echo "$skills_section" | grep "\.py" || true)
if [ -z "$py_in_skills" ]; then
    assert_pass "断言8: dry-run skills section 不含 .py 条目"
else
    assert_fail "断言8: dry-run skills section 不含 .py 条目" "found .py in skills section: $py_in_skills"
fi

# KR4: learning template clearly distinguishes tools (.py) and skills (.md)
echo ""
echo "=== KR4: learning 模板明确区分 tools(.py) 和 skills(.md) ==="

# Assertion 9: learning.md does NOT contain skills/*.py (old format)
old_format=$(grep "skills/\*\.py" internal/prompt/templates/learning.md || true)
if [ -z "$old_format" ]; then
    assert_pass "断言9: learning.md 不含旧格式 skills/*.py"
else
    assert_fail "断言9: learning.md 不含旧格式 skills/*.py" "found: $old_format"
fi

# Assertion 10: learning.md contains both tools/*.py AND skills/*.md
tools_py=$(grep "tools/\*\.py" internal/prompt/templates/learning.md || true)
skills_md=$(grep "skills/\*\.md" internal/prompt/templates/learning.md || true)
if [ -n "$tools_py" ] && [ -n "$skills_md" ]; then
    assert_pass "断言10: learning.md 含 tools/*.py 且含 skills/*.md"
else
    detail=""
    [ -z "$tools_py" ] && detail="${detail}missing tools/*.py; "
    [ -z "$skills_md" ] && detail="${detail}missing skills/*.md"
    assert_fail "断言10: learning.md 含 tools/*.py 且含 skills/*.md" "$detail"
fi

echo ""
echo "=== 结果汇总 ==="
echo "PASS: $PASS / $((PASS + FAIL))"
echo "FAIL: $FAIL / $((PASS + FAIL))"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
