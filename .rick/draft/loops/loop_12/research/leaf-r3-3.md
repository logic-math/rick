# Research: skill vs agent 的业界划分原则 + 开源生态

等级：A=一手规范/仓库原文；B=官方文档；C=线索（未深验）。

## 一、规范级定义

1. **Skill 的最小形态是「目录 + SKILL.md」**，可含 scripts/ references/ assets/。［https://agentskills.io/specification｜A］
2. **frontmatter 必填 name（≤64，小写字母数字连字符，须与父目录同名）与 description（≤1024，须说明「做什么 + 何时用」）；可选 license、compatibility（≤500）、metadata（string→string）、allowed-tools（空格分隔，标注实验性）。**［同 spec｜A］
3. **progressive disclosure 三级：① metadata ~100 tokens 启动即常驻；② SKILL.md 正文（建议 <5000 tokens、<500 行）在技能被激活时加载；③ 资源文件按需读取。**［spec｜A；Anthropic 工程博客同口径｜B］
4. **Agent（Claude Code subagent）= 独立上下文窗口 + 自定义 system prompt + 独立工具权限，只把摘要回传主会话**；定义方式为 markdown + frontmatter（name/description/tools）。［https://code.claude.com/docs/en/sub-agents｜B］
5. **官方 skill vs subagent 表述：「Skills add knowledge to your current conversation…Subagents run in a separate context」；技能是「培训材料」，子代理是「独立员工」；子代理可使用 skill。**［https://academy.claude.com/courses/introduction-to-agent-skills/skills-vs-other-claude-code-features｜B；https://claude.com/resources/articles/skills-explained｜B］
6. **skill vs MCP 分工：skill = 程序性知识（怎么做），MCP = 工具连通性/工具定义；prompt=一次性指令，Project=背景知识。**［skills-explained｜B］
7. **OpenAI Agents SDK 术语：Agent = LLM + instructions + tools（+handoffs/guardrails）；Tools 含 hosted、local、function tool、agents-as-tools；Skills 官方定位是「知识/过程包」，由模型自行决定如何用，与函数调用式 tool 不同。**［https://openai.github.io/openai-agents-python/agents/ 、/tools/｜B；维护者表述 https://github.com/openai/openai-agents-js/issues/1137｜B］
8. **Google ADK：Skill 依 Agent Skill 规范实现，经 SkillToolset 以 L1 元数据/L2 指令/L3 资源三级加载，实验性；Agent=LlmAgent 执行单元，Skill 作为 toolset 挂载。**［https://adk.dev/skills/｜B］
9. **开放标准现状：agentskills.io 为开放标准（原由 Anthropic 开发）；client showcase 列 45+/46 个产品；其中 11 家有自家文档明示读取 SKILL.md（Claude Code、Claude API、Cowork、Codex、Cursor、GitHub Copilot、VS Code、Gemini CLI、OpenCode、goose、Amp）。**［https://agentskills.io/clients｜B；https://www.skillsboard.sh/agent-skills-support｜C］
10. **pi 实现 Agent Skills 规范：启动扫描目录，仅把 name/description/path 注入 system prompt；匹配后读 SKILL.md；`/skill:name` 强制加载并追加参数；`disable-model-invocation: true` 仅允许命令触发；支持 ~/.agents/skills 与 .agents/skills。**［https://pi.dev/docs/latest/skills｜B］
11. **pi 自定义 agent = markdown 文件：YAML frontmatter（name/description/tools/defaultContext/thinking/timeoutMs/systemPromptMode）+ 下方 system prompt，作为子会话启动。**［/Users/logic/.rick/pi/agent/agents/teach.md、research.md｜A］
12. **本工作区实证：prompts/skill_teach.md 为无 frontmatter 的纯 markdown；agents/teach.md = 同一正文 + frontmatter 包裹（defaultContext: fresh、systemPromptMode: replace、tools 白名单、timeoutMs）。即同一文本两用，差别只在「是否新开会话 + 工具白名单」。**［skill_teach.md 首行自述「本 skill 同时是 pi 自定义 agent teach 的 system prompt」｜A］

### 业界判据清单

