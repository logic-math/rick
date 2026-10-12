#!/usr/bin/env python3
"""task3 验收测试：落地 env 模块（四职责：pi + 生态扩展 + rick 定制 + 就绪 check）。

按 skill:tdd 四要素（前置条件/输入/操作/预期）实现测试方法三类断言：

1. 正常路径（前置=task2/task4 完成；输入=无；操作=build + `rick tools init-pi`，
   隔离 HOME + PATH 指向含 fake pi/node/npm/npx 的目录；预期=exit 0 且 stdout 含
   `✅ pi environment ready`，且 managed runtime 真正落盘（fake npm 物化二进制））。

2. 边界（幂等 + 四职责 check；前置=init-pi 已成功一次；操作=再次 init-pi +
   `go test ./internal/env/... -run TestIsPIReady -v`；预期=第二次 exit 0 且无
   `newly installed`；env 包定义 IsPIReady + CheckPIInstalled/CheckEcosystemExtensions/
   CheckRickAgents/CheckRickHooks，且有 TestIsPIReady 单测通过）。

3. 异常（前置=runtime 未装；操作=`HOME=<tmp> PATH=<空 tmp>` 跑 init-pi；预期=stderr
   含 `requires Node.js`、exit 1、不 panic；另 env 测试覆盖 CheckEcosystemExtensions
   在某扩展缺失时返回非空切片——用 `go test ./internal/env/...` 全量跑 + 源码断言）。

脚本只读代码 + 跑 go build/go test + 用 fake 脚本跑真实 CLI 分支（fake 脚本开头恢复
PATH，见 bugs.md「fake pi PATH 替换」）；构建产物落临时目录；幂等；仅向 stdout 输出一行 JSON。
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile


# 共享的 fake pi：`--version` 打印版本、`list` 打印两个生态扩展（幂等跳过 install）。
# 开头恢复系统 PATH——PATH 被替换为只含 fake 目录后，cat/mkdir/chmod 等外部命令找不到。
FAKE_PI = """#!/bin/sh
export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH
case "$1" in
  --version) echo "0.84.1" ;;
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access" ;;
  install) echo "installing $2" ;;
  *) exit 0 ;;
esac
exit 0
"""

# fake npm：模拟 `npm install --prefix <dir> ...`，把 fake pi 物化到
# <dir>/node_modules/.bin/pi（即 runtime.RuntimeBin() 的真实安装落盘路径），
# 使 ensurePI/installManagedPI 的「安装 → 检查 RuntimeBin 存在」分支真实生效。
FAKE_NPM = """#!/bin/sh
export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH
prefix=""
prev=""
for a in "$@"; do
  if [ "$prev" = "--prefix" ]; then prefix="$a"; fi
  prev="$a"
done
if [ -z "$prefix" ]; then
  echo "fake npm: --prefix missing" >&2
  exit 1
