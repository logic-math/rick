/**
 * 模态对话框基类（extui 弹窗 / NewSessionModal 复用）。
 *
 * - Esc 关闭 + 点击遮罩关闭（可通过 closable=false 关掉）
 * - 标题 + 关闭按钮 + 内容槽（children）
 * - 深色 R&M 视觉；body 滚动锁定
 */

import { useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface DialogProps {
  open: boolean;
  title: ReactNode;
  onClose: () => void;
  /** false = 禁止关闭（关键确认场景） */
  closable?: boolean;
  /** 内容区附加 class（如宽度） */
  contentClassName?: string;
  children: ReactNode;
}

export default function Dialog({
  open,
  title,
  onClose,
  closable = true,
  contentClassName = "w-full max-w-lg",
  children,
}: DialogProps) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && closable) onClose();
    };
    window.addEventListener("keydown", onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [open, closable, onClose]);

  if (!open) return null;

  // createPortal 渲染到 document.body：避免 fixed 定位被 backdrop-filter/
  // transform 祖先（侧边栏 aside 带 backdrop-blur）劫持为相对祖先定位——
  // 否则弹窗被锁在侧边栏内无法全屏（job_36 用户实测反馈）。z-[999]
  // 高于侧边栏 z-30 / 移动端遮罩 z-20。
  return createPortal(
    <div
      className="fixed inset-0 z-[999] flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && closable) onClose();
      }}
      role="presentation"
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`flex max-h-[85vh] flex-col overflow-hidden rounded-xl border border-line bg-surface-raised shadow-2xl shadow-black/50 ${contentClassName}`}
        onMouseDown={(e) => e.stopPropagation()}
      >
        <header className="flex items-center gap-3 border-b border-line px-4 py-3">
          <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-ink">
            {title}
          </h2>
          {closable && (
            <button
              type="button"
              onClick={onClose}
              aria-label="关闭"
              className="rounded-md px-2 py-0.5 text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
            >
              ✕
            </button>
          )}
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto p-4">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
