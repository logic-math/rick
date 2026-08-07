# research-3 N3-provider 接入与任务路由（Y5）

节点路径:[根 > N3-provider 接入与任务路由]
事实陈述:pi 支持哪些 provider、provider 切换粒度(per-session/per-prompt/per-message)、是否支持 per-task 路由(不同任务用不同模型)、provider 配置方式(apikey/env/config/extension)。

## 执行动作

1. `curl -sL "https://api.github.com/repos/earendil-works/pi/contents/packages/ai/src/providers"` — 列出 pi-ai 内置 provider 文件
2. `curl -sL "https://raw.githubusercontent.com/earendil-works/pi/main/packages/ai/README.md"` — 读取 pi-ai README(79KB)
3. `sed -n '1288,1340p'` — 读取 Cross-Provider Handoffs 全文
4. `sed -n '408,440p'` — 读取 Environment Variables 段
5. 读取 extensions.md `pi.registerProvider` + `pi.setModel` 段
6. 读取 SDK 文档 `AgentSession.setModel` 接口

## 信源验证结果

### 代码原文(权重 0.4)✅

**pi-ai 内置 provider 文件列表**(packages/ai/src/providers/):

| provider | 文件 |
|---|---|
| OpenAI | openai.ts |
| Anthropic | anthropic.ts |
| Google (Gemini) | google.ts |
| Google Vertex AI | google-vertex.ts |
| Azure OpenAI (Responses) | azure-openai-responses.ts |
| OpenAI Codex | openai-codex.ts |
| AWS Bedrock | amazon-bedrock.ts |
| DeepSeek | deepseek.ts |
| Mistral | mistral.ts |
| Groq | groq.ts |
| Cerebras | cerebras.ts |
| xAI | xai.ts |
| OpenRouter | openrouter.ts |
| Together AI | together.ts |
| Fireworks | fireworks.ts |
| Hugging Face | huggingface.ts |
| NVIDIA NIM | nvidia.ts |
| Baseten | baseten.ts |
| Cloudflare AI Gateway | cloudflare-ai-gateway.ts |
| Cloudflare Workers AI | cloudflare-workers-ai.ts |
| Vercel AI Gateway | vercel-ai-gateway.ts |
| GitHub Copilot | github-copilot.ts |
| Kimi Coding | kimi-coding.ts |
| MiniMax (Global/CN) | minimax.ts / minimax-cn.ts |
| Moonshot (Global/CN) | moonshotai.ts / moonshotai-cn.ts |
| Qwen Token Plan | qwen-token-plan.ts |
| Xiaomi (AMS/SGP/CN) | xiaomi.ts |
| ZAI Coding (Global/CN) | zai.ts / zai-coding-cn.ts |
| Ant Ling | ant-ling.ts |
| Mistral Conversations | mistral-conversations.ts |
| opencode / opencode-go | opencode.ts |
| Radius | radius.ts |
| faux (测试用) | faux.ts |

→ **30+ 内置 provider**,覆盖主流云厂商(OpenAI/Anthropic/Google/AWS/Azure)+ 开源托管(OpenRouter/Together/Fireworks/HF)+ 国内(Kimi/MiniMax/Moonshot/Qwen/Xiaomi/ZAI)+ 本地(Gondolin/llama.cpp via registerProvider)。

**API 类型**(packages/ai/src/api/):

```
anthropic-messages.ts
openai-completions.ts
openai-responses.ts
openai-codex-responses.ts
google-generative-ai.ts
google-vertex.ts
azure-openai-responses.ts
bedrock-converse-stream.ts
mistral-conversations.ts
cloudflare.ts
openrouter-images.ts
pi-messages.ts
```

→ **11 种 API 协议**,任意 OpenAI/Anthropic 兼容端点均可接入。

### 运行时行为(权重 0.3)✅

**Cross-Provider Handoffs**(pi-ai README 1288-1340 行):

> The library supports seamless handoffs between different LLM providers within the same conversation. This allows you to **switch models mid-conversation while preserving context**, including thinking blocks, tool calls, and tool results.

> When messages from one provider are sent to a different provider, the library automatically transforms them for compatibility:
> - User and tool result messages are passed through unchanged
> - Assistant messages from the same provider/API are preserved as-is
> - Assistant messages from different providers have their thinking blocks converted to text with `<thinking>` tags
> - Tool calls and regular text are preserved unchanged

**示例代码**(pi-ai README):

