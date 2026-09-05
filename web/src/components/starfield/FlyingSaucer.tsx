/**
 * FlyingSaucer：背景中巡航飞行的飞船（Rick and Morty 原画风，research-rm-art §三）。
 *
 * canvas 纯绘制函数（由 StarfieldBackground 集成到同一画布）：
 * - 银灰金属碟身（#c8cdd4 → #8d949e）+ 半透明淡蓝玻璃穹顶 + 白色高光
 * - 舱内并排 Rick 与 Morty 极简剪影（深色描边）
 * - 底部暖黄光束（#ffe9a8，悬停脉动）
 * - 三色航灯（红/绿/黄）错相闪烁
 * - 悬停微摆（2-3s ease-in-out，±2.5% 位移）
 *
 * 巡航：位置由调用方按 spec.vx/vy 归一化速度推进（越界环绕）；staticMode 时静置。
 */

export interface FlyingSaucerSpec {
  /** 初始归一化位置（巡航起点） */
  x: number;
  y: number;
  /** 巡航速度（归一化单位/秒；x 主导水平飞行） */
  vx: number;
  vy: number;
  /** 尺寸（相对 min(w,h)，碟宽 ≈ size*2.2） */
  size: number;
  /** 整体透明度 0..1 */
  alpha: number;
  /** 悬停摆相位（错开多艘飞船的摆动） */
  phase: number;
  /** 镜像（朝左飞） */
  flipped?: boolean;
}

const BODY_LIGHT = "#c8cdd4";
const BODY_DARK = "#8d949e";
const GLASS = "rgba(174, 225, 242, 0.5)";
const BEAM = "#ffe9a8";
const LIGHTS = ["#ff5a5a", "#7eff6a", "#ffd54a"];

/**
 * 在 ctx 上绘制一艘巡航中的飞船。
 * @param ctx 目标 2D 上下文
 * @param spec 飞船规格（归一化坐标/速度）
 * @param w 视口宽（CSS px）
 * @param h 视口高（CSS px）
 * @param timeSec 动画时间（秒）
 * @param staticMode true = 静置固定（reduced-motion）
 */
export function drawFlyingSaucer(
  ctx: CanvasRenderingContext2D,
  spec: FlyingSaucerSpec,
  w: number,
  h: number,
  timeSec: number,
  staticMode: boolean,
): void {
  const alpha = Math.max(0, Math.min(1, spec.alpha));
  const S = spec.size * Math.min(w, h); // 基准尺度（碟高参考）
  const bob = staticMode ? 0 : Math.sin(timeSec * 2.8 + spec.phase) * 0.025;
  const cx = spec.x * w;
  const cy = (spec.y + bob) * h;
  const flip = spec.flipped ? -1 : 1;

  ctx.save();
  ctx.translate(cx, cy);
  ctx.scale(flip, 1);

  // ---- 底部暖黄光束（悬停脉动） ----
  const beamPulse = staticMode ? 0.16 : 0.1 + 0.08 * (0.5 + 0.5 * Math.sin(timeSec * 3.9 + spec.phase));
  ctx.globalAlpha = alpha * beamPulse;
  ctx.fillStyle = BEAM;
  ctx.beginPath();
  ctx.moveTo(-S * 0.5, S * 0.42);
  ctx.lineTo(S * 0.5, S * 0.42);
  ctx.lineTo(S * 0.95, S * 0.95);
  ctx.lineTo(-S * 0.95, S * 0.95);
  ctx.closePath();
  ctx.fill();

  // ---- 碟身（银灰金属） ----
  ctx.globalAlpha = alpha;
  ctx.fillStyle = BODY_DARK;
  ctx.beginPath();
  ctx.ellipse(0, S * 0.16, S * 1.1, S * 0.32, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillStyle = BODY_LIGHT;
  ctx.beginPath();
  ctx.ellipse(0, S * 0.1, S * 1.1, S * 0.3, 0, 0, Math.PI * 2);
  ctx.fill();
  // 碟沿反光
  ctx.strokeStyle = "#eef1f5";
  ctx.lineWidth = Math.max(0.5, S * 0.03);
  ctx.globalAlpha = alpha * 0.5;
  ctx.beginPath();
  ctx.ellipse(0, S * 0.06, S * 0.95, S * 0.24, 0, 0, Math.PI * 2);
  ctx.stroke();

  // ---- 舱内剪影：左 Morty 右 Rick（深色描边，玻璃后） ----
  ctx.globalAlpha = alpha;
  const hw = S * 0.5; // 舱半宽
  // Morty：黄T + 深棕短发
  ctx.fillStyle = "#fff874";
  roundRect(ctx, -hw * 0.85, -S * 0.42, hw * 0.62, S * 0.3, S * 0.05);
  ctx.fillStyle = "#44281d";
  roundRect(ctx, -hw * 0.85, -S * 0.5, hw * 0.62, S * 0.14, S * 0.05);
  // Rick：白大褂 + 淡蓝蓬发
  ctx.fillStyle = "#f4f6f9";
  roundRect(ctx, -hw * 0.1, -S * 0.42, hw * 0.62, S * 0.3, S * 0.05);
  ctx.fillStyle = "#9fe6e8";
  roundRect(ctx, -hw * 0.1, -S * 0.52, hw * 0.62, S * 0.14, S * 0.05);

  // ---- 玻璃穹顶（半透明淡蓝 + 白高光） ----
  ctx.globalAlpha = alpha * 0.55;
  ctx.fillStyle = GLASS;
  ctx.beginPath();
  ctx.moveTo(-hw * 1.15, -S * 0.28);
  ctx.quadraticCurveTo(0, -S * 1.0, hw * 1.15, -S * 0.28);
  ctx.closePath();
  ctx.fill();
  ctx.globalAlpha = alpha * 0.6;
  ctx.strokeStyle = "#ffffff";
  ctx.lineWidth = Math.max(0.5, S * 0.035);
  ctx.beginPath();
  ctx.moveTo(-hw * 0.55, -S * 0.44);
  ctx.quadraticCurveTo(0, -S * 0.82, hw * 0.55, -S * 0.44);
  ctx.stroke();

  // ---- 三色航灯（错相闪烁） ----
  for (let i = 0; i < 3; i++) {
    const x = (i - 1) * hw * 0.95;
    const blink = staticMode ? 0.7 : 0.35 + 0.6 * Math.abs(Math.sin(timeSec * 4.4 + i * 1.2 + spec.phase));
    ctx.globalAlpha = alpha * blink;
    ctx.fillStyle = LIGHTS[i];
    ctx.beginPath();
    ctx.arc(x, S * 0.3, Math.max(0.7, S * 0.06), 0, Math.PI * 2);
    ctx.fill();
  }

  ctx.restore();
  ctx.globalAlpha = 1;
}

/** ctx 圆角矩形路径（不填充）。 */
function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
): void {
  const rr = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + rr, y);
  ctx.lineTo(x + w - rr, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + rr);
  ctx.lineTo(x + w, y + h - rr);
  ctx.quadraticCurveTo(x + w, y + h, x + w - rr, y + h);
  ctx.lineTo(x + rr, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - rr);
  ctx.lineTo(x, y + rr);
  ctx.quadraticCurveTo(x, y, x + rr, y);
  ctx.closePath();
}
