/**
 * rick web 应用壳（task12 组装版）。
 *
 * 布局（768/1024 两档断点，与 routes/hooks.ts 的 JS 断点单一常量源对齐）：
 * - 左侧栏：logo（传送门）+ 功能区导航 + 工作区列表（各工作区含会话入口）+
 *   「新建会话」按钮；<768px 折叠 drawer（汉堡切换+遮罩）
 * - 顶部导航：功能区标识 + SSE 连接状态点（绿=open/黄=reconnecting/红=closed）+
 *   Settings 齿轮
 * - 主内容区：路由出口（Sessions/Jobs/Knowledge/Settings/Session/:id）
 * - 全局：TokenGate 锁页（authRequired）+ 全局错误横幅
 *
 * 数据接线（main.tsx 统一 wire）：wireUiEvents/wireSessions/wireSessionEvents/
 * wireJobs + sse.connect()。
 */

import { useEffect, useState } from "react";
import { BrowserRouter, NavLink, Route, Routes } from "react-router-dom";
import StarfieldBackground from "./components/starfield/StarfieldBackground";
import Portal from "./components/starfield/Portal";
import Saucer from "./components/starfield/Saucer";
import Button from "./components/common/Button";
import ErrorBanner from "./components/common/ErrorBanner";
import NewSessionModal from "./components/sessions/NewSessionModal";
import TokenGate from "./components/sessions/TokenGate";
import { useSessionsStore } from "./stores/sessions";
import { useUiStore } from "./stores/ui";
import { useWorkspacesStore } from "./stores/workspaces";
import type { SseConnectionState } from "./api/sse";
import Jobs from "./routes/Jobs";
import Knowledge from "./routes/Knowledge";
import SessionPage from "./routes/SessionPage";
import Sessions from "./routes/Sessions";
import Settings from "./routes/Settings";

const NAV_ITEMS = [
  { to: "/", label: "Sessions" },
  { to: "/jobs", label: "Jobs" },
  { to: "/knowledge", label: "Knowledge" },
];

// ============================================================
// SSE 连接状态点（TopNav 右侧）
// ============================================================

const CONNECTION_META: Record<SseConnectionState, { color: string; label: string; pulse: boolean }> = {
  connecting: { color: "#ffd54a", label: "连接中", pulse: true },
  open: { color: "#39ff88", label: "已连接", pulse: false },
  reconnecting: { color: "#ffd54a", label: "重连中", pulse: true },
  closed: { color: "#ff5d5d", label: "已断开", pulse: false },
};

function ConnectionDot() {
  const connection = useUiStore((s) => s.connection);
  const meta = CONNECTION_META[connection] ?? CONNECTION_META.closed;
  return (
    <span className="flex items-center gap-1.5 text-[10px] text-ink-3" title={`SSE ${connection}`}>
      <span
        className={`inline-block h-2 w-2 rounded-full ${meta.pulse ? "animate-rm-pulse" : ""}`}
        style={{ backgroundColor: meta.color }}
      />
      <span className="hidden sm:inline">{meta.label}</span>
    </span>
  );
}

// ============================================================
// 侧栏工作区/会话树
// ============================================================

/** 侧栏导航点击后收起 drawer（移动端）——模块级回调，App 挂载时绑定 */
let closeSidebarOnNavigate: () => void = () => {};

