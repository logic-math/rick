/**
 * MessageBubble：单条消息气泡。
 *
 * - user：右侧对齐、surface-raised 底、nebula 色调边框（最大宽 80%）
 * - assistant：左侧全宽（markdown 正文直排）+ mini Portal 头像标识
 * - notice：内联卡片（info/warning/error 三级——agent 内容错误二分的「内联」侧）
 */

import type { AssistantTextItem, NoticeItem, UserItem } from "./viewModel";
import Markdown from "./Markdown";
import StreamText from "./StreamText";
import Portal from "../starfield/Portal";
import { memo, useState } from "react";

/** 长文本截断阈值（16KB）——复查场景防卡顿，展开全部看全文 */
const MAX_TEXT = 16 * 1024;

/** 相对时间：刚刚 / N 分钟前 / HH:MM / M月D日（>24h 显示完整，hover 看原始时间） */
function relativeTime(ts?: number): string {
  if (!ts) return "";
  const diff = Date.now() - ts;
  if (diff < 60_000) return "刚刚";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`;
  const d = new Date(ts);
  const now = new Date();
  if (d.toDateString() === now.toDateString()) {
    return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
  }
  return `${d.getMonth() + 1}月${d.getDate()}日`;
}

/** 消息时间戳（相对文本 + hover 原始时间） */
function TimeStamp({ ts }: { ts?: number }) {
  if (!ts) return null;
  const label = relativeTime(ts);
  if (!label) return null;
  const full = new Date(ts).toLocaleString();
  return (
    <span
      title={full}
      className="shrink-0 select-none text-[10px] text-ink-3"
      aria-label={`发送时间 ${full}`}
    >
      {label}
    </span>
  );
}

/** 长文本 + 展开全部（user/assistant 共用） */
function LongText({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);
  const truncated = !expanded && text.length > MAX_TEXT;
  const shown = truncated ? text.slice(0, MAX_TEXT) : text;
  return (
    <>
      <p className="whitespace-pre-wrap break-words text-sm leading-relaxed text-ink">{shown}</p>
      {truncated && (
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="mt-1 text-[11px] text-portal hover:text-portal-strong"
        >
          展开全部（{Math.ceil((text.length - MAX_TEXT) / 1024)}K 已截断）
        </button>
      )}
    </>
  );
}

export const UserBubble = memo(
  function UserBubble({ item }: { item: UserItem }) {
    // 用户消息作为事件流的一种类型（左侧上下文流布局，与 AssistantBubble 对齐）：
    // - user 标签 + 时间戳（不是靠右特殊布局）——用户反馈：不要右侧特殊展示
    // - `w-fit max-w-full`：气泡宽=内容宽、上限全宽；flex 项内不塌缩（修复「继续」
    //   竖排成「继\n续」——旧版 `max-w-[80%]` 在 flex items-end 内 min-width:auto
    //   + break-words 导致中文被拆成单字列，实测 p 宽 28px）
    return (
      <div className="my-2 flex gap-2.5">
        <div className="mt-0.5 shrink-0" aria-hidden="true">
          <UserBadge />
        </div>
        <div className="min-w-0 flex-1">
          <div className="mb-0.5 flex items-center gap-2 text-[10px] text-ink-3">
            <span className="font-semibold uppercase tracking-wide text-ink-2">user</span>
            <TimeStamp ts={item.ts} />
          </div>
          <div className="w-max max-w-full rounded-2xl rounded-tl-sm border border-nebula/40 bg-surface-raised px-4 py-2.5">
            <LongText text={item.text} />
          </div>
        </div>
      </div>
    );
  },
  // 流式期间 vm 每帧新建 item 对象——浅比较必然不等 → 历史 user 消息每帧重渲
  // （打字机闪烁机制三）。自定义比较：内容字段不变则跳过渲染。
  (prev, next) =>
    prev.item.id === next.item.id &&
    prev.item.text === next.item.text &&
    prev.item.ts === next.item.ts,
);

/** 用户标识：cel-shade 青色小星球（R&M 星球规范：亮/暗两段硬边界 + 大气辉光，
 *  主色青色 #5cc8c2 与 AI 的传送门绿 #97ce4c 区分——user/assistant 一眼可辨） */
function UserBadge() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 20 20"
      aria-hidden="true"
      className="mt-0.5"
    >
      {/* 大气辉光（低透明青色晕） */}
      <circle cx="10" cy="10" r="9.5" fill="#5cc8c2" opacity="0.25" />
      {/* 球体：整体青绿底 */}
      <circle cx="10" cy="10" r="8" fill="#5cc8c2" />
      {/* cel-shade 暗段（右下方硬边界，单一光源左上） */}
      <path
        d="M10 2 A8 8 0 0 0 10 18 A6 8 0 0 1 10 2Z"
        fill="#2e7d78"
        opacity="0.9"
      />
      {/* 云带（两条浅色横纹，卡通感） */}
      <ellipse cx="6.5" cy="7.5" rx="2.6" ry="0.9" fill="#b8ece6" opacity="0.85" transform="rotate(-12 6.5 7.5)" />
      <ellipse cx="11" cy="11.5" rx="3" ry="0.8" fill="#b8ece6" opacity="0.6" transform="rotate(8 11 11.5)" />
    </svg>
  );
}

export const AssistantBubble = memo(
  function AssistantBubble({ item }: { item: AssistantTextItem }) {
    const [expanded, setExpanded] = useState(false);
    const truncated = !expanded && !item.streaming && item.text.length > MAX_TEXT;
    const shown = truncated ? item.text.slice(0, MAX_TEXT) : item.text;
    return (
      <div className="my-2 flex gap-2.5">
        <div className="mt-0.5 shrink-0" aria-hidden="true">
          {/* 只有正在流式的那条消息旋转头像——历史消息静态（否则每条消息一个
              无限 SVG 动画 → 主线程逐帧样式/布局重算；见 Portal.spin 注释） */}
          <Portal size={20} spin={item.streaming} />
        </div>
        <div className="min-w-0 flex-1">
          {item.streaming ? (
            // 流式期间：轻量纯文本直渲（防高频 markdown 重排卡顿）+ 光标闪烁
            <StreamText text={item.text} />
          ) : (
            <>
              {/* message_end 落定：authoritative markdown 渲染（16KB 截断 + 展开） */}
              <Markdown>{shown}</Markdown>
              {truncated && (
                <button
                  type="button"
                  onClick={() => setExpanded(true)}
                  className="mt-1 text-[11px] text-portal hover:text-portal-strong"
                >
                  展开全部（{Math.ceil((item.text.length - MAX_TEXT) / 1024)}K 已截断）
                </button>
              )}
              {!item.streaming && <TimeStamp ts={item.ts} />}
            </>
          )}
        </div>
      </div>
    );
  },
  // streaming=true（live 打字机）：text 每帧变 → 必须重渲；
  // 落定后（streaming=false）：text/id/ts 不变则跳过（历史正文不随流式帧重渲）。
  (prev, next) =>
    prev.item.id === next.item.id &&
    prev.item.ts === next.item.ts &&
    prev.item.streaming === next.item.streaming &&
    (next.item.streaming ? false : prev.item.text === next.item.text),
);

export function NoticeCard({ item }: { item: NoticeItem }) {
  const tone =
    item.level === "error"
      ? "border-danger/50 bg-danger/10 text-danger"
      : item.level === "warning"
        ? "border-morty/50 bg-morty/10 text-morty"
        : "border-line bg-space/60 text-ink-2";
  const icon = item.level === "error" ? "⛔" : item.level === "warning" ? "⚠" : "ℹ";
  return (
    <div className={`my-2 flex items-start gap-2 rounded-md border px-3 py-2 text-xs ${tone}`}>
      <span aria-hidden="true">{icon}</span>
      <p className="min-w-0 flex-1 whitespace-pre-wrap break-words">{item.text}</p>
    </div>
  );
}
