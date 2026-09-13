/**
 * 聊天视图模型构建器（纯函数）——pi rpc 事件流 → 渲染条目列表。
 *
 * 渲染分层共识（OpenHands/LibreChat 模式，research-L1-r2-leaf-4）：
 * - 文本 delta 直渲（message_update 增量拼接为 live 条目）
 * - message_end 落定（authoritative message 覆盖 live 条目）
 * - 工具调用折叠卡（tool_execution_* 事件驱动）
 * - thinking 折叠块
 * - 错误二分：连接/服务错误由 ChatView 顶部 ErrorBanner 展示；
 *   agent 内容错误（isError 工具/stopReason=error/auto_retry）为内联卡片
 *
 * 输入：events store 的 SSEEnvelope[]（session_event 透传，data.event=pi rpc 原始事件）
 * 输出：ChatItem[]（渲染条目，按时间序）+ streaming/compacting 状态
 */

import type { SSEEnvelope } from "../../types";

// ============================================================
// 类型
// ============================================================

export type ToolStatus = "running" | "done" | "error";

export interface ToolItem extends SeqAnchor {
  kind: "tool";
  id: string; // toolCallId
  toolCallId: string;
  toolName: string;
  args: unknown;
  /** 累积输出（tool_execution_update.partialResult 覆盖式 / end.result 终值） */
  output: string;
  status: ToolStatus;
  /** 结果截断/全文路径（details 字段） */
  truncated: boolean;
  fullOutputPath: string | null;
}

/** envelope 全局序号（Hub 单调 seq）——live 项的**时序锚点**。
 *  乐观用户消息按「发送时刻的会话最大 seq」插入，保证严格 timeline：
 *  发送前产生的内容（含上一条 AI 回复的尾巴，可能尚未落盘）在其前，
 *  发送后产生的内容在其后（此前用「history 与 live 之间」的固定位置，
 *  在上一轮回复尾巴仍在 live 时会把新消息插到它前面 → 用户实测乱序）。 */
export interface SeqAnchor {
  seq?: number;
}

export interface UserItem extends SeqAnchor {
  kind: "user";
  id: string;
  text: string;
  /** epoch ms（历史=entry 时间戳；live=事件到达时）——气泡相对时间显示用 */
  ts?: number;
}

export interface AssistantTextItem extends SeqAnchor {
  kind: "assistant-text";
  id: string;
  text: string;
  streaming: boolean;
  ts?: number;
}

export interface ThinkingItem extends SeqAnchor {
  kind: "thinking";
  id: string;
  text: string;
  streaming: boolean;
}

export interface NoticeItem extends SeqAnchor {
  kind: "notice";
  id: string;
  /** info | warning | error */
  level: "info" | "warning" | "error";
  text: string;
}

export type ChatItem = UserItem | AssistantTextItem | ThinkingItem | ToolItem | NoticeItem;

export interface ChatViewModel {
  items: ChatItem[];
  /** agent 处理中（agent_start…agent_settled） */
  streaming: boolean;
  /** 上下文压缩中（compaction_start…compaction_end） */
  compacting: boolean;
  /** 挂起的 extension_ui 对话框请求（select/confirm/input/editor——需回响应） */
  pendingDialogs: ExtensionUIDialogRequest[];
  /** fire-and-forget notify 通知（最近 N 条，可关闭） */
  notifications: Array<{ id: string; message: string; notifyType: string }>;
}
export interface ExtensionUIDialogRequest {
  id: string;
  method: string; // select | confirm | input | editor
  title?: string;
  message?: string;
  options?: string[];
  placeholder?: string;
  prefill?: string;
}

// ============================================================
// 工具函数
// ============================================================

/** message content（string | content blocks）→ 纯文本 */
export function messageText(message: unknown): string {
  const m = message as { content?: string | Array<{ type?: string; text?: string }> } | undefined;
  if (!m) return "";
  if (typeof m.content === "string") return m.content;
  if (Array.isArray(m.content)) {
    return m.content
      .filter((c) => c?.type === "text" && typeof c.text === "string")
      .map((c) => c.text)
      .join("");
  }
  return "";
}

