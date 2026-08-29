/**
 * 状态点（会话/任务/连接状态的单色指示）。
 *
 * - status：语义色映射（pending 灰 / running 传送门绿脉动 / success Rick 蓝灰 /
 *   error Morty 黄 / closed 灰 / 通用 ok/warn/err）
 * - pulse：脉动动画（rm-pulse——keyframes 定义在 theme.css）
 */

export type DotStatus =
  | "pending"
  | "running"
  | "success"
  | "error"
  | "closed"
  | "active"
  | "ok"
  | "warn"
  | "err"
  | "muted";

const DOT_STYLE: Record<DotStatus, { color: string; pulse: boolean; label: string }> = {
  pending: { color: "#5c6690", pulse: false, label: "pending" },
  running: { color: "#39ff88", pulse: true, label: "running" },
  active: { color: "#39ff88", pulse: true, label: "active" },
  success: { color: "#a6c8dd", pulse: false, label: "success" },
  ok: { color: "#39ff88", pulse: false, label: "ok" },
  error: { color: "#ffd54a", pulse: false, label: "error" },
  warn: { color: "#ffd54a", pulse: false, label: "warning" },
  err: { color: "#ff5d5d", pulse: false, label: "error" },
  closed: { color: "#5c6690", pulse: false, label: "closed" },
  muted: { color: "#5c6690", pulse: false, label: "" },
};

interface StatusDotProps {
  status: DotStatus;
  /** 覆盖默认 aria-label（如展示具体任务状态文案） */
  label?: string;
  /** 像素尺寸（默认 8） */
  size?: number;
}

export default function StatusDot({ status, label, size = 8 }: StatusDotProps) {
  const s = DOT_STYLE[status] ?? DOT_STYLE.muted;
  return (
    <span
      role="img"
      aria-label={label ?? s.label}
      title={label ?? s.label}
      className="inline-block shrink-0 rounded-full"
      style={{
        width: size,
        height: size,
        backgroundColor: s.color,
        boxShadow: s.pulse ? `0 0 6px ${s.color}` : undefined,
        animation: s.pulse ? "rm-pulse 1.8s ease-in-out infinite" : undefined,
      }}
    />
  );
}
