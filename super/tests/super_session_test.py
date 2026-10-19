# super_session_test.py — job_48/task4：中控会话 prompt 工作区上下文 + CLI resume 互通 e2e。
#
# 覆盖（spec 三段）：
#   1. fake rick 工作区（.rick/loops/super-loop.md + skills/gen_env_skill/skill.md
#      + super_env/）——严格加载的最小 fixture
#   2. CLI `rick super --name demo`（fake pi，HOME 隔离 + config.pi_path）→
#      断言 prompt.md 含：gen_env skill 全文证据（门禁系统设计引导）+ super_env
#      绝对路径 + 观察手段清单（events.jsonl/archive.json/codes/heartbeat.json）
#      + 干预手段（rick super event / 单写者）
#   3. CLI 互通：`rick super --name demo --resume`（fake pi dump argv）→
#      断言 --session <uuid> 与首建 session_id 一致 + 无 --append-system-prompt
#      （resume 是加载语义不是新提示词注入）
#
# stdlib-only；Python >=3.9（禁 PEP604 联合类型语法）。
import json
import os
import stat
import subprocess
import sys
import tempfile
import unittest

_REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# gen_env skill fixture 正文（含独特关键词，验证全文 inline 而非目录引用）
_GEN_ENV_SKILL = """# skill:gen-env（env 建设与升级）

## 触发场景

super 会话中 env 缺失或需升级。

## 预期效果

产出可运行 env（目录协议：env/{env.json, feature_space.json, code/, skills/, data/}）。

## 核心内容

访谈式流程：任务定义 → 评分方向 → **门禁系统设计引导**（正确性门/性能门/
exploit 注册表初始化）→ 初始特征空间 → 初始解。
"""

_SUPER_LOOP = """---
name: super-loop
trigger: "当启动或操作 rick super 模式时触发"
scope: "super 会话（全局）"
---

# Loop: super 中控

Step 0 环境自检 → Step 1 env 建设引导（gen_env）→ Step 2 启动（门先证红）
→ Step 3 监控 → Step 4 干预（event 队列）→ Step 5 env 升级 → Step 6 停止。
"""


