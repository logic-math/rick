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
import { deriveSessionDisplay } from "../../lib/sessionDisplay";
import SubagentPanel, { type SubagentTarget } from "./SubagentPanel";
import { useSessionEventsStore } from "../../stores/events";
import { useSessionsStore } from "../../stores/sessions";
import { useUiStore } from "../../stores/ui";
import type { SessionStatus, SessionPrompt, SessionType, SessionStats } from "../../types";
import { ApiError } from "../../types";
import MessageList from "./MessageList";
import SteerBar, { type SessionPhase } from "./SteerBar";
import ExtensionUIDialog, { NotifyToasts } from "../extui/ExtensionUIDialog";
import {
  buildChatViewModel,
  buildHistoryItems,
  optimisticEchoed,
  type HistoryBuildResult,
  type ChatItem,
} from "./viewModel";
import type { SlashCommand } from "./ChatInput";
import Portal from "../starfield/Portal";

/** 迭代 3（human 反馈）：items 里最后一张完整压缩卡（非 pending/aborted/
 *  无 errorMessage——历史卡缺省字段同样算成功）之后是否存在新对话项
 *  （user/assistant/thinking/tool；notice 不算对话）。
 *  - 存在 → 可压缩；不存在 → 前置拦截不调 API（免无谓 resume worker + pi 调用）。
 *  - 无压缩卡或 items 为空（历史未加载）→ true 放行（后端 pi 的
 *    Already compacted 错误兕底，前端 catch 处映射中文）。 */
function hasDialogueSinceLastCompaction(items: ChatItem[]): boolean {
  let lastCompaction = -1;
  for (let i = items.length - 1; i >= 0; i -= 1) {
    const it = items[i];
    if (it.kind === "compaction" && !it.pending && !it.aborted && !it.errorMessage) {
      lastCompaction = i;
      break;
    }
  }
  if (lastCompaction < 0) return true;
  return items
    .slice(lastCompaction + 1)
    .some(
      (it) =>
        it.kind === "user" ||
        it.kind === "assistant-text" ||
        it.kind === "thinking" ||
        it.kind === "tool",
    );
}
// 文件阅读器（Provider + Panel）已提升到 App 壳层（job_43 增量 5）——
// 消息链接的 useFileReader 在 Markdown/LocalAwareLink 内部，走全局 Context。
import Collapse from "../common/Collapse";
import type { HistoryMeta } from "./viewModel";
import ModelSwitchDialog from "./ModelSwitchDialog";

/** 稳定空数组引用（选择器 `?? EMPTY` 避免每次 store 更新建新 [] 触发重渲染） */
const EMPTY_ENVELOPES: Parameters<typeof buildChatViewModel>[0] = [];

/** 历史分页页大小（最近 N 条先渲染；滚动到顶/顶部按钮加载更早） */
const HISTORY_PAGE_SIZE = 50;

/** 挂载自动补拉上限（超长会话显示最近 MAX_HISTORY 条 + 顶部「加载更早」按钮继续） */
const MAX_HISTORY = 500;

// ============================================================
// 会话头（类型徽标 + 状态点）
// ============================================================

// Bug #5（human 2026-11-14）：已删除类型不显示历史标签（Partial + 条件渲染）
const TYPE_LABEL: Partial<Record<SessionType, string>> = {
  easy: "EASY",
  "human-loop": "HUMAN-LOOP",
  dream: "DREAM",
  super: "SUPER",
};

const TYPE_TONE: Partial<Record<SessionType, string>> = {
  easy: "border-portal/50 bg-portal-soft text-portal",
  "human-loop": "border-nebula/60 bg-nebula/15 text-ink",
  dream: "border-nebula/60 bg-nebula/15 text-ink",
  super: "border-morty/60 bg-morty/15 text-morty",
};

/** 会话头状态文案（挂起说「已挂起」而不是原始英文状态——用户看不出 suspended 是啥） */
const STATUS_LABEL: Record<SessionStatus, string> = {
  active: "active",
  running: "running",
  closed: "closed",
  error: "error",
  suspended: "已挂起",
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
    case "suspended":
      // 挂起（平台升级）：静态灰点 + ⏸ —— 与 error 红点、closed 灰点区分
      return (
        <span className="text-ink-3" aria-label="已挂起（平台升级）">
          ⏸
        </span>
      );
  }
}

// ============================================================
// 会话信息折叠区（系统提示词来源 + 元信息 + 模型/思考档位）
// ============================================================

/** 会话类型的 prompt 文件来源（显示路径提示——内容经 job files API 查看） */
const PROMPT_SOURCE: Partial<Record<SessionType, string>> = {
  easy: "<job>/doing/prompts/easy_main_prompt.md（+ method 系统提示词）",
  "human-loop": "<draft>/loops/loop_N/prompts/*.md（SENSE 四文件协议）",
  dream: ".rick/dream/（跨 job 反思，扫描已完成 jobs）",
  super: "<ws>/.rick/super/env_N/：super-loop + gen_env skill 中控（进化运行/观察/干预）",
};

function MetaRow({ label, value }: { label: string; value: React.ReactNode }) {
  if (value == null || value === "") return null;
  return (
    <div className="flex gap-2 py-0.5">
      <span className="w-20 shrink-0 text-right text-[10px] text-ink-3">{label}</span>
      <span className="min-w-0 flex-1 break-all font-mono text-[11px] text-ink-2">{value}</span>
    </div>
  );
}

