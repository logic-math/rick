/**
 * Jobs 看板视图（/ws/:wsId/jobs）。
 *
 * 单工作区 jobs 列表 / job 详情（文件浏览/任务状态）。工作区上下文由
 * 路由 /ws/:wsId 提供（去掉原页面内工作区选择器）。
 *
 * 详情态用本地 state（onOpen/onBack）而非嵌套路由——JobsList/JobDetail 的
 * props 契约（onOpen/onBack）由 L3 组件定死，包装层薄封装即可。
 */

import { useEffect, useState } from "react";
import { api } from "../api/client";
import JobDetail from "../components/jobs/JobDetail";
import JobsList from "../components/jobs/JobsList";
import type { JobSummary } from "../types";

/** 归档 job 折叠区：默认折叠；恢复后回主列表 */
function ArchivedSection({
  workspaceId,
  onRestored,
}: {
  workspaceId: string;
  onRestored: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [archived, setArchived] = useState<JobSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!open) return;
    setError(null);
    api
      .listJobs(workspaceId, true)
      .then((list) => {
        if (!cancelled) setArchived(list.filter((j) => j.archived));
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [open, workspaceId]);

  async function restore(jobId: string): Promise<void> {
    setError(null);
    setBusy(true);
    try {
      await api.unarchiveJob(workspaceId, jobId);
      setArchived((prev) => (prev ?? []).filter((j) => j.job_id !== jobId));
      onRestored(); // 主列表刷新（该 job 回到可见区）
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const count = archived?.length ?? 0;
  return (
    <section className="flex flex-col gap-2">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-fit items-center gap-2 rounded-md border border-line px-3 py-1.5 text-xs text-ink-3 transition-colors hover:border-portal/40 hover:text-portal"
      >
        <span aria-hidden="true">{open ? "▾" : "▸"}</span>
        已归档{open && count > 0 ? ` (${count})` : ""}
      </button>
      {open && error && <p className="text-xs text-danger">{error}</p>}
      {open && archived && archived.length === 0 && !error && (
        <p className="text-xs text-ink-3">暂无已归档 job</p>
      )}
      {open && archived && archived.length > 0 && (
        <ul className="flex flex-col gap-1">
          {archived.map((j) => {
            const byDream = j.archived_by === "dream";
            return (
              <li
                key={j.job_id}
                className="flex items-center gap-3 rounded-md border border-dashed border-line px-3 py-1.5 text-xs"
              >
                <span className="font-mono text-ink-2">{j.job_id}</span>
                <span className="text-ink-3">
                  {j.tasks.length > 0 ? `${j.tasks.filter((t) => t.status === "success").length}/${j.tasks.length} 完成` : ""}
                </span>
                {byDream ? (
                  <span className="rounded bg-portal/10 px-1.5 py-0.5 text-[10px] text-portal" title="该 job 已被 dream 学习沉淀，自动归档">
                    🏭 dream 已学习
                  </span>
                ) : (
                  <button
                    type="button"
                    onClick={() => void restore(j.job_id)}
                    disabled={busy}
                    className="ml-auto rounded border border-line px-2 py-0.5 text-[10px] text-portal hover:border-portal/50 hover:bg-portal/10 disabled:opacity-40"
                  >
                    恢复
                  </button>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

export default function Jobs({ workspaceId }: { workspaceId: string }) {
  const [openJob, setOpenJob] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-5">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Jobs</h1>
      </header>

      {openJob ? (
        <JobDetail workspaceId={workspaceId} jobId={openJob} onBack={() => setOpenJob(null)} />
      ) : (
        <>
          {/* 归档后主列表重挂（refresh key） */}
          <JobsList key={tick} workspaceId={workspaceId} onOpen={(jobId) => setOpenJob(jobId)} />
          <ArchivedSection workspaceId={workspaceId} onRestored={() => setTick((t) => t + 1)} />
        </>
      )}
    </div>
  );
}
