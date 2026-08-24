# Bugs Domain

**最后更新**: —  **来源 Job**: —

项目已知问题与精确解决方案。由 `rick learning` 和 `rick dream` 自动追加，勿手动覆盖。

## 已知问题与解决方案

### Python 集成测试：subprocess 读取真实 ~/.rick/config.json 导致触发真实 Claude 调用

**根因**: `subprocess.run()` 不显式传 `env=` 时，子进程继承父进程 HOME，`LoadConfig()` 读取真实 `~/.rick/config.json`，触发真实 Claude CLI 调用，测试超时或行为不符合预期。

**精确解决步骤**:
```python
env = os.environ.copy()
env["HOME"] = work_dir  # work_dir 下有 mock ~/.rick/config.json
result = subprocess.run([rick_bin, ...], env=env, cwd=work_dir, timeout=30, ...)
```

**首次发现**: job_26 / task1 / commit d50b255a  **验证状态**: ✅ 已修复

### pi 扩展安装"假成功"：`pi install <本地源码目录>` 写 settings.json 但 loader 不加载

**根因**: pi 的 extension loader（`dist/core/extensions/loader.js`）只认两种入口：`package.json` 含 `pi.extensions` 字段，或 `~/.pi/agent/extensions/<name>/index.ts` 子目录结构。pi 仓库自带的 `examples/extensions/subagent` 是裸 .ts 源码（无 package.json），`pi install <path>` 把路径写进 settings.json 的 packages 数组，但启动时 loader 不加载它 → `pi list` 显示已装，但 LLM 报 "no subagent tool available"。

**精确解决步骤**:
```bash
# 错误（假成功）:
pi install ~/.local/lib/node_modules/@earendil-works/pi-coding-agent/examples/extensions/subagent
# pi list 显示已装, 但 tool 不注册

# 正确（用 npm 包）:
pi install npm:pi-subagents          # subagent 扩展
pi install npm:pi-web-access         # web search 扩展
pi install npm:@wishx127/pi-tokyo-night  # 主题
# 验证真实生效（不只看 pi list）:
pi --mode json -p 'Use the subagent tool...' | grep '"toolName":"subagent"'
```

**首次发现**: job_30 / commit 940b0a9  **验证状态**: ✅ 已修复（改用 npm:pi-subagents）

### pi `--session` vs `--session-id` 语义差异（加载已有 vs 创建新会话）

**根因**: pi 的 `--session <id>` 是**加载已有会话**（找不到报 "No session found matching"），`--session-id <id>` 才是**指定 ID 创建新会话**（"creating a new session with that id"）。loop_2 研究 brief（research-5-N2）说 pi `--session` "接受 path 或 id"，但未标"加载 vs 创建"语义差异。迁移时误把 claude code 的 `--session-id`→pi 的 `--session`，导致 `rick easy` 报错。

**精确解决步骤**:
```bash
# 创建新会话用 --session-id:
piagent.CallCLI(cfg, mainFile, ModeInteractive, "--session-id", sessionID)
# 加载已有会话用 --session (resume):
piagent.CallCLI(cfg, "", ModeInteractive, "--session", sessionID)
# 验证 flag 语义（用一个不存在的 id 测）:
echo OK | pi --<flag> newuuid123 /dev/null 2>&1 | head -3
# "No session found" = 加载语义(错); "creating a new session" = 创建语义(对)
```

**首次发现**: job_30 / commit 098a1fb  **验证状态**: ✅ 已修复

### pi 解析器：pi 对 user 和 assistant 轮次都发 `message_end`，误取 user 输入作 FinalMessage

**根因**: pi 的 `--mode json` 事件流对 user turn（echo prompt）和 assistant turn 都发 `message_end` 事件，内部 message.content schema 相同。迁移时按 claude code 习惯取最后一条 message_end 的 text 作 FinalMessage，会把 user 输入误当 agent 回复。

**精确解决步骤**: 解析 `message_end` 时必须检查 `message.role == "assistant"`，只取 assistant 轮的 text：
```go
case "message_end":
    if ev.Message.Role != "assistant" { continue }  // 跳过 user echo
    for _, c := range ev.Message.Content {
        if c.Type == "text" { sess.finalMessage = truncate(c.Text, 200) }
    }
```

**首次发现**: job_30 / commit a81eda0  **验证状态**: ✅ 已修复


### go:embed 主题文件修改后未重建 → 旧 embed 覆盖磁盘新文件

**根因**: rick 内置主题（`internal/cmd/themes/*.json`）通过 go:embed 打进二进制。修改 json 后若只跑 `rick tools theme <name>` 而未先 `go build`，theme 命令用**旧二进制里的旧 embed** 重写磁盘主题文件 —— 表现是"改了没生效"，且再次激活时旧内容继续覆盖新文件。

**精确解决步骤**:
```bash
# 修改 themes/*.json 后必须：先重建，再激活
go build -o bin/rick ./cmd/rick
./bin/rick tools theme <name>
# 验证安装副本（非源码）确实是新内容：
python3 -c "import json; print(json.load(open('$HOME/.rick/pi/agent/themes/rick.json'))['colors']['mdHeading'])"
```

**首次发现**: job_33 / commit dac5370  **验证状态**: ✅ 已修复

### fake pi 脚本在 PATH 替换测试中静默失败（command not found 被吞）

**根因**: Go 测试用 `t.Setenv("PATH", tmp)` 只留 fake bin 目录后，fake 脚本里 `cat`/`ls` 等**外部命令**找不到（`command not found`），若脚本再 `2>/dev/null` 或测试用 `cmd.Output()`（吞 stderr），错误完全不可见——表现为"某分支静默返回空"。`echo` 是 shell 内建所以该分支正常，造成"部分命令行、部分不行"的迷惑现象。

**精确解决步骤**:
```bash
# fake 脚本开头恢复系统 PATH（首选）
#!/bin/sh
export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH
# 或只用内建：while IFS= read -r line; do echo "$line"; done < "$FILE"
# 调试：去掉 2>/dev/null、加 >&2 诊断、sh -x ./fake 独立跑
```

**首次发现**: job_33 / commit b018a41  **验证状态**: ✅ 已修复

### pi 解析"托管运行时优先"后，PATH-fake 测试静默命中真实 pi（挂死/联网）

**根因**: `FindBinary`/`piCommand` 等把解析顺序改为 `cfg.PiPath → RuntimeBin()（~/.rick/pi/agent/runtime）→ PATH` 后，测试只 `t.Setenv("PATH", fakeDir)` 不再够——真实托管 pi 存在时优先命中，fake 不生效；部分测试（plan/learning workflow）甚至拉起真实 pi 交互会话 → 全套件从 265s 挂到 10 分钟超时（panic 无具体 --- FAIL）。

**精确解决步骤**:
```go
// 所有依赖 PATH fake pi 的测试，隔离托管运行时解析根：
t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())  // piagent.AgentDir() → temp，RuntimeBin() 不存在 → 回退 PATH
// 或等价：t.Setenv("HOME", t.TempDir())    // AgentDir() = $HOME/.rick/pi/agent
// 配置类测试用 setupPiSettings（HOME 隔离）模式
```
排查：`go test -timeout 60s -run <可疑测试>` 逐个定位（全量 6 分钟太慢）；超时 panic 的 "running tests:" 直接点名。

**首次发现**: job_34 / commit c4812dc  **验证状态**: ✅ 已修复

### Go raw string 里嵌 JS 模板字符串（反引号）会截断字符串