/** 工具 result（pi：{content:[{type,text}]...} 或字符串）→ 纯文本 */
export function resultText(result: unknown): { text: string; truncated: boolean; fullOutputPath: string | null } {
  if (result == null) return { text: "", truncated: false, fullOutputPath: null };
  if (typeof result === "string") return { text: result, truncated: false, fullOutputPath: null };
  const r = result as {
    content?: Array<{ type?: string; text?: string }>;
    details?: { truncation?: unknown; fullOutputPath?: string | null };
  };
  let text = "";
  if (Array.isArray(r.content)) {
    text = r.content.map((c) => (typeof c?.text === "string" ? c.text : "")).join("");
  } else {
    try {
      text = JSON.stringify(result, null, 2);
    } catch {
      text = String(result);
    }
  }
  const truncated = Boolean(r.details?.truncation);
  const fullOutputPath = r.details?.fullOutputPath ?? null;
  return { text, truncated, fullOutputPath };
}

/** 工具参数摘要（单行） */
export function summarizeArgs(args: unknown): string {
  if (args == null) return "";
  if (typeof args === "string") return args;
  const a = args as Record<string, unknown>;
  for (const key of ["command", "file_path", "path", "pattern", "query", "url", "name"]) {
    const v = a?.[key];
    if (typeof v === "string" && v) return v.length > 120 ? `${v.slice(0, 117)}…` : v;
  }
  try {
    const s = JSON.stringify(args);
    return s.length > 120 ? `${s.slice(0, 117)}…` : s;
  } catch {
    return "";
  }
}

// ============================================================
// 构建器
// ============================================================

interface BuildState {
  items: ChatItem[];
  streaming: boolean;
  compacting: boolean;
  pendingDialogs: ExtensionUIDialogRequest[];
  notifications: Array<{ id: string; message: string; notifyType: string }>;
  /** live 流式块（message_update 拼接；message_end 落定后清除） */
  liveText: AssistantTextItem | null;
  liveThinking: ThinkingItem | null;
  /** live 块的创建顺序（text/think）——收尾按真实到达序展示（否则落定时 block 序
   *  与流式序不一致 → 位置跳变） */
  liveOrder: Array<"text" | "think">;
  seq: number;
  /** 当前事件的服务端全局 seq（envelope.seq）——落定条目稳定 id 源 */
  idSeq: number;
  /** 打开块锚点：最近一次 message_end 的 envelope.seq（0=会话首条消息）。
   *  结构帧永不裁 → 锚点跨裁剪/重放稳定；live 块与落定块共用该锚点生成 id，
   *  于是 live→落定是**同一个 React key 的就地更新**（不重挂、不重放 enter 动画、
   *  useExpandState 的展开 override 不丢）——bug3。 */
  boundarySeq: number;
  /** 历史回放指纹（live 重叠跳过） */
  skip: Set<string> | undefined;
}

const MAX_NOTIFICATIONS = 5;

/** 消息块稳定 id：锚点（前置 message_end 的 seq）+ 块类别 + 同类块序号。
 *  live 块用 0 号（首个同类块）——与落定后的首个同类块 id 一致。 */
function msgBlockId(anchor: number, kind: "text" | "think", idx: number): string {
  return idx > 0 ? `msg-${anchor}-${kind}-${idx}` : `msg-${anchor}-${kind}`;
}

/**
 * 由事件流构建视图模型。
 * @param envelopes session_event 信封列表（时间序）
 * @param optimisticUser 本地乐观 user 消息（发送后未收到 echo 前）；echo 到达时由调用方清除
 * @param skipFingerprints 历史回放已渲染的内容指纹——live 事件与之重叠时跳过
 *   （挂载时 REST 补历史与 SSE 在途事件的重叠消解；指纹见 buildHistoryItems）
 */
