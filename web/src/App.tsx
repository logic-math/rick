/**
 * rick 应用壳（导航重构 v2）。
 *
 * 布局（768/1024 两档断点，与 routes/hooks.ts 的 JS 断点单一常量源对齐）：
 * - 左侧栏：logo（传送门）+「rick」+ 工作区树（每工作区展开
 *   Sessions/Jobs/Dreams 三视图 + 最近会话列表）+
 *   「新建会话」按钮；<768px 折叠 drawer（汉堡切换+遮罩）
 * - 顶部导航：功能区标识 + SSE 连接状态点（绿=open/黄=reconnecting/红=closed）+
 *   Settings 齿轮
 * - 主内容区：路由出口（/ws/:wsId/{sessions|jobs|dreams}、/session/:id、
 *   /settings、/ 重定向到首个工作区 sessions）
 * - 全局：TokenGate 锁页（authRequired）+ 全局错误横幅
 *
 * 数据接线（main.tsx 统一 wire）：wireUiEvents/wireSessions/wireSessionEvents/
 * wireJobs + sse.connect()。
 */

import { useEffect, useState } from "react";
import { BrowserRouter, Navigate, NavLink, Route, Routes, useParams } from "react-router-dom";
import StarfieldBackground from "./components/starfield/StarfieldBackground";
import Portal from "./components/starfield/Portal";
import Saucer from "./components/starfield/Saucer";
import Button from "./components/common/Button";
import ErrorBanner from "./components/common/ErrorBanner";
import EmptyState from "./components/common/EmptyState";
import Spinner from "./components/common/Spinner";
import NewSessionModal from "./components/sessions/NewSessionModal";
import TokenGate from "./components/sessions/TokenGate";
import WorkspaceTree from "./components/layout/WorkspaceTree";
import { useUiStore } from "./stores/ui";
import { useWorkspacesStore } from "./stores/workspaces";
import type { SseConnectionState } from "./api/sse";
import Jobs from "./routes/Jobs";
import Dreams from "./routes/Dreams";
import SessionPage from "./routes/SessionPage";
import Sessions, { AddWorkspaceCard } from "./routes/Sessions";
import Settings from "./routes/Settings";

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

/**
 * ConnectionBanner：断线/重连中的轻量提示（fixed 底部，不占布局、不阻塞操作）。
 *
 * 后台 pi worker 与 job 跑在 Go server 进程内——页面断连只影响「显示」，
 * 故文案明确「后台会话与任务继续运行」，避免用户误以为任务被打断；
 * 重连成功（open）后自动消失。首次加载的 connecting 不提示（避免闪一下）。
 */
function ConnectionBanner() {
  const connection = useUiStore((s) => s.connection);
  if (connection !== "reconnecting" && connection !== "closed") return null;
  const meta = CONNECTION_META[connection] ?? CONNECTION_META.closed;
  return (
    <div
      role="status"
      aria-live="polite"
      className="pointer-events-none fixed bottom-4 left-1/2 z-[100] -translate-x-1/2 px-3"
    >
      <div className="flex items-center gap-2 rounded-full border border-line bg-space-2/95 px-3 py-1.5 text-[11px] text-ink-2 shadow-lg backdrop-blur">
        <span
          className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${meta.pulse ? "animate-rm-pulse" : ""}`}
          style={{ backgroundColor: meta.color }}
        />
        <span>{connection === "reconnecting" ? "连接已断开，正在重连…" : "连接已断开"}</span>
        <span className="text-ink-3">后台会话与任务继续运行</span>
      </div>
    </div>
  );
}

// ============================================================
// 路由辅助
// ============================================================

/**
 * / 重定向：有工作区 → 首个工作区的 Sessions；无工作区 → 空态引导添加。
 * 等待 workspaces 加载完成前显示 Spinner（避免空列表闪烁误判）。
 */
function HomeRedirect() {
  const list = useWorkspacesStore((s) => s.list);
  const loading = useWorkspacesStore((s) => s.loading);
  const refresh = useWorkspacesStore((s) => s.refresh);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (loading && list.length === 0) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-sm text-ink-3">
        <Spinner size={16} /> 加载工作区…
      </div>
    );
  }
  if (list.length === 0) {
    return (
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
        <EmptyState
          message="还没有工作区"
          hint="添加一个含 .rick 目录的项目路径（或浏览/创建），开始用浏览器驱动 rick"
        />
        <AddWorkspaceCard onAdded={() => void refresh()} />
      </div>
    );
  }
  return <Navigate to={`/ws/${list[0].id}/sessions`} replace />;
}

/** /ws/:wsId/{sessions|jobs|dreams} —— 按路由参数渲染对应工作区视图 */
function WorkspaceRoutePage({ kind }: { kind: "sessions" | "jobs" | "dreams" }) {
  const { wsId = "" } = useParams<{ wsId: string }>();
  const list = useWorkspacesStore((s) => s.list);
  const loading = useWorkspacesStore((s) => s.loading);
  const ws = list.find((w) => w.id === wsId);

  if (!ws) {
    if (loading && list.length === 0) {
      return (
        <div className="flex items-center justify-center gap-2 py-16 text-sm text-ink-3">
          <Spinner size={16} /> 加载工作区…
        </div>
      );
    }
    return (
      <EmptyState message="工作区不存在" hint="从侧边栏工作区树选择一个工作区" />
    );
  }

  if (kind === "sessions") return <Sessions workspaceId={wsId} />;
  if (kind === "jobs") return <Jobs workspaceId={wsId} />;
  return <Dreams workspaceId={wsId} />;
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

  const closeSidebar = (): void => setSidebarOpen(false);

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
            onClick={closeSidebar}
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
            <span className="text-base font-semibold tracking-wide text-ink">rick</span>
          </div>

          {/* 工作区树（含 Sessions/Jobs/Dreams 子项 + 最近会话） */}
          <div className="mt-2 min-h-0 flex-1 overflow-y-auto pb-2">
            <WorkspaceTree onNewSession={(wsId) => openNewSession(wsId)} onNavigate={closeSidebar} />
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
            <span className="text-sm font-medium text-ink-2">rick</span>
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
              <Route path="/" element={<HomeRedirect />} />
              <Route path="/ws/:wsId/sessions" element={<WorkspaceRoutePage kind="sessions" />} />
              <Route path="/ws/:wsId/jobs" element={<WorkspaceRoutePage kind="jobs" />} />
              <Route path="/ws/:wsId/dreams" element={<WorkspaceRoutePage kind="dreams" />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="/session/:id" element={<SessionPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </main>
        </div>
      </div>

      {/* 断线/重连提示（后台任务不受影响） */}
      <ConnectionBanner />

      {/* 全局新建会话弹窗 */}
      <NewSessionModal
        open={newSessionOpen}
        onClose={() => setNewSessionOpen(false)}
        presetWorkspaceId={newSessionWs}
      />
    </BrowserRouter>
  );
}
