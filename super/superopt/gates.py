"""gates.py — job_47 task7：级联门框架（框架终验）+ 门先证红

级联三级 + 软门禁 + 门先证红（L2-Q4/L2-Q5，design-tree §2.6/§3.3）：
  stage1 环境健康 → stage2 正确性 → stage3 性能 → 软门禁
  前级失败短路（后级不再跑）；门先证红在启动时全量跑（exploit 拒识 100% + 金样例零误杀）。

L3-Q2 架构红线（本模块的唯一设计约束）：**框架零阈值假设**——门禁/容差/评分函数/阈值
属于 env 范畴；本模块只提供结构与机制（级联短路/多计时中位数/容差带比较/证红流程），
全部判定输入读 env 侧声明：

  env.json 框架消费键（相对 env_dir）：
    perf_type      性能值字段名（默认 "runtime_ms"——bench 输出 JSON 的同名字段即 perf 值）
    perf_desc      方向："low"（越小越好，缺省）/ "high"（越大越好）；其他值=env 配置错误
    tolerance.correctness.max_error   正确性容差——**data/test_correctness.py 自己读并比对**，
                                      框架只消费脚本的 passed 结论（框架不做数值比对）
    tolerance.soft_gate.relative_band 软门禁相对容差带（缺省 0.05）
    tolerance.soft_gate.baseline      可选基线 perf（无精英无历史时的比较锚点；
                                      缺省=首解豁免 passed=True 不带 tolerance_band）
    gates.correctness_timeout_s       stage2 子进程超时秒（缺省 120）
    gates.bench_timeout_s             stage3 单次 bench 子进程超时秒（缺省 600；
                                      注意与 bench 脚本内部的逐次计时超时区分——后者由
                                      env 自管，本超时是整进程兜底）

  data 脚本协议：`<python> data/<script> --candidate <path>`，stdout 输出单行 JSON
  （框架从后往前找最后一行可解析的 JSON object，容忍前置噪声输出）：
    data/test_correctness.py → {"passed": bool, "max_error": num, ...}
        缺 "passed" / 非 JSON / 超时 → stage2 失败
    data/bench.py → {"<perf_type>": num, "score": num(可选，env 定义，越高越好),
                     "ref_<perf_type>": num(可选), ...}
        也可输出 {"passed": false, "error": ...} 表示测量失败 → stage3 失败
        score 缺省时框架按 perf_desc 方向归一兜底（low→-value / high→+value，越高越好）

  软门禁基线解析顺序：当前格精英 perf（verify(elite_perf=...)）→ 同格历史中位数
  （verify(history=[...])，invalidation 后精英清空 history 尚在的场景）→ env.json
  tolerance.soft_gate.baseline → 首解豁免。带内（|ratio-1| ≤ band）→ passed 且
  metrics.soft.tolerance_band=True（触发延迟入库 L2-Q6）；带外劣化 → failure_stage=
  soft_gate（劣化解不杀格只不入格——格处置是 archive/loop 职责）；带外更优 → 放行。

  门先证红（gates_dir 相对）：
    exploit_registry.json = [{"id", "name"?, "pattern"?, "sample_path"}, ...]
      （顶层 JSON 数组；也容忍 {"exploits": [...]} 包裹形态。sample_path 相对路径按
        gates_dir → ws（gates_dir 父目录）两级解析；样本文件不存在=注册表完整性错误）
    golden_examples/      金样例目录（参考实现即首个金样例），其下每个文件 verify 须通过
    preverify_red() → {"ok": bool, "leaked": [exploit_id...],
                       "false_killed": [golden 文件名...], "exploits_checked": int,
                       "goldens_checked": int, "error"?: str}
      注册表缺失/空/malformed → ok=False（无红样本即无法证红，启动必须中止）

职责边界：gate 不写 events（调度器 loop 写 gate_result 事件）；gate 不判同格替换
（archive.admit 的职责）；解释器优先 env/.venv/bin/python，缺省 sys.executable。

每次 verify 重读 env.json（env 热更新下一迭代生效，J4）。
stdlib-only，Python ≥3.9（禁 PEP604 X|Y 注解，用 Optional[...]）。
"""
import json
import os
import subprocess
import sys
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Tuple

