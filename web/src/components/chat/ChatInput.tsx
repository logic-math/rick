/**
 * ChatInput：多行输入框（v2 极简版，job_43 增量迭代 2）。
 *
 * - **Enter=换行（固定行为）**；Ctrl/Cmd+Enter=发送（桌面隐形快捷键）
 * - Shift+Enter / Ctrl+J=换行（对齐 pi 终端键位）；IME 组合中不拦截 Enter
 * - 空内容/纯空白不发送（防误发）
 * - 输入条仅一个发送按钮（human 反馈：多按钮太丑，只要一个发送）
 * - textarea 随内容自适应扩大（scrollHeight 驱动——长句自动折行也撑高，
 *   不只按 \n 计数），上限约 6 行（240px），超出内部滚动
 * - agent streaming 时变为 steer 输入（发送即 steer——由 SteerBar 注入 onSend）
 * - 斜杠命令：输入以 / 开头弹出命令面板；/abort /close 实装（调 API），/compact
 *   提示暂不可用
 * - 软键盘适配：VisualViewport resize/scroll 监听——输入条贴软键盘上沿
 *   （fixed 底部 + 底偏移 = innerHeight - (vv.height + vv.offsetTop)）
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

/** textarea 自适应上限（px）：约 6 行（240px = max-h-60），超出内部滚动 */
const TA_MAX_HEIGHT = 240;

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
  const taRef = useRef<HTMLTextAreaElement | null>(null);
  const keyboardOffset = useKeyboardOffset();

  // textarea 高度自适应（KR6c）：scrollHeight 驱动——长句自动折行也撑高。
  // 先置 auto 再量 scrollHeight（否则 height 固定时 scrollHeight 量的是内容高，
  // 无法收缩）；上限 240px，超出后 textarea 自身出现滚动条。
  useEffect(() => {
    const ta = taRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    ta.style.height = `${Math.min(ta.scrollHeight, TA_MAX_HEIGHT)}px`;
  }, [text]);

  /** 在光标处插入换行（Ctrl+J 用）：插入后回焦并把光标移到新行 */
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
    // Ctrl+J 换行（对齐 pi 终端键位；IME 组合中不拦截）
    if (e.key.toLowerCase() === "j" && e.ctrlKey && !e.metaKey && !e.altKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      insertNewline();
      return;
    }
    if (e.key !== "Enter" || e.nativeEvent.isComposing) return;
    // IME 组合中的 Enter 不拦截（确认候选词）
    // Ctrl/Cmd+Enter=发送（桌面隐形快捷键）；Enter / Shift+Enter=原生换行（不拦截）
    if (e.ctrlKey || e.metaKey) {
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

      {/* 输入条（极简）：textarea + 唯一的发送按钮 */}
      <div className="flex items-end gap-2 rounded-xl border border-line bg-surface px-3 py-2 focus-within:border-portal/50">
        <textarea
          ref={taRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={onKeyDown}
          placeholder={placeholder ?? "输入消息，Enter 换行 · Ctrl+Enter 发送 · / 命令"}
          disabled={disabled}
          rows={1}
          className="max-h-60 min-h-[46px] flex-1 resize-none overflow-y-auto bg-transparent text-sm leading-relaxed text-ink outline-none placeholder:text-ink-3 disabled:opacity-50"
          aria-label="消息输入"
        />

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
