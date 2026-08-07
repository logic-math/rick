# RFC — 升级 human-loop 使其更具批判性

## 主题
升级 human-loop 使其更具批判性

## 完成日期
2026-08-02

## 哲学基础
AI 作为人类智能放大器——对人类心智的扩展，通过对 research(调查) / think(质疑) / exporter(表达) / sense(调度/复核) 四能力的抽象实现智能扩展

## 主要矛盾
矛盾 1：现有协议骨架（7 步线性推进 + 单 sense subagent）↔ V11 全新设计重写（sense 升级为 main agent + 四 agent 架构 + research loop 独立）

## 控制手段
1A 完全重写：四文件架构替代现有 sense_subagent.md + human_loop.md

## 四文件架构（go embed 内嵌模板，存于 rick 源码仓库）

### sense_loop.md
- 职责：sense 子步骤推进 + 简报落盘逻辑 + 前序历史加载 + 派发/核验循环逻辑
- 形式：markdown + ASCII 流程图
- 派发要素：子步骤 + 主题 + 草稿路径 + 前序判断 + 任务派发 + 结果核验（保持现有 4 + 补充 2）
- 核验循环：升级派发 + 最大重试 5 次 + human 介入

### think.md
- 职责：分析假设 → 打分 → 选最高风险（pipeline 化）
- 哲学重构：删除显式"价值性假设生成"职责（隐含生成保留）
- think 工作 = 选择导致失败的最高风险

### research.md
- 职责：基于事实调研树遍历，不断澄清所有事实模糊性，尽调到极限（= 事实模糊性为 0）
- 新协议规则 R7：无法被调研的事实属事实性假设，需 research 上报 + sense 呈现 + human 决策

### exporter.md
- 职责：基于某种格式把思想汇报出来
- 自行设定沉淀结果 + 先向 human 确认大纲后执行（默认 SENSE 思考方式表达）

## sense agent 角色定义
- sense = 调度能力 = main agent 复核层具象化
- 二分职责：派发层（提供上下文 + 描述目标 + 描述交付标准） + 复核层（基于交付标准检查 + 升级派发 + 最大重试 5 次 + human 介入）
- 批判门禁由 sense agent 自己执行（subagent 执行 + sense 检查）

## 三层复核机制
- 第一层：research agent（事实性假设澄清）
- 第二层：main agent（通用假设澄清确认，二层门禁拦截）
- 第三层：human（最终确认）

## 阶段门禁与整体结束
- 阶段门禁推进条件：该阶段所有假设（事实性 + 价值性）被澄清
- 整体结束条件：良质跃迁 + 所有假设澄清 + human 确认 exporter 形式与大纲（F1 修正）

## 哲学重构（H5 系列）
1. 删除原则：think 不再有"价值性假设生成"显式职责（隐含生成保留）
2. 更根本性的启发：寻找最高风险假设 = 澄清所有价值性 + 事实性假设（深化 S1 B）
3. 二分统一：事实性 vs 价值性不再区分，无第三类
4. 无法被调研的事实属事实性假设，需 research 上报 + sense 呈现 + human 决策（新协议规则 R7）

## 派生修订需求
- D-R1：S1 B 措辞从"标准"修订为"启发方法"
- D-R2：sense_subagent.md 假设澄清引擎循环重构（按可调研性分流）
- D-R3：think agent 职责描述重写（显式职责删除 + 隐含行为保留）
- D-R4：新协议规则 R7 落地

## 简报与判断记录
- 悬置项规则 R1：AI 草案仅列客观矛盾点不带推理/建议方向，human 原创问题描述并确认，不可证伪的理由不能悬置
- 简报格式 R5：事实陈列在前 + 提问结尾引导 human 判断
- judgment.md 仅记 human 原话，禁止 AI 推理

## 架构 RFC 剥离
架构 RFC（含 sense/research/think/exporter 四 agent 详细设计）剥离至 draft/rfc/ 独立路径，sense_loop.md 等协议文档内仅留一句话指针

## 遗留逻辑漏洞（待落地时澄清）
- L1 事实模糊性归零的可判定性
- L2 良质跃迁的客观判定
- L3 sense 与 think 边界（X5 延续）
- L4 go embed 与现有两文件的替换路径
- L5 派生修订 D-R1~D-R4 落地载体
- L6 核验循环"最大重试 5 次"的依据

## SENSE 流程记录
- S1 Subject 还原：已通过（5 次门禁）
- S2 Subject 假设枚举：已通过（V1-V15 全部成立，含 V11 全新设计）
- E1 Perspective 概念地图：已通过
- E2 Perspective 视角选择：已通过（视角E 架构论 + 哲学基础）
- N Judgment 主要矛盾：已通过（矛盾 1 + 1A 完全重写 + 四文件架构）
- EC Critique：已通过（28 项假设成立 + 维持跃迁选项）