__all__ = [
    "GateResult",
    "CascadeGate",
    "STAGE1_ENV",
    "STAGE2_CORRECTNESS",
    "STAGE3_PERF",
    "SOFT_GATE",
    "FAILURE_STAGES",
]

# 失败阶段名（failure_stage 取值域；gate_result 事件 payload 消费同名约定）
STAGE1_ENV = "stage1_env"
STAGE2_CORRECTNESS = "stage2_correctness"
STAGE3_PERF = "stage3_perf"
SOFT_GATE = "soft_gate"
FAILURE_STAGES = (STAGE1_ENV, STAGE2_CORRECTNESS, STAGE3_PERF, SOFT_GATE)

# 框架级默认值（全部可被 env.json 覆盖；见模块 docstring 消费键清单）
DEFAULT_K = 3                       # 软门禁计时次数（构造参数 k，config.soft_gate_k 可调）
DEFAULT_RELATIVE_BAND = 0.05        # tolerance.soft_gate.relative_band 缺省
DEFAULT_CORRECTNESS_TIMEOUT_S = 120.0   # gates.correctness_timeout_s 缺省
DEFAULT_BENCH_TIMEOUT_S = 600.0         # gates.bench_timeout_s 缺省
DEFAULT_PERF_TYPE = "runtime_ms"    # perf_type 缺省
DEFAULT_PERF_DESC = "low"           # perf_desc 缺省

# details 里 stdout/stderr 尾部截断长度（环境失败结论化的诊断素材）
_TAIL_CHARS = 200


# ---- 纯函数 ----

