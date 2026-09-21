/**
 * SessionBadge：侧栏会话条目（工作区展开后的会话列表行）。
 *
 * 四态徽标（与 task-session-state 的状态语义对齐）：
 *   - 活跃（active/running）= 传送门绿呼吸点 + 「活跃」
 *   - 终止（error）        = Morty 黄 + 「已中断」
 *   - 挂起（suspended）    = 灰 + 「⏸ 已挂起（平台升级）」——可一键恢复（不自动）
 *   - 完成（closed）       = 灰 + 「已完成」
 *
 * 整行可点击 → /session/:id（NavLink）；类型徽标 + 标题 + 状态徽标；
 * 行尾 hover ⋮ → 会话设置（人工归档/恢复，防误触入口）。
 * 归档/恢复成功后 load(workspace_id) 重拉（默认列表不含归档会话）。
 */

import { useState } from "react";
import { NavLink } from "react-router-dom";
import { useSessionsStore } from "../../stores/sessions";
import type { SessionInfo, SessionType } from "../../types";
import SessionSettingsDialog, { sessionStatusBadge } from "../sessions/SessionSettingsDialog";

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

export default function SessionBadge({
  session,
  onNavigate,
}: {
  session: SessionInfo;
  onNavigate?: () => void;
}) {
  const load = useSessionsStore((s) => s.load);
  const [settingsOpen, setSettingsOpen] = useState(false);
  // 标题兜底：优先**用户给 job 起的任务名**（server 返回 job_name）→ title →
  // 参数里的 job 编号（doing/ctrl/learning）→ 类型·短 id。
  // 「doing job_3」看不出在干什么——用户实测要求能给 job 起有意义的名字。
  const jobParam = (session.params as { job?: unknown } | undefined)?.job;
  const jobName = session.job_name?.trim() || "";
  const title =
    jobName ||
    session.title ||
    (typeof jobParam === "string" && jobParam ? `${session.type} ${jobParam}` : "") ||
    `${session.type} · ${session.id.slice(0, 8)}`;
  return (
    <div className="group relative flex min-w-0 items-center rounded-md">
      <NavLink
        to={`/session/${session.id}`}
        onClick={onNavigate}
        title={title}
        className={({ isActive }) =>
          `flex min-w-0 flex-1 items-center gap-1.5 rounded-md py-1 pl-2 pr-7 text-xs ${
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
        {/* job 归属徽标：会话被命名后标题可能不含编号，这里始终可见（点开会话 →
            会话设置里可复制完整编号与 job 工作目录）。 */}
        {typeof jobParam === "string" && jobParam && !title.includes(jobParam) && (
          <span
            className="shrink-0 rounded border border-line bg-space-2/60 px-1 font-mono text-[8px] leading-4 text-ink-3"
            title={`归属 job：${jobParam}`}
          >
            {jobParam}
          </span>
        )}
        {session.status === "active" || session.status === "running" ? (
          <span className="flex shrink-0 items-center gap-1 rounded-full border border-portal/40 bg-portal/10 px-1.5 py-0.5 text-[9px] leading-none text-portal">
            <span className="inline-block h-1.5 w-1.5 animate-rm-pulse rounded-full bg-portal" />
            活跃
          </span>
        ) : (
          sessionStatusBadge(session.status)
        )}
      </NavLink>
      {/* ⋮ 会话设置（hover 显示；stopPropagation 避免触发导航） */}
      <button
        type="button"
        aria-label={`${title} 设置`}
        title="会话设置（任务名重命名 / 归档 / 恢复）"
        onClick={() => setSettingsOpen(true)}
        /* 常显（低对比）+ hover 提亮：触摸设备没有 hover，旧版 opacity-0 会让
           「重命名/归档」入口**完全点不到**（用户实测：easy job_72 找不到重命名入口）。 */
        className="absolute right-0.5 top-1/2 z-10 -translate-y-1/2 rounded px-0.5 py-0.5 text-sm leading-none text-ink-3/70 transition-colors hover:bg-white/10 hover:text-ink"
      >
        ⋮
      </button>

      <SessionSettingsDialog
        session={session}
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        onChanged={() => void load(session.workspace_id)}
      />
    </div>
  );
}
