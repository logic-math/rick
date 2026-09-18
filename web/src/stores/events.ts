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
import { useSessionsStore } from "./sessions";
import type { SSEEnvelope, SessionInfo } from "../types";

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

/** 打开块（in-flight message）增量帧上限——防御失控流（单条消息数万帧）。
 *  超限才退化为丢最旧增量（极端情况文本头可能缺失），正常消息远低于此。 */
const OPEN_BLOCK_MAX = 20000;

/** 打开块起点：最近一次 message_end 之后的帧属于「正在流式的消息」。
 *  vm 在 message_end 清空 liveText/liveThinking（applyMessageEnd）——打开块的
 *  完整证据（增量帧）只存在于该边界之后：裁掉它们 =
 *  ① live 文本（= 存活 delta 的顺序拼接）头部被吃 → 内容每帧跳变；
 *  ② live 块身份随之漂移（bug3 修复后身份锚定 message_end，但工具/文本累积
 *     仍依赖增量帧齐全）——两者都会让用户看到「思考过程不断闪烁刷新」。
 *  返回首个属于打开块的索引（无 message_end 时整段都是打开块 → 0）。 */
export function openBlockStart(buf: SSEEnvelope[]): number {
  for (let i = buf.length - 1; i >= 0; i--) {
    const env = buf[i];
    if (isDeltaFrame(env)) continue; // 增量帧不构成边界
    const ev = (env.data as { event?: { type?: string } })?.event;
    if (env.type === "session_event" && ev?.type === "message_end") return i + 1;
  }
  return 0;
}

/** 环形裁剪：
 *  - 结构事件（message_end / tool_execution_* / agent_* 等）永不裁；
 *  - **打开块（最近 message_end 之后）的增量帧永不裁**——它是当前流式块文本与
 *    身份（key）的唯一来源，裁掉即「思考过程不断闪烁刷新」（bug3）；
 *  - 只裁「已落定（打开块之前）」的最旧增量帧——完整事实由 message_end /
 *    tool_execution_end 承载，丢中间帧只损失打字机中间态。
 *  注：打开块 + 结构帧可能使缓冲暂时超过 MAX_PER_SESSION（打开块完整性优先）。 */
export function trimBuffer(buf: SSEEnvelope[]): SSEEnvelope[] {
  if (buf.length <= MAX_PER_SESSION) return buf;
  const openFrom = openBlockStart(buf);
  const drop = new Set<number>();

  // 打开块增量帧：仅超 OPEN_BLOCK_MAX 时退化丢最旧（防御失控流）
  const openDeltas: number[] = [];
  for (let i = openFrom; i < buf.length; i++) {
    if (isDeltaFrame(buf[i])) openDeltas.push(i);
  }
  if (openDeltas.length > OPEN_BLOCK_MAX) {
    for (const i of openDeltas.slice(0, openDeltas.length - OPEN_BLOCK_MAX)) drop.add(i);
  }

  // 已落定增量帧：可裁（丢最旧 overflow 条，顺序不变）
  const settledDeltas: number[] = [];
  for (let i = 0; i < openFrom; i++) {
    if (isDeltaFrame(buf[i])) settledDeltas.push(i);
  }
  const overflow = buf.length - drop.size - MAX_PER_SESSION;
  if (overflow > 0 && settledDeltas.length > 0) {
    const take = Math.min(overflow, settledDeltas.length);
    for (const i of settledDeltas.slice(0, take)) drop.add(i);
  }

  if (drop.size === 0) return buf;
  return buf.filter((_, i) => !drop.has(i));
}

export interface SessionEventState {
  /** session_id → 该会话的 envelope 环形缓冲（最近 N 条，时间序） */
  buffers: Map<string, SSEEnvelope[]>;
  /** session_id → 最新一次 session_state 事件 data（状态机视图） */
  states: Map<string, { status: string; reason?: string; busy?: boolean }>;
  /** 全局版本号（每次 flush ++）——订阅组件的选择器依赖 */
  version: number;
  /** SSE 重连后的全量刷新信号（版本号跳变 + resync 计数） */
  resyncCount: number;
}

interface SessionEventActions {
  /** SSE session_event 入缓冲（由 wireSessionEvents 接线调用） */
  push: (envelope: SSEEnvelope) => void;
  /** SSE session_state 更新（即时——状态变更低频无需批处理） */
  pushState: (sessionId: string, data: { status: string; reason?: string; busy?: boolean }) => void;
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

let flushRafId: number | null = null;
let flushTimerId: ReturnType<typeof setTimeout> | null = null;

/** 立即应用 pending 事件（可见性变化/卸载前调用，取消已排期的刷新）。 */
function flushNow(): void {
  if (flushRafId !== null && typeof cancelAnimationFrame === "function") {
    cancelAnimationFrame(flushRafId);
    flushRafId = null;
  }
  if (flushTimerId !== null) {
    clearTimeout(flushTimerId);
    flushTimerId = null;
  }
  flushScheduled = false;
  useSessionEventsStore.getState().flush();
}

function scheduleFlush(): void {
  if (flushScheduled) return;
  flushScheduled = true;
  const run = () => {
    if (!flushScheduled) return;
    flushNow();
  };
  // **双保险**：rAF 负责同帧合并（前台最顺滑、零额外延迟），定时器兜底保证
  // 「后台标签页 rAF 被浏览器暂停/节流」时仍按 ~50ms 批量刷新。
  // 旧实现是 rAF **或** setTimeout 二选一——后台标签页里事件会一直堆在 pending
  // 不应用，出现「agent 已返回内容、UI 长时间不更新」，直到切回标签页才补刷
  // （用户实测反馈的「迟迟不更新」路径之一）。
  if (typeof requestAnimationFrame === "function") {
    flushRafId = requestAnimationFrame(run);
  }
  flushTimerId = setTimeout(run, FLUSH_FALLBACK_MS);
}

// 可见性/卸载：立即落盘 pending（切走前把已收到的事件刷进 UI；切回时也刷一次，
// 保证回来即见最新——不依赖 rAF 恢复时机）。
if (typeof document !== "undefined") {
  document.addEventListener("visibilitychange", () => {
    if (flushScheduled) flushNow();
  });
}
if (typeof window !== "undefined") {
  window.addEventListener("pagehide", () => {
    if (flushScheduled) flushNow();
  });
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
    const data = envelope.data as { status?: string; reason?: string; busy?: boolean };
    if (!data?.status) return;
    useSessionEventsStore
      .getState()
      .pushState(envelope.session_id, { status: data.status, reason: data.reason, busy: data.busy });
    // 列表/侧栏状态同步（服务端权威）：session_state 必须同时更新 sessions store——
    // 此前只喂 events store（仅 ChatView 用），侧栏与 Sessions 卡片状态会**永久陈旧**
    // （只有 REST 重拉才更新）：resume/close/中断后列表不刷新 → 出现「会话已 active
    // 但卡片仍显示 Resume」→ 重复点击 Resume 得到 409（用户实测）。
    useSessionsStore
      .getState()
      .applyState(envelope.session_id, data.status as SessionInfo["status"], data.reason);
  });

  window.addEventListener(SSE_RESYNC_EVENT, () => {
    // 重连成功：版本跳变 + resync 计数——消费方据此重新拉 REST 快照
    useSessionEventsStore.setState((s) => ({
      version: s.version + 1,
      resyncCount: s.resyncCount + 1,
    }));
  });
}
