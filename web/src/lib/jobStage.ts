/**
 * jobStage：job 阶段判定 + 「下一步该用哪个会话模式」的指引。
 *
 * 用途：新建会话（doing/ctrl/learning）选择 job 时，让用户一眼看到
 * 「这个 job 现在什么状态、下一步该干什么」——而不是只看进度数字
 * （用户反馈：选 job 时应标记状态并指引下一步模式）。
 *
 * 判定依据（全部来自服务端 job 列表）：
 * - stage=planned：plan 已产出 task*.md，但还没跑 doing
 * - 有 doing/tasks.json：按 task 状态细分（运行中 / 有失败 / 有待执行 / 全部完成）
 * - archived_by=dream：已被 dream 学习（知识沉淀过了）
 */
import type { JobSummary } from "../types";

export type JobPhase = "planned" | "running" | "failed" | "pending" | "done" | "dreamed";

export interface JobStageInfo {
  phase: JobPhase;
  /** 短徽标（下拉标签用，尽量精简） */
  badge: string;
  /** 下一步建议的模式（null = 已无明确下一步） */
  nextMode: "doing" | "ctrl" | "learning" | "dream" | null;
  /** 一句话指引（选中后展示） */
  hint: string;
  /** 徽标色调（tailwind class 片段） */
  tone: "morty" | "portal" | "danger" | "ink";
}

export function jobStageInfo(job: JobSummary): JobStageInfo {
  const tasks = job.tasks ?? [];
  const done = tasks.filter((t) => t.status === "success").length;
  const failed = tasks.filter((t) => t.status === "error").length;
  const running = tasks.filter((t) => t.status === "running").length;

  if (job.stage === "planned" || tasks.length === 0) {
    return {
      phase: "planned",
      badge: "plan 就绪",
      nextMode: "doing",
      hint: "plan 已产出任务清单，但还没执行——下一步用 doing 开始执行",
      tone: "morty",
    };
  }
  if (running > 0) {
    return {
      phase: "running",
      badge: `执行中 ${done}/${tasks.length}`,
      nextMode: "ctrl",
      hint: "该 job 正在执行——如需追加指令/重置 task 用 ctrl；想继续跑用 doing",
      tone: "portal",
    };
  }
  if (failed > 0) {
    return {
      phase: "failed",
      badge: `${failed} 个失败`,
      nextMode: "doing",
      hint: "有失败的 task——下一步用 doing 重试（会把未完成的 task 重新编排）",
      tone: "danger",
    };
  }
  if (done === tasks.length) {
    if (job.archived_by === "dream") {
      return {
        phase: "dreamed",
        badge: "已完成 · dream 已学习",
        nextMode: null,
        hint: "该 job 已完成且被 dream 学习沉淀过——通常无需再处理",
        tone: "ink",
      };
    }
    return {
      phase: "done",
      badge: "已完成",
      nextMode: "learning",
      hint: "全部 task 已完成——下一步建议 learning（沉淀 SUMMARY/skills/loops）",
      tone: "portal",
    };
  }
  return {
    phase: "pending",
    badge: `${done}/${tasks.length} 完成`,
    nextMode: "doing",
    hint: "还有未完成的 task——下一步用 doing 继续执行",
    tone: "morty",
  };
}

/** 选中的 job 与当前所选会话模式是否匹配；不匹配时返回提示语（否则 null）。 */
export function modeFitWarning(mode: string | null, job: JobSummary): string | null {
  const info = jobStageInfo(job);
  switch (mode) {
    case "doing":
      if (info.phase === "done" || info.phase === "dreamed") {
        return "该 job 的 task 已全部完成——doing 没有待执行的 task；建议改用 learning（知识沉淀）";
      }
      return null;
    case "learning":
      if (info.phase === "planned" || info.phase === "pending" || info.phase === "failed" || info.phase === "running") {
        return "该 job 还没执行完——learning 需要已完成的 job 才能沉淀（先跑 doing）";
      }
      return null;
    case "ctrl":
      if (info.phase === "planned") {
        return "该 job 还没开始执行——ctrl 用于干预运行中的 job；先用 doing 启动";
      }
      return null;
    default:
      return null;
  }
}
