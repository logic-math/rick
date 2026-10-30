# research-L1a：superopt v5 资产盘点（代码考古尽调）

> 阶段：L1a 事实调研 | 对象：/workdir/sunquan20/INFER_RSI/superopt（主包 15 文件 ~3300 行）| 基准：仓库现状 2025-08 快照 + v5 workdir 运行时数据 | 信源：代码直读为主（置信 0.9+），运行时 DB/事件流佐证（置信 0.85）

## §1 模块地图（superopt/superopt/，文件→职责→关键行号）

| 文件 | 行数 | 职责 | 关键位置 |
|---|---|---|---|
| evolution.py | 766 | 进化主循环：选点→父选→灵感→prompt→agent→验证门→入档 | EvolutionLoop:85；run:125(默认30迭代,可挂strategy_interval)；_step:146；_run_and_record:498；_force_validate:395；_retry_once:458；_record_failure:475；_try_run_benchmark:311；_append_work_log:726 |
| db.py | 341 | SQLite 三表 + 全部写入路径 | SCHEMA:8（solutions:10-24/elite_map:26-33/feature_cells:35-39）；MIGRATIONS:49；insert_solution:107；update_visit:201；update_weight:213；update_strategy_desc:234；get_feature_space_state:246；get_elite:268；get_all_elites:295；upsert_feature_cell:309 |
| feature_space.py | 185 | 动态特征格持有者 + UCB 采样 | FeatureSpace:12；_load_from_db:20；add_cell:66；sample_target_cell:95-162（UCB 公式）；find_nearest_elite:165（对称差距离） |
| strategy.py | 258 | StrategyLoop 元优化器：读精英池→策略 agent→应用权重/新格 | StrategyLoop:27；_step:86；权重应用:164-180；冻结保护:165-171；_read_perf_log_tail:240；死常量 HUMAN_ADVICE_FILENAME:24（全仓无消费者） |
| agent.py | 176 | worktree 隔离 + LLM CLI 调用 | agent_worktree:20（codes/{uuid}）；run_agent:68（subprocess）；read_agent_result:129；write_agent_result:150；read_agent_work_log:162 |
| check.py | 119 | env 结构验证门（启动前） | check_env:59；_scan_skill_types:25；必需:code/+skills/+domain_knowledge.md+env_mode.txt+SOP两节 |
| schema.py | 58 | agent_result.json 六字段合约（唯一定义点） | F_*:10-15；REQUIRED_FIELDS:19；make_agent_result:22；validate_agent_result:34 |
| prompt_builder.py | 177 | 从 prompts/ 模板组装 prompt | build_coding_prompt:41；build_init_prompt:101；build_strategy_prompt:128；_build_skills_section:28（只列 .sh）；_substitute:164 |
| stats.py | 139 | events.jsonl 分析报表 | run_stats:33（收敛曲线/耗时P50P95/retry分布/cell详情） |
| recorder.py | 43 | 事件流+性能日志落盘 | _emit_event:17（events.jsonl append）；_append_perf_log:27；_build_perf_log_entry:34 |
| validate_env.py | 110 | env 端到端验证（4步级联） | validate_env:11（结构→正确性→benchmark→结果有效性）；print_report:97 |
| init.py | 87 | 首次运行初始化特征空间 | _DEFAULT_CELLS:14（3 个兜底格）；initialize:52（策略 agent 1 步+失败兜底） |
| __main__.py | 512 | CLI 入口/子命令 | main:221；子命令 stats:41/init:47(→_create_env_template:160)/validate:264；_setup_workdir:13；_get_db_path:37；_validate_baseline:501（调用:435）；进化默认流:373-499；mode 门:419-428 |
| skills/base.py | 117 | 可执行验证器机制 | SkillType:13（CORRECTNESS/BENCHMARK/AUXILIARY）；SkillResult:34；_scan_skills:53；Skill:81；run_logged:102 |
| config.py | 104 | FeatureCell 定义 + grid_config 读写 | FeatureCell:7（frozenset）；GRID_CONFIG_FILENAME:49；save/load/init/get_active_config:68-104 |
| prompts/ | 4 模板 | optimize/debug × coding/strategy | coding:环境约束+必须实现目标方向+SOP 步骤+agent_result 自评（框架覆盖）；strategy:三文件输出+主动查代码+profile_log 强制读 |

## §2 RFC 点名缺陷的代码定位（逐条核实）

