/**
 * 工作区列表状态：列表 + 增删 + loading 态。
 *
 * 数据源：ApiClient workspaces 接口；SSE 无工作区事件（注册表变更是
 * 本机操作触发——POST/DELETE 后主动 refresh 即可，无需事件驱动）。
 */

import { create } from "zustand";
import { api } from "../api/client";
import type { WorkspaceEntry } from "../types";

export interface WorkspacesState {
  /** 注册的工作区列表（按 added_at 倒序展示由组件处理） */
  list: WorkspaceEntry[];
  /** 当前选中的工作区 id（null = 未选择） */
  selectedId: string | null;
  loading: boolean;
  /** 最近一次加载错误（null = 无） */
  error: string | null;

  /** 拉取列表（幂等；失败写 error 不抛出——组件按 error 展示） */
  refresh: () => Promise<void>;
  /** 注册工作区（成功后刷新列表并选中） */
  add: (path: string, name?: string) => Promise<WorkspaceEntry>;
  /** 移除工作区（成功后刷新列表；若移除的是选中项则清空选择） */
  remove: (id: string) => Promise<void>;
  /** 选中工作区 */
  select: (id: string | null) => void;
}

export const useWorkspacesStore = create<WorkspacesState>((set, get) => ({
  list: [],
  selectedId: null,
  loading: false,
  error: null,

  refresh: async () => {
    set({ loading: true, error: null });
    try {
      const list = await api.getWorkspaces();
      // 保序：服务端返回顺序即展示顺序
      set({ list, loading: false });
    } catch (err) {
      set({
        loading: false,
        error: err instanceof Error ? err.message : String(err),
      });
    }
  },

  add: async (path, name) => {
    const entry = await api.addWorkspace({ path, name });
    await get().refresh();
    set({ selectedId: entry.id });
    return entry;
  },

  remove: async (id) => {
    await api.removeWorkspace(id);
    const wasSelected = get().selectedId === id;
    await get().refresh();
    if (wasSelected) set({ selectedId: null });
  },

  select: (id) => set({ selectedId: id }),
}));
