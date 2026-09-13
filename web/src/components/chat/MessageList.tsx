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

/** 贴底哨兵：远大于任何真实内容高度——赋值即被浏览器 clamp 到最大滚动位置，
 *  无需读取 scrollHeight（避免长会话下的强制同步重排，见 debug/bug4）。 */
const BOTTOM_SENTINEL = 10_000_000;

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
  /** 内容/视口高度缓存（ResizeObserver 维护——回调内布局已干净，读 offsetHeight/
   *  clientHeight 不触发 forced synchronous layout）。滚动路径不得直接读
   *  `scrollContainer.scrollHeight`：长会话（数万 DOM 节点）下每帧多次读取会强制
   *  同步重排（实测流式 600 帧累计 12s，帧间隔 40ms）——详见 debug/bug4。 */
  const contentRef = useRef<HTMLDivElement | null>(null);
  const contentHRef = useRef(0);
  const viewHRef = useRef(0);
  const [pinned, setPinned] = useState(true);
  // 尺寸缓存：ResizeObserver 在浏览器完成布局后回调 → 其中的读取不产生强制重排。
  // 依赖 hasContent：列表在空态时不渲染滚动容器/内容节点（首帧 refs 为 null），
  // 必须在列表真正出现后再挂观察器，否则高度缓存恒为 0 → 贴底失效。
  const hasContent = items.length > 0;
  useEffect(() => {
    if (!hasContent) return;
    const el = scrollRef.current;
    const content = contentRef.current;
    if (!el || !content) return;
    const measure = () => {
      contentHRef.current = content.offsetHeight;
      viewHRef.current = el.clientHeight;
    };
    measure();
    if (typeof ResizeObserver === "function") {
      const ro = new ResizeObserver(measure);
      ro.observe(content);
      ro.observe(el);
      return () => ro.disconnect();
    }
  }, [hasContent]);

  // pinned 的 ref 镜像（rAF 回调里读最新值，避开闭包过期陷阱）
  const pinnedRef = useRef(true);
  useEffect(() => {
    pinnedRef.current = pinned;
  }, [pinned]);
  // 贴底：items 变化（补拉前置合并/流式 delta flush）→ pinned 则 rAF 滚到底。
  // 首帧（空→有内容）也触发一次 → 默认滚到底。onScroll 的方向守卫已避免
  // scroll anchoring（前置插入）误置 pinned=false（那是旧版「打开停在中间」根因）。
  const followRaf = useRef(0);
  const jumpToBottom = useCallback(() => {
    cancelAnimationFrame(followRaf.current);
    followRaf.current = requestAnimationFrame(() => {
      const el = scrollRef.current;
      // 写一个大于任何真实内容高度的哨兵值——浏览器 clamp 到底。不读 scrollHeight
      // （每次流式 flush 读它会触发一次全文档同步重排），也不依赖可能滞后一帧的
      // 高度缓存（缓存只用于「是否真的有可滚动内容」的廉价判断）。
      if (el && pinnedRef.current && contentHRef.current > viewHRef.current) {
        el.scrollTop = BOTTOM_SENTINEL;
      }
    });
  }, []);
  useEffect(() => {
    if (items.length > 0) jumpToBottom();
  }, [items, jumpToBottom]);
  // 卸载时清理 pending rAF（兜底，防泄漏）
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

  // 用户主动滚离底部：以 scrollTop 方向判断——scroll anchoring（历史分页
  // 前置插入）只让 scrollTop 增加（保持视口内容位置）；scrollTop 减少只来自
  // 用户向上滚动浏览更早内容 → 此时才解除贴底。
  const lastScrollTop = useRef(0);
  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    // 距离用缓存的 content/view 高度计算（不读 scrollHeight → 不强制重排）
    const distance = contentHRef.current - viewHRef.current - el.scrollTop;
    if (distance < 60) {
      setPinned(true);
    } else if (el.scrollTop < lastScrollTop.current) {
      setPinned(false); // 用户向上滚（scrollTop 减小）→ 离开底部
    }
    lastScrollTop.current = el.scrollTop;
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
          <div ref={contentRef}>
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
          </div>
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
      // thinking 默认折叠（含流式中的最后一块）：历史+实时都不自动展开。
      // 理由（job_36 用户实测「思考持续追加时整个上下文闪烁」）：展开的流式
      // thinking 块每帧增长会推动下方消息（max-h 内未溢出前），在长历史会话
      // 中部造成视觉跳动；折叠后仅标题行存在，流式更新不产生布局位移。
      // 用户想观看思考时可手动展开单块（override 优先，流式内容照常增长）。
      const isLastStreamingThink = false;
      out.push(renderItem(item, isLastStreamingThink));
    }
  }
  flushTools();

  return out;
}
