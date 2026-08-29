/**
 * ChatView：会话主视图（交互型会话——plan/easy/ctrl/human-loop/learning/dream）。
 *
 * 结构：会话头（类型徽标+title+状态点+Close）→ ErrorBanner（连接/服务错误）
 *      → MessageList（事件流渲染分层）→ ExtensionUIDialog（pending 对话框）
 *      → SteerBar（idle=prompt / streaming=steer+abort / closed=resume）
 *
 * 数据流：events store（该 session 的 SSE 环形缓冲）→ buildChatViewModel（纯函数）
 *      → items；挂载时 REST getEntries 补历史；乐观 user 消息 echo 消解。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "../../api/client";
import { useSessionEventsStore } from "../../stores/events";
import { useSessionsStore } from "../../stores/sessions";
import type { SessionStatus, SessionType } from "../../types";
import MessageList from "./MessageList";
import SteerBar, { type SessionPhase } from "./SteerBar";
import ExtensionUIDialog, { NotifyToasts } from "../extui/ExtensionUIDialog";
import {
  buildChatViewModel,
  buildHistoryItems,
  optimisticEchoed,
  type ChatItem,
} from "./viewModel";
import type { SlashCommand } from "./ChatInput";
import Portal from "../starfield/Portal";

/** 稳定空数组引用（选择器 `?? EMPTY` 避免每次 store 更新建新 [] 触发重渲染） */
const EMPTY_ENVELOPES: Parameters<typeof buildChatViewModel>[0] = [];

// ============================================================
// 会话头（类型徽标 + 状态点）
// ============================================================

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
  "human-loop": "border-nebula/60 bg-nebula/15 text-ink",
  learning: "border-rick/50 bg-rick/10 text-rick",
  dream: "border-nebula/60 bg-nebula/15 text-ink",
  doing: "border-morty/50 bg-morty/10 text-morty",
};

function StatusDot({ status, streaming }: { status: SessionStatus; streaming: boolean }) {
  if (streaming) {
    // 生成中：传送门绿呼吸（快）
    return (
      <span
        className="inline-block h-2.5 w-2.5 rounded-full bg-portal"
        style={{ animation: "rm-pulse 0.9s ease-in-out infinite" }}
        aria-label="生成中"
      />
    );
  }
  switch (status) {
    case "active":
      return (
        <span
          className="inline-block h-2.5 w-2.5 rounded-full bg-portal"
          style={{ animation: "rm-pulse 2.4s ease-in-out infinite" }}
          aria-label="活跃"
        />
      );
    case "running":
      return <span className="text-ink-3">▶</span>;
    case "closed":
      return <span className="inline-block h-2.5 w-2.5 rounded-full bg-ink-3/60" aria-label="已关闭" />;
    case "error":
      return <span className="inline-block h-2.5 w-2.5 rounded-full bg-danger" aria-label="错误" />;
  }
}

// ============================================================
// 主组件
// ============================================================

interface ChatViewProps {
  sessionId: string;
}

