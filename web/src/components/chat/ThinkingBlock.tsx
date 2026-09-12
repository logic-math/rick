/**
 * ThinkingBlock：思考过程折叠块（默认折叠——「思考过程」可展开）。
 *
 * 展开控制接入时间线 ExpandCtx（默认/全部展开/全部折叠 + 单项覆盖）：
 * - streaming（流式中）强制展开（用户手动折叠后尊重用户——override 优先）
 * - 历史块默认折叠；当前活跃块默认展开
 * - 超长 thinking 16KB 截断 + 「展开全部」
 */

import { memo, useLayoutEffect, useRef, useState } from "react";
import { useExpandState } from "./expand";

/** 长文本截断阈值（16KB——与 assistant 正文一致） */
const MAX_TEXT = 16 * 1024;

interface ThinkingBlockProps {
  text: string;
  streaming: boolean;
  /** 稳定标识（live=liveId(seq)、历史=h-think-*）——useExpandState 的 key 必须稳定，
   *  否则流式文本变化导致 key 每帧变 → 展开 override 失效 + 重渲染抖动（闪烁根因）。 */
  id: string;
  /** 默认展开态（MessageList 的「最近活跃段」策略传入） */
  defaultOpen?: boolean;
}

function ThinkingBlockImpl({ text, streaming, id, defaultOpen = false }: ThinkingBlockProps) {
  const { open, onToggle } = useExpandState(`think-${id}`, defaultOpen);
  const [expandedFull, setExpandedFull] = useState(false);
  const bodyRef = useRef<HTMLDivElement | null>(null);

  const truncated = !expandedFull && text.length > MAX_TEXT;
  const shown = truncated ? text.slice(0, MAX_TEXT) : text;

  // 流式期间展开时贴底跟随：仅在内容即将溢出块底时才滚（距离 < 24px）。
  // useLayoutEffect（而非 useEffect）：在**浏览器绘制前**同步滚动位置——否则
  // 每帧先按旧 scrollTop 绘制（内容已增长 → 文字看起来先下移一行）再被纠正，
  // 形成逐帧抖动（bug3「思考刷新感」的次要来源）。effect 内只读布局属性，无副作用。
  useLayoutEffect(() => {
    if (open && streaming) {
      const el = bodyRef.current;
      if (el && el.scrollHeight - el.scrollTop - el.clientHeight < 24) {
        el.scrollTop = el.scrollHeight;
      }
    }
  }, [open, streaming, text]);

  return (
    <div className="my-1 rounded-md border border-dashed border-line bg-space/60">
      <button
        type="button"
        onClick={() => onToggle(!open)}
        className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs text-ink-3 hover:text-ink-2"
        aria-expanded={open}
      >
        <span aria-hidden="true">{open ? "▾" : "▸"}</span>
        <span>思考过程</span>
        {streaming && (
          <span
            className="inline-block h-1.5 w-1.5 rounded-full bg-rick opacity-80"
            aria-label="思考中"
          />
        )}
        <span className="ml-auto font-mono text-[10px] text-ink-3">
          {text.length > 1024 ? `${Math.ceil(text.length / 1024)}K` : `${text.length}B`}
        </span>
      </button>
      {open && (
        <div
          ref={bodyRef}
          className="max-h-72 overflow-y-auto border-t border-dashed border-line px-3 py-2"
        >
          <p className="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-ink-2">
            {shown}
            {streaming && (
              <span
                aria-hidden="true"
                className="ml-0.5 inline-block h-3 w-[6px] translate-y-[1px] rounded-[1px] bg-rick opacity-80"
              />
            )}
          </p>
          {truncated && (
            <button
              type="button"
              onClick={() => setExpandedFull(true)}
              className="mt-2 text-[11px] text-portal hover:text-portal-strong"
            >
              展开全部（{Math.ceil((text.length - MAX_TEXT) / 1024)}K 已截断）
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// 流式稳定性：vm 每帧重建、text 每帧变 → streaming 中必须重渲（比较返回 false）；
// 已落定（streaming=false）的思考块 id/text 不变 → 跳过渲染（历史思考块不随
// 打字机帧重渲——闪烁机制三）。
const ThinkingBlock = memo(
  ThinkingBlockImpl,
  (prev, next) =>
    prev.id === next.id &&
    prev.defaultOpen === next.defaultOpen &&
    prev.streaming === next.streaming &&
    (next.streaming ? false : prev.text === next.text),
);

export default ThinkingBlock;
