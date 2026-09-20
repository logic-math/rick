/**
 * job 详情：TaskStatus 表（TaskBoard 任务卡横排简化版）+ job 文件浏览器。
 *
 * - 头部：job_id / 更新时间 / 完成度摘要 + 返回列表按钮
 * - 任务行：复用 TaskCard compact 形态（状态徽标 + commit 短哈希）
 * - JobFiles：plan/doing 约定树 + markdown/日志内容视图
 * - 数据源：jobs store（同 JobsList 的快照——SSE 实时同步）
 */

import { useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import { useJobsStore } from "../../stores/jobs";
import Button from "../common/Button";
import EmptyState from "../common/EmptyState";
import { TaskCard } from "../monitor/TaskBoard";
import JobFiles from "./JobFiles";
import { jobProgress } from "./JobsList";

interface JobDetailProps {
  workspaceId: string;
  jobId: string;
  /** 返回列表（路由回退由上层决定） */
  onBack: () => void;
}

export default function JobDetail({ workspaceId, jobId, onBack }: JobDetailProps) {
  const version = useJobsStore((s) => s.version);
  const getJobs = useJobsStore((s) => s.getJobs);
  const jobs = useMemo(
    () => getJobs(workspaceId),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [workspaceId, version, getJobs],
  );
  const job = jobs.find((j) => j.job_id === jobId) ?? null;

  // 归档 job 不在默认列表里——挂载时额外拉 include_archived 兜底查找
  // （否则归档 job 的详情页永远显示「不在快照中」，文件树根本不渲染——用户实测
  // 「job 页面打开的文件都打不开」的根因）
  const [archivedJob, setArchivedJob] = useState<typeof job>(null);
  useEffect(() => {
    if (job || !workspaceId || !jobId) return;
    let cancelled = false;
    api
      .listJobs(workspaceId, true)
      .then((list) => {
        if (cancelled) return;
        setArchivedJob(list.find((j) => j.job_id === jobId) ?? null);
      })
      .catch(() => {});
    return () => { cancelled = true; };
  }, [job, workspaceId, jobId]);

  const effectiveJob = job ?? archivedJob;

  if (!effectiveJob) {
    return (
      <div className="flex flex-col items-center gap-3 py-10">
        <EmptyState message={`${jobId} 不在快照中`} hint="可能尚未加载或 job 目录无 tasks.json" />
        <Button size="sm" onClick={onBack}>
          ← 返回列表
        </Button>
      </div>
    );
  }

  const { done, total, failed } = jobProgress(effectiveJob);
  const lastDiff = useJobsStore.getState().lastDiff;
  const flashTaskId = lastDiff && lastDiff.job_id === effectiveJob.job_id ? lastDiff.task_id : null;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      {/* 头部 */}
      <header className="flex flex-wrap items-center gap-3">
        <Button size="sm" onClick={onBack} aria-label="返回 jobs 列表">
          ←
        </Button>
        <h2 className="font-mono text-base font-semibold text-ink">{effectiveJob.job_id}</h2>
        <span className="text-xs text-ink-3" title={effectiveJob.updated_at}>
          {effectiveJob.updated_at}
        </span>
        <span
          className={`rounded px-2 py-0.5 text-xs ${
            failed > 0
              ? "bg-morty/15 text-morty"
              : done === total && total > 0
                ? "bg-portal/15 text-portal"
                : "bg-white/5 text-ink-2"
          }`}
        >
          {done}/{total} 完成{failed > 0 ? ` · ${failed} 失败` : ""}
        </span>
      </header>

      {/* 任务表（横排简化） */}
      <section aria-label="任务状态">
        <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-ink-3">
          Tasks
        </h3>
        <ul className="flex flex-col gap-1.5">
          {effectiveJob.tasks.map((t) => (
            <TaskCard key={t.task_id} task={t} flash={flashTaskId === t.task_id} compact />
          ))}
        </ul>
      </section>

      {/* 文件浏览器 */}
      <section className="flex min-h-0 flex-1 flex-col" aria-label="job 文件">
        <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-ink-3">
          Files
        </h3>
        <JobFiles workspaceId={workspaceId} jobId={effectiveJob.job_id} />
      </section>
    </div>
  );
}