**根因**: Go raw string（`` ` ``...`` ` ``）无法包含反引号——测试 fixture 要复刻含 `${...}` 模板字符串的 JS 代码时，直接写 `` \`-${x} ` `` 会在第一个反引号处截断，编译报错。

**精确解决步骤**:
```go
// 用"raw 段 + 解释型段"拼接，反引号放解释型段里：
const fixture = `result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${removedLine}`" + `));`
```

**首次发现**: job_34 / commit 740756d  **验证状态**: ✅ 已修复

### 幂等字符串 patch 的"锚点仍存在"陷阱（helper 插入重复）

**根因**: 字符串替换式 patch 若 old 锚点在 new 里**仍然存在**（如往 `function formatBashCall(args) {` 前插 helper，锚点本身未变）→ 二次运行 old 再次命中 → 重复插入。

**精确解决步骤**: 用**整函数替换**——old 包含完整函数体（含会被改掉的行），替换后被消费；或让 old 含一段唯一且必被修改的文本。验证：同一命令跑两遍，第二遍应报 no-op，文件逐字节不变。

**首次发现**: job_34 / commit 740756d  **验证状态**: ✅ 已修复

### `git add bin/rick` 静默失败：bin/ 在 .gitignore，未加 -f 会漏提交二进制

**根因**: `bin/` 在项目 `.gitignore` 中，`git add bin/rick` **无报错但静默 no-op**（文件未被暂存）。agent 只看 commit 成功消息就以为二进制已提交，实际 `bin/rick` 漏 commit，后续用旧二进制跑 check/验收。job_35 中 task7/task8/task10/task11 反复踩中。

**精确解决步骤**:
```bash
go build -o bin/rick ./cmd/rick
# 必须 force 暂存（bin/ 被 ignore）:
git add -f bin/rick
git commit -m "chore(taskX): rebuild bin/rick"
# 验证确实已暂存:
git status --short bin/          # 应显示 M bin/rick
```

**首次发现**: job_35 / task7 / commit 387225d  **验证状态**: ✅ 已修复（force-add 固化）

### tasks.json `updated_at` 缺时区 → Go time.Parse 失败

**根因**: tasks.json 中 task 的 `updated_at` 若写成无时区的 `2026-08-17T06:01:04.789297`，Go 按 RFC3339 解析（`time.Parse(time.RFC3339, ...)`）会报 `parsing time "..." as "2006-01-02T15:04:05Z07:00"` 失败，门禁报 `tasks.json not found or invalid`。手工编辑 tasks.json 时容易漏 `+08:00` 后缀。

**精确解决步骤**:
```bash
# 每条 updated_at 必须带时区（RFC3339 完整格式）:
#   "updated_at": "2026-08-17T06:01:04.789297+08:00"
# 用 mark_task_success.py 自动写（已用 timezone(timedelta(hours=8)) 修复）:
python3 .rick/skills/mark_task_success_skill/mark_task_success.py --job job_N --task taskX
# 验证 JSON 可被 Go 解析:
python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing
```

**首次发现**: job_35 / task12 / commit a080458  **验证状态**: ✅ 已修复

### glm-5.3 thinking 耗尽输出 token → 子代理空响应、简报永不落盘

**根因**: catpaw-proxy/glm-5.3 是推理型模型，thinking 与正文共享单次响应输出预算。think 类纯分析任务（无中间工具动作）会让模型在 thinking 里起草整个简报全文（实测 14K-17K chars），thinking 在句子中间被截断（服务端输出上限），响应中没有任何 text/tool_use 块 → turn 结束、`write` 永远不执行。pi 侧表现：run "completed" 但输出是中间过程叙述，或 `Subagent produced no output (possible model cold-start or empty response)`。resume 时说「直接产出简报全文」会复现同一死法。

**精确解决步骤**（v4.0.4 修复）:
1. 子代理契约改为**分节增量落盘**：首节 `write`、后续 `bash cat >> file << 'RICK_EOF'` 逐节追加；thinking 只做当前节的局部推理（见 `templates/skills/think.md`、`templates/skills/research.md`）
2. think agent frontmatter 加 `thinking: low`（产出结构模板化，低思考足够）
3. sense_loop 交付门禁的 resume 指令改为「立即开始分节落盘」，禁止「直接产出简报全文」
4. 诊断方法：子会话 `run-0/session.jsonl` 最后一条 assistant 消息若只含 thinking 且 mid-sentence 截断，即此病

**精确机制**（v4.0.6 终版复核，修正 v4.0.5 的部分结论）：瓶颈既不是 max_tokens（客户端 1M）也不是 thinking 预算（已验证 thinking:high+32768 生效时仍在 ~8K 处被掐），而是 **catpaw 代理服务端对单次响应的输出上限 ≈8192 tokens**（英文 thinking ≈4 chars/token，30.7K chars ≈ 8K tokens 即死；8d6ef677 实测 output=8246 后零产出）。客户端任何配置都绕不过。glm-5.3 在「read 大材料（>3K chars）后的同一回合」倾向一次性推演全部分析（30K+ chars thinking）→ 必被掐断。

**死亡高危点**：`read 大材料 → 同回合想完全部分析 → 写`。think-S 曾成功是因为 resume task 未让它再 read；think-E 两次暴毙都是「先 read research-E.md」触发的。

**v4.0.6 修复（先骨架后填充）**: ① think/research skill 硬规则「read 大输入后当前回合只允许 write 骨架（节标题+清单），禁止同回合分析」；② sense_loop resume 指令模板改为「第 1 步本回合 write 骨架，第 2 步起逐节填充；读材料用 offset/limit 分段」；③ 空响应类错误（基础设施）重试上限 2→3 次，且 resume 从已有进度继续不重写。v4.0.5 的 thinking:high + 32768 预算保留（有正面作用但非根治）。

**首次发现**: job_35 验收期 loop_2；复发于 loop_3 think-E / v4.0.4→4.0.6  **验证状态**: ✅ 已修复（read-dump-first + 门禁 3 次重试）

### write 工具单次调用有大小限制 → 大简报一次性写入被截断

**根因**: pi 的 `write` 工具单次调用内容过大时会被截断（research 写 38KB 简报实测踩坑，自我纠正为分块追加才成功）。

**精确解决步骤**: 大文件一律分块——首块 `write`（≤60 行），后续 `bash cat >> <file> << 'RICK_EOF'` 逐块追加，写完 `grep -c '^## '` + `wc -c` 校验。已固化进 research/think skill 落盘契约（v4.0.4）。

**首次发现**: job_35 验收期 / loop_2 research-S / v4.0.4  **验证状态**: ✅ 已修复（分块契约）

### 子代理自交付契约在 ≈8K 输出上限下不可稳定 → 改为 inline 简报 + parent 代写

**根因**: catpaw 代理对单次响应有 ≈8K tokens 服务端硬上限（v4.0.5 的 thinking:high+32768 预算已验证生效但仍在 ~8.2K 处被掐，客户端配置无法绕过）。自交付契约（v4.0.3-4.0.6）要求子代理自己 write 大简报（实测 19-25KB），无论分节/骨架/分块怎么设计，都依赖「模型每回合自觉小输出」——glm-5.3 在「读大材料后同回合想完全部分析」的模式下反复超限（loop_3 think-E 两连毙）。而 loop_1 的 inline 交付（think 无写工具、最终回复即简报）在同模型上从未因此失败。