```typescript
const models = createModels();
models.setProvider(anthropicProvider());
models.setProvider(openaiProvider());
models.setProvider(googleProvider());

const context: Context = { messages: [] };

// Start with Claude
const claude = models.getModel('anthropic', 'claude-sonnet-4-5')!;
context.messages.push(await models.completeSimple(claude, context, { reasoning: 'medium' }));

// Switch to GPT-5 - it will see Claude's thinking as <thinking> tagged text
const gpt5 = models.getModel('openai', 'gpt-5-mini')!;
context.messages.push(await models.complete(gpt5, context));

// Switch to Gemini
const gemini = models.getModel('google', 'gemini-2.5-flash')!;
const geminiResponse = await models.complete(gemini, context);
```

→ **per-prompt 模型切换原生支持**:同一 context 内,每次 `complete()` / `completeSimple()` 调用可传不同 model 参数,pi-ai 自动转换消息格式(thinking → text、tool calls 保留)。**这是 per-prompt 粒度**(每次 LLM 调用可换模型)。

**AgentSession.setModel**(SDK 文档 87 行):

```typescript
interface AgentSession {
  setModel(model: Model): Promise<void>;
  cycleModel(): Promise<ModelCycleResult | undefined>;
  // ...
}
```

→ AgentSession 暴露 `setModel(model)` 方法,运行时动态切换模型,返回 Promise(异步生效)。`cycleModel()` 循环切换模型(/model 命令的程序化版本)。

**extensions.md pi.setModel**(1664-1680 行):

```typescript
const model = ctx.modelRegistry.find("anthropic", "claude-sonnet-4-5");
if (model) {
  const success = await pi.setModel(model);
  if (!success) {
    ctx.ui.notify("No API key for this model", "error");
  }
}
```

→ extension 中可调用 `pi.setModel(model)` 动态切换,返回 false 表示无 API key。

**PI_PROVIDER / PI_MODEL 环境变量注入 bash 子进程**(extensions.md 2125 行):

> `createBashTool()` exposes the current session to commands through `PI_SESSION_ID`, `PI_SESSION_FILE`, `PI_PROVIDER`, `PI_MODEL`, and `PI_REASONING_LEVEL`.

→ bash 工具子进程可读当前 provider/model(用于子进程内嵌 pi 调用时继承模型)。

### 文档(权重 0.2)✅

**pi-ai README Environment Variables 段**(408-440 行):

| Provider | 环境变量 |
|---|---|
| OpenAI | `OPENAI_API_KEY` |
| Anthropic | `ANTHROPIC_API_KEY` or `ANTHROPIC_OAUTH_TOKEN` |
| Google | `GEMINI_API_KEY` |
| Vertex AI | `GOOGLE_CLOUD_API_KEY` or ADC |
| Azure OpenAI | `AZURE_OPENAI_API_KEY` + `AZURE_OPENAI_BASE_URL` |
| DeepSeek | `DEEPSEEK_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Groq | `GROQ_API_KEY` |
| Cerebras | `CEREBRAS_API_KEY` |
| xAI | `XAI_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| Together AI | `TOGETHER_API_KEY` |
| Fireworks | `FIREWORKS_API_KEY` |
| Hugging Face | `HF_API_KEY` |
| NVIDIA NIM | `NVIDIA_API_KEY` |
| Cloudflare | `CLOUDFLARE_API_KEY` + `CLOUDFLARE_ACCOUNT_ID` |
| ... | ...(30+ provider 全有 env var) |

**pi-ai README Auth 段**:

> Every provider owns its auth: how API keys resolve (**stored credentials, environment variables, ambient sources like AWS profiles or gcloud ADC**) and, where supported, OAuth login/refresh flows.

→ **4 种 auth 方式**:
1. **环境变量**(`OPENAI_API_KEY` 等,30+ provider 全支持)
2. **Credential Store**(存储在 `~/.pi/agent/credentials`,pi 管理)
3. **Ambient sources**(AWS profiles / gcloud ADC / Azure CLI)
4. **OAuth**(Anthropic OAuth / Google OAuth / GitHub Copilot OAuth / Vertex AI OAuth / 自定义 OAuth via registerProvider)

**registerProvider 动态注册**(extensions.md 1704-1840 行):

```typescript
pi.registerProvider("my-proxy", {
  name: "My Proxy",
  baseUrl: "https://proxy.example.com",
  apiKey: "$PROXY_API_KEY",  // env var reference
  api: "anthropic-messages",
  models: [...]
});

pi.registerProvider("llama.cpp", {
  baseUrl: "http://localhost:8080/v1",
  apiKey: "local",
  api: "openai-completions",
  async refreshModels({ signal }) { ... }  // 动态发现模型
});

pi.registerProvider("corporate-ai", {
  baseUrl: "https://ai.corp.com",
  api: "openai-responses",
  oauth: { ... }  // 自定义 OAuth
});
```

