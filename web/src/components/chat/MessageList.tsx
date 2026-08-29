/**
 * MessageList：消息滚动容器。
 *
 * - 新消息自动贴底；用户上滚（离开底部 > 60px）取消贴底 + 「回到底部」悬浮按钮
 * - ≥2 个连续工具调用自动折叠为一组「N 次工具调用」（ToolGroup）
 * - 空态：飞碟 + 引导文案
 */

import { useEffect, useRef, useState } from "react";
import type { ChatItem } from "./viewModel";
import { AssistantBubble, NoticeCard, UserBubble } from "./MessageBubble";
import ThinkingBlock from "./ThinkingBlock";
import ToolCallCard from "./ToolCallCard";
import Saucer from "../starfield/Saucer";

/** 连续工具调用组（≥2 折叠） */
function ToolGroup({ tools }: { tools: Extract<ChatItem, { kind: "tool" }>[] }) {
  const [expanded, setExpanded] = useState(false);
  const running = tools.some((t) => t.status === "running");
  const errorCount = tools.filter((t) => t.status === "error").length;

  return (
    <div className="my-1">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 rounded-md border border-line bg-space/60 px-3 py-1.5 text-left text-xs hover:bg-white/5"
        aria-expanded={expanded}
      >
        <span aria-hidden="true">{expanded ? "▾" : "▸"}</span>
        {running && <Saucer size={22} flying />}
        <span className="text-ink-2">{tools.length} 次工具调用</span>
        {running && <span className="text-[11px] text-portal">运行中…</span>}
        {errorCount > 0 && <span className="text-[11px] text-danger">{errorCount} 失败</span>}
        <span className="ml-auto font-mono text-[10px] text-ink-3">
          {tools.map((t) => t.toolName).join(" · ")}
        </span>
      </button>
      {expanded && (
        <div className="mt-1 space-y-0.5 pl-2">
          {tools.map((t) => (
            <ToolCallCard key={t.id} item={t} />
          ))}
        </div>
      )}
    </div>
  );
}

interface MessageListProps {
  items: ChatItem[];
  streaming: boolean;
}

/** 单条目渲染（工具组在上方折叠处理） */
function renderItem(item: ChatItem): React.ReactNode {
  switch (item.kind) {
    case "user":
      return <UserBubble key={item.id} item={item} />;
    case "assistant-text":
      return <AssistantBubble key={item.id} item={item} />;
    case "thinking":
      return <ThinkingBlock key={item.id} text={item.text} streaming={item.streaming} />;
    case "tool":
      return <ToolCallCard key={item.id} item={item} />;
    case "notice":
      return <NoticeCard key={item.id} item={item} />;
  }
}

export default function MessageList({ items, streaming }: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const [pinned, setPinned] = useState(true);

  // 贴底跟随：新内容到达且用户未上滚时滚动到底
  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !pinned) return;
    el.scrollTop = el.scrollHeight;
  }, [items, pinned]);

  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    const distance = el.scrollHeight - el.scrollTop - el.clientHeight;
    setPinned(distance < 60);
  };

  const scrollToBottom = () => {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
    setPinned(true);
  };

  if (items.length === 0 && !streaming) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-ink-3">
        <Saucer size={64} flying />
        <p className="text-sm">会话已就绪——发送第一条消息开始</p>
      </div>
    );
  }

  return (
    <div className="relative h-full">
      <div
        ref={scrollRef}
        onScroll={onScroll}
        className="h-full overflow-y-auto px-4 py-3"
        role="log"
        aria-live="polite"
      >
        {renderGrouped(items)}
      </div>
      {!pinned && (
        <button
          type="button"
          onClick={scrollToBottom}
          className="absolute bottom-3 left-1/2 -translate-x-1/2 rounded-full border border-line bg-surface-raised px-3 py-1.5 text-xs text-ink-2 shadow-lg hover:text-ink"
        >
          ↓ 回到底部
        </button>
      )}
    </div>
  );
}

/** 连续工具条目折叠为组（≥2） */
function renderGrouped(items: ChatItem[]): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  let toolRun: Extract<ChatItem, { kind: "tool" }>[] = [];

  const flushTools = () => {
    if (toolRun.length === 0) return;
    if (toolRun.length === 1) {
      out.push(renderItem(toolRun[0]));
    } else {
      out.push(<ToolGroup key={`grp-${toolRun[0].id}`} tools={toolRun} />);
    }
    toolRun = [];
  };

  for (const item of items) {
    if (item.kind === "tool") {
      toolRun.push(item);
    } else {
      flushTools();
      out.push(renderItem(item));
    }
  }
  flushTools();

  return out;
}
