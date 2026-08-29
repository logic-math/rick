import { useState } from "react";
import { BrowserRouter, NavLink, Route, Routes } from "react-router-dom";
import StarfieldBackground from "./components/starfield/StarfieldBackground";
import Portal from "./components/starfield/Portal";
import Saucer from "./components/starfield/Saucer";

/**
 * 布局壳（骨架版）：
 * - 左侧栏（md+ 静态；移动端 drawer 覆盖态）
 * - 顶部导航条（功能区标识 + 移动端汉堡）
 * - 主内容区（占位路由：Sessions / Jobs / Knowledge——task12 完整实现）
 */
function PagePlaceholder({ title, hint }: { title: string; hint: string }) {
  return (
    <section className="mx-auto max-w-3xl">
      <h1 className="text-xl font-semibold text-ink">{title}</h1>
      <p className="mt-2 text-sm text-ink-2">{hint}</p>
    </section>
  );
}

const NAV_ITEMS = [
  { to: "/", label: "Sessions" },
  { to: "/jobs", label: "Jobs" },
  { to: "/knowledge", label: "Knowledge" },
];

export default function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <BrowserRouter>
      <StarfieldBackground />
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
            <span className="text-base font-semibold tracking-wide text-ink">
              rick web
            </span>
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
          <div className="mt-auto flex items-center gap-2 px-4 py-3 text-xs text-ink-3">
            <Saucer size={20} />
            <span>skeleton v0.1</span>
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
            <span className="text-sm text-ink-2">rick web</span>
            <span className="ml-auto text-xs text-ink-3">
              对抗上下文熵增 · AICoding = Humans + Agents
            </span>
          </header>
          <main className="min-h-0 flex-1 overflow-y-auto p-6">
            <Routes>
              <Route
                path="/"
                element={
                  <PagePlaceholder
                    title="Sessions"
                    hint="会话列表（多工作区 + cmd 类型）——task12 完整实现"
                  />
                }
              />
              <Route
                path="/jobs"
                element={
                  <PagePlaceholder
                    title="Jobs"
                    hint="jobs 看板与任务状态——task10/12 实现"
                  />
                }
              />
              <Route
                path="/knowledge"
                element={
                  <PagePlaceholder
                    title="Knowledge"
                    hint="domain / loops / skills 知识库浏览——task10/12 实现"
                  />
                }
              />
            </Routes>
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
}
