# design-tree：env_10 验证 run（用 env_9 复用资产验证五要素/成本归因/格锁）

> 活文档：逐层追加。状态：L1 已消解（research-L1.md）→ L2 判断节点已提炼 → **L3 待 human 裁决（2026-10-26 12:45 提出批量问题 Q1-Q7）**。

---

## 第 1 层（根层 OKR + pipeline）

### O（全局目标）

**用 env_9 的既有资产（env 定义 + 门禁种子）在 env_10 完成 2 迭代真跑验证，产出「五要素 / 成本归因 / 格锁」三特性的判定结论（每特性：通过/失败 + 可复核证据），并汇报实验记录与成本分布（落盘 REPORT.md + 会话呈现）。**

可验证判据：① run 结束 state=stopped、iteration_cursor=2、session_ended 落盘；② REPORT.md 落盘且三特性各有判定与证据；③ 实验记录（六段链）与成本分布表（by_role/角色级两口径）呈现在报告中。

### KR 分解（演绎：KR1∧KR2∧KR3∧KR4∧KR5 ⟹ O）

| KR | 模块 | 职责 | 输入→输出 |
|---|---|---|---|
| KR1 复用落地 | M1 资产搬运 | env_9/env+gates → env_10/env+gates，五件 skill 可跑、键一致 | env_9 资产 → env_10 可运行 env |
| KR2 门先证红 | M2 门禁验证 | preverify_red：45 exploit 全拒 + 金样例零误杀 | env_10/env+gates → ok=true |
| KR3 2 迭代运行 | M3 启动与运行 | scheduler 跑 2 迭代（builder→coding→[debug]→gate→admit/拒） | 空 archive → cursor=2 + 实验记录链 |
| KR4 三特性证据 | M4 证据收集 | 五要素/成本归因/格锁 各自的可观测证据采集 | events/archive/locks/codes → 证据集 |
| KR5 汇报 | M5 报告产出 | 实验记录摘要 + 成本分布 + 三特性判定 → REPORT.md | 证据集 → REPORT.md |

pipeline：M1 → M2 → M3（含 M4 伴随采集）→ M5。

**OKR 充分性自检**：KR1-KR3 保证「2 迭代真跑发生」；KR4 保证「三特性有证据」；KR5 保证「汇报成形」。三特性验证全部依赖真跑产生的可观测面（archive 字段/事件流/锁文件/实验记录）——联合达成即可推出 O。无缺 KR；切分维度（准备→运行→证据→汇报）无重叠。✓

### L1 调研消解

本层事实全部自查消解 → 简报：`research-L1.md`（三特性代码位置 / env_9 资产清单 / 机制事实 10 条 / 不可消解问题 7 项转 L2）。

---

## 第 2 层（各模块判断节点）

### M1 资产搬运

- 事实（已消解）：initSuperWorkspace 不拷 env；gates 固定 `<ws>/gates`；expand_cell 会写 env/feature_space.json（原位引用会污染已收档的 env_9）。
- **J1 复用方式**：拷贝 env_9/env → env_10/env + 拷贝 env_9/gates → env_10/gates（推荐）｜原位引用 env_9/env + 仅拷 gates。
- **J2 .venv 处理**：随 env 整拷 + numpy 冒烟，失败重建（推荐）｜不拷 venv 用系统 python（需系统有 numpy，改变 env_9 验证过的执行口径）｜重建 venv。

### M3 启动与运行

- 事实（已消解）：adjust_interval=2 时 2 迭代零 adjust；=1 时恰一次（iter1 边界启动、收尾 join）；adjust 成本进角色级不进迭代级；pi_path 必须预写 super.json；单迭代 ~30-50 分钟。
- **J3 archive 初态**：全新空 archive（推荐）｜继承 env_9 archive（绝对路径指向 env_9/codes、权重已塑形、验证语义混淆）。
- **J4 运行参数**：--max-iterations 2 --adjust-interval 1 --max-wall-time 4 --daemon + 预写 pi_path（推荐）｜adjust-interval 2（零 adjust）｜不设墙钟（默认 24h）。
  - 知会项（非判断）：super.json config.pi_path 写入 env_9 同款路径（技术必要性，运行元信息允许写）。

