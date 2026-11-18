/**
 * CompactionCard：上下文压缩过程/结果卡（job_53 Bug #1 修复 4，human 裁决：
 * 压缩过程可视化——压缩后作为一条会话消息显示，压缩提示词全文折叠可展开
 * 滚动阅读，感知「压缩是怎么发生的」；迭代 2 裁决：压缩卡一直在会话流中
 * （刷新后历史重建也渲染完整卡）且整卡可折叠）。
 *
 * 三形态：
 *  - pending：压缩中占位（compaction_start → end 之间，不折叠）
 *  - 完成/中止/失败：结果卡（compaction_end；中止/失败不折叠）
 *  - historical：历史回放的既有压缩点（REST /entries 的 compaction 条目；
 *    完整卡但默认收起——多张历史卡全展开占屏）
 *
 * 完成卡（live/historical）支持整卡折叠：头部一行常驻（✂ 标题 + 规模摘要），
 * 点击折叠/展开整个卡身（策略行/摘要折叠区/提示词折叠区全部收起）。
 * 数据全部来自既有 SSE 事件（compaction_start/end 的 result）与历史条目，
 * 无新接口。提示词原文副本见 compactionPrompts.ts（pi 0.84.2 对齐）。
 */
import { memo, useState } from "react";
import type { CompactionItem } from "./viewModel";
import { PI_COMPACTION_PROMPT_VERSION, PI_COMPACTION_REQUEST_SHAPE, PI_SUMMARIZATION_PROMPT, PI_SUMMARIZATION_SYSTEM_PROMPT, PI_UPDATE_SUMMARIZATION_PROMPT } from "./compactionPrompts";

function reasonLabel(reason: string): string {
  switch (reason) {
    case "manual":
      return "手动";
    case "threshold":
      return "自动 · 阈值触发";
    case "overflow":
      return "自动 · 上下文溢出";
    case "historical":
      return "历史压缩点";
    default:
      return reason;
  }
}

/** 折叠节（原生 details——默认折叠，点击展开；长内容 max-h 内滚动阅读） */
function Fold({ title, body, tone }: { title: string; body: string; tone?: string }) {
  return (
    <details className={`group mt-1 rounded border border-line/60 bg-space/60 text-xs ${tone ?? ""}`}>
      <summary className="cursor-pointer select-none px-2 py-1 text-ink-2 hover:text-ink">
        <span className="mr-1 inline-block transition-transform group-open:rotate-90" aria-hidden="true">
          ▸
        </span>
        {title}
      </summary>
      <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words border-t border-line/60 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-ink-2">
        {body}
      </pre>
    </details>
  );
}

function fmtTokens(n: number | undefined): string {
  if (typeof n !== "number" || !Number.isFinite(n)) return "—";
  return n.toLocaleString("en-US");
}

