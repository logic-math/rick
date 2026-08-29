/**
 * ExtensionUIDialog：extension_ui 双向桥（web 弹窗侧）。
 *
 * 自包含约束：不 import 同层 task10 的 components/common/（并行中间态缺模块
 * tsc 红）——样式 token 直接用 theme.css 变量 / Tailwind @theme utility。
 *
 * pi rpc extension_ui_request 子协议（rpc.md Extension UI Protocol 节）：
 * - 对话框方法（select/confirm/input/editor）：agent 侧阻塞等待——本组件渲染表单，
 *   提交/取消经 api.uiResponse 回写 extension_ui_response（request_id 对应）
 * - fire-and-forget（notify/setStatus/...）：notify 渲染为可关闭 toast；其余忽略
 * - 同 session 多个 pending 请求排队展示（agent 逐个阻塞——全部渲染，先进先答）
 * - dompurify 双保险清洗服务端字符串（React 默认转义之外再剥标签）
 */

import { useState } from "react";
import { api } from "../../api/client";
import { sanitizePlainText } from "../chat/sanitize";
import type { ExtensionUIDialogRequest } from "../chat/viewModel";

// ============================================================
// 通知 toast（fire-and-forget notify）
// ============================================================

interface NotifyToastsProps {
  sessionId: string;
  notifications: Array<{ id: string; message: string; notifyType: string }>;
}

export function NotifyToasts({ notifications }: NotifyToastsProps) {
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());
  if (notifications.length === 0) return null;

  const visible = notifications.filter((n) => !dismissed.has(n.id));
  if (visible.length === 0) return null;

  const dismiss = (id: string) => {
    setDismissed((prev) => new Set(prev).add(id));
  };

  return (
    <div className="pointer-events-none absolute right-3 top-3 z-20 flex w-72 flex-col gap-2">
      {visible.map((n) => {
        const tone =
          n.notifyType === "error"
            ? "border-danger/60 bg-danger/10 text-danger"
            : n.notifyType === "warning"
              ? "border-morty/60 bg-morty/10 text-morty"
              : "border-line bg-surface-raised text-ink-2";
        return (
          <div
            key={n.id}
            className={`pointer-events-auto flex items-start gap-2 rounded-md border px-3 py-2 text-xs shadow-lg ${tone}`}
            role="status"
          >
            <span aria-hidden="true">
              {n.notifyType === "error" ? "⛔" : n.notifyType === "warning" ? "⚠" : "ℹ"}
            </span>
            <p className="min-w-0 flex-1 whitespace-pre-wrap break-words">
              {sanitizePlainText(n.message)}
            </p>
            <button
              type="button"
              onClick={() => dismiss(n.id)}
              className="shrink-0 text-ink-3 hover:text-ink"
              aria-label="关闭通知"
            >
              ✕
            </button>
          </div>
        );
      })}
    </div>
  );
}

// ============================================================
// 对话框（select/confirm/input/editor）
// ============================================================

interface ExtensionUIDialogProps {
  sessionId: string;
  requests: ExtensionUIDialogRequest[];
  /** 响应回执后的本地回调（错误提示等） */
  onResponse?: (err: string | null) => void;
}

