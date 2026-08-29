/**
 * 飞碟（Rick 的飞船意象）。
 *
 * - size：像素尺寸（viewBox 64x40）
 * - flying：true 时悬停 + 光束 + 航灯闪烁（job 运行指示——task10 接线）；
 *   false 时静态（装饰/空态）
 * - keyframes（rm-saucer-hover / rm-beam）定义在 theme.css
 */
interface SaucerProps {
  size?: number;
  flying?: boolean;
}

export default function Saucer({ size = 24, flying = false }: SaucerProps) {
  return (
    <svg
      width={size}
      height={(size * 40) / 64}
      viewBox="0 0 64 40"
      role="img"
      aria-label={flying ? "任务运行中" : "saucer"}
      className="shrink-0"
    >
      {flying && (
        <polygon
          points="24,24 40,24 52,38 12,38"
          fill="#39ff88"
          opacity="0.18"
          style={{ animation: "rm-beam 1.6s ease-in-out infinite" }}
        />
      )}
      <g
        style={
          flying
            ? { animation: "rm-saucer-hover 2.2s ease-in-out infinite" }
            : undefined
        }
      >
        {/* 穹顶 */}
        <path d="M22 16 Q32 4 42 16 Z" fill="#a6c8dd" opacity="0.9" />
        {/* 碟身 */}
        <ellipse cx="32" cy="18" rx="26" ry="7" fill="#5d3fd3" />
        <ellipse cx="32" cy="16.5" rx="26" ry="6" fill="#8b6ff0" opacity="0.85" />
        {/* 航灯 */}
        {[12, 22, 32, 42, 52].map((x) =>
          flying ? (
            <circle key={x} cx={x} cy={18.5} r="1.6" fill="#ffd54a" opacity="0.9">
              <animate
                attributeName="opacity"
                values="0.9;0.25;0.9"
                dur="1.4s"
                begin={`${((x % 10) / 10) * 0.14}s`}
                repeatCount="indefinite"
              />
            </circle>
          ) : (
            <circle key={x} cx={x} cy={18.5} r="1.6" fill="#ffd54a" opacity="0.55" />
          )
        )}
      </g>
    </svg>
  );
}
