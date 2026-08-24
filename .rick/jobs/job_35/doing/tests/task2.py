#!/usr/bin/env python3
"""task2 验收测试：rick 第一份 spec（.rick/domain/rick-spec.md）。

覆盖测试方法中的三类断言（前置条件/输入/操作/预期四要素已核对）：
1. 正常路径：前置条件 .rick/domain/spec.md（task1）存在 + .rick/domain/rick-spec.md 已写
   （对应 `test -f .rick/domain/rick-spec.md` 返回 0）
2. 边界（5 模块 + env 四职责 + 四层架构覆盖）：
   - 5 模块名各自命中：cli handler env builder runtime
   - env 四职责各自命中：安装 生态扩展 定制 就绪
   - 四层架构关键词各自命中：第一层 第二层 第三层 第四层 CLI TUI WEB-UI
3. 异常（与 RFC 一致 + 无变量泄漏 + 扩展 seam）：
   - dag / 门禁 / sessionID 各自命中（与 RFC 一致）
   - RuntimeBuilder / RuntimeEnv / runtime 三 seam 各自命中
   - '{{' 命中 == 0（无模板变量泄漏）

本脚本只读不写，幂等；仅向 stdout 输出一行 JSON。
"""
import json
import os
import sys


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task2.py，向上 5 层即仓库根。
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
    rick_spec_path = os.path.join(domain_dir, 'rick-spec.md')

    # ---------- 正常路径 ----------

    # 断言 1：前置条件 .rick/domain/spec.md（task1）存在
    if not os.path.isfile(spec_path):
        errors.append('precondition .rick/domain/spec.md (task1) does not exist')

    # 断言 2：.rick/domain/rick-spec.md 已写（对应 `test -f .rick/domain/rick-spec.md` 返回 0）
    rick_spec_exists = os.path.isfile(rick_spec_path)
    if not rick_spec_exists:
        errors.append('.rick/domain/rick-spec.md does not exist')

    # ---------- 内容断言（仅在 rick-spec.md 存在时可执行） ----------
    if rick_spec_exists:
        content = read_file(rick_spec_path)
        if content is None:
            errors.append('Failed to read .rick/domain/rick-spec.md')
        else:
            # ---------- 边界：5 模块名各自命中 ----------
            # 对应 `for w in cli handler env builder runtime; do grep -q "$w" ...; done`
            for kw in ['cli', 'handler', 'env', 'builder', 'runtime']:
                if kw not in content:
                    errors.append(
                        ".rick/domain/rick-spec.md missing module name: '%s'" % kw)

            # ---------- 边界：env 四职责各自命中 ----------
            # 对应 `for w in 安装 生态扩展 定制 就绪; do grep -q "$w" ...; done`
            for kw in ['安装', '生态扩展', '定制', '就绪']:
                if kw not in content:
                    errors.append(
                        ".rick/domain/rick-spec.md missing env responsibility keyword: '%s'" % kw)

            # ---------- 边界：四层架构关键词各自命中 ----------
            # 对应 `for w in 第一层 第二层 第三层 第四层 CLI TUI WEB-UI; do grep -q "$w" ...; done`
            for kw in ['第一层', '第二层', '第三层', '第四层', 'CLI', 'TUI', 'WEB-UI']:
                if kw not in content:
                    errors.append(
                        ".rick/domain/rick-spec.md missing 4-layer keyword: '%s'" % kw)

            # ---------- 异常：与 RFC 一致（dag / 门禁 / sessionID） ----------
            # 对应 `for w in dag 门禁 sessionID; do grep -q "$w" ...; done`
            for kw in ['dag', '门禁', 'sessionID']:
                if kw not in content:
                    errors.append(
                        ".rick/domain/rick-spec.md missing RFC keyword: '%s'" % kw)

            # ---------- 异常：扩展 seam（RuntimeBuilder / RuntimeEnv / runtime） ----------
            # 对应 `for w in RuntimeBuilder RuntimeEnv runtime; do grep -q "$w" ...; done`
            for kw in ['RuntimeBuilder', 'RuntimeEnv', 'runtime']:
                if kw not in content:
                    errors.append(
                        ".rick/domain/rick-spec.md missing extension seam: '%s'" % kw)

            # ---------- 异常：无变量泄漏（'{{' 命中 == 0） ----------
            # 对应 `grep -c '{{' .rick/domain/rick-spec.md`（= 0）
            if content.count('{{') != 0:
                errors.append(
                    ".rick/domain/rick-spec.md contains leaked template variable '{{' "
                    "(0 required)")

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
