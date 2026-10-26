# 实验记录 · 迭代 15 · dispatch_target=np_matmul|impl_strategy=full_blas_dispatch

## 1. 背景与价值（为什么现在做这个实验）

- **格内史**：精英 = iter003 plain 整阵 np.matmul（430.83ms，profile matmul self 100%，
  包装零开销）。本格已重访两次：iter13（V2 fresh+memset+out=，447.25 平价拒绝）、
  iter14（P1 持久缓冲/P5/P6 spacer/P7 dot+out，探针 6 形态同 floor ≈433ms，
  minflt/call=0，gate 432.81 band 仍被框架 worse_than_elite 拒）。
  **缓冲管理维度（分配/out=/预 fault/持久/布局）全部证伪**；结构维度（M 分解、换入口）
  在相邻格全部坍缩 434-488 平价带。
- **新事实（本轮 sysctl + 探针，历次迭代从未刻画）**：本机 Apple M5 Max = 6 P 核 +
  12 E 核（18 线程）。vecLib 默认 BLAS 池撒满 18 核——**sgemm floor 里含 E 核线程
  过栅栏的 straggler 成本**。交错 A/B 实测：`VECLIB_MAXIMUM_THREADS=6` floor
  429.5/425.3 vs default 441.6（深 12-16ms，~3%），且 6 线程臂两轮 floor 低破 430
  （精英录取线 430.83 之下的唯一合法读数来源）。
- **理论依据**：arXiv:1506.08988 / 1511.02171——AMP 上 BLAS 线程撒满全核是公认
  次优形态，限制线程到快集群是标准修正（详见 research_report §2/§3）。
- **时机**：预算最后一轮（15/15）。这是本格**最后一个未测过且有真实 floor 收益的
  合法微变体**——线程池配置维度；失败即收口「该格 = BLAS 平价天花板」的最终证据。

## 2. 实验假设（一句话，可证伪）

「在 solution.py 模块加载期设置 `VECLIB_MAXIMUM_THREADS=6`（= P 核数，先于首次
BLAS 调用、进程级对称），可使 benchmark.py 协议（fresh 进程 warmup1+K3 中位）下的
8192 sgemm runtime 中位数比同窗口未设置旋钮的父本低 ≥10ms（floor 深至 ≤435ms），
且 verify max_error = 0.0（逐位一致）。」

## 3. 验证逻辑（什么结果支持假设、什么结果否定它）

- **支持**：① benchmark timings 的 min（floor）≤435ms 且中位数低于同窗口 plain 父本
  对照 ≥10ms（A/B 探针口径：v6 floor 429.5/425.3 vs default 441.6）；② verify
  max_error=0.0（双规模 [64,8192]、双 seed、两遍一致）；③ score ≈ 1.0 平价
  （ref 同池同步变快，ref≈429-435 落 [380,700] 带内）。
- **否定**：① 候选 floor ≥441ms 且与 default 不可区分（噪声级）→ vecLib 默认池
  实际未含 E 核线程（旋钮对本机无 straggler 可省）；② max_error > 0（按 SHA 证据
  预期不会发生；若发生 = 池行为与探针不符，立即否决）；③ 任何加固通道触发
  （score>2 / ref 出带 / 残留子进程）→ 实现走偏，否决。
- **对照**：父本 plain np.matmul（无旋钮）= default 臂；探针数据已给同窗口对照
  （probes/thread_ab.json 三轮换序交错）。
- **已知风险（如实声明，非假设内容）**：录取是框架级 perf < 430.83 的比较
  （iter14 先例：432.81 band 仍拒）。v6 安静窗中位 428.9-433.2 → 恰在录取线两侧；
  当日存在 580-590ms 外部争用窗（protocol 模拟臂实测撞上），撞窗即 band/worse。
  **机制成立与否（floor 深度 + max_error）与录取与否（绝对中位 vs 430.83）是两件事，
  第 6 段必须分开记录。**

## 4. 具体实验步骤（可复现：改哪、怎么改、跑什么命令）

1. **复制父本**：`codes/iter003-0de54e75/solution.py` → 本迭代 `solution.py`，
   其余逐行不动（契约：matmul(a,b) asarray f32 → (n,m) ndarray；main 自检协议；
   np.seterr 三 ignore）。
2. **唯一改动（单变量）**：模块顶部 `import os`，并在 `import numpy as np` **之前**加：
   `os.environ.setdefault("VECLIB_MAXIMUM_THREADS", "6")`（依据 research_report §4 第 1 条：
   1506.08988/1511.02171 限线程到快集群 + 本机 A/B v6 最深 floor；setdefault 不覆盖
   外部显式配置；BLAS 池在首次 gemm 调用才创建，late-set 已探针证明生效）。
   numpy 版签名与求和序零改动。