fi
mkdir -p "$prefix/node_modules/.bin"
cat > "$prefix/node_modules/.bin/pi" <<'FAKEPI_EOF'
""" + FAKE_PI + """
FAKEPI_EOF
chmod +x "$prefix/node_modules/.bin/pi"
exit 0
"""

FAKE_NODE = "#!/bin/sh\nexit 0\n"
FAKE_NPX = "#!/bin/sh\nexit 0\n"


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task3.py，向上 5 层即仓库根。
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


def tail(text, n=1500):
    """截取文本尾部，用于错误信息（避免刷屏）。"""
    if not text:
        return ''
    return text[-n:]


def write_fake_bin(dirpath, name, content):
    """写一个可执行 fake 脚本到 dirpath 下，名为 name。"""
    path = os.path.join(dirpath, name)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)
    os.chmod(path, 0o755)
    return path


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
        try:
            with open(p, 'r', encoding='utf-8') as f:
                parts.append(f.read())
        except Exception:
            parts.append('')
    return '\n'.join(parts)


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    internal_dir = os.path.join(repo_root, 'internal')
    env_dir = os.path.join(internal_dir, 'env')
    tools_init_pi_go = os.path.join(internal_dir, 'cmd', 'tools_init_pi.go')
    tools_theme_go = os.path.join(internal_dir, 'cmd', 'tools_theme.go')

    # ---------- 正常路径 0：go build ./cmd/rick（产物落临时目录，不污染仓库 bin/） ----------
    build_dir = tempfile.mkdtemp(prefix='rick_task3_build_')
    rick_bin = os.path.join(build_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', rick_bin, './cmd/rick'], cwd=repo_root, timeout=300)
    build_ok = (rc == 0) and os.path.isfile(rick_bin)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
    elif not os.path.isfile(rick_bin):
        errors.append('go build ./cmd/rick succeeded but binary not produced at %s' % rick_bin)

    if build_ok:
        # ============ 正常路径：隔离 HOME + fake pi/node/npm/npx ============
        home = tempfile.mkdtemp(prefix='rick_task3_home_')
        fake_dir = tempfile.mkdtemp(prefix='rick_task3_fakebin_')
        write_fake_bin(fake_dir, 'pi', FAKE_PI)
        write_fake_bin(fake_dir, 'npm', FAKE_NPM)
        write_fake_bin(fake_dir, 'node', FAKE_NODE)
        write_fake_bin(fake_dir, 'npx', FAKE_NPX)

        cli_env = os.environ.copy()
        cli_env['HOME'] = home
        cli_env['PATH'] = fake_dir

        rc, out, err = run([rick_bin, 'tools', 'init-pi'], cwd=repo_root, timeout=180, env=cli_env)
        if rc != 0:
            errors.append('init-pi (正常路径) exit=%d, want 0. stderr:\n%s' % (rc, tail(err)))
        else:
            if '✅ pi environment ready' not in out:
                errors.append('init-pi (正常路径) stdout 缺 `✅ pi environment ready`')
            if 'newly installed' not in out:
                errors.append('init-pi (正常路径) 首次安装未打印 `newly installed`（安装分支未生效）')
            # 职责 1 真实落地：fake npm 应把 managed runtime 物化到 RuntimeBin()。
            runtime_bin = os.path.join(home, '.rick', 'pi', 'agent', 'runtime', 'node_modules', '.bin', 'pi')
            if not os.path.isfile(runtime_bin):
                errors.append('managed pi runtime 二进制未落盘: %s' % runtime_bin)

        # ============ 边界：幂等（同一隔离 HOME 再跑一次） ============
        rc, out, err = run([rick_bin, 'tools', 'init-pi'], cwd=repo_root, timeout=180, env=cli_env)
        if rc != 0:
            errors.append('init-pi (幂等重跑) exit=%d, want 0. stderr:\n%s' % (rc, tail(err)))
        else:
            if 'newly installed' in out:
                errors.append('init-pi (幂等重跑) 打印了 `newly installed`（不幂等）')
            if '✅ pi environment ready' not in out:
                errors.append('init-pi (幂等重跑) 缺 `✅ pi environment ready`')

        # ============ 异常：缺 node/npm（PATH 指向空目录，runtime 未装） ============
        empty_home = tempfile.mkdtemp(prefix='rick_task3_emptyhome_')
        empty_path = tempfile.mkdtemp(prefix='rick_task3_emptypath_')
        exc_env = os.environ.copy()
        exc_env['HOME'] = empty_home
        exc_env['PATH'] = empty_path
        rc, out, err = run([rick_bin, 'tools', 'init-pi'], cwd=repo_root, timeout=180, env=exc_env)
        if rc != 1:
            errors.append('init-pi (缺 node/npm) exit=%d, want 1. stdout:\n%s stderr:\n%s'
                          % (rc, tail(out), tail(err)))
        else:
            if 'requires Node.js' not in err:
                errors.append('init-pi (缺 node/npm) stderr 缺 `requires Node.js`')
            if 'panic:' in err:
                errors.append('init-pi (缺 node/npm) 发生 panic')

    # ---------- 边界/异常：go test ./internal/env/... ----------
    # TestIsPIReady 覆盖 IsPIReady + 四 check 函数（测试方法边界）；全量覆盖
    # CheckEcosystemExtensions 缺失→非空切片（测试方法异常）。Go 测试按仓库约定
    # 自行隔离 HOME/PATH（t.Setenv），故此处用默认 env 即可。
    rc, out, err = run(
        [go, 'test', '-timeout', '120s', './internal/env/...', '-run', 'TestIsPIReady', '-v'],
        cwd=repo_root, timeout=180,
    )
    if rc != 0:
        errors.append('go test ./internal/env/... -run TestIsPIReady failed:\n' + tail(err or out))

    rc, out, err = run(
        [go, 'test', '-timeout', '120s', './internal/env/...'],
        cwd=repo_root, timeout=180,
    )
    if rc != 0:
        errors.append('go test ./internal/env/... (全量, 含 CheckEcosystemExtensions 缺失分支) failed:\n' + tail(err or out))

    # ---------- 结构：internal/env 包存在 + 四职责函数 + 扩展 seam ----------
    if not os.path.isdir(env_dir):
        errors.append('internal/env directory does not exist')
    else:
        env_files = list_go_files(env_dir)
        env_src = [f for f in env_files if not f.endswith('_test.go')]
        env_test = [f for f in env_files if f.endswith('_test.go')]
        env_src_txt = concat_files(env_src)
        env_test_txt = concat_files(env_test)
        if not env_src:
            errors.append('internal/env has no non-test .go files')

        # 职责 4：就绪 check 函数（测试方法边界明确点名的五个符号）。
        for sym in ('IsPIReady', 'CheckPIInstalled', 'CheckEcosystemExtensions',
                    'CheckRickAgents', 'CheckRickHooks'):
            if sym not in env_src_txt:
                errors.append('internal/env missing %s (职责4 就绪 check)' % sym)

        # 职责 3：rick 自有定制落盘入口。
        if 'DeployRickCustomizations' not in env_src_txt:
            errors.append('internal/env missing DeployRickCustomizations (职责3 rick 定制)')

        # 扩展 seam：RuntimeEnv 接口 + piEnv 实现（为将来 dsh 留位）。
        if 'type RuntimeEnv interface' not in env_src_txt:
            errors.append('internal/env missing `type RuntimeEnv interface` (扩展 seam)')
        for sym in ('DeployCustomizations', 'CheckReady'):
            if sym not in env_src_txt:
                errors.append('internal/env missing RuntimeEnv 方法 %s' % sym)
        if 'piEnv' not in env_src_txt:
            errors.append('internal/env missing piEnv 实现')

        # 测试方法点名的单测 + CheckEcosystemExtensions 缺失分支覆盖。
        if 'TestIsPIReady' not in env_test_txt:
            errors.append('internal/env tests missing TestIsPIReady (IsPIReady ok=true + 四 check 就绪)')
        if 'CheckEcosystemExtensions' not in env_test_txt:
            errors.append('internal/env tests missing CheckEcosystemExtensions 覆盖（缺失→非空切片）')

    # ---------- 结构：cmd 变薄（tools_init_pi.go / tools_theme.go 调用 env） ----------
    init_pi_txt = concat_files([tools_init_pi_go])
    theme_txt = concat_files([tools_theme_go])
    if 'internal/env' not in init_pi_txt:
        errors.append('internal/cmd/tools_init_pi.go does not import internal/env (未变薄为 Cobra 入口)')
    if 'internal/env' not in theme_txt:
        errors.append('internal/cmd/tools_theme.go does not import internal/env (theme 相关未随 env 迁移)')

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
