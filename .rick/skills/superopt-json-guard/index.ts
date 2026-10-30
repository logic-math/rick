// superopt-json-guard — superopt agent 的 JSON 输出确定性 hook。
//
// 问题（env_11 实测）：agent 完成实验但最终回复缺少 JSON 契约块 →
// client 层标 valid=False → 触发 debug 重试 → 真实工作轮 ~40% 浪费；
// adjust 的 5 项决策曾因「想对了没发出去」丢失（≥6 历史判例）。
//
// 方案（实测验证过的 pi 运行时机制，见 agent-loop.js）：
//   hook 挂在 agent_end（agent 运行结束、尚未 settled）→ 无 JSON 时
//   sendUserMessage(deliverAs:"followUp") 进 agent 的 followUp 队列 →
//   agent-loop 主循环检查 getFollowUpMessages() 非空 → continue 续跑
//   修复轮（同一进程同一会话、同一次 _runAgentPrompt）→ print 模式的
//   prompt() 自然等它完成才 resolve。这是运行时注释明示的官方模式：
//   "Any messages here were queued by agent_end extension handlers and
//    need a continuation."（agent_settled 里再发消息会撞 stale-ctx
//   重入问题——实测 agent_settled 路径嵌套运行死在半路）。
//
// 激活条件（防误伤）：仅当环境变量 SUPEROPT_JSON_GUARD=1（superopt 的
// PiAgentClient 派发 pi 子进程时注入；其它 pi 会话完全不注册本逻辑）。
// JSON 检查：```json ... ``` 合法代码块，或裸 JSON 对象。
//
// 部署：<repo>/.rick/skills/superopt-json-guard/ → <agentDir>/extensions/
// superopt-json-guard/（internal/env/customizations.go 自动复制）。

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/** 最多修复几次（防无限循环：修复轮结束又触发 agent_end 本 handler） */
const MAX_JSON_REPAIR = 2;

function isActive(): boolean {
	return process.env.SUPEROPT_JSON_GUARD === "1";
}

/** 检查文本是否包含合法的 JSON 代码块或裸 JSON 对象 */
function hasJsonBlock(text: string): boolean {
	if (!text || !text.trim()) return false;
	const m = /```json\s*\n([\s\S]*?)```/.exec(text);
	if (m) {
		try {
			JSON.parse(m[1]);
			return true;
		} catch { /* fence 内不是合法 JSON */ }
	}
	const trimmed = text.trim();
	if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
		try {
			JSON.parse(trimmed);
			return true;
		} catch { /* 裸对象不合法 */ }
	}
	return false;
}

function textOf(content: unknown): string {
	if (typeof content === "string") return content;
	if (Array.isArray(content)) {
		return content
			.filter((c): c is { type: "text"; text: string } =>
				typeof c === "object" && c !== null && (c as { type?: string }).type === "text")
			.map((c) => c.text ?? "")
			.join("");
	}
	return "";
}

export default async function activate(api: ExtensionAPI): Promise<void> {
	if (!isActive()) return; // 非 superopt 派发的 pi：完全不注册

	let repairCount = 0;
	let lastAssistantText = "";

	// 追踪最后一条 assistant 消息（一次 agent 运行内含工具调用的中间
	// 消息也会触发 message_end——但后续消息会覆盖，agent_end 时刻的
	// lastAssistantText 就是本轮最终回复）
	api.on("message_end", (event) => {
		if (event.message?.role === "assistant") {
			lastAssistantText = textOf(event.message.content);
		}
	});

	api.on("agent_end", async () => {
		const text = lastAssistantText;
		if (hasJsonBlock(text)) {
			repairCount = 0; // 合规 → 重置（供后续失约计数）
			return;
		}
		if (repairCount >= MAX_JSON_REPAIR) {
			return; // 上限内修不好 → 放行（client 层 R2 产物抢救兜底）
		}
		repairCount++;
		// deliverAs:"followUp"：agent_end 时刻 agent 尚在运行（isStreaming），
		// 消息进 followUp 队列 → agent-loop 主循环 continue 续跑修复轮
		await api.sendUserMessage(
			`你的最终回复缺少必需的 JSON 输出块。请按系统提示词的输出契约，只输出那个 JSON 代码块（\`\`\`json ... \`\`\`），不要任何其它内容。（JSON 输出修复第 ${repairCount}/${MAX_JSON_REPAIR} 次）`,
			{ deliverAs: "followUp" },
		);
	});
}