**精确解决步骤**（v4.0.7 设计反转）:
1. think 工具收为 `read, grep, find, ls`；research 收为 `read, grep, find, ls, web_search, fetch_content`（无 write/bash，物理上不能写文件）
2. 交付契约改为 inline：**最终回复 = 简报全文**（research ≤3000 字 / think ≤2500 字，紧凑结构），简报确实是简报（保结论数值结构、弃过程细节）
3. sense_loop：串行链把 research 的 r.output 直接注入 think 的 task（消灭「read 大文件」回合）；**parent 校验返回值后自己 write 落盘**（确定性动作，必然交付性的新保证人）
4. 门禁：校验 r.output 非空且形似简报 → parent 落盘；叙述性回复/空响应 → resume「直接把最终简报作为回复输出」（≤3 次）

**首次发现**: loop_3 think-E 连毙 + 用户设计决策 / v4.0.7  **验证状态**: ✅ 已实施（待 loop_3 后续验证）

**v4.0.8（人工验收反馈：递归外包 + 自落盘，取代 v4.0.7 的零写权限一刀切）**:
1. research 恢复 write+bash 并持 subagent（fanout child）：自落盘调研报告（write 首块 + bash cat >> 分批追加）；预估内容 >1500 字或子问题 ≥3 时拆 ≤4 个叶子 worker（agent:'worker'，各写各的 research-<阶段>-leaf-<i>.md）递归尽调，research 复核置信度后汇总（不采信叶子自报）
2. think 恢复 write（无 bash，≤2500 字一次 write）并持 subagent：自落盘思考简报，parent 读文件；重载思考先拆解（≤4 子命题）→ 外包子命题推演（叶子只读，结论走回复 ≤600 字）→ 汇总（打分权留在 think）
3. 递归封底靠 pi maxSubagentDepth=2（叶子 worker 无 subagent 工具，天然不会无限递归）；fanout ≤4 由模板约束
4. sense_loop 门禁改为「回执 + 文件校验（wc -c ≥800 / grep -c '^## ' ≥2）」；未交付 resume「立即落盘」≤3 次；3 次后降级 inline 回复 parent 代写（v4.0.7 安全网保留）
5. 单写者改为按文件隔离：exporter 写 rfc/，research 写 briefs/research-*，think 写 briefs/think-*，parent 写 judgment.md
6. 串行链不再把 r.output 全文注入 think 的 task（那是 v4.0.7 inline 遗留，现在 r.output 是回执）——改为传简报文件路径，think 用 read 分段读取

### `runs.run` 的 resume 与 agent 同传 → 子代理编排脚本直接报 mutually exclusive

**根因**: pi-subagents 硬校验（scripted-workflow.ts）：`runs.run(key, { resume, agent, ... })` 中 `resume` 与 `agent` **互斥**——resume 沿用被恢复子运行的原 agent/model/工具契约，不允许再指定 agent。v4.0.8 的 research skill 只写了「叶子空响应 → resume 对应叶子一次」没给语法约束，glm-5.3 直觉性地 `runs.run('leaf-2-retry', { agent:'worker', resume:'8f6b5929', task })` 同传两者 → `runs.run('leaf-2-retry') resume and agent are mutually exclusive.` 且失败 run 的 status 还会带 `Resume: unavailable; no child session file was persisted`（校验失败发生在派发前，无子会话）。

**精确解决步骤**（v4.0.9 已修模板）:
1. 子代理/叶子失败重试一律 **fresh 重派**：新 key + 原 task（不传 resume）
2. 仅当叶子已落盘部分内容需续写、且其 session 已持久化时才用 resume，且 **resume 单独出现、省略 agent**：`runs.run(key, { resume: '<run-id>', task: '<续写指令>' })`
3. 已写进三个模板的硬约束：skills/research.md（外包纪律）、skills/think.md（外包纪律）、sense_loop.md（1.5 交付门禁 resume 语法）
4. 顺带修正叶子选型：联网调研叶子用 `agent:'researcher'`（持 web_search/fetch_content）；`agent:'worker'` 无网络工具，联网只能 bash+curl（信源降档）

**诊断方法**: 子会话 transcript 里搜 `mutually exclusive`；run status 带 `Resume: unavailable` 即校验前失败。

**首次发现**: job_35 验收期 loop_5（test 目录实测）/ v4.0.9  **验证状态**: ✅ 已修复（模板硬约束 + fresh 重派语义）

**v4.0.10（人工验收反馈：简报有效性优化——面向决策，不面向过程）**:
1. research 简报去掉尽调树（尽调树是内部调研方法）：只呈现「事实性结论 + 成立前提 + 来源/信源等级」（≤12 条一行式）+ R7 + 阶段特定产出
2. think 简报去掉全部过程产物（假设列表/4 维打分表/期望分表均为内部筛选方法，不进简报）：只呈现 top-N 需 human 回答的问题，统一隐含前提格式「若 [X] 成立，那么也假设了 [Y] 的成立——[Y] 真的正确吗？」，每条附依据（引自 research 简报）/改变判断的证据/性质标注（事实性 vs 判断性）
3. 原 3 问内化：Q2 前提=问题主体，Q3 反例=「改变判断的证据」，Q1 信念=human 自答方向
4. sense_loop 新增 **1.6 事实性模糊消解循环**：think 简报中存在事实性 Y（可调研澄清）→ parent 追加 research→think 串行链（新落盘路径 `-r2/-r3`）→ 循环直到问题全为判断性或追加达 2 轮上限（剩余标注「未消解」上呈）；目标是 human 只被问「必须由人回答」的问题
5. 批判门禁 think-gate 输出同样改为隐含前提问题式
6. v4.0.11 措辞修正：问题尾部「[Y] 真的正确吗」→「这真的正确吗」（实测 glm-5.3 会照抄字面 Y 占位符；改「这」回指前文根除该问题，并加注「X/Y 用实际内容代入，禁止保留字面占位符」）

### pi 子运行默认 30 分钟超时会恰好在交付前掐死长调研（SIGINT，进度丢失）

**根因**: pi-subagents 前台运行与异步子运行默认 `timeoutMs=1800000`（30 分钟）。glm-5.3 经 catpaw 代理单轮 TTFB 可达 8.5 分钟（thinking:high 首轮策略），research 叶子扇出（runs.all 3 个联网 researcher）再吃 10-15 分钟——E 阶段 research 实测 27 分钟完成全部调研、正要落盘简报时被 SIGINT 掐死（meta: `Subagent timed out after 1800000ms`, durationMs=1800026, turns=4）。超时后子会话进度不保留，靠 revive 机制重建上下文又要 3 分钟。research 实际最坏情况 ≈30-35 分钟 > 默认 30 分钟，**必炸**。

**精确解决步骤**（v4.0.12）:
1. agents.go：think/research/exporter frontmatter 显式 `timeoutMs: 3600000`（单运行默认 60 分钟）；env_test 断言
2. sense_loop.md：所有派发模板（串行链/单发/门禁/extra/exporter/1.6 追加链）workflowScript 调用带 `timeoutMs: 3600000`；运行语义节注明理由
3. skills/research.md：叶子扇出带 per-leaf `timeoutMs: 900000`（15 分钟止损，防单叶子拖死 research 预算）；叶子超时按空响应同规则（重派一次→R7）；新增「自己预算 60 分钟，派叶子前看剩余时间」纪律

**诊断方法**: `*_meta.json` 的 `durationMs` 恰为 1800026 + `error: Subagent timed out after 1800000ms`；transcript 尾部 tool_start 无对应 tool_end。

**首次发现**: job_35 验收期 loop_7（test 目录 E 阶段 research）/ v4.0.12  **验证状态**: ✅ 已修复（60min 显式超时 + 叶子 15min 止损）

