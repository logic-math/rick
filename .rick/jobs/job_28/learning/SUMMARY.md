APPROVED: true

# Job job_28 执行总结

## 执行概述

**项目目标**: 实现 RFC"升级 human-loop 使其更具批判性",重构 human-loop 协议从 v2(7 步线性+单 sense_subagent)到 v3.1(5 阶段非线性+四文件架构+批判门禁嵌入+反向回流)。

**实际完成**: 11 commits,从 v2.10.9 升级到 v2.11.9,完整落地:
- sense_loop v3(5 阶段+反向回流+系统论描述符)
- think v3.1(推理驱动假设+4维打分+3启发性问题+假设数量保障)
- research v2(尽调树+信源加权+subagent 上下文隔离)
- exporter(大纲+内容两阶段)
- 简化产物(4→2)+ 配置化所有阈值
- wiki 文档(human-loop.md)

**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **四文件架构替代两文件**:sense_loop/think/research/exporter 替代 human_loop/sense_subagent,职责清晰,main agent 协议(sense_loop)与 subagent 协议分离。

2. **5 阶段非线性流程**:S/E/N/S-R/EC 替代 v2 7 步线性(S1/S2/E1/E2/N/S-R/EC),合并相关阶段(S1+S2→S, E1+E2→E),拆分复杂阶段(N→N1+N2),引入反向回流机制(后续可重启前序)。

3. **批判门禁嵌入各阶段**:think 不再独立步骤,嵌入各阶段(human 实质性回答后触发),提升门禁频率,避免门禁沦为独立步骤。

4. **系统论描述符(N1 阶段)**:5 要素 node/input/output/inner/edge 替代模糊的概念地图,结构化分析系统,推演稳态迁移。

5. **EC human 自判**:不替 human 提议跃迁方向,AI 只呈现回顾,human 自判降维/升维/维持,强化"不替 human 判断"原则。

6. **假设数量保障 + 3 启发性问题**:
   - 最低假设数 `think_min_assumptions` 默认 5,多视角强制,补强流程(反事实/边缘/隐含)
   - 每假设 3 启发性问题(信念/前提/反例),总提问 ≥ 15 问题
   - 解决"think 提问偏少"+"启发性不足"两大问题

7. **尽调树 + 信源加权**:research v2 用 MECE 树+加权置信度替代 v1 BFS+全量收集,终止条件明确(所有叶节点置信度高)。

8. **配置化所有阈值**:5+ 配置项(max_retries/sense_max_backflows/think_top_n/think_min_assumptions/research_source_weights),适应不同场景。

9. **wiki 文档**:新建 wiki/modules/human-loop.md(230 行),完整描述 v3.1 设计。

## 问题与教训

### 问题1: think.md 漏 commit

**根本原因**: 首次 commit 时 git add 命令参数列表中 think.md 在最后,可能某种原因未被 add 成功(具体原因未深查)。
**解决方案**: 后续 commit 63e9957 补充 think.md。
**经验教训**: 大改动 commit 后必须 `git status` 验证无遗漏文件,不能只看 commit 成功消息。

### 问题2: ASCII 文本绘图约束被回退

**根本原因**: 4119da4 在 N1 阶段强制 ASCII 文本绘图,但 briefs/N1.md 已含详细描述,提示词中强制格式反而限制 LLM 自由发挥。
**解决方案**: 用户回退(6ed89b0),恢复为"模型自由发挥"。
**经验教训**: 提示词不应过度约束格式,尤其是已有详细产物落盘的情况下;让 LLM 自由发挥更高效。

### 问题3: 测试断言与模板措辞不一致

**根本原因**: 测试断言"判断目标"但 v3 改为"启发性 3 问"后该词不存在;测试断言"Subject/Perspective/Judgment/Reverse/Critique"但 v3 用中文名。
**解决方案**: 测试断言同步更新为中文阶段名 + 删除过时英文词。
**经验教训**: 改模板措辞时必须同步改测试断言,否则测试失败;测试断言应基于"核心约束"而非"具体表述"(允许 LLM 自由发挥)。

### 问题4: macOS Gatekeeper 静默 SIGKILL

**根本原因**: 安装到 ~/.rick/bin/rick 的二进制被 macOS 静默 SIGKILL(exit 137),即使无 quarantine xattr。
**解决方案**: `codesign --force --sign -` 重新签名(adhoc)。
**经验教训**: macOS 对未签名二进制在某些目录下会静默 kill,install.sh 应加 codesign 步骤。

## 知识沉淀清单

- [x] skills/multi_phase_protocol_skill/skill.md - 多阶段协议设计方法论(5 阶段+反向回流+批判门禁嵌入+系统论描述符)
- [x] loops/protocol-redesign-loop.md - 协议重构循环(7 Step 流程,从分析痛点到反思)
- [x] domain/architecture.md 更新 - human-loop 模块段从 v2(job_26 时代)更新到 v3.1
- [x] wiki/modules/human-loop.md - 完整 v3.1 设计文档(230 行)

## 遗留观察项

1. ⚠️ 反向回流机制实战验证:允许 3 次回流,但复杂场景可能不够,需观察
2. ⚠️ 假设数量保障的"补强"质量:LLM 可能凑数,需观察假设质量
3. ⚠️ 启发性提问的 LLM 执行度:LLM 是否能稳定生成启发性问题(而非退化为确认性),需观察
4. ⚠️ 信源权重的场景适应性:默认 0.4/0.3/0.2/0.1 是否适用所有场景,需观察
5. ⚠️ 简化产物(2 产物)对 learning 阶段的影响:概念地图来源切断,需观察 learning 复盘质量

## RFC 决策落地核对

| RFC 项 | 落地状态 |
|---|---|
| 1A 完全重写:四文件架构 | ✅ sense_loop/think/research/exporter |
| sense_loop = main agent 协议 | ✅ |
| think = 推理驱动假设+4维打分 | ✅ |
| research = 尽调树+信源加权 | ✅ |
| exporter = 大纲+内容两阶段 | ✅ |
| 5 阶段(S/E/N/S-R/EC) | ✅ |
| N 拆 N1+N2 | ✅ |
| 反向回流机制 | ✅(上限 3) |
| 批判门禁嵌入 | ✅ |
| EC human 自判 | ✅ |
| 系统论描述符 | ✅(5 要素) |
| 简化产物(2 产物) | ✅ |
| 配置化所有阈值 | ✅(5+ 配置项) |
| 假设数量保障 | ✅(min_assumptions 默认 5) |
| 3 启发性问题 | ✅(信念/前提/反例) |
| wiki 文档 | ✅(human-loop.md) |

## 版本

- 起始: v2.10.9
- 结束: v2.11.9(minor +1)
- 已 push 远程(origin/main)
