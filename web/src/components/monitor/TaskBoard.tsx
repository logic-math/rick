/**
 * 任务状态看板（监控视图左栏）。
 *
 * - 任务卡：task_id / name / status 徽标（pending 灰 / running 传送门绿脉动 /
 *   success Rick 蓝灰 / error Morty 黄+重试角标）+ commit_hash 短哈希
 * - jobs_update diff 驱动高亮：刚变更的 task 闪一下（ring 过渡）
 * - GateResult 横幅由父层（MonitorView）拼接在板顶
 */

import { useEffect, useState } from "react";
import type { JobSummary, TaskBrief } from "../../types";
import StatusDot from "../common/StatusDot";

/** TaskBrief.status → StatusDot 语义 */
function dotStatusOf(status: string): "pending" | "running" | "success" | "error" {
  switch (status) {
    case "running":
      return "running";
    case "success":
      return "success";
    case "error":
      return "error";
    default:
      return "pending";
  }
}

const STATUS_LABEL: Record<string, string> = {
  pending: "待执行",
  running: "执行中",
  success: "已完成",
  error: "失败",
};

function shortHash(h?: string): string {
  if (!h) return "";
  return h.slice(0, 7);
}

interface TaskCardProps {
  task: TaskBrief;
  /** 刚发生状态变更（diff 命中）——闪烁高亮 */
  flash: boolean;
  compact?: boolean;
}

export function TaskCard({ task, flash, compact = false }: TaskCardProps) {
  const [flashing, setFlashing] = useState(false);

  useEffect(() => {
    if (!flash) return;
    setFlashing(true);
    const t = setTimeout(() => setFlashing(false), 1600);
    return () => clearTimeout(t);
  }, [flash]);

  const dot = dotStatusOf(task.status);

  if (compact) {
    // JobDetail 横排简化版：单行
    return (
      <li
        className={`flex items-center gap-2 rounded-lg border px-3 py-2 transition-colors ${
          flashing ? "border-portal/70 bg-portal-soft" : "border-line bg-surface/50"
        }`}
      >
        <StatusDot status={dot} label={STATUS_LABEL[task.status] ?? task.status} />
        <span className="shrink-0 font-mono text-xs text-ink-3">{task.task_id}</span>
        <span className="min-w-0 flex-1 truncate text-sm text-ink-2" title={task.name}>
          {task.name}
        </span>
        <span
          className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium ${
            task.status === "error"
              ? "bg-morty/15 text-morty"
              : task.status === "running"
                ? "bg-portal/15 text-portal"
                : task.status === "success"
                  ? "bg-rick/15 text-rick"
                  : "bg-white/5 text-ink-3"
          }`}
        >
          {STATUS_LABEL[task.status] ?? task.status}
        </span>
        {task.commit_hash && (
          <span className="hidden shrink-0 font-mono text-[10px] text-ink-3 sm:inline" title={task.commit_hash}>
            {shortHash(task.commit_hash)}
          </span>
        )}
      </li>
    );
  }

  // MonitorView 完整卡
  return (
    <li
      className={`rounded-lg border p-3 transition-all duration-500 ${
        flashing
          ? "border-portal/70 bg-portal-soft shadow-[0_0_12px_rgba(57,255,136,0.15)]"
          : "border-line bg-surface/50"
      }`}
    >
      <div className="flex items-center gap-2">
        <StatusDot status={dot} label={STATUS_LABEL[task.status] ?? task.status} />
        <span className="font-mono text-xs font-semibold text-ink">{task.task_id}</span>
        <span
          className={`ml-auto rounded px-1.5 py-0.5 text-[10px] font-medium ${
            task.status === "error"
              ? "bg-morty/15 text-morty"
              : task.status === "running"
                ? "bg-portal/15 text-portal"
                : task.status === "success"
                  ? "bg-rick/15 text-rick"
                  : "bg-white/5 text-ink-3"
          }`}
        >
          {STATUS_LABEL[task.status] ?? task.status}
        </span>
      </div>
      <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-ink-2" title={task.name}>
        {task.name}
      </p>
      {task.commit_hash && (
        <p className="mt-1 font-mono text-[10px] text-ink-3" title={task.commit_hash}>
          commit {shortHash(task.commit_hash)}
        </p>
      )}
    </li>
  );
}

interface TaskBoardProps {
  job: JobSummary | null;
  /** 刚变更的 task_id（jobs_update diff 命中）——闪烁高亮 */
  flashTaskId: string | null;
  /** dream（无单一 job）时的占位说明 */
  placeholder?: string;
}

export default function TaskBoard({ job, flashTaskId, placeholder }: TaskBoardProps) {
  if (!job) {
    return (
      <div className="rounded-lg border border-dashed border-line bg-surface/30 px-4 py-6 text-center text-xs text-ink-3">
        {placeholder ?? "暂无任务快照（等待 jobs_update 事件）"}
      </div>
    );
  }

  const total = job.tasks.length;
  const done = job.tasks.filter((t) => t.status === "success").length;
  const failed = job.tasks.filter((t) => t.status === "error").length;

  return (
    <div className="flex min-h-0 flex-col gap-2">
      <div className="flex items-baseline gap-2">
        <h3 className="font-mono text-sm font-semibold text-ink">{job.job_id}</h3>
        <span className="text-xs text-ink-3">
          {done}/{total} 完成{failed > 0 ? ` · ${failed} 失败` : ""}
        </span>
      </div>
      <ul className="flex flex-col gap-2">
        {job.tasks.map((t) => (
          <TaskCard key={t.task_id} task={t} flash={flashTaskId === t.task_id} />
        ))}
      </ul>
    </div>
  );
}
