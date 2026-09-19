/**
 * 编排事件流（监控视图右栏——密度优先）。
 *
 * 与 ChatView 富渲染的区别：本组件把 session_event 里的「非 chat 类」事件压成
 * 精简单行卡（时间戳 + 图标 + 摘要）：
 * - tool_execution_start/end → 🔧 工具名 + 参数摘要 / 结果状态；**行可点击展开
 *   简版详情（完整参数 + 结果）**——复查行为轨迹
 * - agent_start / agent_settled → 生命周期
 * - message_end（assistant）→ 文本首行摘要（截断）
 * - message_update（流式 delta）→ 跳过（密度优先；chat 组件的职责）
 * - extension_ui_request → 对话框请求提示行
 *
 * 最多渲染最近 MAX_RENDER 行（滚动容器，自动贴底——用户上滚取消贴底）。
 */

import { useEffect, useMemo, useRef, useState } from "react";
import type { PiRpcEvent, SSEEnvelope } from "../../types";
import Collapse from "../common/Collapse";

const MAX_RENDER = 300;
/** 展开详情的输出截断（密度优先——监控视图简版） */
const MAX_DETAIL = 4 * 1024;

type LineKind = "tool" | "tool-error" | "lifecycle" | "message" | "ui" | "state";

interface EventLine {
  id: string;
  time: string;
  kind: LineKind;
  icon: string;
  text: string;
  /** 可展开详情（tool 行：参数 + 结果） */
  detail?: { args?: unknown; result?: unknown; isError?: boolean };
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
      return {
        id,
        time,
        kind: "tool",
        icon: "🔧",
        text: `${event.toolName ?? "tool"}${text ? ` ${text}` : ""}`,
        detail: { args: event.args },
      };
    }
    case "tool_execution_end": {
      if (event.isError) {
        return {
          id,
          time,
          kind: "tool-error",
          icon: "⚠",
          text: `${event.toolName ?? "tool"} 失败`,
          detail: { result: event.result, isError: true, args: event.args },
        };
      }
      return {
        id,
        time,
        kind: "tool",
        icon: "✓",
        text: `${event.toolName ?? "tool"} 完成`,
        detail: { result: event.result, args: event.args },
      };
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
      return { id, time, kind: "message", icon: "💬", text: text.length > 120 ? `${text.slice(0, 120)}…` : text, detail: undefined };
    }
    case "extension_ui_request": {
      const method = (event as { method?: unknown }).method ?? "dialog";
      return { id, time, kind: "ui", icon: "🛸", text: `界面请求：${String(method)}` };
    }
    default:
      return null;
  }
}

/** SSEEnvelope[] → 渲染行（session_state 也压成行；同工具的 start/end 行合并详情） */
export function toEventLines(events: SSEEnvelope[]): EventLine[] {
  const lines: EventLine[] = [];
  const toolLineById = new Map<string, EventLine>();
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
    const data = env.data as
      | { event?: PiRpcEvent; kind?: string; job_id?: string; task_id?: string; from?: string; to?: string; note?: string }
      | undefined;
    // **合成会话事件（非 pi rpc）**：doing 的任务态变更 / 轮次进度说明。
    // 旧实现只认 data.event（pi rpc 形状）→ doing 会话的事件流**永远空白**
    // （用户实测「doing 静默执行、没有任何事件更新」）。
    if (data?.kind === "doing_progress" || data?.kind === "doing_note") {
      const isNote = data.kind === "doing_note";
      const text = isNote
        ? data.note || "进度更新"
        : `${data.job_id ?? ""} ${data.task_id ?? ""}: ${data.from ?? "?"} → ${data.to ?? "?"}`.trim();
      lines.push({
        id: `d-${env.seq}`,
        time: "",
        kind: isNote ? "lifecycle" : "state",
        icon: isNote ? "ℹ" : "📋",
        text,
      });
      continue;
    }
    const event = data?.event;
    if (!event?.type) continue;
    const line = toLine(env, event);
    if (!line) continue;

    // 同一工具调用的 start/end 合并：end 行的 result 并入 start 行（保留首行位置）
    const callId = event.toolCallId;
    if (callId && line.detail && (event.type === "tool_execution_start" || event.type === "tool_execution_end")) {
      const existing = toolLineById.get(callId);
      if (existing && existing.detail) {
        existing.detail = {
          ...existing.detail,
          ...line.detail,
          args: existing.detail.args ?? line.detail.args,
        };
        // end 到达后把首行状态文本更新为终态摘要
        if (event.type === "tool_execution_end") {
          existing.icon = event.isError ? "⚠" : "✓";
          existing.kind = event.isError ? "tool-error" : "tool";
          existing.text = `${event.toolName ?? existing.text} ${event.isError ? "失败" : "完成"}`;
        }
        continue; // end 行不单独渲染
      }
      if (event.type === "tool_execution_start") {
        toolLineById.set(callId, line);
      }
    }

    lines.push(line);
  }
  return lines.length > MAX_RENDER ? lines.slice(lines.length - MAX_RENDER) : lines;
}

