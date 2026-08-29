/**
 * 主题按钮（R&M 配色）。
 *
 * - variant：primary（传送门绿描边+填充 hover）/ ghost（幽灵）/ danger（红）
 * - size：sm / md
 * - loading：内置传送门旋涡小指示 + 禁用
 */

import type { ButtonHTMLAttributes, ReactNode } from "react";
import Portal from "../starfield/Portal";

export type ButtonVariant = "primary" | "ghost" | "danger";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: "sm" | "md";
  loading?: boolean;
  children: ReactNode;
}

const VARIANT_CLASS: Record<ButtonVariant, string> = {
  primary:
    "border-portal/60 bg-portal-soft text-portal hover:bg-portal/20 hover:border-portal",
  ghost:
    "border-line bg-transparent text-ink-2 hover:bg-white/5 hover:text-ink",
  danger:
    "border-danger/50 bg-danger/10 text-danger hover:bg-danger/20 hover:border-danger",
};

const SIZE_CLASS: Record<"sm" | "md", string> = {
  sm: "px-2.5 py-1 text-xs gap-1.5",
  md: "px-3.5 py-1.5 text-sm gap-2",
};

export default function Button({
  variant = "ghost",
  size = "md",
  loading = false,
  disabled,
  className = "",
  children,
  ...rest
}: ButtonProps) {
  const isDisabled = disabled || loading;
  return (
    <button
      type="button"
      disabled={isDisabled}
      className={`inline-flex items-center justify-center rounded-lg border font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${VARIANT_CLASS[variant]} ${SIZE_CLASS[size]} ${className}`}
      {...rest}
    >
      {loading && <Portal size={size === "sm" ? 12 : 14} loading />}
      {children}
    </button>
  );
}