3. **TDD（rick canon）**：先写 `test_solution.py` 看红（父本无该 env 设置）再实现。
   区分性测试：加载 solution 模块后 `os.environ["VECLIB_MAXIMUM_THREADS"] == "6"`；
   已显式设置时不覆盖（setdefault 语义）；同输入两次调用逐位一致；list/array 双形态
   输入契约。其余照抄 iter014 测试骨架（契约/不变量测试）。
4. **门禁三件每件至多一次、读数即记录（iter14 过程纪律）**：
   - 自检：`env/.venv/bin/python solution.py --n 8192` → 预期
     `max_error_vs_matmul = 0.0`。
   - `env/.venv/bin/python env/skills/verify.py --candidate solution.py` → 预期
     passed=true, max_error=0.0。
   - `env/.venv/bin/python env/skills/benchmark.py --candidate solution.py` → 预期
     timings floor ≤436、score≈1.0。
   - `env/.venv/bin/python env/skills/gate_check.py --candidate solution.py
     --baseline 430.8255829999998` → 软门禁判定（band/better/worse 如实记录）。
5. **第 6 段回写**：候选 timings/min + 同窗口 default 对照 + 框架判定分开记录；
   撞争用窗（≥550 读数）如实标注，不复跑、不追噪声。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- **正确性**：verify passed=true，max_error=0.0（n=64/8192 双 seed 两遍）；
  自检 max_error_vs_matmul=0.0（SHA 逐位证据：跨线程数 n=64/8192 哈希相同）。
- **性能**（runtime_ms，越低越好；软门禁 band=0.05，基线 430.83）：
  - 安静窗：timings ≈ [428-437] 三元组，floor ≈425-433，中位 ≈429-437；
    score≈1.00-1.03（ref 同池 ≈429-440，[380,700] 带内）。
  - 争用窗（当日实测存在）：中位可至 455-590（与 iter13/14 现象同源），判定可能
    band/worse——此为机器状态，不构成对假设的否定（判否定只看 floor 对照与 max_error）。
  - 机制信号：与 default 臂同窗口 floor 差 ≥10ms 为「机制生效」；
    差 <5ms（噪声级）为「机制不成立」。
- **录取预期（如实）**：安静窗中位有几率 <430.83（探针 v6 两轮 floor 429.5、
  late6 425.3/433.8）；但属边缘命中而非稳赢。**框架按 worse_than_elite 拒收 band 内
  读数的规则下，本实验最坏产出 = 该格平价天花板的最终收口证据。**

## 6. 执行过程与结论（占位——由 coding 填「实现与测试」，debug 填「卡点/根因/解决」）

<!-- coding：实现要点（含 TDD 红绿）+ verify/benchmark/gate_check JSON 原文读数 -->
<!-- debug：若失败——对照第 2/3 段判断假设被否定还是实现有误；争用窗读数标注 -->

### 实现与测试（coding 交付 + debug attempt2 复验补全）

- 实现：iter003 父本逐行复制，唯一改动 = 模块加载期（`import numpy` 之前）
  `os.environ.setdefault("VECLIB_MAXIMUM_THREADS", "6")`（单变量：线程池配置维度）。
- TDD：`test_solution.py` 12/12 PASS（3 个区分性测试在父本必红；setdefault 语义、
  fresh 子进程生效、list/array 契约、逐位一致、非缓存、单次派发等全绿）。
- 自检（attempt2 复跑）：`{"ref": false, "n": 8192, "runtime_ms": 442.254,
  "max_error_vs_matmul": 0.0}`。
- verify（attempt2 复跑）：`{"passed": true, "max_error": 0.0, "sizes": [64, 8192]}`。
- benchmark（attempt2，安静窗）：`{"runtime_ms": 428.051208, "score": 1.0096,
  "timings": [428.051208, 428.011125, 430.099542], "ref_runtime_ms": 432.159,
  "n": 8192, "k": 3}` —— **中位 428.05 / floor 428.01，低于精英基线 430.83**。
- benchmark（coding 期，温和争用窗）：timings [452.14, 444.52, 445.14]，ref 481.41
  （score 1.08）—— ref 481 表明当时机器已非安静态。
- gate_check（attempt2，撞争用尖峰）：`passed=false, soft_gate worse, ratio 1.055337,
  value 454.666, timings [436.603, 545.211, 454.666], ref 488.02, score 1.0734`。