### 2.1 update_visit 零调用（visit 断链）— 置信 0.95（代码直读+全仓 grep）
- 定义：db.py:201-211 `update_visit(feature_id, delta=1)`，SQL `UPDATE elite_map SET visit = visit + ?`，功能完好。
- 调用方：**superopt/ 生产代码零调用**。全仓 grep 仅命中 `.rick/jobs/job_8/doing/tests/task*.py`（历史 job 测试）与 `.rick/skills/check_db_schema.py`（job 验证脚本）。evolution.py 的 _step/get_elite 路径、strategy.py 的 _step 路径均不触碰 visit。
- 运行时佐证：v5 workdir DB 中 elite_map 全部 77 格 visit=0（DEFAULT 0 从未递增）。
- 断链形态：方法存在、语义正确、无人调用——「公式在代码里≠机制在运行」的典型。

### 2.2 UCB 公式与探索项退化 — 置信 0.95
- 公式位置：feature_space.py:95-162 `sample_target_cell`。注释宣称 `UCB(cell_i) = softmax(fitness)[i] + c*sqrt(ln(N)/visit[i])`（:97）。
- 实际实现：exploit = `math.exp(fit - fitness_max)`（:152，softmax 式利用项）；explore = `c*sqrt(log(N)/visit)`（:153）。
- 退化链：visit 恒 0 → :148 `visit = r.get("visit", 1) or 1` 把 0 归一为 1 → N = sum(visits) = 已填格数（每格贡献 1）→ explore = c·sqrt(ln(已填格数)/1) = **对所有已填格是同一常数**，不改变格间排序 → 采样实为「纯 exploit 排序 + 空格填充 + 权重接管」。
- 空格近似：:159 `inf_ucb = 1.0 + c*sqrt(log(N+1))`（visit=0 → +inf 的近似）。
- fitness 计算：:130-139 perf_desc=True→fit=pv；False→fit=1/pv；pv 缺失或≤0→0.0。

### 2.3 StrategyAgent 权重接管采样 — 置信 0.95（代码+v5 DB 双证）
- 接管点：feature_space.py:135-141——`weight>0 → filled_weights.append(w)`（**直接用权重替代 UCB 分数**）；`weight==0 → 0.0`（冻结，永不采样）；无权重才走 UCB。
- 权重写入路径：strategy agent 落盘 workdir/weight_updates.json → strategy.py:164-180 逐条 `db.update_weight(feature_id, weight)` → elite_map.weight（空格回退 feature_cells.weight，db.py:213-232）。
- 冻结保护：strategy.py:165-171 已填格跳过 weight=0（防精英被冻死）——但 0<weight<1 压制无保护。
- 运行时分布（v5 DB，77 格）：0.1×24、0.2×19、0.3×11、0.4×2、0.5×7、0.8×3、1.0×5、1.5×4、2.0×1、2.5×1。与 RFC A4「24 格压 0.1、5 格 1.0、最高 2.5」**逐数吻合**。
- prompt 依据：optimize_strategy_agent_prompt.md:226-229 明示权重范围建议 0.1~3.0、默认 1.0、>1 提升/<1 收敛压制——压制行为是 prompt 授权的。

### 2.4 人工修剪（cur=bak 删事件+DB 改写）— 置信 0.9（运行时证据+代码无此能力）
- 证据：v5 workdir 同时存在 events.jsonl（197 行，尾事件 ts=1779923394）与 events.jsonl.bak（207 行，尾事件 ts=1780366979，**比 cur 新 10 个事件**）→ 操作形态=备份后删尾 10 事件重写 cur，DB 同期被改写（精英/权重与事件流不再对齐）。
- 代码侧：superopt 全部代码**无任何修剪/回滚/事件删除能力**（recorder.py 只有 append）→ 该操作是人工 shell 干预，零操作记录、零审计事件。
- 连带损失：15.78ms 真突破（elite_updated ts=1780360119, cell=lm_head_fp8_triton_only/o_proj_w8a16_triton）**只存在于 .bak**，被修剪一并回滚出 cur——与 RFC A6「连真突破一并回滚」吻合。

## §3 可继承语义细节

### 3.1 worktree 隔离 — 置信 0.95
- agent.py:20-64 `agent_worktree(workdir, initial_code_dir)`：创建 `{workdir}/codes/{uuid}/`（**普通目录拷贝，非 git worktree**）；父代码整目录复制（剔除 .git/）；**目录不删除**（持久 code_path 供回放）；每次快照在 workdir git 提交 `superopt: {uuid8}`（agent.py:49-63，无 git 静默失败）。
- v5 实况：codes/ 下 124 个 uuid 目录；git log 全为快照提交。

