/**
 * CompactionCard：上下文压缩过程/结果卡（job_53 Bug #1 修复 4，human 裁决：
 * 压缩过程可视化——压缩后作为一条会话消息显示，压缩提示词全文折叠可展开
 * 滚动阅读，感知「压缩是怎么发生的」）。
 *
 * 三形态：
 *  - pending：压缩中占位（compaction_start → end 之间）
 *  - 完成/中止/失败：结果卡（compaction_end）
 *  - historical：历史回放的既有压缩点（轻量分割线——REST /entries 的
 *    compaction 条目；不带事件详情，不重放摘要全文）
 *
 * 数据全部来自既有 SSE 事件（compaction_start/end 的 result）与历史条目，
 * 无新接口。提示词原文副本见 compactionPrompts.ts（pi 0.84.2 对齐）。
 */
import { memo } from "react";
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
  // 历史压缩点：轻量分割线（不重放摘要全文——它对回放只是边界标记）
  if (item.historical) {
    const scale =
      typeof item.tokensBefore === "number" ? `（压缩前上下文 ${fmtTokens(item.tokensBefore)} token）` : "";
    return (
      <div className="my-2 flex items-center gap-2 text-[11px] text-ink-2" title="此处之前的消息已被压缩为摘要（完整历史仍保留在会话文件中）">
        <span className="h-px flex-1 bg-line" />
        <span aria-hidden="true">⬆</span>
        <span>此处之前已压缩{scale}</span>
        <span className="h-px flex-1 bg-line" />
      </div>
    );
  }

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

  // 完成卡
  const before = item.tokensBefore;
  const after = item.estimatedTokensAfter;
  const pct =
    typeof before === "number" && typeof after === "number" && before > 0
      ? `（↓${Math.max(0, Math.round((1 - after / before) * 100))}%）`
      : "";
  return (
    <div className="my-2 rounded-md border border-portal/40 bg-space/60 px-3 py-2 text-xs">
      <div className="flex items-center gap-2 font-medium text-ink">
        <span aria-hidden="true">✂</span>
        上下文压缩完成 · {reasonLabel(item.reason)}
      </div>
      <p className="mt-1 text-ink-2">
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
      {item.summary ? (
        <Fold title="📄 压缩摘要（点击展开）" body={item.summary} />
      ) : null}
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
    prev.item.tokensBefore === next.item.tokensBefore &&
    prev.item.estimatedTokensAfter === next.item.estimatedTokensAfter,
);
