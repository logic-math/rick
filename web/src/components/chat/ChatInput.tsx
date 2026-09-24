/**
 * ChatInput：多行输入框。
 *
 * - Enter 发送 / Shift+Enter 换行 / Ctrl+J 换行（对齐 pi 终端键位）
 * - Enter 语义可切换（rick.chat.enterMode，localStorage 持久化）：
 *   send（默认）Enter=发送；newline Enter=换行、Ctrl/Cmd+Enter=发送（中文 IM 习惯）
 * - 空内容/纯空白不发送（两模式统一，防误发）；IME 组合中不拦截 Enter
 * - 移动端（<768px）显式换行按钮：光标处插入换行（无 Shift 键可换行）
 * - agent streaming 时变为 steer 输入（发送即 steer——由 SteerBar 注入 onSend）
 * - 斜杠命令：输入以 / 开头弹出命令面板；/abort /close 实装（调 API），/compact
 *   提示暂不可用
 * - 软键盘适配：VisualViewport resize/scroll 监听——输入条贴软键盘上沿
 *   （fixed 底部 + 底偏移 = innerHeight - (vv.height + vv.offsetTop)）
 * - 附件按钮占位（disabled + tooltip「v1 暂不支持」）
 */

import { useEffect, useMemo, useRef, useState } from "react";

export interface SlashCommand {
  name: string;
  desc: string;
  available: boolean;
  /** 命令参数（如 `/model glm-5.3` 的 `glm-5.3`；无参数为空串） */
  arg?: string;
}

export const SLASH_COMMANDS: SlashCommand[] = [
  { name: "/model", desc: "切换模型", available: true },
  { name: "/thinking", desc: "切换思考档位", available: true },
  { name: "/abort", desc: "中止当前生成", available: true },
  { name: "/compact", desc: "压缩上下文（长会话降 token；可附指令引导摘要侧重点）", available: true },
  { name: "/close", desc: "关闭会话", available: true },
  { name: "/compact", desc: "压缩上下文（v1 暂不可用）", available: false },
];

interface ChatInputProps {
  /** 发送回调（prompt 或 steer 由调用方决定语义） */
  onSend: (message: string) => void;
  /** 斜杠命令执行回调（实装命令才回调；不可用命令就地提示） */
  onCommand: (cmd: SlashCommand) => void;
  placeholder?: string;
  disabled?: boolean;
  /** 发送中（请求已发出等待 echo——防连击） */
  busy?: boolean;
}

/** Enter 语义模式：send=Enter 发送（默认）；newline=Enter 换行、Ctrl/Cmd+Enter 发送 */
type EnterMode = "send" | "newline";

/** Enter 语义持久化 key（localStorage；与 starfield 设置各自独立 key） */
const ENTER_MODE_KEY = "rick.chat.enterMode";

/** 读 localStorage 的 Enter 模式（缺省/异常回退 send；隐私模式 localStorage 不可用） */
function readEnterMode(): EnterMode {
  try {
    return localStorage.getItem(ENTER_MODE_KEY) === "newline" ? "newline" : "send";
  } catch {
    return "send";
  }
}

/** 软键盘高度偏移（px；无 VisualViewport/桌面端为 0） */
function useKeyboardOffset(): number {
  const [offset, setOffset] = useState(0);
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;
    const update = () => {
      // 软键盘弹出时 vv.height 缩小、vv.offsetTop 可能 >0
      const kb = window.innerHeight - (vv.height + vv.offsetTop);
      setOffset(kb > 0 ? kb : 0);
    };
    vv.addEventListener("resize", update);
    vv.addEventListener("scroll", update);
    update();
    return () => {
      vv.removeEventListener("resize", update);
      vv.removeEventListener("scroll", update);
    };
  }, []);
  return offset;
}