class SuperSessionE2E(unittest.TestCase):
    """中控会话 e2e：prompt 上下文契约 + resume 互通（fake pi，秒级）。"""

    @classmethod
    def setUpClass(cls):
        cls.root = tempfile.mkdtemp(prefix="super-session-e2e-")
        # fake rick 工作区：<root>/ws/.rick/...（cwd=ws 时 superWsRoot →
        # <ws>/.rick/super_env——job_48 存储模型）
        cls.wsdir = os.path.join(cls.root, "ws")
        rick = os.path.join(cls.wsdir, ".rick")
        for d in ("loops", os.path.join("skills", "gen_env_skill"),
                  "jobs", os.path.join("super_env", "demo")):
            os.makedirs(os.path.join(rick, d), exist_ok=True)
        with open(os.path.join(rick, "loops", "super-loop.md"), "w",
                  encoding="utf-8") as f:
            f.write(_SUPER_LOOP)
        with open(os.path.join(rick, "skills", "gen_env_skill", "skill.md"),
                  "w", encoding="utf-8") as f:
            f.write(_GEN_ENV_SKILL)

        # fake pi：dump argv 到文件后退出 0（bugs.md：fake 脚本恢复 PATH）
        cls.argv_path = os.path.join(cls.root, "pi-argv.txt")
        fake_pi = os.path.join(cls.root, "fake-pi.sh")
        with open(fake_pi, "w", encoding="utf-8") as f:
            f.write("#!/bin/sh\n"
                    "export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n"
                    'printf \'%s\\n\' "$@" > "' + cls.argv_path + '"\n'
                    "exit 0\n")
        os.chmod(fake_pi, os.stat(fake_pi).st_mode | stat.S_IEXEC)

        # HOME 隔离 + config.pi_path 指向 fake pi（CallCLI 消费）
        cls.home = os.path.join(cls.root, "home")
        os.makedirs(os.path.join(cls.home, ".rick"), exist_ok=True)
        with open(os.path.join(cls.home, ".rick", "config.json"), "w",
                  encoding="utf-8") as f:
            json.dump({"pi_path": fake_pi}, f)

        # 构建 rick（一次）
        cls.rick = os.path.join(cls.root, "rick")
        r = subprocess.run(["go", "build", "-o", cls.rick, "./cmd/rick"],
                           cwd=_REPO, capture_output=True, text=True, timeout=600)
        if r.returncode != 0:
            raise unittest.SkipTest("go build 失败：%s" % r.stderr[-300:])

    def _run_super(self, *args):
        env = dict(os.environ)
        env["HOME"] = self.home
        env.pop("RICK_STATE_DIR", None)  # 测试不受外部 state-dir 覆盖影响
        return subprocess.run(
            [self.rick, "super"] + list(args),
            cwd=self.wsdir, env=env, capture_output=True, text=True, timeout=120)

    def _read_argv(self):
        with open(self.argv_path, encoding="utf-8") as f:
            return [l.rstrip("\n") for l in f if l.strip()]

    def test_prompt_workspace_context_and_resume_interop(self):
        # ---- 1) 新会话：BuildSuperPrompt + fake pi 消费 ----
        r = self._run_super("--name", "demo")
        self.assertEqual(r.returncode, 0,
                         "super session 失败：%s%s" % (r.stdout, r.stderr))
        out = r.stdout + r.stderr
        self.assertIn("Session ID:", out, "回执缺 Session ID")

        # session 工件：sessions/<uuid>/{session_id, method.md, prompt.md}
        sess_root = os.path.join(self.wsdir, ".rick", "super_env", "demo", "sessions")
        uuids = sorted(os.listdir(sess_root))
        self.assertEqual(len(uuids), 1, "应恰好一个会话目录：%s" % uuids)
        sid_file = os.path.join(sess_root, uuids[0], "session_id")
        self.assertTrue(os.path.isfile(sid_file), "session_id 未落盘（resume 契约）")
        with open(sid_file, encoding="utf-8") as f:
            sid = f.read().strip()
        self.assertTrue(sid, "session_id 内容为空")

        prompt_path = os.path.join(sess_root, uuids[0], "prompt.md")
        self.assertTrue(os.path.isfile(prompt_path), "prompt.md 未落盘")
        with open(prompt_path, encoding="utf-8") as f:
            p = f.read()

        # ---- 2) prompt 上下文契约（task4 核心） ----
        # 严格加载证据：gen_env skill 全文 inline（独特关键词命中）
        self.assertIn("门禁系统设计引导", p,
                      "prompt 未内嵌 gen_env skill 全文")
        self.assertIn("super-loop", p, "prompt 未内嵌 super-loop")
        # 工作区上下文块
        self.assertIn("## 工作区上下文", p, "缺「## 工作区上下文」块")
        ws_abs = os.path.join(self.wsdir, ".rick", "super_env", "demo")
        self.assertIn(ws_abs, p, "缺 super_env 绝对路径 %s" % ws_abs)
        self.assertIn(self.wsdir, p, "缺工作区根 %s" % self.wsdir)
        # env 生成约定
        self.assertIn(os.path.join(ws_abs, "env"), p, "缺 env 生成路径约定")
        # 观察手段清单
        for kw in ("观察手段", "rick super status --json", "events.jsonl",
                   "archive.json", "codes", "heartbeat.json", "pi agent",
                   "任意信息可读"):
            self.assertIn(kw, p, "观察手段缺关键词 %r" % kw)
        # 干预手段
        for kw in ("干预手段", "rick super event --type", "单写者"):
            self.assertIn(kw, p, "干预手段缺关键词 %r" % kw)
        # 无未替换变量
        self.assertNotIn("{{", p, "prompt 含未替换 {{...}} 变量")

        # fake pi 收到新会话参数链：--session-id <uuid> + prompt 注入
        # （macOS：Go 侧解析出物理路径 /private/var/...，python tempfile 是逻辑
        #   /var/...——两边 realpath 归一后比较，仓库既有惯例）
        argv1 = self._read_argv()
        self.assertIn("--session-id", argv1, "fake pi 未收到 --session-id：%s" % argv1)
        self.assertIn(sid, argv1, "fake pi 的 --session-id 值与落盘 uuid 不符")
        self.assertIn(os.path.realpath(prompt_path),
                      [os.path.realpath(a) for a in argv1],
                      "fake pi 未收到 prompt 注入（--append-system-prompt）：%s" % argv1)

        # ---- 3) resume 互通：--session <uuid> 加载语义 ----
        os.remove(self.argv_path)
        r2 = self._run_super("--name", "demo", "--resume")
        self.assertEqual(r2.returncode, 0,
                         "resume 失败：%s%s" % (r2.stdout, r2.stderr))
        argv2 = self._read_argv()
        self.assertIn("--session", argv2, "resume 未走 --session 加载：%s" % argv2)
        i = argv2.index("--session")
        self.assertLess(i + 1, len(argv2), "--session 缺值")
        self.assertEqual(argv2[i + 1], sid,
                         "resume 的 session id 与首建不一致（互通断裂）")
        self.assertNotIn("--append-system-prompt", argv2,
                          "resume 不应注入新提示词（加载语义）：%s" % argv2)


if __name__ == "__main__":
    unittest.main()
