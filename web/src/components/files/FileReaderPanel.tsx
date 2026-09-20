/**
 * FileReaderPanel：右侧富文本文件阅读器（聊天页就地查阅本地 md/日志）。
 *
 * 与 Dreams（知识库）页复用同一套渲染：FileTree 之外的 FileContentView
 * （knowledge/MarkdownView）——markdown → GFM 富文本，日志 → 等宽原样。
 *
 * 布局：桌面端与对话并排（可调比例基线 w-[46%] max-w-[46rem]），移动端整屏抽屉；
 * 关闭后对话恢复全宽。文件不存在/不在工作区内 → 面板内给出原因 + 复制路径，
 * 不改动对话状态（用户要求：任何时候都不破坏常驻对话）。
 */

import { useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import { ApiError } from "../../types";
import FileContentView, { isMarkdownPath } from "../knowledge/MarkdownView";
import Spinner from "../common/Spinner";

interface FileReaderPanelProps {
  workspaceId: string | null;
  /** 要打开的文件路径（null = 关闭） */
  path: string | null;
  onClose: () => void;
  /** 打开另一个本地文件（阅读器内的链接点击） */
  onOpen: (path: string) => void;
}

/** 面包屑式路径显示（过长时保留尾部） */
function PathHeader({ path }: { path: string }) {
  return (
    <span className="min-w-0 truncate font-mono text-[11px] text-ink-2" title={path}>
      {path}
    </span>
  );
}

export default function FileReaderPanel({
  workspaceId,
  path,
  onClose,
  onOpen,
}: FileReaderPanelProps) {
  const [content, setContent] = useState<string | null>(null);
  const [displayPath, setDisplayPath] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tooLarge, setTooLarge] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!path || !workspaceId) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    setTooLarge(false);
    setContent(null);
    setDisplayPath(path);
    api
      .readWorkspaceFile(workspaceId, path)
      .then((res) => {
        if (cancelled) return;
        setContent(res.content);
        setDisplayPath(res.path || path);
      })
      .catch((e: unknown) => {
        if (cancelled) return;
        const msg = e instanceof Error ? e.message : String(e);
        if (e instanceof ApiError && e.code === "file_too_large") setTooLarge(true);
        else setError(msg);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId, path]);

  const copyPath = useMemo(
    () => () => {
      if (!path) return;
      void navigator.clipboard?.writeText(path).catch(() => {});
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    },
    [path],
  );

  if (!path) return null;

  return (
    <aside
      className="fixed inset-0 z-40 flex min-h-0 flex-col border-line bg-space/95 backdrop-blur md:static md:z-auto md:w-[46%] md:max-w-[46rem] md:shrink-0 md:border-l md:bg-space/60 md:backdrop-blur-none"
      aria-label="文件阅读器"
    >
      {/* 头部 */}
      <header className="flex shrink-0 items-center gap-2 border-b border-line px-3 py-2">
        <span className="shrink-0 text-xs" aria-hidden="true">
          {isMarkdownPath(displayPath) ? "📄" : "🗒"}
        </span>
        <PathHeader path={displayPath} />
        <div className="ml-auto flex shrink-0 items-center gap-1">
          <button
            type="button"
            onClick={copyPath}
            title="复制完整路径"
            className="rounded border border-line px-1.5 py-0.5 text-[10px] text-ink-3 hover:border-portal/50 hover:text-portal"
          >
            {copied ? "已复制" : "复制路径"}
          </button>
          <button
            type="button"
            onClick={onClose}
            aria-label="关闭文件阅读器"
            title="关闭（Esc）"
            className="rounded border border-line px-1.5 py-0.5 text-[10px] text-ink-3 hover:border-danger/50 hover:text-danger"
          >
            ✕
          </button>
        </div>
      </header>

      {/* 内容 */}
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {loading && <Spinner label="读取文件…" />}
        {!loading && tooLarge && (
          <p className="rounded-lg border border-morty/40 bg-morty/10 px-3 py-6 text-center text-xs text-morty">
            文件过大，请在仓库中查看（路径已可复制）
          </p>
        )}
        {!loading && !tooLarge && error && (
          <div className="flex flex-col items-center gap-2 rounded-lg border border-line bg-surface/40 px-4 py-6 text-center">
            <p className="text-xs text-ink-2">⚠ {error}</p>
            <p className="text-[11px] leading-relaxed text-ink-3">
              只能就地打开工作区内（或 <span className="font-mono">~/.rick</span>）的文件。
              该路径可能是相对路径、目录，或者不在本工作区。
            </p>
            <button
              type="button"
              onClick={copyPath}
              className="rounded border border-line px-2 py-1 text-[10px] text-ink-3 hover:border-portal/50 hover:text-portal"
            >
              {copied ? "已复制" : "复制路径"}
            </button>
          </div>
        )}
        {!loading && !tooLarge && !error && content !== null && (
          <FileContentView
            path={displayPath}
            content={content}
            onOpenFile={onOpen}
          />
        )}
      </div>
    </aside>
  );
}
