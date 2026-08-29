/**
 * Sessions 总览页（默认路由 /）。
 *
 * Sidebar 主内容化：每工作区一区块（会话卡片列表 + 新建按钮 + 移除工作区）；
 * closed 会话卡显示 Resume；无工作区时 EmptyState 引导添加。
 * 会话卡点击 → /session/:id。
 */

import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api/client";
import { useSessionsStore } from "../stores/sessions";
import { useWorkspacesStore } from "../stores/workspaces";
import type { SessionInfo } from "../types";
import Button from "../components/common/Button";
import EmptyState from "../components/common/EmptyState";
import ErrorBanner from "../components/common/ErrorBanner";
import Spinner from "../components/common/Spinner";
import NewSessionModal from "../components/sessions/NewSessionModal";
import { useIsMobile } from "./hooks";

// ============================================================
// 会话卡片
// ============================================================

const TYPE_TONE: Record<string, string> = {
  plan: "border-rick/50 bg-rick/10 text-rick",
  easy: "border-portal/50 bg-portal-soft text-portal",
  ctrl: "border-morty/50 bg-morty/10 text-morty",
  "human-loop": "border-nebula/60 bg-nebula/15 text-ink",
  learning: "border-rick/50 bg-rick/10 text-rick",
  dream: "border-nebula/60 bg-nebula/15 text-ink",
  doing: "border-morty/50 bg-morty/10 text-morty",
};

function sessionSubtitle(s: SessionInfo): string {
  const p = s.params as Record<string, unknown>;
  switch (s.type) {
    case "plan":
      return typeof p?.requirement === "string" ? p.requirement.slice(0, 90) : "";
    case "easy":
      return typeof p?.requirement === "string" ? p.requirement.slice(0, 90) : "";
    case "human-loop":
      return typeof p?.topic === "string" ? p.topic.slice(0, 90) : "";
    case "doing":
    case "ctrl":
    case "learning":
      return typeof p?.job === "string" ? p.job : "";
    case "dream":
      return `job_num=${String(p?.job_num ?? "?")} mode=${String(p?.mode ?? "?")}`;
    default:
      return "";
  }
}

