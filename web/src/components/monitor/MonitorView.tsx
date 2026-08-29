/**
 * 监控视图（doing / dream 后台型会话的主视图）。
 *
 * - 头部：类型徽标 + title + Saucer 飞碟运行指示（running 悬停+光束）+ 状态点
 *   + Abort 按钮（running/active 时可中止）
 * - 主体双栏（移动端上下堆叠）：左 TaskBoard（+GateResult 横幅），右 EventStream
 * - 数据源：sessions store（会话元信息）+ session events store（事件缓冲）+
 *   jobs store（任务快照——挂载时确保该工作区已加载）
 * - jobId 推导：doing → params.job；dream（后台）→ 无单一 job（TaskBoard 显示
 *   最近更新 job 的快照或占位说明）
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import { useJobsStore } from "../../stores/jobs";
import { useSessionEventsStore } from "../../stores/events";
import { useSessionsStore } from "../../stores/sessions";
import type { JobSummary, SessionInfo } from "../../types";
import Button from "../common/Button";
import Saucer from "../starfield/Saucer";
import StatusDot from "../common/StatusDot";
import Spinner from "../common/Spinner";
import EventStream from "./EventStream";
import GateResult from "./GateResult";
import TaskBoard from "./TaskBoard";

interface MonitorViewProps {
  sessionId: string;
}

const TYPE_LABEL: Record<string, string> = {
  doing: "DOING",
  dream: "DREAM",
};

/** 从 params 提取 job（doing 型） */
function jobIdOf(session: SessionInfo | null): string | null {
  if (!session) return null;
  if (session.type === "doing") {
    const job = (session.params as Record<string, unknown>)?.job;
    return typeof job === "string" ? job : null;
  }
  return null;
}

export default function MonitorView({ sessionId }: MonitorViewProps) {
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [aborting, setAborting] = useState(false);
  const [abortError, setAbortError] = useState<string | null>(null);

  // 会话事件缓冲（版本号驱动重渲染）
  const eventsVersion = useSessionEventsStore((s) => s.version);
  const sessionEvents = useSessionEventsStore((s) => s.buffers.get(sessionId));
  const events = useMemo(
    () => sessionEvents ?? [],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [sessionEvents, eventsVersion],
  );

  // jobs store（快照 + diff）
  const jobsVersion = useJobsStore((s) => s.version);
  const lastDiff = useJobsStore((s) => s.lastDiff);
  const getJobs = useJobsStore((s) => s.getJobs);
  const loadJobs = useJobsStore((s) => s.load);

  // 会话元信息（sessions store 缓存优先）
  useEffect(() => {
    let cancelled = false;
    void useSessionsStore
      .getState()
      .get(sessionId)
      .then((info) => {
        if (!cancelled) {
          if (info) setSession(info);
          else setSessionError("会话不存在或已被清除");
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) setSessionError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [sessionId]);

  // 会话状态实时同步（SSE session_state → sessions store.applyState）
  const applyState = useSessionsStore((s) => s.applyState);
  useEffect(() => {
    const state = useSessionEventsStore.getState().states.get(sessionId);
    if (state?.status && session && session.status !== state.status) {
      setSession({ ...session, status: state.status as SessionInfo["status"] });
      applyState(sessionId, state.status as SessionInfo["status"]);
    }
  }, [eventsVersion, sessionId, session, applyState]);

  // 确保该工作区 jobs 已加载（REST 快照兜底）
  useEffect(() => {
    if (session?.workspace_id) void loadJobs(session.workspace_id);
  }, [session?.workspace_id, loadJobs]);

  const workspaceId = session?.workspace_id ?? null;
  const jobs = useMemo(
    () => (workspaceId ? getJobs(workspaceId) : []),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [workspaceId, jobsVersion, getJobs],
  );

  const paramJobId = jobIdOf(session);
  const job: JobSummary | null = useMemo(() => {
    if (!paramJobId) {
      // dream：无单一 job——展示最近更新的 job 快照
      if (jobs.length === 0) return null;
      return jobs[0]; // store 按 updated_at 降序（后端契约）
    }
    return jobs.find((j) => j.job_id === paramJobId) ?? null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paramJobId, jobs]);

  // diff 闪烁：本会话关联 job 的最近一次 task 变更
  const flashTaskId =
    lastDiff && job && lastDiff.job_id === job.job_id ? lastDiff.task_id : null;

  const isRunning = session?.status === "running" || session?.status === "active";

  const onAbort = useCallback(async () => {
    setAborting(true);
    setAbortError(null);
    try {
      await api.abort(sessionId);
    } catch (err: unknown) {
      setAbortError(err instanceof Error ? err.message : String(err));
    } finally {
      setAborting(false);
    }
  }, [sessionId]);

  if (sessionError && !session) {
    return (
      <div className="mx-auto max-w-xl py-8 text-center text-sm text-danger">{sessionError}</div>
    );
  }
  if (!session) {
    return <Spinner center label="加载会话…" />;
  }

  const typeLabel = TYPE_LABEL[session.type] ?? session.type.toUpperCase();

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      {/* 头部 */}
      <header className="flex flex-wrap items-center gap-3 rounded-lg border border-line bg-surface/50 px-4 py-3">
        <Saucer size={40} flying={isRunning} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="rounded bg-nebula/25 px-1.5 py-0.5 font-mono text-[10px] font-bold tracking-wider text-ink">
              {typeLabel}
            </span>
            <StatusDot
              status={
                session.status === "running" || session.status === "active"
                  ? "running"
                  : session.status === "error"
                    ? "error"
                    : session.status === "closed"
                      ? "closed"
                      : "pending"
              }
              label={session.status}
            />
            <span className="font-mono text-[10px] text-ink-3">{session.status}</span>
          </div>
          <h1 className="mt-0.5 truncate text-sm font-semibold text-ink" title={session.title ?? session.id}>
            {session.title ?? `${typeLabel} 会话`}
          </h1>
        </div>
        {isRunning && (
          <Button variant="danger" size="sm" loading={aborting} onClick={() => void onAbort()}>
            中止
          </Button>
        )}
      </header>

      {abortError && (
        <p className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-xs text-danger">
          中止失败：{abortError}
        </p>
      )}

      {/* 主体双栏（移动端上下堆叠） */}
      <div className="flex min-h-0 flex-1 gap-3 max-md:flex-col">
        <section className="flex w-1/2 min-w-0 flex-col gap-3 overflow-y-auto max-md:w-full">
          <GateResult events={events} />
          <TaskBoard
            job={job}
            flashTaskId={flashTaskId}
            placeholder={
              session.type === "dream"
                ? "dream 批量处理多个 job——任务快照随 jobs_update 流入"
                : undefined
            }
          />
        </section>
        <section className="flex min-h-64 w-1/2 min-w-0 flex-col max-md:w-full">
          <EventStream events={events} />
        </section>
      </div>
    </div>
  );
}