### pi-subagents 0.51.0 strict 工具校验：显式 tools 里列 intercom 会硬失败

**根因**: pi-subagents 0.51.0 起，agent frontmatter 的 `tools` 是 strict allowlist + required 校验（tool-availability.ts：子会话实际注册工具 ∩ required，缺即报 `requested unavailable child tools: <name>` 硬失败）。intercom 是 pi-intercom 扩展提供的工具，launcher（intercom bridge / native supervisor channel）会把它和 contact_supervisor 自动注入 agent 的解析后工具；但子会话未加载 pi-intercom 扩展时该工具不存在 → 校验失败。job_35 验收期 think-gate 运行实测报 `Agent 'think' requested unavailable child tools: intercom`（8-19 13:44，meta 01d7a75d）。

**精确解决步骤**（v4.1.1 全量开放策略）:
1. tools 列表**不写 intercom/contact_supervisor**——launcher 按需自动注入（实测 think 子会话显示 contact_supervisor 已注入）
2. 三 agent（think/research/exporter）统一全量工具集：`read, grep, find, ls, bash, write, edit, web_search, fetch_content, subagent`
3. **subagent 必须显式保留在 tools 里**：省略 `tools` 字段 = 无 explicit allowlist = 不加载 fanout-child 扩展 → think/research 失去派发叶子能力（实测验证：无 tools 的临时 agent fanout 被 degrade）。pi-args.ts `fanoutAuthorized = declaredBuiltinTools.includes("subagent")`
4. 叶子（worker/researcher builtin）的深度封底不受影响：maxSubagentDepth=2 仍生效

**诊断方法**: `*_meta.json` 的 `error` 含 `requested unavailable child tools`；对照 pi-subagents src/runs/shared/tool-availability.ts。

**首次发现**: job_35 验收期 loop_7（think-gate 报错）/ v4.1.1  **验证状态**: ✅ 已修复（全量工具 + 不列 intercom）

### human-loop 协议 v4.2.0：四段链 + exporter 教学综合（表达层升级）

**设计**（人工验收驱动，"这才是真正的 human-loop"）:
1. 每阶段四段链：`research（事实）→ think（追问）→ {事实模糊性消解循环} → exporter（第一性原理教学简报）`，human 读到的永远是 exporter 的教学简报，不再直接展示 research/think 原始简报
2. exporter 双模式：①教学简报模式（每阶段）——教师身份，结构 = 发生了什么（≤10 行因果链）+ 这个领域的知识是什么样子（≤25 行第一性原理重建关键概念，保留来源）+ 启发式追问（承接 think 隐含前提问题，建立在已讲清的知识上，附改变判断的证据）；②RFC 固化模式（最终，大纲+内容两阶段，不变）
3. 教学纪律：先给足信息再追问（禁止裸问）；≤2500 字；不替 human 判断；自落盘 `briefs/teach-<阶段>.md` + 回执交付
4. sense_loop 新增 **1.7 教学综合**（1.6 消解循环收敛后、展示前）：派发 exporter（输入 = 最新轮 research/think 简报路径），走 1.5 门禁（teach 文件 ≥800 字节、≥2 节标题）；EC 阶段回顾版教学简报也由 exporter 产出（read 各阶段 teach-*.md + judgment.md）
5. 各阶段「简报追加」统一更名为「教学简报内容」（exporter 综合时必含的阶段特定部分，如 S 三连追问/E 视角候选/N2 打分表）

**实测**: exporter 教学模式真机验收通过（MECE 主题测试简报：教师口吻/第一性原理/类比/启发式追问+改变判断证据，结构与契约完全一致）。

**首次发现**: 设计决策（用户）/ v4.2.0  **验证状态**: ✅ 已实施（待完整 loop 实测）

**v4.2.1（人工验收反馈：教学详实化）**: 教学简报从「≤2500 字压缩」转向「详实优先」——篇幅服从理解需要，压缩废话不压缩知识；领域知识节讲透机制/边界/常见误区，关键主张标注来源（链接/书籍章节/信源等级，来自 research 简报，不足可 web_search 补充引用）；新增第 5 节**延伸学习指导**（若决策需更系统的领域理解：书籍+章节/链接+为什么读——让 human 先补课再决策，教师职责）；长内容分批落盘（write 首块 + bash cat >> 追加，每批 ≤60 行，规避 ≈8K 单响应上限）；sense_loop teach 门禁下限提高到 ≥2000 字节（过短=未讲透）。

### 全局派发规范收敛 + plan grilling 子代理化（v4.2.2）

**全局收敛**（human-loop 的护栏推广到全部编排模板）：
- plan.md：六维评审 fanout 补 `timeoutMs: 3600000` + 全局派发规范节
- dream.md：四路验证链补 timeoutMs + 规范
- doing.md：角色定义处加全局派发规范（timeoutMs 3600000 / worker 不递归 / 单写者串行 / resume-agent 互斥警告）
- sense_loop.md：已有（基准）
- ctrl.md：声明无编排权不派发（保持）；easy：单会话无子代理（保持）

**plan grilling 子代理化**（设计树 + 事实消解在前、判断上呈在后）：
- 4a parent 生成 MECE 设计树（落盘 `plan/grilling/design-tree.md`；非叶子层=pipeline 澄清，叶子层=四维度落实）
- 4b 派 `agent:'research'` 逐层下钻：可调研节点消解（附来源），不可消解→判断节点（选项+权衡）；自落盘+回执
- 4c 派 `agent:'think'` 对判断节点提取隐含前提问题；自落盘+回执
- 4d parent 把全部判断节点合并成**一轮**批量追问（附上下文/选项/权衡/推荐答案）；不问已消解问题
- 4b' 回流 ≤1 轮（human 答案打开新事实问题→追加 research-r2）；超出写风险点进 task
- 4e 按 grilling skill 终止条件收敛声明
- 复用 research/think（不新建 agent）；grilling skill 的设计树模型/追问规范保留为方法论基准

**首次发现**: 设计决策（用户，与 human-loop v4.2.0 哲学同构）/ v4.2.2  **验证状态**: ✅ 已实施（dry-run 渲染验证；待真实 plan 实测）

### doing v4.3.0 四重改造（git init / 原生轨迹 / hook 提交 / 双 agent TDD）

**改造内容**（人工验收驱动，doing 卡死在环境考察的根治）:
1. **doing 前置 git init**（handler.ensureGitRepo）：cwd 非 git 仓库时确定性 `git init` + 初始 commit（rick 身份），不留给 parent 纠结
2. **act-path 提取层废除**：ctrl/dream/learning 全部改为直接读**原生行为轨迹**——`<cwd>/.pi/subagents/artifacts/*_{meta,transcript,output}.*` + pi 会话 jsonl（doing/session_id 定位）；删除 trace.md/raw_session_coding.log 期望（从未真正生成过）
3. **commit 确定性 hook**（rick-gates 升级为真 pi extension）：`~/.rick/pi/agent/extensions/rick-gates/index.ts` 注册 `task_complete` 工具——worker 声明完成 → hook 跑测试（`python3 <doing_dir>/tests/<task_id>/run_tests.py`，退出码为准，非 0 拒绝提交）→ `git add -A && git commit -m "feat(<task_id>): <summary>"`（git 身份固定 rick）→ tasks.json 原子写回 status=success + commit_hash；重复调用拒绝；nothing-to-commit 时复用最近 feat(task_id) commit
4. **双 agent TDD 编排**（orchestration.go renderWorkflowSection 重写）：每 task 两段 `runs.run`——`<id>-test` worker 按 # 测试方法 落盘 run_tests.py+test_impl.py（pytest，确认 RED，禁止写实现）→ `<id>-impl` worker 按 # 任务目标/# 关键结果 写实现跑绿（禁止改测试）→ 调 task_complete 由 hook 提交；全部带 timeoutMs: 3600000

