# research-SR 简报

阶段:S-R 辩证逆转尽调|主题:pi 稳定假设(X)→双循环进化(Y)|基准 2026-08-27
信源标记:[码]=代码原文(0.4)/[文]=文献/公开文档(0.2,已 web 抽验)/[循]=前序简报引证

## ① 逆转命题形式化与阻碍定位

- X(前提,human 三轮拍板确认):继续基于 pi 开发;runtime 重评悬置;唯一撤销条件=遇到 pi 扩展无法满足的功能需求(Y1;长期触发=层 B 骨架替换 Y2)。[循]judgment.md S门禁3/4轮+N2
- Y(目标,N2 五环节操作化定义):human-loop 下判断→plan 细化判断→doing 执行→learning 落盘→dream 统一回收判断轨迹+行为轨迹,持续进化。[循]judgment.md N2
- 逆转命题形式:「若 X 必然,则 Y 应当___」。X 与 Y 的直接交集为空(Y 不要求换 runtime),阻碍全部位于 Y 自身的缺失实现+X 的间接税。
- 阻碍定位(描述符 node/edge,均可溯):
  1. node plan:现实现=需求→任务分解(模块化/粒度/可验证/依赖四原则),输入无 judgment 结构化数据→「plan=细化判断」是规定性(目标态),改造量未计价。[码]internal/prompt/templates/plan.md
  2. node dream:现实现=待回收 job 扫描+prompt 生成+agent 写日志(dream.go 共 91 行),无判断回收/度量/巡检代码→三重负载全为规划中。[码]internal/handler/dream.go
  3. edge human→dream:M1(终审带宽)+M3(判断以天计)恰在唯一双轨迹汇聚点叠加=单点瓶颈。[循]think-gate-N2 Q2
  4. 缺失耦合边:判断→行为、行为→判断两条回路均无机制。[循]think-gate-N2 Q4
  5. edge rick↔pi:悬置期 128 处方言继续膨胀(M4/M8);Y 的进化若绑定 pi 行为方言,进化与漂移将不可分。[循]teach-N1 2.1.3

## ② 候选填空(逆转路径 5 条,跨域,仅列举)

- F1 工程路径:「若 X 必然,Y 应当把 rick-pi 耦合固化为可执行契约」——协议清单+消费者驱动契约测试+静态规则(semgrep/eslint 枚举 128 方言点)+canary 升级跑契约套件。来源:Pact 消费者驱动契约;Ford 等《Building Evolutionary Architectures》2017(fitness functions)。适用:可枚举可断言的接口;不适用:涌现性行为效果。
- F2 度量路径:「Y 应当把双轨迹度量做成自动聚合+抽样审计+human 只审摘要」——固定窗口控制图+例外上报。来源:审计统计抽样(AICPA);SPC 管理用例外;Google SRE 复盘只追 action item。适用:大批量+指标定义稳定;不适用:早期小样本。
- F3 协议路径:「Y 的全部功能只依赖 runtime 中立层」——seam/端口适配隔离 pi 方言。来源:Cockburn 六边形架构 2005;rick 已有 Runtime seam。[循]research-N1。适用:接口面可抽象;代价:适配层维护双写。
- F4 组织路径:「回收仪式分布化」——每 loop 微 AAR+定期汇总,dream 只做终审聚合。来源:美军 AAR TC 25-20(1993-09-30,TRADOC)[文];敏捷回顾。适用:高频低延迟反馈;不适用:需全局统计的判断打分。
- F5 学习系统路径:「判断轨迹=校准训练信号,行为轨迹=经验/技能库,dream=经验汇合点」。来源:Reflexion(NeurIPS 2023,arXiv 2303.11366)、Voyager 技能库(arXiv 2305.16291)、ExpeL(AAAI-24,doi 10.1609/aaai.v38i17.29936)[文]。适用:轨迹可结构化重放。

## ③ 双循环系统先例(结构/成功条件/失败模式)

- 预测锦标赛/Good Judgment Project(Tetlock,IARPA ACE 2011-15):判断轨迹量化=正确性(Brier 分)+校准曲线+更新频率,最接近已拍板三维向量(正确率/赔率/影响)的现成体系;成功=定期打分+反馈闭环;失败=单判断样本稀疏、指标博弈。[文]
- 美军 AAR(TC 25-20 1993):计划vs实际(行为轨迹)+指挥员决策(判断轨迹)同场复盘;成功=标准化四问+心理安全+教训改条令;失败=沦为打卡仪式、教训无人认领。[文]
- 刻意练习(Ericsson 等 1993):成功三条件=任务明确+即时反馈+难度适配的重复;失败=天真练习平台期——「进化vs漂移」判据的直接来源。[文]
- Toyota Kata(Rother 2010):每日改善循环+教练环=行为-元认知双环制度化;失败=产线压力下教练环被跳过。[文]
- RLHF(Christiano 2017/Ouyang 2022):human 判断聚合成模型再驱动行为=判断→行为耦合先例;失败=reward hacking/Goodhart。[文]
- SRE blameless postmortem(Google SRE Book 2016):行为轨迹→系统变更;失败=action item 无主而复发。[文]
- Quantified Self(Wolf&Kelly 2007 起):失败=只采不用的数据坟场;成功=绑定具体决策。[文]
- PKM:Zettelkasten+间隔重复(Anki)=知识轨迹定期回收(dream 形态近亲);失败=收藏癖无输出。[文]
- 共性(归纳):成功系均有「固定回收仪式+结构化记录+反馈到下一轮的显式通道」;失败集中于仪式空转、反馈不闭环、指标被博弈。

## ④ dream 汇聚设计模式(解单点)

