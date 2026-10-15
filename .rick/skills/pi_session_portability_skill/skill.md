# skill:pi-session-portability（pi 会话跨路径/跨机移植）

## 触发场景

当需要手工搬迁或修复 pi 会话历史，或排查「迁移后 resume 找不到会话 / 历史丢失」时使用。信号词：

- `--continue` / `--session` / `--session-id` 在新路径下找不到会话
- 报 `Stored session working directory does not exist`（exit 1）
- resume 成功但历史是空的（静默新建了会话）
- 需要理解 `rick tools dump/import` 对会话目录做了什么

## 预期效果

- 最小改写集（两步）做完后，pi 在新路径完整加载历史（job_46 实测：9962 input tokens 历史加载 + 模型续答）
- 不做无谓重写：消息 content 文本里的旧路径**不需要清洗**（resume 不解析）

## 核心内容

**转义规则**（pi session-manager.js:242-246，实测锁定）：会话目录名 = `--` + cwd 去首 `/` 后把 `/`、`\`、`:` 替换为 `-` + `--`；**下划线不替换**，无截断无冲突处理。
例：`/workdir/sunquan20/AI_CODING/rick` → `--workdir-sunquan20-AI_CODING-rick--`。目录内布局：`<ISO时间戳>_<sessionId>.jsonl` 平铺，无索引文件无全局缓存。

**最小改写集**（12 组隔离实验矩阵实测，缺一即失败）：

1. `sessions/` 下目录改名到**新 cwd 的转义名**（所有恢复方式必需）
2. 会话文件**首行** JSON 的 `cwd` 字段重写为新绝对路径（`--session`/`--session-id` 必需；仅 `--continue` 可省——它只看目录不看 header）

```bash
# 首行重写（其余行逐字节保留）：
python3 -c "
import json
f='<session>.jsonl'; new='<new-cwd>'
ls=open(f).readlines(); h=json.loads(ls[0]); h['cwd']=new
ls[0]=json.dumps(h,ensure_ascii=False)+'\n'; open(f,'w').writelines(ls)"
```

只做 ① 不做 ②：报 `Stored session working directory does not exist` exit 1。都不做：`--session-id` 静默丢历史（新建空会话）。fork 会话的可选第三步：首行 `parentSession` 前缀重写。

**resume 语义**（决定哪步不能省）：

| 方式 | 定位 | 失败行为 |
|---|---|---|
| `--continue` | 当前 cwd 转义目录内 mtime 最新，不校验 header.cwd | 找不到就新建 |
| `--session <id>` | 本地精确/前缀 → 全局扫描兜底；全局命中且 cwd 不同 → fork 交互确认 | 非交互（stdin 关）exit 0 零输出 = 自动化死路 |
| `--session-id <id>` | **仅本地精确匹配，无全局兜底** | 未找到 → 警告 + 新建空会话（历史静默丢失） |

rick 的两条恢复链：job 链用 `--session-id`（doing.go）、web 恢复链用 `--session`（sessions.go）——所以 import 移植必须 ①+② 全做。

**关系**：`rick tools import` 已自动完成这两步（含子工作区 cwd 的重转义）；本技能用于手工搬迁、排障、以及理解 dump/import 的会话移植行为。排障时先对照「转义名算对了吗（下划线陷阱）→ header.cwd 改了吗 → 用的是哪种 resume 方式」三问。