**extension 技术要点**: pi 自动发现 `~/.rick/pi/agent/extensions/*/index.ts`；工具定义用 `defineTool` + `@earendil-works/pi-ai` 的 Type（**不要写 `ToolDefinition<typeof Type.Object(...)>` 泛型注解**——pi 的 TS 解析器报 ParseError，catpaw-proxy/官方 examples 均为 defineTool 模式）。

**真机验收**（test 目录，2026-08-20）: extension 加载（TASK_COMPLETE_TOOL=yes）→ RED（模块缺失 2 failed）→ GREEN（2 passed）→ task_complete 提交（commit b13dce3d + tasks.json success）→ 重复调用拒绝。全部通过。

**首次发现**: 设计决策（用户四条指示）/ v4.3.0  **验证状态**: ✅ 已实施并真机验收（完整 job 跑批待测）

### doing v4.3.1：分层 DAG 并行 + 动态超时 + parent 监督 + 拓扑门禁

1. **分层 DAG**（orchestration.go topoLevelsOrch）：Kahn 分层——层间严格顺序，层内 test-worker `runs.all` 并行（写域 tests/<id>/ 互斥），impl-worker **全局串行**（单写者：hook 的 git add -A 要求同一时刻只有一个实现写者——全并行 impl 需 worktree 隔离+确定性 merge，成本高未做）
2. **动态超时**（estimateTimeoutMs）：按 task.md 工作量估算——base 15min + KR 条数×8min + 测试场景×4min，clamp [20,90]min；实测渲染出 27-90min 梯度
3. **拓扑门禁**（生成时确定性校验）：环检测 + 依赖引用存在性（依赖必须在 pending 集或 satisfied 集——已 success 的依赖合法忽略，**不构成层间约束**；引用不存在 id = ⛔ 拒绝派发并报错指引修正 plan）。test 目录实测抓到过「依赖已完成的 task」误判（已修：satisfied 集合传入门禁）
4. **parent 监督**（doing.md 新增监督节）：`{action:"status", view:"transcript", id}` tail 运行中 worker 实时轨迹；`{action:"steer"}` 中途纠偏；`{action:"stop"}` 死循环止损 + fresh 重派；层间检查点（层内全 success 才进下层）

**首次发现**: 设计决策（用户）/ v4.3.1  **验证状态**: ✅ 已实施（dry-run 渲染验证 5 层 DAG + 动态超时梯度；完整跑批待测）

### doing v4.3.2：层检查点提交（level_complete）——impl 并行 + 单次层 commit

**设计**（用户提议）：提交点从 task 粒度挪到**层检查点**，impl-worker 并行期间零 git 操作（不调任何提交工具），唯一 git 写在检查点单次执行——并行与确定性提交的矛盾消解。
1. rick-gates extension 新增 **`level_complete`** 工具（{level_tasks[], doing_dir, summary}）：逐 task 跑测试 → **全部全绿**才继续（合跑抓同层集成回归）→ `git add -A` 单次 commit（feat(layer): task2+task4: ...）→ tasks.json 批量写 success+commit_hash（同层共享同一 hash）；任一失败拒绝提交并列清单（parent 只重派失败 task 的 impl，其他产物稳定）
2. 编排重写为**每层 4 步**：① runs.all 并行 test-worker ② runs.all 并行 impl-worker（task 文本明确「不碰 git、只写本 task 声明文件域」）③ parent 本会话直接调 level_complete（extension 全局加载，parent 也有该工具）④ 校验 tasks.json 层内全 success 进下一层
3. **同层写域互斥**进 plan.md 任务分解原则（互相独立的 task 文件域不重叠，否则补依赖分层）——impl 并行的安全前提
4. timeoutMs 动态估算不变；task_complete 保留（单 task 补跑场景）

**真机验收**（test 目录层 1：task2+task4）：RED（各自 failed）→ GREEN → level_complete 合跑全绿 → 单 commit 4d6a261e + tasks.json 批量 success（共享 hash）；中间 constants.py 意外丢失被层检查点合跑拦截（拒绝提交）——集成回归拦截能力实证。

**首次发现**: 设计决策（用户）/ v4.3.2  **验证状态**: ✅ 已实施并真机验收

### v4.4.0：plan/doing 深度重构——doing pipeline（分层 DAG + 层门禁）+ 写域确定性门禁

**架构**（用户设计，DAG → 分层 pipeline）:
1. **plan 双树**：设计树（grilling 澄清）+ 分解树（6a 分层 DAG：层间递进、层内写域独立可并行；6b **层门禁 human 设计**：plan/gates/gate{N}.py 确定性程序（exit 0=层验收），草案 agent 生成但必须 human 逐层确认；6c pipeline.md 执行契约）
2. **七维评审**（六维 + #7 写域与门禁检查：同层写域两两不相交 + gate 存在可执行 + pipeline 一致性）
3. **task.md 增加 # 写域 节**（目录尾 / =前缀语义，文件精确名）
4. **写域确定性门禁**（Go 侧，doing 生成编排时）：同层多 task 必须全员声明写域且两两不相交（相等/前缀包含=冲突）；单 task 层豁免（向后兼容）；缺失 gate{N}.py 也 ⛔ 拦截
5. **doing 每层 5 步**：①门禁判别力验证（跑 gate 应为红——TDD 思想用于门禁本身）②并行 test-worker ③并行 impl-worker（按匹配 loop 执行，写域互斥）④level_complete（gate_cmd 模式跑 human 门禁 → 绿 → 单次 commit → tasks.json 批量）⑤debug 压缩传递（前层教训注入下层 worker task）
6. **level_complete 升级**：gate_cmd 参数（human 的 gate{N}.py 优先；缺省=逐 task 测试）；门禁失败拒绝提交输出详情
7. **loop 与 pipeline 正交**：doing parent = 结对导航员（不执行、读行为轨迹全局监督纠偏）；worker 按项目 loop 工作方法执行
8. **easy 同步**：grilling 段升级为「设计树澄清 + 单会话分解树/层门禁设计（human 确认门禁后编码，层门禁绿才进下层）」

**真机验收**（test/job_1）：写域门禁拦截无声明同层（task5,task7 ⛔）→ 补写域 → gate 缺失 ⛔ → 补 gate1-5 → gate1 红（判别力）→ GREEN → level_complete(gate_cmd) → commit 743343cc + tasks.json。**门禁程序自身 bug 实证**：gate 路径上溯层数错（4 层应为 5 层）导致误报「测试未落盘」——修复后通过；这正是「步骤 ① 门禁判别力验证」存在的理由（门禁也要先验证再信）。

**首次发现**: 架构决策（用户）/ v4.4.0  **验证状态**: ✅ 已实施并真机验收（完整 job 跑批进行中）

### v4.4.1：实现流水线 skill 化 + 结构门禁下沉 hook（Go 保持薄）

