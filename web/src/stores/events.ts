/**
 * 流式事件批处理 store（OpenHands 防卡顿模式）。
 *
 * 数据源：SseClient session_event 透传（pi rpc 事件原样入 Data.event）。
 *
 * 设计：
 * - ring buffer：每 session 保留最近 MAX_PER_SESSION 条 envelope
 * - 批量 flush：事件先入 pending 缓冲，rAF（退化 setTimeout 50ms）批量
 *   应用并通知订阅组件——高频 message_update delta 不逐条触发 React 渲染
 * - 版本号（version）单调递增：订阅组件以版本号 + sessionId 选择器感知变更
 * - SSE 重连（"rick-web:sse-resync"）触发全量刷新信号——消费方重新拉
 *   REST 快照对齐服务端真相（本 store 保留乐观事件流）
 */

import { create } from "zustand";
import { sse, SSE_RESYNC_EVENT } from "../api/sse";
import type { SSEEnvelope } from "../types";

const MAX_PER_SESSION = 500;
const FLUSH_FALLBACK_MS = 50;

/** 纯增量帧（message_update/tool_execution_update）——可安全裁剪：
 *  落定事实由 message_end / tool_execution_end 承载，丢增量帧只损失流式
 *  中间态（打字机跳帧），不丢已落定的消息/工具卡。
 *  结构事件（落定/工具里程碑/回合边界）必须保留——否则环形裁剪把已渲染
 *  的「上一条 AI 消息」挤出窗口 → 消息从 UI 消失（无 resync 不回来）→
 *  用户实测「工具/思考流式时上一条 AI 消息一闪一闪」（bug2）。 */
export function isDeltaFrame(env: SSEEnvelope): boolean {
  if (env.type !== "session_event") return false;
  const ev = (env.data as { event?: { type?: string } })?.event;
  return ev?.type === "message_update" || ev?.type === "tool_execution_update";
}

/** 环形裁剪：优先丢最旧增量帧；仅当增量帧不足时才丢结构帧（防御退化）。 */
export function trimBuffer(buf: SSEEnvelope[]): SSEEnvelope[] {
  if (buf.length <= MAX_PER_SESSION) return buf;
  const deltas = buf.filter(isDeltaFrame);
  const overflow = buf.length - MAX_PER_SESSION;
  if (deltas.length >= overflow) {
    // 保留较新的 delta（丢最旧 overflow 条），结构帧全留，顺序不变
    const keep = new Set(deltas.slice(overflow));
    return buf.filter((env) => !isDeltaFrame(env) || keep.has(env));
  }
  // delta 不够（理论不会：结构事件量远小于上限）——全丢 delta 再裁最旧结构帧
  const rest = buf.filter((env) => !isDeltaFrame(env));
  return rest.length > MAX_PER_SESSION ? rest.slice(rest.length - MAX_PER_SESSION) : rest;
}

export interface SessionEventState {
  /** session_id → 该会话的 envelope 环形缓冲（最近 N 条，时间序） */
  buffers: Map<string, SSEEnvelope[]>;
  /** session_id → 最新一次 session_state 事件 data（状态机视图） */
  states: Map<string, { status: string; reason?: string }>;
  /** 全局版本号（每次 flush ++）——订阅组件的选择器依赖 */
  version: number;
  /** SSE 重连后的全量刷新信号（版本号跳变 + resync 计数） */
  resyncCount: number;
}

interface SessionEventActions {
  /** SSE session_event 入缓冲（由 wireSessionEvents 接线调用） */
  push: (envelope: SSEEnvelope) => void;
  /** SSE session_state 更新（即时——状态变更低频无需批处理） */
  pushState: (sessionId: string, data: { status: string; reason?: string }) => void;
  /** 手动 flush（测试/立即渲染场景） */
  flush: () => void;
  /** 读取某 session 的事件缓冲（快照副本） */
  getEvents: (sessionId: string) => SSEEnvelope[];
}

export type SessionEventStore = SessionEventState & SessionEventActions;

let pending: SSEEnvelope[] = [];
let flushScheduled = false;

export const useSessionEventsStore = create<SessionEventStore>((set, get) => ({
  buffers: new Map(),
  states: new Map(),
  version: 0,
  resyncCount: 0,

  push: (envelope) => {
    pending.push(envelope);
    scheduleFlush();
  },

  pushState: (sessionId, data) => {
    const states = new Map(get().states);
    states.set(sessionId, data);
    set({ states, version: get().version + 1 });
  },

  flush: () => {
    if (pending.length === 0) return;
    const batch = pending;
    pending = [];

    const buffers = new Map(get().buffers);
    for (const env of batch) {
      const sid = env.session_id;
      if (!sid) continue;
      const buf = [...(buffers.get(sid) ?? []), env];
      // 环形裁剪：结构事件（落定/工具里程碑）全保留，只丢最旧的增量帧
      buffers.set(sid, trimBuffer(buf));
    }
    set({ buffers, version: get().version + 1 });
  },

  getEvents: (sessionId) => get().buffers.get(sessionId) ?? [],
}));

function scheduleFlush(): void {
  if (flushScheduled) return;
  flushScheduled = true;
  const run = () => {
    flushScheduled = false;
    useSessionEventsStore.getState().flush();
  };
  if (typeof requestAnimationFrame === "function") {
    requestAnimationFrame(run);
  } else {
    setTimeout(run, FLUSH_FALLBACK_MS);
  }
}

// ----------------------------------------------------------
// SSE 接线（App 挂载时调用一次；幂等）
// ----------------------------------------------------------

let wired = false;

export function wireSessionEvents(): void {
  if (wired || typeof window === "undefined") return;
  wired = true;

  sse.on("session_event", (envelope) => {
    useSessionEventsStore.getState().push(envelope);
  });

  sse.on("session_state", (envelope) => {
    if (!envelope.session_id) return;
    const data = envelope.data as { status?: string; reason?: string };
    if (!data?.status) return;
    useSessionEventsStore
      .getState()
      .pushState(envelope.session_id, { status: data.status, reason: data.reason });
  });

  window.addEventListener(SSE_RESYNC_EVENT, () => {
    // 重连成功：版本跳变 + resync 计数——消费方据此重新拉 REST 快照
    useSessionEventsStore.setState((s) => ({
      version: s.version + 1,
      resyncCount: s.resyncCount + 1,
    }));
  });
}
