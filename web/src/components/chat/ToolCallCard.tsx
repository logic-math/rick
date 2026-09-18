/**
 * ToolCallCard：工具调用折叠卡。
 *
 * - 收起态：工具名 + 参数摘要单行 + 状态（运行中 spinner / 完成 ✓ / 失败 ✗）
 * - 展开态：完整参数（代码块）+ 结果
 *   - bash → ansi-to-react 渲染（保留终端色彩）
 *   - read/write → 代码块
 *   - edit → @git-diff-view/react 渲染 diff（LCS 行级对齐 + unified hunk）
 * - 结果截断 → 提示 + fullOutputPath
 */

import { memo, useMemo, useState, useEffect } from "react";
import Ansi from "ansi-to-react";
import { DiffView, DiffModeEnum } from "@git-diff-view/react";
import "@git-diff-view/react/styles/diff-view.css";
import { summarizeArgs, type ToolItem } from "./viewModel";
import { useExpandState } from "./expand";

/** 工具输出截断阈值（8KB） */
const MAX_OUTPUT = 8 * 1024;

// ============================================================
// LCS 行级 diff（轻量实现——edit 工具卡的 old/new 字符串对齐）
// ============================================================

type DiffOp = { t: " " | "-" | "+"; line: string };

function lcsLineDiff(oldLines: string[], newLines: string[]): DiffOp[] {
  const n = oldLines.length;
  const m = newLines.length;
  // dp[i][j] = oldLines[i:] 与 newLines[j:] 的最长公共子序列长度
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = oldLines[i] === newLines[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  const ops: DiffOp[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (oldLines[i] === newLines[j]) {
      ops.push({ t: " ", line: oldLines[i] });
      i++;
      j++;
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      ops.push({ t: "-", line: oldLines[i] });
      i++;
    } else {
      ops.push({ t: "+", line: newLines[j] });
      j++;
    }
  }
  while (i < n) {
    ops.push({ t: "-", line: oldLines[i] });
    i++;
  }
  while (j < m) {
    ops.push({ t: "+", line: newLines[j] });
    j++;
  }
  return ops;
}

/** LCS ops → 单 hunk unified diff 字符串 */
function opsToHunk(ops: DiffOp[]): string {
  const oldCount = ops.filter((o) => o.t !== "+").length;
  const newCount = ops.filter((o) => o.t !== "-").length;
  const lines = [`@@ -1,${oldCount} +1,${newCount} @@`];
  for (const op of ops) lines.push(`${op.t}${op.line}`);
  return lines.join("\n");
}

// ============================================================
// 工具参数/结果提取
// ============================================================

function argString(args: unknown, key: string): string | undefined {
  if (args && typeof args === "object" && key in (args as Record<string, unknown>)) {
    const v = (args as Record<string, unknown>)[key];
    if (typeof v === "string") return v;
  }
  return undefined;
}

/** edit 工具的 old/new 字符串（多命名防御：pi edit 参数名以实测为准） */
function editStrings(args: unknown): { filePath: string; oldStr: string; newStr: string } | null {
  const oldStr = argString(args, "old_string") ?? argString(args, "oldText") ?? argString(args, "old");
  const newStr = argString(args, "new_string") ?? argString(args, "newText") ?? argString(args, "new");
  const filePath = argString(args, "file_path") ?? argString(args, "path") ?? argString(args, "filePath");
  if (oldStr === undefined || newStr === undefined || !filePath) return null;
  return { filePath, oldStr, newStr };
}

/** 从文件路径猜语言（diff 语法高亮） */
function langOf(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  const map: Record<string, string> = {
    ts: "typescript",
    tsx: "tsx",
    js: "javascript",
    jsx: "jsx",
    go: "go",
    py: "python",
    rs: "rust",
    md: "markdown",
    json: "json",
    yaml: "yaml",
    yml: "yaml",
    sh: "bash",
    css: "css",
    html: "xml",
    sql: "sql",
    toml: "ini",
  };
  return map[ext] ?? "plaintext";
}

// ============================================================
// 状态图标
// ============================================================

/** 工具运行的起始时刻（模块级缓存）：vm 每帧从事件重放重建 item，无法把
 *  「第一次看到它运行」的时间放进纯函数产物，故用 toolCallId（唯一）→ 首次观察到
 *  running 的时刻。用于显示已运行时长——**卡死检测**（用户实测：agent 的 bash 工具
 *  调用陷入死循环 python 9m48s，UI 只显示「运行中」，用户只能干等、反复重发消息）。 */
const runStartedAt = new Map<string, number>();

/** 记录工具首次被观察到「运行中」的时刻（幂等）——组标题与卡片共用。 */
export function noteToolRunStart(toolId: string): void {
  if (!runStartedAt.has(toolId)) runStartedAt.set(toolId, Date.now());
}

function useTick(active: boolean, ms = 1000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    setNow(Date.now());
    const t = setInterval(() => setNow(Date.now()), ms);
    return () => clearInterval(t);
  }, [active, ms]);
  return now;
}

