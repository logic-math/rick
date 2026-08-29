/**
 * rick web SSE 客户端 —— 单流多路复用（GET /api/events?token=）。
 *
 * 契约（api-contract.md SSE 节）：
 * - 每事件 id: <seq>（全局单调，服务端重启归零）+ data: <envelope>
 * - envelope: {seq, type, session_id, data}
 * - 心跳：15s 注释行（EventSource 透明）
 * - 断线重连：EventSource 自动重连并携带 Last-Event-ID（服务端重放缓冲）
 *   —— 但 EventSource 的自动重连不可控（间隔固定、无上限退避），
 *   故本类在 onerror 时主动 close() 并用指数退避重建连接（1s→30s 封顶）。
 * - frontend_reload → location.reload()（覆盖层 dist 变更自动生效）
 * - 重连成功（收到 server_info）后派发 "rick-web:sse-resync"——stores 收到后
 *   触发全量刷新（乐观保留的旧状态对齐服务端真相）
 * - 连接状态变更派发 "rick-web:sse-state"（detail=state）——ui store 的信号源
 */

import { getStoredToken } from "./client";
import type { SSEEnvelope, SSEEventType } from "../types";

export const SSE_RESYNC_EVENT = "rick-web:sse-resync";
export const SSE_STATE_EVENT = "rick-web:sse-state";

export type SseHandler = (envelope: SSEEnvelope) => void;

/** 连接状态（ui store 的信号源） */
export type SseConnectionState = "connecting" | "open" | "reconnecting" | "closed";

const MAX_RETRY_MS = 30_000;
const BASE_RETRY_MS = 1_000;

export class SseClient {
  private readonly eventsPath: string;
  private source: EventSource | null = null;
  private handlers = new Map<SSEEventType, Set<SseHandler>>();
  private retryCount = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private closedByUser = false;
  private lastSeq = -1;

  constructor(eventsPath: string = "/api/events") {
    this.eventsPath = eventsPath;
  }

  // ----------------------------------------------------------
  // 生命周期
  // ----------------------------------------------------------

  connect(): void {
    if (typeof window === "undefined") return; // SSR/测试环境保护
    this.closedByUser = false;
    if (this.source) return; // 已连接
    this.open();
  }

  close(): void {
    this.closedByUser = true;
    this.cleanupSource();
    if (this.retryTimer !== null) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
    this.emitState("closed");
  }

  // ----------------------------------------------------------
  // 订阅
  // ----------------------------------------------------------

  /** 按 envelope.type 订阅；返回退订函数 */
  on(type: SSEEventType, handler: SseHandler): () => void {
    let set = this.handlers.get(type);
    if (!set) {
      set = new Set();
      this.handlers.set(type, set);
    }
    set.add(handler);
    return () => {
      set!.delete(handler);
    };
  }

  /** 最近一次收到的 seq（诊断用） */
  get lastEventSeq(): number {
    return this.lastSeq;
  }

  // ----------------------------------------------------------
  // 内部
  // ----------------------------------------------------------

  private emitState(state: SseConnectionState): void {
    window.dispatchEvent(new CustomEvent(SSE_STATE_EVENT, { detail: state }));
  }

  private cleanupSource(): void {
    if (this.source) {
      // 先摘 handlers 防止 close 触发 onerror→重连
      this.source.onopen = null;
      this.source.onerror = null;
      this.source.close();
      this.source = null;
    }
  }

  private open(): void {
    const token = getStoredToken();
    const url = `${this.eventsPath}?token=${encodeURIComponent(token)}`;
    const source = new EventSource(url);
    this.source = source;
    this.emitState(this.retryCount > 0 ? "reconnecting" : "connecting");

    source.onmessage = (ev: MessageEvent<string>) => {
      let envelope: SSEEnvelope;
      try {
        envelope = JSON.parse(ev.data) as SSEEnvelope;
      } catch {
        return; // 非 JSON 行——忽略（心跳是注释行根本不会到这）
      }
      if (typeof envelope?.seq !== "number" || !envelope?.type) return;

      // seq 单调去重（重连后服务端重放缓冲可能带来旧事件）
      if (envelope.seq <= this.lastSeq) {
        // 服务端重启会使 seq 归零——探测回绕：明显小于 lastSeq 视为新纪元
        if (this.lastSeq - envelope.seq > 1_000_000) {
          this.lastSeq = -1;
        } else {
          return;
        }
      }
      this.lastSeq = envelope.seq;

      this.handlers.get(envelope.type)?.forEach((h) => h(envelope));

      if (envelope.type === "server_info") {
        const wasRetrying = this.retryCount > 0;
        this.retryCount = 0;
        this.emitState("open");
        if (wasRetrying) {
          // 重连成功——通知 stores 全量刷新（乐观保留状态对齐服务端）
          window.dispatchEvent(new CustomEvent(SSE_RESYNC_EVENT));
        }
      }

      if (envelope.type === "frontend_reload") {
        // 覆盖层 dist 变更——整页重载拿新资产
        window.location.reload();
      }
    };

    source.onerror = () => {
      // 主动关闭并按指数退避重连（EventSource 内建重连不可控）
      this.cleanupSource();
      if (this.closedByUser) return;

      const delay = Math.min(MAX_RETRY_MS, BASE_RETRY_MS * 2 ** this.retryCount);
      this.retryCount += 1;
      this.emitState("reconnecting");
      this.retryTimer = setTimeout(() => {
        this.retryTimer = null;
        if (!this.closedByUser) this.open();
      }, delay);
    };
  }
}

/** 全局单例（App 挂载时 connect；组件经 on() 订阅） */
export const sse = new SseClient();
