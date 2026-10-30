# task35：gen_env 检索三问 + super-loop 检索定位（突破局部最优）

# 任务目标
把「检索 skill」变成 gen_env 的生成与追问对象（human 要求：建设 env 时引导用户创造 search skill），并在 super-loop 里确立「检索 = 突破局部最优的手段」。

# 关键结果
- `.rick/skills/gen_env_skill/skill.md`：
  - 三要素追问之后增**「检索三问（第四类资产）」**：
    1. **搜索方法**：你希望 agent 用什么方式检索？（web_search 关键词组合 / 官方文档站内 / GitHub 代码搜索 / arXiv 站内与 API）
    2. **搜索信源**：本域最可信来源清单（按角色分：coding/debug=使用手册与代码样例；builder/adjust=论文）
    3. **关注点**：每类 agent 在该域该关注什么（**必须与特征格绑定**——如「该格关注 BLAS 派发 API 约束」vs「该格关注 cache 分块实验数据」）
    - 判据：能写成 `skills/research.py` 的 `ROLE_PROFILES`（四角色各一组方法/信源/关注点 + 查询模板）
    - 纪律：human 亲自描述 + AI 追问（如「你的域里，开发者查什么文档最有效？」「论文该关注创新点还是实验细节？」）
  - **生成清单升级**：`skills/{profile,benchmark,verify,gate_check,research}.py` **五件**
  - 验收自检增：五件齐备 + research.py 四角色输出合法 + 三元组非空
  - 第 7 节轨迹分析增：读 `designs/*.papers` 判断「检索是否真的被用上/是否只检索不落地」
  - **定位说明**：检索是语义空间进化的「认知注入」通道——当 agent 在同一格以小角度反复微调（局部最优征兆），检索能引入外部新原理；轨迹分析发现「某格连续多轮 idea 同质」→ 检查 research 三元组是否太窄
- `.rick/loops/super-loop.md`：
  - Step 2 增「检索三问」为 env 建设的一部分
  - Step 7 增：检索质量复盘（papers 数量/质量/是否落到计划；同质 idea → 放宽或改信源）

# 写域
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

# 依赖关系
- 无（L1，与 task32 并行）

# 测试方法
关键词断言：gen_env 含「检索三问」「research.py」「五件」；super-loop 含「检索」

# 禁止事项
- 不改框架/模板
