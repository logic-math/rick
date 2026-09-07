/**
 * Sessions 对话视图（/ws/:wsId/sessions）。
 *
 * 单工作区会话列表（卡片 + 新建 + 移除）；closed 会话卡显示 Resume；
 * 会话卡点击 → /session/:id。工作区上下文由路由 /ws/:wsId 提供。
 *
 * 另导出 AddWorkspaceCard：全局「添加/浏览/创建工作区」卡片（无工作区
 * 空态首页使用；browse/create 为后端契约接口，后端未就绪时优雅降级）。
 */

import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, type FsStatusResult } from "../api/client";
import { useSessionsStore } from "../stores/sessions";
import { useWorkspacesStore } from "../stores/workspaces";
import type { SessionInfo, WorkspaceEntry } from "../types";

// 稳定引用兑底常量：selector 里 ?? [] 每次返回新数组 → React #185 黑屏（job_36 实测）。
const EMPTY_SESSIONS: SessionInfo[] = [];
import Button from "../components/common/Button";
import Dialog from "../components/common/Dialog";
import ErrorBanner from "../components/common/ErrorBanner";
import Spinner from "../components/common/Spinner";
import NewSessionModal from "../components/sessions/NewSessionModal";
import SessionSettingsDialog from "../components/sessions/SessionSettingsDialog";
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

function SessionCard({
  session,
  onMenu,
}: {
  session: SessionInfo;
  onMenu: (s: SessionInfo) => void;
}) {
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
        {/* ⋮ 会话设置（归档/恢复）——不触发卡片导航 */}
        <button
          type="button"
          aria-label={`${session.title || session.id} 设置`}
          title="会话设置（归档/恢复）"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onMenu(session);
          }}
          className="shrink-0 rounded px-1 py-0.5 text-sm leading-none text-ink-3 opacity-60 transition-colors hover:bg-white/10 hover:text-ink hover:opacity-100"
        >
          ⋮
        </button>
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
// 添加工作区卡片（注册/浏览/创建/多级选择——后端契约接口，未就绪优雅降级）
// ============================================================

/** 后端 browse 接口契约：GET /api/workspaces/browse?path=… → {workspaces:[{path,name}]} */
interface BrowseResult {
  workspaces: { path: string; name?: string }[];
}

function browseWorkspaces(path: string): Promise<BrowseResult> {
  const token = localStorage.getItem("rick-web-token") ?? "";
  return fetch(
    `/api/workspaces/browse?path=${encodeURIComponent(path || "/")}`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} },
  ).then(async (resp) => {
    if (!resp.ok) throw new Error(`浏览失败（HTTP ${resp.status}）`);
    return (await resp.json()) as BrowseResult;
  });
}

function createWorkspace(path: string, name?: string): Promise<WorkspaceEntry> {
  const token = localStorage.getItem("rick-web-token") ?? "";
  return fetch("/api/workspaces/create", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ path, name }),
  }).then(async (resp) => {
    if (!resp.ok) throw new Error(`创建失败（HTTP ${resp.status}）`);
    return (await resp.json()) as WorkspaceEntry;
  });
}

