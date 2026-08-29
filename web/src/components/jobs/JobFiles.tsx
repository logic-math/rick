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

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import type { TaskBrief } from "../../types";
import FileTree, { type FileTreeEntry } from "../knowledge/FileTree";
import Spinner from "../common/Spinner";
import FileContentView from "../knowledge/MarkdownView";

const TOO_LARGE_HINT = "文件过大，请在仓库查看";

interface JobFilesProps {
  workspaceId: string;
  jobId: string;
  tasks: TaskBrief[];
}

/** tasks → 约定文件清单 */
function conventionEntries(tasks: TaskBrief[]): FileTreeEntry[] {
  const entries: FileTreeEntry[] = [];
  for (const t of tasks) {
    const id = t.task_id;
    entries.push({ path: `plan/${id}.md` });
    entries.push({ path: `doing/tasks/${id}/act-path.md` });
    entries.push({ path: `doing/tasks/${id}/raw_session_coding.log` });
    entries.push({ path: `doing/debug/debug_${id}.md` });
  }
  entries.push({ path: "doing/tasks.json" });
  entries.push({ path: "doing/session_id" });
  entries.push({ path: "doing/raw_session_coding.log" });
  entries.push({ path: "doing/act-path.md" });
  entries.push({ path: "doing/prompts/doing_prompt.md" });
  entries.push({ path: "plan/requirement.md" });
  // 去重（tasks 空/重复时）+ 字典序
  const seen = new Set<string>();
  return entries
    .filter((e) => {
      if (seen.has(e.path)) return false;
      seen.add(e.path);
      return true;
    })
    .sort((a, b) => a.path.localeCompare(b.path));
}

export default function JobFiles({ workspaceId, jobId, tasks }: JobFilesProps) {
  const entries = useMemo(() => conventionEntries(tasks), [tasks]);
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
    <div className="grid min-h-0 gap-3 lg:grid-cols-[16rem_1fr]">
      <aside className="max-h-80 overflow-y-auto rounded-lg border border-line bg-surface/50 p-2 lg:max-h-none">
        <p className="px-1.5 pb-2 text-[11px] font-semibold uppercase tracking-wider text-ink-3">
          {jobId} 文件（约定视图）
        </p>
        <FileTree entries={entries} selectedPath={selectedPath} onSelect={onSelect} />
      </aside>
      <section className="min-w-0 overflow-y-auto rounded-lg border border-line bg-surface/30 p-4">
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
