import { useEffect, useState } from "react";

/**
 * 飞碟（Rick 的飞船意象，Rick and Morty 原画风，research-rm-art §三）。
 *
 * - size：像素尺寸（viewBox 64x40）
 * - flying：true 时悬停 + 暖黄光束 + 三色航灯错相闪烁（job 运行指示——task10 接线）；
 *   false 时静态（装饰/空态）
 * - 碟身银灰渐变（#c8cdd4 上 → #8d949e 下，替换旧紫色）
 * - 穹顶半透明淡蓝（#aee1f2，玻璃感）+ 白色高光描边
 * - 舱内并排 Rick 与 Morty 极简剪影（Rick=白大褂圆角梯形+淡蓝蓬发；Morty=黄T圆角矩形+深棕短发）
 * - 航灯三色交错：红 #ff5a5a / 绿 #7eff6a / 黄 #ffd54a
 * - keyframes（rm-saucer-hover / rm-beam）定义在 theme.css
 * - 移动端降级（job_43 二轮 KR7b）：窄屏（<768px）时 SMIL <animate> 不渲染
 *   （航灯静态）——CSS 管不到 SMIL，须组件内条件渲染；悬停/光束的 CSS 动画
 *   由 theme.css 窄屏媒体查询块关闭。桌面零回归。
 */
interface SaucerProps {
  size?: number;
  flying?: boolean;
}

/** 航灯三色交错（5 灯：红绿黄红绿） */
const LIGHT_COLORS = ["#ff5a5a", "#7eff6a", "#ffd54a", "#ff5a5a", "#7eff6a"];

/** 窄屏判定（<768px，与 routes/hooks.ts BREAKPOINTS.md 对齐）。
 * 订阅 matchMedia change（旋转/分屏跨断点实时切换）；Safari <14 无
 * addEventListener 时仅取初值（极端环境可接受，与 StarfieldBackground 同款防御）。 */
function useIsNarrowViewport(): boolean {
  const [narrow, setNarrow] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false;
    return window.matchMedia("(max-width: 767px)").matches;
  });
  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return;
    const mql = window.matchMedia("(max-width: 767px)");
    if (typeof mql.addEventListener !== "function") return;
    const onChange = () => setNarrow(mql.matches);
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, []);
  return narrow;
}

export default function Saucer({ size = 24, flying = false }: SaucerProps) {
  const narrow = useIsNarrowViewport();
  // 窄屏航灯静态（SMIL 无 CSS 降级路径）；悬停/光束 CSS 动画由 theme.css 关。
  const animateLights = flying && !narrow;
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
          fill="#ffe9a8"
          opacity="0.16"
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
        {/* 碟身（银灰金属，下缘深） */}
        <ellipse cx="32" cy="18" rx="26" ry="7" fill="#8d949e" />
        <ellipse cx="32" cy="16.5" rx="26" ry="6" fill="#c8cdd4" opacity="0.92" />
        {/* 碟沿反光细线 */}
        <ellipse cx="32" cy="15.8" rx="22" ry="4.6" fill="none" stroke="#eef1f5" strokeWidth="0.6" opacity="0.5" />

        {/* 舱内剪影（在碟身上、穹顶内）：左 Morty 右 Rick */}
        <g>
          {/* Morty：黄T + 深棕短发 */}
          <rect x="26.4" y="11.8" width="4.6" height="4" rx="1.1" fill="#fff874" />
          <path d="M26.4 12.6 q1.15 -1.9 4.6 -0.2 z" fill="#44281d" />
          {/* Rick：白大褂 + 淡蓝蓬发 */}
          <rect x="32.2" y="11.8" width="4.6" height="4" rx="1.1" fill="#f4f6f9" />
          <path d="M32.2 12.6 q1.15 -2 4.6 -0.2 z" fill="#9fe6e8" />
        </g>

        {/* 穹顶（半透明淡蓝玻璃 + 白色高光） */}
        <path d="M22 16 Q32 4 42 16 Z" fill="#aee1f2" opacity="0.45" />
        <path d="M25 13.5 Q32 6.5 39 13.5" fill="none" stroke="#ffffff" strokeWidth="0.9" opacity="0.65" />

        {/* 航灯（三色交错，flying 且非窄屏时错相闪烁；窄屏静态——theme.css 关不了 SMIL） */}
        {[12, 22, 32, 42, 52].map((x, idx) =>
          animateLights ? (
            <circle key={x} cx={x} cy={18.5} r="1.6" fill={LIGHT_COLORS[idx]} opacity="0.95">
              <animate
                attributeName="opacity"
                values="0.95;0.25;0.95"
                dur="1.4s"
                begin={`${(idx % 5) * 0.14}s`}
                repeatCount="indefinite"
              />
            </circle>
          ) : (
            <circle key={x} cx={x} cy={18.5} r="1.6" fill={LIGHT_COLORS[idx]} opacity={flying ? "0.95" : "0.5"} />
          )
        )}
      </g>
    </svg>
  );
}