// 目录浏览器（多级路径选择）——文件对话框式逐级导航。
function DirBrowser({
  initialPath,
  onPick,
  onClose,
}: {
  initialPath: string;
  /** 用户选定目录（回填主输入框） */
  onPick: (dir: string) => void;
  onClose: () => void;
}) {
  const [cur, setCur] = useState(initialPath || "/");
  const [parent, setParent] = useState("");
  const [entries, setEntries] = useState<{ path: string; name: string }[]>([]);
  const [status, setStatus] = useState<FsStatusResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newName, setNewName] = useState("");

  async function loadDir(dir: string): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      const res = await api.fsList(dir);
      setCur(res.path);
      setParent(res.parent);
      setEntries(res.entries);
      try {
        setStatus(await api.fsStatus(res.path));
      } catch {
        setStatus(null);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setEntries([]);
    } finally {
      setBusy(false);
    }
  }

  // 初始载入（path 输入框当前值或 /）
  useEffect(() => {
    void loadDir(initialPath || "/");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function doMkdir(): Promise<void> {
    const name = newName.trim();
    if (!name) return;
    setBusy(true);
    setError(null);
    try {
      const res = await api.fsMkdir(cur, name);
      setNewName("");
      // 直接插入新目录（fs/list 有 100 条上限，新建的排字母序末尾可能被截断）
      setEntries((prev) => {
        const next = [...prev];
        if (!next.some((en) => en.path === res.path)) {
          next.push({ path: res.path, name });
          next.sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0));
        }
        return next;
      });
    } catch (e) {
      setError(`新建目录失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }

  // 面包屑：/ a b c
  const crumbs: { label: string; path: string }[] = [{ label: "/", path: "/" }];
  {
    let acc = "";
    for (const seg of cur.split("/")) {
      if (!seg) continue;
      acc += "/" + seg;
      crumbs.push({ label: seg, path: acc });
    }
  }

  const statusHint = status
    ? status.exists && status.is_dir && status.has_rick
      ? "该目录已是工作区（含 .rick）——返回后点「注册」"
      : status.exists && status.is_dir
        ? "该目录无 .rick——返回后点「创建」将初始化 .rick 结构"
        : "目录不存在——「创建」时会自动创建目录并初始化 .rick 结构"
    : "";

  return (
    <div className="flex flex-col gap-3">
      {/* 面包屑（逐级可点） */}
      <div className="flex flex-wrap items-center gap-1 rounded-lg border border-line bg-space/40 px-2 py-1.5">
        {crumbs.map((c, i) => (
          <span key={c.path} className="flex items-center gap-1">
            {i > 0 && <span className="text-ink-3">/</span>}
            <button
              type="button"
              onClick={() => void loadDir(c.path)}
              className="rounded px-1 py-0.5 font-mono text-[11px] text-ink-2 hover:bg-white/5 hover:text-portal"
              title={c.path}
            >
              {c.label}
            </button>
          </span>
        ))}
        <span className="ml-auto flex items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            onClick={() => void loadDir(parent || "/")}
            disabled={!parent}
            title="上级目录"
          >
            ⬆ 上级
          </Button>
        </span>
      </div>

      {/* 子目录列表 */}
      <div className="max-h-64 min-h-0 overflow-y-auto rounded-lg border border-line bg-space/40">
        {busy && entries.length === 0 ? (
          <p className="p-3 text-xs text-ink-3">加载中…</p>
        ) : entries.length === 0 ? (
          <p className="p-3 text-xs text-ink-3">（无子目录）</p>
        ) : (
          entries.map((en) => (
            <button
              key={en.path}
              type="button"
              onClick={() => void loadDir(en.path)}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-ink-2 hover:bg-white/5 hover:text-portal"
              title={en.path}
            >
              <span className="shrink-0 text-ink-3">📁</span>
              <span className="truncate font-mono">{en.name}</span>
            </button>
          ))
        )}
      </div>

      {/* 新建子目录 */}
      <div className="flex flex-col gap-1.5">
        <span className="text-[10px] uppercase tracking-wide text-ink-3">新建子目录</span>
        <div className="flex gap-2">
          <input
            className="min-w-0 flex-1 rounded-lg border border-line bg-space/60 px-3 py-1.5 text-sm text-ink placeholder:text-ink-3 focus:border-portal/60 focus:outline-none"
            placeholder="目录名（如 my-project）"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void doMkdir();
            }}
          />
          <Button size="sm" variant="ghost" onClick={() => void doMkdir()} disabled={!newName.trim() || busy}>
            创建
          </Button>
        </div>
      </div>

      {error && <p className="text-xs text-danger">{error}</p>}
      {statusHint && <p className="text-xs text-ink-2">{statusHint}</p>}

      {/* 底部操作：使用此目录 / 取消 */}
      <div className="flex items-center justify-end gap-2 border-t border-line pt-3">
        <Button variant="ghost" onClick={onClose}>
          取消
        </Button>
        <Button variant="primary" onClick={() => onPick(cur)}>
          使用此目录
        </Button>
      </div>
    </div>
  );
}

export function AddWorkspaceCard({ onAdded }: { onAdded: (id: string) => void }) {
  const add = useWorkspacesStore((s) => s.add);
  const [path, setPath] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // 浏览态
  const [browseResults, setBrowseResults] = useState<{ path: string; name?: string }[] | null>(null);
  const [browseBusy, setBrowseBusy] = useState(false);
  // 多级选择（目录浏览器）
  const [browserOpen, setBrowserOpen] = useState(false);

  async function submit(): Promise<void> {
    if (!path.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const entry = await add(path.trim(), name.trim() || undefined);
      setPath("");
      setName("");
      setBrowseResults(null);
      onAdded(entry.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function doBrowse(): Promise<void> {
    if (!path.trim()) return;
    setBrowseBusy(true);
    setError(null);
    try {
      const res = await browseWorkspaces(path.trim());
      setBrowseResults(res.workspaces ?? []);
      if (!res.workspaces || res.workspaces.length === 0) {
        setError("该目录下未发现含 .rick 的项目（可直接「创建」新工作区）");
      }
    } catch (e) {
      setError(`浏览接口暂不可用：${e instanceof Error ? e.message : String(e)}`);
      setBrowseResults(null);
    } finally {
      setBrowseBusy(false);
    }
  }

  async function doCreate(): Promise<void> {
    if (!path.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const entry = await createWorkspace(path.trim(), name.trim() || undefined);
      setPath("");
      setName("");
      setBrowseResults(null);
      onAdded(entry.id);
    } catch (e) {
      setError(`创建接口暂不可用：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }

  const FIELD =
    "w-full rounded-lg border border-line bg-space/60 px-3 py-2 text-sm text-ink placeholder:text-ink-3 focus:border-portal/60 focus:outline-none";

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-dashed border-line bg-space/30 p-4">
      <span className="text-sm font-medium text-ink">工作区</span>
      <p className="text-xs text-ink-3">
        输入含 <code className="rounded bg-white/5 px-1">.rick</code> 的绝对路径，或点「多级选择」逐级浏览——
        已有项目「注册」、无 .rick 目录「创建」、搜机器上已有的工作区「浏览」
      </p>
      <div className="flex flex-col gap-2 sm:flex-row">
        <input
          className={FIELD}
          placeholder="/abs/path/to/project"
          value={path}
          onChange={(e) => {
            setPath(e.target.value);
            setBrowseResults(null);
          }}
        />
        <input
          className={`${FIELD} sm:w-36`}
          placeholder="名称（可选）"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Button
          variant="primary"
          onClick={() => void submit()}
          loading={busy}
          disabled={!path.trim()}
          className="sm:w-24"
        >
          注册
        </Button>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          variant="ghost"
          onClick={() => void doCreate()}
          disabled={!path.trim()}
          title="目录不存在则自动创建并初始化 .rick 结构"
          className="border-portal/40 text-portal hover:border-portal hover:bg-portal/10"
        >
          ✨ 创建新工作区
        </Button>
        <Button
          variant="ghost"
          onClick={() => setBrowserOpen(true)}
          title="文件对话框式逐级选择目录（含新建子目录）"
          className="border-rick/40 text-rick hover:border-rick hover:bg-rick/10"
        >
          🖥 多级选择
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => void doBrowse()}
          loading={browseBusy}
          disabled={!path.trim()}
        >
          🔍 浏览已有工作区
        </Button>
      </div>
      <Dialog
        open={browserOpen}
        title="选择目录（多级）"
        onClose={() => setBrowserOpen(false)}
        contentClassName="w-full max-w-xl"
      >
        <DirBrowser
          initialPath={path}
          onClose={() => setBrowserOpen(false)}
          onPick={(dir) => {
            setPath(dir);
            setBrowseResults(null);
            // 名称未填时自动取目录名
            if (!name.trim()) {
              const segs = dir.split("/").filter(Boolean);
              if (segs.length > 0) setName(segs[segs.length - 1]);
            }
            setBrowserOpen(false);
          }}
        />
      </Dialog>
      {browseResults && browseResults.length > 0 && (
        <div className="flex flex-col gap-1 rounded-lg border border-line bg-space/40 p-2">
          <span className="text-[10px] uppercase tracking-wide text-ink-3">发现的工作区</span>
          {browseResults.map((r) => (
            <button
              key={r.path}
              type="button"
              onClick={() => {
                setPath(r.path);
                setBrowseResults(null);
              }}
              className="truncate rounded-md px-2 py-1 text-left text-xs text-ink-2 hover:bg-white/5 hover:text-portal"
              title={r.path}
            >
              {r.name ? `${r.name} · ` : ""}
              {r.path}
            </button>
          ))}
        </div>
      )}
      <ErrorBanner message={error} onDismiss={() => setError(null)} />
    </div>
  );
}

