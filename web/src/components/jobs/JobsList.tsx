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
import { jobStageInfo } from "../../lib/jobStage";
import { useSessionsStore } from "../../stores/sessions";
import { ApiError } from "../../types";
import { useNavigate } from "react-router-dom";
import type { SessionInfo } from "../../types";

const EMPTY_SESSIONS: SessionInfo[] = [];
import Spinner from "../common/Spinner";
import ErrorBanner from "../common/ErrorBanner";
import EmptyState from "../common/EmptyState";

/**
 * RenameJobDialog：给 job 起任务名（展示层别名，不改 job 目录）。
 * 从 Jobs 卡片标题旁的 ✏ 打开；保存后由父级 reload 快照。
 */
function RenameJobDialog({
  workspaceId,
  job,
  open,
  onClose,
  onSaved,
}: {
  workspaceId: string;
  job: JobSummary;
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [draft, setDraft] = useState(job.name ?? "");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  if (!open) return null;
  const save = async (next: string): Promise<void> => {
    setBusy(true);
    setErr(null);
    try {
      await api.setJobName(workspaceId, job.job_id, next);
      onSaved();
      onClose();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={onClose}>
      <div
        className="w-full max-w-sm rounded-lg border border-line bg-space p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="mb-2 text-sm font-semibold text-ink">
          重命名 <span className="font-mono text-portal">{job.job_id}</span>
        </p>
        <input
          type="text"
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              void save(draft);
            }
            if (e.key === "Escape") onClose();
          }}
          maxLength={60}
          placeholder="起个有意义的名字（如：BERT 环境搭建）"
          className="w-full rounded border border-line bg-space-2/60 px-2 py-1.5 text-sm text-ink outline-none placeholder:text-ink-3/60 focus:border-portal/60"
        />
        <p className="mt-1.5 text-[10px] leading-relaxed text-ink-3">
          仅 web 展示用（侧栏会话行 + Jobs 页）。不重命名 job 目录、不动 tasks.json，
          CLI 与正在跑的任务都不受影响。
        </p>
        {err && <p className="mt-1.5 text-[11px] text-danger">{err}</p>}
        <div className="mt-3 flex items-center gap-2">
          <button
            type="button"
            onClick={() => void save(draft)}
            disabled={busy || !draft.trim()}
            className="rounded-md border border-portal/50 bg-portal/10 px-3 py-1.5 text-xs text-portal hover:bg-portal/20 disabled:opacity-40"
          >
            {busy ? "保存中…" : "保存"}
          </button>
          {job.name && (
            <button
              type="button"
              onClick={() => void save("")}
              disabled={busy}
              className="rounded-md border border-line px-3 py-1.5 text-xs text-ink-3 hover:border-danger/50 hover:text-danger disabled:opacity-40"
            >
              清除任务名
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            className="ml-auto rounded-md border border-line px-3 py-1.5 text-xs text-ink-3 hover:text-ink"
          >
            取消
          </button>
        </div>
      </div>
    </div>
  );
}

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
  // 正在重命名任务名的 job（null = 未打开弹窗）
  const [renaming, setRenaming] = useState<JobSummary | null>(null);

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

  async function archive(jobId: string, opts?: { force?: boolean }): Promise<void> {
    setError(null);
    setArchiving((s) => new Set(s).add(jobId));
    try {
      await api.archiveJob(workspaceId, jobId, opts);
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

  const navigate = useNavigate();
  const sessions = useSessionsStore((s) => s.byWorkspace.get(workspaceId) ?? EMPTY_SESSIONS);
  const [resuming, setResuming] = useState<Set<string>>(new Set());
  // 归档会话也要能找到（大多数会话完成后被自动归档，默认列表不含它们）
  const [archivedSessions, setArchivedSessions] = useState<SessionInfo[]>([]);
  useEffect(() => {
    let cancelled = false;
    api
      .listArchivedSessions(workspaceId, { limit: 200, offset: 0 })
      .then((page) => { if (!cancelled) setArchivedSessions(page.items ?? []); })
      .catch(() => { /* 归档会话拉取失败不阻塞 */ });
    return () => { cancelled = true; };
  }, [workspaceId]);

  /** 找该 job 关联的最近会话（params.job === jobId，含归档；取最新） */
  function findSession(jobId: string): SessionInfo | null {
    const all = [...sessions, ...archivedSessions];
    const list = all.filter(
      (s) => (s.params as { job?: unknown } | undefined)?.job === jobId,
    );
    if (list.length === 0) return null;
    return list.sort((a, b) => (a.created_at < b.created_at ? 1 : -1))[0];
  }

  /** 恢复/打开该 job 的会话：active 直接跳；closed/error 先 Resume 再跳；
   *  无 web 会话 → 尝试导入 CLI 启动的会话（读 doing/session_id） */
  async function openSession(jobId: string, importCli?: boolean): Promise<void> {
    const sess = findSession(jobId);
    if (!sess) {
      if (importCli) {
        // CLI 启动的 job：导入为 web 会话（后端读 doing/session_id 并立即恢复）
        setResuming((s) => new Set(s).add(jobId));
        setError(null);
        try {
          const imported = await api.importSession(workspaceId, jobId);
          navigate(`/session/${imported.id}`);
        } catch (e) {
          setError(e instanceof Error ? e.message : String(e));
        } finally {
          setResuming((s) => { const n = new Set(s); n.delete(jobId); return n; });
        }
      } else {
        setError(`该 job 没有关联的 web 会话——如它是 CLI 启动的，可点「📥 导入」导入并恢复`);
      }
      return;
    }
    if (sess.status === "active" || sess.status === "running") {
      navigate(`/session/${sess.id}`);
      return;
    }
    setResuming((s) => new Set(s).add(jobId));
    setError(null);
    try {
      if (sess.archived) {
        await api.unarchiveSession(sess.id).catch(() => {});
      }
      await api.resumeSession(sess.id);
      navigate(`/session/${sess.id}`);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        navigate(`/session/${sess.id}`);
      } else {
        setError(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setResuming((s) => { const n = new Set(s); n.delete(jobId); return n; });
    }
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
          <li key={job.job_id} className="group relative flex flex-col gap-1.5 rounded-lg border border-line bg-surface/50 p-3 transition-colors hover:border-portal/50">
            <button
              type="button"
              onClick={() => onOpen(job.job_id)}
              className="flex w-full flex-col gap-2 text-left"
            >
              <div className="flex items-baseline gap-2">
                <span className="font-mono text-xs font-semibold text-ink-3">{job.job_id}</span>
                {/* 任务名：用户起的名字才是「在干什么」的主要信息 */}
                <span
                  className={`min-w-0 flex-1 truncate text-sm font-semibold ${
                    job.name ? "text-ink" : "text-ink-3/70"
                  }`}
                  title={job.name || "未命名——点 ✏ 起个有意义的名字"}
                >
                  {job.name || "未命名"}
                </span>
                <span className="shrink-0 text-[10px] text-ink-3" title={job.updated_at}>
                  {relativeTime(job.updated_at)}
                </span>
              </div>
              <ProgressDots tasks={job.tasks} />
              <p className="text-xs text-ink-3">
                {done}/{total} 完成
                {failed > 0 ? ` · ${failed} 失败 ⚠` : done === total && total > 0 ? " ✅" : ""}
                {/* 下一步模式指引（与新建会话的 job 下拉一致） */}
                {(() => {
                  const info = jobStageInfo(job);
                  const tone =
                    info.tone === "portal"
                      ? "text-portal"
                      : info.tone === "morty"
                        ? "text-morty"
                        : info.tone === "danger"
                          ? "text-danger"
                          : "text-ink-3";
                  return (
                    <span className={`ml-1.5 ${tone}`}>
                      · {info.badge}
                      {info.nextMode ? ` · 下一步 ${info.nextMode}` : ""}
                    </span>
                  );
                })()}
              </p>
            </button>
            {/* ✏ 重命名任务名（与「打开详情」分离的独立按钮——弹窗不能嵌在按钮里） */}
            <button
              type="button"
              onClick={() => setRenaming(job)}
              title={job.name ? `重命名任务名（当前：${job.name}）` : "给它起个有意义的名字"}
              className="absolute right-2 top-2 rounded px-1 text-[11px] leading-none text-ink-3 opacity-0 transition-opacity hover:text-portal group-hover:opacity-100"
            >
              ✏
            </button>
            {/* 恢复/打开会话：有 web 会话直接跳；无会话但 job 有 doing/ → 可导入 CLI 会话 */}
            {(() => {
              const sess = findSession(job.job_id);
              const isBusy = resuming.has(job.job_id);
              if (sess) {
                return (
                  <button
                    type="button"
                    onClick={() => void openSession(job.job_id)}
                    disabled={isBusy}
                    title={
                      sess.status === "active" || sess.status === "running"
                        ? `打开会话 ${sess.title || sess.id.slice(0, 8)}（进行中）`
                        : `恢复会话 ${sess.title || sess.id.slice(0, 8)}（${sess.status} → 重新拉起）`
                    }
                    className="flex w-fit items-center gap-1 rounded-md border border-portal/40 px-2 py-0.5 text-[10px] text-portal transition-colors hover:bg-portal/10 disabled:opacity-40"
                  >
                    {isBusy ? "恢复中…" : sess.status === "active" || sess.status === "running" ? "💬 打开会话" : "▶ 恢复会话"}
                  </button>
                );
              }
              // 无 web 会话：job 有 doing 目录（stage=doing/planned 且有 session_id）→ 可导入
              if (job.stage === "doing" || job.stage === "planned" || job.stage === "started") {
                return (
                  <button
                    type="button"
                    onClick={() => void openSession(job.job_id, true)}
                    disabled={isBusy}
                    title="导入 CLI 启动的会话到 web 并恢复（读 job 目录里的 pi 会话标识）"
                    className="flex w-fit items-center gap-1 rounded-md border border-rick/50 px-2 py-0.5 text-[10px] text-rick transition-colors hover:bg-rick/10 disabled:opacity-40"
                  >
                    {isBusy ? "导入中…" : "📥 导入会话"}
                  </button>
                );
              }
              return null;
            })()}
            {complete ? (
              <button
                type="button"
                onClick={() => void archive(job.job_id)}
                disabled={busy}
                title="归档：从列表隐藏（可随时恢复）"
                className="flex w-fit items-center gap-1 rounded-md border border-line px-2 py-0.5 text-[10px] text-ink-3 transition-colors hover:border-portal/50 hover:text-portal disabled:opacity-40"
              >
                {busy ? "归档中…" : "📦 归档"}
              </button>
            ) : (
              // 未完成（含历史 blocked/error 卡住）的 job 也要能清出列表：关闭=归档
              // 到归档区（不伪造 task 状态、文件不动、可恢复）。
              <button
                type="button"
                onClick={() => void archive(job.job_id, { force: true })}
                disabled={busy}
                title="关闭：从列表隐藏（未完成也可关闭；文件保留，可在归档区「恢复」）"
                className="flex w-fit items-center gap-1 rounded-md border border-morty/50 px-2 py-0.5 text-[10px] text-morty transition-colors hover:bg-morty/10 disabled:opacity-40"
              >
                {busy ? "关闭中…" : "🚪 关闭"}
              </button>
            )}
          </li>
        );
      })}
      {renaming && (
        <RenameJobDialog
          workspaceId={workspaceId}
          job={renaming}
          open
          onClose={() => setRenaming(null)}
          onSaved={() => void load(workspaceId)}
        />
      )}
    </ul>
  );
}
