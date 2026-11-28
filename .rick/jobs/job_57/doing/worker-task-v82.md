# Worker 任务书：human-loop v8.2 增量（呈现去标签化——每个概念科普讲清楚）

## 工作目录

`/Users/logic/workdir/rick-dev`（dev/self-evolve，当前 HEAD 03ab25d）

## 背景

v8.1（03ab25d）给呈现层加了三态标注（已知结论/争议/知识边界）、张力标注、引导性问题标签。human 实测反馈：**不要这些结构化描述——每个概念，按科普风格描述呈现清楚即可**。呈现层核心：概念讲清楚（科普风格），其余框架全部去掉。

## 改动清单

### 1. `internal/prompt/templates/skills/teacher_polish.md`（重写「内容要求」节）

核心要求改为：**每个概念，按科普风格描述呈现清楚**。

- 一个概念一个概念讲：每个概念用最少的句子讲清楚（是什么、为什么、例子）；引入任何概念附一句人话定义
- 内容的确定性状态自然融入行文（「这一点已被反复验证」「这一点仍有争议」「目前没人知道」——自然写出，不加标签）
- 保留：准确（事实性与来源不损失）、通俗、有逻辑、有趣味（具体例子/反差/数字）
- 删除：三态标注的结构化要求（「必须标注三态」节）

### 2. `internal/prompt/templates/skills/orig.md` 的「返回」节

- 删除「事实（带来源）+ 张力（可选）+ 引导性问题」三件套结构
- 改为：返回**科普风格的启发内容**——把检索到的概念讲清楚（每个概念是什么、为什么、例子，附来源）；与用户思考的关联和开放问题若自然带出就写，不做结构要求

### 3. 五个阶段 skill 的「启发呈现纪律」节（sense_reconstruct/perspective/contradiction/reversal/critique.md）

- 去掉张力标注（证实/证伪/扩展）与「引导性问题」标签化要求
- 改为：orig 返回的启发内容经教学润色后呈现——每个概念科普讲清楚；呈现后自然问用户一个开放问题（不写「引导性问题：」前缀，就是自然的一句话提问）

### 4. `internal/prompt/templates/sense_loop.md`

- 「呈现启发」环节：改为「orig 返回的启发内容经教学润色 skill 润色后呈现——每个概念按科普风格描述清楚（teacher_polish skill 为准），末尾自然问用户一个开放问题」

### 5. 测试同步

- v8.1 加的断言若含「知识边界」「张力」「引导性问题」等关键词：删除这些词的断言（保留 teacher_polish/教学润色/短句等核心断言）

## 验证

- [ ] `go build ./...`
- [ ] `go test ./internal/handler/ ./internal/cmd/ ./internal/prompt/ ./internal/builder/ ./internal/env/`
- [ ] `go run ./cmd/rick human-loop --dry-run <主题>`：占位符 0 残留；模板渲染产物不含「张力」「引导性问题」「知识边界」「已知结论」
- [ ] grep 八个 skill 文件（五阶段 + orig + teacher_polish + sense_loop）：无标签化表述残留

## Commit

单 commit：`refactor(human-loop): v8.2 呈现去标签化——每个概念科普风格讲清楚，三态/张力/引导性问题全部去掉`

## 完成回执

改动文件清单 + 验证结果。