function CompactionCardInner({ item }: { item: CompactionItem }) {
  // 整卡折叠态（hooks 置顶：pending/aborted 形态早退不影响 hooks 规则）。
  // live 完成卡默认展开（即时看到压缩结果）；historical 卡默认收起（多张
  // 历史卡全展开占屏；同一实例从 pending→填充时 historical 恒 undefined，
  // 折叠态不被重置）。pending/aborted/error 形态不折叠（状态必须可见）。
  const [open, setOpen] = useState(!item.historical);

  // 压缩中占位
  if (item.pending) {
    return (
      <div className="my-2 flex items-center gap-2 rounded-md border border-line bg-space/60 px-3 py-2 text-xs text-ink-2">
        <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-portal" aria-hidden="true" />
        上下文压缩中…（{reasonLabel(item.reason)}）
      </div>
    );
  }

  // 中止 / 失败
  if (item.aborted || item.errorMessage) {
    return (
      <div className="my-2 flex items-start gap-2 rounded-md border border-morty/50 bg-morty/10 px-3 py-2 text-xs text-morty">
        <span aria-hidden="true">{item.aborted ? "⏹" : "⛔"}</span>
        <p className="min-w-0 flex-1 whitespace-pre-wrap break-words">
          上下文压缩{item.aborted ? "已中止" : "失败"}（{reasonLabel(item.reason)}）
          {item.errorMessage ? `：${item.errorMessage}` : ""}
        </p>
      </div>
    );
  }

  // 完成卡（live / historical）——整卡可折叠：头部一行常驻，点击折叠/展开卡身
  const before = item.tokensBefore;
  const after = item.estimatedTokensAfter;
  const pct =
    typeof before === "number" && typeof after === "number" && before > 0
      ? `（↓${Math.max(0, Math.round((1 - after / before) * 100))}%）`
      : "";
  // 头部规模摘要：live 卡有前后对比；historical 卡无 estimatedTokensAfter
  //（JSONL 条目不存），如实展示压缩前规模，不伪造对比。
  const headerScale =
    typeof after === "number"
      ? `${fmtTokens(before)} → ${fmtTokens(after)} token${pct}`
      : typeof before === "number"
        ? `压缩前 ${fmtTokens(before)} token`
        : "";
  return (
    <div className="my-2 rounded-md border border-portal/40 bg-space/60 px-3 py-2 text-xs">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 text-left font-medium text-ink hover:text-ink"
        aria-expanded={open}
        title={open ? "点击折叠压缩卡" : "点击展开压缩卡"}
      >
        <span
          className="inline-block transition-transform select-none text-ink-3"
          style={{ transform: open ? "rotate(90deg)" : "rotate(0deg)" }}
          aria-hidden="true"
        >
          ▸
        </span>
        <span aria-hidden="true">✂</span>
        <span>上下文压缩完成 · {reasonLabel(item.reason)}</span>
        {headerScale ? <span className="text-ink-3 font-normal">· {headerScale}</span> : null}
      </button>
      {open ? (
        <div className="mt-1">
          <p className="text-ink-2">
            策略：保留最近约 2 万 token 原始消息，之前的历史由模型总结为结构化摘要
            （Goal / 约束 / 进度 / 下一步；迭代压缩继承前次摘要）
            {item.firstKeptEntryId ? `；压缩边界 ${item.firstKeptEntryId.slice(0, 8)}… 起保留` : ""}
          </p>
          <p className="mt-0.5 text-ink-2">
            规模：{fmtTokens(before)} → {fmtTokens(after)} token{pct}
          </p>
          {item.customInstructions ? (
            <p className="mt-0.5 text-ink-2">
              压缩指令：<span className="whitespace-pre-wrap">{item.customInstructions}</span>
            </p>
          ) : null}
          {item.summary ? <Fold title="📄 压缩摘要（点击展开）" body={item.summary} /> : null}
          <Fold
            title={`🔬 压缩提示词全文（pi v${PI_COMPACTION_PROMPT_VERSION}，点击展开滚动阅读）`}
            body={PI_COMPACTION_REQUEST_SHAPE}
          />
          <details className="group rounded border border-line/40 bg-transparent text-[11px]">
            <summary className="cursor-pointer select-none px-2 py-1 text-ink-2 hover:text-ink">
              <span className="mr-1 inline-block transition-transform group-open:rotate-90" aria-hidden="true">
                ▸
              </span>
              └ System prompt
            </summary>
            <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-relaxed text-ink-2">
              {PI_SUMMARIZATION_SYSTEM_PROMPT}
            </pre>
          </details>
          <details className="group rounded border border-line/40 bg-transparent text-[11px]">
            <summary className="cursor-pointer select-none px-2 py-1 text-ink-2 hover:text-ink">
              <span className="mr-1 inline-block transition-transform group-open:rotate-90" aria-hidden="true">
                ▸
              </span>
              └ 首次压缩模板
            </summary>
            <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-relaxed text-ink-2">
              {PI_SUMMARIZATION_PROMPT}
            </pre>
          </details>
          <details className="group rounded border border-line/40 bg-transparent text-[11px]">
            <summary className="cursor-pointer select-none px-2 py-1 text-ink-2 hover:text-ink">
              <span className="mr-1 inline-block transition-transform group-open:rotate-90" aria-hidden="true">
                ▸
              </span>
              └ 迭代压缩模板（已有前次摘要时）
            </summary>
            <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-relaxed text-ink-2">
              {PI_UPDATE_SUMMARIZATION_PROMPT}
            </pre>
          </details>
          <p className="mt-1 text-[11px] text-ink-3">
            提示词副本对齐 pi v{PI_COMPACTION_PROMPT_VERSION}；被压缩的对话原文见会话历史（完整内容保留在会话文件中，不丢失）。
          </p>
        </div>
      ) : null}
    </div>
  );
}

export const CompactionCard = memo(
  CompactionCardInner,
  (prev, next) =>
    prev.item.id === next.item.id &&
    prev.item.pending === next.item.pending &&
    prev.item.summary === next.item.summary &&
    prev.item.errorMessage === next.item.errorMessage &&
    prev.item.aborted === next.item.aborted &&
    prev.item.historical === next.item.historical &&
    prev.item.tokensBefore === next.item.tokensBefore &&
    prev.item.estimatedTokensAfter === next.item.estimatedTokensAfter,
);
