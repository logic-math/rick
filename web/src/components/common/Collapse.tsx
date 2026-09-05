/**
 * Collapse：通用可交互折叠容器（受控/非受控）。
 *
 * - 标题行（button 语义，键盘可达）+ ▸/▾ 指示 + 内容区
 * - 受控：open/onToggle；非受控：defaultOpen（内部 useState）
 * - 展开过渡：CSS grid rows 动画（grid-template-rows 0fr→1fr）——
 *   无 JS 测高，内容变化自然适应
 * - 标题行内容完全由调用方定制（children of header），指示器固定在前
 */

import { useId, useState, type ReactNode } from "react";

export interface CollapseProps {
  /** 标题行内容（▸/▾ 指示器之外的任意定制） */
  title: ReactNode;
  /** 折叠内容 */
  children: ReactNode;
  /** 受控展开态（提供则忽略 defaultOpen/内部态） */
  open?: boolean;
  /** 受控切换回调 */
  onToggle?: (open: boolean) => void;
  /** 非受控默认态（默认 false） */
  defaultOpen?: boolean;
  /** 无障碍名称（aria-label；缺省用标题文本） */
  label?: string;
  /** 外层容器附加类 */
  className?: string;
  /** 标题按钮附加类 */
  headerClassName?: string;
  /** 内容区附加类 */
  contentClassName?: string;
  /** 禁用（折叠功能关闭，常开展示） */
  disabled?: boolean;
}

export default function Collapse({
  title,
  children,
  open: controlled,
  onToggle,
  defaultOpen = false,
  label,
  className = "",
  headerClassName = "",
  contentClassName = "",
  disabled = false,
}: CollapseProps) {
  const [internal, setInternal] = useState(defaultOpen);
  const isControlled = controlled !== undefined;
  const open = disabled ? true : isControlled ? controlled : internal;
  const titleId = useId();

  const toggle = (): void => {
    if (disabled) return;
    const next = !open;
    if (!isControlled) setInternal(next);
    onToggle?.(next);
  };

  return (
    <div className={`overflow-hidden rounded-md border border-line bg-space/60 ${className}`}>
      <button
        type="button"
        onClick={toggle}
        aria-expanded={open}
        aria-controls={`${titleId}-region`}
        aria-label={label}
        className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-white/5 ${headerClassName}`}
      >
        <span aria-hidden="true" className="shrink-0 text-ink-3">
          {open ? "▾" : "▸"}
        </span>
        {title}
      </button>
      {/* grid rows 过渡：0fr↔1fr 平滑展开（无测高） */}
      <div
        id={`${titleId}-region`}
        role="region"
        aria-labelledby={titleId}
        className="grid transition-[grid-template-rows] duration-150 ease-out"
        style={{ gridTemplateRows: open ? "1fr" : "0fr" }}
      >
        <div className="min-h-0 overflow-hidden">
          <div className={`border-t border-line/60 ${contentClassName}`}>{children}</div>
        </div>
      </div>
    </div>
  );
}
