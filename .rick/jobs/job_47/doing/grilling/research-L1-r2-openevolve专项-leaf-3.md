# leaf-3 三角色映射可行性（builder/coding/debug）｜openevolve main@9196d87 代码直读

## builder 承载判定：纯模板填充器，非 agent
- PromptSampler=TemplateManager 填充：读父本代码+metrics、岛内 top+diverse 程序、inspirations、feature_dimensions 名单与特征坐标串（sampler.py:159,239-244 仅信息性 "exploring region"）、父本 artifacts、最近尝试 parent_metrics 对比。**无格子级元信息**（无 cell ID/visit/占据度进 prompt）。
- **选父在 prompt 构建之前**：主进程 `_submit_iteration`（process_parallel.py:1016,1029）已定父本+inspirations，sampler 只接收既定结果。
- 注入缝=模板级：set_templates/template_key/custom dir（sampler.py:37-48）、`**kwargs` 占位符（sampler.py:181）、每岛 system_message。LLM 调用单轮 `generate_with_context(system,[user])`（process_parallel.py:204-208）。
- 承载独立多步 builder agent 仅两路：(a) init_client 自定义 LLM 客户端把 agent 塞进"单 prompt→单响应"壳（输入已被框架定型）；(b) fork `_run_iteration_worker`。"读格元信息→选父→实验计划"三步均无原生缝。

## debug 失败链事实：无修复循环（三层）
1. worker 解析失败（LLM错/无diff/SEARCH不匹配/无变化/超长/越出EVOLVE-BLOCK）→`SerializableResult(error=)`→主进程仅 log 丢弃（process_parallel.py:750-753），traceback 不入库。
2. evaluator 异常→**同代码基础设施重试** max_retries 次（1s 间隔，非修复，evaluator.py:161-283）；全败→`{"error":0.0}` 仍构造候选入库（evaluator.py:300-305），可被 admit 钩子拒收（process_parallel.py:763-765 `_CandidateRejected`）；超时→直接 error:0.0 不重试（evaluator.py:270-285）。
3. 反馈进下轮：artifacts 通道（父本 artifacts→prompt `_render_artifacts`）+规则式 improvement_areas；error:0.0 候选排名垫底、其失败 traceback 难进 prompt——对失败候选的反馈近乎失效。LLM 反馈=_llm_evaluate 质量评委（judge）混分 0.7/0.3（evaluator.py:564,222-226），是评委非 debugger。

## coding 承载
worker 内置单步执行器（diff 应用/enforce_evolve_blocks/长度门禁，process_parallel.py:230-355），无外部 agent 接管缝（除 fork worker）。

## 结论（各≤40字）
1. builder 无原生载体：选父在上游+单轮调用+无格元信息，仅 init_client 壳或 fork 可承载。
2. 无 debug 修复循环：失败即丢或 0 分入库，反馈通道对失败候选基本失效。
3. coding 角色须 fork worker；artifacts 通道（evaluator 捕获→库→snapshot→prompt）是唯一现成反馈缝。

## 证据（文件:行号）
- prompt/sampler.py:51-200（build_prompt 模板填充）、202-260（规则式改进点）、262-470（进化历史格式化）、657-716（artifacts 渲染+截断）、717-746（安全过滤）
- process_parallel.py:140-200（快照重建+既定父本入 prompt）、204-208（单轮 LLM）、230-355（diff/enforce/长度）、367/771（artifacts 存取）、750-775（error 丢弃 vs add）、590-608（snapshot 限 max_snapshot_artifacts）
- evaluator.py:71/161-283（重试循环）/222-226（0.7/0.3 混分）/266-285（异常/超时捕获 traceback 入 artifacts）/300-305（error:0.0）/327-337（pending_artifacts）/564-603（LLM 评委）
- llm/base.py:9-18（LLMInterface 仅 2 方法=单次补全形态）
