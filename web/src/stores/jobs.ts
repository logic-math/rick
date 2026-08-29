/**
 * Jobs 看板状态：各工作区 jobs 快照 + SSE jobs_update diff 应用。
 *
 * 数据源：
 * - REST：listJobs(workspaceId) 快照
 * - SSE：jobs_update {job_id, diff, snapshot} —— snapshot 直接替换该 job
 *   （watcher 每次变更发全量 snapshot，diff 供 UI 高亮变更用）
 */

import { create } from "zustand";
import { api } from "../api/client";
import { sse, SSE_RESYNC_EVENT } from "../api/sse";
import type { JobSummary, TaskBrief } from "../types";

export interface JobsState {
  /** workspace_id → jobs 快照（REST 拉取 + SSE snapshot 合并） */
  byWorkspace: Map<string, JobSummary[]>;
  loadingByWorkspace: Map<string, boolean>;
  /** 最近一次 task 状态变更（job_id/task_id/from/to）——TaskBoard 闪一下高亮 */
  lastDiff: {
    job_id: string;
    task_id: string;
    from: string;
    to: string;
  } | null;
  /** 全局版本号（每次变更 ++）——订阅组件选择器依赖 */
  version: number;
  resyncCount: number;

  load: (workspaceId: string) => Promise<void>;
  /** SSE jobs_update 应用（snapshot 替换 + diff 记录） */
  applyUpdate: (workspaceId: string, data: {
    job_id: string;
    diff: Array<{ task_id: string; from: string; to: string }>;
    snapshot: TaskBrief[];
  }) => void;
  /** 读取某工作区 jobs（快照副本） */
  getJobs: (workspaceId: string) => JobSummary[];
  resyncAll: () => Promise<void>;
}

function applySnapshot(list: JobSummary[], jobId: string, tasks: TaskBrief[]): JobSummary[] {
  const i = list.findIndex((j) => j.job_id === jobId);
  const entry: JobSummary = {
    job_id: jobId,
    updated_at: new Date().toISOString(),
    tasks,
  };
  if (i === -1) return [entry, ...list]; // 新 job 置顶
  const next = [...list];
  next[i] = { ...next[i], ...entry };
  return next;
}

export const useJobsStore = create<JobsState>((set, get) => ({
  byWorkspace: new Map(),
  loadingByWorkspace: new Map(),
  lastDiff: null,
  version: 0,
  resyncCount: 0,

  load: async (workspaceId) => {
    set((s) => ({
      loadingByWorkspace: new Map(s.loadingByWorkspace).set(workspaceId, true),
    }));
    try {
      const list = await api.listJobs(workspaceId);
      set((s) => ({
        byWorkspace: new Map(s.byWorkspace).set(workspaceId, list),
        version: s.version + 1,
      }));
    } finally {
      set((s) => ({
        loadingByWorkspace: new Map(s.loadingByWorkspace).set(workspaceId, false),
      }));
    }
  },

  applyUpdate: (workspaceId, data) => {
    const last = data.diff[data.diff.length - 1] ?? null;
    const lastDiff = last ? { job_id: data.job_id, ...last } : null;
    set((s) => {
      const list = s.byWorkspace.get(workspaceId);
      if (!list) {
        // 该工作区尚未拉取快照——记 diff 等 REST 首载
        return { lastDiff, version: s.version + 1 };
      }
      return {
        byWorkspace: new Map(s.byWorkspace).set(
          workspaceId,
          applySnapshot(list, data.job_id, data.snapshot),
        ),
        lastDiff,
        version: s.version + 1,
      };
    });
  },

  getJobs: (workspaceId) => get().byWorkspace.get(workspaceId) ?? [],

  resyncAll: async () => {
    const ids = [...get().byWorkspace.keys()];
    await Promise.allSettled(ids.map((id) => get().load(id)));
    set((s) => ({ resyncCount: s.resyncCount + 1 }));
  },
}));

// ----------------------------------------------------------
// SSE 接线（App 挂载时调用一次；幂等）
// ----------------------------------------------------------

let wired = false;

export function wireJobs(): void {
  if (wired || typeof window === "undefined") return;
  wired = true;

  sse.on("jobs_update", (envelope) => {
    // jobs_update 的 session_id 恒为 null——工作区维度从 data 无法直接取，
    // 保守策略：对所有已加载工作区尝试应用（snapshot 含 job_id 命中才生效）。
    // 后续若契约增加 workspace_id 字段可精确路由（当前按契约实现）。
    const data = envelope.data as {
      job_id: string;
      diff: Array<{ task_id: string; from: string; to: string }>;
      snapshot: TaskBrief[];
    };
    if (!data?.job_id) return;
    const store = useJobsStore.getState();
    const wsIds = [...store.byWorkspace.keys()];
    for (const wsId of wsIds) {
      store.applyUpdate(wsId, data);
    }
  });

  window.addEventListener(SSE_RESYNC_EVENT, () => {
    void useJobsStore.getState().resyncAll();
  });
}
