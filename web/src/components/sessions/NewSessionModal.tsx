/**
 * NewSessionModal：新建会话弹窗（cmd 参数流）。
 *
 * 三步：选工作区（下拉）→ 选 cmd 类型（卡片组）→ 动态参数表单 → 提交。
 * 参数面（api-contract Sessions 节）：
 * - plan:       requirement（多行必填）+ job 可选（复用已有 plan 目录）
 * - easy:       requirement + ctx_path 可选
 * - ctrl/learning/doing: job（下拉，取该工作区 jobs 列表）
 * - human-loop: topic
 * - dream:      job_num（数字，默认 5）+ mode（interactive/background 单选）
 * 必填缺失时禁用提交；提交成功跳转 /session/:id。
 */

import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useSessionsStore } from "../../stores/sessions";
import { useWorkspacesStore } from "../../stores/workspaces";
import { api } from "../../api/client";
import { ApiError } from "../../types";
import type {
  CreateSessionRequest,
  DreamParams,
  JobSummary,
  SessionType,
} from "../../types";
import Button from "../common/Button";
import Dialog from "../common/Dialog";
import ErrorBanner from "../common/ErrorBanner";
import Spinner from "../common/Spinner";
import Saucer from "../starfield/Saucer";


// ============================================================
// cmd 类型卡片（图标=内联 SVG，一句话说明）
// ============================================================

interface CmdMeta {
  type: SessionType;
  label: string;
  desc: string;
  icon: (active: boolean) => React.ReactNode;
}

const ICON_CLS = (active: boolean) =>
  `h-6 w-6 shrink-0 ${active ? "text-portal" : "text-ink-3"}`;

const CMD_META: CmdMeta[] = [
  {
    type: "plan",
    label: "plan",
    desc: "需求 → 开发计划（job 分解 + task.md 产出）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <rect x="4" y="3" width="16" height="18" rx="2" stroke="currentColor" strokeWidth="1.6" />
        <path d="M8 8h8M8 12h8M8 16h5" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    type: "easy",
    label: "easy",
    desc: "轻量直通执行（单需求一把梭）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <path d="M13 2 4 14h6l-1 8 9-12h-6l1-8z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
      </svg>
    ),
  },
  {
    type: "doing",
    label: "doing",
    desc: "门禁化执行 job（监控型：task 看板 + 事件流）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="1.6" />
        <path d="M12 7v5l3.5 2" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    type: "ctrl",
    label: "ctrl",
    desc: "干预运行中的 job（追加指令/重置 task）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <path d="M5 5v14l6-7-6-7zM13 5v14l6-7-6-7z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
      </svg>
    ),
  },
  {
    type: "human-loop",
    label: "human-loop",
    desc: "SENSE 五阶段深度思考（产出 RFC）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <path d="M12 3a6 6 0 0 1 6 6c0 2.5-1.5 3.7-2.5 5-.8 1-1.5 2.2-1.5 4h-4c0-1.8-.7-3-1.5-4C7.5 12.7 6 11.5 6 9a6 6 0 0 1 6-6z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
        <path d="M10 21h4" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    type: "learning",
    label: "learning",
    desc: "单 job 知识沉淀（SUMMARY + skills/loops）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <path d="M4 19V5a2 2 0 0 1 2-2h13v14H6a2 2 0 0 0-2 2zm0 0a2 2 0 0 0 2 2h13" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
  },
  {
    type: "dream",
    label: "dream",
    desc: "跨 job 全局反思（交互式，服务端常驻）",
    icon: (a) => (
      <svg viewBox="0 0 24 24" fill="none" className={ICON_CLS(a)} aria-hidden="true">
        <path d="M20 14.5A8.5 8.5 0 0 1 9.5 4 8.5 8.5 0 1 0 20 14.5z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
        <path d="M17 4h4M19 2v4" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      </svg>
    ),
  },
];

const FIELD_CLS =
  "w-full rounded-lg border border-line bg-space/60 px-3 py-2 text-sm text-ink placeholder:text-ink-3 focus:border-portal/60 focus:outline-none";

// ============================================================
// 参数表单（按 type 动态）
// ============================================================

/** jobs 下拉选项加载（ctrl/learning/doing 需要）+ dream 素材预检（交互式 dream
 * 需要 workspace 里有“已完成且未被 dream 归档”的 job——提交前先看，避免 409 玄学）。 */
