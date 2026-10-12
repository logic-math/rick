#!/usr/bin/env python3
"""task4 验收测试：重构 runtime 模块（pi 调用逻辑收口到 runtime 层）。

覆盖测试方法中的三类断言（前置条件/输入/操作/预期四要素已核对）：

1. 正常路径：`go build -o <tmp>/rick ./cmd/rick` + `go test ./internal/runtime/...`
   → build 成功、runtime 测试全绿（cli/executor/agentdir 迁移后仍绿）。
   附加：`go build ./...` 编译全量（验证 internal/cmd 等 import 从
   internal/agent/piagent 改为 internal/runtime 后不编译断裂）。

2. 边界（session 就绪判定 + config 默认值）：
   - `internal/runtime` 源码定义 `isSessionReady`（sessionID 非空 && settled），
     且 runtime 测试覆盖（fake JSONL 有/无 agent_settled 两种）。
   - config 增加 `runtime` 字段（json tag "runtime"），`GetDefaultConfig()` 设
     `Runtime:"pi"`，`LoadConfig()` 归一化空值 → "pi"，且存在
     `TestLoadConfig_RuntimeDefault` 单测。

3. 异常（pi 缺失 + 接口保留）：
   - `FindBinary` 错误信息含 `pi binary not found`，且迁移后的
     `TestFindBinary_FallsBackToPathLookup` 仍在 runtime 测试中。
   - `grep -rn "internal/agent/piagent" internal/` 无残留（旧包目录无 .go），
     但 `internal/agent` 接口 + `internal/actpath` 仍保留。
   - runtime 继续实现 `internal/agent` 的 AgentExecutor/AgentSession。
   - `Runtime` 接口（Name/Run）+ `piRuntime` 骨架 + `Trace` 结构体 +
     `--append-system-prompt` 注入（为将来 dsh runtime 留扩展位）。

本脚本只读代码 + 跑 `go build`/`go test`（不修改源码，构建产物落临时目录），
幂等；仅向 stdout 输出一行 JSON。
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

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task4.py，向上 5 层即仓库根。
    """
    d = os.path.dirname(os.path.abspath(start_file))

    probe = d
    while True:
        if os.path.isdir(os.path.join(probe, '.git')):
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


def concat_files(paths):
    """拼接多个文件内容为一个字符串，读取失败的文件记为 ''。"""
    parts = []
    for p in paths:
        txt = read_text(p)
        if txt is not None:
            parts.append(txt)
    return '\n'.join(parts)


def run(cmd, cwd, timeout=300, env=None):
    """运行子进程，返回 (returncode, stdout, stderr)。"""
    try:
        p = subprocess.run(
            cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env
        )
        return p.returncode, p.stdout, p.stderr
    except FileNotFoundError as e:
        return 127, '', str(e)
    except subprocess.TimeoutExpired as e:
        out = e.stdout or ''
        err = (e.stderr or '') + '\n[timeout after %ds]' % timeout
        return 124, out, err


