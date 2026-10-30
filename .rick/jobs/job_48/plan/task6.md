# task6：super 会话前端表单 v2（requirement + 可选标题，去 name）

## 任务目标

NewSessionModal 的 super 表单对齐 job 模式：**需求框必填**（textarea，照 easy/plan 的 requirement 输入形态）+ **可选会话标题**（title 输入，默认空 → 后端默认 `super env_{n}`）；去掉 name/工作区概念。

## 关键结果

- `web/src/components/sessions/NewSessionModal.tsx`：
  - super 表单：`requirement` textarea（必填、placeholder 示例：「优化 512×512 矩阵乘的性能，从纯 Python 三重循环起点开始进化」）+ `title` 输入（可选、placeholder「会话标题（默认 super env_N）」）
  - 删除 name 输入与相关校验（/ 与 .. 检查移除——无 name 了）
  - 提交 params：`{requirement, title?}`（title 空不传）
  - 副文案：「需求将先经设计树追问（grilling）理清，再生成 env 并启动进化（.rick/super/env_N/）」
- `web/src/routes/Sessions.tsx` / `ChatView.tsx`：super 徽章/渲染不变（env_N 路径提示保留——后端 title 会带）
- api-client：无新端点（title 是 sessions 通用字段）
- 构建：`cd web && npm run build` 通过

## 写域

- web/src/components/sessions/NewSessionModal.tsx

## 依赖关系

- task5（后端参数签名 requirement/title 就位）

## 实现要点

- textarea 照 easy/plan 的 requirement 输入样式复用（找到既有实现直接同型）
- title 留空时不传（后端默认）；用户填了传 `title`
- 后续改名走既有 SessionSettingsDialog（已支持，无需改）

## 测试方法

`cd web && npm run build` 通过；手工冒烟：super 表单出现 requirement textarea + title 输入；requirement 空时提交禁用

## 禁止事项

- 不动其它会话类型表单；不做监控面板
