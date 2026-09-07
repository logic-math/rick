# 依赖关系
无

# 写域
web/src/api/client.ts
web/src/components/chat/ChatView.tsx
web/src/components/chat/ChatInput.tsx
web/src/components/chat/SessionInfoPanel.tsx（或现有会话信息组件）

# 任务目标
前端模型切换 UI（对应后端新接口；后端并行 task 在实现，契约如下，前端按契约调用）。

# 契约（后端 task-model-backend 实现，本 task 直接调用）
- `GET /api/sessions/{id}/models` → 200 {"models":[{id,name,provider}...],"current":{...} | null}
  - active 且有 worker → 200 模型列表；worker 不可用 → 409
- `POST /api/sessions/{id}/model {provider, model_id}` → 202
- `POST /api/sessions/{id}/thinking {level}` → 202

# 关键结果

1. `web/src/api/client.ts` 加：`getSessionModels(id)` / `setSessionModel(id, provider, modelId)` / `setSessionThinking(id, level)`（类型 SessionModel{id,name,provider}）
2. **ChatView 会话头部/信息区加「模型」交互**：
   - 会话信息区「运行时」的模型行改为可交互：显示当前模型名 + 「切换」按钮
   - 点击「切换」→ 弹 Dialog（复用 common/Dialog）：加载 GET models（loading 态）→ 模型列表（radio 选择：name + provider + 当前项 ✓）→ 底部「切换」按钮 → POST model → 成功后更新显示 + 刷新模型列表；thinking 档位选择（若 models 接口或单独逻辑支持——可选：显示 thinking 档位切换（off/low/medium/high 等）→ POST thinking）
   - 409（worker 不可用）→ 显示「会话不在运行（刷新后重连 worker 后可用）」降级提示，不崩溃
3. **ChatInput 斜杠命令 `/model`**：输入 `/model` 发送时拦截为打开模型切换 Dialog（不做文本发送）；支持 `/model <name>` 直接切换（模糊匹配模型名，匹配到唯一则直接切，多则打开选择）
4. 无页面错误；优雅降级（后端接口不可用 → 按钮禁用 + 提示）

# 测试方法
npx tsc --noEmit + npm run build；playwright 冒烟（6173 验证实例）：打开真实 plan 会话 → 会话信息 → 点「切换」→ Dialog 出现（若 409 则显示降级提示——两种情况都断言不崩溃）。

# 上下文提示
- 会话信息组件在 web/src/components/chat/ChatView.tsx 的「会话信息」Collapse 区（meta.model 已有显示——把静态文本改交互）
- Dialog 已 Portal 全屏；Button/Spinner 在 common/
- 移动端可用（Dialog 自适应）
- 模型列表可能为空/接口 409——都要降级处理
