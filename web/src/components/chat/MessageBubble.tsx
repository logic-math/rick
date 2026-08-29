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

export function UserBubble({ item }: { item: UserItem }) {
  return (
    <div className="my-2 flex justify-end">
      <div className="max-w-[80%] rounded-2xl rounded-br-sm border border-nebula/40 bg-surface-raised px-4 py-2.5">
        <p className="whitespace-pre-wrap break-words text-sm leading-relaxed text-ink">
          {item.text}
        </p>
      </div>
    </div>
  );
}

export function AssistantBubble({ item }: { item: AssistantTextItem }) {
  return (
    <div className="my-2 flex gap-2.5">
      <div className="mt-0.5 shrink-0" aria-hidden="true">
        <Portal size={20} />
      </div>
      <div className="min-w-0 flex-1">
        {item.streaming ? (
          // 流式期间：轻量纯文本直渲（防高频 markdown 重排卡顿）+ 光标闪烁
          <StreamText text={item.text} />
        ) : (
          // message_end 落定：authoritative markdown 渲染
          <Markdown>{item.text}</Markdown>
        )}
      </div>
    </div>
  );
}

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