### 3.2 重试门禁（attempt/period 机制）— 置信 0.95
- 主循环：evolution.py:498-720 `_run_and_record`，`for attempt in range(max_retries+1)`（CLI 默认 5+1 次机会）。
- 验证级联：`_force_validate`（evolution.py:395-444）：correctness → benchmark → auxiliary（benchmark 通过后跑，结果只记日志不进精英池决策）。
- attempt 计数：非显式传递——由 worktree 内已有 `skill_correctness.attempt_{n}.log` 文件数推导+1（:401-403、:414-416）。
- 重试注入：`_build_retry_error`（:446，error+profile_log.md 全历史）→ `_retry_once`（:458）在**同一 worktree** 重跑 agent（parent_code_dir=wt_path，agent 看得到自己上次的产出）。
- 超时兜底：`_try_run_benchmark`（:311）agent 超时但有代码 → 直接 bash 跑 benchmark .sh shim（解析 SCORE=/RUNTIME_MS= 输出）补 agent_result.json。
- 四结局：A 改进入档 / B 无改进静默丢弃（只写日志+事件）/ C 正确性失败重试 / D 重试耗尽记失败事件。

### 3.3 演替规则 — 置信 0.7（代码无「演替」字样）
- 全仓 grep 演替/succession 零命中。语义上最接近的是**同格精英替换条件**（见 3.5）；RFC 所指「演替规则」需 human 澄清确切定义（R7-1）。

### 3.4 跨格重组/组合库 — 置信 0.9
- 灵感采样：evolution.py:210-239——从已填格（排除目标格）过滤 score>0，按 score 加权**无放回采 2 个**作 inspiration_dirs 注入 prompt；全 0 分退化为均匀。
- 空格父选：evolution.py:197-207 + feature_space.py:165-177——最近精英=对称差距离 `len(A.features ^ B.features)` 最小，并列取 perf_value 最优。
- 「82 格组合库」=**运行时状态非代码常量**：v5 DB 现 77 elite 格/186 策略格，events 全史 89 个不同 cell_id；15.78ms 突破时刻的组合库约 82 格（RFC 口径）。代码形态即上述加权灵感采样+近邻父选。
- 15.78ms 事件实证：elite_updated ts=1780360119，cell=lm_head_fp8_triton_only/o_proj_w8a16_triton（跨格特征组合格名自证重组产物）。

### 3.5 MAP-Elites 入档规则（同格替换条件）— 置信 0.95
- db.py:151-178 insert_solution 内：空格→直接入档；双 perf_value→按 perf_desc 方向比较（True 大者胜/False 小者胜）；仅新有值→更新；均无值→score 高者胜。胜出时旧精英 is_best=0、新者 is_best=1、elite_map 指针换新（UPSERT ON CONFLICT）。
- 框架级守门：evolution.py Case A 仅当 perf 优于当前精英才调 insert_solution；无改进 Case B 根本不入档。
- lineage 断链：insert 调用恒 parent_hash=None、generation=0（evolution.py:646-647）——**血缘不入库**，只能从 events/code_path 考古。

### 3.6 agent 合约（六字段确认）— 置信 0.95
- schema.py：共 **6 字段**：valid(bool)/score(num)/perf_value(num|None)/perf_type(str)/perf_desc(bool)/summary(str)。必填 5（REQUIRED_FIELDS:19），summary 缺省补 ""（make_agent_result:22/write_agent_result 补齐）。perf_value 键必须在但值可 None。perf_desc=False 低优/True 高优。
- 双写者：agent 自评写一份（prompt 明示会被覆盖）；框架实测后**覆盖写**为权威值（evolution.py:617-624）。

## §4 example 任务域对比（example/，三者同构 env 目录规范）— 置信 0.9

