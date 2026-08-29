# Research: E 阶段候选来源理论核验（仅文献验证，无映射）
## Summary
六项理论的一手文献全部核实：所给期刊卷期页码与出版信息（Kelly 35(4)、Brier 78(1)、Murphy 12(4)、G&S 70(3)、Coase 4(16)、Williamson=Free Press）均正确。核心主张可忠实转述；两处术语归属需不确定标记。
## Findings
### 1. Kelly 1956
引用：Kelly, J. L., Jr. (1956). "A New Interpretation of Information Rate." *Bell System Technical Journal*, 35(4), 917–926.（1956 年 7 月）✓
转述：赌徒可借信道接收的信息使资金指数增长，最大指数增长率等于信道信息传输率；原文以含本金返还倍数 α 的记法给出最优下注份额，单事件解等价于 p−q/b_net，故 f*=(bp−q)/b 是其标准等价形式。原文明确指出每次全押者若无限期继续将以概率 1 破产，并论证只在“不复利、固定注额”情形才应最大化单次期望值；Kelly 准则最大化长期复合（对数）增长率。
不确定标记：原文未逐字出现“(bp−q)/b”（原文用 αs 记法），该写法是后世标准化等价式；主张本身无不确定。

### 2. Brier 1950 / Murphy 1973
引用：Brier, G. W. (1950). "Verification of forecasts expressed in terms of probability." *Monthly Weather Review*, 78(1), 1–3. ✓；Murphy, A. H. (1973). "A new vector partition of the probability score." *Journal of Applied Meteorology*, 12(4), 595–600. ✓
转述：Brier 提出以预报概率与事件结果的平方误差评分（Brier score）验证概率预报。Brier 原文未做分解；Murphy 1973 将其分解为 reliability、resolution、uncertainty 三项，成为校准评估的标准工具。
不确定标记：无重大不确定；三项分解术语应归于 Murphy，不应归于 Brier。

### 3. Tetlock & Gardner 2015
引用：Tetlock, P. E., & Gardner, D. (2015). *Superforecasting: The Art and Science of Prediction*. New York: Crown Publishers. ✓
转述：基于 IARPA 资助的 ACE 预测锦标赛中 Good Judgment Project 的多年数据：普通预测者经遴选与校准/去偏差训练后，长期概率预测准确性显著超过受控对照组与情报分析师。配套论文显示约一小时的训练使 Brier 准确度提升 6–11%；超预测者的关键习惯是频繁、小幅、渐进地更新信念——Atanasov, Witkowski, Ungar, Mellers & Tetlock (2020, *OBHDP*, 160:19–35) 以 GJP 数据证实增量式更新者更准。
不确定标记：6–11% 数字出自配套学术论文而非书内；“更新频率与准确性”的严格实证以 Atanasov et al. (2020) 为准。

### 4. Grossman & Stiglitz 1980
引用：Grossman, S. J., & Stiglitz, J. E. (1980). "On the Impossibility of Informationally Efficient Markets." *American Economic Review*, 70(3), 393–408.（1980 年 6 月）✓
转述：若价格完全反映一切可得信息，则花成本获取信息者得不到回报，无人再有激励获取信息；若无人获取信息，价格又无从信息有效。故完全信息有效的均衡不可能存在，均衡只能是部分反映信息的（含噪声的）近似有效。原文（AEA PDF）：“套利者从其（私人）有成本的活动中得不到（私人）回报”。
不确定标记：无；所给卷期页码正确。

### 5. Coase 1937
引用：Coase, R. H. (1937). "The Nature of the Firm." *Economica*, New Series, 4(16), 386–405.（1937 年 11 月）✓
转述：使用价格机制（市场）本身有成本：发现相关价格、谈判与签约成本。当市场组织一笔交易的成本高于在企业内以权威组织同一交易的成本时，交易被内部化，企业因此存在。企业边界扩张至“在企业内组织一笔额外交易的成本等于通过市场组织它的成本”之处。
不确定标记：无重大不确定；“边界=两种成本比较”为忠实转述，原文另含多厂企业等限定讨论。

### 6. Williamson 1985
引用：Williamson, O. E. (1985). *The Economic Institutions of Capitalism: Firms, Markets, Relational Contracting*. New York: Free Press.（xiv+450 页）✓
转述：以有限理性与机会主义为行为假设，主张按交易属性（资产专用性、频率、不确定性）把治理结构（市场、混合、科层）与交易做效率匹配。资产专用性使事前大量竞标的关系事后发生“根本性转化”（原文 pp. 61–63）——变为双边依赖的小数目关系，产生套牢/要挟风险，从而决定 make-or-buy 边界。
不确定标记：“hold-up”一词原始出处更常归于 Klein–Crawford–Alchian (1978)；作为 Williamson 体系标签属通行转述。

## Sources
- Kept（一手/官方）：Wiley/IEEE + archive.org（bstj35-4-917）+ 原文 PDF 镜像（Kelly）；AMS 官方期刊页（Brier 1950、Murphy 1973）；PenguinRandomHouse/Google Books（Superforecasting, Crown, 2015）；AEA top20 PDF + RePEc（Grossman & Stiglitz 1980）；Wiley *Economica* 官方页（Coase 1937）；Internet Archive/Google Books/gbv.de（Williamson 1985, Free Press）；ScienceDirect（Atanasov et al. 2020）；SAGE（Mellers et al. 2014，GJP 夺冠）。
- Dropped：Wikipedia、R Discovery、ResearchGate、ebrary 等二手聚合页——仅交叉参考，不作引用依据。

## Gaps
- Superforecasting 仅书目级验证，未逐页核对书内章节；训练 6–11% 出自配套论文（Cambridge Core: Developing expert political judgment），其作者列表未逐字核对（低置信，建议引 Chang et al.）。
- 若需逐字引用 Kelly 原文公式，应以原文 PDF 记法（αs）为准；f*=(bp−q)/b 为标准换算。

