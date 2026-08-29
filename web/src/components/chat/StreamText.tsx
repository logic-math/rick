/**
 * StreamText：流式文本直渲 + 光标闪烁。
 *
 * delta 拼接的 live 文本直接渲染（message_end 后由 MessageBubble 换用 Markdown
 * 落定版）——流式期间走轻量纯文本渲染（避免高频 markdown 重排卡顿），
 * 光标 ▮ 以 rm-pulse 呼吸闪烁。
 */

interface StreamTextProps {
  text: string;
  className?: string;
}

export default function StreamText({ text, className }: StreamTextProps) {
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
