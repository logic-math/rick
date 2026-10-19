# task5：super 会话目录模型 v2（.rick/super/env_{n} + requirement + grilling 追问）

## 任务目标

super 会话去掉「工作区名」概念：自动编号目录 `.rick/super/env_{n}/`（照 job_N 扫描模式）；表单改为 **requirement 必填** + 可选 title；requirement 作为使命注入中控会话 prompt；**复用 grilling skill**（追问树）——super 会话先跑设计树追问理清需求，再进 env 创建流程。

## 关键结果

- **目录模型**（internal/handler/super.go + internal/web/sessions.go）：
  - superWsRoot 的路径段 `super_env` → `super`（即 `<工作区>/.rick/super/`）
  - web prepareInteractive(super)：参数改为 `requirement`（必填非空字符串）+ `title`（可选）；**不再接受 name**（校验层：name 出现即忽略或 invalid——按 job 模式惯例）
  - 自动编号：scanNextEnvID（照 scanNextJobID：读 `.rick/super/` 下 `env_%d` 取 max+1；空则 env_1）
  - 会话工作目录 = `.rick/super/env_{n}/`：写 `requirement.md`（需求原文）；init 骨架（super.json 等照旧 init 于该目录）
  - prep：title 默认 `super env_{n}`（用户填了 title 用用户的）；persistDir=`.rick/super/env_{n}/sessions/<uuid>`
- **中控会话 prompt 增强**（internal/super/session.go BuildSuperPrompt）：
  - 新增参数/上下文：requirement（使命——本次进化任务的目标描述，requirement.md 路径）+ grilling 工作目录约定（`.rick/super/env_{n}/grilling/`）
  - **inline grilling skill 全文**（embedded 模板 internal/prompt/templates/skills/grilling.md，照 gen_env 的 readInline 模式——从 Go embedded 读取；变量 `{{grilling_workdir}}` 替换为 `.rick/super/env_{n}/grilling`）
  - prompt 流程段：**第一步 grilling 追问**（对 requirement 跑设计树追问循环，直到需求树落盘达标）→ 第二步 gen_env（env 生成落 `.rick/super/env_{n}/env/`）→ 第三步运行/观察/干预（既有内容不变）
- **严格加载**：grilling skill 是 embedded（编译期内嵌，永不缺失）——不需运行时预检；super-loop/gen_env skill 运行时预检保持（task2 已做）
- CLI `rick super` 五命令：`--name` 语义不变（env_N 也是合法 name——web 创建的 env_1 用 `rick super start --name env_1` 即达同一目录，天然互通）
- 测试：
  - scanNextEnvID 用例（空目录→env_1；env_1,env_3 存在→env_4）
  - prepareInteractive(super) 新签名：requirement 必填校验 / requirement.md 落盘 / prompt 含 requirement 原文 + grilling skill 关键词（「设计树」「追问」）+ `.rick/super/env_{n}` 路径
  - superWsRoot 路径段变更的既有用例适配
  - BuildSuperPrompt：TestBuildSuperPromptWorkspaceContext 增 grilling 段断言

## 写域

- internal/handler/super.go
- internal/web/sessions.go
- internal/web/registry.go（super 参数校验改 requirement/title）
- internal/super/session.go
- internal/handler/super_test.go、internal/web/*_test.go、internal/super/session_test.go（适配+新增）

## 依赖关系

- 无（基于 task1-4 已交付之上；job_48 迭代 2）

## 实现要点

- grilling.md 模板的 `{{grilling_workdir}}` 占位替换：readInline 后做字符串替换（照 super-loop inline 的既有处理模式）
- requirement.md 与 super.json 同级（`.rick/super/env_{n}/requirement.md`）
- 旧 `.rick/super_env/` 目录不迁移（dev 阶段无生产数据）
- 前端不动（task6 做）；本 task 后端先兼容旧 name 参数（web 前端发 requirement，CLI 无涉）

## 测试方法

`go build ./... && go test ./internal/handler/ ./internal/web/ ./internal/super/ -timeout 900s -skip TestReleaseCmd.*` 全绿

## 禁止事项

- 不动 py 调度器与 super CLI 五命令语义（--name 仍可用）；不迁移旧目录
