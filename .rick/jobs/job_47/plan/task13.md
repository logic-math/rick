# task13：matmul env 模板 + super/README.md

## 任务目标

交付第一个 env 模板（矩阵乘，L3-Q5 裁决：CPU numpy 512×512 float32，2 轴初始特征空间）+ super/ 顶层 README（模块说明书）。gen_env skill（task12）以本模板为 matmul 起步参照；e2e（task14）用本模板现场生成 env 跑全链。

## 关键结果

- 新建 `super/env_templates/matmul/` 全套：
  - `env.json`：{task: "matmul", perf_desc: "low"（runtime_ms 越小越好）, perf_type: "runtime_ms", tolerance: {"correctness": {"max_error": 1e-3}, "soft_gate": {"relative_band": 0.05}}, description: "512×512 float32 矩阵乘性能优化"}
  - `feature_space.json`：axes 2 轴——{"name": "impl_strategy", "type": "enum", "values": ["naive_loops", "blocked", "vectorized", "strassen_like"], "desc": "实现策略"}；{"name": "memory_layout", "type": "enum", "values": ["row_major", "col_major", "tiled"], "desc": "内存布局"}（正交：策略轴=算法维度、布局轴=数据排布维度）
  - `code/reference.py`：numpy.matmul 参考实现（正确性锚点：`def matmul(a,b): return np.matmul(a,b)` + main 生成固定 seed 矩阵自跑输出耗时）
  - `code/naive_start.py`：纯 python 三重循环 naive 起点（进化起点；不用 numpy——展示「待优化」状态，bench 慢是特性）
  - `data/test_correctness.py`：正确性门——参数：candidate 路径 + 参考输出；生成固定 seed 的 64×64 小矩阵（快速校验），对比 candidate.matmul vs np.matmul，max_error ≤ tolerance → 输出 JSON `{"passed": true, "max_error": 1.2e-05}`；异常/超时 → `{"passed": false, "error": "..."}`；**输出协议 JSON 单行 stdout**（gates 框架契约）
  - `data/bench.py`：性能门——参数：candidate 路径；512×512 float32 固定 seed；预跑 1 次预热 + 计时 K 次（框架要求多计时，脚本自身只做单次计时由 gates 调 K 次；或脚本内做 K=3——**取脚本内 K=3** 输出中位数：`{"runtime_ms": 123.4, "score": 56.7, "timings": [...]}`，score = 基准分（相对参考实现的加速比））；候选超时（默认 60s）→ `{"passed": false, "error": "timeout"}`（v5 环境失败 78% 的防线之一）
  - `skills/self_check.py`：agent 自检脚本（coding agent 用的快速反馈，非终验）：跑 test_correctness + 单次 bench，输出 JSON 摘要（结果仅供参考，入格以框架 gates.py 为准——J6 裁决「以框架为准」）
  - `domain_knowledge.md`：领域知识 + `## 性能分析 SOP` + `## 性能调优 SOP` 两节（缓存友好转置/分块/向量化/Strassen 边界；profiling 方法：timeit/cProfile）
  - `requirements.txt`：numpy（gen_env 引导装 env/.venv）
- 新建 `super/README.md`：superopt 模块说明书——架构图（三层）/目录结构/快速上手（gen_env 生成 env → rick super start）/events 事件类型表/archive schema 概览/门禁协议（env 定义如何被 gates 消费）/mock 与 realpi 测试跑法
- **numpy 依赖隔离**：模板不依赖系统 numpy——gen_env/使用者建 `env/.venv`（`python3 -m venv .venv && .venv/bin/pip install -r requirements.txt`）；data/ 脚本 shebang 用 `#!/usr/bin/env python3` 但**由 gates 以显式解释器调用**（优先 `.venv/bin/python`，fallback `python3`）——解释器选择逻辑在 task7 gates（框架），模板只声明 requirements

# 写域

- super/env_templates/matmul/（整目录，新建）
- super/README.md（新建）

## 依赖关系

- 无（Kahn L1；纯数据/文档资产。与 task7 的接口契约：data/ 脚本 stdout 单行 JSON 协议——双方以设计树 §3.3 为事实源）

## 实现要点

- test_correctness 用小矩阵（64×64）求快、bench 用 512×512（L3-Q5）；固定 seed 保可复现
- naive_start 纯 python 三重循环 512×512 约 5-10s（可接受）；64×64 正确性校验约 0.1s
- 防作弊样例设计（给 exploit_registry 用的思路在 README 提示）：返回预存结果/篡改计时/降低精度换速度——e2e 的污染注入演练素材（task14 消费）

## 测试方法

- `python3 data/test_correctness.py --candidate code/reference.py`（以系统 python3+numpy？——**本机无 numpy**：验证方式=在临时 venv 装 numpy 跑一遍：参考实现 passed=true、naive passed=true、假解（返回错误结果的脚本）passed=false）
- `python3 data/bench.py --candidate code/reference.py`（venv 内）→ runtime_ms 合理（512² numpy matmul ~1-5ms）+ score 有值
- 目录/文件齐全性断言（gate1 做）：env.json/feature_space.json/code×2/data×2/skills×1/domain_knowledge.md/requirements.txt

## 禁止事项

- 不得用 torch（L3-Q6 修正：numpy，torch 留 CUDA 域）；不得把门禁阈值硬编码进 gates 框架（阈值在 env.json——模板自身就是示范）