- gate_check（debug attempt1，深争用窗）：worse ratio 1.1817，timings
  [523.94, 492.52, 509.12]，ref 443.71。

### 机制判定 vs 录取判定（按计划 §3 分开记录）

- **机制（假设主句）：成立**。① floor 428.01（本窗）vs default 臂 pooled floor
  441.58（thread_ab.json 3 轮交错 A/B）→ **深 13.6ms**（≥10ms 判据达成）；
  v6 pooled median 433.16 vs default 455.49 → 深 22.3ms。② max_error=0.0
  （verify 双规模 + final_confirm.json SHA 逐位相同）。③ score 1.0096、ref 432.16
  落 [380,700] 带内——旋钮对 candidate/ref 对称生效（同池同速），无任何加固通道触发。
- **录取（框架判定）：本 pass 未过**。gate_check 中位 454.67 > 430.83（worse）。
  timings 中 545.21 尖峰夹在 436.60/454.67 之间 = 争用窗伪影：`winedevice.exe`
  双进程自 10/18 常驻吃 ~270% CPU（ps 实测 168.3%+103.1%，load 3.58-3.82）；
  同候选 60 秒前安静窗中位 428.05 < 430.83。445-545ms 读数全部为机器状态，
  候选真实性能 ≈428-432（安静窗与精英平价、floor 略优）。
- **收口**：该格三维度证据链完整——iter13/14 缓冲管理维度证伪、结构维度相邻格坍缩
  平价带、iter15 线程池维度机制成立但录取受机器状态支配。「该格 = BLAS 平价天花板」
  的最终收口证据落袋。

### 卡点与解决（debug，attempt 2/3，触发：自检失败）

- 触发类型：自检失败（procedural）——诊断包 `{"passed": false, "stage": "self_check",
  "metrics": {}}` metrics 为空；实因 coding agent 在完成门禁三件与第 6 段回写前被
  turn-budget kill（agent_coding.json：exit_code 143，turn-budget-exceeded: 24）；
  debug attempt1 同因阵亡（16 turns）。
- 复现方式与关键读数：attempt2 依序复跑全部验证——test 12/12、自检 0.0、
  verify passed max_error=0.0、benchmark 428.05（安静窗，低于基线）、gate_check
  worse 1.0553（撞尖峰）。跨时间 5 组 benchmark 读数（445.14 / 428.05 / 454.67 /
  509.12 / thread_ab 三轮）确认性能读数双态分布 = 机器争用，非候选回退。
- 三个假设与验证结论（逐条）：
  - H1「self_check 失败为程序性 kill，实现无缺陷」→ **成立**：全部正确性门复验绿
    （12/12、自检 0.0、verify 0.0），诊断包 metrics 空 + exit 143 佐证。
  - H2「gate worse 为争用窗伪影（winedevice.exe ~270% CPU 常驻）」→ **成立**：
    60 秒前同候选 428.05 < 430.83；gate timings 内 545.21 尖峰；ps/load 佐证。
  - H3「旋钮在 benchmark 协议下失效（BLAS 池先于候选 exec 创建）」→ **不成立**：
    交错 A/B v6 floor 429.48 vs default 441.58（差 12.1ms）+ 本窗 floor 428.01 +
    score 1.0096（ref 同池）——旋钮生效且对称。
- 根因：self_check 失败信号 = coding turn-budget 程序性死亡（非实现缺陷）；gate
  worse = 外部 CPU 争用窗（winedevice.exe 双进程），属计划 §3/§5 预声明的不构成
  假设否定的机器状态。
- 处置：**keep_original**——候选逐行不动（无缺陷可修）；复核依据 = verify 0.0 +
  12/12 + 安静窗 428.05 < 基线 + 机制 A/B 13.6-22.3ms floor 差。按计划 §4 纪律
  （门禁每件至多一次、读数即记录、撞争用窗不复跑不追噪声）不再复跑 gate_check。
- 复验结果：`gate_check --baseline 430.8255829999998` → `passed=false,
  failure_stage=soft_gate, verdict=worse, ratio=1.055337, value=454.666,
  timings [436.603, 545.211, 454.666], ref=488.02`；争用尖峰已如实标注；
  正确性门（内嵌 verify）passed max_error=0.0。

### 复验补录（debug，attempt 3/3，触发：自检失败·末次尝试）

- 机器状态（本 attempt 前后实测）：winedevice.exe 双进程 169.3%+102.8% CPU（约 272%）、
  load 3.56-3.85——外部争用源仍常驻，本窗为「温和争用带」（无大于等于 490 的尖峰）。
