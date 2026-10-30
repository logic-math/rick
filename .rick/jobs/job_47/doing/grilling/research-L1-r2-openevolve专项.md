# research-L1-r2 openevolve 专项简报：4-agent 调度管线适配性

元信息：阶段 L1-r2｜对象 openevolve main@9196d87（2026-10-06，/tmp/openevolve-src 代码直读）｜聚焦：①adjust 类角色存在性 ②自研 vs fork 裁决事实 ③机制借鉴清单｜时间基准 2026-10-10｜前置：loop_13 简报已覆盖架构/A-G/嵌入方式/风险面，本文不重复

## 1. adjust 类角色存在性（必答①）——结论：不存在【置信度高：代码0.4+叶子交叉】
- 全仓 grep（meta/strategy_agent/self_modif/dynamic_feature/cell_add）零命中，唯 controller.py:347 注释。无策略 agent/元优化器/动态特征空间/格增删（leaf-1，经主 agent 独立复核）。
- 唯一"元层"=PopulationStrategy 六钩子（population.py:51-68）：admit/replace_cell/archive/evict/migration_due/migrate，调用点 database.py:365/411/1280/1935/2022/2043。全部 frozen snapshot 入参+纯数据返回=**决策建议器，无副作用通道**（docstring 自述 ProgramDatabase owns all changes）。
- 承载判定：增格/删格❌（格=feature_bins^len(dims) 静态网格，database.py:441，运行期不可变）；格降采样权❌（无格 visit/UCB，权重仅程序级 fitness，database.py:1599-1638）；正交性审计❌执行（仅 admit/replace_cell 内只读判断可，审计结果只能影响单候选接纳，不能改格结构与采样权）。
- 边界补充：全仓仅有的 LLM 辅助角色=novelty judge（NOVEL/NOT_NOVEL 拒收，database.py:1134，ShinkaEvolve 移植）与 llm_feedback 评分（evaluator.py:200-228）——均纯门控/评分，不调搜索空间；manual mode queue（controller.py:198-225）是 human-in-loop 任务通道，亦非 adjust。
- 对 rick 含义：adjust agent（每 N 迭代调格）无任何现成载体——fork 须在 process_parallel.run_evolution 主循环自加 iteration_end 回调（现仅有 checkpoint_callback 一缝）。

## 2. 迭代循环解剖（必答②）【置信度高：代码0.4+叶子交叉】
- 步骤序列（spawn 并行模式）：主进程选岛(:604-633)→采样父本+灵感(:1024-1030)→建快照发 worker(:574-602)→worker 建 prompt(:200-215)→单次 LLM 调用(:211-215)→解析 diff/全文+EVOLVE-BLOCK 校验(:220-330)→评估+收 artifacts(:335-346)→主进程收 future 入库+trace+迁移(:754-866)→checkpoint/早停(:867-950)。future 完成驱动，非严格迭代序。
- agent 编排缝：**无**。无 iteration_start/end 回调、无多 agent 会话、无消息传递；唯一回调缝=checkpoint_callback。
- LLMInterface 调用形态：2 抽象方法（llm/base.py:11-23），单次补全（system+1 条 user 拼单串，claude_code.py:60-116）；retry 仅 API/超时错误级，非多步工作流。

