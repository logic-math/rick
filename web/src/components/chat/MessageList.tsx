/**
 * MessageList：消息滚动容器（渲染层虚拟化，job_43 task7）。
 *
 * - react-virtuoso：只渲染可视区条目（500 条历史从全量 DOM → 可视区 ~10 条，
 *   移动端滚动/内存主收益；桌面同享）
 * - 新消息自动贴底（followOutput + pinned 时的 scrollToIndex 补充——覆盖流式
 *   原地增高，Virtuoso 的 followOutput 只管追加）；用户上滚（离开底部）取消贴底
 *   +「回到底部」悬浮按钮
 * - 历史前置插入防跳：firstItemIndex 随前置条数递减（Virtuoso 约定）
 * - 顶部触达分页（startReached）+ 顶部「加载更早历史」按钮（components.Header）
 * - ≥2 个连续工具调用自动折叠为一组「N 次工具调用」（ToolGroup）
 * - 折叠展开控制（ExpandCtx 上下文）：状态提升在本组件（overrides Map 在条目
 *   之上）→ 虚拟化卸载/重挂不丢展开态
 * - 长文本 16KB 截断 + 展开全部（user/assistant 气泡内）
 *
 * 迁移说明（旧机制 → 新机制，见 task7 KR8b-e）：
 * - BOTTOM_SENTINEL/ResizeObserver 高度缓存/onScroll 方向判定 → Virtuoso
 *   atBottomStateChange（贴底态）+ startReached（触顶分页）
 * - rAF jumpToBottom → followOutput="auto"（追加）+ scrollToIndex(LAST)（原地增高）
 * - scroll anchoring（前置插入防跳）→ firstItemIndex 递减约定
 */

