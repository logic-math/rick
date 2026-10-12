/**
 * humans 浏览器（.rick/draft 只读）——human-loop 学习材料与进度。
 *
 * - 左：draft 文件树（GET humans/tree）
 * - 右：文件内容（GET humans/file → MarkdownView 分派）
 * - 移动端（<768px）：树折叠为顶部下拉选择（顶层目录即 optgroup）
 * - 空态：尚无学习记录 → 引导以 human-loop 模式开始一次学习
 *
 * 交互镜像 KnowledgeBrowser（同一 FileTree / MarkdownView / EmptyState），
 * **不修改** KnowledgeBrowser 以避免知识库回归。human 明确拒绝一切额外
 * 装饰性 UI 面板——本页只有纯文件树 + 内容查看。
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import type { KnowledgeFileNode } from "../../types";
import FileTree, { buildTree, type FileTreeEntry } from "../knowledge/FileTree";
import Spinner from "../common/Spinner";
import ErrorBanner from "../common/ErrorBanner";
import EmptyState from "../common/EmptyState";
import FileContentView from "../knowledge/MarkdownView";

interface HumansBrowserProps {
  workspaceId: string;
}

type TreeNode = ReturnType<typeof buildTree>[number];

/** 移动端下拉：按顶层目录（loops / rfc）分组，递归收集所有叶子文件 */
function MobilePicker({
  tree,
  selectedPath,
  onSelect,
}: {
  tree: TreeNode[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
}) {
  const groups = useMemo(() => {
    const walk = (node: TreeNode, out: { path: string }[]): void => {
      if (node.isDir) {
        node.children.forEach((c) => walk(c, out));
      } else {
        out.push({ path: node.path });
      }
    };
    return tree.map((root) => {
      const files: { path: string }[] = [];
      walk(root, files);
      return { root, files };
    });
  }, [tree]);

  return (
    <select
      className="w-full rounded-lg border border-line bg-surface px-2 py-1.5 text-xs text-ink"
      value={selectedPath ?? ""}
      onChange={(e) => {
        if (e.target.value) onSelect(e.target.value);
      }}
      aria-label="选择学习材料文件"
    >
      <option value="" disabled>
        选择文件…
      </option>
      {groups.map((g) => (
        <optgroup key={g.root.path} label={g.root.name}>
          {g.files.map((f) => (
            <option key={f.path} value={f.path}>
              {f.path}
            </option>
          ))}
        </optgroup>
      ))}
    </select>
  );
}

export default function HumansBrowser({ workspaceId }: HumansBrowserProps) {
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
      .humansTree(workspaceId)
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
        .humansFile(workspaceId, path)
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

  if (loading) return <Spinner center label="加载学习材料…" />;
  if (error) return <ErrorBanner message={error} />;

  const fileCount = entries.length;
  if (fileCount === 0) {
    return (
      <EmptyState
        message="尚无学习记录——在会话中以 human-loop 模式开始一次学习"
        hint="学习材料与进度（mission / research / lessons / judgment / rfc）会出现在 .rick/draft/ 下，可在此复习"
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
            学习材料 · {fileCount} 文件
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
            <EmptyState
              message="选择左侧文件浏览"
              hint="loops=学习记录（mission / research / lessons / judgment）；rfc=历史总结"
            />
          )}
          {!fileLoading && !fileError && file && (
            <>
              <p className="mb-3 truncate font-mono text-xs text-ink-3" title={file.path}>
                {file.path}
              </p>
              <FileContentView
                path={file.path}
                content={file.content}
                // md 里的相对链接就地切换（`.rick/draft/x`、`.rick/x`、`x` 都认），
                // 外链仍走新标签（LocalAwareLink 的重定向语义）
                onOpenFile={(p) => {
                  const rel = p
                    .replace(/^\.?\/?\.rick\/draft\//, "")
                    .replace(/^\.?\/?\.rick\//, "")
                    .replace(/^\.\//, "");
                  onSelect(entries.some((e) => e.path === rel) ? rel : p);
                }}
              />
            </>
          )}
        </section>
      </div>
    </div>
  );
}
