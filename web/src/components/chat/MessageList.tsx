/**
 * MessageList：消息滚动容器。
 *
 * - 新消息自动贴底；用户上滚（离开底部 > 60px）取消贴底 + 「回到底部」悬浮按钮
 * - ≥2 个连续工具调用自动折叠为一组「N 次工具调用」（ToolGroup）
 * - 折叠展开控制（ExpandCtrl 上下文）：
 *   · 默认策略：只展开「最近活跃片段」——最后一个工具组 + 流式 thinking
 *   · 「全部展开 / 全部折叠 / 默认」三态切换（时间线顶部小按钮）
 * - 长文本 16KB 截断 + 展开全部（user/assistant 气泡内）
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { ChatItem } from "./viewModel";
import { AssistantBubble, NoticeCard, UserBubble } from "./MessageBubble";
import ThinkingBlock from "./ThinkingBlock";
import ToolCallCard from "./ToolCallCard";
import Saucer from "../starfield/Saucer";
import { ExpandCtx, useExpandState, type ExpandCtrl, type ExpandMode } from "./expand";

// ============================================================
// 工具组（≥2 连续工具调用折叠）
// ============================================================

/** 连续工具调用组（≥2 折叠）——完整参数/输出在组内逐卡展开 */
function ToolGroup({
  tools,
  groupKey,
  defaultOpen,
}: {
  tools: Extract<ChatItem, { kind: "tool" }>[];
  groupKey: string;
  defaultOpen: boolean;
}) {
  const { open, onToggle } = useExpandState(groupKey, defaultOpen);
  const running = tools.some((t) => t.status === "running");
  const errorCount = tools.filter((t) => t.status === "error").length;

  return (
    <div className="my-1">
      <button
        type="button"
        onClick={() => onToggle(!open)}
        className="flex w-full items-center gap-2 rounded-md border border-line bg-space/60 px-3 py-1.5 text-left text-xs hover:bg-white/5"
        aria-expanded={open}
      >
        <span aria-hidden="true">{open ? "▾" : "▸"}</span>
        {running && <Saucer size={22} flying />}
        <span className="text-ink-2">{tools.length} 次工具调用</span>
        {running && <span className="text-[11px] text-portal">运行中…</span>}
        {errorCount > 0 && <span className="text-[11px] text-danger">{errorCount} 失败</span>}
        <span className="ml-auto max-w-[45%] truncate font-mono text-[10px] text-ink-3">
          {tools.map((t) => t.toolName).join(" · ")}
        </span>
      </button>
      {open && (
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
  /** 滚动到顶部时回调（历史分页：加载更早）——触发方防抖/幂等 */
  onReachTop?: () => void;
  /** 可能还有更早历史（显示顶部「加载更早历史」按钮——显式兑底，不只靠滚动触发） */
  hasMore?: boolean;
  /** 加载更早进行中（按钮「加载中…」禁用态） */
  loadingEarlier?: boolean;
}

/** 单条目渲染（工具组在上方折叠处理） */
function renderItem(item: ChatItem, defaultOpen: boolean): ReactNode {
  switch (item.kind) {
    case "user":
      return (
        <div key={item.id} className="chat-item-enter">
          <UserBubble item={item} />
        </div>
      );
    case "assistant-text":
      return (
        <div key={item.id} className="chat-item-enter">
          <AssistantBubble item={item} />
        </div>
      );
    case "thinking":
      return (
        <div key={item.id} className="chat-item-enter">
          <ThinkingBlock
            id={item.id}
            text={item.text}
            streaming={item.streaming}
            defaultOpen={defaultOpen}
          />
        </div>
      );
    case "tool":
      return (
        <div key={item.id} className="chat-item-enter">
          <ToolCallCard item={item} />
        </div>
      );
    case "notice":
      return (
        <div key={item.id} className="chat-item-enter">
          <NoticeCard item={item} />
        </div>
      );
  }
}

export default function MessageList({
  items,
  streaming,
  onReachTop,
  hasMore = false,
  loadingEarlier = false,
}: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const [pinned, setPinned] = useState(true);
  // pinned 的 ref 镜像（rAF 回调里读最新值，避开闭包过期陷阱）
  const pinnedRef = useRef(true);
  useEffect(() => {
    pinnedRef.current = pinned;
  }, [pinned]);
  // 贴底滚动 rAF 合并：items 流式时每帧变化，同步 scrollTop=scrollHeight 会造成
  // 整页抖动/内容跳动（打字机闪烁的机制二）。合并到 requestAnimationFrame——
  // 一帧内多次 items 更新只滚一次；且仅 pinned（用户近底）时跟随。
  const followRaf = useRef(0);
  useEffect(() => {
    if (!pinned) return;
    cancelAnimationFrame(followRaf.current);
    followRaf.current = requestAnimationFrame(() => {
      const el = scrollRef.current;
      // 仅当内容确实超出容器（可滚）且用户仍近底时才拽到底——
      // 内容不足一屏时 scrollHeight<=clientHeight，赋值无副作用。
      if (el && pinnedRef.current && el.scrollHeight > el.clientHeight) {
        el.scrollTop = el.scrollHeight;
      }
    });
    return () => cancelAnimationFrame(followRaf.current);
  }, [items, pinned]);
  // 卸载时清理 pending rAF
  useEffect(() => () => cancelAnimationFrame(followRaf.current), []);
  const [mode, setMode] = useState<ExpandMode>("default");
  const [overrides, setOverrides] = useState<ReadonlyMap<string, boolean>>(new Map());
  const setOverride = useCallback((key: string, open: boolean) => {
    setOverrides((prev) => {
      const next = new Map(prev);
      next.set(key, open);
      return next;
    });
  }, []);

  const ctx = useMemo<ExpandCtrl>(
    () => ({ mode, overrides, setOverride }),
    [mode, overrides, setOverride],
  );

  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    const distance = el.scrollHeight - el.scrollTop - el.clientHeight;
    setPinned(distance < 60);
    // 顶部触达（分页加载更早历史）——近顶阈值；onReachTop 由调用方防抖/幂等
    if (el.scrollTop <= 32) {
      onReachTop?.();
    }
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
        className="rm-chat-scroll h-full overflow-y-auto px-4 py-3"
        role="log"
        aria-live="polite"
      >
        <ExpandCtx.Provider value={ctx}>
          {/* 顶部「加载更早历史」按钮（hasMore 时显示——显式兑底；
              滚动到顶自动加载保留，按钮让超长会话可一步一页继续往上） */}
          {hasMore && (
            <div className="mb-2 flex justify-center">
              <button
                type="button"
                onClick={() => onReachTop?.()}
                disabled={loadingEarlier}
                className="rounded-full border border-line bg-surface-raised/70 px-3 py-1 text-[11px] text-ink-3 transition-colors hover:border-portal/50 hover:text-portal disabled:opacity-50"
              >
                {loadingEarlier ? "加载中…" : "↑ 加载更早历史"}
              </button>
            </div>
          )}
          {renderGrouped(items)}
        </ExpandCtx.Provider>
      </div>

      {/* 展开控制（时间线顶部悬浮） */}
      <div className="pointer-events-none absolute right-3 top-2 flex gap-1">
        <div className="pointer-events-auto flex overflow-hidden rounded-md border border-line bg-surface-raised/90 text-[10px] backdrop-blur">
          {(
            [
              ["default", "默认"],
              ["all-open", "全部展开"],
              ["all-closed", "全部折叠"],
            ] as Array<[ExpandMode, string]>
          ).map(([m, label]) => (
            <button
              key={m}
              type="button"
              onClick={() => {
                setMode(m);
                setOverrides(new Map());
              }}
              className={`px-2 py-1 text-ink-3 hover:text-ink ${
                mode === m ? "bg-portal-soft text-portal" : ""
              }`}
              aria-pressed={mode === m}
            >
              {label}
            </button>
          ))}
        </div>
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

/**
 * 连续工具条目折叠为组（≥2）+ 默认展开策略：
 * 「最近活跃片段」= 最后一个工具组（或流式 thinking）默认展开，其余折叠。
 */
function renderGrouped(items: ChatItem[]): ReactNode[] {
  // 分段：找出最后一个工具组的索引
  let lastToolGroupIdx = -1;
  let runStart = -1;
  for (let i = 0; i < items.length; i++) {
    if (items[i].kind === "tool") {
      if (runStart === -1) runStart = i;
      lastToolGroupIdx = runStart; // 组的起始索引
    } else {
      runStart = -1;
    }
  }

  const out: ReactNode[] = [];
  let toolRun: Extract<ChatItem, { kind: "tool" }>[] = [];
  let currentRunStart = -1;

  const flushTools = () => {
    if (toolRun.length === 0) return;
    if (toolRun.length === 1) {
      out.push(renderItem(toolRun[0], false));
    } else {
      out.push(
        <ToolGroup
          key={`grp-${toolRun[0].id}`}
          tools={toolRun}
          groupKey={`grp-${currentRunStart}`}
          defaultOpen={currentRunStart === lastToolGroupIdx}
        />,
      );
    }
    toolRun = [];
    currentRunStart = -1;
  };

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.kind === "tool") {
      if (toolRun.length === 0) currentRunStart = i;
      toolRun.push(item);
    } else {
      flushTools();
      // thinking：流式中的最后一块默认展开；历史 thinking 全折叠
      const isLastStreamingThink =
        item.kind === "thinking" && item.streaming && i === items.length - 1;
      out.push(renderItem(item, isLastStreamingThink));
    }
  }
  flushTools();

  return out;
}
