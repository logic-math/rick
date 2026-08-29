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

import { useMemo, useState } from "react";
import Ansi from "ansi-to-react";
import { DiffView, DiffModeEnum } from "@git-diff-view/react";
import "@git-diff-view/react/styles/diff-view.css";
import { summarizeArgs, type ToolItem } from "./viewModel";

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

export default function ToolCallCard({ item }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);
  const summary = summarizeArgs(item.args);
  const isBash = item.toolName === "bash";
  const isEdit = item.toolName === "edit";
  const edit = useMemo(() => (isEdit ? editStrings(item.args) : null), [isEdit, item.args]);

  const hunk = useMemo(() => {
    if (!edit) return null;
    return opsToHunk(lcsLineDiff(edit.oldStr.split("\n"), edit.newStr.split("\n")));
  }, [edit]);

  const hasResult = item.output.length > 0;

  return (
    <div
      className={`my-1 overflow-hidden rounded-md border bg-space/60 ${
        item.status === "error" ? "border-danger/60" : "border-line"
      }`}
    >
      {/* 头部：工具名 + 摘要 + 状态 */}
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-white/5"
        aria-expanded={expanded}
      >
        <span aria-hidden="true" className="text-ink-3">
          {expanded ? "▾" : "▸"}
        </span>
        <StatusIcon status={item.status} />
        <code className="shrink-0 rounded bg-surface-raised px-1.5 py-0.5 font-mono text-[11px] text-portal">
          {item.toolName}
        </code>
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
                    <Ansi>{item.output}</Ansi>
                  </div>
                ) : (
                  <pre className="p-2 font-mono text-[11px] leading-relaxed text-ink-2">
                    {item.output}
                  </pre>
                )}
              </div>
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
