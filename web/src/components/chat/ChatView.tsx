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
import type { SessionStatus, SessionPrompt, SessionType } from "../../types";
import { ApiError } from "../../types";
import MessageList from "./MessageList";
import SteerBar, { type SessionPhase } from "./SteerBar";
import ExtensionUIDialog, { NotifyToasts } from "../extui/ExtensionUIDialog";
import {
  buildChatViewModel,
  buildHistoryItems,
  optimisticEchoed,
  type HistoryBuildResult,
} from "./viewModel";
import type { SlashCommand } from "./ChatInput";
import Portal from "../starfield/Portal";
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
// 会话信息折叠区（系统提示词来源 + 元信息 + 模型/思考档位）
// ============================================================

/** 会话类型的 prompt 文件来源（显示路径提示——内容经 job files API 查看） */
const PROMPT_SOURCE: Record<SessionType, string> = {
  plan: "<job>/plan/prompts/plan_prompt.md（+ method 系统提示词，--append-system-prompt 注入）",
  easy: "<job>/doing/prompts/easy_main_prompt.md（+ method 系统提示词）",
  ctrl: "<job>/doing/prompts/ctrl_prompt.md",
  "human-loop": "<draft>/loops/loop_N/prompts/*.md（SENSE 四文件协议）",
  learning: "<job>/learning/prompts/learning_prompt.md",
  dream: ".rick/dream/（跨 job 反思，扫描已完成 jobs）",
  doing: "<job>/doing/prompts/doing_prompt.md（parent 编排协议）",
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
  const type = info?.type ?? "plan";
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
          {PROMPT_SOURCE[type]}
        </p>
        <button
          type="button"
          onClick={() => void loadPrompt()}
          disabled={!workspaceId || promptLoading}
          className="mb-2 rounded border border-line px-2 py-0.5 text-[10px] text-portal hover:border-portal/50 hover:bg-portal/10 disabled:opacity-50"
        >
          {promptLoading ? "加载中…" : prompt ? "收起原文" : "查看原文"}
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
  useEffect(() => {
    let cancelled = false;
    const epoch = resyncCount + historyRefresh;
    if (historyEpochRef.current === epoch) return;
    historyEpochRef.current = epoch;
    const isRebuild = hadHistoryRef.current;
    api
      .getEntries(sessionId, isRebuild ? {} : { limit: HISTORY_PAGE_SIZE })
      .then(async (resp) => {
        if (cancelled) return;
        const firstBuilt = buildHistoryItems(
          resp.entries as unknown as Parameters<typeof buildHistoryItems>[0],
        );
        setHistory(firstBuilt);
        // 分页游标：entries 时间序，第一条=最早已加载；满页=可能还有更早
        let earliestId = (resp.entries[0] as { id?: string } | undefined)?.id ?? null;
        let hasMore =
          !isRebuild && resp.entries.length === HISTORY_PAGE_SIZE;
        setPaging({ earliestId, hasMore });
        if (isRebuild || !hasMore || !earliestId) {
          // 重建（全量已拿齐）或首次拉到底：历史已完整
          if (!cancelled) hadHistoryRef.current = true;
          return;
        }

        // A：挂载自动补拉（方案 A——用户实测 254 条会话首屏只见最近 50 条，
        // 更早历史要手动滚动才加载，体验不完整）。首屏已渲染，随后循环用
        // before 游标拉更早页并前置合并，直到：拉到底（不满页）/ 达 MAX_HISTORY
        // 上限 / 组件卸载。补拉不阻塞首屏；与 loadEarlier 共用 loadingEarlierRef 防重入。
        if (hasMore && earliestId && !cancelled) {
          loadingEarlierRef.current = true;
          setLoadingEarlier(true);
          try {
            let loaded = resp.entries.length;
            while (hasMore && earliestId && !cancelled && loaded < MAX_HISTORY) {
              const pageResp = await api.getEntries(sessionId, {
                before: earliestId,
                limit: HISTORY_PAGE_SIZE,
              });
              if (cancelled) return;
              const pageEntries = pageResp.entries as unknown as Parameters<
                typeof buildHistoryItems
              >[0];
              if (pageEntries.length === 0) {
                hasMore = false;
                break;
              }
              const built = buildHistoryItems(pageEntries);
              prependHistory(built);
              loaded += pageEntries.length;
              earliestId = (pageEntries[0] as { id?: string } | undefined)?.id ?? null;
              hasMore = pageEntries.length === HISTORY_PAGE_SIZE;
            }
            if (!cancelled) setPaging({ earliestId, hasMore });
            if (!cancelled) hadHistoryRef.current = true; // 首次补拉完整
          } catch {
            // 自动补拉失败：保留已渲染部分（不阻塞 live 流），hasMore 保持可重试
          } finally {
            loadingEarlierRef.current = false;
            if (!cancelled) setLoadingEarlier(false);
          }
        }
      })
      .catch(() => {
        if (cancelled) return;
        setHistory({ items: [], fingerprints: new Set(), meta: { model: null, provider: null, thinkingLevel: null, startedAt: null } }); // 历史拉取失败不阻塞 live 流
      });
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
        // 本地立即降级为 error（终止）态并引导 Resume，而非停留在 active 反复报错。
        if (err instanceof ApiError && err.status === 409) {
          applyState(sessionId, "error", "worker lost");
          setError("会话已中断（agent 进程不在）——点击 Resume 重新加载历史并恢复");
        } else {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        setBusy(false);
      }
    },
    [phase, sessionId, applyState],
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
        } else if (cmd.name === "/thinking") {
          if (cmd.arg) {
            await api.setSessionThinking(sessionId, cmd.arg);
            setModelHint(`思考档位已设为 ${cmd.arg}`);
          } else {
            setModelDialogOpen(true);
          }
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
        <MessageList
          items={items}
          streaming={vm.streaming}
          hasMore={paging.hasMore}
          loadingEarlier={loadingEarlier}
          onReachTop={() => void loadEarlier()}
        />
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