import {
  forwardRef,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
import type { ChatItem } from "./viewModel";
import { AssistantBubble, NoticeCard, UserBubble } from "./MessageBubble";
import ThinkingBlock from "./ThinkingBlock";
import ToolCallCard, { ToolElapsed, noteToolRunStart } from "./ToolCallCard";
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
        {running && <Saucer size={22} flying={running} />}
        <span className="text-ink-2">{tools.length} 次工具调用</span>
        {running && <span className="text-[11px] text-portal">运行中…</span>}
        {running &&
          (() => {
            const rt = tools.find((t) => t.status === "running");
            if (!rt) return null;
            noteToolRunStart(rt.id);
            return <ToolElapsed toolId={rt.id} />;
          })()}
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

/**
 * 虚拟化扁平条目：单条 or 工具组（组 = 一行虚拟条目，动态高度由 Virtuoso 原生支持）。
 * id 用于 computeItemKey（stable 身份锚——沿用 envelope id 锚定方案，展开态 overrides
 * 跨滚动保持的前提）。
 */
type FlatEntry =
  | { kind: "single"; id: string; item: ChatItem }
  | { kind: "group"; id: string; tools: Extract<ChatItem, { kind: "tool" }>[]; defaultOpen: boolean };

/**
 * 连续工具条目折叠为组（≥2）+ 默认展开策略：
 * 「最近活跃片段」= 最后一个工具组（或流式 thinking）默认展开，其余折叠。
 */
function buildFlat(items: ChatItem[]): FlatEntry[] {
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

  const out: FlatEntry[] = [];
  let toolRun: Extract<ChatItem, { kind: "tool" }>[] = [];
  let currentRunStart = -1;

  const flushTools = () => {
    if (toolRun.length === 0) return;
    if (toolRun.length === 1) {
      out.push({ kind: "single", id: toolRun[0].id, item: toolRun[0] });
    } else {
      out.push({
        kind: "group",
        id: `grp-${toolRun[0].id}`,
        tools: toolRun,
        defaultOpen: currentRunStart === lastToolGroupIdx,
      });
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
      out.push({ kind: "single", id: item.id, item });
    }
  }
  flushTools();

  return out;
}

/** 单条目渲染（工具组在上方折叠处理） */
function renderItem(item: ChatItem, defaultOpen: boolean, animate: boolean): ReactNode {
  const cls = animate ? "chat-item-enter" : undefined;
  switch (item.kind) {
    case "user":
      return (
        <div key={item.id} className={cls}>
          <UserBubble item={item} />
        </div>
      );
    case "assistant-text":
      return (
        <div key={item.id} className={cls}>
          <AssistantBubble item={item} />
        </div>
      );
    case "thinking":
      return (
        <div key={item.id} className={cls}>
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
        <div key={item.id} className={cls}>
          <ToolCallCard item={item} />
        </div>
      );
    case "notice":
      return (
        <div key={item.id} className={cls}>
          <NoticeCard item={item} />
        </div>
      );
  }
}

/** firstItemIndex 初始基准：预留前置空间（历史分页每次前置 N 条 → 递减 N；
 *  Virtuoso 只要求 ≥0 且随前置单调递减，绝对值无意义） */
const FIRST_INDEX_BASE = 100_000;

export default function MessageList({
  items,
  streaming,
  onReachTop,
  hasMore = false,
  loadingEarlier = false,
}: MessageListProps) {
  const virtuosoRef = useRef<VirtuosoHandle | null>(null);
  const [pinned, setPinned] = useState(true);

  // ---- 展开控制（状态提升：overrides 在条目之上 → 虚拟化卸载不丢展开态，KR8e）----
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

  // ---- 虚拟化数据（分组扁平化；items 引用变化即重算——流式 110ms 粒度）----
  const flat = useMemo(() => buildFlat(items), [items]);

  // ---- 前置插入检测 + firstItemIndex（KR8d：历史前置插入防跳）----
  // 判定：首条 id 变化 + 长度增长 = 前置（追加时首条 id 不变；原地更新长度不变）。
  // 重建（SSE resync 全量重拉）长度可能变化——仅 N>0 时递减，位移漂移无害。
  const [firstItemIndex, setFirstItemIndex] = useState(FIRST_INDEX_BASE);
  /** 上一帧快照（firstId + 长度）——渲染期只读，effect 期写入 */
  const prevFrameRef = useRef<{ firstId: string | null; len: number } | null>(null);
  /** 本帧是否为前置插入（渲染期判定，供动画 memo + firstItemIndex effect 复用） */
  const isPrependFrame = (() => {
    const prev = prevFrameRef.current;
    if (!prev || flat.length === 0 || !flat[0].id) return false;
    return prev.firstId !== null && flat[0].id !== prev.firstId && flat.length > prev.len;
  })();
  useEffect(() => {
    if (isPrependFrame) {
      const n = flat.length - (prevFrameRef.current?.len ?? flat.length);
      if (n > 0) setFirstItemIndex((v) => v - n);
    }
    prevFrameRef.current = {
      firstId: flat.length > 0 ? flat[0].id : null,
      len: flat.length,
    };
  }, [flat, isPrependFrame]);

  // ---- 入场动画：仅「新追加的尾部条目」播（KR8c 附带）----
  // 虚拟化下滚动会卸载/重挂条目——若无条件播 chat-item-enter，滚动过程中条目
  // 会反复闪入。以「已见 id 集」为界：仅新增 id（尾部追加的新消息）播动画；
  // 前置历史条目（首条 id 变化的帧）不播；滚动重挂（id 已见）不播。
  const seenIdsRef = useRef<Set<string> | null>(null);
  const animateIds = useMemo(() => {
    if (seenIdsRef.current === null) {
      // 首帧：全部视为已见（历史加载，不播动画）
      seenIdsRef.current = new Set(flat.map((e) => e.id));
      return new Set<string>();
    }
    const seen = seenIdsRef.current;
    const added = new Set<string>();
    for (const e of flat) {
      if (!seen.has(e.id)) {
        seen.add(e.id);
        added.add(e.id);
      }
    }
    // 前置帧：新增 id 是历史条目，不播动画
    if (isPrependFrame) return new Set<string>();
    return added;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [flat]);

  // ---- 贴底（KR8b）----
  // followOutput="auto"：追加且用户在底部 → 立即贴底（Virtuoso 内部时序）。
  // 补充 effect：流式「原地增高」（最后一条内容增长，长度不变）followOutput
  // 不覆盖 → pinned 时 scrollToIndex(LAST)。两个通道目标一致（底部、瞬时），
  // 追加帧冗余执行一次幂等滚动，无平滑动画打架。
  useEffect(() => {
    if (!pinned || flat.length === 0) return;
    virtuosoRef.current?.scrollToIndex({ index: flat.length - 1, align: "end" });
  }, [flat, pinned]);
  const scrollToBottom = useCallback(() => {
    setPinned(true);
    virtuosoRef.current?.scrollToIndex({ index: "LAST", align: "end", behavior: "smooth" });
  }, []);

  // ---- 自定义滚动容器：保留 rm-chat-scroll（scrollbar-gutter）+ a11y 语义 ----
  // ⚠️ Scroller 必须引用稳定（useMemo []）——Virtuoso 的 components 类型变化会
  // 卸载重挂滚动容器 → 滚动位置丢失。onReachTop 经 ref 间接（ChatView 传的是内联
  // lambda，流式期每次渲染都变——若直接进 Header 依赖会让 Header 频繁重挂）。
  const onReachTopRef = useRef(onReachTop);
  onReachTopRef.current = onReachTop;
  const Scroller = useMemo(
    () =>
      // eslint-disable-next-line react/display-name
      forwardRef<HTMLDivElement, { style?: React.CSSProperties; children?: ReactNode }>(
        function VirtuosoScroller({ style, children }, ref) {
          return (
            <div
              ref={ref}
              className="rm-chat-scroll h-full overflow-y-auto px-4 py-3"
              style={style}
              role="log"
              aria-live="polite"
            >
              {children}
            </div>
          );
        },
      ),
    [],
  );

  // ---- Header：顶部「加载更早历史」按钮（随内容滚动，KR8d 显式兑底）----
  const Header = useMemo(() => {
    // eslint-disable-next-line react/display-name
    return function VirtuosoHeader() {
      if (!hasMore) return null;
      return (
        <div className="mb-2 flex justify-center" style={{ minHeight: 28 }}>
          <button
            type="button"
            onClick={() => onReachTopRef.current?.()}
            disabled={loadingEarlier}
            className="rounded-full border border-line bg-surface-raised/70 px-3 py-1 text-[11px] text-ink-3 transition-colors hover:border-portal/50 hover:text-portal disabled:opacity-50"
          >
            {loadingEarlier ? "加载中…" : "↑ 加载更早历史"}
          </button>
        </div>
      );
    };
  }, [hasMore, loadingEarlier]);

  /** components 集合（引用稳定：Scroller 固定 + Header 仅在分页态变化时变） */
  const components = useMemo(() => ({ Scroller, Header }), [Scroller, Header]);

  if (items.length === 0 && !streaming) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-ink-3">
        {/* 空态静态飞船（job_43 二轮 KR7a，全局不限移动端）：空态常驻飞行动画
            （SMIL 航灯 + 悬停/光束 CSS）是新开会话挂机发烫主根因；空态本就静态，
            视觉损失≈0。工具运行期指示（ToolGroup/SteerBar/MonitorView）不动——
            那是功能反馈，仅移动端降级（theme.css 窄屏块 + Saucer 内部 SMIL 条件渲染）。 */}
        <Saucer size={64} />
        <p className="text-sm">会话已就绪——发送第一条消息开始</p>
      </div>
    );
  }

  return (
    <div className="relative h-full">
      <ExpandCtx.Provider value={ctx}>
        <Virtuoso
          ref={virtuosoRef}
          style={{ height: "100%" }}
          data={flat}
          computeItemKey={(_, entry) => entry.id}
          firstItemIndex={firstItemIndex}
          initialTopMostItemIndex={Math.max(0, flat.length - 1)}
          followOutput="auto"
          atBottomStateChange={setPinned}
          startReached={() => onReachTopRef.current?.()}
          components={components}
          itemContent={(_, entry) =>
            entry.kind === "group" ? (
              <ToolGroup
                key={entry.id}
                tools={entry.tools}
                groupKey={entry.id}
                defaultOpen={entry.defaultOpen}
              />
            ) : (
              renderItem(entry.item, false, animateIds.has(entry.id))
            )
          }
        />
      </ExpandCtx.Provider>

      {/* 展开控制（时间线顶部悬浮） */}
      <div className="pointer-events-none absolute right-3 top-2 flex gap-1">
        <div className="pointer-events-auto flex overflow-hidden rounded-md border border-line bg-surface-raised text-[10px] md:bg-surface-raised/90 md:backdrop-blur">
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
