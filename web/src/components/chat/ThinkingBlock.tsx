/**
 * ThinkingBlock：思考过程折叠块（默认折叠——「思考过程」可展开）。
 *
 * 流式期间（streaming=true）自动展开滚动跟随；落定后折叠。
 */

import { useEffect, useRef, useState } from "react";

interface ThinkingBlockProps {
  text: string;
  streaming: boolean;
}

export default function ThinkingBlock({ text, streaming }: ThinkingBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const bodyRef = useRef<HTMLDivElement | null>(null);

  // 流式期间自动展开 + 贴底
  useEffect(() => {
    if (streaming) {
      setExpanded(true);
      const el = bodyRef.current;
      if (el) el.scrollTop = el.scrollHeight;
    }
  }, [streaming, text]);

  return (
    <div className="my-1 rounded-md border border-dashed border-line bg-space/60">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs text-ink-3 hover:text-ink-2"
        aria-expanded={expanded}
      >
        <span aria-hidden="true">{expanded ? "▾" : "▸"}</span>
        <span>思考过程</span>
        {streaming && (
          <span
            className="inline-block h-1.5 w-1.5 rounded-full bg-rick"
            style={{ animation: "rm-pulse 1s ease-in-out infinite" }}
            aria-label="思考中"
          />
        )}
      </button>
      {expanded && (
        <div
          ref={bodyRef}
          className="max-h-64 overflow-y-auto border-t border-dashed border-line px-3 py-2"
        >
          <p className="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-ink-2">
            {text}
            {streaming && (
              <span
                aria-hidden="true"
                className="ml-0.5 inline-block h-3 w-[6px] translate-y-[1px] rounded-[1px] bg-rick"
                style={{ animation: "rm-pulse 0.9s ease-in-out infinite" }}
              />
            )}
          </p>
        </div>
      )}
    </div>
  );
}
