#!/usr/bin/env python3
"""task1 验收测试：spec 规范文档（.rick/domain/spec.md）与 domain 索引。

覆盖测试方法中的三类断言（前置条件/输入/操作/预期四要素已核对）：
1. 正常路径：.rick/domain/spec.md 存在 + .rick/domain/README.md 索引含 spec.md 行
2. 边界（四要素齐备）：模块边界 / 职责 / 接口契约 / 验收标准 四关键词各自命中
3. 异常（验收标准可检索 + 无变量泄漏）：
   - '功能等价' 命中 >= 1
   - '{{' 命中 == 0（无模板变量泄漏）
   - 'dry-run|go test|集成测试' 命中 >= 1（可操作判据被枚举）

本脚本只读不写，幂等；仅向 stdout 输出一行 JSON。
"""
import json
import os
import re
import sys


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task1.py，向上 5 层即仓库根。
    """
    d = os.path.dirname(os.path.abspath(start_file))

    # 方法 1：向上查找 .git 目录
    probe = d
    while True:
        if os.path.isdir(os.path.join(probe, '.git')):
            return probe
        parent = os.path.dirname(probe)
        if parent == probe:
            break
        probe = parent

    # 方法 2：回退到固定相对层级（tests -> doing -> job_35 -> jobs -> .rick -> root）
    for _ in range(5):
        d = os.path.dirname(d)
    return d


def read_file(path):
    """读取文件文本，失败返回 None。"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return f.read()
    except Exception:
        return None


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    domain_dir = os.path.join(repo_root, '.rick', 'domain')
    spec_path = os.path.join(domain_dir, 'spec.md')
    domain_readme_path = os.path.join(domain_dir, 'README.md')

    # ---------- 正常路径 ----------

    # 断言 1：.rick/domain/spec.md 存在（对应 `test -f .rick/domain/spec.md` 返回 0）
    spec_exists = os.path.isfile(spec_path)
    if not spec_exists:
        errors.append('.rick/domain/spec.md does not exist')

    # 断言 2：.rick/domain/README.md 索引含 spec.md 行
    readme_content = read_file(domain_readme_path)
    if readme_content is None:
        errors.append('.rick/domain/README.md does not exist or is not readable')
    elif 'spec.md' not in readme_content:
        errors.append('.rick/domain/README.md index missing spec.md line')

    # ---------- 内容断言（仅在 spec.md 存在时可执行） ----------
    if spec_exists:
        spec_content = read_file(spec_path)
        if spec_content is None:
            errors.append('Failed to read .rick/domain/spec.md')
        else:
            # 边界：四要素齐备 —— 四关键词各自命中，而非合计 >= 1
            four_elements = ['模块边界', '职责', '接口契约', '验收标准']
            for kw in four_elements:
                if kw not in spec_content:
                    errors.append(
                        ".rick/domain/spec.md missing required element keyword: '%s'" % kw)

            # 异常：验收标准可检索 —— '功能等价' 命中 >= 1
            if spec_content.count('功能等价') < 1:
                errors.append(
                    ".rick/domain/spec.md missing '功能等价' acceptance criterion "
                    "(>= 1 required)")

            # 异常：无变量泄漏 —— '{{' 命中 == 0
            if spec_content.count('{{') != 0:
                errors.append(
                    ".rick/domain/spec.md contains leaked template variable '{{' "
                    "(0 required)")

            # 异常：可操作判据被枚举 —— 'dry-run|go test|集成测试' 命中 >= 1
            if not re.search(r'dry-run|go test|集成测试', spec_content):
                errors.append(
                    ".rick/domain/spec.md missing actionable acceptance criteria "
                    "(dry-run / go test / 集成测试)")

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
