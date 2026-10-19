# task3：前端 super 会话表单与渲染

## 任务目标

NewSessionModal 支持 super 类型（name 输入），Sessions 列表/SessionPage 正确渲染 super 会话。

## 关键结果

- `web/src/components/sessions/NewSessionModal.tsx`：
  - SESSION_TYPES（或同型数组）加 `super`：label/description（「super 进化中控：env 生成/启停/观察/干预」）；cmdType=super 时渲染 `name` 文本输入（placeholder：工作区名，默认 default；提示：env 将生成于 .rick/super_env/<name>/）
  - 校验：name 可空（默认 default）；非空合法字符串即可（不含路径分隔符防注入：`/` 与 `..` 拒绝）
- `web/src/routes/Sessions.tsx` + `web/src/components/chat/ChatView.tsx`：
  - super 类型徽章/边色（照 human-loop 样式新增一档，如 amber 系）；ChatView 的 method 提示文案加 super 行（`<ws>/.rick/super_env/<name>：super-loop + gen_env skill 中控`）
- `web/src/api/client.ts`：类型与请求无需新端点（POST /api/sessions 既有），若有 session type 枚举补 "super"
- 构建：`cd web && npm run build` 通过；无 TS 错误

# 写域

- web/src/components/sessions/NewSessionModal.tsx
- web/src/routes/Sessions.tsx
- web/src/components/chat/ChatView.tsx
- web/src/api/client.ts（若需）

## 依赖关系

- task2（类型注册就位，前端表单才能提交成功）

## 实现要点

- 表单字段名与后端 params 对齐（`name`）
- 徽章样式照现有类型映射表扩展，避免新造设计语言
- super 无需额外必填字段——最低摩擦创建（一键 default）

## 测试方法

`cd web && npm run build`（产物落 <dev>/web/dist，dev 实例 overlay 即时生效）；手工冒烟：NewSessionModal 出现 super 选项、name 输入、创建跳转会话页

## 禁止事项

- 不做监控面板/SSE（J1=A）；不动 super 相关之外的前端模块
