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
import { useNavigate } from "react-router-dom";
import { api } from "../api/client";
import JobDetail from "../components/jobs/JobDetail";
import JobsList from "../components/jobs/JobsList";
import { ApiError, type JobSummary, type SessionInfo } from "../types";

/** 归档 job 折叠区：默认折叠；恢复后回主列表 */
function ArchivedSection({
  workspaceId,
  onRestored,
  onOpen,
}: {
  workspaceId: string;
  onRestored: () => void;
  onOpen: (jobId: string) => void;
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
            const src = j.archived_by ?? "manual";
            // manual 归档且未全部完成 → 是用户「关闭」掉的（含历史 blocked/error 卡住
            // 的 job）——与「手动归档的已完成 job」区分标注，且都可「恢复」。
            const incomplete = j.tasks.length > 0 && !j.tasks.every((t) => t.status === "success");
            const srcBadge =
              src === "dream" ? (
                <span className="rounded bg-portal/10 px-1.5 py-0.5 text-[10px] text-portal" title="该 job 已被 dream 学习沉淀，自动归档">
                  🏭 dream 已学习
                </span>
              ) : src === "done" ? (
                <span className="rounded bg-ink-3/10 px-1.5 py-0.5 text-[10px] text-ink-3" title="任务全部完成即自动归档（完成即离开进行中列表）">
                  ✅ 已完成自动归档
                </span>
              ) : incomplete ? (
                <span
                  className="rounded bg-morty/15 px-1.5 py-0.5 text-[10px] text-morty"
                  title="已关闭：该 job 未全部完成（如 blocked/error/skipped 的 task）——关闭只是从列表隐藏，文件保留"
                >
                  🚪 已关闭（未完成）
                </span>
              ) : (
                <span className="rounded bg-morty/10 px-1.5 py-0.5 text-[10px] text-morty" title="手动归档">
                  📦 手动归档
                </span>
              );
            return (
              <li
                key={j.job_id}
                className="group flex items-center gap-3 rounded-md border border-dashed border-line px-3 py-1.5 text-xs"
              >
                <button
                  type="button"
                  onClick={() => onOpen(j.job_id)}
                  className="font-mono text-ink-2 underline-offset-2 group-hover:text-portal group-hover:underline"
                  title={`打开 ${j.job_id}`}
                >
                  {j.job_id}
                </button>
                <span className="text-ink-3">
                  {j.tasks.length > 0 ? `${j.tasks.filter((t) => t.status === "success").length}/${j.tasks.length} 完成` : ""}
                </span>
                {srcBadge}
                {/* 打开/恢复该 job 的会话；无 web 会话但有 CLI 会话目录 → 📥 导入 */}
                <OpenSessionButton
                  workspaceId={workspaceId}
                  jobId={j.job_id}
                  canImport={j.stage === "doing" || j.stage === "planned" || j.stage === "started" || j.tasks.length > 0}
                />
                {src === "manual" && (
                  <button
                    type="button"
                    onClick={() => void restore(j.job_id)}
                    disabled={busy}
                    className="ml-auto rounded border border-line px-2 py-0.5 text-[10px] text-portal hover:border-portal/50 hover:bg-portal/10 disabled:opacity-40"
                    title="解除手动归档（已完成 job 会由系统自动归档，仍在下方列表）"
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

/**
 * OpenSessionButton：在 job 行上显示「打开/恢复会话」按钮。
 * 查找该 job 关联的最近会话（含归档），active 直接跳转，closed/error 先 Resume。
 * 无关联会话则不渲染（避免无意义的按钮）。
 */
/**
 * OpenSessionButton：打开/恢复该 job 的 web 会话；**没有 web 会话但 job 目录里
 * 有 CLI 会话（doing/session_id）时，直接提供「导入」**——归档区的 job 大多是
 * 已完成自动归档的，它们的 CLI 会话此前在 web 里完全没有入口（用户实测：
 * job_69 归档后既没有 ▶ 也没有 📥，无法在 web 里继续对话）。
 */
function OpenSessionButton({
  workspaceId,
  jobId,
  canImport,
}: {
  workspaceId: string;
  jobId: string;
  /** job 目录含 CLI 会话（stage=doing/planned/started）→ 可导入 */
  canImport?: boolean;
}) {
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [sess, setSess] = useState<SessionInfo | null>(null);
  const [probed, setProbed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const find = async () => {
      try {
        // 默认列表
        const live = await api.listSessions(workspaceId);
        let hit = live.find((s) => (s.params as { job?: unknown } | undefined)?.job === jobId);
        if (!hit) {
          // 归档会话
          const page = await api.listArchivedSessions(workspaceId, { limit: 200, offset: 0 });
          hit = (page.items ?? []).find((s) => (s.params as { job?: unknown } | undefined)?.job === jobId);
        }
        if (!cancelled) setSess(hit ?? null);
      } catch { /* 静默 */ } finally {
        if (!cancelled) setProbed(true);
      }
    };
    void find();
    return () => { cancelled = true; };
  }, [workspaceId, jobId]);

  if (!probed) return null;

  const isActive = sess?.status === "active" || sess?.status === "running";

  const go = async () => {
    if (!sess) {
      // 无 web 会话 → 导入 CLI 会话（后端读 doing/session_id 并立即恢复）
      setBusy(true);
      try {
        const imported = await api.importSession(workspaceId, jobId);
        navigate(`/session/${imported.id}`);
      } catch { /* 失败保持原状（下次点击重试） */ } finally { setBusy(false); }
      return;
    }
    if (isActive) { navigate(`/session/${sess.id}`); return; }
    setBusy(true);
    try {
      if (sess.archived) await api.unarchiveSession(sess.id).catch(() => {});
      await api.resumeSession(sess.id);
      navigate(`/session/${sess.id}`);
    } catch (e: unknown) {
      if (e instanceof ApiError && e.status === 409) navigate(`/session/${sess.id}`);
    } finally { setBusy(false); }
  };

  if (!sess && !canImport) return null;

  const label = busy ? "…" : !sess ? "📥" : isActive ? "💬" : "▶";
  const title = !sess
    ? `导入 CLI 启动的会话到 web 并恢复（读 ${jobId} 目录里的 pi 会话标识）`
    : isActive
      ? "打开会话（进行中）"
      : `恢复会话（${sess.status} → 重新拉起）`;
  return (
    <button
      type="button"
      onClick={() => void go()}
      disabled={busy}
      title={title}
      className={`rounded border px-1.5 py-0.5 text-[10px] hover:bg-white/5 disabled:opacity-40 ${
        !sess ? "border-rick/50 text-rick" : "border-portal/40 text-portal hover:bg-portal/10"
      }`}
    >
      {label}
    </button>
  );
}

export default function Jobs({ workspaceId }: { workspaceId: string }) {
  const [openJob, setOpenJob] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  return (
    // 铺满主内容区（同 Dreams 页修法）：文件树 + 内容面板需要全宽/全高
    <div className="flex h-full min-h-0 w-full flex-col gap-4">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Jobs</h1>
      </header>

      {openJob ? (
        <JobDetail workspaceId={workspaceId} jobId={openJob} onBack={() => setOpenJob(null)} />
      ) : (
        <>
          {/* 归档后主列表重挂（refresh key） */}
          <JobsList key={tick} workspaceId={workspaceId} onOpen={(jobId) => setOpenJob(jobId)} />
          <ArchivedSection
            workspaceId={workspaceId}
            onRestored={() => setTick((t) => t + 1)}
            onOpen={(jobId) => setOpenJob(jobId)}
          />
        </>
      )}
    </div>
  );
}
