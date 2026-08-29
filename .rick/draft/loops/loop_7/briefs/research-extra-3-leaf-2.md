# 调研:自改进系统 × 「正确×非共识」 × 预测市场/科学

## 子问题1:自修改系统架构经验
1. Gödel machine(Schmidhuber 2003):任意自身代码可改,但须先由证明搜索器证得"改写有用";效用函数/硬件/初始代码由公理固定为证明基准,证明即改动闸门,且保证全局最优(无局部最优)。[arXiv:cs/0309048]
2. GCC三阶段自举(stage1-3),`make compare`逐对象对拍后两阶段;Rust自宿主以固定beta快照为stage0——把验证面压到差分面。[GCC Internals makefile.texi;rustc-dev-guide "Bootstrapping"]
3. Thompson 1984:自宿主编译器可注入源码不可见的后门——自修改系统的验证面=整条自举链,非单一代码库。[CACM 27(8):761-763]
4. seL4:8,700行C(600行汇编)的全功能正确性验证需20万行Isabelle证明——验证成本≫实现成本,故内核极小化以控验证面。[Klein et al., SOSP 2009;trustworthy.systems证明统计]
5. SPIN(SOSP 1995):应用经语言/链接期安全机制以"扩展"安全改OS接口与实现,内核核心不动——"改扩展不改内核"先例。[Bershad et al., doi:10.1145/224057.224077]
6. 遗传编程:交叉高度破坏性(早期仅~19.5%子代不差于亲代,劣化远多于改善),幸存个体靠中性码膨胀(intron/bloat)自保护——大改动存活率低,自改进实际走小步+保护层。[Langdon bloat_wsc2;Langdon&Poli 1997 "Fitness Causes Bloat"]

## 子问题2:「正确×非共识」论证
7. Marks《The Most Important Thing》(2011)首章"Second-Level Thinking":超额收益要求非共识且正确的价值观点。[书;Oaktree备忘录It's Not Easy 2015重刊]
8. 更正:共识收益表述"与所有人做同样动作不可能跑赢"出自Dare to Be Great(2006,见Dare to Be Great II 2014重引);"Us and Them"(2004-05-07)主题实为"I know/I don't know"两派与过度自信。[oaktreecapital.com备忘录PDF]
9. Steinhardt《No Bull》(2001):variant perception="有充分依据、与市场共识显著不同的观点",称其为唯一管用的分析工具;其基金28年费后年化≈24.5%。[书,书摘acquirersmultiple.com]
10. Buffett 1986股东信:"别人贪婪时我们恐惧,别人恐惧时我们贪婪"。[berkshirehathaway.com/letters/1986.html]
11. Soros《The Alchemy of Finance》(1987):"公认观点是市场永远正确;我持相反观点:市场价格永远错误";仓位角色:"关键不是对错,而是对时赚多少、错时亏多少"(Druckenmiller转述,二手)。[书;irishtimes.com]
12. VoI:Howard 1966"Information Value Theory"(IEEE SSC-2(1):22-26):信息价值=对决策期望值的改进;不改变决策的信息价值为零——信息优势须有可变动作(仓位)才兑现。[doi:10.1109/tssc.1966.300074]
13. Grossman-Stiglitz(1980,AER 70(3):393-408):价格若已完全反映信息,则无人能从昂贵的信息获取中获得回报——完全信息有效市场不可能;超额收益=未被共识定价信息的补偿。[aeaweb.org/aer/top20/70.3.393-408.pdf]

## 子问题3:预测市场与科学
14. Kuhn《科学革命的结构》(1962/1970):常规科学=范式共识内解谜(ch.IV);异常积累→危机→革命(ch.VI-IX);"拒绝一个范式总是同时接受另一个",不由证明裁决(ch.IX)。[SSR 2nd ed.;plato.stanford.edu/entries/scientific-revolutions]
15. Planck《Scientific Autobiography》(1949):新科学真理胜出"是因为反对者终将死去"——反共识判断的兑现以代际计(Planck原则)。[mathshistory.st-andrews.ac.uk/Planck/quotations]
16. 预测市场:价格聚合分散信息,预测误差常低于常规方法[Arrow等,Science 320:877,2008];对数效用等条件下价格≈财富加权平均信念——个体影响力(下注财富=仓位)即对共识的权重。[Wolfers&Zitzewitz,JEP 18(2) 2004/NBER w12200]
17. 方法学综合(推断,标注):由VoI+G-S+财富加权定价:已入共识定价的判断边际信息价值≈0,价值形态转为执行价值(大仓位可执行、低边际风险);认知性超额只存在于偏离共识处。库恩对应:范式内=执行生产力,范式外=认知增量。

## 方法学注
- 一手来源优先(论文/原书/官方文档);引语以可核对文本为准;二手转述(Soros"赚亏"语)已标注;Us and Them按原文更正定位。

## 结论
1. 自修改系统工程共识:固定不变量+证明/对拍闸门+改扩展不改内核,把验证面压到差分面。
2. 投资文献一致:超额收益=正确×非共识×仓位,VoI与Grossman-Stiglitz提供其决策论基础。
3. 共识定价使边际认知价值趋零:影响力大的共识判断价值形态=执行/规模,认知增量在反共识。
