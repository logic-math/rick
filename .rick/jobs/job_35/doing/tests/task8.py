#!/usr/bin/env python3
r"""task8 验收测试：做薄 cutover（dag 调度/门禁下沉 pi + 删除冗余 Go 包）。

覆盖测试方法的三类断言（前置条件/输入/操作/预期四要素已核对）：

1. 正常路径：`doing job_N --dry-run` 输出含 pi 编排语法（workflowScript | runs.run）。
   前置 = 一个含 3 task（task2 依赖 task1）、无 tasks.json 的 job；输入 = doing job_N --dry-run；
   预期 = grep -cE 'workflowScript|runs\.run' ≥ 1（doing 提示词含 pi 编排语法）。

2. 边界（dag 拓扑 + 跳过已完成 + 门禁脚本 + 冗余包已删）：
   - `test -f .rick/skills/rick-gates/helper.py`（门禁脚本已部署，且非占位骨架）；
   - `for d in executor parser actpath logging git agent` → `internal/$d` 全部不存在（6 冗余包已删）；
   - `internal/cmd/tools_doing_check.go` / `tools_plan_check.go` 已删；
   - 提取 `runs.run('taskN'` 编排序列：全 pending → task1 在 task2 之前；task1 已 success → 序列不含 task1。

3. 异常（门禁语义不丢 + runtime 签名 + 重试收敛）：
   - `python3 .rick/skills/rick-gates/helper.py <doing_dir>`：success 但 commit_hash 空 → 报
     `missing commit_hash` 且 exit 非 0；tasks.json 不可解析 / 存在 zombie running → exit 非 0；
     合法 doing_dir → exit 0；
   - runtime `Run` 在 fake JSONL 缺 `agent_settled` 时返回 error（未就绪）——通过向
     internal/runtime 临时写入 Go 测试、`go test -run` 后移除，验证真实行为（非 mock 行为）；
   - 门禁检测「workflow 未触发/未完成」→ handler 重试（重新生成只含剩余 pending 的编排，
     上限 max_retries）——源码级断言 handler.Doing 不再依赖 internal/executor/git/parser，
     且保留 MaxRetries 重试上界。

附加结构检查（对齐 task8 关键结果）：
- 6 个模板（learning/learning_loop/gen-skill/gen-loop/dream/ctrl）改写 act-path.md 引用为 runtime trace。

本脚本只读源码 + 跑 go build/go test + 在独立临时 .rick 工作区跑 CLI（产物/夹具均落临时目录，
临时 Go 测试用后即删，不污染仓库），幂等；仅向 stdout 输出一行 JSON。
"""
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 go.mod（Go 项目根标记）；找不到则回退为脚本相对位置向上 6 层：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task8.py。
    """
    d = os.path.dirname(os.path.abspath(start_file))
    probe = d
    while True:
        if os.path.isfile(os.path.join(probe, 'go.mod')):
            return probe
        parent = os.path.dirname(probe)
        if parent == probe:
            break
        probe = parent

    for _ in range(6):
        d = os.path.dirname(d)
    return d


def read_text(path):
    """读取文本文件内容，失败返回 None。"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return f.read()
    except Exception:
        return None


def run(cmd, cwd, timeout=300, env=None):
    """运行子进程，返回 (returncode, stdout, stderr)。"""
    try:
        p = subprocess.run(
            cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env
        )
        return p.returncode, p.stdout or '', p.stderr or ''
    except FileNotFoundError as e:
        return 127, '', str(e)
    except subprocess.TimeoutExpired as e:
        out = e.stdout or ''
        err = (e.stderr or '') + '\n[timeout after %ds]' % timeout
        return 124, out, err


def tail(text, n=1600):
    """截取文本尾部，用于错误信息（避免刷屏）。"""
    if not text:
        return ''
    return text[-n:]


def write_task_md(path, deps, name):
    """写一份符合 parser.ParseTask 的 task*.md（# 依赖关系/任务名称/任务目标/关键结果/测试方法）。"""
    content = (
        "# 依赖关系\n%s\n\n"
        "# 任务名称\n%s\n\n"
        "# 任务目标\n%s\n\n"
        "# 关键结果\n- kr\n\n"
        "# 测试方法\n- tm\n"
    ) % (deps, name, 'goal of ' + name)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)