export function buildChatViewModel(
  envelopes: SSEEnvelope[],
  optimisticUser: string | null,
  skipFingerprints?: Set<string>,
): ChatViewModel {
  const st: BuildState = {
    items: [],
    streaming: false,
    compacting: false,
    pendingDialogs: [],
    notifications: [],
    liveText: null,
    liveThinking: null,
    liveOrder: [],
    seq: 0,
    idSeq: 0,
    boundarySeq: 0,
    skip: skipFingerprints,
  };

  for (const env of envelopes) {
    const ev = (env.data as { event?: Record<string, unknown> })?.event;
    if (!ev || typeof ev.type !== "string") continue;
    applyEvent(st, ev, env.seq);
  }

  // 收尾：live 块作为条目展示（streaming 中未落定）——按创建序（与落定 block 序一致）
  const items = [...st.items];
  for (const kind of st.liveOrder) {
    if (kind === "text" && st.liveText) items.push(st.liveText);
    else if (kind === "think" && st.liveThinking) items.push(st.liveThinking);
  }
  if (optimisticUser) {
    items.push({ kind: "user", id: "optimistic-user", text: optimisticUser });
  }

  return {
    items,
    streaming: st.streaming,
    compacting: st.compacting,
    pendingDialogs: st.pendingDialogs,
    notifications: st.notifications,
  };
}

/**
 * 工具终态化兜底：任何「推进」事件（新工具开始 / agent 开始回文本 / 回合结束）
 * 到达时，把仍卡在 running 的工具强制标 done——保证 UI 不会因缺失的
 * tool_execution_end / 未配对的 toolResult 而永久旋转（job_36 验收期实测：
 * 上一个 edit 已返回内容但仍在旋转）。
 */
function supersedeRunningTools(st: BuildState): void {
	for (const it of st.items) {
		if (it.kind === "tool" && it.status === "running") {
			it.status = "done";
		}
	}
}

/**
 * 事件分发。envSeq = envelope 顶层 seq（服务端 Hub.Publish 全局单调）——
 * 落定条目的稳定身份源：客户端 buffers 环形裁剪 / resync 重放只改变 replay
 * 起点，不改变 envelope.seq → 用 envSeq 生成的条目 id 跨裁剪/重放稳定，
 * React key 不变 → 不重挂（旧版用 st.seq（replay 局部计数）+ items.length
 * 作 id，裁剪平移全部已渲染条目 key → 重挂 + enter 动画重放 = 「上一条 AI
 * 消息在工具/思考流式时闪烁」（bug2）。
 */
