/**
 * PortalArt：绿色传送门背景元素（Rick and Morty 原画风，research-rm-art §二/6.2）。
 *
 * canvas 纯绘制函数（由 StarfieldBackground 集成到同一画布）：
 * - 3-5 条阿基米德螺旋臂（vortex disc 核心视觉，替换旧同心椭圆层）
 * - 颜色沿臂渐变：内亮黄绿 #d4f5a8 → 主 #97ce4c → 外深绿 #4c7a1f
 * - 外缘 12-20 个锯齿能量粒子（沿圆周旋转 + 径向微颤）
 * - 中心白绿核心 #f2ffd8 → #d4f5a8（~1.2s 脉动）
 * - 外部辉光（#97ce4c 径向渐变）+ 1-2 段弧形高光扫过
 *
 * 动画：涡旋旋转由调用方传入 timeSec 驱动（speed≈0.9 rad/s ≈ 7s/圈，背景氛围）；
 * staticMode=true 时固定相位（reduced-motion / 静态绘制）。
 */

export interface PortalSpec {
  x: number; // 归一化 0..1（相对视口宽）
  y: number; // 归一化 0..1（相对视口高）
  r: number; // 归一化半径（相对 min(w,h)）
  alpha: number; // 整体透明度 0..1
  /** 涡旋旋转速度（弧度/秒；默认 0.9 ≈ 7s/圈） */
  speed?: number;
  /** 螺旋臂数量（3-5） */
  arms?: number;
}

// R&M 传送门绿系（tvthemes：绿 #97ce4c）
const GREEN_GLOW = "151, 206, 76"; // #97ce4c
const GREEN_BRIGHT = "#d4f5a8";
const GREEN_MAIN = "#97ce4c";
const GREEN_DARK = "#4c7a1f";
const GREEN_PARTICLE = "#b8e94e";
const CORE_WHITE = "#f2ffd8";
const HIGHLIGHT = "#eaffc9";

const TWO_PI = Math.PI * 2;

/**
 * 在 ctx 上绘制绿色传送门（螺旋涡旋盘面）。
 * @param ctx 目标 2D 上下文
 * @param spec 传送门规格（归一化坐标）
 * @param w 视口宽（CSS px）
 * @param h 视口高（CSS px）
 * @param timeSec 动画时间（秒）
 * @param staticMode true = 固定相位（reduced-motion / 静态绘制）
 */
