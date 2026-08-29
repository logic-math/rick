/**
 * /session/:id——按会话类型分派主视图。
 *
 * - 交互型（plan/easy/ctrl/human-loop/learning/dream[interactive]）→ ChatView
 * - 监控型（doing / dream[background]）→ MonitorView
 * 加载中 Spinner；加载失败 ErrorBanner（含返回）。
 */

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useSessionsStore } from "../stores/sessions";
import type { SessionInfo } from "../types";
import ChatView from "../components/chat/ChatView";
import MonitorView from "../components/monitor/MonitorView";
import ErrorBanner from "../components/common/ErrorBanner";
import Spinner from "../components/common/Spinner";

/** dream 交互模式（params.mode）也走聊天窗 */
function isMonitorType(session: SessionInfo | null): boolean {
  if (!session) return false;
  if (session.type === "doing") return true;
  if (session.type === "dream") {
    const mode = (session.params as Record<string, unknown>)?.mode;
    return mode !== "interactive"; // 契约默认 background
  }
  return false;
}

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

  return isMonitorType(session) ? (
    <MonitorView sessionId={session.id} />
  ) : (
    <ChatView sessionId={session.id} />
  );
}
