/**
 * 绿色传送门（Rick and Morty 标志性元素）。
 *
 * 同心圆旋涡 + 旋转动画（keyframes 定义在 theme.css 的 rm-portal-spin）。
 * - size：像素尺寸
 * - loading：true 时快速旋转（加载指示）；false 时缓慢旋转（静态装饰）
 */
interface PortalProps {
  size?: number;
  loading?: boolean;
}

export default function Portal({ size = 32, loading = false }: PortalProps) {
  const rings = [46, 34, 22, 11];
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      role="img"
      aria-label={loading ? "加载中" : "portal"}
      className="shrink-0"
    >
      <g
        style={{
          transformOrigin: "50% 50%",
          animation: `rm-portal-spin ${loading ? 1.2 : 6}s linear infinite`,
        }}
      >
        {rings.map((r, i) => (
          <circle
            key={r}
            cx="50"
            cy="50"
            r={r}
            fill="none"
            stroke={i % 2 === 0 ? "#39ff88" : "#2ee67a"}
            strokeOpacity={0.9 - i * 0.12}
            strokeWidth={i === 0 ? 3 : 2.5}
            strokeDasharray={
              i % 2 === 0 ? "none" : `${Math.round(r * 0.9)} ${Math.round(r * 0.4)}`
            }
          />
        ))}
      </g>
      <circle cx="50" cy="50" r="6" fill="#39ff88" opacity="0.95" />
    </svg>
  );
}