function SidebarWorkspaces({ onNewSession }: { onNewSession: (wsId: string) => void }) {
  const workspaces = useWorkspacesStore((s) => s.list);
  const refresh = useWorkspacesStore((s) => s.refresh);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (workspaces.length === 0) {
    return (
      <p className="px-3 py-2 text-xs leading-relaxed text-ink-3">
        暂无工作区——到 Sessions 页添加
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-1">
      {workspaces.map((w) => (
        <SidebarWorkspaceGroup key={w.id} workspaceId={w.id} onNewSession={onNewSession} />
      ))}
    </div>
  );
}

function SidebarWorkspaceGroup({
  workspaceId,
  onNewSession,
}: {
  workspaceId: string;
  onNewSession: (wsId: string) => void;
}) {
  const workspace = useWorkspacesStore((s) => s.list.find((w) => w.id === workspaceId));
  const load = useSessionsStore((s) => s.load);
  const sessions = useSessionsStore((s) => s.byWorkspace.get(workspaceId) ?? []);

  useEffect(() => {
    void load(workspaceId);
  }, [workspaceId, load]);

  if (!workspace) return null;

  const sorted = sessions.slice().sort((a, b) => (a.created_at < b.created_at ? 1 : -1));

  return (
    <div className="flex flex-col">
      <div className="group flex items-center gap-1.5 px-3 py-1.5">
        <span className="truncate text-xs font-semibold uppercase tracking-wide text-ink-2">
          {workspace.name || workspace.path}
        </span>
        <button
          type="button"
          className="ml-auto hidden rounded px-1 text-xs text-ink-3 hover:text-portal group-hover:block"
          title={`在 ${workspace.name || workspace.path} 新建会话`}
          onClick={() => onNewSession(workspace.id)}
        >
          +
        </button>
      </div>
      {sorted.slice(0, 8).map((s) => (
        <NavLink
          key={s.id}
          to={`/session/${s.id}`}
          onClick={() => closeSidebarOnNavigate()}
          className={({ isActive }) =>
            `flex min-w-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-xs ${
              isActive ? "bg-portal-soft text-portal" : "text-ink-3 hover:bg-white/5 hover:text-ink-2"
            }`
          }
          title={s.title || `${s.type} 会话`}
        >
          <span
            className="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
            style={{
              backgroundColor:
                s.status === "active" || s.status === "running"
                  ? "#39ff88"
                  : s.status === "error"
                    ? "#ffd54a"
                    : "#5c6690",
            }}
          />
          <span className="truncate">{s.title || `${s.type} · ${s.id.slice(0, 8)}`}</span>
        </NavLink>
      ))}
      {sorted.length > 8 && (
        <span className="px-3 py-1 text-[10px] text-ink-3">…还有 {sorted.length - 8} 个</span>
      )}
    </div>
  );
}

// ============================================================
// 应用壳
// ============================================================

export default function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [newSessionOpen, setNewSessionOpen] = useState(false);
  const [newSessionWs, setNewSessionWs] = useState<string | null>(null);
  const authRequired = useUiStore((s) => s.authRequired);
  const globalError = useUiStore((s) => s.globalError);
  const setGlobalError = useUiStore((s) => s.setGlobalError);

  useEffect(() => {
    closeSidebarOnNavigate = () => setSidebarOpen(false);
  }, []);

  const openNewSession = (wsId: string | null): void => {
    setNewSessionWs(wsId);
    setNewSessionOpen(true);
  };

  return (
    <BrowserRouter>
      <StarfieldBackground />

      {/* 全局锁页 */}
      {authRequired && <TokenGate />}

      <div className="flex h-full">
        {/* 移动端遮罩 */}
        {sidebarOpen && (
          <div
            className="fixed inset-0 z-20 bg-black/50 md:hidden"
            onClick={() => setSidebarOpen(false)}
            aria-hidden="true"
          />
        )}

        {/* 侧栏 */}
        <aside
          className={`fixed inset-y-0 left-0 z-30 flex w-60 shrink-0 flex-col border-r border-line bg-space-2/80 backdrop-blur transition-transform md:static md:translate-x-0 ${
            sidebarOpen ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <div className="flex items-center gap-2 px-4 py-4">
            <Portal size={28} />
            <span className="text-base font-semibold tracking-wide text-ink">rick web</span>
          </div>

          <nav className="px-2">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === "/"}
                onClick={() => setSidebarOpen(false)}
                className={({ isActive }) =>
                  `block rounded-md px-3 py-2 text-sm ${
                    isActive
                      ? "bg-portal-soft text-portal"
                      : "text-ink-2 hover:bg-white/5 hover:text-ink"
                  }`
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>

          {/* 工作区 + 会话列表 */}
          <div className="mt-3 min-h-0 flex-1 overflow-y-auto pb-2">
            <SidebarWorkspaces onNewSession={(wsId) => openNewSession(wsId)} />
          </div>

          <div className="flex items-center gap-2 border-t border-line px-4 py-3">
            <Button
              size="sm"
              variant="primary"
              className="w-full"
              onClick={() => openNewSession(null)}
            >
              + 新建会话
            </Button>
          </div>
          <div className="flex items-center gap-2 px-4 pb-3 text-xs text-ink-3">
            <Saucer size={20} />
            <span>guardians of the context</span>
          </div>
        </aside>

        {/* 主区 */}
        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex items-center gap-3 border-b border-line bg-space/70 px-4 py-3 backdrop-blur">
            <button
              type="button"
              className="rounded-md border border-line px-2 py-1 text-sm text-ink-2 hover:text-ink md:hidden"
              onClick={() => setSidebarOpen((v) => !v)}
              aria-label="切换侧栏"
            >
              ☰
            </button>
            <span className="text-sm font-medium text-ink-2">rick web</span>
            <span className="ml-auto flex items-center gap-3">
              <ConnectionDot />
              <NavLink
                to="/settings"
                title="Settings"
                className="rounded-md border border-line px-2 py-1 text-sm text-ink-2 hover:border-portal/50 hover:text-portal"
              >
                ⚙
              </NavLink>
            </span>
          </header>

          {/* 全局错误横幅 */}
          {globalError && (
            <div className="px-4 pt-3">
              <ErrorBanner message={globalError} onDismiss={() => setGlobalError(null)} />
            </div>
          )}

          <main className="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
            <Routes>
              <Route path="/" element={<Sessions />} />
              <Route path="/jobs" element={<Jobs />} />
              <Route path="/knowledge" element={<Knowledge />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="/session/:id" element={<SessionPage />} />
              <Route
                path="*"
                element={
                  <div className="mx-auto max-w-2xl py-10 text-center">
                    <p className="text-sm text-ink-3">未知路由</p>
                  </div>
                }
              />
            </Routes>
          </main>
        </div>
      </div>

      {/* 全局新建会话弹窗 */}
      <NewSessionModal
        open={newSessionOpen}
        onClose={() => setNewSessionOpen(false)}
        presetWorkspaceId={newSessionWs}
      />
    </BrowserRouter>
  );
}