export default function ChatInput({ onSend, onCommand, placeholder, disabled, busy }: ChatInputProps) {
  const [text, setText] = useState("");
  const [menuIndex, setMenuIndex] = useState(0);
  const [enterMode, setEnterMode] = useState<EnterMode>(readEnterMode);
  const taRef = useRef<HTMLTextAreaElement | null>(null);
  const keyboardOffset = useKeyboardOffset();

  const toggleEnterMode = () => {
    setEnterMode((m) => {
      const next: EnterMode = m === "send" ? "newline" : "send";
      try {
        localStorage.setItem(ENTER_MODE_KEY, next);
      } catch {
        // localStorage 不可用（隐私模式等）：仅本组件内生效
      }
      return next;
    });
  };

  /** 在光标处插入换行（移动端换行按钮 / Ctrl+J 共用）：插入后回焦并把光标移到新行 */
  const insertNewline = () => {
    const ta = taRef.current;
    if (!ta || disabled) return;
    const start = ta.selectionStart ?? text.length;
    const end = ta.selectionEnd ?? start;
    const next = text.slice(0, start) + "\n" + text.slice(end);
    setText(next);
    // 受控组件：等 React 把新值刷进 DOM 再定位光标（requestAnimationFrame 后值已生效）
    requestAnimationFrame(() => {
      const pos = start + 1;
      ta.focus();
      ta.setSelectionRange(pos, pos);
    });
  };

  // 斜杠命令面板匹配
  const slashMatches = useMemo(() => {
    if (!text.startsWith("/") || text.includes("\n")) return [];
    const q = text.slice(1).toLowerCase();
    return SLASH_COMMANDS.filter((c) => c.name.slice(1).toLowerCase().startsWith(q));
  }, [text]);

  useEffect(() => {
    setMenuIndex(0);
  }, [slashMatches.length]);

  const send = () => {
    const value = text.trim();
    if (!value || disabled || busy) return;

    // 斜杠命令
    if (value.startsWith("/") && !value.includes("\n")) {
      const exact = SLASH_COMMANDS.find((c) => c.name === value);
      const target = exact ?? slashMatches[menuIndex];
      if (target) {
        if (target.available) {
          onCommand({ ...target, arg: value.slice(target.name.length).trim() });
          setText("");
        } else {
          // 不可用命令就地提示（不清空输入——用户可看到提示）
          setNotice(`${target.name} 暂不可用`);
        }
      }
      return;
    }

    onSend(value);
    setText("");
  };

  const [notice, setNotice] = useState<string | null>(null);
  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(null), 1800);
    return () => clearTimeout(t);
  }, [notice]);

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // 命令面板导航
    if (slashMatches.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setMenuIndex((i) => (i + 1) % slashMatches.length);
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setMenuIndex((i) => (i - 1 + slashMatches.length) % slashMatches.length);
        return;
      }
      if (e.key === "Tab") {
        e.preventDefault();
        setText(slashMatches[menuIndex].name);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setText("");
        return;
      }
    }
    // Ctrl+J 换行（两模式通用，对齐 pi 终端键位；IME 组合中不拦截）
    if (e.key.toLowerCase() === "j" && e.ctrlKey && !e.metaKey && !e.altKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      insertNewline();
      return;
    }
    if (e.key !== "Enter" || e.nativeEvent.isComposing) return;
    // IME 组合中的 Enter 不拦截（确认候选词）；组合外的 Enter 才进入语义分支
    const cmd = e.ctrlKey || e.metaKey;
    if (enterMode === "newline") {
      // newline 模式：Enter=换行（原生插入，不 preventDefault）、Ctrl/Cmd+Enter=发送
      if (cmd && !e.shiftKey) {
        e.preventDefault();
        send();
      }
      // 其余（Enter / Shift+Enter）：原生换行
    } else if (!e.shiftKey) {
      // send 模式：Enter（含 Ctrl/Cmd+Enter）发送；Shift+Enter 原生换行
      e.preventDefault();
      send();
    }
  };

  return (
    <div className="relative" style={{ paddingBottom: keyboardOffset > 0 ? 0 : undefined }}>
      {/* 斜杠命令面板 */}
      {slashMatches.length > 0 && (
        <div className="absolute bottom-full left-0 z-10 mb-1 w-72 overflow-hidden rounded-md border border-line bg-surface-raised shadow-xl">
          {slashMatches.map((c, i) => (
            <button
              key={c.name}
              type="button"
              onMouseEnter={() => setMenuIndex(i)}
              onClick={() => {
                if (c.available) {
                  onCommand(c);
                  setText("");
                } else {
                  setNotice(`${c.name} 暂不可用`);
                }
              }}
              className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs ${
                i === menuIndex ? "bg-portal-soft text-portal" : "text-ink-2"
              }`}
            >
              <code className="font-mono">{c.name}</code>
              <span className="truncate text-ink-3">{c.desc}</span>
            </button>
          ))}
        </div>
      )}

      {/* 不可用命令提示 */}
      {notice && (
        <div className="absolute bottom-full left-0 z-10 mb-1 rounded border border-morty/50 bg-morty/10 px-3 py-1 text-xs text-morty">
          {notice}
        </div>
      )}

      <div className="flex items-end gap-2 rounded-xl border border-line bg-surface px-3 py-2 focus-within:border-portal/50">
        {/* 附件占位 */}
        <button
          type="button"
          disabled
          title="v1 暂不支持附件"
          className="mb-0.5 shrink-0 rounded-md px-1.5 py-1 text-ink-3 opacity-50"
          aria-label="附件（v1 暂不支持）"
        >
          ＋
        </button>

        <textarea
          ref={taRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={onKeyDown}
          placeholder={
            placeholder ??
            (enterMode === "newline"
              ? "输入消息，Enter 换行（Ctrl+Enter 发送；/ 命令）"
              : "输入消息，Enter 发送（Shift+Enter / Ctrl+J 换行；/ 命令）")
          }
          disabled={disabled}
          rows={Math.min(6, Math.max(1, text.split("\n").length))}
          className="max-h-36 min-h-[46px] flex-1 resize-none bg-transparent text-sm leading-relaxed text-ink outline-none placeholder:text-ink-3 disabled:opacity-50"
          aria-label="消息输入"
        />

        {/* 移动端显式换行按钮（<768px 常显，对齐 Tailwind md 断点；桌面走 Shift+Enter / Ctrl+J） */}
        <button
          type="button"
          onClick={insertNewline}
          disabled={disabled}
          title="插入换行"
          aria-label="插入换行"
          className="mb-0.5 shrink-0 rounded-md px-2 py-1 text-xs text-ink-3 transition-colors hover:bg-white/5 hover:text-ink disabled:cursor-not-allowed disabled:opacity-40 md:hidden"
        >
          ↵
        </button>

        {/* Enter 语义切换：send=Enter 发送 / newline=Enter 换行、Ctrl+Enter 发送 */}
        <button
          type="button"
          onClick={toggleEnterMode}
          title={
            enterMode === "send"
              ? "Enter=发送，Shift+Enter / Ctrl+J=换行（点击切换为 Enter=换行）"
              : "Enter=换行，Ctrl+Enter=发送（点击切换为 Enter=发送）"
          }
          aria-label={`切换 Enter 键行为（当前：${enterMode === "send" ? "Enter 发送" : "Enter 换行"}）`}
          className="mb-0.5 shrink-0 rounded-md px-1.5 py-1 text-[10px] leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
        >
          ↵{enterMode === "send" ? "发送" : "换行"}
        </button>

        <button
          type="button"
          onClick={send}
          disabled={disabled || busy || !text.trim()}
          className="mb-0.5 flex shrink-0 items-center gap-1.5 rounded-lg bg-portal/15 px-3 py-1.5 text-xs font-medium text-portal transition-colors hover:bg-portal/25 disabled:cursor-not-allowed disabled:opacity-40"
          aria-busy={busy}
        >
          {busy && (
            <span
              className="inline-block h-3 w-3 animate-spin rounded-full border border-current border-t-transparent"
              aria-hidden="true"
            />
          )}
          {busy ? "发送中…" : "发送"}
        </button>
      </div>
    </div>
  );
}
