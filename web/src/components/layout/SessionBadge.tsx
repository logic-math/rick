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

import { useEffect, useState } from "react";
import { NavLink } from "react-router-dom";
import { useSessionsStore } from "../../stores/sessions";
import type { SessionInfo, SessionType } from "../../types";
import { deriveSessionDisplay } from "../../lib/sessionDisplay";
import { api } from "../../api/client";
import { useUiStore } from "../../stores/ui";
import SessionSettingsDialog, { sessionStatusBadge } from "../sessions/SessionSettingsDialog";

// Bug #5（human 2026-11-14 裁决）：已删除类型（plan/doing/learning/ctrl）不再显示
// 历史标签（历史数据保留给 agent 阅读感知，UI 不展示）——无映射类型不渲染徽标。
const TYPE_LABEL: Partial<Record<SessionType, string>> = {
  easy: "EASY",
  "human-loop": "HUMAN-LOOP",
  dream: "DREAM",
  super: "SUPER",
};

const TYPE_TONE: Partial<Record<SessionType, string>> = {
  easy: "border-portal/50 bg-portal-soft text-portal",
  "human-loop": "border-nebula/50 bg-nebula/10 text-nebula",
  dream: "border-nebula/50 bg-nebula/10 text-nebula",
  super: "border-morty/60 bg-morty/15 text-morty",
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
  // Bug #8 2A 终版（human 裁决 2026-11-15）：活跃 subagent 下拉——active 会话
  // 拉取活跃 subagent 列表（30s 轮询），点击跳转会话页右侧栏（只读子会话）。
  const openSubagentFromSidebar = useUiStore((s) => s.openSubagentFromSidebar);
  const [subagents, setSubagents] = useState<
    Array<{ runId: string; state: string; steps: Array<{ agent: string; label: string; status: string; currentTool?: string }> }>
  >([]);
  const [subagentsOpen, setSubagentsOpen] = useState(false);
  useEffect(() => {
    if (!subagentsOpen || session.status !== "active") return;
    let cancelled = false;
    const pull = () => {
      void api
        .listSubagents(session.id)
        .then((r) => {
          if (!cancelled) setSubagents(r.subagents ?? []);
        })
        .catch(() => {});
    };
    pull();
    const t = setInterval(pull, 30_000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [subagentsOpen, session.id, session.status]);
  // 标题兜底：优先**用户给 job 起的任务名**（server 返回 job_name）→ title →
  // 参数里的 job 编号（doing/ctrl/learning）→ 类型·短 id。
  // 「doing job_3」看不出在干什么——用户实测要求能给 job 起有意义的名字。
  const jobParam = (session.params as { job?: unknown } | undefined)?.job;
  // Bug #5 迭代5b（human 反馈：会话页也显示任务名）：显示名派生抽公共单源
  //（lib/sessionDisplay.ts）——侧栏与会话页共用（会话名 > job 任务名 > 输入主题
  // > title > 类型·短 id；idTag: params.id_tag 显式传递，title 正则兜底历史）。
  const { title: baseTitle, idTag } = deriveSessionDisplay(session);
  return (
    <div className="group relative flex min-w-0 items-center rounded-md">
      <NavLink
        to={`/session/${session.id}`}
        onClick={onNavigate}
        title={baseTitle}
        className={({ isActive }) =>
          `flex min-w-0 flex-1 items-center gap-1.5 rounded-md py-1 pl-2 pr-7 text-xs ${
            isActive ? "bg-portal-soft text-portal" : "text-ink-2 hover:bg-white/5 hover:text-ink"
          }`
        }
      >
        {TYPE_LABEL[session.type] && (
          <span
            className={`shrink-0 rounded border px-1 font-mono text-[8px] font-semibold tracking-wider ${TYPE_TONE[session.type] ?? ""}`}
          >
            {TYPE_LABEL[session.type]}
          </span>
        )}
        <span className="min-w-0 flex-1 truncate">{baseTitle}</span>
        {/* 归属徽标：job（会话被命名后标题可能不含编号，始终可见）；Bug #2：
            super 的 env_N / human-loop 的 loop_N 同款右侧徽标。 */}
        {typeof jobParam === "string" && jobParam && !baseTitle.includes(jobParam) && (
          <span
            className="shrink-0 rounded border border-line bg-space-2/60 px-1 font-mono text-[8px] leading-4 text-ink-3"
            title={`归属 job：${jobParam}`}
          >
            {jobParam}
          </span>
        )}
        {idTag && (
          <span
            className="shrink-0 rounded border border-line bg-space-2/60 px-1 font-mono text-[8px] leading-4 text-ink-3"
            title={`归属：${idTag}`}
          >
            {idTag}
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
        {session.status === "active" && (
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setSubagentsOpen((o) => !o);
            }}
            className="shrink-0 rounded px-1 text-[10px] text-ink-3 hover:text-portal"
            title="活跃 subagent 列表"
            aria-expanded={subagentsOpen}
          >
            {subagentsOpen ? "▾" : "▸"}⧉
          </button>
        )}
      </NavLink>
      {subagentsOpen && session.status === "active" && (
        <div className="ml-4 flex flex-col gap-0.5 py-0.5">
          {subagents.length === 0 && <p className="px-2 text-[10px] text-ink-3">无活跃 subagent</p>}
          {subagents.flatMap((run) =>
            run.steps.map((st) => (
              <button
                key={`${run.runId}-${st.label}`}
                type="button"
                onClick={() => {
                  openSubagentFromSidebar({ runId: run.runId, label: st.label, agent: st.agent });
                  onNavigate?.();
                }}
                className="flex min-w-0 items-center gap-1.5 rounded px-2 py-1 text-left text-[10px] text-ink-2 hover:bg-portal/10 hover:text-portal"
                title={`${st.agent} · ${st.label}——点击在会话页右侧栏打开（只读）`}
              >
                <span className="shrink-0">{st.status === "running" ? "⟳" : "✓"}</span>
                <span className="shrink-0 rounded bg-white/5 px-1 font-mono">{st.agent}</span>
                <span className="shrink-0">{st.label}</span>
                <span className="min-w-0 flex-1 truncate font-mono text-ink-3">{st.currentTool ?? ""}</span>
              </button>
            )),
          )}
        </div>
      )}
      {/* ⋮ 会话设置（hover 显示；stopPropagation 避免触发导航） */}
      <button
        type="button"
        aria-label={`${baseTitle} 设置`}
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