- 复现（attempt3 独立复跑，读数即记录、不重跑）：
  - test_solution.py：12/12 PASS。
  - 自检（触发步骤本身复跑）：`{"ref": false, "n": 8192, "runtime_ms": 520.277,
    "max_error_vs_matmul": 0.0}`——触发所报「自检失败 metrics 空」**不复现**，
    与 H1（程序性 kill）一致。
  - gate_check `--baseline 430.8255829999998`：**passed=true**、failure_stage=null；
    correctness passed max_error=0.0（sizes [64,8192] 双 seed）；perf value 440.309
    （timings [440.309, 439.947, 442.753]、ref 443.644、score 1.0076）；soft
    verdict=**band**、ratio 1.022011、tolerance_band=true——正负 5% 容差带内通过。
- 读数谱收口（同候选全部合法读数；多态分布=机器状态，非候选回退）：安静窗 428.05
  （floor 428.01）｜温和争用带 436-455（本窗 440.3，ref 443.6）｜尖峰 490-545
  （attempt1/2 撞窗）。attempt2 gate worse（454.67，545 尖峰夹中）与 attempt3 gate
  band-pass（440.31，无尖峰）为**同候选不同窗读数**，互为佐证。
- 用户手册核对（research_brief role=debug，来源小于等于 3）：
  - **ACCELERATE(7)**（Apple 官方 man page）：VECLIB_MAXIMUM_THREADS 为官方文档化的
    「限制 Accelerate 内部多线程、避免与外部线程引擎争用」机制，并明示「the value of
    VECLIB_MAXIMUM_THREADS may be cached by the library and reused」→ 须**先于首次
    BLAS 调用**设置——本实现 setdefault 于 `import numpy` 之前，符合官方约束
    （late-set 探针证明首次 gemm 前生效）。
  - Stanford rcpedia 并行指南：`os.environ["VECLIB_MAXIMUM_THREADS"]="6"` 先于
    numpy import 为标准用法；本实现 setdefault（不覆盖外部显式配置）为其超集。
  - Apple BLAS_THREADING 文档页（developer.apple.com/documentation/accelerate/
    blas_threading；JS 渲染未能直抓，检索摘要标题一致）。
  - 结论：旋钮用法与时机**无误用**，H3 再次排除。
- 三假设（attempt3 复验结论）：H1 程序性 kill、实现无缺陷→成立（12/12 + 自检 0.0 +
  verify 0.0 + 诊断包 metrics 空 + agent_coding exit 143）；H2 gate worse=争用尖峰
  伪影→成立（本窗同候选 band 通过、安静窗 428.05 基线之下、尖峰读数全部对应
  winedevice 突发窗）；H3 旋钮误用/失效→不成立（官方文档吻合 + A/B floor 差
  13.6ms + score 约 1.0 ref 对称）。
- 根因：self_check 失败信号 = coding turn-budget 程序性死亡（exit 143，metrics 空）；
  其后 gate worse = winedevice.exe 外部争用尖峰伪影。二者均非候选缺陷。
- 处置：**keep_original**——候选逐行不动（无缺陷可修）；末次尝试以 gate_check
  passed=true（band）收口，触发阶段的自检失败不复现。
- 复验结果（attempt3，最终判定步=gate_check）：passed=true、verdict=band、
  ratio=1.022011、value=440.309、ref=443.644、score=1.0076、max_error=0.0。

### 复验收口（debug，attempt 200（超限末次），触发：容差带内复验）

- 触发类型：**容差带内复验（④）**——框架 recheck `passed=true, stage=recheck,
  verdict=band, ratio=1.008758, perf_gain_pct=-0.876`（value 434.599，timings
  [437.408, 434.599, 433.085]，ref 437.810，score 1.0074，基线 430.826）。
  问题：是真没收益，还是被测噪/热点误判掩盖了收益？
- 机器状态（本 attempt 实测）：winedevice.exe 双进程 168.3%+103.1%（约 271% CPU，
  自 10/18 常驻）、load 4.19/3.92/3.83——外部争用源仍在，本窗为温和争用带
  （与 attempt3 同型）。
- 复现方式与关键读数（每件门禁至多一次、读数即记录）：
  - verify：`passed=true, max_error=0.0, sizes=[64,8192]`（双 seed）。
  - benchmark：median **436.818**，timings [521.833, 436.263, 436.818]（首拍 521.8
    为争用尖峰，同型于已记录 490-545 尖峰），ref 435.758，score 0.9976。
  - gate_check `--baseline 430.8255829999998`：**passed=true**、failure_stage=null、
    correctness max_error=0.0；perf value **435.543**（timings [435.543, 466.658,
    435.464]，ref 444.135，score 1.0197）；soft verdict=**band**、ratio 1.010951。
  - 与框架 recheck（434.599/band/1.0088）构成同窗两次独立判读：**同候选同窗稳定
    band，434-441 读数簇不是测量内噪声，是窗口态**。