| 维度 | demo_env | debug_env | prefill_infer_env |
|---|---|---|---|
| 任务 | matvec（N=600 numpy）145ms→目标 1ms | stats.py 修 bug：compute_variance 整除 `//` 应为 `/`（code/stats.py:23 自带 BUG 注释） | Qwen3 prefill 推理延迟（v5 生产域） |
| code/ | solution.py（唯一入口 matvec） | stats.py（3 函数含 1 故意 bug） | prefill_infer/ 完整包（engine/models/qwen3/layers/attention/loader）+_bench_inner.py/_infer_inner.py/setup.py/requirements.txt |
| data/ | bench.py+test_correctness.py | 同左 | 同左（bench timeout 1800s vs demo 300s） |
| skills/ | correctness_skill.py+benchmark_skill.py（**无 .sh shim**） | +2 个 .sh shim | +2 .sh shim+profile_skill.py（AUXILIARY） |
| env_mode.txt | optimize | optimize+debug | optimize |
| 评分 | score=clamp((145-ms)/(145-1),0,1)；正确性 max_abs_diff<1e-6 | pass_rate（测试通过率） | runtime_ms + 远端服务 |
| 依赖 | numpy，纯 CPU，秒级 | 纯 Python，CPU，秒级 | torch==2.6.0/triton/**GPU+远端推理服务**（HTTP/RPC 端口 30000/30001，v5 事件显示 batch_size 13-15 接流） |
| 时长量级 | bench 单次秒级；e2e(mock) 秒级 | 秒级 | v5 实测单迭代 agent_ms≈2.0e6-3.5e6ms、total_ms≈6.5e6ms（**约 1.8-2.5 小时/迭代**，事件流实证） |

- 接口约定（三者统一）：skills 以 subprocess 跑 data/ 脚本，cwd=work_dir、PYTHONPATH=work_dir，解析 **stdout 最后一行 JSON**；bench 输出 {"runtime_ms","score"}，correctness 输出 {"passed","max_error"}。
- **demo_env CPU 快速跑通：是**——tests/e2e_check.py 官方定位即「self-contained demo_env 回归，纯 CPU+numpy 无 GPU/LLM，mock CLI 全程秒级」。
- 注意点：demo_env 无 .sh shim，而 prompt_builder._build_skills_section（prompt_builder.py:28-40）只列 .sh 文件→demo_env 的 coding prompt skills 段恒为 "(no skills found)"（.py/.sh 双轨不一致，实际靠 domain_knowledge.md 补偿）。

## §5 测试现状 — 置信 0.9

- superopt/tests/（6 文件，**非 pytest**，直跑 python3 脚本）：
  - e2e_test.py：标准 E2E 验收（check_env 过→策略 agent 初始化格子→codes/≥1→elite_map≥1→精英目录含新代码）。`python3 tests/e2e_test.py [--cli tests/mock_cli.py --iterations 1]`，亦有 `rick tools e2e_test` 入口。
  - e2e_check.py：**标准回归工具**（demo_env 全管线+mock CLI 秒级；换 --cli claude 跑真 agent）。任何 superopt 改动后必跑。
  - e2e_debug_test.py：debug 模式变体（配 mock_cli_debug.py）。
  - mock_cli.py/mock_cli_demo.py/mock_cli_debug.py：模拟 claude CLI（按 prompt 内容写 strategy_updates.json / fused_attention.cu+agent_result.json）。
  - test_validate_env.py：unittest。
- example/prefill_infer_env/tests/test_job19_e2e.py：job_19 专项（profile 目录结构/版本管理、check_env SOP 校验、prompt_builder parent_profile_path、coding/strategy prompt 内容、ProfileSkill 集成），unittest。
- 历史作业测试（非产品资产）：.rick/jobs/job_8/doing/tests/task1-6.py（DB/UCB/is_best/code_path/add_cell 验证）、.rick/skills/check_db_schema.py 等。

## §6 运行时数据 — 置信 0.9

- workdir 布局（--workdir 指定，_setup_workdir 建 .superopt/+git init+空提交）：`.superopt/db.sqlite`（唯一 DB）；`codes/{uuid}/`×124（v5）；`events.jsonl`（事件流，recorder append-only）；`events.jsonl.bak`（人工修剪残留）；`agent_work_log.md`（153KB）；`perf_log.md`；`opt_log.md`（策略 agent 写）；`baseline_perf.json`（_validate_baseline 产物）；`grid_config.json`（策略 agent 可改）；`weight_updates.json`/`strategy_updates.json`（瞬态，应用后即删）；`monitor_log.md`（**superopt 代码不写，外部人工监控产物**）。
- v5 workdir：/workdir/sunquan20/SLM/prefill_only_infer_server_opt_env_workdir_v5/（77 elite/186 策略格/86 solutions/124 codes/80 次 elite_updated）。
- 其余 workdir（/opt/meituan/dolphinfs_sunquan20/ai_coding/SLM/models/ 下）：prefill_infer_opt 59e/85c、torch_compile_opt 72e/77c、ctr_deepfm 35e/57c、dense_arch_opt 20e/34c、mixer/evolution 11e/15c、ctr_deepfm_debug 2e/10c、kernels 35e。**「77 格」恰为 v5 elite_map 计数**。
- **JSON 导出：无内置能力**。--status 仅文本打印精英池；stats 只读 events.jsonl；精英/组合库无 JSON 序列化器（grid_config.json 仅存 2 特征格的约束元组）。导出只能 sqlite3 手工查询（SQL→JSON 自行拼）。

## §7 配置面 — 置信 0.95

- **无全局配置文件**：config.py 只有 FeatureCell+grid_config 工具；一切运行参数走 CLI flag（__main__.py:281-355）：--workdir（进化必填）/--env（必填）/--iterations(30)/--cli($AI_CLI 或 claude)/--timeout(3600)/--status/--strategy/--strategy-interval(0=关)/--max-retries(5)/--mode(optimize|debug)/--skip-baseline-check/--stats。
- 超时双默认：__main__ 3600 vs EvolutionLoop/StrategyLoop 构造默认 1800（实际以 CLI 传入为准）。
- env 定义=目录约定：code/（参考实现，每迭代整拷）+skills/（.py 必含 CORRECTNESS/BENCHMARK 各一）+data/（bench/正确性脚本，约定非强制）+domain_knowledge.md（必需：≥200 字、含技术关键词、含「## 性能分析 SOP」「## 性能调优 SOP」两节，strategy.py:52-76+check.py:88-96 双重校验）+env_mode.txt（模式过滤器）。
- 模式守门：__main__.py:419-428 读 env_mode.txt，--mode 必须在其声明集合内。
- 文件名常量散布：agent_result.json(schema)/work_log.md/profile_log.md(agent)/opt_log.md+weight_updates.json+strategy_updates.json+human_advice.md(strategy，human_advice **定义未用**)/perf_log.md+events.jsonl(recorder)/baseline_perf.json(__main__)/grid_config.json(config)。

## §8 对重写的建议清单（事实底座上的继承判定）

**可继承（语义级移植，非文件级）**：
1. agent_result 六字段合约（schema.py 全套语义含覆盖写权威值机制）——直接照搬语义。
2. 验证级联+四结局模型（correctness→benchmark→auxiliary；A/B/C/D；_force_validate 语义）。
3. 重试门禁骨架：attempt 由落盘文件计数推导、同 worktree 重试（agent 见己作）、错误+profile_log 历史注入、超时补跑 benchmark 兜底。
4. 隔离 codes/{uuid} 持久快照 + code_path 入库 + git 快照提交（可平移为 rick worktree/git 原语）。
5. env 目录规范（code/skills/data/domain_knowledge.md/env_mode.txt + skill_type 角色制 + SOP 双节校验）。
6. MAP-Elites 同格替换条件（perf_desc 方向比较+is_best 维护）与空格近邻父选/加权灵感采样（跨格重组的 v5 实证有效形态）。
7. demo_env CPU 秒级 e2e + mock CLI 测试范式（rick super 首跑 e2e 的现成模板）。

**须重写（结构缺陷已实证）**：
1. visit/采样：UCB 名实分离（§2.1/2.2）→ 按识别型目标重做分配+visit 真实递增（RFC §5.2）。
2. 权重机制：StrategyAgent 权重直接替代 UCB 分数（寡头化实证 24/77 格压 0.1）→ 提案-裁决分离+软硬淘汰。
3. 真相源：SQLite 三表+双写（DB 与 events 可分叉，人工修剪实证可旁路）→ events.jsonl 单写者 append-only+干预即事件。
4. lineage：parent_hash 恒 None/generation 恒 0（血缘断链）→ 事件流重建 lineage。

**明确弃用**：
1. update_visit 死方法与其调用幻觉（保留语义修复，弃用「存在即生效」的实现方式）。
2. HUMAN_ADVICE_FILENAME 死常量。
3. _build_skills_section 只认 .sh 的双轨不一致（demo_env prompt 失效实证）。
4. 人工 shell 修剪路径（无审计旁路）；monitor_log.md 外部黑箱监控。
5. 三表 SQLite 作为真相源（降级为派生缓存）。

## R7 上报项
1. 「演替规则」在 v5 代码无对应物（§3.3），RFC 所指需 human 澄清（疑=同格替换+灵感注入的组合语义）。
2. 「82 格」精确口径：v5 现库 77 elite/186 策略格/事件史 89 cell_id，82 应为 15.78ms 突破时刻的策略格数，无精确快照可证。
3. 15.78/12.96ms 无独立复测数据（仅 .bak 事件流），RFC P2 基线重测仍待做。
4. v5 修剪的 DB 改写细节（哪些行被删/改）不可从产物反推（无操作记录）。
5. prefill_infer_env 远端服务的重建成本未调研（端口 30000/30001 服务端实现位置）。

> 叶子文件：无（本轮全量自研读，未外包）。信源：代码直读 0.4×多文件+运行时 DB/事件流 0.3+wiki/RFC 交叉 0.2。
