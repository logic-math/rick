# skill:runtime-hook-verify（外部运行时 hook 三关验证协议）

## 触发场景

当你为**外部运行时**（pi extension / 任何 hook / 进程间协议 / 事件订阅）写集成代码时使用。信号词：写 `index.ts` 扩展、`on("agent_*")` 事件 handler、`sendUserMessage`/跨进程消息、自定义 hook、「写完部署了应该能用」。

**为什么需要**：job_47 实证（commit 9667d13）：pi hook「写完+部署+提交」后从未真实跑过——human 质询后补测，一次真实进程暴露 3 个 bug（导出形态错 / API 属性不存在 / hook 点撞运行时限制）。**「client 层能想到的兜底」和「运行时真实行为」之间隔着一层没有验证的想象**。

## 预期效果

- 外部运行时集成提交前 100% 过三关，杜绝「提交后才发现在想象中编程」
- 每个 bug 在最小验证内暴露（加载错误当场报、行为断言看事件流），不需要等真实事故
- 修一次就定位到**官方模式**（读运行时源码找设计内路径），而不是试错猜 API 组合

## 核心内容

### 前置：写 hook 前先读运行时源码（比读 API 表面文档更快更稳）

目标运行时的核心循环源码里通常写着**官方模式**。job_47 案例：pi 的 `agent-loop.js` 主循环结尾处

```js
// Agent would stop here. Check for follow-up messages.
const followUpMessages = (await config.getFollowUpMessages?.()) || [];
if (followUpMessages.length > 0) { pendingMessages = followUpMessages; continue; }
```

以及 `_handlePostAgentRun` 的注释：「Any messages here were queued by **agent_end** extension handlers and need a continuation.」——这两处直接说明了「agent_end handler 里 sendUserMessage + deliverAs:"followUp"」是设计内路径。**从 API 类型定义表面猜组合（如选了 agent_settled）会踩到重入限制**——而限制写在错误信息和源码注释里，读一遍就能避开。

### 三关协议（顺序不可换，任何一关失败即停下修复）

#### 第 1 关：静态关——对照目标运行时类型定义逐条核

对你写的每个 API 调用，去运行时的 `.d.ts` / 类型定义里**逐条**核对：

| 你写的 | 核什么 | job_47 反例 |
|---|---|---|
| 事件 handler 签名 | `ExtensionHandler<E>` 的实际参数 `(event, ctx)`，事件 payload 字段是否真的有 | —— |
| 对象属性 | `api.session?.messages` —— `ExtensionContext` 上**有没有**这个属性（答案：没有；消息要走 message_end 事件或 sessionManager.getEntries） | ❌ 首版就写错了 |
| 函数存在性 + 调用位置 | `sendUserMessage` 在 `ExtensionAPI` 上存在，但 `ExtensionContext`（事件 handler 的 ctx）上没有——closure 捕获 `api` 才能用 | —— |
| 导出形态 | 运行时 loader 期望 `export default async function`（查 `rick-gates/index.ts` 等已知能跑的样例） | ❌ 首版 `export function` 加载即报 "does not export a valid factory function" |
| 死代码 | 防误伤/激活条件函数**真的被调用了吗** | ❌ 首版 `isSuperoptAgent` 定义了但从未调用——会对所有会话生效 |

**最低标准：每个用到的 API 都能在类型文件或已验证样例里指出对应行。**

#### 第 2 关：加载关——真实进程跑一次看加载日志

不要只写文件+部署。起一个**隔离的真实进程**（不碰生产环境）：

```bash
# 隔离 agent home：复制 auth/models，只放待测扩展
rm -rf /tmp/<name>-test && mkdir -p /tmp/<name>-test/extensions/<your-ext>
cp <auth.json models.json settings.json> /tmp/<name>-test/
cp <your-ext>/index.ts /tmp/<name>-test/extensions/<your-ext>/

# 真实进程 + 事件流模式（json 模式能看到完整时间序）
cd /tmp/<name>-test/work && \
  <必要激活 env> PI_CODING_AGENT_DIR=/tmp/<name>-test \
  <runtime 二进制> --mode json --append-system-prompt <系统提示词> -p '<测试消息>' \
  > run.jsonl 2> run.err; echo "exit=$?"
grep -iE "error|fail" run.err   # 加载错误/Extension error 都在这里当场暴露
```

典型当场暴露的 bug：导出形态错（`Failed to load extension ... does not export a valid factory function`）、类型 import 路径错、handler 抛异常（`Extension error (<path>): ...`）。

**隔离要点**：环境变量门控（如 `SUPEROPT_JSON_GUARD=1`）先验证「无 env 的普通进程完全不受影响」——防误伤验证与功能验证同样重要。

#### 第 3 关：行为关——事件流时间序断言修复路径真的触发

对修复类 hook（「检测 X → 触发 Y」），用事件流证明 Y 真的发生了，且**时间序**正确。job_47 的黄金标准（run5.jsonl）：

```
turn_start
user       '香蕉的英文单词是什么？'
assistant  '香蕉的英文单词是 **banana**。'     ← 无 JSON（触发条件）
agent_end                                        ← hook 检测点
turn_start                                       ← 修复轮自动开始（同进程）
user       '你的最终回复缺少必需的JSON...（修复第 1/2 次）'  ← 注入证据
assistant  '```json\n{"answer": "banana"}\n```'  ← 修复生效
agent_end                                        ← 二次检测：合规 → 通过
agent_settled                                    ← 最终 settle（证明进程没提前退出）
```

**三个断言缺一不可**：①触发条件后出现**注入的修复消息**（不是原消息重放）②修复轮的 assistant 回复真的改变行为 ③**终止事件**（settled/exit）在修复轮之后——证明主流程等待了嵌套/续跑逻辑，没有竞态提前退出。

边界情况同法验证：**上限防死循环**（顽固不合规 agent → 修复注入恰好 N 次后放行）+ **门控隔离**（无激活条件的进程单轮不受影响）。

### 提交纪律

- 三关证据（run*.jsonl 事件流 + exit 码 + stderr 干净）在提交信息里**引用具体文件**，不写「已验证」空话
- 调试期可以往 handler 里加 stderr 日志（`console.error('[<ext>] ...')`），**验证通过后移除**再提交
- 发现运行时限制（如 stale-ctx 重入限制）→ 当场回到「前置：读源码找官方模式」重新设计，不要在限制上叠 workaround

## 真实案例（job_47，供对照自查）

- **失败链**（9667d13）：`agent_settled` 里 `sendUserMessage` → 嵌套运行死在半路（agent_start 后无 turn 无回复无 settled）→ 第二次调用报 stale-ctx（`This extension ctx is stale after session replacement...`）。三个 bug 全是「跑一次真实进程」能暴露的。
- **修复**（e2bfd6f）：换 `agent_end` + `deliverAs:"followUp"`（运行时官方模式）→ 三测通过（修复路径 / gate 隔离 / 上限防死循环）。
- 教训定位：**「写完+部署+提交」≠「验证过」**——commit 信息里的「实测验证」必须是事件流证据，不是自我声明。
