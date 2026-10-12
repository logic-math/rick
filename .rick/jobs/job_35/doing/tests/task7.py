#!/usr/bin/env python3
"""task7 验收测试：完成 handler 覆盖 human-loop/ctrl/dream/learning 并让 cli 全量变薄。

覆盖测试方法的三类断言（前置条件/输入/操作/预期四要素已核对）：

1. 正常路径：`go build -o <tmp>/rick ./cmd/rick` + `go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v`
   → build 成功、cmd/handler 测试全绿。

2. 边界（human-loop dry-run + dream 扫描过滤 + ctrl --job 缺失）：
   - `human-loop --dry-run '测试主题'` 输出含 `sense_loop`；
   - `dream --dry-run` 只列「完成且未 dream」的 job（排除未完成/已 dream，按 job 号升序截断到 job_num=5）；
   - `ctrl`（无 --job）报 `--job flag is required` 且 exit 非 0。

3. 异常（learning 缺数据 + ctrl doing 目录不存在）：
   - `learning job_9`（doing/ 存在但 tasks.json 不存在）报 `tasks.json not found`，exit 非 0，不 panic；
   - `ctrl --dry-run --job job_999`（doing 目录不存在）报 `doing directory not found`，exit 非 0。

附加结构检查（对齐 task7 关键结果，这是 RED 阶段的「功能缺失」断言）：
- internal/handler 定义 7 个迁移编排函数 HumanLoop/Ctrl/CtrlDryRun/Dream/DreamDryRun/Learning/LearningDryRun；
- handler 不得 import internal/cmd（跨包循环依赖）；
- internal/cmd/{human_loop,ctrl,dream,learning}.go 变薄为「路由 + 参数解析 + 调 handler」
  （须 import internal/handler；dream/learning 不得再 import internal/executor）；
- dream 的 4 个扫描函数 selectPendingJobs/getDreamProcessedJobs/discoverCompletedJobs/jobNumber 迁 internal/workspace；
- ctrl.md 模板改写为 pi JSONL 语义（去 claude code NDJSON `type = "system/assistant/user/result"`，
  出现 agent_settled/tool_execution_start/tool_execution_end/message_end 等 pi 事件）。

本脚本只读源码 + 跑 go build/go test + 在独立临时 .rick 工作区跑 CLI（产物/夹具均落临时目录，
不污染仓库），幂等；仅向 stdout 输出一行 JSON。
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

    优先向上查找 go.mod（Go 项目根标记）；找不到则回退为脚本相对位置向上 5 层：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task7.py，向上 5 层即仓库根。
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

    for _ in range(5):
        d = os.path.dirname(d)
    return d


def read_text(path):
    """读取文本文件内容，失败返回 None。"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return f.read()
    except Exception:
        return None


def list_go_files(dirpath):
    """列出目录下（不含子目录）的 .go 文件绝对路径，目录不存在返回 []。"""
    if not os.path.isdir(dirpath):
        return []
    return sorted(
        os.path.join(dirpath, name)
        for name in os.listdir(dirpath)
        if name.endswith('.go')
    )


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


def write_tasks_json(path, statuses):
    """写一份 tasks.json，statuses 为 [(task_id, status), ...] 列表。"""
    tasks = [
        {"task_id": tid, "task_name": "t_" + tid, "status": st}
        for tid, st in statuses
    ]
    data = {"version": "1.0", "tasks": tasks}
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)


