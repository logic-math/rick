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
 *
 * 游标续传（「页面刷新后台会话不断连」）：
 * 后台 pi worker 与 doing/dream 任务都跑在 Go server 进程内（与浏览器无关），
 * 页面只是显示与操作——**刷新/断网丢的只是事件流，不该丢状态**。为此本类把
 * 「重建点」持久化到 localStorage["rick-web-sse-cursor"]（节流 500ms；
 * pagehide/visibilitychange 隐藏时立即落盘），新建连接带 ?lastEventID=<重建点>，
 * 服务端从 1000 条环形重放缓冲补齐——正在流式的 thinking/text 在刷新后继续渲染。
 * - **重建点 ≠ 已处理前沿**：若某个会话仍有未落定的流式内容（打开块），重建点
 *   回退到该会话最近一次「落定」事件（message_end/tool_execution_end/agent_*）
 *   的 seq——刷新后服务端会重放整个打开块，新文档得以重建完整思考/正文
 *   （仅用前沿只能拿到刷新之后的增量，刷新前的部分会丢：实测 T2 缺失）。
 *   无未落定内容时重建点 = 前沿（增量续传，最省）。
 * - 游标过旧（缓冲已滑出 / 服务端重启后缓冲为空）：服务端回
 *   server_info{reason:"replay_overflow"} → 清游标 + 派发 resync（stores 拉
 *   REST 快照全量对齐）。
 * - seq 回退（重放严格 > 游标，故回退只可能是服务端重启 seq 归零）：视为新纪元
 *   ——重置游标、正常处理该事件并派发 resync（否则新事件会被单调去重永久丢弃）。
 * - 重放窗口可能包含历史 frontend_reload 事件：按 seq 去重（sessionStorage），
 *   否则刷新后重放同一事件会陷入无限重载。
 * - localStorage 不可用（隐私模式）→ 静默退化为无游标（等价旧行为）。
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

/** 续传游标持久化键（值=「重建点」seq 的十进制字符串） */
const CURSOR_STORAGE_KEY = "rick-web-sse-cursor";
/** frontend_reload 去重键（sessionStorage，值=已处理事件的 seq）——防重放循环 */
const RELOAD_SEQ_STORAGE_KEY = "rick-web-frontend-reload-seq";
/** 游标落盘节流间隔——高频 delta 不逐条写 localStorage */
const CURSOR_PERSIST_MS = 500;

/** 「落定」事件类型：该会话在此时刻的流式内容已固化（后续内容才是打开块）。 */
// 回退点 = **回合边界**（agent_end/agent_settled）——回合内事件（message_end /
// tool_execution_end）不能当重放起点：否则重放窗口丢掉本回合的 agent_start，
// 刷新后 server busy 与前端重放推断同时失效的场景会短暂显示 idle。
const SETTLE_EVENT_TYPES = new Set(["agent_end", "agent_settled"]);
/** 「内容」事件类型：出现即说明该会话有未落定的流式内容（打开块）。 */
const CONTENT_EVENT_TYPES = new Set(["message_update", "tool_execution_start", "tool_execution_update"]);

/** 读取持久化游标（无/损坏/不可用 → -1，等价「无游标」）。 */
function readStoredCursor(): number {
  if (typeof window === "undefined") return -1;
  try {
    const raw = window.localStorage.getItem(CURSOR_STORAGE_KEY);
    if (!raw) return -1;
    const n = Number.parseInt(raw, 10);
    return Number.isFinite(n) && n >= 0 ? n : -1;
  } catch {
    return -1; // localStorage 不可用（隐私模式等）——静默降级
  }
}

export class SseClient {
  private readonly eventsPath: string;
  private source: EventSource | null = null;
  private handlers = new Map<SSEEventType, Set<SseHandler>>();
  private retryCount = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private closedByUser = false;
  /** 最近一次已处理 seq（去重前沿；初始值来自持久化重建点——刷新后续传） */
  private lastSeq = -1;
  /** 缺口自愈：上次回退重连时间与连续失败次数（限流 + 退化为全量 resync）。 */
  private lastGapRecoveryAt = 0;
  private gapRecoveryStreak = 0;
  /** 本次断开开始时刻（用于判断是否为「短暂抖动」——短暂重连只靠游标重放补齐，
   *  不做全量 resync，避免无谓的重拉与界面刷新）。 */
  private downSince = 0;
  /** 本 document 首个数据事件 seq（0=未收到）——中途接入时的重建点下界 */
  private firstDataSeq = 0;
  /** session_id → 最近一次【落定】事件 seq（该会话打开块的重建点上界） */
  private settleSeq = new Map<string, number>();
  /** 仍有未落定流式内容的会话（打开块） */
  private openSessions = new Set<string>();
  /** 待落盘重建点 + 节流 timer */
  private pendingCursor = -1;
  private persistTimer: ReturnType<typeof setTimeout> | null = null;
  private flushHooked = false;

