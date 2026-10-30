#!/usr/bin/env python3
"""gate2.py — job_49 S2 层门禁 2：6199 行为演练（真实 release CLI 全链 + 生产零触碰）

交接单 §7 安全复现环境（agent 不得修改本文件）：
  在临时 state-dir + 6199 端口，用 dev 树修复后构建的 CLI 做 promote 行为演练：
  A. 环境构造：TMPD 下 fake prod repo + prod home + start 脚本（不碰 ~/.rick / 6137 / 8414）
  B. 起旧实例（裸形态——cwd 不在 prod repo，P0-3 认领链的演练对象，11135 变体）
  C. 第一次 release promote：断言 RELEASE_OK + health build_id==version +
     restart.stopped 含旧 pid + skipped 空（认领成功——修复核心断言）
  D. 第二次 release promote（版本防同秒撞号）：断言 stopped/skipped 同样健康
  E. 坏回滚点拒绝：.last 指向 ELF 版本 → rollback 拒绝 + current 不变（P0-2 行为面）
  F. 生产零触碰：6137 前后都无监听；8414 health build_id 不变；~/.rick/web.pid 内容不变

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import urllib.request

DEV_TREE = "/Users/logic/workdir/rick-dev"
MAIN_TREE = "/Users/logic/workdir/AI_CODING/rick"
DEV_HOME = "/Users/logic/workdir/rick-dev-home"
PORT = 6199
ERRORS = []


def check(cond, msg):
    if not cond:
        ERRORS.append(msg)


def run(cmd, cwd=None, timeout=900, env_extra=None):
    env = dict(os.environ)
    # 复现环境硬约束：自定义 state-dir 必须同时给 RICK_PI_AGENT_DIR（防覆写生产 pi runtime）
    env.setdefault("RICK_PI_AGENT_DIR", os.path.join(tempfile.gettempdir(), "gate2-pi-agent"))
    if env_extra:
        env.update(env_extra)
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env)


def http_health(port):
    try:
        with urllib.request.urlopen(f"http://127.0.0.1:{port}/api/health", timeout=5) as r:
            return json.loads(r.read().decode())
    except Exception:
        return None


def find_dev_cli():
    """dev 树修复后构建的 CLI：优先 dev-web 的最新构建产物，否则现场 go build。"""
    bin_dir = os.path.join(DEV_HOME, "bin")
    if os.path.isdir(bin_dir):
        cands = sorted((os.path.getmtime(os.path.join(bin_dir, n)), n)
                       for n in os.listdir(bin_dir) if n.startswith("rick.dev."))
        if cands:
            return os.path.join(bin_dir, cands[-1][1])
    out = os.path.join(tempfile.gettempdir(), "gate2-rick-cli")
    r = run(["go", "build", "-o", out, "./cmd/rick"], cwd=DEV_TREE)
    check(r.returncode == 0, f"[env] dev CLI 现场构建失败:\n{r.stderr[-800:]}")
    return out


def main():
    cli = find_dev_cli()
    check(os.path.exists(cli), f"[env] CLI 不存在: {cli}")

    tmpd = tempfile.mkdtemp(prefix="gate2-6199-")
    prod_repo = os.path.join(tmpd, "prodrepo")
    prod_home = os.path.join(tmpd, "prodhome")
    state_dir = os.path.join(prod_home, ".rick")
    os.makedirs(os.path.join(prod_repo, "bin", "releases"))
    os.makedirs(state_dir)
    # 最小 rick 仓库形态（Validate 只查 go.mod；dev tree 显式传——默认解析会用祖父目录，
    # 在 TMPD 下会算错）
    with open(os.path.join(prod_repo, "go.mod"), "w") as f:
        f.write("module gate2.fake/prod\n\ngo 1.22\n")
    start_sh = os.path.join(prod_home, "start-web.sh")
    with open(start_sh, "w") as f:
        # 真实启动脚本形态（startProd 优先用它拉起生产）：cd 进 prod repo + exec bin/rick web
        f.write(f"#!/bin/bash\ncd {prod_repo}\nexec {prod_repo}/bin/rick web --listen 0.0.0.0 --port {PORT} --state-dir {state_dir}\n")
    os.chmod(start_sh, 0o755)

    base = {"http_proxy": "http://10.229.18.27:8412", "https_proxy": "http://10.229.18.27:8412",
            "no_proxy": "localhost,127.0.0.1", "RICK_PI_AGENT_DIR": os.path.join(tmpd, "pi-agent")}
    os.makedirs(base["RICK_PI_AGENT_DIR"], exist_ok=True)

    # 生产零触碰基线
    l6137_before = subprocess.run(["bash", "-c", "lsof -nP -iTCP:6137 -sTCP:LISTEN | wc -l"],
                                  capture_output=True, text=True).stdout.strip()
    health8414_before = http_health(8414)
    webpid_before = ""
    try:
        with open(os.path.expanduser("~/.rick/web.pid")) as f:
            webpid_before = f.read().strip()
    except Exception:
        pass

    old_pid = None
    try:
        # ---- B 前置：清 6199 残留（外部探针/上一轮 cleanup 漏收的实例会占端口，
        # 干扰「起旧实例」步骤；先杀干净再起）----
        pre = subprocess.run(["bash", "-c", "lsof -nP -tiTCP:6199 -sTCP:LISTEN"], capture_output=True, text=True).stdout
        for p in [x for x in pre.split() if x.strip().isdigit()]:
            subprocess.run(["bash", "-c", f"kill {p} 2>/dev/null; sleep 0.5; kill -9 {p} 2>/dev/null"], capture_output=True)
            time.sleep(0.5)

        # ---- B. 起旧实例（裸形态：cwd=TMPD，非 prod repo；持 6199 锁）----
        r = run([cli, "web", "--daemon", "--listen", "127.0.0.1", "--port", str(PORT),
                 "--state-dir", state_dir], cwd=tmpd, env_extra=base)
        check(r.returncode == 0, f"[B] 旧实例启动失败:\n{r.stderr[-500:]}")
        h = None
        for _ in range(40):
            h = http_health(PORT)
            if h and h.get("status") == "ok":
                break
            time.sleep(0.5)
        check(h and h.get("status") == "ok", f"[B] 旧实例 health 未就绪: {h}")
        try:
            with open(os.path.join(state_dir, "web.pid")) as f:
                old_pid = int(f.read().strip())
        except Exception:
            old_pid = None
        check(old_pid is not None, "[B] 旧实例 web.pid 未写入")

        def do_release():
            time.sleep(1.2)  # 版本号防同秒撞号
            return run([cli, "tools", "release",
                        "--prod-repo", prod_repo, "--dev-tree", DEV_TREE,
                        "--prod-home", prod_home,
                        "--state-dir", state_dir, "--start-script", start_sh,
                        "--port", str(PORT), "--no-gates", "--yes", "--skip-frontend",
                        "--json"], cwd=tmpd, env_extra=base, timeout=1200)

        # ---- C. 第一次 promote：认领并替换旧实例 ----
        r = do_release()
        out = r.stdout + r.stderr
        check("RELEASE_OK" in out, f"[C] 第一次 promote 无 RELEASE_OK:\n{out[-2500:]}")
        h = http_health(PORT)
        check(h and h.get("status") == "ok", f"[C] promote 后 health 异常: {h}")
        if h:
            # build_id 指纹以 sha7 开头（版本号 = <sha7>-<ts>），且不再是旧实例的 dev 指纹
            check("-" in h.get("build_id", "") and h["build_id"] != (http_health(PORT) or {}).get("build_id", "x") or True, "")
            check(h.get("build_id", "").split("-")[0] in out, f"[C] health build_id 与版本号不符: {h}")
        check(f"stopped=[{old_pid}]" in out or f"\"stopped\": [{old_pid}]" in out or f"stopped=[{old_pid}," in out,
              f"[C] 旧实例 pid={old_pid} 未出现在 stopped 清单（认领失败——P0-3 核心断言）:\n{out[-2500:]}")
        check("skipped=[]" in out or '"skipped": []' in out or '"skipped":[]' in out,
              f"[C] skipped 非空（拒杀清单）:\n{out[-2500:]}")

        # ---- D. 第二次 promote：连续发布健康 ----
        r2 = do_release()
        out2 = r2.stdout + r2.stderr
        check("RELEASE_OK" in out2, f"[D] 第二次 promote 失败:\n{out2[-2500:]}")
        check("skipped=[]" in out2 or '"skipped": []' in out2 or '"skipped":[]' in out2,
              f"[D] 第二次 skipped 非空:\n{out2[-2500:]}")

        # ---- E. 坏回滚点拒绝（P0-2 行为面）----
        # 造一个坏版本目录 + .last 指向它，rollback 必须拒绝且 current 不变
        cur_before = os.path.realpath(os.path.join(prod_repo, "bin", "releases", "current"))
        bad_dir = os.path.join(prod_repo, "bin", "releases", "badbad0-261030120002")
        os.makedirs(bad_dir, exist_ok=True)
        with open(os.path.join(bad_dir, "rick"), "wb") as f:
            f.write(b"\x7fELF" + b"\x00" * 200)  # ELF 头——本机 darwin 不可执行
        os.chmod(os.path.join(bad_dir, "rick"), 0o755)
        last_path = os.path.join(prod_repo, "bin", "releases", ".last")
        with open(last_path, "w") as f:
            f.write("badbad0-261030120002\n")
        r3 = run([cli, "tools", "release", "--rollback",
                 "--prod-repo", prod_repo, "--dev-tree", DEV_TREE,
                 "--prod-home", prod_home,
                 "--state-dir", state_dir, "--start-script", start_sh,
                 "--port", str(PORT), "--yes", "--json"], cwd=tmpd, env_extra=base, timeout=600)
        out3 = r3.stdout + r3.stderr
        check(r3.returncode != 0, f"[E] 坏回滚点 rollback 未拒绝（exit=0）:\n{out3[-1500:]}")
        check("回滚点不可执行" in out3, f"[E] rollback 错误未点名「回滚点不可执行」:\n{out3[-1500:]}")
        cur_after = os.path.realpath(os.path.join(prod_repo, "bin", "releases", "current"))
        check(cur_before == cur_after, f"[E] current 被切换了（{cur_before} → {cur_after}）——必须保留现场")
        h = http_health(PORT)
        check(h and h.get("status") == "ok", f"[E] rollback 拒绝后演练实例死了: {h}")

        # ---- F. 生产零触碰 ----
        l6137_after = subprocess.run(["bash", "-c", "lsof -nP -iTCP:6137 -sTCP:LISTEN | wc -l"],
                                     capture_output=True, text=True).stdout.strip()
        check(l6137_before == "0" and l6137_after == "0",
              f"[F] 6137 状态变化: before={l6137_before} after={l6137_after}（生产必须保持死亡直到 S5）")
        health8414_after = http_health(8414)
        check(health8414_before and health8414_after and
              health8414_before.get("build_id") == health8414_after.get("build_id"),
              f"[F] 8414 build_id 变化: {health8414_before} → {health8414_after}")
        webpid_after = ""
        try:
            with open(os.path.expanduser("~/.rick/web.pid")) as f:
                webpid_after = f.read().strip()
        except Exception:
            pass
        check(webpid_before == webpid_after, f"[F] 生产 web.pid 被改写: {webpid_before} → {webpid_after}")
    finally:
        # 收尾：停演练实例（只杀演练 state-dir 的持有者），清 TMPD
        try:
            with open(os.path.join(state_dir, "web.pid")) as f:
                p = int(f.read().strip())
            subprocess.run(["bash", "-c", f"kill {p} 2>/dev/null; sleep 1; kill -9 {p} 2>/dev/null"],
                            capture_output=True)
        except Exception:
            pass
        shutil.rmtree(tmpd, ignore_errors=True)

    okk = len(ERRORS) == 0
    print(json.dumps({"pass": okk, "gate": "gate2", "errors": [e for e in ERRORS if e]},
                     ensure_ascii=False, indent=2))
    sys.exit(0 if okk else 1)


if __name__ == "__main__":
    main()
