/**
 * SubagentPanel：subagent 子会话右侧栏（Bug #8 2A 终版，human 裁决 2026-11-15）。
 *
 * - 数据：GET /api/subagent/{runId}/entries（后端按 status.json 定位子会话 JSONL
 *   离线解析）→ buildHistoryItems 渲染完整子会话输出（消息/工具/思考全部展示）。
 * - 轮询 3s（子会话是 pi 进程实时追加——离线解析即最新）；关闭即停。
 * - **只读**：无输入框（所有输入走主页面——steer 主会话；human 裁决）。
 * - 布局：ChatView 主区右侧（flex 分栏，宽 45%）。
 */
import { useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import { buildHistoryItems } from "./viewModel";
import type { SubagentProgress } from "./viewModel";
import MessageList from "./MessageList";

export interface SubagentTarget {
  runId: string;
  label: string;
  agent: string;
}

export default function SubagentPanel({
  target,
  progress,
  onClose,
}: {
  target: SubagentTarget;
  /** 实时进度（subagent_progress SSE 注入——头部状态行） */
  progress?: SubagentProgress;
  onClose: () => void;
}) {
  const [items, setItems] = useState<ReturnType<typeof buildHistoryItems>["items"]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // 轮询拉子会话（3s；关闭卸载即停）
  useEffect(() => {
    let cancelled = false;
    const load = () => {
      void api
        .subagentEntries(target.runId, target.label)
        .then((r) => {
          if (cancelled) return;
          const built = buildHistoryItems(r.entries as Parameters<typeof buildHistoryItems>[0]);
          setItems(built.items);
          setError(null);
        })
        .catch((e) => {
          if (!cancelled) setError(e instanceof Error ? e.message : String(e));
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    };
    load();
    const t = setInterval(load, 3000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [target.runId, target.label]);

  const step = useMemo(
    () => progress?.steps.find((s) => s.label === target.label),
    [progress, target.label],
  );

  return (
    <aside className="flex h-full min-h-0 w-full flex-col border-l border-line bg-space/40 md:w-[45%]">
      {/* 头部：agent · label · 状态 · 关闭 */}
      <div className="flex min-w-0 items-center gap-2 border-b border-line px-3 py-2">
        <span className="shrink-0 rounded bg-white/5 px-1.5 py-0.5 font-mono text-[10px] text-portal">
          subagent
        </span>
        <span className="min-w-0 flex-1 truncate text-xs font-medium text-ink">
          {target.agent} · {target.label}
        </span>
        {step?.status === "running" && <span className="shrink-0 text-[10px] text-portal">⟳ 运行中</span>}
        {step?.status === "completed" && <span className="shrink-0 text-[10px] text-ink-3">✓ 完成</span>}
        <button
          type="button"
          onClick={onClose}
          className="shrink-0 rounded px-1.5 py-0.5 text-xs text-ink-3 hover:bg-white/10 hover:text-ink"
          aria-label="关闭 subagent 会话"
          title="关闭"
        >
          ✕
        </button>
      </div>
      {/* 实时状态行（当前工具） */}
      {step?.currentTool && (
        <div className="border-b border-line/60 px-3 py-1 text-[10px] text-ink-3">
          <span className="font-mono" title={step.currentTool}>
            ⚡ {step.currentTool}
          </span>
        </div>
      )}
      {error && (
        <p className="px-3 py-2 text-xs text-danger">{error}</p>
      )}
      {/* 只读消息流（完整子会话输出；无输入框——输入走主页面） */}
      <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
        {loading && items.length === 0 ? (
          <p className="py-6 text-center text-xs text-ink-3">加载子会话…</p>
        ) : (
          <MessageList items={items} streaming={false} userSenderLabel="main-agent" />
        )}
      </div>
      <p className="border-t border-line/60 px-3 py-1 text-[10px] text-ink-3">
        只读视图——所有输入请在主页面操作（steer 主会话）。
      </p>
    </aside>
  );
}
