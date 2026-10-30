# task32：env 模板 research.py（检索三要素：方法/信源/关注点）

# 任务目标
第四类 env 资产：**检索 skill**——把「搜索方法 + 信源 + 关注点」三元组按 agent 角色分别固化，让 agent 复用前人认知、缩小探索空间、突破语义空间局部最优（human 洞察：纯组内变异会以小角度反复微调，出不来）。

**设计定位**：`research.py` **不执行检索**（检索由 agent 的 web_search 完成），它产出「检索指引」——约束 agent「怎么搜/去哪搜/关注什么」。

# 关键结果
- 新建 `super/env_templates/matmul/skills/research.py`（stdlib-only）：
  - CLI：`--role <adjust|builder|coding|debug>` [--cell <cell_id>] [--env-dir <dir>]
  - stdout 单行 JSON：
    ```json
    {"role":"builder","cell":"...","method":"...","sources":["arxiv.org",...],
     "focus":"该格/该角色应关注什么（与特征格绑定）","suggested_queries":["...","..."],
     "discipline":"≤3 篇；必须落到可实现动作；记录 arxiv_id/title/insight/applicability"}
    ```
  - **四角色默认三元组**（matmul 域，写进脚本内 `ROLE_PROFILES`，可用 env.json `research.<role>` 覆盖）：
    - `coding`/`debug`：method=产品文档与代码库检索（官方文档站内搜索 + GitHub 代码搜索）；sources=["numpy 官方文档","BLAS/Accelerate 文档","GitHub 优秀实现","Stack Overflow"]；focus=「该格实现细节：API 用法与限制（np.dot out= 的行切片约束/dtype 转换成本/已知坑）」
    - `builder`：method=论文检索（arXiv 站内/Google Scholar）；sources=["arxiv.org","paperswithcode.com"]；focus=「实验细节与数据：benchmark 配置/规模影响/硬件差异对比/可复现方法」
    - `adjust`：method=论文检索（arXiv/OpenReview）；sources=["arxiv.org","openreview.net"]；focus=「创新点与研究趋势：尚未覆盖的方法族/近期新方向（用于 expand_cell 判断）」
  - `suggested_queries`：按 role+cell descriptor 组合关键词生成（如 builder：`"<impl_strategy> <dispatch_target> GEMM performance benchmark"`）
  - 缺 role / 未知 role → 输出合法 JSON 带 `error` 字段，退出码 0（不阻断 agent）
- `super/env_templates/matmul/README.md`：五件 skill 说明（profile/benchmark/verify/gate_check/research）+ 三元组语义
- `domain_knowledge.md`：增「检索要点」段（本域哪些信源最有效、哪些关键词踩坑）

# 写域
- super/env_templates/matmul/skills/research.py（新建）
- super/env_templates/matmul/README.md
- super/env_templates/matmul/domain_knowledge.md

# 依赖关系
- 无（L1，与 task35 并行）

# 测试方法
`python3 skills/research.py --role builder` 等四角色各出合法 JSON（三元组非空 + suggested_queries 非空）；`--role coding --cell "impl_strategy=m_quarters"` focus 含格信息

# 禁止事项
- 不执行网络请求（纯配置/组装）；不改框架