def _median(values: List[float]) -> Optional[float]:
    """中位数（偶数个取中间两数均值；空列表 None）。"""
    vs = sorted(float(v) for v in values)
    m = len(vs)
    if m == 0:
        return None
    if m % 2 == 1:
        return vs[m // 2]
    return (vs[m // 2 - 1] + vs[m // 2]) / 2.0


def _num_or_none(value: Any) -> Optional[float]:
    """真数值才转 float（bool 排除——bool 是 int 子类）；其余 None。"""
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)
    return None


def _tail(text: str, n: int = _TAIL_CHARS) -> str:
    if not text:
        return ""
    return text.replace("\r", "")[-n:]


def _coerce_text(value: Any) -> str:
    """subprocess 异常里的输出可能是 bytes/str/None——统一成 str。"""
    if value is None:
        return ""
    if isinstance(value, bytes):
        return value.decode("utf-8", "replace")
    return str(value)


def _parse_protocol_json(stdout_text: str) -> Optional[dict]:
    """解析 data 脚本的 stdout 协议：从后往前找最后一行可解析的 JSON object。

    协议理想形态是「单行 JSON」；候选脚本在 JSON 行之前打噪声（candidate import 期间
    print 等）不应破坏协议——最后一行 JSON 即权威输出。全部不可解析 → None。
    """
    if not stdout_text:
        return None
    lines = [ln for ln in stdout_text.split("\n") if ln.strip()]
    for raw in reversed(lines):
        try:
            obj = json.loads(raw.strip())
        except ValueError:
            continue
        if isinstance(obj, dict):
            return obj
    return None


def _classify_soft(value: float, baseline: float, band: float, perf_desc: str) -> str:
    """软门禁带判定（纯函数）：返回 "better" / "band" / "worse"。

    带内定义：|value - baseline| ≤ baseline × band（ratio ∈ [1-band, 1+band]）。
    perf_desc="low"（越小越好）：ratio > 1+band 劣化；< 1-band 更优。
    perf_desc="high" 反向。baseline ≤ 0 的退化场景单独处理（等值=带内；
    low：value>0 劣化；high：value<0 劣化）。
    """
    if value == baseline:
        return "band"
    if baseline > 0:
        ratio = value / baseline
        if perf_desc == "high":
            if ratio < 1.0 - band:
                return "worse"
            if ratio > 1.0 + band:
                return "better"
            return "band"
        # low（默认方向）
        if ratio > 1.0 + band:
            return "worse"
        if ratio < 1.0 - band:
            return "better"
        return "band"
    # 退化基线（≤0）：等值已排除
    if perf_desc == "high":
        return "worse" if value < 0 else "better"
    return "worse" if value > 0 else "better"


# ---- 结果结构 ----

@dataclass
class GateResult:
    """级联门终验结果。

    metrics 三段（到达哪级填哪级）：
      correctness: data/test_correctness.py 的输出 dict（passed 归一为 bool）
      perf:        {perf_type, perf_desc, value(K 次中位数), timings[K], score,
                    ref_value, k}
      soft:        {relative_band, baseline, baseline_source(elite|history|env|none),
                    tolerance_band, exempt, verdict(better|band|worse|exempt), ratio?}
    tolerance_band 便捷属性供 loop 做延迟入库判定（L2-Q6）。
    """

    passed: bool = False
    failure_stage: Optional[str] = None
    metrics: Dict[str, Any] = field(default_factory=dict)
    details: str = ""

    @property
    def tolerance_band(self) -> bool:
        return bool(self.metrics.get("soft", {}).get("tolerance_band"))


# ---- 级联门 ----

class CascadeGate:
    """级联门框架：verify() 单候选终验 / preverify_red() 门先证红。

    用法::

        gate = CascadeGate(ws + "/env", ws + "/gates", k=cfg["soft_gate_k"])
        r = gate.verify(code_path, elite_perf=elite.get("perf_value"),
                        history=[h["perf_value"] for h in cell["history"]])
        if r.passed and r.tolerance_band: ...  # 延迟入库

    构造不读盘（所有 env 读取延迟到 verify——每次重读，env 热更新生效）。
    """

    def __init__(self, env_dir: str, gates_dir: str, k: int = DEFAULT_K) -> None:
        if not isinstance(k, int) or isinstance(k, bool) or k < 1:
            raise ValueError("k must be an int >= 1, got %r" % (k,))
        self.env_dir = env_dir
        self.gates_dir = gates_dir
        self.k = k

    # ---- 解释器选择 ----

    def _python(self) -> str:
        """env/.venv/bin/python 存在则用之（numpy 等域依赖装在 env venv），否则 sys.executable。"""
        venv = os.path.join(self.env_dir, ".venv", "bin", "python")
        if os.path.exists(venv):
            return venv
        return sys.executable

    # ---- env.json 读取 ----

    def _load_env_json(self) -> Tuple[Optional[dict], Optional[str]]:
        path = os.path.join(self.env_dir, "env.json")
        if not os.path.isfile(path):
            return None, "env.json missing: %s" % path
        try:
            with open(path, "r", encoding="utf-8") as f:
                cfg = json.load(f)
        except (OSError, ValueError) as e:
            return None, "env.json unreadable: %s (%s)" % (path, e)
        if not isinstance(cfg, dict):
            return None, "env.json top level must be a JSON object: %s" % path
        return cfg, None

    @staticmethod
    def _gates_cfg(cfg: dict) -> dict:
        g = cfg.get("gates")
        return g if isinstance(g, dict) else {}

    @staticmethod
    def _soft_cfg(cfg: dict) -> dict:
        tol = cfg.get("tolerance")
        if not isinstance(tol, dict):
            return {}
        sg = tol.get("soft_gate")
        return sg if isinstance(sg, dict) else {}

    def _timeouts(self, cfg: dict) -> Tuple[float, float]:
        g = self._gates_cfg(cfg)
        c = _num_or_none(g.get("correctness_timeout_s"))
        b = _num_or_none(g.get("bench_timeout_s"))
        return (c if c is not None and c > 0 else DEFAULT_CORRECTNESS_TIMEOUT_S,
                b if b is not None and b > 0 else DEFAULT_BENCH_TIMEOUT_S)

    def _band_value(self, cfg: dict) -> float:
        v = _num_or_none(self._soft_cfg(cfg).get("relative_band"))
        return v if v is not None else DEFAULT_RELATIVE_BAND

    def _validate_cfg(self, cfg: dict) -> List[str]:
        """env.json 框架消费键的静态校验（配错=环境不健康，stage1 拒）。"""
        errors: List[str] = []
        perf_desc = cfg.get("perf_desc", DEFAULT_PERF_DESC)
        if perf_desc not in ("low", "high"):
            errors.append("env.json perf_desc must be 'low' or 'high', got %r" % (perf_desc,))
        perf_type = cfg.get("perf_type", DEFAULT_PERF_TYPE)
        if not isinstance(perf_type, str) or not perf_type:
            errors.append("env.json perf_type must be a non-empty string, got %r" % (perf_type,))
        sg = self._soft_cfg(cfg)
        band = sg.get("relative_band")
        if band is not None:
            b = _num_or_none(band)
            if b is None or b < 0:
                errors.append(
                    "tolerance.soft_gate.relative_band must be a number >= 0, got %r" % (band,))
        baseline = sg.get("baseline")
        if baseline is not None and _num_or_none(baseline) is None:
            errors.append("tolerance.soft_gate.baseline must be a number, got %r" % (baseline,))
        g = self._gates_cfg(cfg)
        for key in ("correctness_timeout_s", "bench_timeout_s"):
            v = g.get(key)
            if v is not None:
                n = _num_or_none(v)
                if n is None or n <= 0:
                    errors.append("gates.%s must be a number > 0, got %r" % (key, v))
        return errors

    # ---- 子进程 ----

    def _probe_interpreter(self, py: str, timeout_s: float) -> Tuple[bool, str]:
        """stage1 解释器探活：跑不通=环境坏（含坏 venv）。"""
        try:
            r = subprocess.run([py, "-c", "import sys"], timeout=timeout_s,
                               capture_output=True, text=True)
        except subprocess.TimeoutExpired:
            return False, "interpreter probe timeout after %.1fs: %r" % (timeout_s, py)
        except OSError as e:
            return False, "interpreter not runnable: %r (%s)" % (py, e)
        if r.returncode != 0:
            return False, ("interpreter probe failed rc=%s: %r; stderr_tail=%r"
                           % (r.returncode, py, _tail(_coerce_text(r.stderr))))
        return True, ""

    def _run_script(self, py: str, script_rel: str, candidate: str,
                    timeout_s: float) -> Tuple[Optional[dict], Optional[str]]:
        """跑一个 data 脚本：成功 → (parsed_json, None)；失败 → (None, 诊断串)。

        失败含：子进程超时（TimeoutExpired）/ 无法 spawn / stdout 无可解析 JSON。
        诊断串带 stderr 尾 200 字（环境失败结论化素材）。
        """
        script = os.path.join(self.env_dir, script_rel)
        cmd = [py, script, "--candidate", str(candidate)]
        try:
            r = subprocess.run(cmd, timeout=timeout_s, capture_output=True, text=True)
        except subprocess.TimeoutExpired as e:
            msg = "timeout after %.1fs: %s %s" % (timeout_s, py, script_rel)
            tail = _tail(_coerce_text(e.stderr))
            if tail:
                msg += "; stderr_tail=%r" % tail
            return None, msg
        except OSError as e:
            return None, "failed to spawn %r: %s" % (py, e)
        stdout = _coerce_text(r.stdout)
        stderr = _coerce_text(r.stderr)
        parsed = _parse_protocol_json(stdout)
        if parsed is None:
            return None, ("non-JSON stdout (rc=%s); stdout_tail=%r; stderr_tail=%r"
                          % (r.returncode, _tail(stdout), _tail(stderr)))
        return parsed, None

    # ---- verify：三级级联 + 软门禁 ----

    def verify(self, candidate_dir: str,
               elite_perf: Optional[float] = None,
               history: Optional[List[float]] = None) -> GateResult:
        """级联终验一个候选。前级失败短路（后级不跑）。

        candidate_dir：候选路径（文件或目录，透传给 data 脚本的 --candidate；
        env 的脚本决定如何消费）。
        elite_perf / history：软门禁比较锚点（当前格精英 perf / 同格历史 perf 列表），
        由调度器从 archive 取（依赖注入——框架不读档案）。
        """
        metrics: Dict[str, Any] = {}

        # ---- stage1 环境健康 ----
        cfg, err = self._load_env_json()
        if err is not None:
            return GateResult(False, STAGE1_ENV, metrics,
                              "stage1 env unhealthy: %s" % err)
        py = self._python()
        c_timeout, b_timeout = self._timeouts(cfg)

        problems = self._validate_cfg(cfg)
        for rel in ("data/test_correctness.py", "data/bench.py"):
            if not os.path.isfile(os.path.join(self.env_dir, rel)):
                problems.append("missing env/%s" % rel)
        ref_path = os.path.join(self.env_dir, "code", "reference.py")
        if not os.path.isfile(ref_path):
            problems.append("missing env/code/reference.py")
        if not candidate_dir or not os.path.exists(candidate_dir):
            problems.append("candidate not found: %r" % (candidate_dir,))
        if not problems:
            ok, msg = self._probe_interpreter(py, c_timeout)
            if not ok:
                problems.append(msg)
        if not problems:
            # 参考实现必须过正确性门（参考挂=环境坏——v5 环境失败 78% 的头部防线）
            parsed, rerr = self._run_script(py, "data/test_correctness.py",
                                            ref_path, c_timeout)
            if rerr is not None:
                problems.append("reference failed correctness gate: %s" % rerr)
            elif not parsed.get("passed"):
                problems.append("reference failed correctness gate: %s"
                                % json.dumps(parsed, ensure_ascii=False))
        if problems:
            return GateResult(False, STAGE1_ENV, metrics,
                              "stage1 env unhealthy: " + "; ".join(problems))

        # ---- stage2 正确性（容差由 env 脚本自己比对，框架只消费 passed） ----
        parsed, serr = self._run_script(py, "data/test_correctness.py",
                                        candidate_dir, c_timeout)
        if serr is not None:
            metrics["correctness"] = {"passed": False, "error": serr}
            return GateResult(False, STAGE2_CORRECTNESS, metrics,
                              "stage2 correctness failed: %s" % serr)
        if "passed" not in parsed:
            msg = "correctness output missing 'passed': %r" % (parsed,)
            metrics["correctness"] = {"passed": False, "error": msg}
            return GateResult(False, STAGE2_CORRECTNESS, metrics,
                              "stage2 correctness failed: %s" % msg)
        c_metrics = dict(parsed)
        c_metrics["passed"] = bool(parsed["passed"])
        metrics["correctness"] = c_metrics
        if not c_metrics["passed"]:
            return GateResult(False, STAGE2_CORRECTNESS, metrics,
                              "stage2 correctness failed: %s"
                              % json.dumps(parsed, ensure_ascii=False))

        # ---- stage3 性能：K 次独立进程计时 → 中位数 ----
        perf_type = cfg.get("perf_type", DEFAULT_PERF_TYPE)
        perf_desc = cfg.get("perf_desc", DEFAULT_PERF_DESC)
        values: List[float] = []
        scores: List[float] = []
        refs: List[float] = []
        for i in range(self.k):
            bparsed, berr = self._run_script(py, "data/bench.py",
                                             candidate_dir, b_timeout)
            if berr is not None:
                metrics["perf"] = {"error": berr}
                return GateResult(
                    False, STAGE3_PERF, metrics,
                    "stage3 perf failed (bench run %d/%d): %s" % (i + 1, self.k, berr))
            if bparsed.get("passed") is False:
                msg = ("bench reported failure (run %d/%d): %s"
                       % (i + 1, self.k, json.dumps(bparsed, ensure_ascii=False)))
                metrics["perf"] = {"error": msg}
                return GateResult(False, STAGE3_PERF, metrics,
                                  "stage3 perf failed: %s" % msg)
            v = _num_or_none(bparsed.get(perf_type))
            if v is None:
                msg = ("bench output missing numeric %r (run %d/%d): %r"
                       % (perf_type, i + 1, self.k, bparsed))
                metrics["perf"] = {"error": msg}
                return GateResult(False, STAGE3_PERF, metrics,
                                  "stage3 perf failed: %s" % msg)
            values.append(v)
            s = _num_or_none(bparsed.get("score"))
            if s is not None:
                scores.append(s)
            rv = _num_or_none(bparsed.get("ref_" + perf_type))
            if rv is not None:
                refs.append(rv)

        value = _median(values)
        if len(scores) == self.k:
            score = _median(scores)
        else:
            # env bench 未声明 score → 按 env.json perf_desc 方向归一兜底（越高越好）
            score = -value if perf_desc == "low" else value
        ref_value = _median(refs) if len(refs) == self.k else None
        metrics["perf"] = {
            "perf_type": perf_type,
            "perf_desc": perf_desc,
            "value": value,
            "timings": values,
            "score": score,
            "ref_value": ref_value,
            "k": self.k,
        }

        # ---- 软门禁 ----
        ok_soft, soft_metrics, sdetails = self._soft_gate(
            value, perf_desc, cfg, elite_perf, history)
        metrics["soft"] = soft_metrics
        if not ok_soft:
            return GateResult(False, SOFT_GATE, metrics, sdetails)

        return GateResult(
            True, None, metrics,
            "passed: correctness(max_error=%s), perf(%s=%s, k=%d, score=%s), "
            "soft(%s)" % (c_metrics.get("max_error"), perf_type, value, self.k,
                          score, soft_metrics.get("verdict")))

    # ---- 软门禁 ----

    def _soft_gate(self, value: float, perf_desc: str, cfg: dict,
                   elite_perf: Optional[float],
                   history: Optional[List[float]]) -> Tuple[bool, dict, str]:
        """与基线比较：带内 → 通过+tolerance_band；带外劣化 → soft_gate 拒；带外更优 → 通过。"""
        band = self._band_value(cfg)
        baseline: Optional[float] = None
        source = "none"
        ep = _num_or_none(elite_perf)
        if ep is not None:
            baseline, source = ep, "elite"
        else:
            nums: List[float] = []
            if history:
                for h in history:
                    hn = _num_or_none(h)
                    if hn is not None:
                        nums.append(hn)
            if nums:
                baseline, source = _median(nums), "history"
            else:
                eb = _num_or_none(self._soft_cfg(cfg).get("baseline"))
                if eb is not None:
                    baseline, source = eb, "env"
        soft: Dict[str, Any] = {
            "relative_band": band,
            "baseline": baseline,
            "baseline_source": source,
            "tolerance_band": False,
            "exempt": False,
        }
        if baseline is None:
            # 首解豁免：无任何基线可比（新格首解）——通过且不带 tolerance_band
            soft["exempt"] = True
            soft["verdict"] = "exempt"
            return True, soft, ""
        if baseline > 0:
            soft["ratio"] = value / baseline
        verdict = _classify_soft(value, baseline, band, perf_desc)
        soft["verdict"] = verdict
        if verdict == "worse":
            return False, soft, (
                "soft gate: perf %s=%s worse than %s baseline=%s beyond "
                "relative_band=%.4f (劣化解不杀格，仅不入格)"
                % (perf_desc, value, source, baseline, band))
        if verdict == "band":
            soft["tolerance_band"] = True
        return True, soft, ""

    # ---- 门先证红 ----

    def _resolve_sample_path(self, sample_path: Any) -> Optional[str]:
        """绝对路径原样；相对路径按 gates_dir → ws（gates_dir 父目录）两级解析。"""
        if not isinstance(sample_path, str) or not sample_path:
            return None
        if os.path.isabs(sample_path):
            return sample_path
        for base in (self.gates_dir, os.path.dirname(self.gates_dir)):
            cand = os.path.join(base, sample_path)
            if os.path.exists(cand):
                return cand
        return os.path.join(self.gates_dir, sample_path)

    def preverify_red(self) -> Dict[str, Any]:
        """门先证红：exploit 注册表全拒（拒识 100%）+ 金样例全过（零误杀）。

        返回 {"ok", "leaked", "false_killed", "exploits_checked", "goldens_checked"}
        (+失败时 "error")。注册表缺失/空/malformed/样本缺失 → ok=False（无红样本
        即无法证红，调用方（调度器 --preverify-red）须以非 0 退出中止启动）。
        """
        result: Dict[str, Any] = {
            "ok": False,
            "leaked": [],
            "false_killed": [],
            "exploits_checked": 0,
            "goldens_checked": 0,
        }
        reg_path = os.path.join(self.gates_dir, "exploit_registry.json")
        if not os.path.isfile(reg_path):
            result["error"] = "exploit registry missing: %s" % reg_path
            return result
        try:
            with open(reg_path, "r", encoding="utf-8") as f:
                reg = json.load(f)
        except (OSError, ValueError) as e:
            result["error"] = "exploit registry unreadable: %s (%s)" % (reg_path, e)
            return result
        if isinstance(reg, dict):
            # 容忍包裹形态 {"exploits": [...]} / {"samples": [...]}
            reg = reg.get("exploits", reg.get("samples"))
        if not isinstance(reg, list):
            result["error"] = ("exploit registry must be a JSON array of "
                               "{id, sample_path}: %s" % reg_path)
            return result
        if not reg:
            result["error"] = ("exploit registry is empty (need >=1 known-bad "
                               "sample to prove red): %s" % reg_path)
            return result

        for entry in reg:
            if (not isinstance(entry, dict) or "id" not in entry
                    or "sample_path" not in entry):
                result["error"] = ("exploit registry entry malformed "
                                   "(need id+sample_path): %r" % (entry,))
                return result
            sp = self._resolve_sample_path(entry["sample_path"])
            if sp is None or not os.path.exists(sp):
                result["error"] = ("exploit sample not found (id=%r sample_path=%r)"
                                   % (entry.get("id"), entry["sample_path"]))
                return result
            r = self.verify(sp)
            result["exploits_checked"] += 1
            if r.passed:
                result["leaked"].append(entry["id"])

        golden_dir = os.path.join(self.gates_dir, "golden_examples")
        if os.path.isdir(golden_dir):
            for fname in sorted(os.listdir(golden_dir)):
                if fname.startswith("."):
                    continue
                fpath = os.path.join(golden_dir, fname)
                if not os.path.isfile(fpath):
                    continue
                r = self.verify(fpath)
                result["goldens_checked"] += 1
                if not r.passed:
                    result["false_killed"].append(fname)

        result["ok"] = (not result["leaked"]) and (not result["false_killed"])
        return result


def _module_selfcheck() -> int:
    """python3 super/superopt/gates.py → 纯函数快速自检（中位数 + 软门禁带判定）。"""
    assert _median([100, 300, 200]) == 200.0
    assert _median([100, 300]) == 200.0
    assert _median([5]) == 5.0
    assert _median([]) is None
    assert _classify_soft(120.0, 100.0, 0.05, "low") == "worse"
    assert _classify_soft(105.0, 100.0, 0.05, "low") == "band"
    assert _classify_soft(102.0, 100.0, 0.05, "low") == "band"
    assert _classify_soft(80.0, 100.0, 0.05, "low") == "better"
    assert _classify_soft(50.0, 100.0, 0.05, "high") == "worse"
    assert _classify_soft(102.0, 100.0, 0.05, "high") == "band"
    assert _classify_soft(150.0, 100.0, 0.05, "high") == "better"
    assert _classify_soft(100.0, 100.0, 0.05, "low") == "band"
    assert _parse_protocol_json('noise\n{"passed": true}\n') == {"passed": True}
    assert _parse_protocol_json("") is None
    assert _parse_protocol_json("no json at all") is None
    print("gates.py selfcheck OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(_module_selfcheck())
