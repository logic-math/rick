# task31：matmul env 模板三要素化

## 任务目标
模板对齐三要素契约：`skills/{profile,benchmark,verify,gate_check}.py` 四件，删除 self_check.py（AI 按需调三者，不再有混合自检）。

## 关键结果
- `super/env_templates/matmul/skills/`：
  - `profile.py`：迁移并**泛化** perf_profile（热点剖析 + 环境事实：numpy 版本/BLAS 后端/CPU 核数/内存带宽估算 + 父本结构摘要），输出单行 JSON `{hotspots: [...], env_facts: {...}, parent_facts: {...}}`
  - `benchmark.py`：由 data/bench.py 迁移（K 次计时/score/加固不变），输出 `{runtime_ms, score, timings, k, ref_runtime_ms}`
  - `verify.py`：由 data/test_correctness.py 迁移（双规模/多 seed/双跑/静态检查不变），输出 `{passed, max_error, stage, ...}`
  - `gate_check.py`：**新封装**——依次 verify → benchmark → 软门禁（`--baseline` 与 env.json `tolerance.soft_gate.relative_band` 比较）→ 输出 `{passed, failure_stage, metrics, details}`；`--env-dir` 缺省取脚本上两级
  - 删除 `skills/self_check.py`；`data/` 保留兼容（或留薄 wrapper 转发，避免破坏 env_7/8/9 之外的引用）——**决策：data/ 下改为薄转发（调 skills/ 对应脚本），保持 legacy 兼容**
- `super/env_templates/matmul/README.md` + `domain_knowledge.md`：三要素语义与用法（agent 侧怎么用/框架侧怎么用同一份资产）
- 验证（venv 内）：`profile.py --candidate code/reference.py` 出 JSON；`verify.py --candidate code/reference.py` passed=true；`benchmark.py --candidate code/reference.py` 出 runtime/score；`gate_check.py --candidate code/reference.py --baseline <ref*1.5>` passed=true 且 `--baseline 0.001` 时软门禁拒

# 写域
- super/env_templates/matmul/skills/**（新建四件 + 删 self_check）
- super/env_templates/matmul/data/**（改薄转发）
- super/env_templates/matmul/README.md、domain_knowledge.md

# 依赖关系
- 无（L2，写域独立）

# 测试方法
venv 四验（profile/verify/benchmark/gate_check 各出合法 JSON + reference 全过）+ 假解被 verify 拒 + 软门禁 band 判定正确

# 禁止事项
- 不改 super/superopt/**；不改 env_7/8/9 实际 env 目录（另由 parent 迁移 env_9）
