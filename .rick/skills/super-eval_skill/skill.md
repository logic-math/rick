---
name: super-eval
description: super mod 进化系统的统一质量评估——ESQ 四维度指标体系（搜索质量/自适应/知识/稳定性），用于跨版本对比自进化效果
execution_plane: local
---

# skill:super-eval

## 触发场景

需要评估 super mod 进化系统的自进化是否有效时使用——产出可跨版本比较的统一 ESQ 分数。

（激活词：super 评估、ESQ、进化效果、系统质量、super-eval、分数对比）

## 核心命令

```bash
# Markdown 报告（人类可读）
python3 <repo>/super/tools/esq_report.py <workspace>

# JSON 输出（机器可比）
python3 <repo>/super/tools/esq_report.py <workspace> --json
```

## ESQ 四维度指标体系

```
ESQ = D1^0.35 × D2^0.20 × D3^0.15 × D4^0.30
```

| 维度 | 权重 | 子指标 | 数据来源 |
|---|---|---|---|
| **D1 搜索质量** | 0.35 | Coverage（非空格/总格）· Speedup geomean（封顶 2×）· Sample Efficiency（admitted/valid）· Convergence Efficiency（最近 20 实验新格发现率） | super.db cells/experiments |
| **D2 自适应能力** | 0.20 | Decision Accuracy（升权格产出率——v2 修正：排除 infra 吃掉 + unvisited）· Self Correction（主动回调率）· Budget Utilization（预算利用率） | weight_changed / budget_adjusted events |
| **D3 知识积累** | 0.15 | Knowledge Growth（domain_knowledge 字节，封顶 50KB）· Transfer Rate（知识沉淀节/adjust 轮次）· Proposal Consumption（提案消费率） | domain_knowledge.md / humans 表 |
| **D4 长跑稳定性** | 0.30 | Monotonic Improvement · Infra Fail Rate · Recovery · Token Efficiency（avg token/exp 归一化）· Revisit Improvement（同格重访改进率）· Resource Utilization（GPU 计算/wall 时间比） | experiments + events |

## 维度间的 geomean 逻辑

任一维度为 0 → ESQ 为 0（**不偏科**）——与 KernelBench 的 Correct Speedup geomean 同理：
一个进化系统如果搜索好但知识积累为零（不学习），或搜索好但不稳定（长跑退化），都不能算高质量。

## 跨版本对比方法

```bash
# 版本 A（改前）
python3 super/tools/esq_report.py <ws_A> --json > esq_v1.json

# 版本 B（改后）
python3 super/tools/esq_report.py <ws_B> --json > esq_v2.json

# 对比（改了什么→哪个维度变了→ESQ 变了多少）
python3 -c "
import json
a = json.load(open('esq_v1.json'))
b = json.load(open('esq_v2.json'))
print(f'ESQ: {a[\"ESQ\"]} → {b[\"ESQ\"]} ({b[\"ESQ\"]-a[\"ESQ\"]:+.4f})')
for d in ('D1_search','D2_adaptivity','D3_knowledge','D4_stability'):
    dv = b['dimensions'][d] - a['dimensions'][d]
    print(f'  {d}: {a[\"dimensions\"][d]:.4f} → {b[\"dimensions\"][d]:.4f} ({dv:+.4f})')
"
```

## 竞品对比口径

| 对比对象 | 怎么比 |
|---|---|
| **One-shot LLM**（GPT-5.5 等） | 只比 D1（Coverage + Speedup）——架构增益 = 进化系统 D1 / one-shot D1 |
| **KernelFoundry**（唯一公开进化系统） | D1 同口径（KernelBench 子集）+ D2/D3/D4 为我们的架构优势 |
| **Best-of-N 采样** | 同预算下 D1 对比——进化系统的多步修正 vs 随机采样 |
| **跨版本 super**（自己改自己） | 全四维度对比——D2/D3/D4 是自进化效果的直接度量 |

## 输出格式

每次运行输出完整报告（含四维度分数+短板分析+竞品口径），落盘到 `<ws>/esq_report_<timestamp>.md` 留档。

## 设计原则

1. **所有指标从 super.db 自动计算**——无人值守自动出报告
2. **geomean 不偏科**——任一维度为零则总分零
3. **归一化封顶**——speedup 封顶 2×/knowledge 封顶 50KB（避免单一维度 outlier 淹没其他）
4. **权重可调**——D1 0.35 / D4 0.30 > D2 0.20 > D3 0.15（搜索和稳定性是生命线，自适应和知识是长程增益）
