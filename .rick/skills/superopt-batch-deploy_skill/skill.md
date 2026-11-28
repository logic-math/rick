---
name: superopt-batch-deploy
description: superopt 批量提案的实施-部署模式——human 批准 N 项提案后的「批合并实施→全量测试→commit→release→章法重启兑付」一条链，避免逐项部署的重复重启成本
execution_plane: local
---

# skill:superopt-batch-deploy

## 触发场景

human 批准 superopt 的多项提案（humans 表 proposal 队列批量 applied）后需要实施部署时使用。信号：连续批准 ≥2 项、或 humans 队列积压多项同域提案（restart/verifier 类为主）。

## 预期效果

- 批量实施一次部署全部生效（边际成本 ≈ 0——一次重启兑付整批）
- 部署链完整（测试→commit→release→章法重启→新代码验证）
- py 层与 Go 层的部署形态分清（py 源码即运行时 vs Go 需重打二进制）

## 核心内容

### 批合并实施链（job_56 实证：24 项提案分 3 批）

```
①标 applied：批量 UPDATE humans + mark_proposal_applied（闭环事件）
②worker 批实施：单 worker 串行做整批（读 humans 表提案全文 → 逐项实施 →
  每项验证 import → 全量 unittest 绿；#33 类含越权风险的提案按裁决裁剪）
③测试全绿：timeout 560 python3 -m unittest discover -s super/tests -p 'test_*.py'
④git commit（批描述：逐项列出）
⑤部署分层：
  py 层（loop/agents/sampler 等）：源码即运行时——commit 即部署源，
    下次 super start 自动加载（不需要重打二进制）
  Go 层（internal/）：go build -o bin/rick + release（--version 语义化版本
    可选）+ releases/current 更新——release 命令一条链（门禁+构建+换链+
    生产重启）
⑥章法重启兑付：python3 super/tools/watchdog.py <ws> --restart
  [--max-iterations N --max-wall-time H]（混战清理+config 体检+工作量
  匹配估算+带旗标启动）
⑦生效验证：新事件类型按场景触发观察（如 admission_review/experiment_resumed
  ——零触发=场景未现属正常待命）
```

### 部署形态速查

| 改动层 | 部署方式 | 生效时机 |
|---|---|---|
| py 源码（super/superopt/） | git commit 即部署源（源码即运行时） | 下次 super start |
| prompt 模板（super/prompts/） | 热读（每轮派发时读文件） | 下一轮派发 |
| Go 二进制 | go build + release + current 换链 | 生产重启时 |
| ~/.rick 安装快照 | release 联动部署（自动） | 与 Go 二进制同版 |
| env 资产（WS/env/） | 文件即改即生效（热读或重启） | 即时或下轮 |

### 红线

1. 单写者：批实施用一个 worker 串行（loop.py 等同域文件不并行写）
2. 全量测试绿才部署（攒批的 pre-existing 失败要先修——stash 对照法区分新旧失配）
3. 越权裁剪：自动裁决器类提案（hard 提案 human 裁决权不可代行）按裁决裁剪实施
4. 生效验证是「按场景触发」观察——不要为验证而造场景
