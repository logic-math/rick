# leaf-2 迭代循环解剖（openevolve main@9196d87）

## 步骤序列（spawn 并行模式，每迭代）
1. 主进程选岛：默认轮询/IslandSelector 策略注入（process_parallel.py:604-633,676-697）
2. 主进程采样父本+灵感：database.sample_from_island（process_parallel.py:1024-1030）
3. 主进程建库快照传 worker：全程序+限量 artifacts（process_parallel.py:574-602）
4. worker 建 prompt：build_prompt 含父/top/灵感/artifacts/特征维（process_parallel.py:200-215）
5. worker 单次 LLM 调用：generate_with_context(system+1条user)（process_parallel.py:211-215）
6. worker 解析 diff/全重写+EVOLVE-BLOCK 校验+长度门（process_parallel.py:220-330）
7. worker 评估子程序+收集 artifacts（process_parallel.py:335-346）
8. 主进程收 future：入库 add(target_island 修#391)/存artifacts/trace/岛代际+迁移（process_parallel.py:754-866）
9. 主进程 checkpoint（按 interval）+目标分/早停检查（process_parallel.py:867-950）
注：future 完成驱动循环，异步非严格迭代序（process_parallel.py:698-715）

## 缝清单（可注入点）
- checkpoint_callback：唯一回调缝（controller.py:505→process_parallel.py:645,867,939）
- IslandSelector：选岛策略（process_parallel.py:604）
- population_strategy：准入/驱逐（database.add 内，leaf-1 详）
- 无 iteration_start/end 回调、无 agent 编排层（全仓 grep 仅 checkpoint_callback）

## LLM 调用形态
- LLMInterface 仅 2 抽象方法（llm/base.py:11-23）
- 单次补全无会话：--no-session-persistence、多 user 消息拼单串（claude_code.py:60-116）
- retry 仅 API/超时错误重试，非多步 agent 工作流（claude_code.py:87-115）
- 无多 agent 会话/消息传递机制（全仓 grep 无命中）

## 结论3条
1. 主循环=无状态单发流水线，每迭代 LLM 一次补全，零 agent 编排。
2. 唯一迭代回调是 checkpoint，adjust 每 N 迭代无现成缝。
3. 编排注入点硬编码在 worker 三步，4-agent 管线须 fork _run_iteration_worker。

置信度高（代码直读）：controller.py:232-404,505；process_parallel.py:141-409,640-1051；llm/base.py:11-23；claude_code.py:60-116