/** 工具行详情（参数 + 结果，监控简版——4KB 截断） */
function ToolDetail({ detail }: { detail: NonNullable<EventLine["detail"]> }) {
  const argsText = useMemo(() => {
    if (detail.args == null) return null;
    if (typeof detail.args === "string") return detail.args;
    try {
      return JSON.stringify(detail.args, null, 2);
    } catch {
      return String(detail.args);
    }
  }, [detail.args]);

  const resultText = useMemo(() => {
    if (detail.result == null) return null;
    if (typeof detail.result === "string") return detail.result;
    const r = detail.result as { content?: Array<{ type?: string; text?: string }> };
    if (Array.isArray(r.content)) {
      return r.content.map((c) => (typeof c?.text === "string" ? c.text : "")).join("");
    }
    try {
      return JSON.stringify(detail.result, null, 2);
    } catch {
      return String(detail.result);
    }
  }, [detail.result]);

  return (
    <div className="flex flex-col gap-1 px-2 pb-2 pt-1">
      {argsText != null && (
        <div>
          <div className="text-[10px] uppercase tracking-wide text-ink-3">参数</div>
          <pre className="mt-0.5 max-h-40 overflow-auto rounded bg-space p-1.5 font-mono text-[10px] leading-relaxed text-ink-2">
            {argsText.length > MAX_DETAIL ? `${argsText.slice(0, MAX_DETAIL)}…` : argsText}
          </pre>
        </div>
      )}
      {resultText != null && (
        <div>
          <div className="text-[10px] uppercase tracking-wide text-ink-3">
            结果{detail.isError ? "（失败）" : ""}
          </div>
          <pre className={`mt-0.5 max-h-40 overflow-auto rounded bg-space p-1.5 font-mono text-[10px] leading-relaxed ${
            detail.isError ? "text-danger" : "text-ink-2"
          }`}>
            {resultText.length > MAX_DETAIL ? `${resultText.slice(0, MAX_DETAIL)}…` : resultText}
          </pre>
        </div>
      )}
    </div>
  );
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
          {lines.map((l) =>
            l.detail ? (
              <li key={l.id}>
                <Collapse
                  className="!rounded !border-0 !bg-transparent"
                  headerClassName="!px-1 !py-0.5 rounded hover:bg-white/[0.03]"
                  contentClassName="!border-t-0"
                  title={
                    <>
                      <span className="w-12 shrink-0 text-right text-[10px] leading-5 text-ink-3">{l.time}</span>
                      <span aria-hidden="true" className="shrink-0 leading-5">{l.icon}</span>
                      <span className={`min-w-0 flex-1 break-all leading-5 ${KIND_STYLE[l.kind]}`}>{l.text}</span>
                    </>
                  }
                  label={`展开 ${l.text}`}
                >
                  <ToolDetail detail={l.detail} />
                </Collapse>
              </li>
            ) : (
              <li key={l.id} className="flex items-start gap-2 rounded px-1 py-0.5 hover:bg-white/[0.03]">
                <span className="w-14 shrink-0 text-right text-[10px] leading-5 text-ink-3">{l.time}</span>
                <span aria-hidden="true" className="shrink-0 leading-5">{l.icon}</span>
                <span className={`min-w-0 flex-1 break-all leading-5 ${KIND_STYLE[l.kind]}`}>{l.text}</span>
              </li>
            ),
          )}
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
