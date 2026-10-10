# job_44 试用指引 + 教学成果评审 rubric（KR2 / KR4e）

> 本文件两部分：**① 给 human 的 8414 试用指引**（如何起步一次新 human-loop）；**② 5 维 rubric 评审表**（供 S4 human 在 dev 实例试运行后填分）。结论区留空，等 human 填。

---

## ① 试用指引：在 dev 实例（8414）起步一次新 human-loop

前置：dev 实例已起。快速确认：

```bash
curl -s --noproxy '*' http://127.0.0.1:8414/api/health
# => {"status":"ok","build_id":"57f599c-261010152340","started_at":"..."}
```

1. **打开界面**：浏览器访问 `http://127.0.0.1:8414/?token=devtok-b8512037`
   （dev token 亦可从 `<dev-home>/DEV_TOKEN` 读取；`<dev-home>=/workdir/sunquan20/rick-dev-home`）。
2. **选工作区**：dev 实例已注册工作区 `rick-dev`（id `8b549c5f`，路径 `/workdir/sunquan20/rick-dev`）。左侧栏选它。
3. **新建会话**：新建会话类型选 **human-loop**，填入一个**真实主题**（`topic`），例如「SENSE 教学法如何用于一次真实的架构决策」。
4. **mission 面试（首个环节）**：因 `mission.md` 缺失，`sense_loop` 会进入面试——teaching agent 会问你的学习目标 / 考核标准 / 起点。如实回答，产出落 `<ws>/.rick/draft/loops/loop_N/mission.md`。
5. **教学循环（每回合一课）**：agent 依次产出
   - `research/research_{N}.md`（事实材料：尽调树 + 信源等级 + `## Gaps`）
   - `lessons/teach_{N}.md`（教学节：一课一事 + 一次有形收益 + 引注事实材料）
   在会话里继续对话推进；**你的每次表态/判断原话**记入 `judgment.md`（表达即进度）。
6. **查看学习材料**：左侧栏 **Humans** 页面浏览 `.rick/draft/` 下的学习材料与进度（只读）。书写草稿在**刷新/重载/中断**后不丢（localStorage 按会话持久化）。
7. **收尾**：mission 学习目标达成且你明示完成后收尾（**不再产 RFC**）。需要总结时另起 easy/plan job 消费该 human-loop 工作文件夹。
8. **中途挂起**：dev 实例 `restart` 后会话进入「挂起」；刷新页面后逐条点「恢复继续」即可（**不自动续跑**）。

> ⚠️ 试用对象是 **8414 dev 实例**，**不要**动生产 8413。试用结束如实填下方 rubric，并在 `doing/rsi/approval.md` 决定是否放行 S4。

---

## ② 教学成果评审 rubric（5 维 × 1-5 分）

评分对象：本次 dev 实例试运行产生的 `loop_N/`（mission + research + lessons + judgment）。
每维 1-5 分（1=很差，3=达标，5=优秀），并填一句证据。

| 维度 | 1 分 | 3 分（达标） | 5 分（优秀） | 得分 | 证据 / 备注 |
|---|---|---|---|---|---|
| 教学流畅度 | 机械跳步、答非所问 | 节奏连贯、每回合一课、无硬性阶段跳转感 | 教学如真人导览，认知负荷低、环环相扣 |  |  |
| SENSE 内化质量 | 只贴标签、无思考痕迹 | 还原事实/呈现视角/洞察矛盾/启发逆转/良质批判 隐式贯穿 | 五关注面自然内化于叙事，读者被带动完成 SENSE |  |  |
| 事实锚定 | 结论无出处、凭空断言 | 关键论断有引注、区分事实与推断 | 每条事实可溯源到 primary source，Gaps 交代清楚 |  |  |
| 提问启发性 | 只灌输、不请读者表态 | 呈现分析前先请 human 表态（认知强制） | 提问直击要害，能引发读者自己的判断与逆转 |  |  |
| 学习成果质量 | 学不到可迁移的东西 | mission 目标达成、有可见收获 | 形成可复用的心智模型，能迁移到新问题 |  |  |

## ③ 结构指标（客观，逐条勾选）

- [ ] `lessons/` 教学节数量 `>= 3`（实际：___）
- [ ] `judgment.md` human 判断条数 `>= 5`（实际：___）
- [ ] 每一课都**引注**了对应 `research/research_{N}.md` 的事实（抽查：___）
- [ ] 全程无机械阶段标签/固定追问集/三维打分等已删方法论残留

## ④ 结论区（**留空，由 human 填**）

- 总评（是否达成 mission 学习目标）：
- 主要优点：
- 主要问题 / 改进点：
- S4 放行意见（对应 `doing/rsi/approval.md`）：
