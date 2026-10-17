# skill:gen-env（super 进化 env 生成器）

## 触发场景

super 会话（super-loop）中出现以下情形时加载本 skill：

- **env 缺失**：`rick super start` 前目标工作区还没有 `env/`，需要现场生成一个可运行的进化环境
- **env 升级**：既有 env 需要演进——门禁加强（新增 exploit 样本/收紧容差）、评分方向调整、特征轴增改、任务域换版本
- 信号词：「建一个 env」「这个域的进化环境」「门禁怎么设计」「加特征轴」「收紧正确性容差」

matmul 类任务不必从零访谈：`super/env_templates/matmul/` 模板已是完整实例，本 skill 的流程用于**定制**它或从零建设其它任务域。

## 预期效果

产出一个**可运行的 env**——通过验收自检（见核心内容第 6 节）的完整环境，super 调度器可立即在其上跑进化循环：

- 目录协议落地（相对 super 工作区 `~/.rick/super/<name>/`）：

```
env/
├── env.json                 # 任务定义：任务名 / perf_desc（评分方向）/ perf_type（单位）/ 容差声明
├── feature_space.json       # 初始特征空间：2-4 轴（name/type/取值域），轴间正交
├── code/                    # 参考实现（正确性锚点）+ naive 起点（进化起点）
├── skills/                  # agent 自检脚本（快速反馈，非终验）
├── data/                    # 框架终验脚本：test_correctness.py + bench.py（stdout 单行 JSON 协议）
├── domain_knowledge.md      # 领域知识 + 「## 性能分析 SOP」「## 性能调优 SOP」两节
├── requirements.txt         # 非标准库依赖声明（如 numpy）
└── .venv/                   # 可选：依赖隔离（python3 -m venv；门禁优先用 .venv/bin/python）
gates/                       # 工作区级（非 env 内）：
├── exploit_registry.json    # 已知作弊/坏解样本（门先证红的种子）
└── golden_examples/         # 金样例（参考实现即首个金样例）
```

- 关键纪律：**门禁的容差/评分函数/阈值属于 env 设计范畴（human 决策），本 skill 只引导设计，不代答、不写死**；rick 框架（superopt gates.py）只提供级联门结构（环境健康→正确性→性能 + 软门禁多计时中位数）并读 env 的定义执行。

## 核心内容

访谈式流程——**逐环节引导 human 决策**，每环节产出物落盘后再进下一环节：

### 1. 任务定义

- 确定任务名、输入输出（例：两个 512×512 float32 矩阵 → 乘积矩阵）。
- **参考实现**：写 `env/code/reference.py`——用该生态最权威直白的写法（如 numpy.matmul）。它是正确性锚点（门先证红的金样例之首）。
- **naive 起点**：写 `env/code/naive_start.py`——无优化的朴实写法（如纯 Python 三重循环）。进化从这里起步，「慢」是特性不是缺陷。

### 2. 评分方向

- `perf_desc`：**low**（越小越好，如 runtime_ms）或 **high**（越大越好，如 score/吞吐）。
- `perf_type`：单位（runtime_ms / score / …）。
- 写入 `env.json`；同格替换的比较方向由框架按此字段感知——这是 human 的领域判断，不是框架默认。

### 3. 门禁系统设计引导（核心环节）

引导 human 逐项设计门禁（每项给出该域的具体选项，不代答）：

- **正确性门**（`env/data/test_correctness.py`）：输出 stdout 单行 JSON `{"passed": bool, "max_error": float}`。引导决策：
  - 校验规模（小矩阵快速校验，例 64×64）与固定 seed（可复现）
  - **容差 ε**：该域可接受的最大数值误差（1e-3？1e-5？按任务精度要求定）
  - 候选异常/超时如何判（框架把非 JSON/超时按该 stage 失败处理）