/** 运行中工具的已运行时长；≥2 分钟转警示色并提示可能卡住（可 Abort）。
 *  独立子组件自带计时器——父卡被 memo 跳过重渲时它仍能每秒自更新。 */
export function ToolElapsed({ toolId }: { toolId: string }) {
  const now = useTick(true);
  const started = runStartedAt.get(toolId) ?? now;
  const sec = Math.max(0, Math.round((now - started) / 1000));
  const label = sec < 60 ? `${sec}s` : `${Math.floor(sec / 60)}m${String(sec % 60).padStart(2, "0")}s`;
  const stuck = sec >= 120;
  return (
    <span
      className={`shrink-0 rounded px-1 font-mono text-[10px] ${
        stuck ? "bg-morty/15 text-morty" : "text-ink-3"
      }`}
      title={
        stuck
          ? "该工具调用已运行较久，可能卡住（例如命令陷入死循环/等待输入）——可用下方 Abort 中断本轮后重试"
          : "工具调用已运行时长"
      }
    >
      {stuck ? `⚠ ${label}` : label}
    </span>
  );
}

function StatusIcon({ status }: { status: ToolItem["status"] }) {
  if (status === "running") {
    return (
      <span
        className="inline-block h-2.5 w-2.5 rounded-full border-2 border-portal border-t-transparent"
        style={{ animation: "rm-portal-spin 0.8s linear infinite" }}
        aria-label="运行中"
      />
    );
  }
  if (status === "error") {
    return (
      <span className="text-xs text-danger" aria-label="失败">
        ✗
      </span>
    );
  }
  return (
    <span className="text-xs text-portal" aria-label="完成">
      ✓
    </span>
  );
}

// ============================================================
// 主组件
// ============================================================

interface ToolCallCardProps {
  item: ToolItem;
}