function useJobOptions(workspaceId: string | null, active: boolean): {
  jobs: Array<{ job_id: string; label: string }>;
  loading: boolean;
  /** 可 dream 素材数：已完成（tasks 全 success）且未被 dream 学习的 job
   *  （含 done/manual 归档来源——dream 会扫描文件系统，不受归档视图影响）。
   *  口径 = include_archived 全量中 completed && archived_by !== "dream"。 */
  dreamPending: number;
  /** 是否已完成一次检查（区分 loading 与空结果） */
  checked: boolean;
} {
  const [list, setList] = useState<JobSummary[] | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!active || !workspaceId) {
      setList(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    api
      .listJobs(workspaceId, true) // 全量（含归档）——job 下拉需含已完成；dream 素材需跨归档口径
      .then((all) => {
        if (!cancelled) setList(all);
      })
      .catch(() => {
        if (!cancelled) setList([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId, active]);

  const jobs = useMemo(() => {
    if (!list) return [];
    return list
      .slice()
      .sort((a, b) => (a.updated_at < b.updated_at ? 1 : -1))
      .map((j) => {
        const done = j.tasks.filter((t) => t.status === "success").length;
        const archivedMark = j.archived ? ` 📦` : "";
        return {
          job_id: j.job_id,
          label: `${j.job_id}（${done}/${j.tasks.length} 完成${archivedMark}）`,
        };
      });
  }, [list]);

  const dreamPending = useMemo(() => {
    if (!list) return 0;
    return list.filter(
      (j) =>
        j.tasks.length > 0 &&
        j.tasks.every((t) => t.status === "success") &&
        j.archived_by !== "dream",
    ).length;
  }, [list]);

  return { jobs, loading, dreamPending, checked: !loading };
}

interface NewSessionModalProps {
  open: boolean;
  onClose: () => void;
  /** 预选工作区（Sidebar 的「新建会话」在工作区上下文点出时传入） */
  presetWorkspaceId?: string | null;
  /** 预选类型（预留：从 Jobs 页对某 job 直接发起 doing/ctrl） */
  presetType?: SessionType | null;
  /** doing/ctrl/learning 预填 job */
  presetJob?: string | null;
}

export default function NewSessionModal({
  open,
  onClose,
  presetWorkspaceId = null,
  presetType = null,
  presetJob = null,
}: NewSessionModalProps) {
  const navigate = useNavigate();
  const workspaces = useWorkspacesStore((s) => s.list);
  const create = useSessionsStore((s) => s.create);

  const [workspaceId, setWorkspaceId] = useState<string | null>(presetWorkspaceId);
  const [cmdType, setCmdType] = useState<SessionType | null>(presetType);

  // plan
  const [requirement, setRequirement] = useState("");
  const [planJob, setPlanJob] = useState("");
  // easy
  const [ctxPath, setCtxPath] = useState("");
  // human-loop
  const [topic, setTopic] = useState("");
  // ctrl/learning/doing
  const [jobId, setJobId] = useState(presetJob ?? "");
  // dream
  const [jobNum, setJobNum] = useState(5);
  // 通用
  const [title, setTitle] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 弹窗开启时重置（预填项以 props 为准）
  useEffect(() => {
    if (open) {
      setWorkspaceId(presetWorkspaceId ?? null);
      setCmdType(presetType ?? null);
      setRequirement("");
      setPlanJob("");
      setCtxPath("");
      setTopic("");
      setJobId(presetJob ?? "");
      setJobNum(5);
      setTitle("");
      setError(null);
      setSubmitting(false);
    }
  }, [open, presetWorkspaceId, presetType, presetJob]);

  const jobActive = cmdType !== null && (needsJob(cmdType) || cmdType === "dream");
  const { jobs: jobOptions, loading: jobsLoading, dreamPending, checked: dreamChecked } =
    useJobOptions(jobActive ? workspaceId : null, jobActive);

  // ----------------------------------------------------------
  // 校验 + 提交
  // ----------------------------------------------------------

  const missing: string[] = [];
  if (!workspaceId) missing.push("工作区");
  if (!cmdType) missing.push("命令类型");
  if (cmdType === "plan" || cmdType === "easy") {
    if (!requirement.trim()) missing.push("需求描述");
  }
  if (cmdType === "human-loop" && !topic.trim()) missing.push("主题");
  if (needsJob(cmdType) && !jobId) missing.push("job");
  if (cmdType === "dream" && jobNum < 1) missing.push("job 数量");
  // dream 无可学习 job：明确阻塞（提示已给出，后端 409 兜底）
  const dreamBlocked = cmdType === "dream" && !!workspaceId && dreamChecked && dreamPending === 0;
  if (dreamBlocked) missing.push("可学习的 job");

  const canSubmit = missing.length === 0 && !submitting;

  async function handleSubmit(): Promise<void> {
    if (!cmdType || !workspaceId) return;
    setSubmitting(true);
    setError(null);
    try {
      const params = buildParams({
        type: cmdType,
        requirement,
        planJob,
        ctxPath,
        topic,
        jobId,
        jobNum,
      });
      const req: CreateSessionRequest = {
        workspace_id: workspaceId,
        type: cmdType,
        params,
        ...(title.trim() ? { title: title.trim() } : {}),
      };
      const info = await create(req);
      onClose();
      navigate(`/session/${info.id}`);
    } catch (err) {
      // 后端 409 no_pending_jobs（dream 无可学习素材）——给可行动引导而非裸英文报错
      if (err instanceof ApiError && err.code === "no_pending_jobs") {
        setError(
          "已无可学习的 job：该工作区所有已完成 job 都已被 dream 学习过（Jobs 页归档区可查）。\n" +
            "dream 只学习 tasks 全部完成的 job；中断/未完成的 job 不参与学习。\n" +
            "请先完成一个新的 job（plan/doing 会话跑完），或切换到还有其他已完成 job 的工作区。",
        );
      } else {
        setError(err instanceof Error ? err.message : String(err));
      }
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={
        <span className="flex items-center gap-2">
          <Saucer size={20} />
          新建会话
        </span>
      }
      contentClassName="w-[min(92vw,640px)]"
    >
      <div className="flex max-h-[70vh] flex-col gap-5 overflow-y-auto px-1 py-1">
        {/* ① 工作区 */}
        <label className="flex flex-col gap-1.5">
          <span className="text-xs font-medium uppercase tracking-wide text-ink-3">工作区</span>
          <select
            className={FIELD_CLS}
            value={workspaceId ?? ""}
            onChange={(e) => setWorkspaceId(e.target.value || null)}
          >
            <option value="">选择工作区…</option>
            {workspaces.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name || w.path}（{w.path}）
              </option>
            ))}
          </select>
        </label>

        {/* ② cmd 类型卡片组 */}
        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium uppercase tracking-wide text-ink-3">命令类型</span>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {CMD_META.map((meta) => {
              const active = cmdType === meta.type;
              return (
                <button
                  key={meta.type}
                  type="button"
                  onClick={() => setCmdType(meta.type)}
                  className={`flex items-start gap-2.5 rounded-lg border p-3 text-left transition-colors ${
                    active
                      ? "border-portal/60 bg-portal-soft"
                      : "border-line bg-space/40 hover:border-line hover:bg-white/5"
                  }`}
                >
                  {meta.icon(active)}
                  <span className="flex min-w-0 flex-col gap-0.5">
                    <span className={`text-sm font-semibold ${active ? "text-portal" : "text-ink"}`}>
                      {meta.label}
                    </span>
                    <span className="text-xs leading-snug text-ink-2">{meta.desc}</span>
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* ③ 动态参数表单 */}
        {cmdType && (
          <div className="flex flex-col gap-4 rounded-lg border border-line bg-space/40 p-4">
            <span className="text-xs font-medium uppercase tracking-wide text-ink-3">
              参数（{cmdType}）
            </span>

            {(cmdType === "plan" || cmdType === "easy") && (
              <label className="flex flex-col gap-1.5">
                <span className="text-sm text-ink-2">
                  需求描述 <span className="text-portal">*</span>
                </span>
                <textarea
                  className={`${FIELD_CLS} min-h-24 resize-y`}
                  placeholder="描述本次要完成的需求（plan：将分解为 jobs/tasks；easy：直接执行）"
                  value={requirement}
                  onChange={(e) => setRequirement(e.target.value)}
                />
              </label>
            )}

            {cmdType === "plan" && (
              <label className="flex flex-col gap-1.5">
                <span className="text-sm text-ink-2">复用已有 job（可选）</span>
                <input
                  className={FIELD_CLS}
                  placeholder="如 job_5——留空则创建新 job"
                  value={planJob}
                  onChange={(e) => setPlanJob(e.target.value)}
                />
              </label>
            )}

            {cmdType === "easy" && (
              <label className="flex flex-col gap-1.5">
                <span className="text-sm text-ink-2">上下文目录（可选）</span>
                <input
                  className={FIELD_CLS}
                  placeholder="ctx 继承的工作区路径（留空用当前工作区）"
                  value={ctxPath}
                  onChange={(e) => setCtxPath(e.target.value)}
                />
              </label>
            )}

            {cmdType === "human-loop" && (
              <label className="flex flex-col gap-1.5">
                <span className="text-sm text-ink-2">
                  思考主题 <span className="text-portal">*</span>
                </span>
                <input
                  className={FIELD_CLS}
                  placeholder="要深度思考的问题（SENSE 五阶段）"
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                />
              </label>
            )}

            {needsJob(cmdType) && (
              <label className="flex flex-col gap-1.5">
                <span className="text-sm text-ink-2">
                  job <span className="text-portal">*</span>
                </span>
                {jobsLoading && jobOptions.length === 0 ? (
                  <span className="flex items-center gap-2 text-xs text-ink-3">
                    <Spinner size={14} /> 加载 jobs…
                  </span>
                ) : jobOptions.length === 0 ? (
                  <span className="text-xs text-ink-3">
                    该工作区暂无 job（先跑 plan 或手写 plan 目录）
                  </span>
                ) : (
                  <select
                    className={FIELD_CLS}
                    value={jobId}
                    onChange={(e) => setJobId(e.target.value)}
                  >
                    <option value="">选择 job…</option>
                    {jobOptions.map((j) => (
                      <option key={j.job_id} value={j.job_id}>
                        {j.label}
                      </option>
                    ))}
                  </select>
                )}
              </label>
            )}

            {cmdType === "dream" && (
              <>
                {/* 素材可用性预检：无素材时提交前就告知（避免裸 409）；交互/后台都需要素材 */}
                <div className="flex flex-col gap-1">
                  <span className="text-sm text-ink-2">可用素材</span>
                  {!workspaceId || !dreamChecked ? (
                    <p className="rounded-md border border-line bg-space/40 px-2.5 py-1.5 text-xs text-ink-3">
                      选择工作区后检查可 dream 的已完成 job…
                    </p>
                  ) : dreamPending === 0 ? (
                    <p className="rounded-md border border-morty/40 bg-morty/10 px-2.5 py-1.5 text-xs leading-relaxed text-ink-2">
                      ⚠ <b className="text-ink-2">已无可学习的 job</b>——该工作区所有已完成 job
                      都已被 dream 学习过（Jobs 页归档区可查）。dream 只学习 tasks 全部完成的
                      job；中断/未完成的 job 不参与学习。请先跑完一个 job 再来。
                    </p>
                  ) : (
                    <p className="rounded-md border border-portal/40 bg-portal/5 px-2.5 py-1.5 text-xs leading-relaxed text-portal">
                      ✓ 可学习素材：{dreamPending} 个已完成 job（dream 将取前 job_num 个反思）
                    </p>
                  )}
                </div>
                <label className="flex flex-col gap-1.5">
                  <span className="text-sm text-ink-2">处理数量（job_num）</span>
                  <input
                    type="number"
                    min={1}
                    max={50}
                    className={FIELD_CLS}
                    value={jobNum}
                    onChange={(e) => setJobNum(Number(e.target.value) || 1)}
                  />
                </label>
                <div className="flex flex-col gap-1">
                  <span className="text-sm text-ink-2">模式</span>
                  <p className="rounded-md border border-line bg-space/40 px-2.5 py-1.5 text-xs leading-relaxed text-ink-3">
                    <b className="text-ink-2">交互模式</b>（web UI 仅支持）——服务端常驻 rpc 会话，
                    刷新页面或关闭浏览器后 pi 仍在后台工作，状态由 server 维护。
                  </p>
                </div>
              </>
            )}

            <label className="flex flex-col gap-1.5">
              <span className="text-sm text-ink-2">会话标题（可选）</span>
              <input
                className={FIELD_CLS}
                placeholder="留空自动生成"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
            </label>
          </div>
        )}

        <ErrorBanner message={error} onDismiss={() => setError(null)} />

        {/* 提交 */}
        <div className="flex items-center justify-between gap-3">
          <span className="text-xs text-ink-3">
            {missing.length > 0 ? `待填：${missing.join("、")}` : "就绪"}
          </span>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={onClose} disabled={submitting}>
              取消
            </Button>
            <Button
              variant="primary"
              onClick={() => void handleSubmit()}
              loading={submitting}
              disabled={!canSubmit}
            >
              启动会话
            </Button>
          </div>
        </div>
      </div>
    </Dialog>
  );
}

// ============================================================
// 纯函数：类型 → 参数构造
// ============================================================

function needsJob(type: SessionType | null): boolean {
  return type === "ctrl" || type === "learning" || type === "doing";
}

function buildParams(input: {
  type: SessionType;
  requirement: string;
  planJob: string;
  ctxPath: string;
  topic: string;
  jobId: string;
  jobNum: number;
}): CreateSessionRequest["params"] {
  switch (input.type) {
    case "plan":
      return {
        requirement: input.requirement.trim(),
        ...(input.planJob.trim() ? { job: input.planJob.trim() } : {}),
      };
    case "easy":
      return {
        requirement: input.requirement.trim(),
        ...(input.ctxPath.trim() ? { ctx_path: input.ctxPath.trim() } : {}),
      };
    case "ctrl":
    case "learning":
    case "doing":
      return { job: input.jobId };
    case "human-loop":
      return { topic: input.topic.trim() };
    case "dream": {
      // web UI 只保留交互模式（后台模式移除：无进度反馈、与交互语义重复；
      // CLI 侧后台 dream 不受影响）。
      const params: DreamParams = {
        job_num: input.jobNum,
        mode: "interactive",
      };
      return params;
    }
  }
}