- 读数谱（同候选全部合法读数 vs 基线 430.83）：安静窗 428.05（floor 428.01，三次
  计时全部 < 430.83）｜温和争用带 434.6-440.3（本 attempt 435.5/436.8，ref
  435.8-444.1）｜尖峰 490-545｜**default 池同窗对照（thread_ab 3 轮交错）pooled
  floor 441.58 / median 455.49，v6 pooled floor 429.48 / median 433.16**。
- 用户手册核对（本 attempt 第一手复核，来源 ≤3）：**ACCELERATE(7) 官方 man page**
  （manpagez.com / keith.github.io 镜像原文）："you can use methods such as the
  kvImageDoNotTile flag or the VECLIB_MAXIMUM_THREADS environment variable to
  disable internal multithreading so that it does not contend with your threading
  engine (note: the value of VECLIB_MAXIMUM_THREADS **may be cached by the library
  and reused**…)"——旋钮为官方文档化的限线程机制、须先于首次 BLAS 调用设置；
  本实现 setdefault 于 `import numpy` 之前，符合官方约束，**无误用**（与 attempt3
  检索结论一致，H3 再次排除）。
- 三个假设与验证结论（逐条）：
  - H1「-0.876% 为常驻外部争用（winedevice 约 271% CPU）造成的窗口通胀读数，
    安静窗真值 428.05 < 430.83」→ 若成立应观察到：ps/load 显示争用源常驻；同窗
    多次独立读数簇 434-441（高于安静窗 428）；同窗 default 池臂更高（441-455）。
    验证：ps 实测 271%、load 4.19；本 attempt 435.5/436.8 + 框架 434.6 同簇；
    thread_ab default pooled floor 441.58/median 455.49。**成立**。
  - H2「候选对精英基线真无收益（安静窗中位数 ≥ 430.83，428.05 为侥幸）」→ 若成立
    应观察到：安静窗读数簇 ≥ 430.83、v6 安静窗 floor 不破线。验证：安静窗 benchmark
    三次计时 [428.05, 428.01, 430.10] 全部 < 430.83；v6 A/B floor 429.54/429.48、
    late6 floor 425.25 均破线。**不成立（严格形式被证伪）——真值为安静窗约 +0.65%
    边际正收益，属带内边际量级**。
  - H3「旋钮在 recheck 协议下未生效或被误用（池先于 env 读入/比较不对称）」→ 若
    成立应观察到：score 偏离 1.0、ref 以 default 池速（floor 441+）运行、或手册
    核对出误用。验证：本 attempt score 0.9976-1.0197；ref 435.8-444.1（6 线程池速，
    非 default 441-455 floor）；ACCELERATE(7) 原文核对用法正确。**不成立**。
- 根因：recheck 的 -0.876% 为**窗口通胀伪影**（常驻外部争用把当前窗读数抬高
  4-10ms，对照安静窗真值 428.05 实为 +0.65%）；但去掩后的收益本身仅约 0.6-0.7%，
   仍在 ±5% 容差带内且低于机器日态噪声（±3-5%）——「无稳健、跨窗可复现的带外
  收益」这一实质判定成立。真正稳健的收益维度是 v6 池对 default 池的同窗优势
  （floor 12.1ms / median 22.3ms），但录取比较对象是历史安静窗的精英绝对值
  430.83，该优势无法转化为跨窗稳定的中位数超越。
- 处置：**keep_original**——候选逐行不动（无缺陷可修：verify 0.0、12/12、
  ACCELERATE(7) 用法核对无误、score≈1.0 对称、无加固通道触发）；「修复」仅剩
  两条路均被纪律禁止：撞争用窗反复重跑追安静窗（追噪声）或任何形式的作弊。
  band 实质判定维持；-0.876% 数字本身标注为窗口伪影。本格最终收口：**机制真实、
  安静窗边际正收益（约 0.6%）、录取受常驻外部争用支配——该格 = BLAS 平价天花板**
  （iter13/14 缓冲管理证伪 + 结构维度坍缩 + iter15 线程池维度机制成立但不可稳定
  展示，三维度证据链闭合）。
- 复验结果（attempt200，最终判定步=gate_check）：`passed=true, verdict=band,
  ratio=1.010951, value=435.543, ref=444.135, score=1.0197, max_error=0.0`。
