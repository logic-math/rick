# Research: E 阶段视角候选来源理论核验（纯文献验证，不做 rick 映射）

## Summary
五项文献的引用信息均经权威来源交叉确认，核心主张与任务描述一致。三处需标不确定：Baldwin-Clark 期权论证宜表述为金融/实物期权框架而非具体公式；Conway 期号存在 14(4)/14(5) 引用差异；Burnet 原文用"禁忌克隆"，"阴性选择"系后世术语。

## 1) Parnas 1972
①Parnas, D. L., "On the Criteria To Be Used in Decomposing Systems into Modules," Communications of the ACM 15(12):1053–1058, 1972（DOI 10.1145/361598.361623；前身系 1971 年 CMU 技术报告）。②论文以同一设计问题对比常规（按流程）与非常规（按设计决策）分解，主张模块化的目标是提升灵活性、可理解性并缩短开发周期，成败取决于分解准则。核心准则即信息隐藏：每个模块应隐藏一个困难或易变的设计决策，使变更局部化于模块内部。③不确定：无（卷期页码均确认）。

## 2) Baldwin & Clark
①Baldwin, C. Y. & Clark, K. B., Design Rules, Vol. 1: The Power of Modularity, MIT Press, 2000（2000-03-02 出版）；HBR 文 "Managing in an Age of Modularity," Harvard Business Review 75(5):84–93, 1997 年 9–10 月。②模块化把设计切成可见的"设计规则"（架构、接口、测试）与隐藏模块：隐藏模块内部决策不影响其他模块，只需遵守规则即可独立设计。作者用金融期权理论论证期权价值：隐藏模块相当于可实验、可替换的实物期权（论文引用 Merton 期权定价文献，后续论文明确"基于金融期权理论"建模）。设计规则须早期确立并冻结：改架构牵动全部模块，改隐藏模块成本低，构成成本不对称。③不确定：书中是否逐字套用 Black-Scholes 公式未逐页核实，宜表述为"金融期权/实物期权理论框架"。

## 3) Conway 1968
①Conway, M. E., "How Do Committees Invent?" Datamation 14(4):28–31, 1968 年 4 月。②原始表述（论文基本论题原文）："设计系统（广义）的组织，只能产出其组织沟通结构之复制品式设计。"作者论证组织内沟通路径决定接口协调：无沟通路径处接口无法协调，故沟通结构必然映射为设计结构。③不确定：个别来源（部分维基引用）作 14(5)；作者官网确认 1968 年 4 月发表，14(4):28–31 为通行引用。

## 4) Burnet 1959
①Burnet, F. M., The Clonal Selection Theory of Acquired Immunity, Nashville: Vanderbilt University Press, 1959（1958 年 Vanderbilt 大学 Abraham Flexner 讲座；英版由 Cambridge University Press 同年出版）。②克隆选择学说：免疫细胞遗传携带单一特异性受体，抗原"选择"匹配细胞并驱动其克隆增殖分化，形成抗体克隆与免疫记忆。针对自身抗原的反应性克隆在免疫成熟前被清除或抑制（Burnet 称"禁忌克隆"被稳态机制消除），以此解释自我/非我识别与获得性耐受。③不确定："阴性选择（negative selection）"是后世（T 细胞/胸腺文献）术语，非 1959 原文用词；学说雏形亦见其 1957 年论文。

## 5) Matzinger 1994
①Matzinger, P., "Tolerance, Danger, and the Extended Family," Annual Review of Immunology 12:991–1045, 1994 年 4 月（DOI 10.1146/annurev.iy.12.040194.005015）。②危险模型：免疫系统首要驱动不是自我/非我判别，而是检测并防御危险——免疫应答由受损/受胁迫组织发出的危险信号触发。非我判别不充分：共生微生物、食物、胎儿皆"非我"却常不引发应答。机制上由抗原提呈细胞（树突状细胞）感知危险后提供共刺激而启动应答。③不确定：无（卷期页码与摘要由 Annual Reviews 官方页确认）。

## Sources
保留：ACM DL（Parnas 卷期页权威确认）；MIT Press/HBS（专著与 HBR 文出版信息）；melconway.com 作者官网（Conway 原文表述与发表时间）；Archive.org/BHL（Burnet 书名页扫描）；Annual Reviews/Europe PMC（Matzinger 摘要）；SSRN 312404/6508（隐藏/可见模块定义、金融期权理论依据）。舍弃：ResearchGate/Academia.edu 镜像（非权威，仅旁证）。

## Gaps
Black-Scholes 公式在书中的具体数学呈现未逐页验证（SSRN/MIT Press 直连失败，依赖检索摘要与权威转引）；Conway 原刊扫描件未直接获取，仅据通行引用与作者官网。
