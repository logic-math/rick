/**
 * SessionBadge：侧栏会话条目（工作区展开后的会话列表行）。
 *
 * 三态徽标（与 task-session-state 的状态语义对齐）：
 *   - 活跃（active/running）= 传送门绿呼吸点 + 「活跃」
 *   - 终止（error）        = Morty 黄 + 「已中断」
 *   - 完成（closed）       = 灰 + 「已完成」
 *
 * 整行可点击 → /session/:id（NavLink）；类型徽标 + 标题 + 状态徽标。
 */

import { NavLink } from "react-router-dom";
import type { SessionInfo, SessionType } from "../../types";

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

/** 状态徽标三态 */
function StatusBadge({ status }: { status: SessionInfo["status"] }) {
  if (status === "active" || status === "running") {
    return (
      <span className="flex shrink-0 items-center gap-1 rounded-full border border-portal/40 bg-portal/10 px-1.5 py-0.5 text-[9px] leading-none text-portal">
        <span className="inline-block h-1.5 w-1.5 animate-rm-pulse rounded-full bg-portal" />
        活跃
      </span>
    );
  }
  if (status === "error") {
    return (
      <span className="flex shrink-0 items-center gap-1 rounded-full border border-morty/40 bg-morty/10 px-1.5 py-0.5 text-[9px] leading-none text-morty">
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-morty" />
        已中断
      </span>
    );
  }
  return (
    <span className="flex shrink-0 items-center gap-1 rounded-full border border-ink-3/30 bg-ink-3/10 px-1.5 py-0.5 text-[9px] leading-none text-ink-3">
      <span className="inline-block h-1.5 w-1.5 rounded-full bg-ink-3" />
      已完成
    </span>
  );
}

export default function SessionBadge({
  session,
  onNavigate,
}: {
  session: SessionInfo;
  onNavigate?: () => void;
}) {
  const title = session.title || `${session.type} · ${session.id.slice(0, 8)}`;
  return (
    <NavLink
      to={`/session/${session.id}`}
      onClick={onNavigate}
      title={title}
      className={({ isActive }) =>
        `flex min-w-0 items-center gap-1.5 rounded-md px-2 py-1 text-xs ${
          isActive ? "bg-portal-soft text-portal" : "text-ink-2 hover:bg-white/5 hover:text-ink"
        }`
      }
    >
      <span
        className={`shrink-0 rounded border px-1 font-mono text-[8px] font-semibold tracking-wider ${TYPE_TONE[session.type]}`}
      >
        {TYPE_LABEL[session.type]}
      </span>
      <span className="min-w-0 flex-1 truncate">{title}</span>
      <StatusBadge status={session.status} />
    </NavLink>
  );
}
