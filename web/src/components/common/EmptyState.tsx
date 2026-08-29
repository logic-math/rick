/**
 * 空态（飞碟插画 + 主文案 + 引导动作槽）。
 */

import type { ReactNode } from "react";
import Saucer from "../starfield/Saucer";

interface EmptyStateProps {
  /** 主文案（如「还没有注册任何工作区」） */
  message: string;
  /** 次级说明（可选） */
  hint?: string;
  /** 引导动作（如「添加工作区」按钮——槽位由调用方给） */
  action?: ReactNode;
}

export default function EmptyState({ message, hint, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
      <div className="relative flex h-20 items-end justify-center">
        {/* 光束点缀（静态装饰） */}
        <div
          aria-hidden="true"
          className="absolute bottom-0 h-10 w-24"
          style={{
            background:
              "linear-gradient(to top, rgba(57,255,136,0.14), transparent)",
            clipPath: "polygon(35% 0, 65% 0, 100% 100%, 0% 100%)",
          }}
        />
        <Saucer size={64} />
      </div>
      <p className="text-sm text-ink-2">{message}</p>
      {hint && <p className="max-w-sm text-xs text-ink-3">{hint}</p>}
      {action && <div className="mt-1">{action}</div>}
    </div>
  );
}
