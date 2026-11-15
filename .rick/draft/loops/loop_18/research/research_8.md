# research_8 — 心理测量学「题目即仪器」三工艺：题目参数漂移 / 曝光控制 / 锚题等值（探索流点火候选） — 2026-11-15

> 时间基准 2026-11-15。信源等级：A = 同行评审论文/专著/带 DOI 文献记录（书目与摘要经 Crossref+OpenAlex 公开 API 逐条核实）；B = 教科书级通行内容；C = 通行转引（仅线索，已标注）。核实通道说明：本轮 web_search 各服务商均限流或无密钥（Exa 429、Brave/Tavily/Jina/OpenAI 无 key、DDG/Wikipedia 直连被墙），全部人名/年份/出处改经 Crossref/OpenAlex API curl 直连核实，详见 R7。禁重复核对：三概念均不在已给清单（Fisherian runaway/Donabedian/ZKP/法兰西学院/信号检测论/LOSA/Willard/PCC）与他已知域（Goodhart/可证伪性/哥德尔/A-B 与 benchmark 工程常识）内。

## 概念 1｜题目参数标定与题目参数漂移（item parameter drift, IPD）
① 名称与领域：心理测量学/项目反应理论（IRT）的运营质量控制工艺。
② 真实来源：IRT 基石 Lord《Applications of Item Response Theory to Practical Testing Problems》(1980, Routledge, DOI 10.4324/9780203056615)；三参数逻辑模型由 Birnbaum 的贡献章节引入 Lord & Novick《Statistical Theories of Mental Test Scores》(1968)（书目 A，3PL 归属为教科书级通行内容 B）；漂移检测对比 Donoghue & Isham《Comparing the Effectiveness of Procedures to Detect Item Parameter Drift》(1996, ETS RR)；CAT 池内标定与漂移 Glas《Item Calibration and Parameter Drift》(2000, van der Linden & Glas 编 Computerized Adaptive Testing: Theory and Practice)；SAT 尺度漂移 Liu, Curley & Low《A Scale Drift Study》(2009, ETS RR)。（A，另注除外）
③ 是什么：IRT 把每道题标定为带参数的仪器——难度 b、区分度 a、伪猜测 c 决定该题在能力谱哪一段、以多陡的斜率携带信息；题目不是中性的对错过滤器，而是有响应曲线的探头。参数不从原理推出，而从作答数据估计（标定 calibration）。但标定会失效：ETS 把 SAT 1994 与 2001 两个题表在 2005 年原样重施并重新等值，发现语文尺上漂、数学尺下漂（Liu/Curley/Low 2009），据此建议测试机构把旧题表排进周期性复测日程，主动监测尺度稳定性。漂移检测有成熟 bake-off：Donoghue & Isham (1996) 蒙特卡洛对比多种统计量，Lord 的 χ² 胜出（需约束 c 参数跨标定相等）。
④ 与映射目标的结构关系：他的判定器 = 一把标定过的仪器，「担心判定器随迭代漂移」正是 IPD 的标准命题：仪器参数不是常量，但漂移可检测、可量化、可入监测日程。他已有的 task 四元组不变量对应「冻结题表重施协议」——用不变集周期性复测判定器的漂移量，而不是默认它稳定；Lord χ² 给出现成的检测统计量。

## 概念 2｜题目曝光控制（item exposure control）
① 名称与领域：计算机化自适应测验（CAT，背景 Weiss & Kingsbury 1984, J. Educational Measurement 21(4)）的题库安全工程。
② 真实来源：Sympson & Hetter (1985) 会议论文首倡（原作未正式出版，经 van der Linden 2003 JEBS 与 Chen/Spray/Ankenmann 2003 JEM 转引核实，C）；制度化记录 Hetter & Sympson《Item exposure control in CAT-ASVAB》(1997, APA 章节, DOI 10.1037/10244-014)；按能力条件化 Stocking & Lewis (1998, J. Educational and Behavioral Statistics, DOI 10.3102/10769986023001057)；rmax 与重叠率关系 Chen, Spray & Ankenmann (2003, J. Educational Measurement)；偷题伤害实证 Yi, Zhang & Chang《Severity of Organized Item Theft in CAT》(2006, ETS RR)；泄漏参数检测 Liu, Han & Li (2019, Frontiers in Psychology)。（A，另注除外）
③ 是什么：CAT 为省题量按「当前能力估计处信息量最大」贪心选题（maximum item information），高区分度好题因此被几乎所有人看到——单题曝光率失控。工艺：给每题设曝光上限 rmax，即使选题器选中，也只以概率 p 实际施测（Sympson-Hetter 概率门控），p 由模拟迭代整定至实测曝光 ≤ rmax；Stocking-Lewis 把控制条件化到能力分层。Chen/Spray/Ankenmann (2003) 证明控制 rmax 同时直接控制了考生间试题重叠率。泄漏侧：Yi/Zhang/Chang (2006) 模拟有组织偷题，发现伤害「在题池启用早期最严重」；Liu/Han/Li (2019) 为被泄题目建「泄漏参数」模型做在线检测。
④ 与映射目标的结构关系：他对 benchmark 泄漏的工程直觉在此被铸成定量旋钮：rmax = 单题被优化器消耗的速率上限。自我改进系统 ≈ 贪心选题器——查询预算会自动集中到信息量最高（恰是泄漏价值最高）的少数任务上；曝光控制 = 评测循环内对基准子集限流配给（同构他熟的 rate limiting），以测量精度换泄漏面。「题池早期失窃伤害最大」直译为：新建基准集在最初几轮迭代最容易被吸干信息。