- 事件溯源+CQRS(Fowler 2005/Young 2010):轨迹 append-only 为唯一事实,读模型/投影异步构建→dream 只维护投影。适用:轨迹不可变可重放;代价:投影滞后。
- 日志中心化(Kreps《The Log》2013):生产消费解耦,消费方自定步调→human 终审移出在线路径。
- 数据湖/lakehouse(Inmon;Armbrust CIDR 2021):原始落盘+schema-on-read,聚合延后。适用:度量定义未稳期。
- 审计抽样+重要性阈值:全量→抽样,超阈值才上报 human。适用:human 时间稀缺;前提:误差可容忍。
- 自动聚合+例外上报仪表盘(Kaplan&Norton 平衡计分卡 1992;数据可观测性告警):human 只看摘要+异常。
- 数据 mesh(Dehghani 2019):聚合权下放各域,消中心瓶颈。适用:域自治成熟后。
- 仪式化时间盒:固定 cadence+模板限带宽(与已拍板 5-job 窗口直接兼容)。

## ⑤ plan 改造路径(现状→判断细化)

- 差距(事实):现模板输入=需求+domain+loops,输出=任务分解;无判断引用/选项/预期结果字段。[码]
- 选项 P1:plan 模板加判断引用块(判断ID/所虑选项/预期结果+概率/回收时点)——决策日志模式(Mauboussin;Kahneman 倡导)。
- 选项 P2:ADR 块(Nygard 2011;MADR):context/decision/status/consequences 内嵌 plan。
- 选项 P3:意图式计划(任务式指挥 Auftragstaktik;美 ADP 6-0):plan=意图+约束,执行侧自适应——「细化判断」语义最接近的形态。
- 选项 P4:judgment.md 条目→plan 约束/验收标准生成器(决策日志驱动开发)。
- 先例佐证:ADR 已制度化(adr.github.io;Fowler bliki);决策日志+事后打分用于校准训练有预测锦标赛实践支撑。[文]

## ⑥ 双轨迹耦合机制候选

- 双环学习(Argyris&Schön 1978):单环=按规范纠行为(行为轨迹自修正);双环=审规范本身(行为反馈改判断框架)——两条缺失边的现成理论语法。
- 级联控制(控制论):外环(判断)设定值→内环(行为)跟踪;human 延迟→采样保持+低增益防振荡。
- RLHF 式聚合:判断→聚合器→行为模板;已知失败=Goodhart(Campbell 1979/Goodhart 1975)。
- 制度化模板回路:dream 复盘结论→改写下轮 plan 模板/检查单(AAR 教训改条令模式的移植)。
- 元认知校准环:判断带预测→结果解析→校准曲线→下轮置信修正(Tetlock;已拍板 X6 校准实验对应此环)。

## ⑦ 进化 vs 漂移判据

- SPC 控制图(Shewhart 1931;Nelson 规则 1984;CUSUM Page 1954/EWMA Roberts 1959):共因 vs 特因分离;CUSUM 对小漂移敏感。适用:窗口化指标(Y4 周环比已拍板,直接可用);前提:样本量足够。
- 预注册预测:每次变更登记预期效果+方向;进化=预测实现,漂移=未预期变更(Toyota Kata 目标条件模式)。
- 回归基准重放:固定任务套件定期重跑——与已拍板「测评基线降级为环境监测项(选 B)」直接复用。
- Goodhart/Campbell 定律:纯指标进步声明须配定性复核(human 体感已拍板保留,两者互补)。
- CMMI 阶梯:过程定义→度量→优化作为进化阶段判据。
- Ashby 必备多样性(1956):进化声明=可应对扰动类别实际增加,而非活动量增加。

## ⑧ 悬置恢复条件候选(Y1 触发信号源,操作定义清单)

- 现状(事实):Y1/协议清单未写入 domain(pi-runtime.md 无任何触发条款)→恢复条件现仅存 human 记忆。[码].rick/domain/pi-runtime.md
- S1 契约信号:pi 主版本跳变+changelog 涉及 12 钩子/扩展 API(semver 主版本纪律)。检测:release notes diff/semver-diff。
- S2 扩展生态:pi-subagents/pi-web-access 仓库 archived/近 N 月无 commit/issue 无应答,或与锁定版不兼容。检测:仓库元数据轮询。
- S3 需求阻塞事件:具体 rick 功能无法以 12 钩子+扩展事件表达(=Y2 层 B,human 已拍板定义)。
- S4 性能退化阈值:固定探针套件的延迟/成功率越过 SPC 控制限(⑦判据应用于 runtime 指标)。
- S5 上游 EOL:维护模式/仅安全修复公告/仓库转移。
- S6 安全:锁定版本存在未修 CVE(Dependabot 类告警)强制升级决策。
- S7 社区活力塌陷:下载/issue 速率骤降(npm trends)、维护者 bus factor=1。
- S8 canary 契约失败:CI 试升级跑契约套件失败(依赖 F1 先行落地)。
- 先例:锁文件+定距升级(Renovate/Dependabot)、Node LTS 生命周期表、Pact 契约门禁。[文]

## ⑨ 未消解项(R7 上报)

1. 判断回收的 ground-truth 归属:判断「正确性」由谁/何时判定,五环节无归属,需 human 拍板。
2. 赔率共识基线数据源未落地(Y5 方向已定,subagent 搜网络观点的执行规格未定)。
3. plan 消费 judgment 的 schema 不存在,P1–P4 均需新设计+X6 实验。
4. dream 带宽无实测:5-job 窗口需 human 实际分钟数未知,单点结论无法定量。
5. 耦合回路增益/延迟参数无先例可移植,只能实验标定。
6. 128 方言点的静态可枚举性(扫描覆盖率)未验证——F1 的前提。
