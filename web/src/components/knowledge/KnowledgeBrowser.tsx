/**
 * 知识库浏览器（.rick/{domain,loops,skills} 只读）。
 *
 * - 左：三根文件树（GET knowledge/tree）
 * - 右：文件内容（GET knowledge/file → MarkdownView 分派）
 * - 移动端（<768px）：树折叠为顶部下拉选择（目录即 optgroup）
 * - 超限（400 file_too_large）：显示「文件过大，请在仓库查看」
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import type { KnowledgeFileNode } from "../../types";
import FileTree, { buildTree, type FileTreeEntry } from "./FileTree";
import Spinner from "../common/Spinner";
import ErrorBanner from "../common/ErrorBanner";
import EmptyState from "../common/EmptyState";
import FileContentView from "./MarkdownView";

const ROOTS = ["domain", "loops", "skills"] as const;

interface KnowledgeBrowserProps {
  workspaceId: string;
}

/** 移动端下拉用的平铺分组 */
function MobilePicker({
  tree,
  selectedPath,
  onSelect,
}: {
  tree: ReturnType<typeof buildTree>;
  selectedPath: string | null;
  onSelect: (path: string) => void;
}) {
  return (
    <select
      className="w-full rounded-lg border border-line bg-surface px-2 py-1.5 text-xs text-ink"
      value={selectedPath ?? ""}
      onChange={(e) => {
        if (e.target.value) onSelect(e.target.value);
      }}
      aria-label="选择知识库文件"
    >
      <option value="" disabled>
        选择文件…
      </option>
      {tree
        .filter((n) => ROOTS.includes(n.name as (typeof ROOTS)[number]))
        .map((root) => (
          <optgroup key={root.path} label={root.name}>
            {root.children
              .flatMap((c) => (c.isDir ? c.children : [c]))
              .map((f) => (
                <option key={f.path} value={f.path}>
                  {f.path}
                </option>
              ))}
          </optgroup>
        ))}
    </select>
  );
}

export default function KnowledgeBrowser({ workspaceId }: KnowledgeBrowserProps) {
  const [tree, setTree] = useState<KnowledgeFileNode[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [file, setFile] = useState<{ path: string; content: string } | null>(null);
  const [fileLoading, setFileLoading] = useState(false);
  const [fileError, setFileError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    setTree(null);
    setSelectedPath(null);
    setFile(null);
    api
      .knowledgeTree(workspaceId)
      .then((res) => {
        if (!cancelled) setTree(res.tree ?? []);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId]);

  const onSelect = useCallback(
    (path: string) => {
      setSelectedPath(path);
      setFile(null);
      setFileError(null);
      setFileLoading(true);
      api
        .knowledgeFile(workspaceId, path)
        .then((res) => {
          setFile({ path: res.path, content: res.content });
        })
        .catch((err: unknown) => {
          setFileError(err instanceof Error ? err.message : String(err));
        })
        .finally(() => {
          setFileLoading(false);
        });
    },
    [workspaceId],
  );

  const entries: FileTreeEntry[] = useMemo(
    () => (tree ?? []).map((n) => ({ path: n.path, size: n.size })),
    [tree],
  );

  if (loading) return <Spinner center label="加载知识库…" />;
  if (error) return <ErrorBanner message={error} />;

  const fileCount = entries.length;
  if (fileCount === 0) {
    return (
      <EmptyState
        message="知识库为空"
        hint="domain / loops / skills 下还没有可浏览的文件（learning/dream 沉淀后出现）"
      />
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      {/* 移动端下拉（<768px） */}
      <div className="md:hidden">
        <MobilePicker
          tree={buildTree(entries)}
          selectedPath={selectedPath}
          onSelect={onSelect}
        />
      </div>

      <div className="flex min-h-0 flex-1 gap-4 max-md:flex-col">
        {/* 桌面端树（≥768px） */}
        <aside className="hidden w-64 shrink-0 overflow-y-auto rounded-lg border border-line bg-surface/50 p-2 md:block xl:w-80">
          <p className="px-1.5 pb-2 text-[11px] font-semibold uppercase tracking-wider text-ink-3">
            知识库 · {fileCount} 文件
          </p>
          <FileTree
            entries={entries}
            selectedPath={selectedPath}
            onSelect={onSelect}
          />
        </aside>

        {/* 内容区 */}
        <section className="min-w-0 flex-1 overflow-y-auto rounded-lg border border-line bg-surface/30 p-4">
          {fileLoading && <Spinner label="读取文件…" />}
          {!fileLoading && fileError && (
            <ErrorBanner message={fileError} onDismiss={() => setFileError(null)} />
          )}
          {!fileLoading && !fileError && !file && (
            <EmptyState message="选择左侧文件浏览" hint="domain=代码事实 / loops=工作流 / skills=原子能力" />
          )}
          {!fileLoading && !fileError && file && (
            <>
              <p className="mb-3 truncate font-mono text-xs text-ink-3" title={file.path}>
                {file.path}
              </p>
              <FileContentView path={file.path} content={file.content} />
            </>
          )}
        </section>
      </div>
    </div>
  );
}
