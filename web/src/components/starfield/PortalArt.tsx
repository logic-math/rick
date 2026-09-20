/**
 * PortalArt：绿色传送门背景元素（Rick and Morty 原画风）。
 *
 * v2「液态涡旋」重绘（用户反馈：漫画里不是离散的悬臂，而是一坨像液体一样的
 * 绿色漩涡；且应放远一点、淡一点——星空主题为主，传送门只是远处暗示）：
 * - **多层差速对数螺线**（5 层，层间转速递减 12% → 剪切拖曳的液体感），
 *   每层宽描边、低透明度、互相叠加模糊成连续的绿色流体（不再有清晰可数的臂）
 * - canvas blur 滤镜可用时整体柔化（不可用则靠宽线叠层自然发糊——降级安全）
 * - 流体核心：白绿径向渐变缓慢脉动；外缘绿色辉光极淡
 * - 无锯齿粒子、无硬高光（离散元素全部移除——液体不该有颗粒边）
 *
 * 动画：timeSec 驱动（speed≈0.5 rad/s 慢转，更“远”更安静）；staticMode 固定相位。
 * 尺寸由调用方（StarfieldBackground）控制：建议 r≈0.10-0.13、alpha≈0.3。
 */

export interface PortalSpec {
  x: number; // 归一化 0..1（相对视口宽）
  y: number; // 归一化 0..1（相对视口高）
  r: number; // 归一化半径（相对 min(w,h)）
  alpha: number; // 整体透明度 0..1
  /** 涡旋旋转速度（弧度/秒；默认 1.1 ≈ 5.7s/圈——安静但肉眼可见的流涡） */
  speed?: number;
}

// R&M 传送门绿系（tvthemes：绿 #97ce4c）
const GREEN_GLOW = "151, 206, 76"; // #97ce4c
const GREEN_BRIGHT = "#d4f5a8";
const GREEN_MAIN = "#97ce4c";
const GREEN_DEEP = "#3d6b1a";
const CORE_WHITE = "#f2ffd8";

const TWO_PI = Math.PI * 2;

/** 差速层数（越多越“糊”——液体感来自重叠） */
const LAYERS = 5;

/**
 * 在 ctx 上绘制**液态**绿色传送门（多层差速螺线叠成的流体涡旋）。
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
  const speed = spec.speed ?? 1.1;
  const rot0 = staticMode ? 0.3 : timeSec * speed;

  // ---- 远处的底辉（极淡的绿雾，暗示而非宣告）----
  const glow = ctx.createRadialGradient(cx, cy, radius * 0.15, cx, cy, radius * 1.35);
  glow.addColorStop(0, `rgba(${GREEN_GLOW}, ${alpha * 0.14})`);
  glow.addColorStop(1, `rgba(${GREEN_GLOW}, 0)`);
  ctx.fillStyle = glow;
  ctx.beginPath();
  ctx.arc(cx, cy, radius * 1.35, 0, TWO_PI);
  ctx.fill();

  // 液体感不靠 canvas blur 滤镜（每帧 filter 软光栅化极贵：无头实测 17ms→167ms/帧，
  // 低端真机同样有风险）——改用「宽描边 + 低透明 + 多层叠」的视觉模糊：重叠的宽线
  // 半透明叠加在视觉上就是连续流体，零滤镜成本。

  // ---- 多层差速对数螺线（液体主体）----
  // 对数螺线 r = a·e^(bθ)：外张比阿基米德更“卷”，叠层后是连续的漩涡流。
  // 每层：转速 rot0·(1-k·0.12)（差速→剪切）、椭圆率微变（0.82+k·0.04，破除
  // 刚体旋转感）、透明度 0.10-0.18、宽描边（radius·0.14-0.20）。
  for (let k = 0; k < LAYERS; k++) {
    const layerRot = rot0 * (1 - k * 0.12) + (k * TWO_PI) / (LAYERS * 1.3);
    const squash = 0.82 + k * 0.045;
    // 层透明度：中间层最亮（内外层薄）→ 体积感
    const layerAlpha = alpha * (0.09 + 0.11 * Math.sin(((k + 0.5) / LAYERS) * Math.PI));
    // 每层 2 条对角螺线（叠加后 10 条痕，模糊即流体）
    for (let arm = 0; arm < 2; arm++) {
      const theta0 = layerRot + (arm * Math.PI) + k * 0.7;
      const TURNS = Math.PI * 1.9;
      const steps = 44;
      ctx.beginPath();
      for (let i = 0; i <= steps; i++) {
        const t = i / steps;
        const theta = t * TURNS;
        const rr = radius * (0.14 + 0.86 * Math.pow(t, 0.75));
        const px = cx + Math.cos(theta0 + theta) * rr;
        const py = cy + Math.sin(theta0 + theta) * rr * squash;
        if (i === 0) ctx.moveTo(px, py);
        else ctx.lineTo(px, py);
      }
      // 沿半径的色渐变（内亮 → 外深）
      const grad = ctx.createLinearGradient(
        cx - radius, cy - radius, cx + radius, cy + radius,
      );
      grad.addColorStop(0, GREEN_BRIGHT);
      grad.addColorStop(0.5, GREEN_MAIN);
      grad.addColorStop(1, GREEN_DEEP);
      ctx.strokeStyle = grad;
      ctx.lineWidth = radius * (0.22 - k * 0.02); // 内层粗外层细（宽线叠出流体感）
      ctx.globalAlpha = layerAlpha;
      ctx.lineCap = "round";
      ctx.stroke();
    }
  }
  ctx.globalAlpha = 1;

  // ---- 流体核心（白绿渐变 + 极缓脉动；远处的星系核感）----
  const pulse = staticMode ? 1 : 1 + 0.25 * Math.sin(timeSec * 2.8); // ~2.2s/周期，可感的呼吸
  const coreR = radius * 0.34 * pulse;
  const core = ctx.createRadialGradient(cx, cy, 0, cx, cy, coreR * 2);
  core.addColorStop(0, hexToRgba(CORE_WHITE, alpha * 0.5));
  core.addColorStop(0.4, hexToRgba(GREEN_BRIGHT, alpha * 0.28));
  core.addColorStop(1, `rgba(${GREEN_GLOW}, 0)`);
  ctx.fillStyle = core;
  ctx.beginPath();
  ctx.arc(cx, cy, coreR * 2, 0, TWO_PI);
  ctx.fill();

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