## 3. builder/coding/debug 三角色映射可行性（必答③）【置信度高：代码0.4+叶子交叉】
- **builder❌无原生载体**：选父在 prompt 构建之前由主进程完成（_submit_iteration，process_parallel.py:1016,1029）；sampler 纯模板填充（sampler.py:51-200），prompt 无格子级元信息（无 cell ID/visit/占据度，特征坐标仅信息性 "exploring region" 串，sampler.py:239-244）；improvement_areas=规则式片段组装（sampler.py:202-260）非规划。承载仅两路：init_client 把 agent 塞进"单prompt→单响应"壳（输入已被框架定型）或 fork worker。
- **coding❌无接管缝**：worker 内置单步执行器（diff 应用/enforce_evolve_blocks/长度门禁，process_parallel.py:230-355，code_utils.py:97,193,230）；无外部 agent 注入点。
- **debug❌无修复循环**：三层失败链——①worker 解析失败→error 丢弃仅 log（:750-753）；②evaluator 异常→同基础设施重试 max_retries 次 1s 间隔（evaluator.py:163,292-293，非修复）→全败仍构造 {"error":0.0} 候选入库（:300-305，可被 admit 钩子拒收）→超时不重试（:261-285）；③反馈=artifacts 侧通道（evaluator 捕 stderr/traceback→库→snapshot→下轮父本 prompt，process_parallel.py:367,771+sampler.py:657-716）——但 error 候选排名垫底、失败 traceback 难进 prompt，对失败候选的反馈近乎失效。LLM 评委=_llm_evaluate 混分 0.7/0.3（evaluator.py:225-228），是评委非 debugger。
- 判定：三角色映射=「debug=丢弃+下轮反馈（弱版）」；builder/coding 均须 fork 或塞 init_client 壳。

## 4. fork 改造量诚实评估（必答④，若坚持 openevolve 实现 4-agent 管线）
须改清单（文件/函数/量级）：
1. **adjust 每 N 迭代**：process_parallel.py run_evolution(:640-1015) 加 iteration_end 回调（仿 _checkpoint_if_due :633，~30 行）+ adjust 本体（操作 island_feature_maps+新增格级 weight 字段，~250-400 行）+ 快照同步带格权重（:574-602，~10 行）。
2. **格级 visit/UCB**：database.py `_sample_from_island_weighted`(:1599-1638)+三比率采样(:573-587)+sample_from_island(:543) 换 UCB；格结构加 visit/success（feature_maps 平行 dict，~200-300 行）。
3. **外部 invalidation/events.jsonl 真相源**：database 内存权威、仅 init/resume load——须加失效 API+采样/入库一致性检查（~150-250 行）+**语义反转**（外部 JSON 为权威，checkpoint 是 interval 全量快照非 event log，controller.py:430-467）：行数小、风险最高。
4. **多 agent 编排层**：fork `_run_iteration_worker`(:141-408) 为 builder→coding→debug 管线 executor（~400-800 行，最大改造）；父本选择须移进 builder（改采样协议）；worker 经 ProcessPoolExecutor pickle 传递——pi runs.run 只能 worker 内 subprocess 或改 spawn 拓扑。
5. **agent 六字段合约**：自定义解析层换 EVOLVE-BLOCK/diff 解析（code_utils.py:97,193,230 硬编码于 worker，~100-200 行）。
- 总量级 **~1100-1800 行**+语义级风险；superopt 已有 UCB/visit/weight 语义须反向移植进 openevolve。同步税：上游活跃（2026-10 仍提交）单人维护，5 处 fork 点每次同步须重放；loop_13 已核须 pin main+自建回归。

