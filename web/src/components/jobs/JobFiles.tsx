/**
 * job 文件浏览器（plan/ / doing/ 目录——约定驱动的树 + 按需加载内容）。
 *
 * API 契约只有 getJobFile(path)（按路径取内容），无目录列举端点——树结构按
 * rick 的目录约定推导：
 * - plan/task{N}.md（N 来自 tasks 的 task_id）+ plan/prompts/
 * - doing/tasks.json、doing/session_id、doing/raw_session_coding.log
 * - doing/tasks/task{N}/act-path.md、doing/tasks/task{N}/raw_session_coding.log
 * - doing/debug/debug_task{N}.md、doing/prompts/doing_prompt.md
 * 点击时按需 getJobFile——404/不存在 → 行内提示（约定路径未产出属正常）。
 * 超限（400 file_too_large）→「文件过大，请在仓库查看」。
 */

import { useCallback, useEffect, useState } from "react";
import { api } from "../../api/client";
import FileTree, { type FileTreeEntry } from "../knowledge/FileTree";
import Spinner from "../common/Spinner";
import FileContentView from "../knowledge/MarkdownView";

const TOO_LARGE_HINT = "文件过大，请在仓库查看";



export default function JobFiles({ workspaceId, jobId }: { workspaceId: string; jobId: string }) {
  const [entries, setEntries] = useState<FileTreeEntry[]>([]);
  const [treeLoading, setTreeLoading] = useState(true);
  const [treeError, setTreeError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setTreeLoading(true);
    setTreeError(null);
    setEntries([]);
    api
      .listJobFiles(workspaceId, jobId)
      .then((files) => {
        if (cancelled) return;
        setEntries(files.map((f) => ({ path: f.path })).sort((a, b) => a.path.localeCompare(b.path)));
      })
      .catch((e: unknown) => {
        if (!cancelled) setTreeError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setTreeLoading(false);
      });
    return () => { cancelled = true; };
  }, [workspaceId, jobId]);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tooLarge, setTooLarge] = useState(false);

  useEffect(() => {
    // 切换 job 重置
    setSelectedPath(null);
    setContent(null);
    setError(null);
    setTooLarge(false);
  }, [jobId]);

  const onSelect = useCallback(
    (path: string) => {
      setSelectedPath(path);
      setContent(null);
      setError(null);
      setTooLarge(false);
      setLoading(true);
      api
        .getJobFile(workspaceId, jobId, path)
        .then((res) => {
          setContent(res.content);
        })
        .catch((err: unknown) => {
          const msg = err instanceof Error ? err.message : String(err);
          if (/too_large|过大|400/.test(msg)) {
            setTooLarge(true);
          } else {
            setError(msg);
          }
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [workspaceId, jobId],
  );

  return (
    <div className="grid min-h-0 flex-1 gap-3 lg:grid-cols-[18rem_1fr]">
      <aside className="min-h-0 overflow-y-auto rounded-lg border border-line bg-surface/50 p-2">
        <p className="px-1.5 pb-2 text-[11px] font-semibold uppercase tracking-wider text-ink-3">
          {jobId} 文件
        </p>
        {treeLoading && <Spinner size={14} label="加载文件列表…" />}
        {treeError && <p className="px-1.5 text-[11px] text-danger">{treeError}</p>}
        {!treeLoading && !treeError && entries.length === 0 && (
          <p className="px-1.5 text-[11px] text-ink-3">暂无文件</p>
        )}
        {!treeLoading && entries.length > 0 && (
          <FileTree entries={entries} selectedPath={selectedPath} onSelect={onSelect} />
        )}
      </aside>
      <section className="min-h-0 min-w-0 overflow-y-auto rounded-lg border border-line bg-surface/30 p-4">
        {loading && <Spinner label="读取文件…" />}
        {!loading && error && (
          <p className="rounded-lg border border-line bg-surface px-3 py-6 text-center text-xs text-ink-3">
            {error}
          </p>
        )}
        {!loading && tooLarge && (
          <p className="rounded-lg border border-morty/40 bg-morty/10 px-3 py-6 text-center text-xs text-morty">
            {TOO_LARGE_HINT}
          </p>
        )}
        {!loading && !error && !tooLarge && !content && (
          <p className="px-3 py-6 text-center text-xs text-ink-3">
            选择左侧文件查看（树按 rick 目录约定推导——未产出的约定路径会提示不存在）
          </p>
        )}
        {!loading && !error && !tooLarge && content !== null && selectedPath && (
          <>
            <p className="mb-3 truncate font-mono text-xs text-ink-3" title={selectedPath}>
              {jobId}/{selectedPath}
            </p>
            <FileContentView path={selectedPath} content={content} />
          </>
        )}
      </section>
    </div>
  );
}
