/**
 * SteerBar：会话底部操作条（按状态分派）。
 *
 * - active + streaming：Abort 按钮 + steer 输入（ChatInput 复用，发送即 steer）
 * - active + idle：prompt 输入（发送即 prompt）
 * - closed / error：Resume 按钮 + 说明
 * - suspended：因**平台升级/服务重启**挂起——专属文案 +「▶ 恢复继续」
 *   （与 error 的「已中断」文案严格区分：语义不同，且恢复路径也不同）
 */

import ChatInput, { type SlashCommand } from "./ChatInput";
import Saucer from "../starfield/Saucer";

export type SessionPhase =
  | "idle"
  | "streaming"
  | "closed"
  | "error"
  | "suspended"
  | "loading";

interface SteerBarProps {
  phase: SessionPhase;
  /** idle：发 prompt；streaming：发 steer */
  onSend: (message: string) => void;
  /** 斜杠命令（/abort /close 实装；/compact 由 ChatInput 就地提示） */
  onCommand: (cmd: SlashCommand) => void;
  onAbort: () => void;
  onResume: () => void;
  /** suspended：人工确认继续（/continue；doing/dream 会归一化后续跑）。缺省回退 onResume。 */
  onContinue?: () => void;
  /** 请求进行中（防连击） */
  busy?: boolean;
  /** compacting 提示（上下文压缩中——steer 仍可排队） */
  compacting?: boolean;
}

export default function SteerBar({
  phase,
  onSend,
  onCommand,
  onAbort,
  onResume,
  onContinue,
  busy,
  compacting,
}: SteerBarProps) {
  if (phase === "loading") {
    return (
      <div className="flex items-center justify-center gap-2 px-4 py-4 text-xs text-ink-3">
        <Saucer size={26} flying />
        会话加载中…
      </div>
    );
  }

  // 挂起（平台升级/服务重启）：**不自动恢复**——需要人点一下。
  if (phase === "suspended") {
    return (
      <div className="flex flex-wrap items-center gap-3 px-4 py-3">
        <span className="text-xs text-ink-2">
          ⏸ 会话因<b>平台升级</b>挂起 —— 点击恢复继续（不会自动续跑，避免副作用）
        </span>
        <button
          type="button"
          onClick={onContinue ?? onResume}
          disabled={busy}
          className="ml-auto rounded-lg border border-portal/50 bg-portal/10 px-4 py-1.5 text-xs font-medium text-portal hover:bg-portal/20 disabled:opacity-40"
        >
          ▶ 恢复继续
        </button>
      </div>
    );
  }

  if (phase === "closed" || phase === "error") {
    return (
      <div className="flex items-center gap-3 px-4 py-3">
        <span className="text-xs text-ink-3">
          {phase === "error"
            ? "会话已中断（agent 进程不在）——"
            : "会话已关闭——"}
          点击 Resume 重新加载完整历史并恢复 agent 进程
        </span>
        <button
          type="button"
          onClick={onResume}
          disabled={busy}
          className="ml-auto rounded-lg bg-portal/15 px-4 py-1.5 text-xs font-medium text-portal hover:bg-portal/25 disabled:opacity-40"
        >
          ⟳ Resume
        </button>
      </div>
    );
  }

  return (
    <div className="px-4 pb-3">
      {/* streaming 状态条 */}
      {phase === "streaming" && (
        <div className="mb-1.5 flex items-center gap-2 text-xs text-ink-3">
          <Saucer size={22} flying />
          <span>生成中——输入即纠偏（steer），或中止</span>
          {compacting && <span className="text-morty">· 上下文压缩中</span>}
          <button
            type="button"
            onClick={onAbort}
            disabled={busy}
            className="ml-auto rounded-md border border-danger/50 px-2.5 py-1 text-xs text-danger hover:bg-danger/10 disabled:opacity-40"
          >
            ■ Abort
          </button>
        </div>
      )}
      {phase === "idle" && compacting && (
        <div className="mb-1.5 text-xs text-morty">上下文压缩中…</div>
      )}

      <ChatInput
        onSend={onSend}
        onCommand={onCommand}
        placeholder={phase === "streaming" ? "steer 纠偏：排队下一轮注入的指令…" : undefined}
        busy={busy}
      />
    </div>
  );
}