## 5. 机制借鉴清单（必答⑤，完整 13 项+自由发现见 leaf-4，此为 human 呈现版）
高价值直接移植设计：
- **级联评估**：stage1/2/3 逐级+阈值[.5,.75,.9] 早筛，失败记 failure_stage 入 artifacts（evaluator.py:368-440;config.py:396-397）→映射：框架验证=stage1/env 门禁自检=stage2，debug 直读 failure_stage。**协议借鉴**（评估器协议本身：用户 py 文件 evaluate(program_path)→dict/EvaluationResult，evaluator.py:75-101，零侵入）。
- **TemplateManager 目录覆盖**：defaults+custom_dir+fragments.json（templates.py:175-227）→**协议借鉴**，rick prompt 模板体系直接抄此设计。
- **EvaluationResult artifacts 侧通道**：metrics+stderr/traceback 契约（evaluation_result.py:11-40;evaluator.py:180-196）→**协议借鉴**，debug agent 的输入契约模式。
- **IslandSelector frozen 快照协议**（selection.py:12-27）→rick agent 合约范式（只读快照入+纯数据决策出，与 PopulationStrategy 同型）。
中价值（按需选借）：
- 进程并行：ProcessPool+库快照发 worker+模块级 worker fn（process_parallel.py:464,574-602）——快照只读与单写者兼容，可借；岛模型：迁移+去重+migrant 标记（database.py:2032,2105-2118），**注意 main@9196d87 无 reseed**（loop_13 简报该表述有误，实为迁移制）；多模型 ensemble 权重采样+确定性种子（ensemble.py:53-101）；checkpoint interval 全量快照（controller.py:430-467）；diff 健壮解析器（code_utils.py:193,378）；语义去重 embedding 余弦（embedding.py;database.py:1215）；novelty judge NOVEL/NOT_NOVEL 拒收（database.py:1134）；manual mode 目录队列 human-in-loop（controller.py:198-225）；random_seed=42 采样可复现（config.py:363）。
低价值/不建议：LLM 反馈混分 0.7/0.3 **硬编码**于 combined_score 重算（evaluator.py:225-228，与 llm_feedback_weight=0.1 并存且不一致）——语义须 config 驱动再借。

## 6. 对照表：自研调度器+机制借鉴 vs fork openevolve（必答⑥，倾向性事实，不替 human 决策）
| 维度 | 自研+借鉴 | fork openevolve |
|---|---|---|
| 4-agent 管线承载 | 原生设计，agents 一等公民 | 须重写 worker 核心循环（最大改造 ~400-800 行）|
| 真相源单写者 | events.jsonl 原生 | 内存库权威，须语义反转（最高风险点）|
| UCB/visit | superopt 已有语义 | 须换掉三比率采样+建格级结构（~200-300 行）|
| adjust 角色 | 原生设计（每 N 迭代回调自建即成）| 无任何载体，须自建回调+格结构（~280-430 行）|
| 热更新 env | 自主控制 | 评估器协议可载门禁，但进程池/spawn 语义须验证 |
| pi 池集成 | 原生 runs.run | init_client 可注入，pickle/spawn 约束+子进程拓扑须绕 |
| 维护成本 | 全自担但语义自控 | 同步税：5 fork 点重放+上游单人 bus factor |
- 倾向性事实：fork 须改的恰是 openevolve 没有的（4-agent/UCB/adjust/真相源=rick 核心差异化），openevolve 的价值（级联/模板/checkpoint/并行/岛）恰是协议级可借鉴、无须扛整个框架。两向成本同源但方向相反：fork=换心脏，自研+借鉴=装器官。第三方实证（loop_13 leaf-community）：Berkeley/AMD fork 皆做的是替换 LLM 层/负载均衡，**未见**做 agent 管线化改造的先例。

## R7 上报项
1. fork 行数估算为代码推演量级（±50%），未含测试/回归；fork 前建议做 72h 空载压测+resume 回归（loop_13 leaf-risk 已建议）。
2. init_client 塞 agent 壳路线（不 fork 承载 builder）为推演，未实测；pi runs.run 跨 ProcessPoolExecutor 的 pickle 存活性未验证。
3. 热更新 env 在 openevolve 进程池下的语义（worker 重启/评估器重载）未实测。
4. 上游 2026-10-06 后新提交未扫（对象为该版 clone）；reseed 表述已在本文修正，loop_13 其余表述未发现矛盾。

## 叶子文件路径
- research-L1-r2-openevolve专项-leaf-1.md（adjust 角色/钩子面，主 agent 独立复核）
- research-L1-r2-openevolve专项-leaf-2.md（迭代循环解剖，主 agent 独立复核）
- research-L1-r2-openevolve专项-leaf-3.md（三角色映射，关键声明已复核：evaluator.py:163/300-305）
- research-L1-r2-openevolve专项-leaf-4.md（机制借鉴清单 13 项+自由发现，已抽检 reseed/novelty/ensemble）
