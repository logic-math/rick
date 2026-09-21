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
  // 挂起（平台升级/服务重启）：与「已中断（error=执行失败）」「已完成（closed）」
  // 三态明确区分——人类要求重启后**不自动恢复**，靠这个徽标 + 一键恢复入口。
  if (status === "suspended") {
    return (
      <span
        className="flex w-fit items-center gap-1 rounded-full border border-ink-3/40 bg-ink-3/10 px-1.5 py-0.5 text-[9px] leading-none text-ink-2"
        title="因平台升级/服务重启挂起：进程不在但状态完整，点「恢复继续」重新拉起（不会自动续跑）"
      >
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-ink-3" />
        ⏸ 已挂起（平台升级）
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
  /** 「恢复继续」结果回显（归一化了几条 task / 已存活） */
  const [continueMsg, setContinueMsg] = useState<string | null>(null);
  // 任务名（job 展示别名）编辑态：初值 = 服务端当前值
  const [nameDraft, setNameDraft] = useState<string>(session.job_name ?? "");
  const [nameBusy, setNameBusy] = useState(false);
  const [nameMsg, setNameMsg] = useState<string | null>(null);

  const archived = session.archived === true;
  const live = session.status === "active" || session.status === "running";
  /** 因平台升级/服务重启挂起（可一键恢复；与 error=执行失败 区分） */
  const suspended = session.status === "suspended";
  /**
   * 后台型（doing / background dream）：恢复语义是「归一化 running→pending 后重跑剩余
   * task」而不是 spawn 交互 worker——文案需要区分（与后端 continueSession 分派一致）。
   */
  const isBackground =
    session.type === "doing" ||
    (session.type === "dream" &&
      (session.params as { mode?: unknown } | undefined)?.mode === "background");
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

  // 保存任务名（空串 = 清除别名 → 回到显示 job_N）。成功后 onDone() 刷新列表，
  // 侧栏会话行随即显示新名字（服务端返回 job_name，前端优先展示它）。
  async function saveName(next: string): Promise<void> {
    if (!job) return;
    setNameBusy(true);
    setNameMsg(null);
    setError(null);
    try {
      const res = await api.setJobName(session.workspace_id, job, next);
      setNameDraft(res.name);
      setNameMsg(res.name ? "已保存 ✓" : "已清除（回到显示 job 编号）✓");
      onDone();
    } catch (e) {
      setNameMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setNameBusy(false);
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

  /**
   * 人工恢复因平台升级挂起的会话/任务：/continue（交互型 = resume worker；
   * doing/后台 dream = 归一化 running→pending 后重跑剩余 task）。
   * **只由点击触发**，绝不自动重试。成功后留在弹窗并刷新列表（用户可能继续看信息）。
   */
  async function continueRun(): Promise<void> {
    setBusy(true);
    setError(null);
    setContinueMsg(null);
    try {
      const res = await api.continueSession(session.id);
      const normalized = res.normalized_tasks ?? [];
      setContinueMsg(
        res.already_active
          ? "该会话本就活着（已重新同步状态）✓"
          : normalized.length > 0
            ? `已继续：归一化 ${normalized.length} 个遗留 running task（→ pending）并重跑剩余 task ✓`
            : "已继续运行 ✓",
      );
      onDone();
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
          {job && (
            <div className="flex gap-2">
              <dt className="w-16 shrink-0 pt-1 text-ink-3">任务名</dt>
              <dd className="flex min-w-0 flex-1 flex-col gap-1.5">
                <div className="flex min-w-0 items-center gap-1.5">
                  <input
                    type="text"
                    value={nameDraft}
                    onChange={(e) => setNameDraft(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        void saveName(nameDraft);
                      }
                    }}
                    maxLength={60}
                    placeholder="给它起个有意义的名字（如：BERT 环境搭建）"
                    aria-label="job 任务名"
                    className="min-w-0 flex-1 rounded border border-line bg-space-2/60 px-2 py-1 text-xs text-ink outline-none placeholder:text-ink-3/60 focus:border-portal/60"
                  />
                  <button
                    type="button"
                    onClick={() => void saveName(nameDraft)}
                    disabled={nameBusy || !nameDraft.trim()}
                    className="shrink-0 rounded border border-portal/50 bg-portal/10 px-2 py-1 text-[10px] text-portal transition-colors hover:bg-portal/20 disabled:opacity-40"
                  >
                    {nameBusy ? "保存中…" : "保存"}
                  </button>
                  {(session.job_name || nameDraft) && (
                    <button
                      type="button"
                      onClick={() => {
                        setNameDraft("");
                        void saveName("");
                      }}
                      disabled={nameBusy}
                      title="清除任务名，回到显示 job 编号"
                      className="shrink-0 rounded border border-line px-2 py-1 text-[10px] text-ink-3 transition-colors hover:border-danger/50 hover:text-danger disabled:opacity-40"
                    >
                      清除
                    </button>
                  )}
                </div>
                {nameMsg && (
                  <span className="text-[10px] text-ink-2">{nameMsg}</span>
                )}
                <span className="text-[10px] text-ink-3">
                  仅 web 展示用的别名（不重命名 job 目录、不动 tasks.json）——侧栏会话行与
                  Jobs 页会显示它。
                </span>
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
          {suspended && (
            <div className="mb-3 flex flex-col gap-2">
              <p className="text-xs leading-relaxed text-ink-2">
                ⏸ 该会话因<b>平台升级/服务重启</b>挂起：agent 进程不在，但对话与任务状态完整。
                点下方「恢复继续」重新拉起（{isBackground
                  ? "doing/dream 会先把遗留的 running task 归一化为 pending，再续跑剩余 task"
                  : "用 pi 会话恢复，继续对话"}）。
                <span className="text-ink-3">不会自动恢复——避免重复副作用。</span>
              </p>
              <button
                type="button"
                onClick={() => void continueRun()}
                disabled={busy}
                className="w-full rounded-md border border-portal/50 bg-portal/10 px-3 py-2 text-xs text-portal transition-colors hover:bg-portal/20 disabled:opacity-40"
              >
                {busy ? "恢复中…" : "▶ 恢复继续"}
              </button>
              {continueMsg && <p className="text-[11px] text-portal">{continueMsg}</p>}
            </div>
          )}
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