function applyEvent(st: BuildState, ev: Record<string, unknown>, envSeq: number): void {
  st.seq += 1;
  st.idSeq = envSeq > 0 ? envSeq : st.seq; // 容错：无 seq 时退回局部计数
  switch (ev.type) {
    case "agent_start": {
      st.streaming = true;
      break;
    }
    case "agent_settled": {
      st.streaming = false;
      supersedeRunningTools(st); // 回合结束：仍 running 的工具强制终态化
      break;
    }
    case "agent_end": {
      const willRetry = ev.willRetry === true;
      if (!willRetry) {
        st.streaming = false;
        supersedeRunningTools(st);
      }
      break;
    }
    case "compaction_start": {
      st.compacting = true;
      break;
    }
    case "compaction_end": {
      st.compacting = false;
      break;
    }
    case "message_update": {
      // agent 开始返回文本 → 工具调用阶段已结束：强制终态化仍 running 的工具
      supersedeRunningTools(st);
      applyMessageUpdate(st, ev);
      break;
    }
    case "message_end": {
      supersedeRunningTools(st);
      applyMessageEnd(st, ev, envSeq);
      break;
    }
    case "tool_execution_start": {
      const toolCallId = String(ev.toolCallId ?? st.seq);
      // 新工具开始：上一个工具必然已结束（顺序推进）——强制终态化前一个 running
      supersedeRunningTools(st);
      if (!st.skip?.has(`tool:${toolCallId}`)) {
        st.items.push({
          seq: st.idSeq || st.seq,
          kind: "tool",
          id: `tool-${toolCallId}`,
          toolCallId,
          toolName: String(ev.toolName ?? "tool"),
          args: ev.args,
          output: "",
          status: "running",
          truncated: false,
          fullOutputPath: null,
        });
      }
      break;
    }
    case "tool_execution_update": {
      const toolCallId = String(ev.toolCallId ?? "");
      const item = st.items.find((i) => i.kind === "tool" && i.toolCallId === toolCallId);
      if (item && item.kind === "tool") {
        const r = resultText(ev.partialResult);
        item.output = r.text;
        item.truncated = r.truncated;
        item.fullOutputPath = r.fullOutputPath;
      }
      break;
    }
    case "tool_execution_end": {
      const toolCallId = String(ev.toolCallId ?? "");
      const item = st.items.find((i) => i.kind === "tool" && i.toolCallId === toolCallId);
      if (item && item.kind === "tool") {
        const r = resultText(ev.result);
        item.output = r.text;
        item.truncated = r.truncated;
        item.fullOutputPath = r.fullOutputPath;
        item.status = ev.isError === true ? "error" : "done";
      } else if (!st.skip?.has(`tool:${toolCallId}`)) {
        // 未见过 start 的 end（缓冲截断等）——直接落终态卡
        st.items.push({
          seq: st.idSeq || st.seq,
          kind: "tool",
          id: `tool-${toolCallId}`,
          toolCallId,
          toolName: String(ev.toolName ?? "tool"),
          args: ev.args,
          output: resultText(ev.result).text,
          status: ev.isError === true ? "error" : "done",
          truncated: resultText(ev.result).truncated,
          fullOutputPath: resultText(ev.result).fullOutputPath,
        });
      }
      break;
    }
    case "extension_ui_request": {
      applyExtensionUIRequest(st, ev);
      break;
    }
    case "auto_retry_start": {
      const attempt = ev.attempt ?? "?";
      const max = ev.maxAttempts ?? "?";
      const msg = typeof ev.errorMessage === "string" ? ev.errorMessage : "";
      st.items.push({
        seq: st.idSeq || st.seq,
        kind: "notice",
        id: `retry-${st.idSeq || st.seq}`,
        level: "warning",
        text: `自动重试（${attempt}/${max}）${msg ? `：${msg.slice(0, 200)}` : ""}`,
      });
      break;
    }
    case "extension_error": {
      const msg = typeof ev.error === "string" ? ev.error : JSON.stringify(ev.error ?? "");
      st.items.push({
        seq: st.idSeq || st.seq,
        kind: "notice",
        id: `exterr-${st.idSeq || st.seq}`,
        level: "error",
        text: `扩展错误：${msg.slice(0, 300)}`,
      });
      break;
    }
    default:
      break;
  }
}

/** message_update：live 流式块拼接（text_delta / thinking_delta） */
function applyMessageUpdate(st: BuildState, ev: Record<string, unknown>): void {
  const ame = ev.assistantMessageEvent as
    | { type?: string; delta?: string; contentIndex?: number }
    | undefined;
  if (!ame || typeof ame.type !== "string") return;

  if (ame.type === "text_delta" && typeof ame.delta === "string") {
    if (!st.liveText) {
      st.liveText = {
        kind: "assistant-text",
        id: msgBlockId(st.boundarySeq, "text", 0),
        text: "",
        streaming: true,
        seq: st.idSeq || st.seq,
      };
      st.liveOrder.push("text");
    }
    st.liveText.text += ame.delta;
  } else if (ame.type === "thinking_delta" && typeof ame.delta === "string") {
    if (!st.liveThinking) {
      st.liveThinking = {
        kind: "thinking",
        id: msgBlockId(st.boundarySeq, "think", 0),
        text: "",
        streaming: true,
        seq: st.idSeq || st.seq,
      };
      st.liveOrder.push("think");
    }
    st.liveThinking.text += ame.delta;
  }
  // text_end/thinking_end/toolcall_* 由 message_end 落定（authoritative）
}

