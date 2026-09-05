/**
 * StreamText：流式文本直渲 + 光标闪烁。
 *
 * delta 拼接的 live 文本直接渲染（message_end 后由 MessageBubble 换用 Markdown
 * 落定版）——流式期间走轻量纯文本渲染（避免高频 markdown 重排卡顿），
 * 光标 ▮ 以 rm-pulse 呼吸闪烁。
 */

import { memo } from "react";

interface StreamTextProps {
  text: string;
  className?: string;
}

function StreamTextImpl({ text, className }: StreamTextProps) {
  return (
    <span className={`whitespace-pre-wrap break-words text-sm leading-relaxed ${className ?? ""}`}>
      {text}
      <span
        aria-hidden="true"
        className="ml-0.5 inline-block h-4 w-[7px] translate-y-[2px] rounded-[2px] bg-portal"
        style={{ animation: "rm-pulse 0.9s ease-in-out infinite" }}
      />
    </span>
  );
}

// 打字机文本每帧变 → text 变必然重渲；父组件（memo 后的 AssistantBubble）在
// text 未变时不会传新值——此处 memo 防父级无谓重渲传导。
const StreamText = memo(StreamTextImpl, (prev, next) => prev.text === next.text && prev.className === next.className);

export default StreamText;
