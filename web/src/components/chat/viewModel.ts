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

export interface ToolItem {
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

export interface UserItem {
  kind: "user";
  id: string;
  text: string;
}

export interface AssistantTextItem {
  kind: "assistant-text";
  id: string;
  text: string;
  streaming: boolean;
}

export interface ThinkingItem {
  kind: "thinking";
  id: string;
  text: string;
  streaming: boolean;
}

export interface NoticeItem {
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
  seq: number;
}

const MAX_NOTIFICATIONS = 5;

function liveId(seq: number): string {
  return `live-${seq}`;
}

/**
 * 由事件流构建视图模型。
 * @param envelopes session_event 信封列表（时间序）
 * @param optimisticUser 本地乐观 user 消息（发送后未收到 echo 前）；echo 到达时由调用方清除
 */
export function buildChatViewModel(
  envelopes: SSEEnvelope[],
  optimisticUser: string | null,
): ChatViewModel {
  const st: BuildState = {
    items: [],
    streaming: false,
    compacting: false,
    pendingDialogs: [],
    notifications: [],
    liveText: null,
    liveThinking: null,
    seq: 0,
  };

  for (const env of envelopes) {
    const ev = (env.data as { event?: Record<string, unknown> })?.event;
    if (!ev || typeof ev.type !== "string") continue;
    applyEvent(st, ev);
  }

  // 收尾：live 块作为条目展示（streaming 中未落定）
  const items = [...st.items];
  if (st.liveText) items.push(st.liveText);
  if (st.liveThinking) items.push(st.liveThinking);
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

function applyEvent(st: BuildState, ev: Record<string, unknown>): void {
  st.seq += 1;
  switch (ev.type) {
    case "agent_start": {
      st.streaming = true;
      break;
    }
    case "agent_settled": {
      st.streaming = false;
      break;
    }
    case "agent_end": {
      const willRetry = ev.willRetry === true;
      if (!willRetry) st.streaming = false;
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
      applyMessageUpdate(st, ev);
      break;
    }
    case "message_end": {
      applyMessageEnd(st, ev);
      break;
    }
    case "tool_execution_start": {
      const toolCallId = String(ev.toolCallId ?? st.seq);
      st.items.push({
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
      } else {
        // 未见过 start 的 end（缓冲截断等）——直接落终态卡
        st.items.push({
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
        kind: "notice",
        id: `retry-${st.seq}`,
        level: "warning",
        text: `自动重试（${attempt}/${max}）${msg ? `：${msg.slice(0, 200)}` : ""}`,
      });
      break;
    }
    case "extension_error": {
      const msg = typeof ev.error === "string" ? ev.error : JSON.stringify(ev.error ?? "");
      st.items.push({
        kind: "notice",
        id: `exterr-${st.seq}`,
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
      st.liveText = { kind: "assistant-text", id: liveId(st.seq), text: "", streaming: true };
    }
    st.liveText.text += ame.delta;
  } else if (ame.type === "thinking_delta" && typeof ame.delta === "string") {
    if (!st.liveThinking) {
      st.liveThinking = { kind: "thinking", id: liveId(st.seq), text: "", streaming: true };
    }
    st.liveThinking.text += ame.delta;
  }
  // text_end/thinking_end/toolcall_* 由 message_end 落定（authoritative）
}

/** message_end：authoritative 落定（覆盖 live 块） */
function applyMessageEnd(st: BuildState, ev: Record<string, unknown>): void {
  const message = ev.message as
    | {
        role?: string;
        content?: string | Array<Record<string, unknown>>;
        stopReason?: string;
      }
    | undefined;
  if (!message) return;
  const role = message.role;

  if (role === "user") {
    st.liveText = null;
    st.liveThinking = null;
    st.items.push({
      kind: "user",
      id: `user-${st.seq}`,
      text: messageText(message),
    });
    return;
  }

  if (role === "assistant") {
    st.liveText = null;
    st.liveThinking = null;

    const content = message.content;
    const blocks = Array.isArray(content) ? content : [];
    for (const b of blocks) {
      if (b?.type === "text" && typeof b.text === "string" && b.text) {
        st.items.push({
          kind: "assistant-text",
          id: `text-${st.seq}-${st.items.length}`,
          text: b.text,
          streaming: false,
        });
      } else if (b?.type === "thinking" && typeof b.thinking === "string" && b.thinking) {
        st.items.push({
          kind: "thinking",
          id: `think-${st.seq}-${st.items.length}`,
          text: b.thinking,
          streaming: false,
        });
      }
      // toolCall 块：由 tool_execution_* 事件渲染（顺序自然落在消息后）
    }

    if (message.stopReason === "error") {
      st.items.push({
        kind: "notice",
        id: `err-${st.seq}`,
        level: "error",
        text: `模型响应错误：${messageText(message).slice(0, 300) || "（无详情）"}`,
      });
    }
    if (message.stopReason === "aborted") {
      st.items.push({
        kind: "notice",
        id: `abort-${st.seq}`,
        level: "info",
        text: "已中止本轮生成",
      });
    }
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
  message?: {
    role?: string;
    content?: string | Array<{ type?: string; text?: string; thinking?: string }>;
    isError?: boolean;
    stopReason?: string;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

/**
 * 离线/历史条目 → 初始 items（message 条目：user/assistant/toolResult）。
 * 仅取 leaf 分支线性序（服务端已按追加序返回）。
 */
export function buildHistoryItems(entries: HistoryEntry[]): ChatItem[] {
  const items: ChatItem[] = [];
  let seq = 0;
  for (const entry of entries) {
    seq += 1;
    const m = entry.message;
    if (!m || typeof m.role !== "string") continue;

    if (m.role === "user") {
      items.push({ kind: "user", id: `h-user-${seq}`, text: messageText(m) });
    } else if (m.role === "assistant") {
      const blocks = Array.isArray(m.content) ? m.content : [];
      for (const b of blocks) {
        if (b?.type === "text" && b.text) {
          items.push({
            kind: "assistant-text",
            id: `h-text-${seq}-${items.length}`,
            text: b.text,
            streaming: false,
          });
        } else if (b?.type === "thinking" && b.thinking) {
          items.push({
            kind: "thinking",
            id: `h-think-${seq}-${items.length}`,
            text: b.thinking,
            streaming: false,
          });
        }
      }
    } else if (m.role === "toolResult") {
      const r = resultText({ content: m.content });
      items.push({
        kind: "tool",
        id: `h-tool-${seq}`,
        toolCallId: `h-${seq}`,
        toolName: typeof m.toolName === "string" ? m.toolName : "tool",
        args: null,
        output: r.text,
        status: m.isError === true ? "error" : "done",
        truncated: false,
        fullOutputPath: null,
      });
    }
  }
  return items;
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