function ToolCallCardImpl({ item }: ToolCallCardProps) {
  const { open, onToggle } = useExpandState(`tool-${item.toolCallId}`, false);
  const [expandedFull, setExpandedFull] = useState(false);
  const summary = summarizeArgs(item.args);
  const isBash = item.toolName === "bash";
  const isEdit = item.toolName === "edit";
  const edit = useMemo(() => (isEdit ? editStrings(item.args) : null), [isEdit, item.args]);

  const hunk = useMemo(() => {
    if (!edit) return null;
    return opsToHunk(lcsLineDiff(edit.oldStr.split("\n"), edit.newStr.split("\n")));
  }, [edit]);

  const hasResult = item.output.length > 0;
  const outputTruncated = !expandedFull && item.output.length > MAX_OUTPUT;
  const outputShown = outputTruncated ? item.output.slice(0, MAX_OUTPUT) : item.output;
  const expanded = open;

  return (
    <div
      className={`my-1 overflow-hidden rounded-md border bg-space/60 ${
        item.status === "error" ? "border-danger/60" : "border-line"
      }`}
    >
      {/* 头部：工具名 + 摘要 + 状态 */}
      <button
        type="button"
        onClick={() => onToggle(!open)}
        className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-white/5"
        aria-expanded={open}
      >
        <span aria-hidden="true" className="text-ink-3">
          {expanded ? "▾" : "▸"}
        </span>
        <StatusIcon status={item.status} />
        <code className="shrink-0 rounded bg-surface-raised px-1.5 py-0.5 font-mono text-[11px] text-portal">
          {item.toolName}
        </code>
        {item.status === "running" &&
          (() => {
            noteToolRunStart(item.id);
            return <ToolElapsed toolId={item.id} />;
          })()}
        {summary && (
          <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-ink-2">{summary}</span>
        )}
        {!summary && <span className="min-w-0 flex-1" />}
        {isEdit && edit && (
          <span className="shrink-0 truncate font-mono text-[10px] text-ink-3">{edit.filePath}</span>
        )}
      </button>

      {/* 展开态：完整参数 + 结果 */}
      {expanded && (
        <div className="border-t border-line">
          {/* 完整参数 */}
          {item.args != null && (
            <div className="border-b border-line/60 px-3 py-2">
              <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-3">参数</div>
              <pre className="overflow-x-auto rounded bg-space p-2 font-mono text-[11px] leading-relaxed text-ink-2">
                {typeof item.args === "string"
                  ? item.args
                  : JSON.stringify(item.args, null, 2)}
              </pre>
            </div>
          )}

          {/* edit → diff 视图 */}
          {isEdit && edit && hunk && (
            <div className="border-b border-line/60 px-3 py-2">
              <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-3">
                变更 diff
              </div>
              <div className="overflow-x-auto rounded bg-space text-xs">
                <DiffView
                  data={{
                    oldFile: {
                      fileName: edit.filePath,
                      fileLang: langOf(edit.filePath),
                      content: edit.oldStr,
                    },
                    newFile: {
                      fileName: edit.filePath,
                      fileLang: langOf(edit.filePath),
                      content: edit.newStr,
                    },
                    hunks: [hunk],
                  }}
                  diffViewMode={DiffModeEnum.Unified}
                  diffViewTheme="dark"
                  diffViewFontSize={12}
                  diffViewHighlight
                />
              </div>
            </div>
          )}

          {/* 结果 */}
          {hasResult && (
            <div className="px-3 py-2">
              <div className="mb-1 flex items-center gap-2 text-[10px] uppercase tracking-wide text-ink-3">
                <span>结果</span>
                {item.status === "error" && <span className="text-danger">失败</span>}
              </div>
              <div className="max-h-80 overflow-auto rounded bg-space">
                {isBash ? (
                  <div className="p-2 font-mono text-[11px] leading-relaxed">
                    <Ansi>{outputShown}</Ansi>
                  </div>
                ) : (
                  <pre className="p-2 font-mono text-[11px] leading-relaxed text-ink-2">
                    {outputShown}
                  </pre>
                )}
              </div>
              {outputTruncated && (
                <button
                  type="button"
                  onClick={() => setExpandedFull(true)}
                  className="mt-1 text-[10px] text-portal hover:text-portal-strong"
                >
                  展开全部（{Math.ceil((item.output.length - MAX_OUTPUT) / 1024)}K 已截断）
                </button>
              )}
              {(item.truncated || item.fullOutputPath) && (
                <div className="mt-1 text-[10px] text-morty">
                  ⚠ 输出已截断
                  {item.fullOutputPath ? `（全文：${item.fullOutputPath}）` : ""}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// 流式稳定性：vm 每帧重建 → item 对象每帧新建，浅比较必然不等 → 已落定的工具卡
// 每帧重渲（闪烁机制三）。自定义比较：内容字段不变则跳过渲染；running 中 status
// /output 变化会自然失配（重渲），等待中的 running 卡无视觉变化也无需重渲。
function sameToolItem(a: ToolItem, b: ToolItem): boolean {
  return (
    a.id === b.id &&
    a.toolName === b.toolName &&
    a.status === b.status &&
    a.output === b.output &&
    a.args === b.args &&
    a.truncated === b.truncated &&
    a.fullOutputPath === b.fullOutputPath
  );
}

const ToolCallCard = memo(
  ToolCallCardImpl,
  (prev, next) => sameToolItem(prev.item, next.item),
);

export default ToolCallCard;