1. **实现流水线抽成 skill**（templates/skills/pipeline.md）：结构契约（分层 DAG/写域/层门禁三要素）+ 产物落盘 + doing 5 步 + 关键原则；plan.md Step 6 与 easy 第一步均改为引用 skill（去重）；pibuilder BuildPlan 把 pipeline 加入内联技能清单（单文件内聚产物同步）
2. **结构校验从 Go 侧下沉到 rick-gates hook**（新 `pipeline_gate` 工具）：分层 DAG/依赖引用存在性/同层写域两两不相交/gate{N}.py 存在性——全部确定性逻辑集中在 hook；Go（orchestration.go）只渲染「第 0 步：调 pipeline_gate，⛔ 才继续」指令 + 最小环检测（编排无法渲染的硬前提）
3. Go 侧删除 levelWriteDomainGate/writeDomainConflict/gate 存在性检查（约 80 行），renderPipelineGateStep 取代

**真机验收**：写域冲突（src/a/ vs src/a/shared.py 前缀包含）⛔ 拦截 → 修正 ✅ 输出分层结构 → gate 缺失 ⛔。

**首次发现**: 设计决策（用户：skill 复用 + Go 简洁）/ v4.4.1  **验证状态**: ✅ 已实施并真机验收

### v4.4.2：测试收敛到层门禁（task 级自测，模块级集成测试）

**设计**（用户：task 不生成专门测试脚本，自测即可；测试收敛到 human 确认的门禁，门禁测试=模块集成测试）:
1. doing 每层 5 步→**4 步**：删并行 test-worker 步骤——①门禁判别力验证（集成测试此刻必红）②并行 impl-worker（按 # 测试方法**自测**，过程性；不落盘共享测试目录）③level_complete（gate_cmd **必填**，跑模块集成测试→绿→单次 commit→tasks.json 批量）④debug 压缩传递（末层归档 debug-summary.md）
2. **删除 task_complete 工具与 run_tests.py**（per-task 测试目录协议废止）；hook 只剩 pipeline_gate + level_complete
3. plan：6b 门禁=模块集成测试（test_mod{N}.py 或 gate 内联，唯一持久化测试资产）+ 业务验收；# 测试方法 语义=worker 自测指引；评审 #5 改双层检查（自测指引覆盖四要素 + 门禁集成测试覆盖模块集成面）
4. doing_loop：RED/GREEN 改「自测驱动实现（TDD 方法，过程性）」；COMMIT 改「完成回执」（不碰 git）

**踩坑（两次）**: ①删 taskCompleteTool 时误删公共函数（findGitRoot/git/updateTasksJSON 定义在其上方）——已恢复；②gate 路径上溯层数（plan/gates→根=5 层）误写 4 层——已把教训写进 pipeline skill（gate 开头自检 assert isdir）

**真机验收**：判别力红 → 自测绿 → gate 绿 → level_complete commit 4ce076fb + tasks.json success。

**首次发现**: 设计决策（用户）/ v4.4.2  **验证状态**: ✅ 已实施并真机验收

### v4.4.3：设计树恢复动态逐层下钻（不再一轮压平）

**问题**（用户指出）：v4.4 的 grilling 改造把设计树消解压平成了一轮（生成树→research→think→批量追问），丢失了 grilling skill 本有的逐层下钻语义（for each layer: while 未达标: 追问; descend）。

**修复**：plan Step 4 重写为逐层下钻循环——4a 生成根层（design-tree.md 活文档）；4b 逐层循环五步：L1 research 下钻当前层（消解/标注判断节点）→ L2 think 提炼追问 → L3 批量追问 human → L4 事实回流(≤1) → L5 **对照每层终止条件**：非叶子层达标= pipeline 完全澄清 → descend 展开下一层；叶子层达标=四维度落实 → 收口。**当前层未达标禁止下钻**。全部叶子收口 → 4c 分层决策摘要。
- **设计树 ↔ 实现流水线联动**（pipeline skill）：设计树叶子层四维度（代码实现/文件结构/工具调用/环境配置）直接映射 task 的 # 写域/# 关键结果/# 测试方法；流水线分层 DAG = 设计树澄清结果的执行投影
- easy 同步：设计树动态下钻语义（逐层循环+终止条件+descend）
- 落盘形态：grilling/research-L{N}.md、think-L{N}.md 按层编号（替代原单份 research-grilling.md）

**首次发现**: 设计缺陷（用户指出压平语义丢失）/ v4.4.3  **验证状态**: ✅ 已实施（dry-run 渲染验证；多轮下钻实测待真实 plan）

### v4.4.4：设计树顶层 = 具体 OKR（MECE + OKR 充分性双约束）

**设计**（用户）：设计树顶层一定是一组具体的 OKR——O=可验证全局目标，第二层 KR=基于演绎（KR₁∧…∧KRₙ⟹O）或归纳（KR 集是 O 的充分证据）：所有 KR 达成后 O 可行达成。**每层递归满足双约束**：MECE（形状：完备+互斥）+ **OKR 充分性**（推导：子节点联合达成⟹父节点达成——只 MECE 不充分不行，全做完父目标没达成=分解错误）。
- grilling skill：设计树模型升级为「OKR 充分性推导树」；每层终止条件加**充分性自检**（以本层全达成为前提能否推出父目标达成；推不出=缺 KR 或切分维度错，禁止带病下钻）
- plan Step 4：4a 根层=O+KR 集落盘；L5 终止判定加充分性自检门
- pipeline skill：KR 链逐层下传——叶子层 KR 映射 task # 关键结果，四维度映射 # 写域/# 测试方法；流水线分层 DAG=OKR 推导的执行投影（层门禁验证该层 KR 联合达成）
- easy 同步 OKR 根语义

**首次发现**: 设计原则（用户）/ v4.4.4  **验证状态**: ✅ 已实施（dry-run 渲染验证；真实 plan 实测待观察充分性自检执行质量）

### v4.4.5：阶段提示词改走系统提示词（compaction 持久）

**问题**（用户指出）：阶段提示词（plan/doing/easy/ctrl/human-loop/learning/dream）此前作为初始 user 消息注入（pi 位置参数→路径字符串→模型自己 read 文件），长会话 compaction 会压缩掉协议细节——协议会「遗忘」。

**修复**：
- CallCLI buildArgs：promptFile → `--append-system-prompt <path>`（pi 对该参数自动检测文件路径读取内容，resource-loader.js resolvePromptInput）+ 初始 user 消息 = 固定 bootstrap 触发句（「开始：按系统提示词中的 rick 协议立即执行你的职责」）
- piRuntime.Run（doing json 路径）：instance 协议同样走 `--append-system-prompt`（与 method 临时文件并列，append 可多次）+ `-p` bootstrap
- 系统提示词在 compaction 中**永不压缩**（pi 的 compaction 只作用于会话消息，系统提示词每轮都在）

**注入全景（v4.4.5 后）**：阶段协议（method+instance）= --append-system-prompt（追加在 pi 默认提示词后，保基础行为）；think/research/exporter = --system-prompt replace（agent 文件模式）；skill 文件 = 路径引用 read / BuildPlan 内联；task 文本 = child user 消息。

**真机验证**：--append-system-prompt 注入的规则（RULE_ONE_OK）在 -p 会话中被正确引用（yes）。

**首次发现**: 设计决策（用户：协议必须 compaction 持久）/ v4.4.5  **验证状态**: ✅ 已实施（单测 + 真机注入验证；完整会话实测待下轮）

### v4.4.6：doing 实时执行反馈（长 job 不再零输出）+ gate helper 降级

**问题**（用户）：`rick doing job_N` 执行期间终端零反馈（pi --mode json 流被 rick 全吞）。
1. **实时进度流**（executor.go parseStream 加 progress 回调 → Run 打 stderr，单行精简）：
   - `▶ subagent 派发: task1-impl, task2-impl`（workflowScript 里提取 runs key）
   - `▶ bash/read/write <参数摘要>`、`💬 <assistant 文本截断>`、`✗ 工具失败`、`✅ pipeline_gate/level_complete 结果`
   - `✓ 会话收敛（agent_settled，7m0s，17 次工具调用）`
