/**
 * /session/:id——按会话类型分派主视图。
 *
 * - 交互型（plan/easy/ctrl/human-loop/learning/dream[interactive]）→ ChatView
 * -（job_53 Bug #5：doing 监控视图已删，全部类型走 ChatView）
 * 加载中 Spinner；加载失败 ErrorBanner（含返回）。
 */

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useSessionsStore } from "../stores/sessions";
import type { SessionInfo } from "../types";
import ChatView from "../components/chat/ChatView";
import ErrorBanner from "../components/common/ErrorBanner";
import Spinner from "../components/common/Spinner";


export default function SessionPage() {
  const { id: sessionId = "" } = useParams<{ id: string }>();
  const get = useSessionsStore((s) => s.get);
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setError(null);
    setSession(null);
    get(sessionId)
      .then((info) => {
        if (!alive) return;
        if (!info) {
          setError("会话不存在（可能已被清理或 id 无效）");
        } else {
          setSession(info);
        }
      })
      .catch((e) => {
        if (alive) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [sessionId, get]);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center gap-3 text-sm text-ink-3">
        <Spinner size={20} /> 加载会话…
      </div>
    );
  }

  if (error || !session) {
    return (
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-4 py-10">
        <ErrorBanner message={error} />
        <Link to="/" className="text-sm text-portal hover:underline">
          ← 返回 Sessions
        </Link>
      </div>
    );
  }

  // 所有会话类型走聊天窗（job_53 Bug #5：doing 监控视图已随 doing 类型删除；dream
  // 后台会话的进度事件流在聊天窗可见——session_event 事件流含 dream_progress）。
  // key=session.id 强制重挂：ChatView 的历史拉取 epoch 守卫不含 sessionId（ChatView.tsx
  // epoch = resyncCount + historyRefresh），切会话 A→B 时 effect 重跑但 epoch 相同提前
  // return，B 的历史永不拉取、界面保留 A 的消息。key 变化 → 组件整体重挂。
  return <ChatView key={session.id} sessionId={session.id} />;
}