- **性能门**（`env/data/bench.py`）：输出 `{"runtime_ms": float, "score": float, "timings": [...]}`。引导决策：
  - bench 规模（例 512×512）、预热次数、计时次数（多计时 K=3 取中位数由框架机制保证，脚本内落 K 次或单次由框架调 K 次均可——**取脚本内 K=3 中位数**）
  - score 语义（相对参考实现的加速比？绝对吞吐？）
  - 超时上限（坏候选不能拖死门禁）
- **环境健康自检**（`env/skills/self_check.py`）：agent 自检脚本（coding agent 快速反馈用）：跑一遍正确性 + 单次 bench，输出 JSON 摘要。**自检仅供参考，入格以框架终验为准**。
- **exploit 注册表**（`<ws>/gates/exploit_registry.json`）：**引导 human 想「这个域里 agent 会怎么作弊」**，构造 3-5 个已知坏解样本（每条 {id, name, pattern, sample_path}）。常见模式参考：
  - 预存结果（返回缓存/硬编码答案）
  - 篡改计时（计时期间偷工/异步返回）
  - 降精度换速度（float64→float16 之类该域不可接受的精度损失）
  - 破坏输入后部分计算（只算可过校验的子集）
  - 样本写成可执行的小脚本放 `gates/` 下，注册表引用路径
- **金样例**（`<ws>/gates/golden_examples/`）：参考实现即首个金样例；可再补 1-2 个风格不同的已知好解。作用：门禁升级时防误杀（历史合法解必须继续通过）。

### 4. 初始特征空间

`env/feature_space.json` 的 axes：**2-4 轴起步**，每轴 {name, type, values/取值域, desc}。
- 引导原则：**轴间正交**——每根轴刻画解的一个独立维度（例：impl_strategy=算法策略轴 / memory_layout=数据排布轴；一个讲「怎么算」、一个讲「数据怎么放」，互不蕴含）。
- 轴取值域是「可枚举的行为描述子」（enum）或「可分档的量」——格 id 由框架从描述子确定性生成。
- 宁少勿滥：先 2 轴跑起来，进化轨迹显示解空间贫瘠再加轴（加轴走 env 升级路径）。

### 5. 初始解

- `env/code/naive_start.py` 即进化起点（第 1 节已落）。
- 框架行为：每迭代把选定父本整拷到 `codes/{uuid}/` 作为工作副本——env 的 code/ 目录保持只读圣洁。

### 6. 验收自检清单（env 交付前逐条过）

```bash
PY=env/.venv/bin/python   # 或系统 python3（无 .venv 时）
# ① 参考实现过正确性门
$PY env/data/test_correctness.py --candidate env/code/reference.py
# → {"passed": true, "max_error": ...}
# ② bench 出分
$PY env/data/bench.py --candidate env/code/reference.py
# → {"runtime_ms": >0, "score": ..., "timings": [3 个]}
# ③ exploit 样本被正确性门拒识（门先证红地基）
$PY env/data/test_correctness.py --candidate <每个 exploit 样本>
# → 全部 {"passed": false, ...}
# ④ naive 起点过正确性门（慢但必须对——它是进化起点）
$PY env/data/test_correctness.py --candidate env/code/naive_start.py
# → {"passed": true, ...}
```

四项全过 → env 合格，回 super-loop Step 2 启动（start 会再跑全量门先证红：exploit 拒识 100% + 金样例零误杀）。
任一不过 → 修对应环节（容差/样本/脚本），不带着病门禁启动进化。

### env 升级路径（触发场景之二）

- 改门禁（容差/评分/exploit 注册表）→ **必须重跑门先证红**（exploit 全拒 + 金样例全过），建议顺带跑双回归（elite 回放 + exploit 回归）；改完下迭代自动生效（调度器每迭代 subprocess 直调 pi，无需重启）。
- 加特征轴 → 新轴只影响后续采样；既有格不删（不删格原则），旧格降权走 adjust 提案。
- 升级访谈只需覆盖变更涉及的环节 + 第 6 节清单重过。