function SessionCard({ session }: { session: SessionInfo }) {
  const resume = useSessionsStore((s) => s.get);
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const isClosed = session.status === "closed" || session.status === "error";

  async function handleResume(ev: React.MouseEvent): Promise<void> {
    ev.preventDefault();
    ev.stopPropagation();
    setBusy(true);
    setErr(null);
    try {
      await api.resumeSession(session.id);
      navigate(`/session/${session.id}`);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
    void resume; // index 缓存刷新由 SSE session_state 驱动
  }

  return (
    <Link
      to={`/session/${session.id}`}
      className="group flex flex-col gap-2 rounded-xl border border-line bg-surface/60 p-4 transition-colors hover:border-portal/40 hover:bg-surface-raised/60"
    >
      <div className="flex items-center gap-2">
        <span
          className={`rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider ${TYPE_TONE[session.type] ?? "border-line text-ink-2"}`}
        >
          {session.type}
        </span>
        <span className="truncate text-sm font-medium text-ink group-hover:text-portal">
          {session.title || `${session.type} 会话`}
        </span>
        <span className="ml-auto shrink-0 text-[10px] uppercase tracking-wide text-ink-3">
          {session.status}
        </span>
      </div>
      {sessionSubtitle(session) && (
        <p className="line-clamp-2 text-xs leading-relaxed text-ink-2">
          {sessionSubtitle(session)}
        </p>
      )}
      <div className="flex items-center gap-3 text-[10px] text-ink-3">
        <span>{new Date(session.created_at).toLocaleString()}</span>
        {isClosed && (
          <button
            type="button"
            onClick={(e) => void handleResume(e)}
            disabled={busy}
            className="ml-auto rounded border border-portal/50 px-2 py-0.5 text-[10px] font-medium text-portal opacity-90 hover:bg-portal-soft disabled:opacity-50"
          >
            {busy ? "恢复中…" : "▶ Resume"}
          </button>
        )}
      </div>
      {err && <p className="text-[10px] text-danger">{err}</p>}
    </Link>
  );
}

// ============================================================
// 添加工作区表单（内联卡片）
// ============================================================

function AddWorkspaceCard({ onAdded }: { onAdded: (id: string) => void }) {
  const add = useWorkspacesStore((s) => s.add);
  const [path, setPath] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(): Promise<void> {
    if (!path.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const entry = await add(path.trim(), name.trim() || undefined);
      setPath("");
      setName("");
      onAdded(entry.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const FIELD =
    "w-full rounded-lg border border-line bg-space/60 px-3 py-2 text-sm text-ink placeholder:text-ink-3 focus:border-portal/60 focus:outline-none";

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-dashed border-line bg-space/30 p-4">
      <span className="text-sm font-medium text-ink">添加工作区</span>
      <p className="text-xs text-ink-3">
        输入含 <code className="rounded bg-white/5 px-1">.rick</code> 目录的绝对路径（服务端校验）
      </p>
      <div className="flex flex-col gap-2 sm:flex-row">
        <input
          className={FIELD}
          placeholder="/abs/path/to/project"
          value={path}
          onChange={(e) => setPath(e.target.value)}
        />
        <input
          className={`${FIELD} sm:w-40`}
          placeholder="名称（可选）"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Button
          variant="primary"
          onClick={() => void submit()}
          loading={busy}
          disabled={!path.trim()}
          className="sm:w-28"
        >
          注册
        </Button>
      </div>
      <ErrorBanner message={error} onDismiss={() => setError(null)} />
    </div>
  );
}

// ============================================================
// 工作区区块
// ============================================================

function WorkspaceSection({
  workspaceId,
  onNewSession,
  compact,
}: {
  workspaceId: string;
  onNewSession: (wsId: string) => void;
  compact: boolean;
}) {
  const workspace = useWorkspacesStore((s) => s.list.find((w) => w.id === workspaceId));
  const remove = useWorkspacesStore((s) => s.remove);
  const load = useSessionsStore((s) => s.load);
  const loading = useSessionsStore((s) => s.loadingByWorkspace.get(workspaceId) ?? false);
  const sessions = useSessionsStore((s) => s.byWorkspace.get(workspaceId) ?? []);
  const [confirmRemove, setConfirmRemove] = useState(false);

  useEffect(() => {
    void load(workspaceId);
  }, [workspaceId, load]);

  if (!workspace) return null;

  const sorted = sessions.slice().sort((a, b) => (a.created_at < b.created_at ? 1 : -1));

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <h2 className="text-base font-semibold text-ink">
          {workspace.name || workspace.path}
        </h2>
        <span className="truncate text-xs text-ink-3">{workspace.path}</span>
        <span className="ml-auto text-[10px] text-ink-3">
          {typeof workspace.jobs_count === "number" && workspace.jobs_count >= 0
            ? `${workspace.jobs_count} jobs`
            : ""}
        </span>
        <Button size="sm" variant="primary" onClick={() => onNewSession(workspace.id)}>
          + 新建会话
        </Button>
        {confirmRemove ? (
          <span className="flex items-center gap-1 text-xs">
            <button
              type="button"
              className="text-danger hover:underline"
              onClick={() => void remove(workspace.id)}
            >
              确认移除
            </button>
            <button
              type="button"
              className="text-ink-3 hover:underline"
              onClick={() => setConfirmRemove(false)}
            >
              取消
            </button>
          </span>
        ) : (
          <Button size="sm" variant="ghost" onClick={() => setConfirmRemove(true)}>
            移除
          </Button>
        )}
      </div>

      {loading && sorted.length === 0 ? (
        <div className="flex items-center gap-2 py-6 text-sm text-ink-3">
          <Spinner size={16} /> 加载会话…
        </div>
      ) : sorted.length === 0 ? (
        <p className="py-4 text-sm text-ink-3">
          暂无会话——点「+ 新建会话」启动 plan/easy/doing…
        </p>
      ) : (
        <div className={`grid gap-3 ${compact ? "grid-cols-1" : "grid-cols-1 lg:grid-cols-2"}`}>
          {sorted.map((s) => (
            <SessionCard key={s.id} session={s} />
          ))}
        </div>
      )}
    </section>
  );
}

// ============================================================
// 页面
// ============================================================

export default function Sessions() {
  const list = useWorkspacesStore((s) => s.list);
  const loading = useWorkspacesStore((s) => s.loading);
  const error = useWorkspacesStore((s) => s.error);
  const refresh = useWorkspacesStore((s) => s.refresh);
  const isMobile = useIsMobile();

  const [modalOpen, setModalOpen] = useState(false);
  const [modalWs, setModalWs] = useState<string | null>(null);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const openNew = (wsId: string | null): void => {
    setModalWs(wsId);
    setModalOpen(true);
  };

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Sessions</h1>
        <Button size="sm" variant="primary" onClick={() => openNew(null)}>
          + 新建会话
        </Button>
      </header>

      <ErrorBanner message={error} onDismiss={() => void refresh()} />

      {loading && list.length === 0 ? (
        <div className="flex items-center gap-2 py-10 text-sm text-ink-3">
          <Spinner size={16} /> 加载工作区…
        </div>
      ) : list.length === 0 ? (
        <EmptyState
          message="还没有注册工作区"
          hint="添加一个含 .rick 目录的项目路径，开始用浏览器驱动 rick"
          action={
            <Button size="sm" variant="primary" onClick={() => openNew(null)}>
              去新建会话（先添加工作区）
            </Button>
          }
        />
      ) : (
        list.map((w) => (
          <WorkspaceSection
            key={w.id}
            workspaceId={w.id}
            onNewSession={openNew}
            compact={isMobile}
          />
        ))
      )}

      <AddWorkspaceCard onAdded={() => void refresh()} />

      <NewSessionModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        presetWorkspaceId={modalWs}
      />
    </div>
  );
}
