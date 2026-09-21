/**
 * RecoveryBanner：平台升级/重启后的**挂起与恢复台账**横幅。
 *
 * 为什么需要它：`rick tools release` 重启生产后，所有原本在跑的会话/任务会变成
 * `suspended`（human 裁决：**不自动恢复**，避免重复副作用与配额空转）。用户需要一个
 * 「刚才那次升级都挂了谁、恢复了没有」的集中视图，并能就地一键继续。
 *
 * 显示条件：报告里 suspended/failed 非空，或报告时间在 24h 内且 recovered 非空
 * （纯恢复成功的旧报告不值得长期占屏）。可关闭——关闭状态按报告 `at` 记住，同一份
 * 报告不再弹（新的升级会有新的 `at` → 重新出现）。
 *
 * 数据源：GET /api/recovery（server 权威）。动作全是**人工触发**：本组件绝不自动
 * 调 /continue 或重试。
 */

import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../../api/client";
import type { RecoveryItem, RecoveryReport } from "../../types";

const DISMISS_KEY = "rick-web-recovery-banner-dismissed";
/** 无未恢复项时，报告的有效展示窗口（之后不再占屏） */
const RECENT_WINDOW_MS = 24 * 60 * 60 * 1000;

function labelOf(item: RecoveryItem): string {
  const id = item.id.slice(0, 8);
  const title = item.title?.trim();
  if (title) return `${title}（${id}）`;
  if (item.job_id) return `${item.type ?? "session"} ${item.job_id}（${id}）`;
  return `${item.type ?? "session"} ${id}`;
}

export default function RecoveryBanner() {
  const navigate = useNavigate();
  const [report, setReport] = useState<RecoveryReport | null>(null);
  const [open, setOpen] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [dismissedAt, setDismissedAt] = useState<string>(() => {
    try {
      return localStorage.getItem(DISMISS_KEY) ?? "";
    } catch {
      return "";
    }
  });

  const load = useCallback(async (): Promise<void> => {
    try {
      const rep = await api.getRecoveryReport();
      setReport(rep);
    } catch {
      /* 无报告 / 旧后端 → 静默不显示（横幅不是关键路径） */
      setReport(null);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  /** 人工一键继续：**只由点击触发**（绝不自动）——成功后刷新台账并跳转会话页 */
  const continueOne = async (item: RecoveryItem): Promise<void> => {
    setBusyId(item.id);
    setMsg(null);
    try {
      const res = await api.continueSession(item.id);
      const normalized = res.normalized_tasks ?? [];
      setMsg(
        res.already_active
          ? `${labelOf(item)}：本就存活，已同步状态`
          : normalized.length > 0
            ? `${labelOf(item)}：已继续（归一化 ${normalized.length} 个 running task）`
            : `${labelOf(item)}：已继续 ✓`,
      );
      await load();
      navigate(`/session/${item.id}`);
    } catch (e) {
      setMsg(`${labelOf(item)}：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusyId(null);
    }
  };

  if (!report) return null;

  const suspended = report.suspended ?? [];
  const recovered = report.recovered ?? [];
  const failed = report.failed ?? [];
  const total = suspended.length + recovered.length + failed.length;
  if (total === 0) return null;

  const recent = Date.now() - new Date(report.at).getTime() < RECENT_WINDOW_MS;
  // 有未恢复/失败项 → 常显；否则只在 24h 窗口内显示
  if (suspended.length === 0 && failed.length === 0 && !recent) return null;
  if (dismissedAt && dismissedAt === report.at) return null;

  const dismiss = (): void => {
    try {
      localStorage.setItem(DISMISS_KEY, report.at);
    } catch {
      /* localStorage 不可用（隐私模式）→ 仅本次隐藏 */
    }
    setDismissedAt(report.at);
  };

  const tone =
    failed.length > 0
      ? "border-danger/40 bg-danger/10 text-danger"
      : suspended.length > 0
        ? "border-ink-3/40 bg-ink-3/10 text-ink-2"
        : "border-portal/40 bg-portal/10 text-portal";

  return (
    <div className={`mx-4 mt-3 rounded-lg border px-3 py-2 text-xs md:mx-6 ${tone}`}>
      <div className="flex flex-wrap items-center gap-2">
        <span aria-hidden="true">{failed.length > 0 ? "⚠" : suspended.length > 0 ? "⏸" : "✓"}</span>
        <span className="min-w-0 flex-1">
          <b>平台升级</b>（{new Date(report.at).toLocaleString()}）：
          {suspended.length > 0 && <>挂起 {suspended.length} 个</>}
          {recovered.length > 0 && <> · 已恢复 {recovered.length} 个</>}
          {failed.length > 0 && <> · 恢复失败 {failed.length} 个</>}
          {suspended.length > 0 && (
            <span className="text-ink-3">
              {" "}
              —— 点「继续」逐个恢复（不会自动续跑，避免副作用）
            </span>
          )}
        </span>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="rounded border border-line px-1.5 py-0.5 text-[10px] hover:border-portal/50 hover:text-portal"
        >
          {open ? "收起明细" : "明细"}
        </button>
        <button
          type="button"
          onClick={dismiss}
          aria-label="关闭恢复提示"
          className="rounded border border-line px-1.5 py-0.5 text-[10px] hover:border-portal/50 hover:text-portal"
        >
          ✕
        </button>
      </div>

      {open && (
        <div className="mt-2 flex flex-col gap-2 border-t border-line/60 pt-2">
          <Group title="⏸ 挂起（待人工恢复）" items={suspended} tone="text-ink-2">
            {(item) => (
              <button
                type="button"
                onClick={() => void continueOne(item)}
                disabled={busyId === item.id}
                title={
                  item.type === "doing" || item.type === "dream"
                    ? "继续执行：先把遗留 running task 归一化为 pending，再续跑剩余 task"
                    : "恢复继续：重新拉起该会话（不自动续跑）"
                }
                className="rounded border border-portal/50 px-1.5 py-0.5 text-[10px] text-portal hover:bg-portal/10 disabled:opacity-40"
              >
                {busyId === item.id ? "继续中…" : "▶ 继续"}
              </button>
            )}
          </Group>
          <Group title="✓ 已恢复（人工确认）" items={recovered} tone="text-portal" />
          <Group title="⚠ 恢复失败" items={failed} tone="text-danger" />
          {msg && <p className="text-[11px] text-ink-2">{msg}</p>}
        </div>
      )}
    </div>
  );
}

function Group({
  title,
  items,
  tone,
  children,
}: {
  title: string;
  items: RecoveryItem[];
  tone: string;
  children?: (item: RecoveryItem) => React.ReactNode;
}) {
  if (items.length === 0) return null;
  return (
    <div className="flex flex-col gap-1">
      <p className={`text-[10px] font-semibold uppercase tracking-wider ${tone}`}>{title}</p>
      <ul className="flex flex-col gap-1">
        {items.map((item) => (
          <li key={`${item.id}-${item.at}`} className="flex flex-wrap items-center gap-2">
            <span className="min-w-0 flex-1 truncate font-mono text-[10px] text-ink-3" title={item.reason}>
              {labelOf(item)}
              {item.reason ? <span className="ml-1.5 not-italic text-ink-3/80">· {item.reason}</span> : null}
            </span>
            {children?.(item)}
          </li>
        ))}
      </ul>
    </div>
  );
}
