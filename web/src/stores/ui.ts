/**
 * UI 全局状态：token 授权态 / 全局错误 / SSE 连接状态。
 *
 * 信号源（全局事件桥 wireUiEvents 接线，App 挂载时调用一次）：
 * - authRequired ← "rick-web:unauthorized"（ApiClient 401 时派发）
 * - connection   ← "rick-web:sse-state"（SseClient 状态变更时派发）
 */

import { create } from "zustand";
import { UNAUTHORIZED_EVENT } from "../api/client";
import { SSE_STATE_EVENT, type SseConnectionState } from "../api/sse";

export interface UiState {
  /** 401 未授权（token 缺失/错误）——锁页信号（task12 Settings 消费） */
  authRequired: boolean;
  /** 全局错误（banner 展示；组件用 setGlobalError(null) 清除） */
  globalError: string | null;
  /** SSE 连接状态（绿=open / 黄=reconnecting / 红=closed） */
  connection: SseConnectionState;
  /** 当前连续重试次数（0=已连接）——提示分级用（宽限期/网络波动/网络较弱） */
  retryCount: number;

  setAuthRequired: (v: boolean) => void;
  setGlobalError: (msg: string | null) => void;
  setConnection: (state: SseConnectionState) => void;
}

export const useUiStore = create<UiState>((set) => ({
  authRequired: false,
  globalError: null,
  connection: "connecting",
  retryCount: 0,

  setAuthRequired: (v) => set({ authRequired: v }),
  setGlobalError: (msg) => set({ globalError: msg }),
  setConnection: (state) => set({ connection: state }),
}));

// ----------------------------------------------------------
// 全局事件桥（App 挂载时 wire 一次；幂等）
// ----------------------------------------------------------

let wired = false;

export function wireUiEvents(): void {
  if (wired || typeof window === "undefined") return;
  wired = true;

  window.addEventListener(UNAUTHORIZED_EVENT, () => {
    useUiStore.getState().setAuthRequired(true);
  });

  window.addEventListener(SSE_STATE_EVENT, (ev) => {
    // 兼容两种 detail：字符串（旧）或 { state, retryCount }（新）
    const detail = (ev as CustomEvent<SseConnectionState | { state: SseConnectionState; retryCount?: number }>).detail;
    if (!detail) return;
    const state = typeof detail === "string" ? detail : detail.state;
    const retryCount = typeof detail === "string" ? 0 : (detail.retryCount ?? 0);
    if (state) {
      useUiStore.setState({ connection: state, retryCount });
    }
  });
}
