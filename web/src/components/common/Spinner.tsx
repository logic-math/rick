/**
 * 加载指示（传送门旋涡——Portal 组件复用）。
 *
 * - label：可选文案（如「加载 jobs…」）
 * - center：true 时占满父容器居中（页面级）；false 时行内（列表项尾部）
 */

import Portal from "../starfield/Portal";

interface SpinnerProps {
  size?: number;
  label?: string;
  center?: boolean;
}

export default function Spinner({ size = 22, label, center = false }: SpinnerProps) {
  const core = (
    <span className="inline-flex items-center gap-2 text-ink-3">
      <Portal size={size} loading />
      {label && <span className="text-xs">{label}</span>}
    </span>
  );
  if (!center) return core;
  return <div className="flex min-h-24 items-center justify-center py-8">{core}</div>;
}
