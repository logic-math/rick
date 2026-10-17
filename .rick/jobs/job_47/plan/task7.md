# task7：gates.py 级联门框架 + 门先证红

## 任务目标

实现框架终验门：级联三级（环境健康→正确性→性能+软门禁）+ 门先证红（exploit 拒识 100%）+ 软门禁（多计时中位数 + 浮动容差）。**L3-Q2 架构修正的落地**：gates.py 是通用框架，读 env 门禁定义（env.json 容差 + data/ 脚本 + gates/exploit_registry.json），零硬编码阈值——门禁设计属 env 范畴，框架只提供结构与机制。

## 关键结果

- 新建 `super/superopt/gates.py`（stdlib-only）：
  - `@dataclass GateResult`：{passed, failure_stage: Optional[stage1_env|stage2_correctness|stage3_perf|soft_gate], metrics: {correctness: {...}, perf: {...}, soft: {...}}, details: str}
  - `class CascadeGate`：
    - `__init__(self, env_dir: str, gates_dir: str, k: int = 3)`——env_dir=工作区 env/；gates_dir=工作区 gates/；k=软门禁计时次数（K=3 默认，config 可调）
    - `verify(self, candidate_dir: str) -> GateResult`——三级级联（前级失败短路不跑后级）：
      - **stage1 环境健康**：python 解释器可用（优先 env/.venv/bin/python fallback python3）+ data/test_correctness.py 与 data/bench.py 存在可执行 + env/code/reference.py 参考实现可跑（跑一次 test_correctness 对参考实现——参考实现挂=环境坏，是 v5 环境失败 78% 的头部防线）+ candidate 文件存在
      - **stage2 正确性**：subprocess 调 `<python> data/test_correctness.py --candidate <candidate_dir>`（timeout 120s，env.json 可调）→ 解析 stdout 单行 JSON {passed, max_error}——passed=false 或解析失败/超时 → failure_stage=stage2_correctness
      - **stage3 性能**：调 `<python> data/bench.py --candidate <candidate_dir>` **K 次**（每次独立进程）→ 取 runtime_ms 中位数（多计时中位数机制）→ score 按 env.json perf_desc 方向归一 → **软门禁**：与「当前格精英 perf（无精英=env.json 基线或首解豁免）/ 同格历史中位数」比较，在 relative_band（env.json tolerance.soft_gate.relative_band，默认 0.05）容差带内 → 通过但标记 tolerance_band（触发延迟入库 L2-Q6）；超出带且劣于 → failure_stage=soft_gate（劣化解不杀格只不入格）
    - `preverify_red(self) -> dict`——**门先证红**：读 gates_dir/exploit_registry.json 的全部样本（{id, sample_path}），逐个跑 verify()——要求**100% 被拒**（任一 exploit 通过 → 返回 {ok: false, leaked: [ids]}）；金样例 gates_dir/golden_examples/ 全部 verify 通过（任一金样例被拒 → {ok: false, false_killed: [...]}，防误杀）；两者都过 → {ok: true}
  - **解释器选择**：`_python(self) -> str`——env/.venv/bin/python 存在则用之，否则 sys.executable
  - **超时与输出协议容错**：data/ 脚本超时/非 JSON 输出 → 对应 stage 失败 + details 记 stderr 尾 200 字（环境失败结论化的诊断素材）
- 新建 `super/tests/test_gates.py`（unittest）——**用内置 fixture env**（测试自建 mini env：tmpdir 内按目录协议放 env.json + data/test_correctness.py（简化：比对固定值）+ data/bench.py（sleep 模拟）+ exploit 样本（返回错误值的假解）+ 金样例（正确解））：
  - 级联短路：stage2 挂 → 不跑 stage3（bench 调用计数=0）
  - 正确性拒坏解 / 放行好解
  - 多计时中位数：K=3 次中位数断言（mock bench 返回 [100,300,200] → 取 200）
  - 软门禁容差带：band 内 → passed+tolerance_band 标记；band 外劣化 → soft_gate 失败
  - **门先证红**：exploit 全拒 + 金样例全过 → ok=true；放一个 exploit 泄漏 → ok=false leaked 非空；金样例误杀 → false_killed 非空
  - env 定义驱动：改 env.json 容差（1e-3→1e-9）→ 同一坏解从通过变拒绝（**框架零硬编码的证明断言**）

# 写域

- super/superopt/gates.py（新建）
- super/tests/test_gates.py（新建）

## 依赖关系

- task1/task2/task3/task4（Kahn L2；import events 的 EVENT_TYPES 仅用于事件字符串——不 import，用字面量）

## 实现要点

- gate 不写 events（调度器 loop 写 gate_result 事件——职责分离）；gate 不判同格替换（archive.admit 的职责）
- preverify_red 在 rick super start 时由 Go 侧触发（task11 调 subprocess `python3 -m superopt.gates --preverify <ws>`？——**不做 CLI 子模块**：Go 直接读调度器启动输出的判断？取协议：task11 的 start 流程 spawn 调度器时传 `--preverify-red` flag，调度器启动先跑 preverify_red 写 session_started 事件 + 失败即退出非 0，Go 侧由 OnDead/exit code 感知——契约写入 task11/task8 双方 spec）
- verify 的 perf 归一：无精英基线时首解豁免（passed=true 不带 tolerance_band）

## 测试方法

`python3 -m unittest super.tests.test_gates -v` 全绿（fixture env 全 subprocess 真跑，秒级）

## 禁止事项

- 不得硬编码任何阈值/评分函数（全读 env.json + data/ 脚本输出——L3-Q2 的验收红线）；不得 import matmul 模板细节