def write_tasks_json(path, tasks):
    """写一份 tasks.json。tasks 为 list[dict]，dict 直接序列化（可控制 commit_hash 字段有无）。"""
    data = {"version": "1.0", "tasks": tasks}
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)


def extract_orchestrated_task_ids(text):
    """按出现顺序提取 workflowScript 编排中的 task id（runs.run('taskN' / runs.run("taskN"）。"""
    return re.findall(r"runs\.run\(\s*['\"](task\d+)['\"]", text)


# 临时写入 internal/runtime 的 Go 测试：验证 Run 在 fake JSONL 缺 agent_settled 时返回 error。
# 这是真实行为测试（真实 Run 方法 + 真实 fake pi 子进程），非 mock 行为断言。
_TASK8_RUNTIME_GO_TEST = '''package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTask8VerifyRunErrorsWithoutAgentSettled 验证 task8 runtime 契约：
// fake JSONL 流缺少 agent_settled 终止事件（session 未就绪）时，Run 必须返回 error。
// 由 task8 Python 验收测试临时写入，运行后删除。
func TestTask8VerifyRunErrorsWithoutAgentSettled(t *testing.T) {
	tmp := t.TempDir()
	mockPath := filepath.Join(tmp, "mock_pi")
	script := "#!/bin/sh\\necho '{\\"type\\":\\"session\\",\\"id\\":\\"s123\\"}'\\n"
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# context"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := NewPiRuntime(mockPath).Run("", promptFile, nil)
	if err == nil {
		t.Fatal("Run must return an error when agent_settled is missing (session not ready)")
	}
}
'''


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    # =========================================================================
    # 前置：go build -o <tmp>/rick ./cmd/rick（产物落临时目录，不污染 bin/rick）
    # 这也是「6 冗余包删除后调用点全部迁移」的编译级门禁。
    # =========================================================================
    build_dir = tempfile.mkdtemp(prefix='rick_task8_build_')
    bin_path = os.path.join(build_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', bin_path, './cmd/rick'], cwd=repo_root, timeout=300)
    build_ok = rc == 0 and os.path.isfile(bin_path)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
    elif not os.path.isfile(bin_path):
        errors.append('go build ./cmd/rick succeeded but binary not produced at %s' % bin_path)

    # =========================================================================
    # 边界 1：6 冗余包已删（executor/parser/actpath/logging/git/agent）
    # =========================================================================
    deleted_packages = ['executor', 'parser', 'actpath', 'logging', 'git', 'agent']
    still_present = []
    for d in deleted_packages:
        if os.path.isdir(os.path.join(repo_root, 'internal', d)):
            still_present.append(d)
    if still_present:
        errors.append(
            'redundant internal packages still present (must be deleted): %s'
            % ', '.join('internal/' + d for d in still_present)
        )

    # 边界 1b：tools_doing_check.go / tools_plan_check.go 已删（KR4 显式列出）
    for fname in ('tools_doing_check.go', 'tools_plan_check.go'):
        if os.path.exists(os.path.join(repo_root, 'internal', 'cmd', fname)):
            errors.append('internal/cmd/%s still exists (must be deleted)' % fname)

    # 异常（重试收敛）：handler.Doing 不再依赖已删除的 executor/git/parser（迁移到
    # runtime.Run + builder 编排），且保留 MaxRetries 重试上界。这些 import 若残留，
    # 删除包后 go build 会失败（编译级已覆盖）；此处补源码级断言，以便在「build 通过但
    # 逻辑未迁移」时也能 RED。
    doing_src = read_text(os.path.join(repo_root, 'internal', 'handler', 'doing.go'))
    if doing_src is None:
        errors.append('internal/handler/doing.go does not exist')
    else:
        for pkg in ('internal/executor', 'internal/git', 'internal/parser'):
            if ('"github.com/sunquan/rick/' + pkg + '"') in doing_src:
                errors.append(
                    'internal/handler/doing.go still imports %s '
                    '(must migrate to runtime.Run + builder workflowScript)' % pkg
                )
        if 'MaxRetries' not in doing_src:
            errors.append(
                'internal/handler/doing.go missing MaxRetries retry upper bound '
                '(retry convergence not preserved)'
            )

    # =========================================================================
    # 边界 2：门禁脚本已部署且非占位骨架 + 异常：门禁语义不丢（missing commit_hash）
    # =========================================================================
    helper_py = os.path.join(repo_root, '.rick', 'skills', 'rick-gates', 'helper.py')
    if not os.path.isfile(helper_py):
        errors.append('.rick/skills/rick-gates/helper.py does not exist (gate script not deployed)')
    else:
        helper_src = read_text(helper_py) or ''
        # 占位骨架标志：task8 前 helper.py 只是打印 placeholder 并返回 0。
        if 'placeholder' in helper_src and 'missing commit_hash' not in helper_src:
            errors.append(
                '.rick/skills/rick-gates/helper.py is still the placeholder skeleton '
                '(real gate logic not filled in)'
            )

        # 在临时 doing 目录构造门禁夹具，直接跑真实脚本（真实行为，非 mock）。
        gate_fx = tempfile.mkdtemp(prefix='rick_task8_gate_')
        try:
            # 异常 1：success 但 commit_hash 空 → 报 missing commit_hash 且 exit 非 0。
            d_missing = os.path.join(gate_fx, 'missing')
            os.makedirs(d_missing, exist_ok=True)
            write_tasks_json(
                os.path.join(d_missing, 'tasks.json'),
                [{"task_id": "task1", "task_name": "t1", "status": "success"}],
            )
            rc, out, err = run([sys.executable, helper_py, d_missing], cwd=repo_root, timeout=60)
            combined = (out or '') + (err or '')
            if 'missing commit_hash' not in combined:
                errors.append(
                    "rick-gates helper did not report 'missing commit_hash' for success-without-commit_hash "
                    '(exit %d)' % rc
                )
            elif rc == 0:
                errors.append(
                    'rick-gates helper exited 0 for success-without-commit_hash (expected non-zero)'
                )

            # 异常 2：tasks.json 不可解析 → exit 非 0（「可解析」门禁语义不丢）。
            d_bad = os.path.join(gate_fx, 'badjson')
            os.makedirs(d_bad, exist_ok=True)
            with open(os.path.join(d_bad, 'tasks.json'), 'w', encoding='utf-8') as f:
                f.write('{ not valid json')
            rc, _, _ = run([sys.executable, helper_py, d_bad], cwd=repo_root, timeout=60)
            if rc == 0:
                errors.append(
                    'rick-gates helper exited 0 for unparseable tasks.json (expected non-zero)'
                )

            # 异常 3：zombie running → exit 非 0（「无 zombie」门禁语义不丢）。
            d_zombie = os.path.join(gate_fx, 'zombie')
            os.makedirs(d_zombie, exist_ok=True)
            write_tasks_json(
                os.path.join(d_zombie, 'tasks.json'),
                [{"task_id": "task1", "task_name": "t1", "status": "running"}],
            )
            rc, _, _ = run([sys.executable, helper_py, d_zombie], cwd=repo_root, timeout=60)
            if rc == 0:
                errors.append(
                    'rick-gates helper exited 0 for zombie running task (expected non-zero)'
                )

            # 正向：合法 doing_dir（success + commit_hash）→ exit 0。
            d_ok = os.path.join(gate_fx, 'ok')
            os.makedirs(d_ok, exist_ok=True)
            write_tasks_json(
                os.path.join(d_ok, 'tasks.json'),
                [{"task_id": "task1", "task_name": "t1", "status": "success",
                  "commit_hash": "abc123def456"}],
            )
            rc, out, err = run([sys.executable, helper_py, d_ok], cwd=repo_root, timeout=60)
            if rc != 0:
                errors.append(
                    'rick-gates helper exited %d for a valid doing dir (expected 0):\n%s'
                    % (rc, tail((out or '') + (err or '')))
                )
        finally:
            shutil.rmtree(gate_fx, ignore_errors=True)

    # =========================================================================
    # 正常路径 + 边界（dag 拓扑 + 跳过已完成）：
    # 独立临时 .rick 工作区，job_1（无 tasks.json = 全 pending）验证编排语法与依赖顺序，
    # job_2（task1 已 success）验证跳过已完成。
    # =========================================================================
    fx = tempfile.mkdtemp(prefix='rick_task8_fx_')
    try:
        plan1 = os.path.join(fx, '.rick', 'jobs', 'job_1', 'plan')
        doing1 = os.path.join(fx, '.rick', 'jobs', 'job_1', 'doing')
        plan2 = os.path.join(fx, '.rick', 'jobs', 'job_2', 'plan')
        doing2 = os.path.join(fx, '.rick', 'jobs', 'job_2', 'doing')
        os.makedirs(plan1, exist_ok=True)
        os.makedirs(doing1, exist_ok=True)
        os.makedirs(plan2, exist_ok=True)
        os.makedirs(doing2, exist_ok=True)

        # 3 task：task2 依赖 task1；task3 无依赖。
        write_task_md(os.path.join(plan1, 'task1.md'), '无', 'task one')
        write_task_md(os.path.join(plan1, 'task2.md'), 'task1', 'task two')
        write_task_md(os.path.join(plan1, 'task3.md'), '无', 'task three')
        write_task_md(os.path.join(plan2, 'task1.md'), '无', 'task one')
        write_task_md(os.path.join(plan2, 'task2.md'), 'task1', 'task two')
        write_task_md(os.path.join(plan2, 'task3.md'), '无', 'task three')

        # job_2：tasks.json 中 task1 已 success（含 commit_hash，与门禁语义一致）。
        write_tasks_json(
            os.path.join(doing2, 'tasks.json'),
            [
                {"task_id": "task1", "task_name": "t1", "status": "success",
                 "commit_hash": "abc123def456"},
                {"task_id": "task2", "task_name": "t2", "status": "pending"},
                {"task_id": "task3", "task_name": "t3", "status": "pending"},
            ],
        )

        if build_ok:
            # ---------- 正常路径：job_1 --dry-run 含 workflowScript | runs.run ----------
            rc, out, err = run([bin_path, 'doing', 'job_1', '--dry-run'], cwd=fx, timeout=120)
            if rc != 0:
                errors.append('rick doing job_1 --dry-run failed (exit %d):\n%s'
                              % (rc, tail(err or out)))
            else:
                count = len(re.findall(r'workflowScript', out)) + len(re.findall(r'runs\.run', out))
                if count < 1:
                    errors.append(
                        "rick doing job_1 --dry-run output lacks pi orchestration syntax "
                        "(expected >=1 of workflowScript|runs.run):\n" + tail(out, 600)
                    )

                # ---------- 边界（dag 拓扑）：全 pending → task1 在 task2 之前 ----------
                ids = extract_orchestrated_task_ids(out)
                if not ids:
                    errors.append(
                        "rick doing job_1 --dry-run: could not extract runs.run('taskN' orchestration "
                        'sequence (expected task1 before task2, all 3 tasks pending):\n' + tail(out, 600)
                    )
                else:
                    if 'task1' not in ids:
                        errors.append('orchestration missing task1 (all pending): %s' % ids)
                    if 'task2' not in ids:
                        errors.append('orchestration missing task2: %s' % ids)
                    if 'task3' not in ids:
                        errors.append('orchestration missing task3: %s' % ids)
                    if 'task1' in ids and 'task2' in ids and ids.index('task1') > ids.index('task2'):
                        errors.append(
                            'dependency order wrong: task2 before task1 (task2 depends on task1): %s'
                            % ids
                        )

            # ---------- 边界（跳过已完成）：job_2（task1 success）→ 序列不含 task1 ----------
            rc, out, err = run([bin_path, 'doing', 'job_2', '--dry-run'], cwd=fx, timeout=120)
            if rc != 0:
                errors.append('rick doing job_2 --dry-run failed (exit %d):\n%s'
                              % (rc, tail(err or out)))
            else:
                ids2 = extract_orchestrated_task_ids(out)
                if not ids2:
                    errors.append(
                        "rick doing job_2 --dry-run: could not extract runs.run('taskN' orchestration "
                        'sequence (expected task1 skipped, task2+task3 orchestrated):\n' + tail(out, 600)
                    )
                else:
                    if 'task1' in ids2:
                        errors.append(
                            'completed task1 (status=success) should be skipped, but appears in '
                            'orchestration: %s' % ids2
                        )
                    if 'task2' not in ids2:
                        errors.append('orchestration missing pending task2 after skip: %s' % ids2)
                    if 'task3' not in ids2:
                        errors.append('orchestration missing pending task3 after skip: %s' % ids2)
        else:
            errors.append('rick binary missing (build failed), skipping doing --dry-run checks')
    finally:
        shutil.rmtree(fx, ignore_errors=True)

    # =========================================================================
    # 异常（runtime 签名）：Run 在 fake JSONL 缺 agent_settled 时返回 error（未就绪）。
    # 向 internal/runtime 临时写入 Go 测试并运行（真实行为，非 mock 行为断言），用后即删。
    # =========================================================================
    runtime_test_path = os.path.join(repo_root, 'internal', 'runtime', 'task8_verify_test.go')
    wrote_test = False
    try:
        if os.path.exists(os.path.join(repo_root, 'internal', 'runtime')):
            with open(runtime_test_path, 'w', encoding='utf-8') as f:
                f.write(_TASK8_RUNTIME_GO_TEST)
            wrote_test = True
            rc, out, err = run(
                [go, 'test', './internal/runtime/',
                 '-run', 'TestTask8VerifyRunErrorsWithoutAgentSettled', '-count=1'],
                cwd=repo_root, timeout=180,
            )
            if rc != 0:
                errors.append(
                    'runtime.Run did not return error when agent_settled missing '
                    '(session not ready); go test failed:\n' + tail(err or out)
                )
        else:
            errors.append('internal/runtime directory does not exist (cannot verify Run signature)')
    finally:
        if wrote_test and os.path.exists(runtime_test_path):
            os.remove(runtime_test_path)

    # =========================================================================
    # 附加结构检查：6 个模板改写 act-path.md 引用为 runtime trace（KR6/KR7）。
    # =========================================================================
    actpath_templates = [
        os.path.join(repo_root, 'internal', 'prompt', 'templates', 'dream.md'),
        os.path.join(repo_root, 'internal', 'prompt', 'templates', 'learning.md'),
        os.path.join(repo_root, 'internal', 'prompt', 'templates', 'ctrl.md'),
        os.path.join(repo_root, 'internal', 'prompt', 'templates', 'skills', 'gen-loop.md'),
        os.path.join(repo_root, 'internal', 'prompt', 'templates', 'skills', 'gen-skill.md'),
        os.path.join(repo_root, 'internal', 'prompt', 'templates', 'skills', 'learning_loop.md'),
    ]
    stale_templates = []
    for tpl in actpath_templates:
        txt = read_text(tpl)
        if txt is None:
            errors.append('template missing: %s' % os.path.relpath(tpl, repo_root))
        elif 'act-path' in txt:
            stale_templates.append(os.path.relpath(tpl, repo_root))
    if stale_templates:
        errors.append(
            'templates still reference act-path.md (must be rewritten to runtime trace): %s'
            % ', '.join(stale_templates)
        )

    # =========================================================================
    # 回归门禁（KR5）：go test 关键包全绿。注意：须在临时 Go 测试删除后运行。
    # =========================================================================
    rc, out, err = run(
        [go, 'test',
         './internal/builder/...', './internal/prompt/...', './internal/cmd/...',
         './internal/runtime/...', './internal/env/...', './internal/handler/...',
         './internal/workspace/...', '-timeout', '60s'],
        cwd=repo_root, timeout=300,
    )
    if rc != 0:
        errors.append(
            'go test (builder/prompt/cmd/runtime/env/handler/workspace) failed:\n' + tail(err or out)
        )

    # 清理构建产物临时目录
    shutil.rmtree(build_dir, ignore_errors=True)

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
