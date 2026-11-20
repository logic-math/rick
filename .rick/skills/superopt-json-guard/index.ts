// superopt-json-guard — superopt agent 的确定性输出 hook。
//
// job_47/task57：JSON 输出保障（agent_end 检查最终回复缺 JSON 块 →
// sendUserMessage(deliverAs:"followUp") 续跑修复轮——agent-loop 主循环
// 官方 continue 模式）。
//
// job_56 task3（KR-E）扩展两类确定性检查：
//   1. adjust_space proposals schema 结构检：adjust_space 角色的最终 JSON
//      按 proposals 字段校验（type/target_cell+action/evidence 必填；
//      new_cell 必含 name/method/orthogonality_rationale）——缺字段走
//      既有 followUp 修复轮（复用 MAX_JSON_REPAIR 上限）。
//   2. gates/ 禁读拦截（tool_call 事件可 block）：builder/coding/debug
//      角色（SUPEROPT_ROLE 判定）的 read/bash 调用命中 <ws>/env/gates/
//      → block（返回禁读理由）。adjust_* 角色放行（step7 验证器反思需要
//      读改）。防作弊：验证器内容对被测 agent 保密（执行引用、内容不可读）。
//
// 激活条件（防误伤）：SUPEROPT_JSON_GUARD=1（superopt 的 PiAgentClient
// 派发 pi 子进程时注入）；SUPEROPT_ROLE 注入当前角色（job_56 task3）。
//
// 部署：<repo>/.rick/skills/superopt-json-guard/ → <agentDir>/extensions/
// superopt-json-guard/（internal/env/customizations.go 自动复制）。

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/** 最多修复几次（防无限循环：修复轮结束又触发 agent_end 本 handler） */
const MAX_JSON_REPAIR = 2;

/** gates/ 禁读的角色白名单（adjust_* 放行——step7 验证器反思需要读写） */
const GATES_BLOCKED_ROLES = new Set(["builder", "coding", "debug"]);

function isActive(): boolean {
	return process.env.SUPEROPT_JSON_GUARD === "1";
}

function currentRole(): string {
	return process.env.SUPEROPT_ROLE || "";
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

/** 解析最终 JSON（fenced 优先 → 裸对象）；失败返回 null */
function parseFinalJson(text: string): Record<string, unknown> | null {
	const m = /```json\s*\n([\s\S]*?)```/.exec(text);
	const candidates: string[] = [];
	if (m) candidates.push(m[1]);
	const trimmed = text.trim();
	if (trimmed.startsWith("{")) candidates.push(trimmed);
	for (const c of candidates) {
		try {
			const v = JSON.parse(c);
			if (v && typeof v === "object" && !Array.isArray(v)) return v;
		} catch { /* 下一个候选 */ }
	}
	return null;
}

/** adjust_space proposals 的 schema 校验（job_56 KR-E）——返回违规清单 */
function validateSpaceSchema(data: Record<string, unknown>): string[] {
	const errs: string[] = [];
	const proposals = data.proposals;
	if (proposals === undefined) {
		errs.push("缺少 proposals 字段（空提案也须显式给 proposals: []）");
		return errs;
	}
	if (!Array.isArray(proposals)) {
		errs.push("proposals 必须是数组");
		return errs;
	}
	proposals.forEach((p, i) => {
		if (!p || typeof p !== "object") {
			errs.push(`proposals[${i}] 不是对象`);
			return;
		}
		const t = String((p as Record<string, unknown>).type || "");
		if (!t) errs.push(`proposals[${i}].type 必填`);
		if (!("evidence" in p) && !("rationale" in p)) {
			errs.push(`proposals[${i}] 缺 evidence/rationale（证据必附）`);
		}
		if (t === "soft_weight") {
			if (!(p as Record<string, unknown>).target_cell) {
				errs.push(`proposals[${i}].soft_weight 缺 target_cell`);
			}
			if (!(p as Record<string, unknown>).action) {
				errs.push(`proposals[${i}].soft_weight 缺 action（weight=N）`);
			}
		}
		if (t === "new_cell") {
			const q = p as Record<string, unknown>;
			for (const k of ["name", "method", "orthogonality_rationale"]) {
				if (!q[k]) errs.push(`proposals[${i}].new_cell 缺 ${k}`);
			}
		}
		if (t === "modify_cell") {
			const q = p as Record<string, unknown>;
			if (!q.target_cell) errs.push(`proposals[${i}].modify_cell 缺 target_cell`);
			if (!q.method) errs.push(`proposals[${i}].modify_cell 缺 method`);
		}
	});
	return errs;
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

/** read/bash 调用参数中是否命中 gates/ 路径（相对/绝对均匹配） */
function hitsGatesPath(toolName: string, input: unknown): boolean {
	if (!input || typeof input !== "object") return false;
	const inp = input as Record<string, unknown>;
	if (toolName === "read") {
		const p = String(inp.path || inp.file_path || "");
		return /(^|[\/\\])env[\/\\]gates[\/\\]/.test(p);
	}
	if (toolName === "bash") {
		const c = String(inp.command || "");
		return /(^|[\"'\s=:])\.?[^\"'\s]*env\/gates\//.test(c);
	}
	return false;
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

	// ---- gates/ 禁读拦截（job_56 task3 KR-E；tool_call 可 block）----
	api.on("tool_call", (event) => {
		const role = currentRole();
		if (!GATES_BLOCKED_ROLES.has(role)) return; // adjust_*/空：放行
		if (event.toolName !== "read" && event.toolName !== "bash") return;
		if (hitsGatesPath(event.toolName, event.input)) {
			return {
				block: true,
				reason:
					"env/gates/ 是验证器目录（防作弊纪律：验证器内容对实验 agent 保密）。" +
					"verify/benchmark skill 会直接调用其实现（执行引用）——请按实验计划继续，不要试图读取验证器内容。",
			};
		}
		return;
	});

	api.on("agent_end", async () => {
		const text = lastAssistantText;
		const role = currentRole();
		if (hasJsonBlock(text)) {
			// job_56 task3：adjust_space 的 schema 结构检（JSON 块之上）
			if (role === "adjust_space") {
				const data = parseFinalJson(text);
				if (data) {
					const errs = validateSpaceSchema(data);
					if (errs.length > 0 && repairCount < MAX_JSON_REPAIR) {
						repairCount++;
						await api.sendUserMessage(
							`你的 proposals JSON 缺少必需字段：${errs.slice(0, 5).join("；")}。` +
								`请修正后重新输出完整 JSON 代码块（\`\`\`json ... \`\`\`）。` +
								`（schema 结构检修复第 ${repairCount}/${MAX_JSON_REPAIR} 次）`,
							{ deliverAs: "followUp" },
						);
						return;
					}
				}
			}
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
