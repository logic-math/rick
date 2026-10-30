# 设计树：web super 中控会话（job_48 · grilling）

> 源需求：在 rick web 上开发 super 功能——专用 super 模式中控会话（严格加载 gen_env skill），用于启动/生成 env/运行/观察/干预 super 模式下多 agent 并行调度执行过程。
> 本树为活文档：逐层追加。

## 第 1 层（根层）：OKR

**O（目标，可验证表述）**：
web 上新增 `super` 会话类型：NewSessionModal 可创建（表单选 super + 工作区名），会话为 pi 交互会话、**严格加载 super-loop + gen_env skill**（缺失可见报错不静默），作为 super 模式运行的中控——human 在会话内可通过 agent 启动 super（start）、生成 env（gen_env skill 引导）、观察（status/events 查询）、干预（event 投递）；会话工件持久化于 super 工作区（sessions/<uuid>/），CLI `rick super --resume` 与 web 互通。

**KR 集**：
| # | KR | 充分性角色 |
|---|---|---|
| KR1 | 会话类型注册与校验：`super` 类型（参数 `name` 可选——空=默认工作区/列已有；`env_path` 可选）+ ValidateSessionRequest + NewSessionModal 表单（type=super 显示 name 输入） | web 入口 |
| KR2 | 会话生成：prepareInteractive 复用 BuildSuperPrompt（inline super-loop + gen_env skill + method SOP）——**严格加载**：两个文件缺失任一 → 创建失败并显式报错（不降级静默）；prep.persistDir=ws/sessions/<uuid>，title="super <name>" | 中控会话本体 |
| KR3 | CLI/web 互通：web 创建的 super 会话可被 `rick super --resume`（session_id 契约同 CLI SuperSession）；CLI 启动的会话在 web 会话列表可见（若可发现） | 一致性 |
| KR4 | 前端体验：SessionPage 正确渲染 super 会话（ChatView 交互）；Sessions 列表 super 类型徽章 | 可用性 |
| KR5 | 验收：Go 测试（registry 校验/prep 产物/spawn）+ 前端构建 + e2e 冒烟（创建 super 会话 → 会话 prompt 含 gen_env skill 全文 → resume 互通） | 门禁 |

**判断节点（L3 待裁决）**：
| # | 节点 | 选项 |
|---|---|---|
| J1 | **监控数据流形态**：中控会话内 agent 跑 CLI 查询（status/event tail）即可满足「观察」，还是要 web 层的实时事件流推送（SSE 新端点 `/api/workspaces/{ws}/super/{name}/events`）？ | A=仅会话内 CLI 查询（最小闭环）/ B=会话+独立 SSE 事件流端点（实时仪表）/ C=会话+专用 super 监控面板页（前端新页面） |
| J2 | **super 会话的工作区解析**：name 默认值（super_root 下已有工作区列表？默认名？）与 env 参数语义（指向已生成 env / 由会话内 gen_env 现场生成） | A=name 必填（明确工作区）/ B=name 空时默认 `default`（自动 init）/ C=name 空时列出已有让 human 选 |
| J3 | **干预路径**：会话内 agent 执行 `rick super event`（CLI）vs web API 直投队列（POST 端点） | A=会话内 CLI / B=两者都要 |
| J4 | **范围**：本期是否含 super 工作区管理 UI（列表/删除/归档工作区） | A=不含（下期）/ B=含最小列表 |

## 第 2 层：模块与接口（J 裁决后下钻）
（待 L3 裁决回流后填充：registry/prep/前端/（若 J1=B/C）SSE 端点的事件源=events.jsonl tail -f 等价实现）

---

## L3 裁决回流（2026-10-18）
- J1=A 纯会话（观察=会话内 agent 用 CLI + 直读文件系统任意信息：events.jsonl/archive.json/codes/*/pi agent session 等）
- J2=修正案：不绑定 super 工作区——任意 rick 工作区可启动 super；env 统一生成在 <ws>/.rick/super_env/<name>/
- J3=A 会话内 CLI（rick super event 等）
- J4=A 不含工作区管理 UI——统一到中控会话对话实现
- L2-Q1=A：CLI 一并切换——superWsRoot 从 ~/.rick/super → <当前工作区>/.rick/super_env（cwd 向上找 .rick；config default_super_dir 保留为显式覆盖向后兼容；旧 ~/.rick/super 测试工作区留作历史不迁移）

## 第 2 层：模块设计（终案）

### 模块图
```
web NewSessionModal(super 表单: name)
  → POST /api/sessions {type:"super", params:{name}}
  → ValidateSessionRequest（super: name 可选字符串）
  → prepareInteractive(super)：解析 ws 根 → .rick/super_env/<name>（init 骨架）
      → BuildSuperPrompt(rickDir=<ws>/.rick, ws=super_env/<name>)   [复用 CLI 现有]
      → prep{promptFile, methodFile, persistDir=sessions/<uuid>, title="super <name>"}
  → spawn pi 交互会话（ChatView 渲染）
中控会话（pi agent）：
  - 严格加载 super-loop + gen_env skill（BuildSuperPrompt inline，缺失创建失败显式报错）
  - 工作区根上下文：告知 <ws> 根 / super_env 路径 / env 生成约定 / 观察手段（CLI + 直读文件）/ 干预手段（CLI event）
CLI：superWsRoot() = <cwd 工作区>/.rick/super_env（cwd 向上找 .rick；config default_super_dir 显式覆盖）
```

### KR 终案（第 1 层表不变，KR2 细化 + 新 KR6）
- KR6（存储模型迁移）：CLI super 五命令的工作区解析切换到 <工作区>/.rick/super_env；~/.rick/config.json super.default_super_dir 保留显式覆盖；e2e/测试全部随迁