def job_num(job_id):
    """从 'job_N' 提取数字 N，失败返回 -1。"""
    m = re.match(r'^job_(\d+)$', job_id)
    return int(m.group(1)) if m else -1


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    # =========================================================================
    # 正常路径 1：go build -o <tmp>/rick ./cmd/rick（产物落临时目录，不污染 bin/rick）
    # =========================================================================
    build_dir = tempfile.mkdtemp(prefix='rick_task7_build_')
    bin_path = os.path.join(build_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', bin_path, './cmd/rick'], cwd=repo_root, timeout=300)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
    elif not os.path.isfile(bin_path):
        errors.append('go build ./cmd/rick succeeded but binary not produced at %s' % bin_path)

    # =========================================================================
    # 正常路径 2：go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v
    # =========================================================================
    rc, out, err = run(
        [go, 'test', './internal/cmd/...', './internal/handler/...', '-timeout', '60s', '-v'],
        cwd=repo_root, timeout=300,
    )
    if rc != 0:
        errors.append('go test ./internal/cmd/... ./internal/handler/... failed:\n' + tail(err or out))

    # =========================================================================
    # 结构检查（源码级，不依赖 build；这是 RED 阶段识别「迁移未发生」的断言）
    # =========================================================================
    handler_dir = os.path.join(repo_root, 'internal', 'handler')
    handler_src_txt = ''
    if not os.path.isdir(handler_dir):
        errors.append('internal/handler directory does not exist')
    else:
        handler_files = list_go_files(handler_dir)
        src_files = [f for f in handler_files if not f.endswith('_test.go')]
        handler_src_txt = '\n'.join(
            txt for txt in (read_text(f) for f in src_files) if txt is not None
        )
        if not src_files:
            errors.append('internal/handler has no non-test .go files')

    # 结构 1：handler 定义 7 个迁移编排函数
    if handler_src_txt:
        missing_funcs = []
        for fn in ('HumanLoop', 'Ctrl', 'CtrlDryRun', 'Dream', 'DreamDryRun',
                   'Learning', 'LearningDryRun'):
            if not re.search(r'\bfunc\s+' + fn + r'\s*\(', handler_src_txt):
                missing_funcs.append(fn)
        if missing_funcs:
            errors.append(
                'internal/handler missing migrated orchestration functions: %s'
                % ', '.join(missing_funcs)
            )

    # 结构 2：handler 不得 import internal/cmd（跨包循环依赖）
    if handler_src_txt:
        if '"github.com/sunquan/rick/internal/cmd"' in handler_src_txt:
            errors.append(
                'internal/handler must not import internal/cmd (cross-package cycle); '
                'flag values must be passed as Options{...}'
            )

    # 结构 3：internal/cmd/{human_loop,ctrl,dream,learning}.go 变薄为「调 handler」
    thin_files = {
        'human_loop': 'human_loop.go',
        'ctrl': 'ctrl.go',
        'dream': 'dream.go',
        'learning': 'learning.go',
    }
    thin_handler_fn = {
        'human_loop': 'HumanLoop',
        'ctrl': 'Ctrl',
        'dream': 'Dream',
        'learning': 'Learning',
    }
    for cmd_name, fname in thin_files.items():
        path = os.path.join(repo_root, 'internal', 'cmd', fname)
        txt = read_text(path)
        if txt is None:
            errors.append('internal/cmd/%s does not exist' % fname)
            continue
        # 变薄后必须 import internal/handler（路由 + 参数解析 + 调 handler）
        if '"github.com/sunquan/rick/internal/handler"' not in txt:
            errors.append(
                'internal/cmd/%s is not thin: missing import of internal/handler '
                '(must delegate to handler.%s)' % (fname, thin_handler_fn[cmd_name])
            )
        # dream/learning 原本依赖 executor，迁移后该依赖应随编排移出 cmd
        if cmd_name in ('dream', 'learning'):
            if '"github.com/sunquan/rick/internal/executor"' in txt:
                errors.append(
                    'internal/cmd/%s is not thin: still imports internal/executor '
                    '(executor dependency must move to handler/workspace)' % fname
                )

    # 结构 4：dream 的 4 个扫描函数迁 internal/workspace
    ws_dir = os.path.join(repo_root, 'internal', 'workspace')
    ws_src_txt = ''
    if not os.path.isdir(ws_dir):
        errors.append('internal/workspace directory does not exist')
    else:
        ws_files = [f for f in list_go_files(ws_dir) if not f.endswith('_test.go')]
        ws_src_txt = '\n'.join(
            txt for txt in (read_text(f) for f in ws_files) if txt is not None
        )
    if ws_src_txt:
        missing_scan = []
        for fn in ('selectPendingJobs', 'getDreamProcessedJobs', 'discoverCompletedJobs', 'jobNumber'):
            if not re.search(r'\bfunc\s+' + fn + r'\s*\(', ws_src_txt):
                missing_scan.append(fn)
        if missing_scan:
            errors.append(
                'internal/workspace missing dream scan functions: %s'
                % ', '.join(missing_scan)
            )

    # 结构 5：ctrl.md 模板改写为 pi JSONL 语义
    ctrl_tpl = os.path.join(repo_root, 'internal', 'prompt', 'templates', 'ctrl.md')
    ctrl_txt = read_text(ctrl_tpl)
    if ctrl_txt is None:
        errors.append('internal/prompt/templates/ctrl.md does not exist')
    else:
        stale_markers = []
        for marker in ('type = "system"', 'type = "assistant"', 'type = "user"', 'type = "result"'):
            if marker in ctrl_txt:
                stale_markers.append(marker)
        if stale_markers:
            errors.append(
                'internal/prompt/templates/ctrl.md still describes claude code NDJSON '
                'format (stale markers: %s); must be rewritten to pi JSONL semantics'
                % ', '.join(stale_markers)
            )
        pi_markers = ('agent_settled', 'tool_execution_start', 'tool_execution_end', 'message_end')
        if not any(m in ctrl_txt for m in pi_markers):
            errors.append(
                'internal/prompt/templates/ctrl.md lacks pi JSONL event markers '
                '(%s)' % ' / '.join(pi_markers)
            )

    # =========================================================================
    # 边界 + 异常（独立临时 .rick 工作区，避免污染仓库 .rick）
    # =========================================================================
    fx = tempfile.mkdtemp(prefix='rick_task7_fx_')
    try:
        jobs_dir = os.path.join(fx, '.rick', 'jobs')
        dream_dir = os.path.join(fx, '.rick', 'dream')
        for n in (1, 2, 3, 4, 5, 6, 7, 8, 9):
            os.makedirs(os.path.join(jobs_dir, 'job_%d' % n, 'doing'), exist_ok=True)
        os.makedirs(dream_dir, exist_ok=True)

        # job_1/2/5/6/7/8：全 success（完成）；job_4：未完成（running）；job_3：完成但已 dream。
        for n in (1, 2, 3, 5, 6, 7, 8):
            write_tasks_json(os.path.join(jobs_dir, 'job_%d' % n, 'doing', 'tasks.json'),
                             [('task1', 'success')])
        write_tasks_json(os.path.join(jobs_dir, 'job_4', 'doing', 'tasks.json'),
                         [('task1', 'running')])
        # job_9：doing/ 存在但无 tasks.json（供 learning 缺数据异常用例）。
        # dream 已处理标记：job_3 已 dream。
        with open(os.path.join(dream_dir, 'dream_run_job_3_log.md'), 'w', encoding='utf-8') as f:
            f.write('dreamed\n')

        # ---------- 边界 1：human-loop --dry-run 含 sense_loop ----------
        if os.path.isfile(bin_path):
            rc, out, err = run([bin_path, 'human-loop', '--dry-run', '测试主题'],
                               cwd=fx, timeout=120)
            if rc != 0:
                errors.append('rick human-loop --dry-run failed (exit %d):\n%s'
                              % (rc, tail(err or out)))
            elif 'sense_loop' not in out:
                errors.append("rick human-loop --dry-run output does not contain 'sense_loop'")
        else:
            errors.append('rick binary missing, skipping human-loop --dry-run check')

        # ---------- 边界 2：dream --dry-run 只列「完成且未 dream」，升序截断 ----------
        if os.path.isfile(bin_path):
            rc, out, err = run([bin_path, 'dream', '--dry-run'], cwd=fx, timeout=120)
            if rc != 0:
                errors.append('rick dream --dry-run failed (exit %d):\n%s'
                              % (rc, tail(err or out)))
            else:
                m = re.search(r'Pending jobs\s*\(\s*\d+\s*\)\s*:\s*\[([^\]]*)\]', out)
                if not m:
                    errors.append(
                        "rick dream --dry-run output missing 'Pending jobs (N): [...]' line:\n"
                        + tail(out)
                    )
                else:
                    listed = m.group(1).split()
                    expected_in = ['job_1', 'job_2', 'job_5', 'job_6', 'job_7']
                    expected_out = ['job_3', 'job_4', 'job_8', 'job_9']
                    for jid in expected_in:
                        if jid not in listed:
                            errors.append(
                                'dream --dry-run omitted completed-not-dreamed job %s (listed=%s)'
                                % (jid, listed)
                            )
                    for jid in expected_out:
                        if jid in listed:
                            errors.append(
                                'dream --dry-run wrongly listed %s (listed=%s)'
                                % (jid, listed)
                            )
                    # 升序
                    if listed != sorted(listed, key=job_num):
                        errors.append('dream --dry-run jobs not sorted ascending: %s' % listed)
                    # 截断到 job_num 默认 5
                    if len(listed) != 5:
                        errors.append(
                            'dream --dry-run expected 5 pending jobs (truncated), got %d: %s'
                            % (len(listed), listed)
                        )
        else:
            errors.append('rick binary missing, skipping dream --dry-run check')

        # ---------- 边界 3：ctrl 无 --job 报错退出 ----------
        if os.path.isfile(bin_path):
            rc, out, err = run([bin_path, 'ctrl'], cwd=fx, timeout=120)
            combined = out + err
            if '--job flag is required' not in combined:
                errors.append("rick ctrl (no --job) did not report '--job flag is required' "
                              '(exit %d)' % rc)
            elif rc == 0:
                errors.append('rick ctrl (no --job) exited 0 (expected non-zero)')
        else:
            errors.append('rick binary missing, skipping ctrl (no --job) check')

        # ---------- 异常 1：learning job_9（doing 存在，tasks.json 缺失）报 tasks.json not found ----------
        if os.path.isfile(bin_path):
            rc, out, err = run([bin_path, 'learning', 'job_9'], cwd=fx, timeout=120)
            combined = out + err
            if 'tasks.json not found' not in combined:
                errors.append("rick learning job_9 did not report 'tasks.json not found' "
                              '(exit %d)' % rc)
            elif rc == 0:
                errors.append('rick learning job_9 exited 0 (expected non-zero)')
            if 'panic:' in combined or 'goroutine ' in combined:
                errors.append('rick learning job_9 panicked:\n' + tail(combined))
        else:
            errors.append('rick binary missing, skipping learning job_9 check')

        # ---------- 异常 2：ctrl --dry-run --job job_999（doing 目录不存在）报 doing directory not found ----------
        if os.path.isfile(bin_path):
            rc, out, err = run([bin_path, 'ctrl', '--dry-run', '--job', 'job_999'],
                               cwd=fx, timeout=120)
            combined = out + err
            if 'doing directory not found' not in combined:
                errors.append("rick ctrl --dry-run --job job_999 did not report "
                              "'doing directory not found' (exit %d)" % rc)
            elif rc == 0:
                errors.append('rick ctrl --dry-run --job job_999 exited 0 (expected non-zero)')
        else:
            errors.append('rick binary missing, skipping ctrl --dry-run check')
    finally:
        shutil.rmtree(fx, ignore_errors=True)

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
