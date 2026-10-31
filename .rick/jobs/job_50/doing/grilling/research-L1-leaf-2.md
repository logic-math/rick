# L1-leaf2 调研：LLM 导师判别「提取失败 vs 真不会」与节奏控制（advance/regress）

**离线声明**：本沙箱无外网（DNS 劫持至内网，web_search/fetch_content 全部失败，主管决策超时未复）。以下为模型内部知识整理，**所有 URL 均未在线核验，信源等级一律 C（线索）**；括注原性质（A=同行评审，B=官方文档），使用前须人工核验。

## A. 一时想不起来（提取失败） vs 真不会

1. **部分线索复测**（Tulving & Pearlstone 1966，availability vs accessibility）：自由回忆失败 → 给类别/首字母/语境线索重测；线索能救回 = 提取失败（知识在、不可达），线索后仍失败 = 真不会。[原A·未核验] https://doi.org/10.1016/S0022-5371(66)80042-9
2. **换措辞 + 延迟复测**：原题照抄通过仅证短时记忆；换措辞或隔轮重问通过 = 已掌握，两轮多线索仍败 = 缺口（测试效应文献）。[原A·未核验] https://doi.org/10.1111/j.1467-9280.2006.01693.x
3. **teach-back 操作化判据**（AHRQ 健康素养工具包 Tool 5）：只用开放题（「用你自己的话说说你会怎么做」），禁是非问；判懂 = 不照读原文、用自己的话、关键步骤齐全；不过 → 换法重教一次 → 复测，二不过即判缺口。[原B·未核验] https://www.ahrq.gov/health-literacy/improve/precautions/tool5.html
4. **闭环编码**（Schillinger 2003, "Closing the Loop"）：教师澄清直至学生正确复述才记「闭环」，未闭环 = 真没懂。[原A·未核验] https://doi.org/10.1001/archinte.163.1.83
5. **AutoTutor 五级梯度**（Graesser）：pump（"还有呢？"）→ hint（方向提示）→ prompt（挖空填一词）→ prompt again → assertion（直给答案）；级内失败即升一级，多轮未满足 expectation 则 assertion 后回退。[原A·未核验] https://ojs.aaai.org/index.php/aimagazine/article/view/5360
6. **Khanmigo**（2023 泄露提示词及报道）：苏格拉底式、一次只问一问、拒绝直接给答案；卡壳 → 提示（nudge/hint）→ 分解小步（break it down）→ 类似例题。**具体英文措辞离线无法核验**。[C·未核验] https://www.businessinsider.com/khanmigo-tutor-ai-prompt-leaked-gpt-4-2023-6

## B. 节奏控制（advance/regress）

7. **LearnLM 教学指南**（arXiv:2412.16429）：含「响应学习者需求（尤其困惑/挫折）」「不独占对话」「适当脚手架/步调」等原则：阶梯提示、不直接给答案。[原B·未核验] https://arxiv.org/abs/2412.16429
8. **productive confusion**（D'Mello & Graesser 2012）：困惑被解决 → 深度学习；持续未解 → 挫折 → 脱离。操作化：困惑 2 轮未解 → 换措辞/降难度；挫折信号出现 → 立即回退一步 + 情绪确认。[原A·未核验] https://doi.org/10.1016/j.learninstruc.2011.10.003
9. **综合推进/回退规则**（多源综合，C）：答对且能 teach-back → advance；单次错误 → 同层级换措辞复测；连续 2 错 → 升提示层级/分解小步；困惑 2 轮未解或挫折语（「我不会」「没意思」）→ regress + 共情。[C·综合]

## 结论

1. 部分线索或换措辞能救回 = 提取失败；两轮多线索仍败 = 真不会。
2. teach-back 判懂：开放问、自己的话、步骤齐全；二不过 = 缺口。
3. 答对且能讲 → 推进；连错/困惑两轮 → 升提示；挫折信号 → 立即回退共情。

## Gaps

全部 URL 未核验（离线环境）；Khanmigo 提示词逐字原文、LearnLM 指南逐条措辞未能抄录。有网后需逐条核对（arXiv/AHRQ/DOI/BI）并补引句；信源等级可能随核验升级。
