# leaf-1: adjust 类角色存在性 + PopulationStrategy 钩子面

对象：openevolve main@9196d87（/tmp/openevolve-src 代码直读）｜置信度：高（源码全文读+调用点逐一定位）

## 一、adjust 类角色存在性判定：不存在（置信度高）

全仓 grep `meta.optim|strategy_agent|self_modif|dynamic_feature|cell_add|adjust`（排除 test）仅命中 1 处注释（controller.py:347 "adjust the start_iteration"）。无策略 agent、无元优化器、无运行时特征空间调整、无格增删机制。openevolve 的"元层"仅有一个**决策建议器**（population.py:51 PopulationStrategy），且其文档字符串自述边界："Strategies inspect detached snapshots and return decisions. ProgramDatabase owns all changes to programs, islands, feature maps, and the archive."（population.py:4-6）——策略只能"建议"，不能"执行"。

## 二、PopulationStrategy 六钩子面（population.py:51-68）

| 钩子 | 签名 | 语义 | 调用点 | 可变对象 |
|---|---|---|---|---|
| admit | (snapshot, candidate, island_idx) -> bool | 入库前否决候选 | database.py:365-373 | 无（仅读 snapshot） |
| replace_cell | (snapshot, existing, candidate, gen) -> bool | 同格新旧竞争裁决 | database.py:411-420 | 无 |
| archive | (snapshot, candidate) -> ArchiveDecision | 精英档收纳/驱逐 | database.py:1280-1291, 1892-1893 | 无 |
| evict | (snapshot, required, protected_ids) -> Sequence[str] | 种群超额时选牺牲者 | database.py:1935-1950 | 无 |
| migration_due | (snapshot) -> bool | 是否触发岛间迁移 | database.py:2022-2027 | 无 |
| migrate | (snapshot) -> Sequence[MigrationMove] | 迁移哪些程序去哪岛 | database.py:2043-2050 | 无 |

共同形态：frozen snapshot 入参（PopulationSnapshot 为 frozenset/MappingProxyType 快照，population.py:22-36）+ 返回纯数据决策对象 → **无副作用通道**。调用点统一包裹 `_strategy_boundary()`（database.py:39-44）：异常被捕获标记不中断主流程（策略故障不致崩库，但也意味着策略异常静默降级）。

## 三、三承载判定（rick super adjust agent 需求）

| rick 需求 | 能否承载 | 证据与理由 |
|---|---|---|
| (a) 增格/删格 | ❌ 不能 | 格集合=`island_feature_maps` dict（database.py:155,397），仅随程序入库隐式创建（480 行写入）、随死亡程序清理（880-882 行）；无任何 API 主动增删格；feature_dimensions 在 config 静态定义（config.py:332-344）、init 一次性构建 bins（database.py:157-165），运行期不可变。钩子拿到的是 detached snapshot，写回无门 |
| (b) 格子降采样权 | ❌ 不能 | 权重仅存在于**程序级** fitness 采样 `_sample_from_island_weighted`（database.py:1599-1638，weight=max(fitness,0.001)），非格级；无格 visit 计数、无 UCB、无降采样语义（"格子只降权不删格"无处安放）；六钩子无一个作用于采样路径 |
| (c) 正交性审计 | ❌ 不能（只读建议可，执行不可） | feature_dimensions 是扁平字符串 list（config.py:332），无维度相关性/正交性概念；唯一"多样性"是 edit_distance（diversity bin 用，database.py:1080-1089）非格正交性；策略钩子可在 admit/replace_cell 里读 snapshot 做审计判断返回 bool——但审计结果只能影响单个候选接纳，不能改格结构 |

## 结论

1. openevolve 无任何 adjust 类角色：无策略 agent/元优化器/动态特征空间，唯 PopulationStrategy 六钩子且只读快照。
2. 六钩子全部是"决策建议"形态，无副作用通道：增格/删格/格降权/正交性审计四能力均无载体。
3. admit/evict/archive 可承载"精英重评估/延迟入库"类弱干预（拒收/驱逐），但不能动格结构与采样权。