2. **doing.go 编排摘要**：每轮 Run 前打印 pending/total + 逐 task 状态（⬜/✅）；Run 后打印会话统计（耗时/工具数/失败数）
3. **gate helper 降级**（真机踩坑）：工作仓库无 `.rick/skills/rick-gates/`（非 rick 仓库 doing）→ helper.py 找不到导致 5 轮重试全败——降级链：工作仓库 → env 部署副本（~/.rick/pi/agent/extensions/rick-gates/）→ 都无则跳过终态兜底（层门禁是主验收）

**真机验收**（/tmp 最小 fixture 全链路）：pipeline_gate ✅ → 门禁红验证 → 派发 impl-worker（进度可见）→ level_complete commit 0e7cbdc → tasks.json success → 会话收敛 7m0s 统计 → `Job job_1 execution completed!`。

**首次发现**: 用户反馈 + 真机踩坑 / v4.4.6  **验证状态**: ✅ 已实施并真机验收

### v4.4.7：确定性进度信号（tasks.json watcher）替代不固定的 assistant 文本

**问题**（用户指出）：v4.4.6 的实时反馈 grep 会话日志（assistant 💬 文本）——**不固定**（模型说不说/怎么说都漂）。确定性进度信号只有两个：tasks.json 状态变更（hook 独写）与门禁事件（结构化工具调用）。
1. **删 💬 行**（assistant 自由文本不打点；finalMessage 采集保留供 trace）
2. **tasks.json watcher**（doing.go）：Run 期间 goroutine 每 2s 轮询 diff——状态变更打一行（✅ taskN 完成（commit xxx）/🔄 开始执行/＋ 新 task/状态迁移）；Run 返回时补一次终 diff（收尾写不被 tick 遗漏）
3. 保留结构化事件行：▶ 工具调用（subagent 派发 key 提取）/ ✅ 门禁结果 / ✗ 失败 / ✓ 收敛统计——这些是 pi 结构化事件（tool_execution_*），不是模型自由文本

**真机验收**：level_complete 批量写后 ≤2s 打出 `[rick] ✅ task1 完成（commit 1ebba1bb）`。

**首次发现**: 用户反馈（进度信号要确定性）/ v4.4.7  **验证状态**: ✅ 已实施并真机验收

### v4.4.8：门禁检查逻辑先行 + 多维评审（+自洽性维度）+ 并行优先分层 + main agent 主动监督

**四项协议改进**（用户驱动）:
1. **门禁检查逻辑先行**：每层门禁双产物——`gate{N}.md`（检查逻辑说明：逐条断言的检查对象/判定标准/依赖前提）+ `gate{N}.py`（实现）；**human 确认的是检查逻辑**（合理/覆盖/可达成），逻辑定稿后写代码——不是让人审代码
2. **「七维评审」→「多维评审」（8 维）**：新增 #8 门禁自洽性——断言互相矛盾（不可能同时满足）/永远红门禁（依赖不存在产物/超层范围/条件恒假——永远红=流水线死锁）/可达成性推演（层 task 全完成必然转绿）/md 与 py 一致；#7 加分层并行度（假依赖检查：能同层并行的被错误串行）
3. **并行优先分层**：只有真实依赖（数据/接口契约前置）才分层；无真实依赖必须同层——同层内尽量多拆写域独立 task 最大化 DAG 并行度
4. **main agent 主动监督**（doing.md）：最终目标=所有 subagent 真实完成——主动读轨迹（开始/中途/收尾巡检）理解行为；判断无法自行完成即干预（steer→stop+重派→亲自接手/上报 human）；human 的 ctrl 是 human-in-the-loop，自主运行时 main agent 兜底

**首次发现**: 设计决策（用户）/ v4.4.8  **验证状态**: ✅ 已实施（dry-run 渲染验证；真实 plan/doing 实测待观察）

### v4.4.9：全命令 --resume 会话恢复（pi --session-id 语义）

**功能**（用户）：所有 rick 命令支持 `--resume <id>` 重新进入之前的 pi 会话（完整历史+上下文）：
- `rick plan --resume job_2` → 恢复 job_2 的 plan 会话（此前的设计树/流水线讨论都在上下文）
- `rick human-loop --resume loop_1` → 恢复 loop_1 的 SENSE 会话（五阶段进度 + human 判断全在）
- `rick doing --resume job_1` → 交互式恢复 doing parent 会话（编排状态/worker 产出/debug 记录）——人工接管/排查入口
- `rick easy --resume job_N` → 已有（ResumeEasy），全局 --resume 打通别名

**实现**：pi `--session-id <uuid>` 语义 =「不存在则创建，存在则恢复」（幂等恢复原语）：
- handler/session.go 统一助手：ensureSessionID（复用或生成+落盘 `<dir>/session_id`）/ loadSessionIDStrict（带指引的读取）
- 各阶段启动即落盘：plan → plan/session_id；human-loop → loop_N/session_id；doing 已有（piRuntime.Run 返回值）；easy 已有
- 全局 flag：root.go PersistentFlags --resume（与 --job 并列）
- resume 时 CallCLI 传 `--session-id <id>`（human-loop/plan/doing 交互模式）

**真机验证**：human-loop 新会话落盘 loop_9/session_id → pi --session-id 恢复成功（「已恢复」）；plan --resume job_1 走通（Session ID 打印 + pi 创建/恢复语义）；doing/easy dry-run 路径正常。

**首次发现**: 功能需求（用户）/ v4.4.9  **验证状态**: ✅ 已实施并真机验证

### v4.4.10：learning/dream 的 domain 沉淀强化 + 执行链断点修复

**调研发现**（用户指出）：协议完备但执行链断——test 项目 6 个 job 只有 job_5 跑了 learning（domain 4 文件全是它写的），job_1~4 learning 从未执行、dream 一次没跑（dream 目录空）。根因：**doing 完成后零引导**（打印 completed 即退出，没人提示下一步 learning）。

**修复**:
1. doing.go：完成输出加下一步引导（`rick learning <job_id>` 单 job 沉淀 / `rick dream` 跨 job 反思，注明「不跑则 domain/skills 不更新，经验会丢」）
2. learning.md：开头加 domain 沉淀优先级声明（核心产出非附属；没有 domain 沉淀=同样的坑下次还踩；Step 5 不可跳过）；结尾加 dream 引导
3. dream.md：职责区加「domain 是核心产出（跨 job 共性只能 dream 发现）」；Step 1 加 **learning 完整性检查**（逐 job 查 learning/SUMMARY.md，缺失清单报告 human 决定补跑或跳过；处理清单带 ✅/⚠️ 标注）

**首次发现**: 执行链断点（用户调研请求）/ v4.4.10  **验证状态**: ✅ 已实施（dry-run 渲染验证）

### v4.4.11：easy 模式 Grilling 不派 research agent 的根因修复

**问题**（用户调研）：test 项目最新的 easy 会话（job_6 easy）Grilling 阶段**零 subagent 调用**——20 次 bash + 3 次 read，全是 parent 自己 grep/sed。weight 级调研（联网选型/跨领域/大范围考古）没有出口，全烧在 parent 上下文里。