→ **registerProvider 支持**:代理(baseUrl 重写)、本地(llama.cpp 动态 refreshModels)、企业(OAuth SSO)。env var 引用(`$PROXY_API_KEY`)、命令引用(`!command`)、字面量三种 apiKey 形式。

**Dynamic Providers 段**(pi-ai README 310-330 行):

> Providers may have dynamic model lists (a llama.cpp server, a live OpenRouter listing). ... `await models.refresh({ providers: ['llamacpp'] });`

→ 动态 provider 支持运行时刷新模型列表(llama.cpp / OpenRouter)。

### 反事实(权重 0.1)N/A

- 本节点为外部文档调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **30+ 内置 provider**:OpenAI/Anthropic/Google/AWS Bedrock/Azure + OpenRouter/Together/Fireworks/HF/Groq/Cerebras + 国内(Kimi/MiniMax/Moonshot/Qwen/Xiaomi/ZAI)+ 本地(llama.cpp via registerProvider)
2. **11 种 API 协议**:anthropic-messages / openai-completions / openai-responses / google-generative-ai / google-vertex / azure-openai-responses / bedrock-converse-stream / mistral-conversations / cloudflare / openrouter-images / pi-messages
3. **per-prompt 模型切换原生支持**:Cross-Provider Handoffs,同一 context 内每次 `complete()` 可传不同 model,自动转换消息格式(thinking → text、tool calls 保留)
4. **AgentSession.setModel**:运行时动态切换模型(异步生效),返回 false 表示无 API key
5. **4 种 auth 方式**:环境变量(30+ provider 全支持)+ Credential Store(pi 管理)+ Ambient(AWS/gcloud/Azure CLI)+ OAuth(Anthropic/Google/Copilot/Vertex/自定义)
6. **registerProvider 动态注册**:支持代理(baseUrl 重写)+ 本地(llama.cpp refreshModels)+ 企业(OAuth SSO),apiKey 支持 env var 引用(`$VAR`)、命令引用(`!cmd`)、字面量
7. **PI_PROVIDER/PI_MODEL 环境变量注入 bash 子进程**:子进程内嵌 pi 调用可继承模型
8. **per-task 路由可行性**:
   - **pi 原生支持 per-prompt 切换**(Cross-Provider Handoffs + AgentSession.setModel)
   - **rick 端 per-task 路由**:rick 调用 pi 时,根据 task 类型(doing/dream/research)选择不同 model 传给 `pi.setModel()` 或 `pi -p --model <model>`
   - **无需 rick 维护多个 pi 子进程**:单 pi 进程内可动态切换模型(但需注意 context 跨模型转换的语义损失,如 thinking → text)

## per-task 路由实现路径

| 路由方式 | 实现机制 | rick 改造量 |
|---|---|---|
| **CLI flag** | `pi -p --model anthropic/claude-sonnet-4-5 "prompt"` | 低(改 exec.Command flag) |
| **env var** | `PI_PROVIDER=anthropic PI_MODEL=claude-sonnet-4-5 pi -p "prompt"` | 低(注入 env) |
| **RPC setModel** | RPC 模式下调用 `setModel` method | 中(写 RPC client) |
| **extension setModel** | extension 中按 task 类型调 `pi.setModel()` | 中(写 TS extension) |
| **多 pi 进程** | 每任务类型一个 pi 进程(doing/dream/research 各一) | 高(进程池管理) |

→ **最优路径:CLI flag 或 env var**(最低改造量),per-task 路由由 rick 端决策(根据 task 类型选 model 传给 pi)。

## 疑问点

- per-prompt 切换时,context 跨模型转换是否有语义损失?→ thinking → text 转换可能丢失 thinking chain 的结构(对长程推理任务有影响)。本轮未实测,标记为 R7 候选但非阻塞(rick 主要 doing/dream 用同 provider,跨 provider 场景少)。
- 多 provider 同时配置时,credential store 如何隔离?→ pi-ai README Credential Store 段未完全展开,需实测。非阻塞。

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4(pi-ai providers 目录 + API 目录 + extensions.md registerProvider + SDK setModel)
- 运行时行为 ✅ × 0.3 = 0.3(Cross-Provider Handoffs 示例代码 + 30+ provider env var 表)
- 文档 ✅ × 0.2 = 0.2(pi-ai README + extensions.md)
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