export function drawPortal(
  ctx: CanvasRenderingContext2D,
  spec: PortalSpec,
  w: number,
  h: number,
  timeSec: number,
  staticMode: boolean,
): void {
  const radius = spec.r * Math.min(w, h);
  const cx = spec.x * w;
  const cy = spec.y * h;
  const alpha = Math.max(0, Math.min(1, spec.alpha));
  const speed = spec.speed ?? 0.9;
  // 臂数默认 3（螺旋清晰可见；5 臂 + 宽描边会糊成实心圆——job_36 像素实测）
  const arms = Math.max(3, Math.min(5, spec.arms ?? 3));
  const rot = staticMode ? 0.15 : timeSec * speed;

  // ---- 外部辉光（#97ce4c 径向渐变，融入背景） ----
  const glow = ctx.createRadialGradient(cx, cy, radius * 0.2, cx, cy, radius * 1.6);
  glow.addColorStop(0, `rgba(${GREEN_GLOW}, ${alpha * 0.24})`);
  glow.addColorStop(1, `rgba(${GREEN_GLOW}, 0)`);
  ctx.fillStyle = glow;
  ctx.beginPath();
  ctx.arc(cx, cy, radius * 1.6, 0, TWO_PI);
  ctx.fill();

  // ---- 3-5 条阿基米德螺旋臂（核心视觉）----
  // 半径从 0.12R 线性展开到 1.0R（旧实现 `0.12 * (θ/T)^0.85` 最大只有 0.12R，
  // 全部挤在中心 → 实心团块、看不出螺旋——像素实测 ███ 全实）。
  // 圈数 2.2π（臂间留明显空隙）、线宽相对大半径调细，保证螺旋线可见。
  const SPIRAL_TURNS = Math.PI * 2.2; // 展开角（弧度）
  for (let arm = 0; arm < arms; arm++) {
    const theta0 = (arm * TWO_PI) / arms + rot; // 整体绕中心旋转（逆时针向心）
    // 构建螺旋路径点（y 压扁 0.9 → 椭圆视角）
    const pts: Array<{ x: number; y: number }> = [];
    const steps = 48;
    for (let i = 0; i <= steps; i++) {
      const theta = (i / steps) * SPIRAL_TURNS;
      // 0.12R → 1.0R 的平滑展开（幂指数 0.9：中心密、外缘疏，剧内涡旋观感）
      const rr = radius * (0.12 + 0.88 * Math.pow(theta / SPIRAL_TURNS, 0.9));
      pts.push({
        x: cx + Math.cos(theta0 + theta) * rr,
        y: cy + Math.sin(theta0 + theta) * rr * 0.9,
      });
    }
    // 分两段描边：内段粗亮（#d4f5a8→#97ce4c），外段细暗（#97ce4c→#4c7a1f）
    const split = Math.floor(pts.length * 0.55);
    const drawSegment = (from: number, to: number, widthMul: number, c0: string, c1: string) => {
      if (to <= from) return;
      const a = pts[from];
      const b = pts[Math.min(to, pts.length - 1)];
      const grad = ctx.createLinearGradient(a.x, a.y, b.x, b.y);
      grad.addColorStop(0, c0);
      grad.addColorStop(1, c1);
      ctx.strokeStyle = grad;
      ctx.lineWidth = Math.max(1.2, radius * 0.06 * widthMul);
      ctx.globalAlpha = alpha;
      ctx.beginPath();
      ctx.moveTo(a.x, a.y);
      for (let i = from + 1; i <= to && i < pts.length; i++) {
        ctx.lineTo(pts[i].x, pts[i].y);
      }
      ctx.stroke();
      ctx.globalAlpha = alpha * 0.5;
      // 更细的同路径亮描边（增加「果冻」光感）
      ctx.lineWidth = Math.max(0.6, radius * 0.02 * widthMul);
      ctx.stroke();
      ctx.globalAlpha = 1;
    };
    drawSegment(0, split, 1, GREEN_BRIGHT, GREEN_MAIN);
    drawSegment(split, pts.length - 1, 0.4, GREEN_MAIN, GREEN_DARK);
  }

  // ---- 外缘锯齿能量粒子（沿圆周旋转 + 径向微颤）----
  const particleCount = 15;
  for (let i = 0; i < particleCount; i++) {
    const wobble = staticMode ? 0 : Math.sin(timeSec * (1.4 + (i % 3) * 0.4) + i * 1.7) * 0.06;
    const a = rot + (i * TWO_PI) / particleCount;
    const rr = radius * (0.95 + wobble);
    const px = cx + Math.cos(a) * rr;
    const py = cy + Math.sin(a) * rr * 0.9;
    // 确定性伪随机大小/亮度（避免每帧闪烁）
    const det = (i * 37) % 13;
    const size = Math.max(0.8, radius * (0.012 + det * 0.0012));
    ctx.globalAlpha = alpha * (0.35 + det * 0.03);
    ctx.fillStyle = GREEN_PARTICLE;
    ctx.beginPath();
    ctx.arc(px, py, size, 0, TWO_PI);
    ctx.fill();
  }
  ctx.globalAlpha = 1;

  // ---- 中心核心（白绿渐变 + 脉动）----
  const pulse = staticMode ? 1 : 1 + 0.15 * Math.sin(timeSec * 5.2); // ~1.2s/周期
  const coreR = radius * 0.18 * pulse;
  const core = ctx.createRadialGradient(cx, cy, 0, cx, cy, coreR * 2.2);
  core.addColorStop(0, hexToRgba(CORE_WHITE, alpha * 0.95));
  core.addColorStop(0.35, hexToRgba(GREEN_BRIGHT, alpha * 0.55));
  core.addColorStop(1, `rgba(${GREEN_GLOW}, 0)`);
  ctx.fillStyle = core;
  ctx.beginPath();
  ctx.arc(cx, cy, coreR * 2.2, 0, TWO_PI);
  ctx.fill();

  // ---- 高光扫过（1-2 段弧形，低透明）----
  for (let k = 0; k < 2; k++) {
    const a0 = rot * 1.6 + (k * Math.PI) / 1 + 0.4;
    ctx.globalAlpha = alpha * 0.3;
    ctx.strokeStyle = HIGHLIGHT;
    ctx.lineWidth = Math.max(1, radius * 0.03);
    ctx.beginPath();
    ctx.arc(cx, cy, radius * 0.55, a0, a0 + 0.9);
    ctx.stroke();
  }
  ctx.globalAlpha = 1;
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
