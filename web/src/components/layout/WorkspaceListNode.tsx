/**
 * WorkspaceListNode：侧栏工作区节点（平铺列表的一个工作区行）。
 *
 * v4 设计（用户反馈：不要选择器切换，全部工作区直接展示）：
 *   - 最上一级 = 每个已注册工作区一行（名称 + 活跃状态点 + 未完成会话数徽标）
 *   - 点击整行 → 展开/折叠（默认折叠，折叠状态本地 useState）
 *   - 展开后子项：Sessions / Jobs / Dreams 三导航 + 该工作区会话列表（SessionBadge 三态）
 *   - 会话列表按 SSE session_state 实时同步（消费 sessions store）
 *
 * React #185 教训：selector ?? [] 必须用模块级常量（useSyncExternalStore
 * 每次快照新数组 → 无限重渲染）。
 */

import { useEffect, useMemo, useState } from "react";
import { NavLink } from "react-router-dom";
import { useSessionsStore } from "../../stores/sessions";
import { useWorkspacesStore } from "../../stores/workspaces";
import type { SessionInfo } from "../../types";
import SessionBadge from "./SessionBadge";

const EMPTY_SESSIONS: SessionInfo[] = [];

const WS_VIEWS = [
  { to: (id: string) => `/ws/${id}/sessions`, label: "Sessions" },
  { to: (id: string) => `/ws/${id}/jobs`, label: "Jobs" },
  { to: (id: string) => `/ws/${id}/dreams`, label: "Dreams" },
] as const;

interface WorkspaceListNodeProps {
  workspaceId: string;
  onNewSession: (wsId: string) => void;
  onNavigate?: () => void;
}

export default function WorkspaceListNode({
  workspaceId,
  onNewSession,
  onNavigate,
}: WorkspaceListNodeProps) {
  const workspace = useWorkspacesStore((s) => s.list.find((w) => w.id === workspaceId));
  const load = useSessionsStore((s) => s.load);
  // 稳定引用兜底（React #185）
  const sessions = useSessionsStore((s) => s.byWorkspace.get(workspaceId) ?? EMPTY_SESSIONS);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    void load(workspaceId);
  }, [workspaceId, load]);

  // 未完成会话（活跃/终止）——供行徽标与状态点统计
  const incomplete = useMemo(
    () => sessions.filter((s) => s.status === "active" || s.status === "running" || s.status === "error"),
    [sessions],
  );
  const hasLive = useMemo(
    () => incomplete.some((s) => s.status === "active" || s.status === "running"),
    [incomplete],
  );
  const hasError = useMemo(() => incomplete.some((s) => s.status === "error"), [incomplete]);

  if (!workspace) return null;

  const sorted = sessions
    .slice()
    .sort((a, b) => (a.created_at < b.created_at ? 1 : -1));

  const navClass = ({ isActive }: { isActive: boolean }): string =>
    `flex min-w-0 items-center gap-1.5 rounded-md px-3 py-1 text-xs ${
      isActive ? "bg-portal-soft text-portal" : "text-ink-3 hover:bg-white/5 hover:text-ink-2"
    }`;

  return (
    <div className="flex flex-col rounded-md transition-colors hover:bg-white/[0.02]">
      {/* 工作区行：点击展开/折叠 */}
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
        title={workspace.path}
        className="group flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-xs"
      >
        <span className={`shrink-0 text-ink-3 transition-transform ${expanded ? "" : "-rotate-90"}`}>
          ▾
        </span>
        <span className="min-w-0 flex-1 truncate font-semibold uppercase tracking-wide text-ink-2 group-hover:text-ink">
          {workspace.name || workspace.path}
        </span>
        {/* 活跃状态点：有 active/running 会话 → 绿呼吸；有 error → 黄点；否则灰点 */}
        <span
          className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${
            hasLive ? "animate-rm-pulse bg-portal" : hasError ? "bg-morty" : "bg-ink-3/40"
          }`}
          title={
            hasLive
              ? "有活跃会话"
              : hasError
                ? "有已中断会话"
                : "无未完成会话"
          }
        />
        {incomplete.length > 0 && (
          <span className="shrink-0 rounded-full border border-line bg-space-2/60 px-1.5 py-0.5 font-mono text-[9px] leading-none text-ink-2">
            {incomplete.length}
          </span>
        )}
      </button>

      {expanded && (
        <div className="ml-3 flex flex-col gap-0.5 border-l border-line pl-1.5 pb-1">
          {/* 三导航 */}
          {WS_VIEWS.map((v) => (
            <NavLink
              key={v.label}
              to={v.to(workspaceId)}
              end={v.label === "Sessions"}
              onClick={onNavigate}
              className={navClass}
            >
              {v.label}
            </NavLink>
          ))}

          {/* 会话列表（三态徽标） */}
          <div className="mt-1 flex flex-col gap-0.5">
            {sorted.length === 0 ? (
              <p className="px-3 py-1 text-[10px] text-ink-3">暂无会话</p>
            ) : (
              sorted.map((s) => <SessionBadge key={s.id} session={s} onNavigate={onNavigate} />)
            )}
          </div>

          {/* 本工作区新建会话 */}
          <button
            type="button"
            onClick={() => onNewSession(workspaceId)}
            className="mt-0.5 rounded px-2 py-1 text-left text-[10px] text-ink-3 transition-colors hover:bg-white/5 hover:text-portal"
            title={`在 ${workspace.name || workspace.path} 新建会话`}
          >
            + 新建会话
          </button>
        </div>
      )}
    </div>
  );
}
