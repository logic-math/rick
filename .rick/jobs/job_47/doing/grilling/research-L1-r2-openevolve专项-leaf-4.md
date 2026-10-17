# leaf-4 机制借鉴清单（openevolve main@9196d87 代码直读）

| # 机制 | 位置 | 一句话描述 | rick 价值 |
|---|---|---|---|
| 1 级联评估 | evaluator.py:368-440; config.py:396 | evaluate_stage1/2/3 逐级+阈值[.5,.75,.9]早停；失败记 failure_stage 入 artifacts | 高：门禁分层映射（框架验证=stage1/env 自检=stage2），debug 直读 failure_stage |
| 2 评估器协议 | evaluator.py:75-101 | 用户 py 文件 importlib 加载 evaluate(program_path)，返 dict 或 EvaluationResult | 高：env 门禁脚本即 stage1，契约零侵入 |
| 3 岛模型 | database.py:1599,2105; config.py:359 | fitness 加权岛内采样+50 代迁移 10% | 中：与 UCB 格采样并存须改造 |
| 4 多模型 ensemble | llm/ensemble.py:53-101 | 权重归一+random_state 采样模型 | 中：pi 池借鉴权重采样，调度换 UCB |
| 5 prompt 模板体系 | templates.py:175-238; prompts/defaults/(15 文件) | TemplateManager 级联覆盖：defaults+custom 目录+fragments.json | 高：目录覆盖协议直接移植 |
| 6 checkpoint/resume | controller.py:430-495 | 每 N 存 database+best+info json 至 checkpoint_N/ | 中：events.jsonl 已覆盖，best 快照可借 |
| 7 进程并行 | process_parallel.py:464,574,141 | ProcessPool+数据库快照发 worker（artifacts 限 100）+模块级 worker fn | 中：快照只读与单写者兼容 |
| 8 LLM 反馈混分 | evaluator.py:221; config.py:406 | combined_score 在则硬编码 0.7/0.3；否则 weight=0.1 逐指标 | 低：硬编码不一致，须 config 驱动 |
| 9 novelty judge | novelty_judge.py:13-43; database.py:1202 | 嵌入余弦预筛(.99)+LLM 判 NOVEL/NOT_NOVEL 拒重 | 中：全仓唯一第二 LLM 角色，仅拒绝非调格 |
| 10 manual mode 队列 | controller.py:198-225 | 目录式人工任务队列，human-in-loop | 中：门禁需 human 确认时可借 |
| 11 evolution trace | evolution_trace.py:63-133 | 每迭代 jsonl 记 prompt/response/code | 中：同 events 思路但非单写者 |
| 12 diff 进化 | code_utils.py:193,378 | extract_diffs/apply_diff_blocks 健壮解析+回退全文 | 中：builder 差异解析器可移植 |
| 13 语义去重 | embedding.py; database.py:1215 | OpenAI/Azure/Gemini 嵌入+余弦判重 | 中：精英重评估前近重复检测 |

自由发现：selection.py:12-27 IslandSelector frozen dataclass 只读快照协议（rick agent 合约范式，高）；config.py:336 三比率 elite .1/explore .2/exploit .7；config.py:363 random_seed=42 采样可复现；evaluation_result.py:11-40 artifacts 侧通道（metrics+stderr/traceback，debug 输入契约模式，高）；process_parallel.py:604 island_pending 负载均衡（#391 修复实证）。

结论：
1. 最值得移植三件套：级联评估契约、TemplateManager 目录覆盖、EvaluationResult artifacts 侧通道。
2. 混分(8)/novelty judge(9) 是全仓仅有的评估外 LLM 介入，均无调格能力。
3. 岛模型/ensemble 与 UCB 格采样语义重叠，移植须二选一防双调度。