| 维度 | skill | agent | 出处 |
|---|---|---|---|
| 触发方式 | description 语义匹配自动加载；或 `/skill:name` 显式调用 | 父会话显式派发；或按 description 自动委派 | B（pi.dev/skills；code.claude.com sub-agents；academy.claude.com） |
| 上下文加载 | 注入**当前**上下文；三级渐进加载（元数据常驻→正文→资源） | **独立**上下文窗口；只回传摘要 | B（academy/ skills-explained）；A（spec 三级） |
| 工具权限 | 不自带权限；allowed-tools 仅实验性预批准提示 | 自有 tools 白名单，可与父会话不同 | A（spec）；B（sub-agents） |
| 组合方式 | 可被任意 agent/子代理复用；可被多个 skill/agent 引用（一个层级引用深度） | 可被作为 tool 调用（agents-as-tools）；可再派子代理 | B（skills-explained「Can subagents use Skills? Yes」；OpenAI tools 页） |
| 状态 | 无会话状态、跨会话可移植 | 有独立会话状态、模型/超时/thinking 配置 | A（pi agents frontmatter）；B（sub-agents） |

## 二、开源生态（C 级线索，star/更新时间未深验）

1. **anthropics/skills** — 官方 SKILL.md 集合（academy-guide、algorithmic-art、brand-guidelines、canvas-design…，每个 skill 附 LICENSE.txt）；覆盖：能力示范/教学示例。［仓库结构｜A］
2. **obra/superpowers** — 组合式 skills 框架 + SDLC 方法论，含 brainstorming、subagent-driven-development；已出 Claude/Codex/Cursor/Devin/Antigravity/Hermes 多平台 plugin 目录；MIT；star 聚合页读数 ~30 万量级（存疑）。［C］
3. **ComposioHQ/awesome-claude-skills** — 索引型 awesome list；~7.7 万 star。［C］
4. **travisvn/awesome-claude-skills** — 含 skill-creator 交互式建技能；~1.5 万 star。［C］
5. **badlogic/pi-skills / pi-agent-config 包** — pi 生态 skill 包（code-review、debug 等）。［C］
6. **Tencent/WeSmartFlow** — agent-native 自适应学习框架：多 agent 协作做资料研究/章节规划/课件/练习 + 知识图谱与复习排程；Python+Vue+FastAPI；覆盖：大纲→课程→练习→追踪→复习。［C］
7. **zijinz456/OpenTutor** — 本地块状自适应学习工作台：上传材料→笔记/quiz/闪卡/自适应 tutor；MIT；~141 star。覆盖：内容生成→练习→复习。［C］
8. **nageshsinghc4/forge** — LangGraph 编排的 8 专用 agent 学习系统 + SM-2 间隔重复。覆盖：课程规划→教学→复习排程。［C］
9. **atifanawaz/acesat-multiagent** — Planner→Orchestrator→Specialist 层级 + 贝叶斯知识追踪（BKT）+ 前置依赖路由。覆盖：诊断→练习→追踪。［C］
10. **Git-me-Harish/Mentora** — BKT/DKT 掌握度 + SM-2 + LinUCB 推荐 + LangChain tutor。覆盖：追踪→复习→推荐。［C］
11. **bcefghj/multi-agent-education** — 5-agent mesh、事件驱动、苏格拉底式提问；MIT；~101 star。覆盖：个性化教学。［C］
12. **iliaskalalou/agora-agentic-tutor** — 6 agent 计划/教学/评估/诊断/自我重规划闭环；MIT。覆盖：大纲→教学→考核→重规划。［C］

## Gaps

- 本会话仅有 read/write，无 bash/ls，**未能执行 `which pi; pi --help; ls ~/.pi`**；pi 本地事实改由读 `~/.rick/pi/agent/agents/*.md`、`settings.json`（packages: pi-web-access、pi-subagents）与 pi.dev 文档佐证；`~/.pi` 是否存在、版本号未验。
- 生态 star 数据来自搜索聚合页，量级可能失真（如 superpowers ~30 万存疑），许可与最近提交未逐仓库核验。
- OpenAI Agents SDK 是否有独立「Skills」正式文档页未确认（/skills/ 返回 404），仅有 issue 内维护者表述与第三方 RFC 线索。
- 「46 vs 45+ 产品」计数因页面时点不同存在差异。