/** message_end：authoritative 落定（覆盖 live 块）。
 *  envSeq = 本 message_end 的 envelope.seq：处理完后成为下一打开块的锚点（boundarySeq）。
 *  落定块 id 用「**前置** message_end 锚点」而非本帧 seq——与流式中的 live 块 id 一致
 *  → 落定是同一 key 的就地更新（不重挂/不重放 enter 动画/展开态不丢）。 */
function applyMessageEnd(st: BuildState, ev: Record<string, unknown>, envSeq: number): void {
  const message = ev.message as
    | {
        role?: string;
        content?: string | Array<Record<string, unknown>>;
        stopReason?: string;
      }
    | undefined;
  if (!message) return;
  const role = message.role;
  const anchor = st.boundarySeq;

  if (role === "user") {
    st.liveText = null;
    st.liveThinking = null;
    st.liveOrder = [];
    const text = messageText(message);
    if (!st.skip?.has(`user:${text.trim()}`)) {
      st.items.push({
        seq: st.idSeq || st.seq,
        kind: "user",
        id: `user-${st.idSeq || st.seq}`,
        text,
      });
    }
    st.boundarySeq = envSeq > 0 ? envSeq : st.seq; // 服务端 seq 恒≥1；兜底用 replay 计数保证唯一
    return;
  }

  if (role === "assistant") {
    st.liveText = null;
    st.liveThinking = null;
    st.liveOrder = [];

    const content = message.content;
    const blocks = Array.isArray(content) ? content : [];
    // 同类块序号（非共享 blockIdx）：首个同类块 = live 块 id，后续同类块递号
    let textIdx = 0;
    let thinkIdx = 0;
    for (const b of blocks) {
      const block = b as Record<string, unknown>;
      if (block?.type === "text" && typeof block.text === "string" && block.text) {
        if (!st.skip?.has(`text:${block.text.trim()}`)) {
          st.items.push({
            seq: st.idSeq || st.seq,
            kind: "assistant-text",
            id: msgBlockId(anchor, "text", textIdx),
            text: block.text,
            streaming: false,
            ts: Date.now(),
          });
        }
        textIdx += 1;
      } else if (block?.type === "thinking" && typeof block.thinking === "string" && block.thinking) {
        if (!st.skip?.has(`think:${block.thinking.trim()}`)) {
          st.items.push({
            seq: st.idSeq || st.seq,
            kind: "thinking",
            id: msgBlockId(anchor, "think", thinkIdx),
            text: block.thinking,
            streaming: false,
          });
        }
        thinkIdx += 1;
      }
      // toolCall 块：由 tool_execution_* 事件渲染（顺序自然落在消息后）
    }

    if (message.stopReason === "error") {
      st.items.push({
        seq: st.idSeq || st.seq,
        kind: "notice",
        id: `err-${st.idSeq || st.seq}`,
        level: "error",
        text: `模型响应错误：${messageText(message).slice(0, 300) || "（无详情）"}`,
      });
    }
    if (message.stopReason === "aborted") {
      st.items.push({
        seq: st.idSeq || st.seq,
        kind: "notice",
        id: `abort-${st.idSeq || st.seq}`,
        level: "info",
        text: "已中止本轮生成",
      });
    }
    // 本帧成为下一打开块的锚点（结构帧 seq 永不被裁 → 跨裁剪/重放稳定）
    st.boundarySeq = envSeq > 0 ? envSeq : st.seq; // 服务端 seq 恒≥1；兜底用 replay 计数保证唯一
  }
}

