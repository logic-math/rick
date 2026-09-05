/**
 * Planet：cel-shade 风格星球背景元素（Rick and Morty 原画风，research-rm-art §四）。
 *
 * canvas 纯绘制函数（由 StarfieldBackground 集成到同一画布）：
 * - 扁平两段明暗（单一左上光源的亮/暗硬边界，非多级渐变——剧内行星风格）
 * - 默认青色系（亮 #5cc8c2 / 主 #3aa6a0 / 暗 #1d4f4e）；可选浅紫系
 * - 表面环形纹理（自转偏移）+ 暗斑
 * - 大气辉光（外部径向渐变，融入背景）
 * - 可选土星式细环（暖金 #e8c96a + 环内暗缝 #a8843c）
 *
 * 动画：自转由调用方传入 timeSec 驱动（极缓 >30s/圈，纹理漂移）；
 * staticMode=true 时固定相位（reduced-motion / 静态绘制）。
 */

export interface PlanetSpec {
  x: number; // 归一化 0..1（相对视口宽）
  y: number; // 归一化 0..1（相对视口高）
  r: number; // 归一化半径（相对 min(w,h)）
  alpha: number; // 整体透明度 0..1
  ring?: boolean; // 土星式细环
  phase: number; // 自转初始相位
  speed: number; // 自转速度（弧度/秒；背景行星极缓 0.02-0.08）
  /** 球体主色（默认 R&M 青 #3aa6a0） */
  baseColor?: string;
  /** 球体亮面色（默认 #5cc8c2） */
  highlightColor?: string;
  /** 球体暗面色（默认 #1d4f4e） */
  shadowColor?: string;
  /** 细环亮色（默认暖金 #e8c96a） */
  ringLight?: string;
  /** 细环暗缝色（默认 #a8843c） */
  ringDark?: string;
}

/**
 * 在 ctx 上绘制一颗 cel-shade 星球。
 * @param ctx 目标 2D 上下文（已 clear 或叠加绘制）
 * @param spec 星球规格（归一化坐标）
 * @param w 视口宽（CSS px）
 * @param h 视口高（CSS px）
 * @param timeSec 动画时间（秒）
 * @param staticMode true = 固定相位（reduced-motion / 静态绘制）
 */
export function drawPlanet(
  ctx: CanvasRenderingContext2D,
  spec: PlanetSpec,
  w: number,
  h: number,
  timeSec: number,
  staticMode: boolean,
): void {
  const radius = spec.r * Math.min(w, h);
  const cx = spec.x * w;
  const cy = spec.y * h;
  const alpha = Math.max(0, Math.min(1, spec.alpha));
  const rot = staticMode ? 0 : timeSec * spec.speed + spec.phase;

  const base = spec.baseColor ?? "#3aa6a0";
  const highlight = spec.highlightColor ?? "#5cc8c2";
  const shadow = spec.shadowColor ?? "#1d4f4e";
  const ringLight = spec.ringLight ?? "#e8c96a";
  const ringDark = spec.ringDark ?? "#a8843c";

  // ---- 大气辉光（外部径向渐变，融入背景） ----
  const glow = ctx.createRadialGradient(cx, cy, radius * 0.5, cx, cy, radius * 1.7);
  glow.addColorStop(0, hexToRgba(base, alpha * 0.28));
  glow.addColorStop(1, hexToRgba(base, 0));
  ctx.fillStyle = glow;
  ctx.beginPath();
  ctx.arc(cx, cy, radius * 1.7, 0, Math.PI * 2);
  ctx.fill();

  // ---- cel-shade 两段（硬边界）----
  // 主体圆
  ctx.globalAlpha = alpha;
  ctx.fillStyle = base;
  ctx.beginPath();
  ctx.arc(cx, cy, radius, 0, Math.PI * 2);
  ctx.fill();

  // 裁剪在球体内，叠加暗部（右下）与亮部（左上）→ 单一左上光源的硬边界
  ctx.save();
  ctx.beginPath();
  ctx.arc(cx, cy, radius, 0, Math.PI * 2);
  ctx.clip();

  // 暗部：右下偏移大圆（覆盖右下半）
  ctx.fillStyle = shadow;
  ctx.beginPath();
  ctx.arc(cx + radius * 0.3, cy + radius * 0.36, radius * 0.95, 0, Math.PI * 2);
  ctx.fill();

  // 亮部：左上偏移大圆（覆盖左上）
  ctx.fillStyle = highlight;
  ctx.beginPath();
  ctx.arc(cx - radius * 0.32, cy - radius * 0.34, radius * 0.9, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
  ctx.globalAlpha = 1;

  // ---- 表面环形纹理（自转偏移）+ 暗斑 ----
  ctx.save();
  ctx.translate(cx, cy);
  ctx.rotate(rot);
  ctx.globalAlpha = alpha * 0.55;
  ctx.strokeStyle = hexToRgba(shadow, 0.6);
  ctx.lineWidth = Math.max(1, radius * 0.024);
  for (let i = -2; i <= 2; i++) {
    ctx.beginPath();
    ctx.ellipse(0, 0, radius * (0.92 + 0.035 * i), radius * (0.52 + 0.018 * i), 0, 0, Math.PI * 2);
    ctx.stroke();
  }
  // 暗斑（一两个，形成行星表面细节）
  ctx.fillStyle = hexToRgba(shadow, 0.5);
  ctx.beginPath();
  ctx.arc(radius * 0.3, -radius * 0.18, radius * 0.2, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.arc(-radius * 0.34, radius * 0.3, radius * 0.13, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
  ctx.globalAlpha = 1;

  // ---- 土星式细环（可选；后半段被球体遮住 → 用整环低透明度即可） ----
  if (spec.ring) {
    ctx.save();
    ctx.translate(cx, cy);
    ctx.rotate(-0.32 + rot * 0.22);
    // 主环（暖金）
    ctx.globalAlpha = alpha * 0.55;
    ctx.strokeStyle = ringLight;
    ctx.lineWidth = Math.max(1.2, radius * 0.055);
    ctx.beginPath();
    ctx.ellipse(0, 0, radius * 1.42, radius * 0.48, 0, 0, Math.PI * 2);
    ctx.stroke();
    // 环内暗缝（层次）
    ctx.strokeStyle = ringDark;
    ctx.lineWidth = Math.max(0.8, radius * 0.02);
    ctx.beginPath();
    ctx.ellipse(0, 0, radius * 1.28, radius * 0.435, 0, 0, Math.PI * 2);
    ctx.stroke();
    // 外缘细亮线
    ctx.strokeStyle = hexToRgba(ringLight, 0.85);
    ctx.lineWidth = Math.max(0.6, radius * 0.012);
    ctx.beginPath();
    ctx.ellipse(0, 0, radius * 1.5, radius * 0.51, 0, 0, Math.PI * 2);
    ctx.stroke();
    ctx.restore();
    ctx.globalAlpha = 1;
  }
}

/** #rrggbb + alpha → rgba() 字符串（无解析失败时回退 #fff）。 */
function hexToRgba(hex: string, alpha: number): string {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex);
  if (!m) return `rgba(255,255,255,${alpha})`;
  const n = parseInt(m[1], 16);
  const r = (n >> 16) & 0xff;
  const g = (n >> 8) & 0xff;
  const b = n & 0xff;
  return `rgba(${r},${g},${b},${alpha})`;
}
