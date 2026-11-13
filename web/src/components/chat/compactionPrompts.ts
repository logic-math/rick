/**
 * pi 压缩提示词原文副本（job_53 Bug #1 修复 4：压缩过程可视化——human 裁决
 * 「展示全部原文，折叠可展开滚动阅读」）。
 *
 * 来源：dev 8414 实际运行的 pi 安装（runtime/node_modules/@earendil-works/
 * pi-coding-agent/dist/core/compaction/{utils,compaction}.js），版本 **0.84.2**。
 * pi 升级后模板可能变化——届时需重新对齐并更新 PI_COMPACTION_PROMPT_VERSION。
 */

/** pi 版本号（提示词副本的对齐锚点） */
export const PI_COMPACTION_PROMPT_VERSION = "0.84.2";

/** 压缩总结请求的 system prompt（逐字抄录，pi 0.84.2） */
export const PI_SUMMARIZATION_SYSTEM_PROMPT = `You are a context summarization assistant. Your task is to read a conversation between a user and an AI assistant, then produce a structured summary following the exact format specified.

Do NOT continue the conversation. Do NOT respond to any questions in the conversation. ONLY output the structured summary.`;

/** 首次压缩的 user prompt 模板（逐字抄录，pi 0.84.2） */
export const PI_SUMMARIZATION_PROMPT = `The messages above are a conversation to summarize. Create a structured context checkpoint summary that another LLM will use to continue the work.

Use this EXACT format:

## Goal
[What is the user trying to accomplish? Can be multiple items if the session covers different tasks.]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages.`;

/** 迭代压缩（已有前次摘要）的 user prompt 模板（逐字抄录，pi 0.84.2） */
export const PI_UPDATE_SUMMARIZATION_PROMPT = `The messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags.

Update the existing structured summary with new information. RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if the task expanded]

## Constraints & Preferences
- [Preserve existing, add new ones discovered]

## Progress
### Done
- [x] [Include previously done items AND newly completed items]

### In Progress
- [ ] [Current work - update based on progress]

### Blocked
- [Current blockers - remove if resolved]

## Key Decisions
- **[Decision]**: [Brief rationale] (preserve all previous, add new)

## Next Steps
1. [Update based on current state]

## Critical Context
- [Preserve important context, add new if needed]

Keep each section concise. Preserve exact file paths, function names, and error messages.`;

/** prompt 拼装方式说明（pi compaction.js 的实际请求形态，human 感知压缩机制用） */
export const PI_COMPACTION_REQUEST_SHAPE = [
  'system: SUMMARIZATION_SYSTEM_PROMPT（见上）',
  'user: <conversation>…被压缩的对话原文…</conversation>',
  '      （迭代压缩时）<previous-summary>…前次摘要…</previous-summary>',
  "      （用户附了指令时）模板尾部追加 Additional focus: <自定义指令>",
  '      + SUMMARIZATION_PROMPT 或 UPDATE_SUMMARIZATION_PROMPT（见上）',
].join("\n");