/** extension_ui_request：对话框入队；notify 入通知列表 */
function applyExtensionUIRequest(st: BuildState, ev: Record<string, unknown>): void {
  const method = typeof ev.method === "string" ? ev.method : "";
  const id = typeof ev.id === "string" ? ev.id : `ui-${st.seq}`;

  if (method === "select" || method === "confirm" || method === "input" || method === "editor") {
    st.pendingDialogs.push({
      id,
      method,
      title: typeof ev.title === "string" ? ev.title : undefined,
      message: typeof ev.message === "string" ? ev.message : undefined,
      options: Array.isArray(ev.options) ? (ev.options as string[]) : undefined,
      placeholder: typeof ev.placeholder === "string" ? ev.placeholder : undefined,
      prefill: typeof ev.prefill === "string" ? ev.prefill : undefined,
    });
    return;
  }

  if (method === "notify") {
    st.notifications.push({
      id,
      message: typeof ev.message === "string" ? ev.message : "",
      notifyType: typeof ev.notifyType === "string" ? ev.notifyType : "info",
    });
    if (st.notifications.length > MAX_NOTIFICATIONS) {
      st.notifications = st.notifications.slice(st.notifications.length - MAX_NOTIFICATIONS);
    }
  }
  // setStatus/setWidget/setTitle/set_editor_text：TUI 概念——v1 忽略
}

// ============================================================
// 历史回放（REST /entries → ChatItem[]，用于挂载时补历史）
// ============================================================