  constructor(eventsPath: string = "/api/events") {
    this.eventsPath = eventsPath;
    this.lastSeq = readStoredCursor();
    this.pendingCursor = this.lastSeq;
  }

  // ----------------------------------------------------------
  // 生命周期
  // ----------------------------------------------------------

  connect(): void {
    if (typeof window === "undefined") return; // SSR/测试环境保护
    this.closedByUser = false;
    this.hookFlush();
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
    this.flushPersistNow(); // 主动关闭前落盘游标
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
  // 游标持久化（刷新续传）
  // ----------------------------------------------------------

  /**
   * 视图重建点：新文档（刷新后）应从哪个 seq 之后重放事件，才能重建当前视图。
   *
   * - 无未落定内容 → 已处理前沿（只补增量，最省）。
   * - 有未落定内容（打开块）→ 该会话最近一次落定事件的 seq（服务端重放整个
   *   打开块）；若本 document 从流中途接入（未见过落定事件）→ 首帧 seq 前一位。
   */
  private resumeCursor(): number {
    if (this.openSessions.size === 0) return this.lastSeq;
    let point = Number.POSITIVE_INFINITY;
    for (const sid of this.openSessions) {
      const settle = this.settleSeq.get(sid) ?? 0;
      const candidate = settle > 0 ? settle : Math.max(0, this.firstDataSeq - 1);
      if (candidate < point) point = candidate;
    }
    return Number.isFinite(point) ? point : this.lastSeq;
  }

  /** 记录事件对「打开块」状态的影响（重建点计算依据）。 */
  private trackOpenBlock(envelope: SSEEnvelope): void {
    if (envelope.type !== "session_event") return;
    const ev = (envelope.data as { event?: { type?: string } } | null | undefined)?.event;
    const t = ev?.type;
    if (!t) return;
    const sid = envelope.session_id ?? "";
    if (CONTENT_EVENT_TYPES.has(t)) {
      this.openSessions.add(sid);
    } else if (SETTLE_EVENT_TYPES.has(t)) {
      this.settleSeq.set(sid, envelope.seq);
      this.openSessions.delete(sid);
    }
  }

  /** 节流落盘：多次 seq 前进合并为一次 localStorage 写（写的是重建点）。 */
  private scheduleCursorPersist(): void {
    if (typeof window === "undefined") return;
    this.pendingCursor = this.resumeCursor();
    if (this.persistTimer !== null) return;
    this.persistTimer = setTimeout(() => {
      this.persistTimer = null;
      this.writeCursor();
    }, CURSOR_PERSIST_MS);
  }

  private writeCursor(): void {
    if (this.pendingCursor < 0) return;
    try {
      window.localStorage.setItem(CURSOR_STORAGE_KEY, String(this.pendingCursor));
    } catch {
      // 静默降级（隐私模式/配额）——退化为无游标，等价旧行为
    }
  }

  /** 立即写入指定游标值（缺口自愈用；-1 表示清除游标）。 */
  private persistCursorNow(value: number): void {
    this.pendingCursor = value;
    if (typeof window === "undefined") return;
    try {
      if (value < 0) window.localStorage.removeItem(CURSOR_STORAGE_KEY);
      else window.localStorage.setItem(CURSOR_STORAGE_KEY, String(value));
    } catch {
      // 静默降级
    }
  }

  private flushPersistNow(): void {
    if (this.persistTimer !== null) {
      clearTimeout(this.persistTimer);
      this.persistTimer = null;
    }
    this.pendingCursor = this.resumeCursor();
    this.writeCursor();
  }

  /** 清游标：缓冲溢出 / 新纪元——下次连接全新开始（不做增量补齐）。 */
  private clearCursor(): void {
    this.pendingCursor = -1;
    if (this.persistTimer !== null) {
      clearTimeout(this.persistTimer);
      this.persistTimer = null;
    }
    try {
      window.localStorage.removeItem(CURSOR_STORAGE_KEY);
    } catch {
      // 静默降级
    }
  }

  /**
   * 页面隐藏/卸载（含 F5 刷新）前立即落盘——节流窗口内的事件序号不能丢，
   * 否则刷新后缺口比实际大（最坏情况丢掉刚收到的流式事件）。
   */
  private hookFlush(): void {
    if (this.flushHooked || typeof window === "undefined") return;
    this.flushHooked = true;
    const flush = (): void => this.flushPersistNow();
    window.addEventListener("pagehide", flush);
    window.addEventListener("beforeunload", flush);
    window.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "hidden") flush();
    });
  }

  // ----------------------------------------------------------
  // 内部
  // ----------------------------------------------------------

  /** 状态事件携带重试次数：UI 据此实现「短暂抖动不打扰、多次失败才提示」
   *  （用户反馈：断线重连提示老是闪出，显得连接不稳定）。 */
  private emitState(state: SseConnectionState): void {
    window.dispatchEvent(
      new CustomEvent(SSE_STATE_EVENT, { detail: { state, retryCount: this.retryCount } }),
    );
  }

  private dispatch(envelope: SSEEnvelope): void {
    this.handlers.get(envelope.type)?.forEach((h) => h(envelope));
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

  /**
   * server_info 是**控制消息**，不受 seq 单调去重约束：服务端每次连接先发
   * InitialInfo（seq 恒为 0），重连时它必然 ≤ lastSeq——若走去重会被丢掉，
   * 连接状态将卡在 reconnecting 且 resync 永不触发（实测缺陷）。故这里
   * 统一在去重之前处理。
   */
  private onServerInfo(envelope: SSEEnvelope): void {
    const data = envelope.data as { reason?: string } | null | undefined;
    const wasRetrying = this.retryCount > 0;
    this.retryCount = 0;
    this.emitState("open");

    if (data?.reason === "replay_overflow") {
      // 游标过旧（缓冲已滑出）或服务端重启后缓冲为空：本次连接无法补齐缺口。
      // 清持久化游标（下次连接全新开始，不重复撞溢出）；采纳本 envelope 的
      // seq（服务端填的是当前 hub seq）以继续消费实时事件；派发 resync 让
      // stores 拉 REST 快照全量对齐（状态由 server 维护，页面只显示）。
      this.clearCursor();
      // 无条件采纳服务端给的 seq（= 当前 hub seq，权威）——陈旧游标可能比它大
      // （重启后 hub 归零），沿用旧值会继续丢弃实时事件。
      this.lastSeq = envelope.seq;
      this.firstDataSeq = 0;
      this.openSessions.clear();
      this.settleSeq.clear();
      this.scheduleCursorPersist();
      this.dispatch(envelope);
      window.dispatchEvent(new CustomEvent(SSE_RESYNC_EVENT));
      return;
    }

    this.dispatch(envelope);
    if (wasRetrying) {
      // 重连成功：**仅当断线时间较长（≥5s）才做全量 resync**（重拉 sessions/jobs）。
      // 短暂抖动（几秒内自愈）由游标重放（?lastEventID）补齐即可——旧实现在每次重连
      // 都全量重拉，配合「空闲 SSE 被中间层掐断」会导致界面每几秒刷新一次、画面抖动
      // （用户实测）。
      const downMs = this.downSince > 0 ? Date.now() - this.downSince : 0;
      this.downSince = 0;
      if (downMs >= 5000) {
        window.dispatchEvent(new CustomEvent(SSE_RESYNC_EVENT));
      }
    } else {
      this.downSince = 0;
    }
  }

  /**
   * frontend_reload 去重：重放会带回早已处理过的 reload 事件（data 无 id），
   * 用 seq 记入 sessionStorage（跨刷新存活）——同一事件只重载一次。
   */
  private frontendReloadHandled(seq: number): boolean {
    try {
      const seen = window.sessionStorage.getItem(RELOAD_SEQ_STORAGE_KEY);
      if (seen !== null && Number(seen) === seq) return true;
      window.sessionStorage.setItem(RELOAD_SEQ_STORAGE_KEY, String(seq));
    } catch {
      // sessionStorage 不可用——退化为每次重载（旧行为）
    }
    return false;
  }

  /** 缺口自愈：把游标回退到 from 并立即重连（服务端按 lastEventID 重放缺失窗口）。
   *  限流：10s 内只做一次；连续 2 次仍失败 → 清游标并派发全量 resync（对齐 REST 真相）。 */
  private recoverGap(from: number, to: number): void {
    const now = Date.now();
    this.gapRecoveryStreak = now - this.lastGapRecoveryAt < 10_000 ? this.gapRecoveryStreak + 1 : 1;
    this.lastGapRecoveryAt = now;
    console.warn(`[sse] 事件缺口 seq ${from} → ${to}（丢 ${to - from - 1} 条）——回退游标重连补齐`);
    if (typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("rick-web:sse-gap", { detail: { from, to } }));
    }
    if (this.gapRecoveryStreak > 2) {
      // 反复缺口：环缓冲可能已滑出窗口——清游标做全量重连 + 状态 resync
      this.lastSeq = -1;
      this.gapRecoveryStreak = 0;
      this.persistCursorNow(-1);
      this.cleanupSource();
      if (!this.closedByUser) this.open();
      if (typeof window !== "undefined") window.dispatchEvent(new CustomEvent(SSE_RESYNC_EVENT));
      return;
    }
    this.lastSeq = from;           // 回退游标
    this.persistCursorNow(from);
    this.cleanupSource();
    if (!this.closedByUser) this.open();
  }

  private open(): void {
    const token = getStoredToken();
    const params = new URLSearchParams({ token });
    // 游标续传：带上最近已处理 seq，服务端从环形缓冲补齐断连期间事件
    if (this.lastSeq >= 0) params.set("lastEventID", String(this.lastSeq));
    const url = `${this.eventsPath}?${params.toString()}`;
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

      if (envelope.type === "server_info") {
        this.onServerInfo(envelope);
        return;
      }

      // 服务端重放严格 seq > 游标，故出现回退只可能是服务端重启（seq 归零）：
      // 视为新纪元——正常处理该事件并派发 resync（否则新 seq 一直被去重丢弃，
      // 页面将永久停在旧状态）。重启后 worker 已失活，全量对齐是必须的。
      // **seq 缺口检测（实时性兜底）**：正常流 seq 严格 +1。出现空洞说明事件在
      // 服务端（supervisor 队列满丢最旧 / Hub 慢消费丢最旧）或传输中被丢弃——
      // 若不处理，UI 会永久缺少这段内容（例如 agent 已返回、UI 少了一截且不再变化）。
      // 处理：回退游标到最后已处理 seq 并重连，服务端从环形缓冲重放补齐。
      if (this.lastSeq >= 0 && envelope.seq > this.lastSeq + 1) {
        this.recoverGap(this.lastSeq, envelope.seq);
      }
      const epochReset = envelope.seq <= this.lastSeq;
      this.lastSeq = envelope.seq;
      if (this.firstDataSeq === 0) this.firstDataSeq = envelope.seq;
      // 打开块追踪（重建点计算）——在 dispatch 前更新，且重放与实时事件同等对待
      this.trackOpenBlock(envelope);
      this.scheduleCursorPersist();
      if (epochReset) {
        window.dispatchEvent(new CustomEvent(SSE_RESYNC_EVENT));
      }

      this.dispatch(envelope);

      if (envelope.type === "frontend_reload") {
        // 覆盖层 dist 变更——整页重载拿新资产（游标已落盘，重载后自动续传）。
        // 重放窗口可能带回历史 frontend_reload（本就发生过的事件）——按 seq
        // 去重（sessionStorage），否则无限重载循环。
        if (!this.frontendReloadHandled(envelope.seq)) {
          window.location.reload();
        }
      }
    };

    source.onerror = () => {
      if (this.downSince === 0) this.downSince = Date.now();
      // 主动关闭并按指数退避重连（EventSource 内建重连不可控）
      this.cleanupSource();
      if (this.closedByUser) return;

      const delay = Math.min(MAX_RETRY_MS, BASE_RETRY_MS * 2 ** this.retryCount);
      this.retryCount += 1;
      this.emitState("reconnecting");
      this.flushPersistNow(); // 断线时先落盘当前游标，重连才能精准补齐
      this.retryTimer = setTimeout(() => {
        this.retryTimer = null;
        if (!this.closedByUser) this.open();
      }, delay);
    };
  }
}

/** 全局单例（App 挂载时 connect；组件经 on() 订阅） */
export const sse = new SseClient();
