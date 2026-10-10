# 调研：如何呈现信息能提升 human 对 AI 输出的判断质量

## (a) 决策科学：choice architecture 与认知负荷
结论：降低外在认知负荷、强制比较式判断的呈现证据最强；单纯附加解释无效甚至更糟。
- 对比呈现优于单独绝对评分：比较判断比绝对打分更可靠（Thurstone 1927 传统；Christiano et al. 2017 实证）。多次复现。
- 认知负荷理论：减少 extraneous load 提升判断质量（Sweller 1988）。多次复现。
- choice architecture：选项结构与默认值系统性改变选择（Thaler & Sunstein 2008；Johnson et al. 2012）。多次复现，效应量有争议。
- AI 场景：显示 AI 置信度改善信任校准（Zhang et al. 2020 FAT*），单次；结构化清单在医疗强复现（Pronovost et al. 2006 NEJM），AI 场景初步。

## (b) 人类注意力/疲劳
结论：实验室警觉衰减高度复现；真实自定节奏专家场景被反驳；自动化自满是更稳的效应。
- vigilance decrement：持续监控 20–30 分钟检出率下降（Mackworth 1948；Warm et al. 2008 综述）。多次复现。
- 反驳：乳腺癌筛查约 150 万判读，自定节奏无衰减（Suleman et al. 2024 PNAS）。
- automation complacency：系统越可靠人越松懈（Parasuraman & Manzey 2010 Human Factors）。多次复现；2024 PLOS ONE 实验：加人监督反降准确率。
- 标注质量：900 万条 AMT 标注不随月份衰减（Difallah et al. 2016），反驳长期疲劳说；短会话评分疲劳仅单次小样本。
- AI 监督：注入真实错误可恢复审阅警觉（Fang et al. 2025，12 周随机实验，单次）；眼动研究：建议不被采纳也占用注意（ICSE 2026，33 人）。未定位到 Bhatt et al. 原文，以上替代。

## (c) 已验证的「AI 呈现 + human 裁决」实践
结论：工业收敛于成对/排序比较 + 多维结构化 + 批量流水线；须用强制参与防过度依赖。
- Christiano et al. 2017：成对比较比绝对打分对人更易、同样有效。
- Ouyang et al. 2022（InstructGPT）：标注员排序 4–9 条输出，信息效率高于二元选择；1.3B 模型胜 175B 验证有效。
- Bai et al. 2022（Anthropic）：多轴 Likert + 二元比较 + 文字理由，周更批量数据；Constitutional AI：AI 批量生成批评与比较标签，人只维护原则清单，产物更受众包者偏好。
- 认知强制：Buçinca et al. 2021（CSCW）：「先自判再看 AI」（update-then-show）显著降低过度依赖，仅逐题施加有效；附加解释不降反可能升（Vasconcelos et al. 2023 复现）。
- 边界：pairwise 更易受冗长/风格伪特征操纵（翻转约 35% vs 绝对分 9%，arXiv 2504.14716），需随机顺序 + rubric。

## 结论
1. 用对比/排序呈现替代绝对评分，人类判断更稳更易（多次复现）。
2. 纯加解释不降过度依赖；先自主判断再呈现 AI 才有效。
3. 呈现置信度校准信任；高可靠呈现致自满，注入错误维警觉。
4. 长会话警觉衰减实验室稳健，自定节奏专家场景有反驳。
5. pairwise 有冗长/风格偏差，需顺序随机化与评分细则。