def tail(text, n=1200):
    """截取文本尾部，用于错误信息（避免刷屏）。"""
    if not text:
        return ''
    return text[-n:]


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    internal_dir = os.path.join(repo_root, 'internal')
    runtime_dir = os.path.join(internal_dir, 'runtime')
    old_piagent_dir = os.path.join(internal_dir, 'agent', 'piagent')
    agent_iface = os.path.join(internal_dir, 'agent', 'interface.go')
    actpath_dir = os.path.join(internal_dir, 'actpath')
    config_go = os.path.join(internal_dir, 'config', 'config.go')
    loader_go = os.path.join(internal_dir, 'config', 'loader.go')
    config_test_dir = os.path.join(internal_dir, 'config')

    # ---------- 正常路径 1：go build ./cmd/rick ----------
    build_dir = tempfile.mkdtemp(prefix='rick_task4_build_')
    build_out = os.path.join(build_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', build_out, './cmd/rick'], cwd=repo_root, timeout=300)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
    else:
        if not os.path.isfile(build_out):
            errors.append('go build ./cmd/rick succeeded but binary not produced at %s' % build_out)

    # ---------- 正常路径 2：go test ./internal/runtime/... ----------
    rc, out, err = run(
        [go, 'test', '-timeout', '180s', './internal/runtime/...'],
        cwd=repo_root, timeout=240,
    )
    if rc != 0:
        errors.append('go test ./internal/runtime/... failed:\n' + tail(err or out))

    # ---------- 正常路径 3：go build ./...（import 迁移不编译断裂） ----------
    rc, out, err = run([go, 'build', './...'], cwd=repo_root, timeout=300)
    if rc != 0:
        errors.append('go build ./... failed (import migration broke compilation):\n' + tail(err or out))

    # ---------- 正常路径 4：go test ./internal/config/...（验证 RuntimeDefault 归一化真实行为） ----------
    rc, out, err = run(
        [go, 'test', '-timeout', '120s', './internal/config/...'],
        cwd=repo_root, timeout=180,
    )
    if rc != 0:
        errors.append('go test ./internal/config/... failed:\n' + tail(err or out))

    # ---------- 结构：internal/runtime 包存在且迁移了源文件 ----------
    runtime_src = []
    runtime_test = []
    runtime_src_txt = ''
    runtime_test_txt = ''
    if not os.path.isdir(runtime_dir):
        errors.append('internal/runtime directory does not exist')
    else:
        runtime_files = list_go_files(runtime_dir)
        runtime_src = [f for f in runtime_files if not f.endswith('_test.go')]
        runtime_test = [f for f in runtime_files if f.endswith('_test.go')]
        runtime_src_txt = concat_files(runtime_src)
        runtime_test_txt = concat_files(runtime_test)
        if not runtime_src:
            errors.append('internal/runtime has no non-test .go files')
        for name in ('cli.go', 'executor.go', 'agentdir.go'):
            if not any(os.path.basename(f) == name for f in runtime_src):
                errors.append('internal/runtime missing migrated source file: %s' % name)
        if not runtime_test:
            errors.append('internal/runtime has no *_test.go files (cli/executor/agentdir 测试未迁移)')

    # ---------- 结构：旧包 internal/agent/piagent 无残留 .go ----------
    old_go_files = list_go_files(old_piagent_dir)
    if old_go_files:
        errors.append(
            'old package internal/agent/piagent still contains .go files: %s'
            % ', '.join(os.path.basename(f) for f in old_go_files)
        )

    # ---------- 异常：grep 无 "internal/agent/piagent" 残留 ----------
    rc, out, err = run(
        ['grep', '-rn', 'internal/agent/piagent', 'internal/'],
        cwd=repo_root, timeout=60,
    )
    if rc == 0:
        errors.append('residual references to "internal/agent/piagent" found:\n' + tail(out))
    elif rc != 1:
        errors.append('grep for residual references failed:\n' + tail(err or out))

    # ---------- 异常：internal/agent 接口 + internal/actpath 保留 ----------
    agent_iface_txt = read_text(agent_iface)
    if agent_iface_txt is None:
        errors.append('internal/agent/interface.go does not exist (internal/agent 接口应保留)')
    else:
        for sym in ('AgentExecutor', 'AgentSession'):
            if sym not in agent_iface_txt:
                errors.append('internal/agent/interface.go missing %s (接口应保留)' % sym)
    if not os.path.isfile(os.path.join(actpath_dir, 'generator.go')):
        errors.append('internal/actpath/generator.go does not exist (internal/actpath 应保留)')

    # ---------- runtime 继续实现 internal/agent 接口 ----------
    if runtime_src:
        if 'github.com/sunquan/rick/internal/agent' not in runtime_src_txt:
            errors.append('internal/runtime does not import internal/agent (应继续实现 AgentExecutor/AgentSession)')
        if 'agent.AgentSession' not in runtime_src_txt:
            errors.append('internal/runtime missing agent.AgentSession (Execute 返回值)')
        if 'func NewExecutor' not in runtime_src_txt:
            errors.append('internal/runtime missing func NewExecutor (Executor 未迁移)')

    # ---------- 边界：session 就绪判定 isSessionReady ----------
    if runtime_src:
        if 'isSessionReady' not in runtime_src_txt:
            errors.append('internal/runtime source missing isSessionReady (sessionID 非空 && settled)')
    if 'isSessionReady' not in runtime_test_txt:
        errors.append('internal/runtime tests missing isSessionReady coverage (有/无 agent_settled 两种)')

    # ---------- 边界：不改变 parseStream 无 agent_settled 不报错行为 ----------
    if 'TestParseStream_NoAgentSettled' not in runtime_test_txt:
        errors.append('internal/runtime tests missing TestParseStream_NoAgentSettled (缺 agent_settled 不报错、回退计时)')
    if 'agent_settled' not in runtime_test_txt:
        errors.append('internal/runtime tests missing agent_settled fixture')

    # ---------- 扩展 seam：Runtime 接口 + piRuntime + Trace ----------
    if runtime_src:
        for needle, label in (
            ('type Runtime interface', 'type Runtime interface'),
            ('Name() string', 'Runtime.Name() string'),
            ('methodText string', 'Run(methodText string, ...)'),
            ('promptFile string', 'Run(..., promptFile string, ...)'),
            ('cfg *config.Config', 'Run(..., cfg *config.Config)'),
            ('*Trace', 'Run 返回 *Trace'),
            ('piRuntime', 'piRuntime 骨架'),
            ('type Trace struct', 'type Trace struct'),
        ):
            if needle not in runtime_src_txt:
                errors.append('internal/runtime missing %s' % label)

    # ---------- 扩展 seam：--append-system-prompt 注入 ----------
    if '--append-system-prompt' not in runtime_src_txt:
        errors.append('internal/runtime missing --append-system-prompt injection (methodText 会话前注入)')

    # ---------- 边界：config runtime 字段 + 默认值 + 归一化 ----------
    config_txt = read_text(config_go)
    if config_txt is None:
        errors.append('internal/config/config.go does not exist')
    else:
        if 'json:"runtime"' not in config_txt:
            errors.append('config.Config missing `runtime` field (json:"runtime")')

    loader_txt = read_text(loader_go)
    if loader_txt is None:
        errors.append('internal/config/loader.go does not exist')
    else:
        if not re.search(r'Runtime\s*:\s*"pi"', loader_txt):
            errors.append('GetDefaultConfig() missing Runtime:"pi" (覆盖无 config 文件分支)')
        if 'cfg.Runtime' not in loader_txt:
            errors.append('LoadConfig() missing runtime 归一化 (cfg.Runtime 空值 -> "pi")')

    config_test_txt = concat_files(list_go_files(config_test_dir))
    if 'RuntimeDefault' not in config_test_txt:
        errors.append('internal/config tests missing TestLoadConfig_RuntimeDefault (空 config -> cfg.Runtime=="pi")')

    # ---------- 异常：FindBinary 错误信息 + 迁移测试 ----------
    if 'pi binary not found' not in runtime_src_txt:
        errors.append('internal/runtime FindBinary missing "pi binary not found" error message')
    if 'TestFindBinary_FallsBackToPathLookup' not in runtime_test_txt:
        errors.append('internal/runtime tests missing TestFindBinary_FallsBackToPathLookup (pi 缺失时报错)')

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
