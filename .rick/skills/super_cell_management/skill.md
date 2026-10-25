# skill:super-cell-management（adjust 第三/四步：特征格动作手续）

> 服务对象：**adjust agent**
> 加载方式：adjust 提示词声明路径 + 触发条件；SENSE 与 research 完成后 `read` 本文件

## 触发场景

`super_sense` 判定了主要矛盾与突破方向、且 research（按 `research_brief` 检索）找到启发式 idea 后加载。此时要把「想法」变成**合法的格动作**：

- 新方向从未被覆盖 → `new_cell`
- 已有格的方法描述不够准确/不够前瞻 → `modify_cell`
- 某方向值得更多/更少预算 → `soft_weight`
- 整个空间已榨干 → `terminate_recommend`

## 预期效果

一组合法、可审计的格动作提案（JSON proposals），每个动作都带**依据**（来自哪个矛盾/哪篇论文/哪条实验证据）；动作被框架应用后，特征空间朝突破方向演进，采样预算随之前移。

## 核心内容

### 特征格四要素（数据结构）

```json
{"cell_id": "axis1=v1|axis2=v2",
 "description": {"name": "短名", "method": "方法描述（可引论文/调研地址）", "source_refs": ["arxiv:2108.02025"]},
 "weight": 1.0,   // 相对采样份额（0 = 功能性删除）
 "score": 0.93,   // 该格精英得分（框架写入，勿手改）
 "visit": 3, "elite": {...}}
```

四个要素的分工：**名字**（人读标识）/ **方法描述**（builder 的设计方向指引）/ **权重**（资源投入）/ **得分**（框架评估结果）。

### 动作清单与手续

#### 1. `new_cell`（新建格——拓宽探索）

```json
{"type": "new_cell", "action": "create",
 "descriptor": {"impl_strategy": "strassen_split", "memory_layout": "standard"},
 "name": "Strassen 分治",
 "method": "Strassen 7 乘法分治（2 层递归后接 BLAS 块乘）；参考 arXiv:xxxx 的 split 实现；重点观察数值稳定性与递归阈值",
 "source_refs": ["arxiv:xxxx", "research:iter12 调研报告"],
 "orthogonality_rationale": "现有格全部属于「一次整阵 BLAS 派发」族（impl_strategy ∈ {blocked, rowwise, full_blas}），本格引入「算法级分治」维度——乘法次数 O(n^2.81) 而非 O(n^3)，与现有格的优化维度不重叠"}
```

- **正交性判据（硬性）**：与**所有现有格**相比，至少一个优化维度是新的（新轴值 / 新描述维度）。若只是现有格的同义改写或参数微调 → 改为 `modify_cell` 或直接设计实验
- `method` 必填且要**可指导设计**（不是标签）；`source_refs` 建议带论文/调研地址，builder 会据此做定向调研
- 框架行为：走裁决链（proposals/）批准后 → `env/feature_space.json` 热重载加格（下一迭代分殖优先探索）+ archive 建格（带 description）+ `cell_created` 事件

#### 2. `modify_cell`（改进格描述——让方向更准）

```json
{"type": "modify_cell", "action": "improve_description",
 "target_cell": "impl_strategy=blocked_loops|memory_layout=tiled",
 "method": "分块 + 转置融合：内层 microkernel 用 B 的列段（转置后连续），参 arXiv:1609.00076 BLISlab 的 MC×NC×KC 分块结构；目标是装配与计算融合",
 "source_refs": ["arxiv:1609.00076", "research:iter15 调研报告"],
 "rationale": "该格连续 3 轮在同一分块参数上微调无新纪录；方法描述过粗（只说「分块」），builder 无法看出可尝试的结构性变化"}
```

- 用途：research 找到更好的实现路线后，把格的方法描述**细化**（把「分块」升级为「分块+转置融合，含 microkernel 结构」）——这是「改进特征格」的主手段，比新建格更轻
- 框架行为：走裁决链 → `archive.set_description` + `cell_modified` 事件

#### 3. `soft_weight`（权重调控——资源投入）

```json
{"type": "soft_weight", "target_cell": "...", "action": "weight=2.0",
 "evidence": "seq 143：该格连续 2 轮无采样但精英 perf 全场最优，预算不足", "rule_id": "potential_underexplored"}
```

- 语义：**相对采样份额**（概率 ∝ weight）；`weight=0` = **功能性删除**（格与谱系保留、可复活——这是合法的「删除」语义，**不删格**）
- 降权需证据（连续失败 / 长期无新纪录）；提权需证据（有潜力但采样不足）
- 框架行为：立即生效（`archive.json` weight + `weight_changed` 事件）

#### 4. `terminate_recommend`（认输建议，建议性非执行）

```json
{"type": "terminate_recommend", "evidence": "连续 6 迭代无新纪录，全部存活格已触及 env 声明的性能天花板", "rationale": "建议人工决定收手或换轴"}
```

- 不进裁决自动执行；写入 proposals/（`needs_human: true`）+ `proposal` 事件，供中控会话/人决策

### 落盘与决策记录

- 每个动作连同「来自哪个矛盾 / 哪个研究方向 / 哪篇论文」写进决策记录（ctx 的 `decisions_path`）：
  ```markdown
  ## 三、证据（research）
  - 主要矛盾指向 <…>；检索：<arxiv id + insight + 如何映射到格动作>
  ## 四、动作清单
  1. new_cell <name>：<依据>；正交性：<…>
  2. soft_weight <cell> 1.0→0.5：<证据 seq…>
  ```
- 空提案列表是合法输出（「本轮无需调整」），但必须在决策记录里说明理由

### 纪律

- **宁缺毋滥**：格数保持精而少（个位数到十余）——每加一格都稀释其余格的采样份额
- **正交性论证缺失的 `new_cell` 视为无效**（裁决会拒，且浪费一轮）
- **不删格**：淘汰只用 `weight=0`
- **单轮动作有界**：权重变动 ≤5 个格、new_cell ≤1、modify_cell ≤2（防单轮大改导致归因失效）
- 每个动作必须能回答「依据是什么（证据/论文）」——无依据的动作会在决策记录里标记为 `unsubstantiated`