interface HistoryEntry {
  type?: string;
  id?: string;
  parentId?: string;
  timestamp?: string;
  message?: {
    role?: string;
    content?: string | Array<Record<string, unknown>>;
    isError?: boolean;
    stopReason?: string;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

/** 历史回放产物：items + 消息指纹（live 流去重）+ 会话元信息 */
export interface HistoryBuildResult {
  items: ChatItem[];
  /** 已渲染内容的指纹集合（live 事件流与之去重：user:/text:/think:/tool:） */
  fingerprints: Set<string>;
  /** 会话级元信息（会话信息折叠区展示） */
  meta: HistoryMeta;
}

export interface HistoryMeta {
  model: string | null;
  provider: string | null;
  thinkingLevel: string | null;
  /** 首条 user 消息时间（会话启动锚点） */
  startedAt: string | null;
}

/** 内容指纹（归一化文本/工具调用 id——历史与 live 重叠消解） */
function fingerprint(kind: string, key: string): string {
  return `${kind}:${key}`;
}

function normText(s: string): string {
  return s.trim();
}

/** ISO 时间戳（如 2026-01-01T00:00:01.000Z）→ epoch ms；解析失败返回 undefined */
function tsMs(iso: string | undefined): number | undefined {
  if (!iso) return undefined;
  const t = Date.parse(iso);
  return Number.isFinite(t) ? t : undefined;
}

/**
 * 离线/历史条目 → 初始 items + 指纹 + 元信息。
 *
 * 关键配对：assistant 消息的 toolCall 块（name + arguments）与后续 toolResult
 * 条目（toolCallId → content/isError）按 toolCallId 合并——完整工具卡
 * （参数 + 输出 + 状态）才能落进时间线（此前 args 丢失是「历史没法复查」主因）。
 * 服务端已按追加序返回（assistant(toolCalls) → toolResults → …）。
 */
export function buildHistoryItems(entries: HistoryEntry[]): HistoryBuildResult {
  const items: ChatItem[] = [];
  const fingerprints = new Set<string>();
  const meta: HistoryMeta = { model: null, provider: null, thinkingLevel: null, startedAt: null };
  /** toolCallId → ToolItem（引用就地补 result——保持时间线块序） */
  const toolById = new Map<string, ToolItem>();
  let seq = 0;

  for (const entry of entries) {
    seq += 1;
    /** 块内稳定序号：entry.id 可能同 entry 多块（text/thinking/toolCall），
     *  用块内索引区分——避免重建时 id 抖动（key 变 → React 全量重挂 + 动画重放
     *  = 历史闪烁根因；override 展开态也因 key 变而失效）。 */
    const eid = (suffix: string): string =>
      `h-${suffix}-${typeof entry.id === "string" ? entry.id : seq}`;

    // 会话级元信息（model_change / thinking_level_change）
    if (entry.type === "model_change") {
      if (typeof entry.modelId === "string") meta.model = entry.modelId;
      if (typeof entry.provider === "string") meta.provider = entry.provider;
      continue;
    }
    if (entry.type === "thinking_level_change") {
      if (typeof entry.thinkingLevel === "string") meta.thinkingLevel = entry.thinkingLevel;
      continue;
    }

    const m = entry.message;
    if (!m || typeof m.role !== "string") continue;

    if (m.role === "user") {
      const text = messageText(m);
      items.push({ kind: "user", id: eid("user"), text, ts: tsMs(entry.timestamp) });
      fingerprints.add(fingerprint("user", normText(text)));
      if (!meta.startedAt && text) meta.startedAt = entry.timestamp ?? null;
      continue;
    }

    if (m.role === "assistant") {
      const blocks = Array.isArray(m.content) ? m.content : [];
      let blockIdx = 0;
      for (const b of blocks) {
        const block = b as Record<string, unknown>;
        if (block?.type === "text" && typeof block.text === "string" && block.text) {
          items.push({
            kind: "assistant-text",
            id: eid(`text-${blockIdx}`),
            text: block.text,
            streaming: false,
            ts: tsMs(entry.timestamp),
          });
          fingerprints.add(fingerprint("text", normText(block.text)));
          blockIdx += 1;
        } else if (block?.type === "thinking" && typeof block.thinking === "string" && block.thinking) {
          items.push({
            kind: "thinking",
            id: eid(`think-${blockIdx}`),
            text: block.thinking,
            streaming: false,
          });
          fingerprints.add(fingerprint("think", normText(block.thinking)));
          blockIdx += 1;
        } else if (block?.type === "toolCall") {
          // 工具调用块：name + arguments（toolResult 后续按 toolCallId 回填）
          const callId = typeof block.id === "string" ? block.id : eid(`tool-${blockIdx}`);
          const toolName =
            typeof block.name === "string" ? block.name : "tool";
          const item: ToolItem = {
            kind: "tool",
            id: eid(`tool-${blockIdx}`),
            toolCallId: callId,
            toolName,
            args: block.arguments ?? null,
            output: "",
            // 历史回放的工具调用一律视为已结束：历史消息是已落盘内容，不存在
            // 「还在跑」的历史工具；真正的进行中工具由 live 事件（open block）渲染。
            // 旧值 "running" 会让未配对 result 的历史工具组判定为 running →
            // 渲染 flying Saucer（rm-beam + rm-saucer-hover 两个**无限动画**）→
            // 长会话数百个并发无限动画 → 每帧全文档样式/布局重算（debug/bug4 根因）。
            status: "done",
            truncated: false,
            fullOutputPath: null,
          };
          items.push(item);
          toolById.set(callId, item);
          fingerprints.add(fingerprint("tool", callId));
          blockIdx += 1;
        }
      }
      continue;
    }

    if (m.role === "toolResult") {
      const r = resultText({ content: m.content });
      const callId = typeof m.toolCallId === "string" ? m.toolCallId : null;
      const target = callId ? toolById.get(callId) : undefined;
      if (target) {
        // 就地回填（时间线块序不变）
        target.output = r.text;
        target.truncated = r.truncated;
        target.fullOutputPath = r.fullOutputPath;
        target.status = m.isError === true ? "error" : "done";
      } else {
        // 未配对（缓冲截断/历史分支）——独立工具卡（无参数但有输出）
        items.push({
          kind: "tool",
          id: eid("toolr"),
          toolCallId: callId ?? eid("toolr"),
          toolName: typeof m.toolName === "string" ? m.toolName : "tool",
          args: null,
          output: r.text,
          status: m.isError === true ? "error" : "done",
          truncated: r.truncated,
          fullOutputPath: r.fullOutputPath,
        });
        if (callId) fingerprints.add(fingerprint("tool", callId));
      }
    }
  }

  return { items, fingerprints, meta };
}

/**
 * 判定「乐观 user 消息是否已被 echo 消解」：
 * 事件流中出现与乐观文本相同的 user 条目即消解。
 */
export function optimisticEchoed(items: ChatItem[], optimisticText: string | null): boolean {
  if (!optimisticText) return false;
  return items.some(
    (i) => i.kind === "user" && normalize(i.text) === normalize(optimisticText),
  );
}

function normalize(s: string): string {
  return s.trim();
}