function SessionInfoPanel({
  info,
  meta,
  workspaceId,
  onSwitchModel,
}: {
  info: { type: SessionType; params: Record<string, unknown>; pi_session_id?: string; created_at?: string; id: string } | null;
  meta: HistoryMeta;
  workspaceId: string | null;
  /** 点「切换」打开模型切换 Dialog */
  onSwitchModel: () => void;
}) {
  const type: SessionType = info?.type ?? "easy";
  const params = info?.params ?? {};
  const paramRows = Object.entries(params);
  const [prompt, setPrompt] = useState<SessionPrompt | null>(null);
  const [promptLoading, setPromptLoading] = useState(false);
  const [promptError, setPromptError] = useState<string | null>(null);
  const [promptOpen, setPromptOpen] = useState(false);

  async function loadPrompt(): Promise<void> {
    if (!workspaceId || !info?.id || prompt || promptLoading) return;
    setPromptLoading(true);
    setPromptError(null);
    try {
      const p = await api.getSessionPrompt(workspaceId, info.id);
      setPrompt(p);
      setPromptOpen(true);
    } catch (e) {
      setPromptError(e instanceof Error ? e.message : String(e));
    } finally {
      setPromptLoading(false);
    }
  }

  /** 「查看/收起原文」toggle（job_43 task9）：旧实现 onClick 直连 loadPrompt，
   *  但其守卫 `prompt || promptLoading` 在首次加载成功后永远提前 return——
   *  promptOpen 只有置 true 路径，「收起原文」点了没反应（旧缺陷）。
   *  现改分派：未加载 → loadPrompt（加载+展开）；已加载 → 切 promptOpen；
   *  loading 中忽略（按钮已 disabled，双保险）；错误态 prompt 仍 null → 可重试。 */
  function togglePrompt(): void {
    if (promptLoading) return;
    if (!prompt) {
      void loadPrompt();
      return;
    }
    // KR10a 字面分派：已展开 → 收起（false）；已收起 → 再展开（true）
    if (promptOpen) {
      setPromptOpen(false);
    } else {
      setPromptOpen(true);
    }
  }

  return (
    <Collapse
      className="mx-4 mt-2"
      title={
        <>
          <span className="text-ink-3">会话信息</span>
          <span className="ml-1 font-mono text-[10px] text-ink-3">（系统提示词原文 / 参数 / 模型）</span>
        </>
      }
      label="展开会话信息"
    >
      <div className="px-3 py-2">
        <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-3">系统提示词（原文）</div>
        <p className="pb-1 font-mono text-[11px] leading-relaxed text-ink-2">
          {PROMPT_SOURCE[type] ?? "（已删除类型的会话——历史数据保留，无活动协议）"}
        </p>
        <button
          type="button"
          onClick={togglePrompt}
          disabled={!workspaceId || promptLoading}
          className="mb-2 rounded border border-line px-2 py-0.5 text-[10px] text-portal hover:border-portal/50 hover:bg-portal/10 disabled:opacity-50"
        >
          {promptLoading
            ? "加载中…"
            : prompt
              ? promptOpen
                ? "收起原文"
                : "展开原文"
              : "查看原文"}
        </button>
        {promptError && <p className="pb-1 text-[10px] text-danger">原文加载失败：{promptError}</p>}
        {prompt && promptOpen && (
          <div className="mb-2 flex flex-col gap-2 rounded border border-line bg-space/60 p-2">
            <details open>
              <summary className="cursor-pointer text-[10px] uppercase tracking-wide text-ink-3">
                method 系统提示词（{prompt.method.length} 字符）
              </summary>
              <pre className="mt-1 max-h-72 overflow-y-auto whitespace-pre-wrap break-words rounded bg-black/30 p-2 font-mono text-[11px] leading-relaxed text-ink-2">
                {prompt.method}
              </pre>
            </details>
            <details open={promptOpen}>
              <summary className="cursor-pointer text-[10px] uppercase tracking-wide text-ink-3">
                instance prompt（{prompt.instance.length} 字符）
              </summary>
              <pre className="mt-1 max-h-96 overflow-y-auto whitespace-pre-wrap break-words rounded bg-black/30 p-2 font-mono text-[11px] leading-relaxed text-ink-2">
                {prompt.instance}
              </pre>
            </details>
          </div>
        )}
        <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-3">参数</div>
        <div className="pb-2">
          {paramRows.length === 0 ? (
            <p className="font-mono text-[11px] text-ink-3">（无）</p>
          ) : (
            paramRows.map(([k, v]) => (
              <MetaRow key={k} label={k} value={typeof v === "string" ? v : JSON.stringify(v)} />
            ))
          )}
        </div>
        <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-3">运行时</div>
        <div className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <MetaRow label="模型" value={meta.model} />
            <MetaRow label="provider" value={meta.provider} />
            <MetaRow label="思考档位" value={meta.thinkingLevel} />
          </div>
          <button
            type="button"
            onClick={onSwitchModel}
            className="shrink-0 rounded-md border border-portal/50 px-2 py-0.5 text-[10px] text-portal transition-colors hover:bg-portal/10 hover:border-portal"
            title="切换 pi 支持的模型 / 思考档位（/model）"
          >
            切换
          </button>
        </div>
        <MetaRow label="启动时间" value={meta.startedAt} />
        <MetaRow label="pi 会话" value={info?.pi_session_id} />
      </div>
    </Collapse>
  );
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

  const [history, setHistory] = useState<HistoryBuildResult | null>(null);
  // 历史分页：earliestId=已加载最早 entry id（before 游标）；hasMore=是否还有更早
  const [paging, setPaging] = useState<{ earliestId: string | null; hasMore: boolean }>({
    earliestId: null,
    hasMore: false,
  });
  const loadingEarlierRef = useRef(false);
  /** 加载更早进行中（驱动顶部按钮的「加载中…」态——ref 不响应式，需 state 镜像） */
  const [loadingEarlier, setLoadingEarlier] = useState(false);
  const [optimisticUser, setOptimisticUser] = useState<string | null>(null);
  /** 乐观消息的时序锚点：发送时刻该会话已到达的最大 envelope seq（见合并逻辑）。
   *  seq > anchor 的 live 项属于本次发送之后产生的回复。 */
  const optAnchorRef = useRef(0);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const historyEpochRef = useRef(-1);
  const [historyRefresh, setHistoryRefresh] = useState(0);
  const [modelDialogOpen, setModelDialogOpen] = useState(false);
  const [modelHint, setModelHint] = useState<string | null>(null);

  // 会话元数据回源（路由直达 / 刷新场景）
  useEffect(() => {
    if (!info) {
      void loadSession(sessionId);
    }
  }, [info, loadSession, sessionId]);

  // 公共前置合并：把更早一页 built 结果前置合并到已渲染 history（items 前置 + 指纹
  // 并集 + meta 保持最新页）。loadEarlier 与挂载自动补拉共用，避免逻辑分叉。
  const prependHistory = useCallback((built: HistoryBuildResult) => {
    setHistory((prev) =>
      prev
        ? {
            items: [...built.items, ...prev.items],
            fingerprints: new Set([...built.fingerprints, ...prev.fingerprints]),
            meta: prev.meta, // meta 取最新页（首屏已含模型信息）
          }
        : prev,
    );
  }, []);

  // 挂载/SSE 重连/resume 后：REST 补历史（幂等——同 epoch 只拉一次）
  // hadHistoryRef：首次完整加载过历史后置 true——重建（SSE resync/刷新）时
  // 拉**全量**（不带 limit）覆盖，避免「重建只拉最近 50 条 → 254 条历史缩水 →
  // 全部重挂 + enter 动画重放 = 使用中历史闪烁 + override 展开态丢失」。
  const hadHistoryRef = useRef(false);
  // Bug #8 第三批（human 2026-11-15）：滚动游标跨设备保留——打开会话恢复到上次
  // 阅读位置（游标前 N 条 + 定位），滚动时 debounce 存服务端 drafts（scroll:<sid>）。
  const SCROLL_KEY_PREFIX = "scroll:";
  const scrollAnchorRef = useRef<string | null>(null);
  const scrollSaveTimerRef = useRef<number | null>(null);
  /** 视口锚回调（MessageList rangeChanged）——debounce 1s 存服务端（跨设备恢复） */
  const handleViewportAnchor = useCallback(
    (anchorId: string | null) => {
      scrollAnchorRef.current = anchorId;
      if (!anchorId) return;
      if (scrollSaveTimerRef.current !== null) window.clearTimeout(scrollSaveTimerRef.current);
      scrollSaveTimerRef.current = window.setTimeout(() => {
        scrollSaveTimerRef.current = null;
        // DEBUG（游标记录链路定位）：saveDraft 成败可见
        void api.saveDraft(SCROLL_KEY_PREFIX + sessionId, anchorId)
          .then(() => console.log("[scroll-cursor] saved:", anchorId))
          .catch(() => {});
      }, 1000);
    },
    [sessionId],
  );
  // 首次打开的定位锚（历史加载 effect 内拉游标后设置——**时序修复**：此前游标
  // getDraft 与历史加载并行竞争，历史先拉完（默认最近 50 贴底），游标后到永远
  // 不生效（human 实测复现）。现并入同一 effect：先 await 游标 → 再决定拉取参数。
  const [initialScrollAnchor, setInitialScrollAnchor] = useState<string | null>(null);
  // 性能优化：游标判定完成前不渲染消息列表（消除「先贴底→scrollToIndex 跳中间」的闪烁）
  const [cursorReady, setCursorReady] = useState(false);
  useEffect(() => {
    let cancelled = false;
    const epoch = resyncCount + historyRefresh;
    if (historyEpochRef.current === epoch) return;
    historyEpochRef.current = epoch;
    const isRebuild = hadHistoryRef.current;
    if (isRebuild) setCursorReady(true); // 重建：不等游标
    // Bug #8 第三批时序修复（human 实测复现）：先拉游标（恢复阅读位置）再拉历史——
    // 此前 getDraft 与历史加载并行竞争，历史先拉完（默认最近 50 贴底），游标后到
    // 永远不生效。现并入同一 async 流程：await 游标 → 解析 entryId → 决定拉取参数。
    void (async () => {
      // Bug #8 第四批（human 反馈 2026-11-16）：① 游标拉取加重试（网络抖动不丢游标）
      // ② 游标中心懒加载窗口（只渲染游标上下游 N 条——长会话快速首屏；全量加载
      //    在 500+ entries 时慢且无必要）。无游标时保持默认路径（最近 N + auto-comp）。
      const CURSOR_RETRIES = 3;
      const CURSOR_RETRY_DELAY = 200;
      const fetchCursor = async (): Promise<string | null> => {
        for (let i = 0; i < CURSOR_RETRIES; i++) {
          try {
            const d = await api.getDraft(SCROLL_KEY_PREFIX + sessionId);
            if (d.content && d.content.length >= 4) return d.content;
            return null; // 有响应但无游标——不再重试
          } catch {
            if (i < CURSOR_RETRIES - 1) await new Promise((r) => setTimeout(r, CURSOR_RETRY_DELAY));
          }
        }
        return null; // 全部重试失败——降级无游标
      };

      // ① 游标（重试）+ entries **并行**（有游标时用窗口拉取，无游标用默认）
      const cursorPromise = isRebuild ? Promise.resolve(null) : fetchCursor();
      // 无游标路径的 entries 先拉（不依赖游标结果）
      const defaultEntriesPromise = isRebuild
        ? api.getEntries(sessionId, {})
        : api.getEntries(sessionId, { limit: HISTORY_PAGE_SIZE });

      const savedAnchor = await cursorPromise;
      if (cancelled) return;
      setInitialScrollAnchor(savedAnchor);
      setCursorReady(true); // 游标判定完成（有锚或无锚都算完成）
      const anchorEntryId = savedAnchor;

      try {
        let resp: Awaited<ReturnType<typeof api.getEntries>>;
        if (anchorEntryId && !isRebuild) {
          // ② 游标中心懒加载窗口——**顺序拉取**保证游标 entry 包含在窗口中：
          //   step1: before=游标前 N/2（strictly before，不含游标）
          //   step2: since=step1 最后一条的 id → 首条即游标 entry + 后续 N/2
          //   （此前 since+limit 返回最新 N 而非最近 N → 游标 entry 从未进窗口 →
          //     anchorToIndex found:false → scrollToIndex 不触发 → 定位失效）
          const WINDOW = Math.max(20, Math.ceil(window.innerHeight / 48) * 3);
          const half = Math.ceil(WINDOW / 2);
          const beforeResp = await api.getEntries(sessionId, { before: anchorEntryId, limit: half });
          if (cancelled) return;
          const beforeEntries = beforeResp.entries as unknown as Array<Record<string, unknown>>;
          const lastBefore = beforeEntries[beforeEntries.length - 1] as { id?: string } | undefined;
          const sinceResp = await api.getEntries(sessionId, { since: lastBefore?.id ?? anchorEntryId });
          if (cancelled) return;
          // since 结果：首条 = 游标 entry 本身，后续 = 游标后的条目
          // 截取 half+1 条（游标 + 后续 half 条）
          const sinceEntries = (sinceResp.entries as unknown as Array<Record<string, unknown>>).slice(0, half + 1);
          const merged = [...beforeEntries, ...sinceEntries];
          resp = { entries: merged, leaf_id: sinceResp.leaf_id } as typeof resp;
        } else {
          // 无游标：默认路径（最近 N 或全量）
          resp = await defaultEntriesPromise;
        }
        if (cancelled) return;
        const entriesList = resp.entries as unknown as Array<Record<string, unknown>>;

        const firstBuilt = buildHistoryItems(
          entriesList as unknown as Parameters<typeof buildHistoryItems>[0],
        );
        setHistory(firstBuilt);
        // 分页游标 + hasMore（窗口模式可向上翻页加载更早；默认模式 auto-compensation）
        let earliestId = (entriesList[0] as { id?: string } | undefined)?.id ?? null;
        let hasMore = !isRebuild && entriesList.length >= HISTORY_PAGE_SIZE;
        setPaging({ earliestId, hasMore });
        if (isRebuild || !hasMore || !earliestId) {
          if (!cancelled) hadHistoryRef.current = isRebuild || !anchorEntryId;
          return;
        }

        // 默认模式 auto-compensation（游标窗口模式跳过——真懒加载）
        if (hasMore && earliestId && !cancelled && !anchorEntryId) {
          loadingEarlierRef.current = true;
          setLoadingEarlier(true);
          try {
            let loaded = entriesList.length;
            while (hasMore && earliestId && !cancelled && loaded < MAX_HISTORY) {
              const pageResp = await api.getEntries(sessionId, {
                before: earliestId,
                limit: HISTORY_PAGE_SIZE,
              });
              if (cancelled) return;
              const pageEntries = pageResp.entries as unknown as Parameters<
                typeof buildHistoryItems
              >[0];
              if (pageEntries.length === 0) { hasMore = false; break; }
              const built = buildHistoryItems(pageEntries);
              prependHistory(built);
              loaded += pageEntries.length;
              earliestId = (pageEntries[0] as { id?: string } | undefined)?.id ?? null;
              hasMore = pageEntries.length === HISTORY_PAGE_SIZE;
            }
            if (!cancelled) setPaging({ earliestId, hasMore });
            if (!cancelled) hadHistoryRef.current = true;
          } catch { /* 保留已渲染部分 */ } finally {
            loadingEarlierRef.current = false;
            if (!cancelled) setLoadingEarlier(false);
          }
        }
      } catch {
        if (cancelled) return;
        setHistory({ items: [], fingerprints: new Set(), meta: { model: null, provider: null, thinkingLevel: null, startedAt: null } });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [sessionId, resyncCount, historyRefresh, prependHistory]);

  // 视图模型（纯函数重建——环形缓冲 ≤500 条，成本可控；版本号触发重算）
  // 历史指纹传入：live 流与历史重叠的事件去重（挂载时 REST 快照 vs SSE 在途事件）
  const vm = useMemo(
    () => buildChatViewModel(envelopes, null, history?.fingerprints),
    [envelopes, history],
  );

  // 合并视图：history（前置）→ live（SSE 重放，按自带 seq 顺序）→ 乐观用户消息。
  //
  // 顺序语义（严格 timeline）：乐观消息按其**发送时刻的会话最大 seq**（anchorSeq）
  // 插入——seq ≤ anchor 的 live 项（发送前产生，含上一条 AI 回复尚未落盘的尾巴）
  // 在其前，seq > anchor 的在其后（发送后才产生的回复）。
  // 历史旧实现踩过两个坑：
  //   ① 插在 live 尾部 → echo 到达前助手内容先到，用户消息被压到回复下方；
  //   ② 插在 history 与 live 之间 → 上一轮回复尾巴仍在 live 时，新消息被插到它前面
  //      （用户实测「我说的话排在上一句话下面，未按 timeline」）。anchorSeq 同时解决两者。
  const items = useMemo(() => {
    const live = vm.items;
    if (!optimisticUser) {
      return history ? [...history.items, ...live] : [...live];
    }
    const anchor = optAnchorRef.current;
    const before: typeof live = [];
    const after: typeof live = [];
    for (const it of live) {
      if ((it.seq ?? 0) <= anchor) before.push(it);
      else after.push(it);
    }
    const opt = [
      { kind: "user" as const, id: "optimistic-user", text: optimisticUser, seq: anchor + 1 },
    ];
    const merged = [...before, ...opt, ...after];
    return history ? [...history.items, ...merged] : merged;
  }, [history, vm.items, optimisticUser]);

  // 乐观消息 echo 消解：只检查 **live 事件流**（vm.items）——SSE 真实收到新 user 事件才消解。
  // 不能检查合并后 items（含 history）：同一会话重复发送同文本时，历史里的旧条目会误消解
  // 新发送的乐观消息 → 「消息消失」（用户实测反馈）。vm.items 里的 user 条目由
  // applyMessageStart(role=user) 生成（skip 检查在 vm 内），代表真实到达的 echo。
  const echoed = optimisticEchoed(vm.items, optimisticUser);
  useEffect(() => {
    if (echoed && optimisticUser) setOptimisticUser(null);
  }, [echoed, optimisticUser]);

  // 会话状态：SSE 状态事件 > REST 元数据
  const status: SessionStatus = (sseState?.status as SessionStatus) ?? info?.status ?? "active";
  // 流式状态：**服务端权威优先**（session_state.busy / GET session.busy）——
  // 刷新/重连后事件重放窗口可能丢 agent_start（回合中途刷新），客户端推断
  // 会错误回到 idle（发送态）；server busy 与实际 worker 状态一致。
  const streaming = sseState?.busy ?? info?.busy ?? vm.streaming;
  const phase: SessionPhase = useMemo(() => {
    if (!info && !sseState && history === null) return "loading";
    if (status === "closed") return "closed";
    if (status === "error") return "error";
    // 平台升级/重启后挂起：**不自动恢复**（server 权威 status），输入区换成
    // 「恢复继续」入口。必须排在 streaming 判定之前——挂起时 busy 必为 false，
    // 但若列为 idle 就会出现「能输入但发不出去」的假可用态。
    if (status === "suspended") return "suspended";
    return streaming ? "streaming" : "idle";
  }, [info, sseState, history, status, streaming]);

  // ============================================================
  // 动作
  // ============================================================

  const send = useCallback(
    async (message: string) => {
      setBusy(true);
      setError(null);
      // 时序锚点：发送瞬间的会话最大 seq（此后到达的事件都是本次回复）
      const buf = useSessionEventsStore.getState().buffers.get(sessionId) ?? [];
      optAnchorRef.current = buf.length > 0 ? buf[buf.length - 1].seq : 0;
      setOptimisticUser(message);
      try {
        if (phase === "streaming") {
          await api.steer(sessionId, message);
        } else {
          await api.prompt(sessionId, message);
        }
      } catch (err) {
        setOptimisticUser(null);
        // 用户反馈：非活跃会话发送报 409（state_conflict/worker not alive）——
        // 本地立即降级并引导恢复，而非停留在 active 反复报错。
        if (err instanceof ApiError && err.status === 409) {
          // 挂起会话（平台升级）与「异常中断」文案/态不同——别把它们混为一谈。
          if (status === "suspended") {
            applyState(sessionId, "suspended", "suspended; continue requires human confirmation");
            setError("会话因平台升级挂起——点击「▶ 恢复继续」重新拉起（不会自动续跑）");
          } else {
            applyState(sessionId, "error", "worker lost");
            setError("会话已中断（agent 进程不在）——点击 Resume 重新加载历史并恢复");
          }
        } else {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        setBusy(false);
      }
    },
    [phase, sessionId, applyState, status],
  );

  // 历史分页：滚动到顶/顶部按钮 → 用最早已加载 entry 作 before 拉更早一页，前置合并
  const loadEarlier = useCallback(async () => {
    if (loadingEarlierRef.current || !history || !paging.earliestId || !paging.hasMore) return;
    loadingEarlierRef.current = true;
    setLoadingEarlier(true);
    try {
      const resp = await api.getEntries(sessionId, {
        before: paging.earliestId,
        limit: HISTORY_PAGE_SIZE,
      });
      const built = buildHistoryItems(
        resp.entries as unknown as Parameters<typeof buildHistoryItems>[0],
      );
      if (built.items.length === 0) {
        setPaging((p0) => ({ ...p0, hasMore: false }));
        return;
      }
      const firstId = (resp.entries[0] as { id?: string } | undefined)?.id ?? null;
      prependHistory(built);
      setPaging({ earliestId: firstId, hasMore: resp.entries.length === HISTORY_PAGE_SIZE });
    } catch {
      // 更早历史拉取失败：保留现状（不阻塞聊天），hasMore 保持可重试
    } finally {
      loadingEarlierRef.current = false;
      setLoadingEarlier(false);
    }
  }, [sessionId, history, paging.earliestId, paging.hasMore, prependHistory]);

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
        } else if (cmd.name === "/model") {
          if (cmd.arg) {
            // /model <name>：模糊匹配模型名，唯一则直接切，多则打开选择
            const res = await api.getSessionModels(sessionId).catch(() => null);
            const hits = (res?.models ?? []).filter(
              (m) =>
                m.name.toLowerCase().includes(cmd.arg!.toLowerCase()) ||
                m.id.toLowerCase().includes(cmd.arg!.toLowerCase()),
            );
            if (res === null || hits.length === 0) {
              setModelHint(`未找到匹配「${cmd.arg}」的模型——已打开选择列表`);
              setModelDialogOpen(true);
            } else if (hits.length === 1) {
              await api.setSessionModel(sessionId, hits[0].provider, hits[0].id);
              setModelHint(`已切换至 ${hits[0].name}`);
            } else {
              setModelHint(`匹配到 ${hits.length} 个模型（${hits.map((m) => m.name).join(", ")}）——请选择`);
              setModelDialogOpen(true);
            }
          } else {
            setModelDialogOpen(true);
          }
        } else if (cmd.name === "/compact") {
          // 手动压缩：后端转发 pi 的 compact 命令；「压缩中」状态由既有
          // compaction_start/end 事件驱动（vm.compacting，占位卡可见——发起提示
          // 已删，迭代 3 human 反馈：占位卡已可见，冗余）。
          // 迭代 3：距上次成功压缩无新对话 → 不再发起（免无谓 resume worker
          // + pi 调用，pi 会报 Already compacted）。
          if (hasDialogueSinceLastCompaction(items)) {
            await api.compactSession(sessionId, cmd.arg || undefined);
          } else {
            setModelHint("上次压缩后无新对话，无需压缩");
          }
        } else if (cmd.name === "/thinking") {
          if (cmd.arg) {
            await api.setSessionThinking(sessionId, cmd.arg);
            setModelHint(`思考档位已设为 ${cmd.arg}`);
          } else {
            setModelDialogOpen(true);
          }
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        // 迭代 3：pi 端压缩错误中文映射（边缘场景：历史分页窗口未含压缩卡时
        // 前置拦截未触发，由后端 pi 错误兕底——映射为可读提示而非报错条）。
        if (msg.includes("Already compacted")) {
          setModelHint("上次压缩后无新对话，无需再压缩");
        } else if (msg.includes("Nothing to compact")) {
          setModelHint("会话太短，暂无可压缩内容");
        } else {
          setError(msg);
        }
      } finally {
        setBusy(false);
      }
    },
    [applyState, sessionId, items],
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

  // 人工确认继续（/continue）：与 resume 分开——doing/后台 dream 走归一化续跑，
  // 交互型等价 resume；202 already_active 同样视为「已在跑」以服务端为准刷新。
  const continueRun = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await api.continueSession(sessionId);
      applyState(sessionId, "active");
      setHistoryRefresh((n) => n + 1); // worker 可能已推进（或 doing 已续跑）
      const normalized = res.normalized_tasks ?? [];
      if (normalized.length > 0) {
        setModelHint(`已继续：归一化 ${normalized.length} 个遗留 running task（→ pending）并重跑剩余 task`);
      }
    } catch (err) {
      // 409 = 服务端说会话本就 active（本地列表状态陈旧）——以服务端为准刷新。
      if (err instanceof ApiError && err.status === 409) {
        applyState(sessionId, "active");
        setHistoryRefresh((n) => n + 1);
      } else {
        setError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      setBusy(false);
    }
  }, [applyState, sessionId]);

  const resume = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      await api.resumeSession(sessionId);
      applyState(sessionId, "active");
      setHistoryRefresh((n) => n + 1); // resume 后重拉历史（worker 可能已推进）
    } catch (err) {
      // 409 = 服务端说会话本就 active（本地列表状态陈旧）——以服务端为准刷新
      // 显示，而不是抛错（旧行为：报「nothing to resume」，UI 仍是 Resume 态）。
      if (err instanceof ApiError && err.status === 409) {
        applyState(sessionId, "active");
        setHistoryRefresh((n) => n + 1);
      } else {
        setError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      setBusy(false);
    }
  }, [applyState, sessionId]);

  // 右侧文件阅读器已提升到 App 壳层（job_43 增量 5）：openFile/readerApi/
  // Esc 处理/Panel 挂载全部移至 App.tsx；本组件仅消费（Markdown 内 LocalAwareLink
  // 经 useFileReader 拿到全局 reader，行为不变）。

  const unarchive = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      await api.unarchiveSession(sessionId);
      // 以服务端为准刷新该工作区列表（会话回到默认列表）
      const ws = useSessionsStore.getState().index.get(sessionId)?.workspace_id;
      if (ws) await useSessionsStore.getState().load(ws);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, [sessionId]);

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

  const sessionType: SessionType = info?.type ?? "easy";
  // Bug #5 迭代5b（human 反馈）：会话页头部与会话列表共用显示名派生（任务名优先）
  const display = info ? deriveSessionDisplay(info) : null;
  const title = display?.title ?? sessionId.slice(0, 8);

  // Bug #8 2C（human 2026-11-14）：token/成本/上下文使用率常驻显示——不展开会话
  // 信息就能看到（判断是否要 compact）。active 会话轮询 30s + turn_end 后刷新。
  const [stats, setStats] = useState<SessionStats | null>(null);
  const refreshStats = useCallback(() => {
    void api
      .getSessionStats(sessionId)
      .then((r) => setStats(r.stats))
      .catch(() => {});
  }, [sessionId]);
  useEffect(() => {
    if (!info || info.status !== "active") return;
    refreshStats();
    const t = setInterval(refreshStats, 30_000);
    return () => clearInterval(t);
  }, [info?.status, refreshStats]);

  // Bug #8 2C（human 2026-11-14）：上下文使用率超阈值（默认 80%，Settings 可调）
  // 自动触发压缩——一次触发后等下一轮 stats 再评估（防抖：压缩中/本轮已触发不重复）。
  // 阈值配置提升组件级（头部徽标警示色 + 自动压缩共用同一读数）。
  const [autoCompactCfg, setAutoCompactCfg] = useState({ enabled: true, threshold: 80 });
  useEffect(() => {
    try {
      const raw = localStorage.getItem("rick-web:auto-compact");
      if (raw) {
        const v = JSON.parse(raw);
        if (v && typeof v.enabled === "boolean") {
          setAutoCompactCfg({ enabled: v.enabled, threshold: typeof v.threshold === "number" ? v.threshold : 80 });
        }
      }
    } catch {
      // 默认
    }
  }, []);
  const autoCompactThreshold = autoCompactCfg.threshold;
  const autoCompactRef = useRef(0);
  useEffect(() => {
    const pct = stats?.contextUsage?.percent;
    if (pct == null || !info || info.status !== "active") return;
    if (!autoCompactCfg.enabled || pct < autoCompactCfg.threshold) return;
    if (vm.compacting) return;
    if (autoCompactRef.current === pct) return; // 同一水位只触发一次
    autoCompactRef.current = pct;
    void api
      .compactSession(sessionId, "自动压缩：上下文使用率已达 " + pct + "%")
      .then(() => refreshStats())
      .catch(() => {});
  }, [stats?.contextUsage?.percent, vm.compacting, info?.status, sessionId, refreshStats, info]);

  // Bug #8 2A 终版（human 裁决 2026-11-15）：subagent 子会话右侧栏（只读——输入走
  // 主页面）。卡片/侧边栏点击 → 打开右侧栏展示完整子会话输出。侧边栏点击经
  // ui store pendingSubagent 传递（跨组件），挂载时消费。
  const [openSubagent, setOpenSubagent] = useState<SubagentTarget | null>(() =>
    useUiStore.getState().consumePendingSubagent(),
  );
  // 已在会话页时侧边栏点击（组件不重挂，useState 初始不重跑）——订阅式消费：
  // pendingSubagent 一旦被设置立即打开并清除。
  const pendingSubagent = useUiStore((s) => s.pendingSubagent);
  useEffect(() => {
    if (pendingSubagent) {
      setOpenSubagent(pendingSubagent);
      useUiStore.getState().consumePendingSubagent();
    }
  }, [pendingSubagent]);

  return (
    <div className="flex h-full min-h-0 w-full">
    <div className="relative flex h-full min-h-0 min-w-0 flex-1 flex-col">
      {/* 会话头：移动端不透明底（job_43 去.blur 省电）；md: 恢复桌面毛玻璃 */}
      <header className="flex shrink-0 items-center gap-2.5 border-b border-line bg-space px-4 py-2.5 md:bg-space/70 md:backdrop-blur">
        <Portal size={22} spin />
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
          {vm.streaming ? "streaming" : STATUS_LABEL[status] ?? status}
        </span>
        {/* Bug #8 2B（human 2026-11-14 实测要求）：当前模型名常驻显示（此前只在折叠的
            会话信息区可见——运行时看不到当前用的什么模型）。meta 来自会话 JSONL 的
            model_change（含初始默认模型，实测首条即 glm-5.3）。 */}
        {history?.meta.model && (
          <span
            className="shrink-0 rounded border border-line bg-space-2/60 px-1.5 py-0.5 font-mono text-[10px] text-ink-2"
            title={`当前模型：${history?.meta.model}${history?.meta.provider ? `（${history?.meta.provider}）` : ""}`}
          >
            {history?.meta.model}
          </span>
        )}
        {/* Bug #8 2C：上下文使用率 + token/成本常驻（≥80% 警示色——该 compact 了）。
            迭代（human 2026-11-15）：显示模型上下文窗口（如 1M）+ 占比两位小数。 */}
        {stats?.contextUsage?.percent != null && (() => {
          const ctx = stats.contextUsage;
          // 两位小数（pi 的 percent 是整数——自己按 tokens/contextWindow 算精确占比）
          const used = ctx.tokens ?? 0;
          const win = ctx.contextWindow || 1;
          const pct2 = Math.round((used / win) * 10000) / 100;
          // 窗口人类可读（1M / 256k / 131k …）
          const fmtWin = (n: number): string => {
            if (n >= 1_000_000 && n % 1_000_000 === 0) return `${n / 1_000_000}M`;
            if (n >= 1_000) return `${Math.round(n / 1000)}k`;
            return String(n);
          };
          const high = pct2 >= (autoCompactThreshold ?? 80);
          return (
            <span
              className={`shrink-0 rounded border px-1.5 py-0.5 font-mono text-[10px] ${
                high ? "border-morty/60 bg-morty/10 text-morty" : "border-line bg-space-2/60 text-ink-2"
              }`}
              title={`上下文使用：${used.toLocaleString()} / ${win.toLocaleString()} token（${pct2}%，窗口 ${fmtWin(win)}）——超阈值建议压缩`}
            >
              ctx {pct2.toFixed(2)}% / {fmtWin(win)}
            </span>
          );
        })()}
        {stats && (
          <span
            className="shrink-0 rounded border border-line bg-space-2/60 px-1.5 py-0.5 font-mono text-[10px] text-ink-3"
            title={`累计 token：${stats.tokens.total.toLocaleString()}（含 subagent 与压缩生成）；成本 $${stats.cost.toFixed(2)}`}
          >
            {stats.tokens.total >= 10000
              ? `${Math.round(stats.tokens.total / 1000)}k tok`
              : `${stats.tokens.total} tok`}
            {stats.cost > 0 ? ` · $${stats.cost.toFixed(2)}` : ""}
          </span>
        )}
        {vm.compacting && <span className="text-xs text-morty">压缩中</span>}
        {/* 归档状态可见 + 一键取消归档：归档是「从列表隐藏」，会话页本身不会变化——
            若不在页面上明示，用户会误以为「点了归档没生效」（实测反馈）。 */}
        {info?.archived && (
          <span
            className="flex shrink-0 items-center gap-1 rounded-full border border-ink-3/40 bg-ink-3/10 px-1.5 py-0.5 text-[10px] text-ink-3"
            title="该会话已归档（不在工作区默认列表显示）"
          >
            📦 已归档
          </span>
        )}
        <div className="ml-auto flex items-center gap-1">
          {info?.archived && (
            <button
              type="button"
              onClick={() => void unarchive()}
              disabled={busy}
              title="取消归档（回到工作区默认列表）"
              className="rounded-md border border-line px-2 py-1 text-xs text-portal hover:border-portal/50 hover:bg-portal/10 disabled:opacity-40"
            >
              取消归档
            </button>
          )}
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

      {/* 会话信息（系统提示词来源/参数/模型——默认折叠，复查行为轨迹用） */}
      <SessionInfoPanel
        info={info ?? null}
        meta={history?.meta ?? { model: null, provider: null, thinkingLevel: null, startedAt: null }}
        workspaceId={info?.workspace_id ?? null}
        onSwitchModel={() => setModelDialogOpen(true)}
      />

      {/* 模型切换提示（/model 快捷命令反馈） */}
      {modelHint && (
        <div className="flex items-center gap-2 border-b border-portal/30 bg-portal/10 px-4 py-1.5 text-xs text-portal">
          <span aria-hidden="true">⚡</span>
          <span className="min-w-0 flex-1 break-words">{modelHint}</span>
          <button type="button" onClick={() => setModelHint(null)} aria-label="关闭提示">
            ✕
          </button>
        </div>
      )}

      {/* 模型切换 Dialog */}
      <ModelSwitchDialog
        open={modelDialogOpen}
        sessionId={sessionId}
        currentModel={history?.meta.model ?? null}
        currentProvider={history?.meta.provider ?? null}
        currentThinking={history?.meta.thinkingLevel ?? null}
        onClose={() => setModelDialogOpen(false)}
        onChanged={() => setHistoryRefresh((n) => n + 1)}
      />

      {/* fire-and-forget notify toasts */}
      <NotifyToasts sessionId={sessionId} notifications={vm.notifications} />

      {/* 消息流 */}
      <div className="min-h-0 flex-1">
        {/* 游标判定完成前显示 skeleton（消除底闪→跳中间的视觉闪烁） */}
        {!cursorReady ? (
          <div className="flex flex-1 items-center justify-center py-8 text-sm text-ink-3">
            <span className="mr-2 inline-block h-2 w-2 animate-rm-pulse rounded-full bg-portal" />
            恢复会话位置…
          </div>
        ) : (
        <MessageList
          items={items}
          streaming={vm.streaming}
          hasMore={paging.hasMore}
          loadingEarlier={loadingEarlier}
          onReachTop={() => void loadEarlier()}
          onOpenSubagent={(t: SubagentTarget) => setOpenSubagent(t)}
          onViewportAnchor={handleViewportAnchor}
          initialAnchor={initialScrollAnchor}
          onAnchorConsumed={() => setInitialScrollAnchor(null)}
        />
        )}
      </div>

      {/* extension_ui 对话框（agent 阻塞等待——渲染在输入条上方） */}
      <ExtensionUIDialog
        sessionId={sessionId}
        requests={vm.pendingDialogs}
        onResponse={(err) => {
          if (err) setError(err);
        }}
      />

      {/* 底部操作条：移动端不透明底（job_43 去.blur 省电）；md: 恢复桌面毛玻璃 */}
      <div className="shrink-0 border-t border-line bg-space md:bg-space/80 md:backdrop-blur">
        <SteerBar
          phase={phase}
          onSend={(m) => void send(m)}
          onCommand={(c) => void runCommand(c)}
          onAbort={() => void abort()}
          onResume={() => void resume()}
          onContinue={() => void continueRun()}
          busy={busy}
          compacting={vm.compacting}
          sessionId={sessionId}
        />
      </div>
    </div>
    {/* Bug #8 2A 终版：subagent 子会话右侧栏（只读，点击卡片/侧边栏打开） */}
    {openSubagent && (
      <SubagentPanel
        target={openSubagent}
        progress={
          // 最近匹配的 subagent 卡进度（同 runId 的卡）
          (() => {
            const card = vm.items.find(
              (i): i is Extract<ChatItem, { kind: "tool" }> =>
                i.kind === "tool" &&
                i.toolName === "subagent" &&
                i.subagentProgress?.runId === openSubagent.runId,
            );
            return card?.subagentProgress;
          })()
        }
        onClose={() => setOpenSubagent(null)}
      />
    )}
    </div>
  );
}
