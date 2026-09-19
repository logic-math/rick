/**
 * SessionSettingsDialog：会话 ⋮ 设置弹层（信息 + 人工归档/恢复）。
 *
 * 语义（用户确认：不做 agent 自动判定——人工标记完成归档）：
 * - 未归档会话 → 「归档」：active/running 先终止 worker（后端 close）再标记；
 *   closed/error 直接标记完成并从默认列表隐藏。
 * - 已归档会话 → 「恢复」：回默认列表（status 不变；closed 点进走 Resume）。
 * 操作成功后回调 onChanged（父级负责刷新列表）。
 */

import { useState } from "react";
import { useWorkspacesStore } from "../../stores/workspaces";
import { api } from "../../api/client";
import type { SessionInfo, SessionType } from "../../types";
import Dialog from "../common/Dialog";

const TYPE_LABEL: Record<SessionType, string> = {
  plan: "PLAN",
  easy: "EASY",
  ctrl: "CTRL",
  "human-loop": "HUMAN-LOOP",
  learning: "LEARNING",
  dream: "DREAM",
  doing: "DOING",
};

const TYPE_TONE: Record<SessionType, string> = {
  plan: "border-rick/50 bg-rick/10 text-rick",
  easy: "border-portal/50 bg-portal-soft text-portal",
  ctrl: "border-morty/50 bg-morty/10 text-morty",
  "human-loop": "border-nebula/50 bg-nebula/10 text-nebula",
  learning: "border-portal/40 bg-portal-soft text-portal",
  dream: "border-nebula/50 bg-nebula/10 text-nebula",
  doing: "border-rick/40 bg-rick/10 text-rick",
};

export function sessionStatusBadge(status: SessionInfo["status"]): React.ReactNode {
  if (status === "active" || status === "running") {
    return (
      <span className="flex w-fit items-center gap-1 rounded-full border border-portal/40 bg-portal/10 px-1.5 py-0.5 text-[9px] leading-none text-portal">
        <span className="inline-block h-1.5 w-1.5 animate-rm-pulse rounded-full bg-portal" />
        {status === "running" ? "进行中" : "活跃"}
      </span>
    );
  }
  if (status === "error") {
    return (
      <span className="flex w-fit items-center gap-1 rounded-full border border-morty/40 bg-morty/10 px-1.5 py-0.5 text-[9px] leading-none text-morty">
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-morty" />
        已中断
      </span>
    );
  }
  return (
    <span className="flex w-fit items-center gap-1 rounded-full border border-ink-3/30 bg-ink-3/10 px-1.5 py-0.5 text-[9px] leading-none text-ink-3">
      <span className="inline-block h-1.5 w-1.5 rounded-full bg-ink-3" />
      已完成
    </span>
  );
}

function summaryOf(s: SessionInfo): string {
  try {
    const j = JSON.stringify(s.params ?? {});
    return j.length > 160 ? j.slice(0, 160) + "…" : j;
  } catch {
    return String(s.params);
  }
}

interface Props {
  session: SessionInfo | null;
  open: boolean;
  onClose: () => void;
  /** 归档/恢复成功后通知父级刷新 */
  onChanged: () => void;
}

export default function SessionSettingsDialog({ session, open, onClose, onChanged }: Props) {
  // hooks 恒定调用（session 可能在 null↔有值间切换）；内容交给 Body（session 必非空）
  return (
    <Dialog
      open={open && session !== null}
      title={
        session
          ? `会话设置 · ${session.title || `${session.type} · ${session.id.slice(0, 8)}`}`
          : "会话设置"
      }
      onClose={onClose}
      contentClassName="w-full max-w-md"
    >
      {session && (
        <SettingsBody session={session} onDone={onChanged} onClose={onClose} />
      )}
    </Dialog>
  );
}

