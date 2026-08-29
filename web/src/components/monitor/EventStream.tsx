/**
 * 编排事件流（监控视图右栏——密度优先）。
 *
 * 与 ChatView 富渲染的区别：本组件把 session_event 里的「非 chat 类」事件压成
 * 精简单行卡（时间戳 + 图标 + 摘要）：
 * - tool_execution_start/end → 🔧 工具名 + 参数摘要 / 结果状态
 * - agent_start / agent_settled → 生命周期
 * - message_end（assistant）→ 文本首行摘要（截断）
 * - message_update（流式 delta）→ 跳过（密度优先；chat 组件的职责）
 * - extension_ui_request → 对话框请求提示行
 *
 * 最多渲染最近 MAX_RENDER 行（滚动容器，自动贴底——用户上滚取消贴底）。
 */

import { useEffect, useMemo, useRef, useState } from "react";
import type { PiRpcEvent, SSEEnvelope } from "../../types";

const MAX_RENDER = 300;

type LineKind = "tool" | "tool-error" | "lifecycle" | "message" | "ui" | "state";

interface EventLine {
  id: string;
  time: string;
  kind: LineKind;
  icon: string;
  text: string;
}

const KIND_STYLE: Record<LineKind, string> = {
  tool: "text-ink-2",
  "tool-error": "text-danger",
  lifecycle: "text-rick",
  message: "text-ink",
  ui: "text-morty",
  state: "text-portal",
};

/** 参数摘要（截断 + 压缩空白） */
function summarizeArgs(args: unknown): string {
  if (args === undefined || args === null) return "";
  let s: string;
  if (typeof args === "string") {
    s = args;
  } else {
    try {
      s = JSON.stringify(args);
    } catch {
      s = String(args);
    }
  }
  s = s.replace(/\s+/g, " ").trim();
  return s.length > 90 ? `${s.slice(0, 90)}…` : s;
}

function eventTime(event: PiRpcEvent): string {
  const ts = (event as { timestamp?: unknown }).timestamp;
  if (typeof ts === "number") {
    const d = new Date(ts < 1e12 ? ts * 1000 : ts);
    return d.toLocaleTimeString([], { hour12: false });
  }
  if (typeof ts === "string") {
    const d = new Date(ts);
    if (!Number.isNaN(d.getTime())) return d.toLocaleTimeString([], { hour12: false });
  }
  return "";
}

/** 单个 pi 事件 → 行（null = 跳过：流式 delta / user 消息等） */
function toLine(env: SSEEnvelope, event: PiRpcEvent): EventLine | null {
  const id = String(env.seq);
  const time = eventTime(event);
  switch (event.type) {
    case "tool_execution_start": {
      const text = summarizeArgs(event.args);
      return { id, time, kind: "tool", icon: "🔧", text: `${event.toolName ?? "tool"}${text ? ` ${text}` : ""}` };
    }
    case "tool_execution_end": {
      if (event.isError) {
        return { id, time, kind: "tool-error", icon: "⚠", text: `${event.toolName ?? "tool"} 失败` };
      }
      return { id, time, kind: "tool", icon: "✓", text: `${event.toolName ?? "tool"} 完成` };
    }
    case "agent_start":
      return { id, time, kind: "lifecycle", icon: "▶", text: "agent 启动" };
    case "agent_end":
      return { id, time, kind: "lifecycle", icon: "■", text: "agent 轮次结束" };
    case "agent_settled":
      return { id, time, kind: "lifecycle", icon: "◈", text: "会话收敛（settled）" };
    case "turn_start":
      return { id, time, kind: "lifecycle", icon: "↻", text: "新轮次" };
    case "message_start":
      return null; // 噪声
    case "message_update":
      return null; // 流式 delta——chat 组件职责
    case "message_end": {
      if (event.message?.role !== "assistant") return null; // user 回显跳过
      const content = event.message?.content;
      let text = "";
      if (typeof content === "string") text = content;
      else if (Array.isArray(content)) {
        text = content
          .filter((b) => b?.type === "text" && typeof b.text === "string")
          .map((b) => b.text as string)
          .join(" ");
      }
      text = text.replace(/\s+/g, " ").trim();
      if (!text) return null;
      return { id, time, kind: "message", icon: "💬", text: text.length > 120 ? `${text.slice(0, 120)}…` : text };
    }
    case "extension_ui_request": {
      const method = (event as { method?: unknown }).method ?? "dialog";
      return { id, time, kind: "ui", icon: "🛸", text: `界面请求：${String(method)}` };
    }
    default:
      return null;
  }
}

/** SSEEnvelope[] → 渲染行（session_state 也压成行） */
export function toEventLines(events: SSEEnvelope[]): EventLine[] {
  const lines: EventLine[] = [];
  for (const env of events) {
    if (env.type === "session_state") {
      const data = env.data as { status?: string; reason?: string } | undefined;
      if (data?.status) {
        lines.push({
          id: `s-${env.seq}`,
          time: "",
          kind: "state",
          icon: "◉",
          text: `会话状态 → ${data.status}${data.reason ? `（${data.reason}）` : ""}`,
        });
      }
      continue;
    }
    if (env.type !== "session_event") continue;
    const data = env.data as { event?: PiRpcEvent } | undefined;
    const event = data?.event;
    if (!event?.type) continue;
    const line = toLine(env, event);
    if (line) lines.push(line);
  }
  return lines.length > MAX_RENDER ? lines.slice(lines.length - MAX_RENDER) : lines;
}

interface EventStreamProps {
  events: SSEEnvelope[];
}

export default function EventStream({ events }: EventStreamProps) {
  const lines = useMemo(() => toEventLines(events), [events]);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [stickBottom, setStickBottom] = useState(true);

  // 自动贴底（用户上滚取消；回到底部恢复）
  useEffect(() => {
    const el = containerRef.current;
    if (el && stickBottom) el.scrollTop = el.scrollHeight;
  }, [lines, stickBottom]);

  const onScroll = (): void => {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
    setStickBottom(atBottom);
  };

  return (
    <div
      ref={containerRef}
      onScroll={onScroll}
      className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-line bg-space/60 p-2 font-mono text-xs"
      role="log"
      aria-label="编排事件流"
    >
      {lines.length === 0 ? (
        <p className="px-2 py-6 text-center text-ink-3">（等待事件…）</p>
      ) : (
        <ul className="flex flex-col gap-0.5">
          {lines.map((l) => (
            <li key={l.id} className="flex items-start gap-2 rounded px-1 py-0.5 hover:bg-white/[0.03]">
              <span className="w-14 shrink-0 text-right text-[10px] leading-5 text-ink-3">{l.time}</span>
              <span aria-hidden="true" className="shrink-0 leading-5">{l.icon}</span>
              <span className={`min-w-0 flex-1 break-all leading-5 ${KIND_STYLE[l.kind]}`}>{l.text}</span>
            </li>
          ))}
        </ul>
      )}
      {!stickBottom && (
        <button
          type="button"
          className="sticky bottom-1 left-full -translate-x-full rounded-full border border-line bg-surface-raised px-2 py-0.5 text-[10px] text-ink-2 shadow"
          onClick={() => {
            setStickBottom(true);
            const el = containerRef.current;
            if (el) el.scrollTop = el.scrollHeight;
          }}
        >
          ↓ 最新
        </button>
      )}
    </div>
  );
}
