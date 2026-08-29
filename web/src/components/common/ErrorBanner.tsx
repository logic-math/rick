/**
 * 错误横幅（可关闭）。
 *
 * 错误二分（渲染分层共识）：可展示的服务/连接错误走本横幅；
 * agent 内容错误在内容区内联（chat 组件职责）。
 */

import { useEffect, useState } from "react";

interface ErrorBannerProps {
  message: string | null;
  /** 可选的清除回调（提供则显示关闭按钮） */
  onDismiss?: () => void;
}

export default function ErrorBanner({ message, onDismiss }: ErrorBannerProps) {
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    setVisible(true);
  }, [message]);

  if (!message || !visible) return null;

  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger"
    >
      <span aria-hidden="true" className="mt-0.5 shrink-0">
        ⚠
      </span>
      <p className="min-w-0 flex-1 break-words">{message}</p>
      {onDismiss && (
        <button
          type="button"
          aria-label="关闭错误提示"
          className="shrink-0 rounded px-1 text-danger/70 transition-colors hover:bg-danger/15 hover:text-danger"
          onClick={() => {
            setVisible(false);
            onDismiss();
          }}
        >
          ✕
        </button>
      )}
    </div>
  );
}
