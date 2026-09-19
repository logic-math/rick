/**
 * jobs 列表（工作区维度）。
 *
 * - 卡片：job_id / 相对更新时间 / 任务进度点阵（每 task 一个色点）+ 完成度摘要
 * - 点击卡片 → onOpen(job_id)（路由跳转详情由上层决定）
 * - 数据源：jobs store（REST 快照 + SSE jobs_update 实时）
 */

import { useEffect, useMemo, useState } from "react";
import { useJobsStore } from "../../stores/jobs";
import { api } from "../../api/client";
import type { JobSummary, TaskBrief } from "../../types";
import Spinner from "../common/Spinner";
import ErrorBanner from "../common/ErrorBanner";
import EmptyState from "../common/EmptyState";

/** 任务状态 → 点色 */
function dotColorOf(status: string): string {
  switch (status) {
    case "running":
      return "#39ff88";
    case "success":
      return "#a6c8dd";
    case "error":
      return "#ffd54a";
    default:
      return "#5c6690";
  }
}

function relativeTime(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Date.now() - t;
  if (diff < 60_000) return "刚刚";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`;
  return `${Math.floor(diff / 86_400_000)} 天前`;
}

function ProgressDots({ tasks }: { tasks: TaskBrief[] }) {
  const dots = tasks.slice(0, 24);
  const rest = tasks.length - dots.length;
  return (
    <span className="flex flex-wrap items-center gap-1" aria-hidden="true">
      {dots.map((t) => (
        <span
          key={t.task_id}
          title={`${t.task_id}: ${t.status}`}
          className="inline-block h-1.5 w-1.5 rounded-full"
          style={{ backgroundColor: dotColorOf(t.status) }}
        />
      ))}
      {rest > 0 && <span className="text-[10px] text-ink-3">+{rest}</span>}
    </span>
  );
}

export function jobProgress(job: JobSummary): { done: number; total: number; failed: number } {
  const total = job.tasks.length;
  const done = job.tasks.filter((t) => t.status === "success").length;
  const failed = job.tasks.filter((t) => t.status === "error").length;
  return { done, total, failed };
}

interface JobsListProps {
  workspaceId: string;
  /** 点击卡片（job 详情路由） */
  onOpen: (jobId: string) => void;
}

function EmptyJobsView({ workspaceId }: { workspaceId: string }) {
  // 主列表（进行中）为空：区分“从无 job”与“全部已完成自动归档”
  const [anyJob, setAnyJob] = useState<boolean | null>(null);
  useEffect(() => {
    let cancelled = false;
    api
      .listJobs(workspaceId, true)
      .then((list) => {
        if (!cancelled) setAnyJob(list.length > 0);
      })
      .catch(() => {
        if (!cancelled) setAnyJob(false);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId]);
  if (anyJob === true) {
    return (
      <EmptyState
        message="暂无进行中的 job"
        hint="已完成的 job 会自动归档——展开下方「已归档」可查看"
      />
    );
  }
  return (
    <EmptyState
      message="该工作区还没有 job"
      hint="plan/doing 跑起来后 tasks.json 会出现在这里"
    />
  );
}

export default function JobsList({ workspaceId, onOpen }: JobsListProps) {
  const load = useJobsStore((s) => s.load);
  const version = useJobsStore((s) => s.version);
  const getJobs = useJobsStore((s) => s.getJobs);
  const jobs = useMemo(
    () => getJobs(workspaceId),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [workspaceId, version, getJobs],
  );
  const storeLoading = useJobsStore((s) => s.loadingByWorkspace.get(workspaceId) ?? false);
  const [error, setError] = useState<string | null>(null);
  // 归档中 job_id 集合（防连击）
  const [archiving, setArchiving] = useState<ReadonlySet<string>>(new Set());

  useEffect(() => {
    let cancelled = false;
    setError(null);
    load(workspaceId).catch((e: unknown) => {
      if (!cancelled) setError(e instanceof Error ? e.message : String(e));
    });
    return () => {
      cancelled = true;
    };
  }, [workspaceId, load]);

  async function archive(jobId: string): Promise<void> {
    setError(null);
    setArchiving((s) => new Set(s).add(jobId));
    try {
      await api.archiveJob(workspaceId, jobId);
      // 归档成功 → 重新拉列表（store 快照剔除该 job）
      await load(workspaceId);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setArchiving((s) => {
        const next = new Set(s);
        next.delete(jobId);
        return next;
      });
    }
  }

  function isComplete(job: JobSummary): boolean {
    return job.tasks.length > 0 && job.tasks.every((t) => t.status === "success");
  }

  if (storeLoading && jobs.length === 0 && !error) return <Spinner center label="加载 jobs…" />;
  if (error) return <ErrorBanner message={error} onDismiss={() => setError(null)} />;
  if (jobs.length === 0) {
    return <EmptyJobsView workspaceId={workspaceId} />;
  }

  return (
    <ul className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
      {jobs.map((job) => {
        const { done, total, failed } = jobProgress(job);
        const complete = isComplete(job);
        const busy = archiving.has(job.job_id);
        return (
          <li key={job.job_id} className="group flex flex-col gap-1.5 rounded-lg border border-line bg-surface/50 p-3 transition-colors hover:border-portal/50">
            <button
              type="button"
              onClick={() => onOpen(job.job_id)}
              className="flex w-full flex-col gap-2 text-left"
            >
              <div className="flex items-baseline gap-2">
                <span className="font-mono text-sm font-semibold text-ink">{job.job_id}</span>
                <span className="ml-auto text-[10px] text-ink-3" title={job.updated_at}>
                  {relativeTime(job.updated_at)}
                </span>
              </div>
              <ProgressDots tasks={job.tasks} />
              <p className="text-xs text-ink-3">
                {job.stage === "planned" ? (
                  <span className="text-morty">plan 就绪 · 未执行 doing</span>
                ) : (
                  <>
                    {done}/{total} 完成
                    {failed > 0 ? ` · ${failed} 失败 ⚠` : done === total && total > 0 ? " ✅" : ""}
                  </>
                )}
              </p>
            </button>
            {complete && (
              <button
                type="button"
                onClick={() => void archive(job.job_id)}
                disabled={busy}
                title="归档：从列表隐藏（可随时恢复）"
                className="flex w-fit items-center gap-1 rounded-md border border-line px-2 py-0.5 text-[10px] text-ink-3 transition-colors hover:border-portal/50 hover:text-portal disabled:opacity-40"
              >
                {busy ? "归档中…" : "📦 归档"}
              </button>
            )}
          </li>
        );
      })}
    </ul>
  );
}