// ============================================================
// 会话归档区（分页查询 + 恢复）
// ============================================================

const ARCHIVE_PAGE_SIZE = 20;

function relativeTime(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const diff = Date.now() - t;
  const min = Math.floor(diff / 60000);
  if (min < 1) return "刚刚";
  if (min < 60) return `${min} 分钟前`;
  const h = Math.floor(min / 60);
  if (h < 24) return `${h} 小时前`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d} 天前`;
  return new Date(iso).toLocaleDateString();
}

function ArchivedSessionsSection({
  workspaceId,
  onRestored,
}: {
  workspaceId: string;
  /** 恢复成功后通知父级刷新主列表 */
  onRestored: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<SessionInfo[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  async function loadPage(nextOffset: number): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const page = await api.listArchivedSessions(workspaceId, {
        limit: ARCHIVE_PAGE_SIZE,
        offset: nextOffset,
      });
      setItems(page.items);
      setTotal(page.total);
      setOffset(nextOffset);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (open) void loadPage(0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, workspaceId]);

  async function restore(s: SessionInfo): Promise<void> {
    setBusyId(s.id);
    setError(null);
    try {
      await api.unarchiveSession(s.id);
      onRestored();
      // 重拉当前页（该条已离开归档区）
      if (items.length === 1 && offset > 0) {
        await loadPage(Math.max(0, offset - ARCHIVE_PAGE_SIZE));
      } else {
        await loadPage(offset);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusyId(null);
    }
  }

  const pages = Math.max(1, Math.ceil(total / ARCHIVE_PAGE_SIZE));
  const pageNo = Math.floor(offset / ARCHIVE_PAGE_SIZE) + 1;

  return (
    <section className="flex flex-col gap-2">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-fit items-center gap-2 rounded-md border border-line px-3 py-1.5 text-xs text-ink-3 transition-colors hover:border-portal/40 hover:text-portal"
      >
        <span aria-hidden="true">{open ? "▾" : "▸"}</span>
        已归档会话{open && total > 0 ? ` (${total})` : ""}
      </button>
      {open && error && <p className="text-xs text-danger">{error}</p>}
      {open && loading && items.length === 0 && (
        <p className="flex items-center gap-2 text-xs text-ink-3">
          <Spinner size={14} /> 加载归档会话…
        </p>
      )}
      {open && !loading && items.length === 0 && !error && (
        <p className="text-xs text-ink-3">暂无归档会话</p>
      )}
      {open && items.length > 0 && (
        <>
          <ul className="flex flex-col gap-1">
            {items.map((s) => (
              <li
                key={s.id}
                className="group flex items-center gap-3 rounded-md border border-dashed border-line px-3 py-1.5 text-xs"
              >
                <Link
                  to={`/session/${s.id}`}
                  className="flex min-w-0 flex-1 items-center gap-2 text-left"
                  title={`打开 ${s.id}`}
                >
                  <span
                    className={`shrink-0 rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wider ${TYPE_TONE[s.type] ?? "border-line text-ink-2"}`}
                  >
                    {s.type}
                  </span>
                  <span className="truncate text-ink-2 group-hover:text-portal">
                    {s.title || `${s.type} · ${s.id.slice(0, 8)}`}
                  </span>
                </Link>
                <span className="shrink-0 rounded-full border border-ink-3/30 bg-ink-3/10 px-1.5 py-0.5 text-[9px] leading-none text-ink-3">
                  {s.status}
                </span>
                <span className="shrink-0 text-[10px] text-ink-3">
                  {relativeTime(s.created_at)}
                </span>
                <button
                  type="button"
                  onClick={() => void restore(s)}
                  disabled={busyId === s.id}
                  className="ml-auto shrink-0 rounded border border-line px-2 py-0.5 text-[10px] text-portal hover:border-portal/50 hover:bg-portal/10 disabled:opacity-40"
                  title="恢复：回到默认列表（closed 会话点进用 Resume 重启）"
                >
                  {busyId === s.id ? "恢复中…" : "↩ 恢复"}
                </button>
              </li>
            ))}
          </ul>
          {/* 分页 */}
          {total > ARCHIVE_PAGE_SIZE && (
            <div className="flex items-center gap-3 text-[11px] text-ink-3">
              <button
                type="button"
                onClick={() => void loadPage(Math.max(0, offset - ARCHIVE_PAGE_SIZE))}
                disabled={offset <= 0 || loading}
                className="rounded border border-line px-2 py-0.5 hover:border-portal/40 hover:text-portal disabled:opacity-40"
              >
                ‹ 上一页
              </button>
              <span>
                {pageNo} / {pages}
              </span>
              <button
                type="button"
                onClick={() => void loadPage(offset + ARCHIVE_PAGE_SIZE)}
                disabled={offset + ARCHIVE_PAGE_SIZE >= total || loading}
                className="rounded border border-line px-2 py-0.5 hover:border-portal/40 hover:text-portal disabled:opacity-40"
              >
                下一页 ›
              </button>
            </div>
          )}
        </>
      )}
    </section>
  );
}

// ============================================================
// 工作区会话区块（单工作区）
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
  // 稳定引用兜底（同 App.tsx：?? [] 新数组会触发 React #185 无限重渲染）
  const sessions = useSessionsStore((s) => s.byWorkspace.get(workspaceId) ?? EMPTY_SESSIONS);
  const [confirmRemove, setConfirmRemove] = useState(false);
  // 会话设置（⋮ → 归档/恢复）目标
  const [menuSession, setMenuSession] = useState<SessionInfo | null>(null);

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
        <Button size="sm" variant="primary" onClick={() => onNewSession(workspaceId)}>
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
            <SessionCard key={s.id} session={s} onMenu={setMenuSession} />
          ))}
        </div>
      )}

      {/* 会话归档区（分页查询 + 恢复） */}
      <ArchivedSessionsSection
        workspaceId={workspaceId}
        onRestored={() => void load(workspaceId)}
      />

      {/* 会话设置弹层（⋮ → 归档/恢复） */}
      <SessionSettingsDialog
        session={menuSession}
        open={menuSession !== null}
        onClose={() => setMenuSession(null)}
        onChanged={() => void load(workspaceId)}
      />
    </section>
  );
}

// ============================================================
// 页面
// ============================================================

export default function Sessions({ workspaceId }: { workspaceId: string }) {
  const isMobile = useIsMobile();
  const [modalOpen, setModalOpen] = useState(false);

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Sessions</h1>
        <Button size="sm" variant="primary" onClick={() => setModalOpen(true)}>
          + 新建会话
        </Button>
      </header>

      <WorkspaceSection
        workspaceId={workspaceId}
        onNewSession={() => setModalOpen(true)}
        compact={isMobile}
      />

      <NewSessionModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        presetWorkspaceId={workspaceId}
      />
    </div>
  );
}