### M4 证据收集（三特性验证判据）

- **J5 格锁验证方式**：被动观察 + 预放一个死 pid 孤儿锁 fixture 验证启动清理（推荐）｜纯被动｜更激进注入（TTL 陈旧锁/并发争锁——不推荐）。
- **J6 验证判据表**（human 确认）：
  1. **五要素 通过** ⇔ ≥1 次 elite_admitted 后，该格含 `weight`、`elite.score/perf_value`、`elite_code_path`（指向存在文件）、`elite_record_path`（指向存在文件）；`description` 字段存在（内容视 adjust 是否执行 modify_cell/new_cell，单独标注）；name 同理。任何字段缺失/路径失效 = 失败。
  2. **成本归因 通过** ⇔ 每个有效迭代的 `iteration_ended.by_role` 非空且 ≥2 角色（builder/coding[/debug]）tokens>0；adjust 的 agent_ended.usage 六字段齐全（角色级）；cost_report.py 跑通出 role 表 + 迭代表；抽 1 迭代与 agent_ended.usage 对账一致（±舍入）。
  3. **格锁 通过** ⇔ 迭代期间 locks/cell_*.lock 出现、迭代结束被释放（目录无残留活锁）；启动孤儿清理事件出现（fixture 生效时）；无 cell_locked_skip/cell_lock_overflow 死等现象。
  4. **实验记录链 通过** ⇔ 每迭代 experiment_record.md 存在、六段齐全、coding 追加段在、iteration_ended.record_updated=true。
  5. **成本分布汇报** ⇔ by_role 汇总表 + 角色级汇总 + Top 昂贵迭代 + cache 呈现于 REPORT.md。

### M5 汇报

- **J7 汇报落点**：REPORT.md 落盘（env_10/REPORT.md）+ 会话摘要（推荐）｜仅会话内呈现。

### L2 调研消解

本层判断节点全部由 L1 简报支撑（research-L1.md §4 机制事实），无新增不可自查消解问题 → **本层无需新 research 简报**（判断节点裁决见 L3 提问记录）。

---

## 第 3 层（叶子层：执行细节——待 L3 裁决后固化）

> 状态：占位——J1-J7 裁决后把四维度（代码/文件/工具/环境配置）落到本层并标记设计树完成。

- 复用搬运命令清单（cp -R / 冒烟命令）
- super.json 预写字段（pi_path）
- 启动命令：`rick super start --name env_10 --env <abs env_10/env> --daemon --max-iterations 2 --adjust-interval 1 --max-wall-time 4`
- 孤儿锁 fixture（如 J5 裁决采纳）
- 观察/验证命令清单：rick super status --json / events 尾读 / cost_report.py / adjust_trace.py（可选）
- 汇报模板（REPORT.md 骨架）

---

## 提问记录（L3 批量追问——human 裁决）

| # | 问题 | 推荐 | human 裁决 |
|---|---|---|---|
| Q1 | env 复用方式：整拷 env_9/env+gates 到 env_10，还是原位引用？ | 整拷（隔离已收档 env_9、本会话约定产出落 env_10/env、防 expand_cell 写污染） | **待裁决** |
| Q2 | .venv：随拷 + 冒烟（失败重建）？ | 随拷+冒烟 | **待裁决** |
| Q3 | archive 初态：全新空档案？ | 全新（看五要素从空格到 admit 全链路；继承会带入 env_9 绝对路径与塑形权重） | **待裁决** |
| Q4 | 运行参数：--max-iterations 2 --adjust-interval 1 --max-wall-time 4 --daemon + 预写 pi_path？ | 采纳（interval=1 保一次 adjust；4h 墙钟富余；已知 adjust 成本为角色级口径，报告分两口径呈现） | **待裁决** |
| Q5 | 格锁验证：被动 + 死 pid 孤儿锁 fixture？ | 采纳（fixture 只触 release_orphans 路径，安全可逆） | **待裁决** |
| Q6 | 验证判据表（5 条）确认？ | 按树中所列 | **待裁决** |
| Q7 | 汇报落点：REPORT.md + 会话摘要？ | 采纳 | **待裁决** |