**根因（两层）**：
1. easy 的 grilling 段只说「调研消解（可自己查的先查）」——grilling skill 的核心指令是 "explore the codebase instead of asking"，模型遵守了，把「调研」=自己 grep
2. **pibuilder.go 里有一个独立的 buildGrillingSection**（与 easy_prompt.go 的不同步）——dry-run 走 pibuilder 版（旧文案），真实运行走 easy_prompt.go 版。两个函数不同步导致修复不易发现

**修复**：
- easy_prompt.go + pibuilder.go 两处 buildGrillingSection 同步为：调研消解分工——轻量自查 vs 重量级必派 `agent:'research'`（含显式触发语法 + timeoutMs + 简报落盘 + 回执）
- grilling skill：核心指令加分工规则（与 plan 对齐）

**首次发现**: 协议缺陷 + 双函数不同步（pibuilder/easy_prompt 各一份 buildGrillingSection）/ v4.4.11  **验证状态**: ✅ 已实施（dry-run 渲染验证；真实 easy 实测待验证 research 派发）

### v4.4.12：grilling 编排协议单源化（skill 承载完整协议，三处薄壳引用）

**问题**（用户架构审视）：easy/plan/doing 只是外层 prompt 不同，grilling 应引用相同 skill、一致逻辑追问/调 subagent。源码事实：skill 内容单源 ✅（templates/skills/grilling.md → WriteSkillFile 三处同源落盘），但**编排协议三处独立维护** ❌——plan.md Step 4（14 行 L1-L5）、easy_prompt.go buildGrillingSection、pibuilder.go buildGrillingSection 各自演化（昨天 easy 不派 research + 双函数不同步两个 bug 皆源于此）。

**重构**：
1. grilling.md 升级为**完整编排协议单源**：OKR 设计树 + 五步下钻循环（L1 调研消解含 research 显式派发语法 / L2 提炼追问 / L3 批量追问 / L4 事实回流 / L5 终止判定含充分性自检）+ 调研分工 + 追问规范 + {{grilling_workdir}} 变量（research 简报/design-tree 落盘目录）
2. plan.md Step 4 与 easy 的 grilling 段薄壳化：「加载并完整执行 skill:grilling（唯一协议源，本段不重复协议）」
3. Go 侧收敛：BuildGrillingSection 唯一实现导出在 prompt 包；pibuilder 版改为委托（dry-run 与真实路径同一函数，杜绝漂移）
4. 落盘注入：WriteSkillFileWithVars/renderInlineSkillsWithVars 替换 {{grilling_workdir}}（plan→plan/grilling/，easy→doing/grilling/；内联路径同样替换）

**验证**：全量 test 绿；plan/easy dry-run 渲染一致（grilling 薄壳 + skill 内含 workdir 与 research 语法，零占位符残留）。

**首次发现**: 架构问题（用户）+ 重构 / v4.4.12  **验证状态**: ✅ 已实施

### v4.4.13：section builder 双份收敛（prompt 包唯一实现 + builder 全委托）

**架构确认**（用户心智模型）：每个 cmd 的外层阶段模板 = loop（编排节奏）；内部实际操作步骤中**可复用的部分**由 skill 承载——目标是「改一个 skill 模块，全场景 cmd 等价一致生效」。单消费者协议留模板（sense_loop=human-loop 本体、ctrl 监控协议、plan 多维评审、dream 验证流——经审计确认均为单消费者）。

**双份收敛**（grilling 双份不同步的同款问题全面排查）：prompt 包与 builder 包存在平行 section builder 函数群，全部收敛为 prompt 包唯一实现 + builder 委托：
- BuildRequirementSection / BuildSessionWrapSection / BuildCtxSection / BuildGrillingSection（v4.4.12 已做）/ LoadDoingLoopContent（原 loadDoingLoopContent vs renderDoingLoop 双份）——builder/pibuilder.go 五个函数全部改为 `return prompt.Xxx(...)` 一行委托
- 双份漂移在物理上不可能（dry-run 与真实路径跑同一函数）

**单源验证（探针法）**：grilling skill 插入 HTML 注释探针 → 重建 → plan/easy dry-run 均渲染（skill 引用段+内联段各 1，共 2 处/命令）；删除 → 均消失。**「改 skill 一处、全 cmd 等价生效」实证成立**。

**首次发现**: 架构收敛（用户）+ 探针验证法 / v4.4.13  **验证状态**: ✅ 已实施并验证

### v4.4.14：grilling 执行漂移的协议级修复（防「自查替代流程」）

**实测漂移**（用户 job_7 easy 会话）：模型把核心指令「能自查就别问（explore instead of asking）」泛化为「能自查就跳过整个流程」——不建设计树、不派 research、零提问、自查自答到底、把「速度 vs 强度」「目标阈值」等判断节点自行拍板。被用户质询后才忏悔纠偏。

**根因（三层）**：
1. 英文核心指令（explore instead of asking）位于五步协议**前面**，语义权重压过协议——被模型泛化为整体策略
2. 设计树/research 简报/提问**无强制产出物定义**——不做没有「违规感」
3. 「不得假设」在追问规范深处，对「吞判断节点」无针对性条款

**修复（grilling skill + 两处薄壳锚点）**：
1. 核心指令重写：exploring 只消解**事实性子问题**，明示「不替代层循环/不替代 research 派发/不替代对 human 的提问」+ 最高优先级纪律块（点名典型漂移形态，标注最严重协议违规）
2. 五步循环前加**强制产出物**：design-tree.md（活文档）/ research-L{N}.md（凡有不可自查问题必有）/ L3 批量追问（零判断节点须显式记录依据）——缺失=该层未执行，禁止下钻
3. 追问规范加反模式：「不得吞判断节点」（权衡/阈值/取舍是 human 裁决点，可推荐禁拍板）+「不得以自查替代流程」（连续纯自查零派发零提问→立即回协议）
4. plan/easy 薄壳加**执行锚点**：先 read skill 全文；第一动作=建树落盘（O+KR）

**首次发现**: 实测漂移（用户 job_7）/ v4.4.14  **验证状态**: ✅ 已实施（下轮 grilling 实测观察第一动作是否建树）

### v4.4.15：grilling 确定性门禁（grilling_gate）+ 指令精简

**设计**（用户：追问流程做成确定性门禁；核心指令只需强调按 loop 逐步推进）：
1. **grilling_gate 工具**（rick-gates hook 注册，grilling skill 的全局终止条件处调用）：确定性校验产出物——①design-tree.md 存在且含根层 OKR（O/KR 痕迹，全半角括号兼容）②分层结构 ≥1 ③每层有 research-L{N}.md 或树中显式「本层全部消解」④有提问痕迹（推荐/问题编号/判断节点）或显式全消解。任一缺失 ⛔ 给补齐指引；通过才准进实现流水线
2. **指令精简**（用户意见）：删冗余的「exploring never replaces...」三连声明，收敛为一句——「必须按每层追问流程（L1→L5 loop）逐步推进，不得跳步、不得以任何手段替代流程」+ 自查只是 L1 technique；薄壳锚点同步精简
3. grilling skill 的全局终止条件后接门禁调用（含显式语法）；单源生效（plan/easy 共用）

**真机验收**：场景1 无树 → ⛔「第一动作就是建树」；场景2 树无 OKR/无简报/无提问 → ⛔ 四项并列指引；场景3 OKR+分层+research-L1+第2层显式全消解+提问痕迹 → ✅ 通过。全角括号 O（目标）的正则兼容坑已修（O\s*[（(:：]）。

**首次发现**: 设计决策（用户）+ 实测 / v4.4.15  **验证状态**: ✅ 已实施并真机验收