## 概念 3｜锚题等值与链接（test equating & linking）——换题不换尺
① 名称与领域：心理测量学的等值/链接工艺，制度级应用见 TIMSS/PISA/SAT。
② 真实来源：教科书 Kolen & Brennan《Test Equating, Scaling, and Linking》(Springer, 1995/2004/2014, DOI 10.1007/978-1-4757-4310-4)；设计名 NEAT（非等组锚题设计）见 Sinharay & Holland (2009, ETS RR)；锚题规格研究 Ricker & von Davier (2007, ETS RR) 与 Lu, Haberman & Guo (2015, ETS RR)；TIMSS 趋势锚题漂移实证 Park, Lee & Xing (2016, Frontiers in Psychology, DOI 10.3389/fpsyg.2016.00255)；PISA 案例 Cosgrove & Cartwright (2014, Large-scale Assessments in Education) 与 Jerrim et al. (2023, British J. Educational Studies)。（A）
③ 是什么：高利害考试不能复用题表，但跨年分数必须可比——等值 = 换题不换尺：在新旧题表里嵌一组共同锚题（anchor test），锚题把两表分数换算到同一把尺上。锚题须在内容与难度上有代表性且自身参数稳定，否则整条换算链失真（锚长影响等值结果：Ricker & von Davier 2007；锚题选择本身带入偏差：Lu et al. 2015）。制度实践：TIMSS 靠「trended anchor items」跨周期维持趋势可比，但实证发现锚题自身也有漂移证据（Park/Lee/Xing 2016 对 1999/2003/2007 三轮 21 道趋势锚题）；PISA 出过爱尔兰 2009 阅读「38 国最大跌幅」调查（Cosgrove & Cartwright 2014）与 2018 西班牙阅读分被 OECD 拒发（Jerrim et al. 2023）；SAT 亦曾整尺重刻（"Recentered SAT I"，术语见 Bridgeman et al. 2000, ETS RR）。
④ 与映射目标的结构关系：他的 task 四元组不变量在心理测量学里有百年身份——锚题：不变集不必是全部基准，而是让其余题目可以增删、判分尺仍可换算回历史原尺的那座桥。这把不变量从「防御性冻结」升级为「可携带的换算协议」：允许换题（对抗泄漏与过拟合），代价是可量化的 linking error；而锚题自身会漂移（Park/Lee/Xing 2016）恰好接回概念 1——桥不是免费的，锚要定期复检。

## 排序（最可能点燃）
概念 2 曝光控制最可能点燃：它把他已有直觉（泄漏）但缺机制的轴变成定量旋钮 rmax + 概率门控（同构他熟悉的限流原语），并携带反直觉经验律「题池早期失窃伤害最大」，周一就能用的密度最高；概念 3 次之（给其不变量以学科身份并解锁换题自由）；概念 1 收口（把担心的漂移变成可检测协议，与前两者闭环）。

## R7 上报项
- Sympson & Hetter 1985 原始出处仅经后续文献转引，会议名/地点未核到一手记录（正文已标 C）。
- SAT recentering 的具体年份未核到一手出处，正文仅保留论文标题可证的「Recentered SAT I」术语。
- 曝光率目标常用数值（如 0.20）与 randomesque 法（Kingsbury & Zara 1989）的命名归属未核到一手数字/原文，正文均已略去。
- 本轮 web_search 工具全服务商限流/无密钥，核实改经 Crossref/OpenAlex API（书目+摘要级）；若母会话要求严格经 web_search 工具复核，需补跑。

## Gaps
- DIF（差异题目功能——同一题对不同群体测的不是同一个量，Mantel-Haenszel/SIBTEST 谱系）未纳入本轮 2-3 概念限额，留作下一轮候选。
- 各文献仅核到摘要与书目记录，正文未逐页核对；PISA 2018 阅读趋势断裂的技术细节（趋势锚定设计变更）只核到西班牙拒发与爱尔兰案例层面。