export default function ExtensionUIDialog({ sessionId, requests, onResponse }: ExtensionUIDialogProps) {
  const [busy, setBusy] = useState(false);
  const [answered, setAnswered] = useState<Set<string>>(new Set());

  const pending = requests.filter((r) => !answered.has(r.id));
  if (pending.length === 0) return null;

  const respond = async (req: ExtensionUIDialogRequest, payload: Record<string, unknown>) => {
    if (busy) return;
    setBusy(true);
    try {
      await api.uiResponse(sessionId, { request_id: req.id, ...payload });
      setAnswered((prev) => new Set(prev).add(req.id));
      onResponse?.(null);
    } catch (err) {
      onResponse?.(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-2 px-4 pb-2" role="dialog" aria-label="会话交互请求">
      {pending.length > 1 && (
        <div className="text-right text-[10px] text-ink-3">{pending.length} 个待答请求（按序）</div>
      )}
      {pending.map((req) => (
        <DialogCard key={req.id} req={req} busy={busy} respond={respond} />
      ))}
    </div>
  );
}

interface DialogCardProps {
  req: ExtensionUIDialogRequest;
  busy: boolean;
  respond: (
    req: ExtensionUIDialogRequest,
    payload: Record<string, unknown>,
  ) => Promise<void>;
}

function DialogCard({ req, busy, respond }: DialogCardProps) {
  const [text, setText] = useState(
    req.method === "editor" ? (req.prefill ?? "") : "",
  );
  const [selected, setSelected] = useState<string | null>(null);

  const title = sanitizePlainText(req.title);
  const message = sanitizePlainText(req.message);

  return (
    <div className="rounded-xl border border-portal/40 bg-surface-raised p-4 shadow-xl">
      {/* 标题行 */}
      <div className="mb-2 flex items-center gap-2">
        <span className="rounded bg-portal/15 px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wide text-portal">
          {req.method}
        </span>
        {title && <span className="text-sm font-medium text-ink">{title}</span>}
      </div>
      {message && <p className="mb-3 text-xs text-ink-2">{message}</p>}

      {/* 方法分派表单 */}
      {req.method === "select" && (
        <div className="mb-3 flex flex-col gap-1.5">
          {(req.options ?? []).map((opt) => (
            <button
              key={opt}
              type="button"
              disabled={busy}
              onClick={() => setSelected(opt)}
              className={`rounded-md border px-3 py-1.5 text-left text-sm transition-colors ${
                selected === opt
                  ? "border-portal bg-portal-soft text-portal"
                  : "border-line bg-space text-ink-2 hover:border-portal/40 hover:text-ink"
              }`}
            >
              {sanitizePlainText(opt)}
            </button>
          ))}
        </div>
      )}

      {req.method === "input" && (
        <input
          type="text"
          value={text}
          disabled={busy}
          placeholder={sanitizePlainText(req.placeholder)}
          onChange={(e) => setText(e.target.value)}
          className="mb-3 w-full rounded-md border border-line bg-space px-3 py-2 text-sm text-ink outline-none placeholder:text-ink-3 focus:border-portal/50"
        />
      )}

      {req.method === "editor" && (
        <textarea
          value={text}
          disabled={busy}
          onChange={(e) => setText(e.target.value)}
          rows={Math.min(12, Math.max(4, text.split("\n").length))}
          className="mb-3 w-full resize-y rounded-md border border-line bg-space px-3 py-2 font-mono text-xs text-ink outline-none focus:border-portal/50"
        />
      )}

      {/* 动作条 */}
      <div className="flex items-center justify-end gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void respond(req, { cancelled: true })}
          className="rounded-md px-3 py-1.5 text-xs text-ink-3 hover:text-ink"
        >
          取消
        </button>

        {req.method === "confirm" ? (
          <>
            <button
              type="button"
              disabled={busy}
              onClick={() => void respond(req, { confirmed: false })}
              className="rounded-md border border-line px-3 py-1.5 text-xs text-ink-2 hover:bg-white/5"
            >
              否
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => void respond(req, { confirmed: true })}
              className="rounded-md bg-portal/20 px-4 py-1.5 text-xs font-medium text-portal hover:bg-portal/30 disabled:opacity-40"
            >
              是
            </button>
          </>
        ) : (
          <button
            type="button"
            disabled={
              busy ||
              (req.method === "select" && !selected) ||
              ((req.method === "input" || req.method === "editor") && !text.trim())
            }
            onClick={() =>
              void respond(req, {
                value: req.method === "select" ? (selected ?? "") : text,
              })
            }
            className="rounded-md bg-portal/20 px-4 py-1.5 text-xs font-medium text-portal hover:bg-portal/30 disabled:opacity-40"
          >
            提交{busy ? "…" : ""}
          </button>
        )}
      </div>
    </div>
  );
}