export default function ChatView({ sessionId }: ChatViewProps) {
  // events store：环形缓冲（flush 时新数组引用 → 选择器触发重渲染）
  const envelopes = useSessionEventsStore((s) => s.buffers.get(sessionId) ?? EMPTY_ENVELOPES);
  const sseState = useSessionEventsStore((s) => s.states.get(sessionId));
  const resyncCount = useSessionEventsStore((s) => s.resyncCount);

  // sessions store：会话元数据（miss 时回源）
  const info = useSessionsStore((s) => s.index.get(sessionId));
  const loadSession = useSessionsStore((s) => s.get);
  const applyState = useSessionsStore((s) => s.applyState);

  const [historyItems, setHistoryItems] = useState<ChatItem[] | null>(null);
  const [optimisticUser, setOptimisticUser] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const historyEpochRef = useRef(-1);
  const [historyRefresh, setHistoryRefresh] = useState(0);

  // 会话元数据回源（路由直达 / 刷新场景）
  useEffect(() => {
    if (!info) {
      void loadSession(sessionId);
    }
  }, [info, loadSession, sessionId]);

  // 挂载/SSE 重连/resume 后：REST 补历史（幂等——同 epoch 只拉一次）
  useEffect(() => {
    let cancelled = false;
    const epoch = resyncCount + historyRefresh;
    if (historyEpochRef.current === epoch) return;
    historyEpochRef.current = epoch;
    api
      .getEntries(sessionId)
      .then((resp) => {
        if (cancelled) return;
        setHistoryItems(
          buildHistoryItems(resp.entries as unknown as Parameters<typeof buildHistoryItems>[0]),
        );
      })
      .catch(() => {
        if (cancelled) return;
        setHistoryItems([]); // 历史拉取失败不阻塞 live 流
      });
    return () => {
      cancelled = true;
    };
  }, [sessionId, resyncCount, historyRefresh]);

  // 视图模型（纯函数重建——环形缓冲 ≤500 条，成本可控；版本号触发重算）
  const vm = useMemo(() => buildChatViewModel(envelopes, null), [envelopes]);

  // 乐观消息 echo 消解
  const echoed = optimisticEchoed(vm.items, optimisticUser);
  useEffect(() => {
    if (echoed && optimisticUser) setOptimisticUser(null);
  }, [echoed, optimisticUser]);

  // 合并视图：历史（前置）+ live
  const items = useMemo(() => {
    const live = optimisticUser
      ? [...vm.items, { kind: "user" as const, id: "optimistic-user", text: optimisticUser }]
      : vm.items;
    return historyItems ? [...historyItems, ...live] : live;
  }, [historyItems, vm.items, optimisticUser]);

  // 会话状态：SSE 状态事件 > REST 元数据
  const status: SessionStatus = (sseState?.status as SessionStatus) ?? info?.status ?? "active";
  const phase: SessionPhase = useMemo(() => {
    if (!info && !sseState && historyItems === null) return "loading";
    if (status === "closed") return "closed";
    if (status === "error") return "error";
    return vm.streaming ? "streaming" : "idle";
  }, [info, sseState, historyItems, status, vm.streaming]);

  // ============================================================
  // 动作
  // ============================================================

  const send = useCallback(
    async (message: string) => {
      setBusy(true);
      setError(null);
      setOptimisticUser(message);
      try {
        if (phase === "streaming") {
          await api.steer(sessionId, message);
        } else {
          await api.prompt(sessionId, message);
        }
      } catch (err) {
        setOptimisticUser(null);
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(false);
      }
    },
    [phase, sessionId],
  );

  const runCommand = useCallback(
    async (cmd: SlashCommand) => {
      setBusy(true);
      setError(null);
      try {
        if (cmd.name === "/abort") {
          await api.abort(sessionId);
        } else if (cmd.name === "/close") {
          await api.closeSession(sessionId);
          applyState(sessionId, "closed");
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(false);
      }
    },
    [applyState, sessionId],
  );

  const abort = useCallback(async () => {
    setBusy(true);
    try {
      await api.abort(sessionId);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, [sessionId]);

  const resume = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      await api.resumeSession(sessionId);
      applyState(sessionId, "active");
      setHistoryRefresh((n) => n + 1); // resume 后重拉历史（worker 可能已推进）
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, [applyState, sessionId]);

  const close = useCallback(async () => {
    setBusy(true);
    try {
      await api.closeSession(sessionId);
      applyState(sessionId, "closed");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, [applyState, sessionId]);

  // ============================================================
  // 渲染
  // ============================================================

  const sessionType: SessionType = info?.type ?? "plan";
  const title = info?.title ?? sessionId.slice(0, 8);

  return (
    <div className="relative flex h-full min-h-0 flex-col">
      {/* 会话头 */}
      <header className="flex shrink-0 items-center gap-2.5 border-b border-line bg-space/70 px-4 py-2.5 backdrop-blur">
        <Portal size={22} />
        <span
          className={`rounded border px-1.5 py-0.5 font-mono text-[10px] font-semibold tracking-wider ${TYPE_TONE[sessionType]}`}
        >
          {TYPE_LABEL[sessionType]}
        </span>
        <span className="min-w-0 truncate text-sm font-medium text-ink" title={title}>
          {title}
        </span>
        <span className="flex items-center gap-1.5 text-xs text-ink-3">
          <StatusDot status={status} streaming={vm.streaming} />
          {vm.streaming ? "streaming" : status}
        </span>
        {vm.compacting && <span className="text-xs text-morty">压缩中</span>}
        <div className="ml-auto flex items-center gap-1">
          {status === "active" && (
            <button
              type="button"
              onClick={() => void close()}
              disabled={busy}
              title="关闭会话（closed 后可 resume）"
              className="rounded-md border border-line px-2 py-1 text-xs text-ink-3 hover:border-danger/50 hover:text-danger disabled:opacity-40"
            >
              ✕ Close
            </button>
          )}
        </div>
      </header>

      {/* 连接/服务错误（二分之「顶部 banner」侧） */}
      {error && (
        <div className="flex items-start gap-2 border-b border-danger/40 bg-danger/10 px-4 py-2 text-xs text-danger">
          <span aria-hidden="true">⚠</span>
          <p className="min-w-0 flex-1 break-words">{error}</p>
          <button type="button" onClick={() => setError(null)} aria-label="关闭错误提示">
            ✕
          </button>
        </div>
      )}

      {/* fire-and-forget notify toasts */}
      <NotifyToasts sessionId={sessionId} notifications={vm.notifications} />

      {/* 消息流 */}
      <div className="min-h-0 flex-1">
        <MessageList items={items} streaming={vm.streaming} />
      </div>

      {/* extension_ui 对话框（agent 阻塞等待——渲染在输入条上方） */}
      <ExtensionUIDialog
        sessionId={sessionId}
        requests={vm.pendingDialogs}
        onResponse={(err) => {
          if (err) setError(err);
        }}
      />

      {/* 底部操作条 */}
      <div className="shrink-0 border-t border-line bg-space/80 backdrop-blur">
        <SteerBar
          phase={phase}
          onSend={(m) => void send(m)}
          onCommand={(c) => void runCommand(c)}
          onAbort={() => void abort()}
          onResume={() => void resume()}
          busy={busy}
          compacting={vm.compacting}
        />
      </div>
    </div>
  );
}