function SettingsBody({
  session,
  onDone,
  onClose,
}: {
  session: SessionInfo;
  onDone: () => void;
  onClose: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const archived = session.archived === true;
  const live = session.status === "active" || session.status === "running";
  // Job 归属（doing/ctrl/learning 等）：会话被命名后侧栏不再显示编号，用户需要
  // 一个能查到「这是哪个 job / 它的工作目录在哪」的地方（实测反馈）。
  const jobParamRaw = (session.params as { job?: unknown } | undefined)?.job;
  const job = typeof jobParamRaw === "string" && jobParamRaw ? jobParamRaw : null;
  const ws = useWorkspacesStore((st) => st.list.find((w) => w.id === session.workspace_id));
  const jobDir = job && ws ? `${ws.path}/.rick/jobs/${job}` : null;
  const copy = (text: string) => {
    void navigator.clipboard?.writeText(text).catch(() => {});
  };

  async function archive(): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      await api.archiveSession(session.id);
      onDone();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function restore(): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      await api.unarchiveSession(session.id);
      onDone();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
        {/* 信息 */}
        <dl className="flex flex-col gap-1.5 text-xs">
          <div className="flex items-center gap-2">
            <dt className="w-16 shrink-0 text-ink-3">类型</dt>
            <dd className="flex items-center gap-2">
              <span
                className={`rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold tracking-wider ${TYPE_TONE[session.type]}`}
              >
                {TYPE_LABEL[session.type]}
              </span>
              {sessionStatusBadge(session.status)}
              {archived && (
                <span className="rounded bg-ink-3/10 px-1.5 py-0.5 text-[9px] text-ink-3">
                  已归档
                </span>
              )}
            </dd>
          </div>
          <div className="flex gap-2">
            <dt className="w-16 shrink-0 text-ink-3">ID</dt>
            <dd className="min-w-0 break-all font-mono text-[10px] text-ink-3">{session.id}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="w-16 shrink-0 text-ink-3">创建</dt>
            <dd className="text-ink-2">{new Date(session.created_at).toLocaleString()}</dd>
          </div>
          {session.closed_at && (
            <div className="flex gap-2">
              <dt className="w-16 shrink-0 text-ink-3">结束</dt>
              <dd className="text-ink-2">{new Date(session.closed_at).toLocaleString()}</dd>
            </div>
          )}
          {session.archived_at && (
            <div className="flex gap-2">
              <dt className="w-16 shrink-0 text-ink-3">归档</dt>
              <dd className="text-ink-2">{new Date(session.archived_at).toLocaleString()}</dd>
            </div>
          )}
          <div className="flex gap-2">
            <dt className="w-16 shrink-0 text-ink-3">pi id</dt>
            <dd className="min-w-0 break-all font-mono text-[10px] text-ink-3">
              {session.pi_session_id}
            </dd>
          </div>
          {job && (
            <div className="flex gap-2">
              <dt className="w-16 shrink-0 text-ink-3">Job</dt>
              <dd className="flex min-w-0 items-center gap-2">
                <span className="rounded border border-portal/40 bg-portal/10 px-1.5 py-0.5 font-mono text-[10px] text-portal">
                  {job}
                </span>
                <button
                  type="button"
                  onClick={() => copy(job)}
                  className="rounded border border-line px-1.5 py-0.5 text-[10px] text-ink-3 hover:border-portal/50 hover:text-portal"
                  title="复制 job 编号（新建 doing/ctrl/learning 会话或 rick --resume 时可用）"
                >
                  复制
                </button>
              </dd>
            </div>
          )}
          {jobDir && (
            <div className="flex gap-2">
              <dt className="w-16 shrink-0 text-ink-3">Job 目录</dt>
              <dd className="min-w-0 break-all font-mono text-[10px] text-ink-3">
                {jobDir}
                <button
                  type="button"
                  onClick={() => copy(jobDir)}
                  className="ml-1.5 rounded border border-line px-1.5 py-0.5 text-[10px] text-ink-3 hover:border-portal/50 hover:text-portal"
                  title="复制 job 工作目录路径"
                >
                  复制
                </button>
              </dd>
            </div>
          )}
          <div className="flex gap-2">
            <dt className="w-16 shrink-0 text-ink-3">参数</dt>
            <dd className="min-w-0 break-all font-mono text-[10px] text-ink-3">{summaryOf(session)}</dd>
          </div>
        </dl>

        {/* 动作 */}
        <div className="border-t border-line pt-3">
          {archived ? (
            <>
              <p className="mb-2 text-xs leading-relaxed text-ink-3">
                该会话已归档（默认列表隐藏）。恢复后回到列表——若状态为「已完成」，点进会话会用
                Resume 重新拉起执行。
              </p>
              <button
                type="button"
                onClick={() => void restore()}
                disabled={busy}
                className="w-full rounded-md border border-portal/50 bg-portal/10 px-3 py-2 text-xs text-portal transition-colors hover:bg-portal/20 disabled:opacity-40"
              >
                {busy ? "恢复中…" : "↩ 恢复（回到列表）"}
              </button>
            </>
          ) : (
            <>
              <p className="mb-2 text-xs leading-relaxed text-ink-3">
                {live
                  ? "归档会先结束当前任务（终止 agent 进程，会话标记为已完成），再将该会话从默认列表隐藏。"
                  : "归档将该会话标记为完成并从默认列表隐藏（对话历史保留，可在归档列表查看/恢复）。"}
              </p>
              <button
                type="button"
                onClick={() => void archive()}
                disabled={busy}
                className="w-full rounded-md border border-danger/50 bg-danger/10 px-3 py-2 text-xs text-danger transition-colors hover:bg-danger/20 disabled:opacity-40"
              >
                {busy
                  ? "处理中…"
                  : live
                    ? "结束任务并归档"
                    : "归档（标记完成，从列表隐藏）"}
              </button>
            </>
          )}
        </div>
        {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}
