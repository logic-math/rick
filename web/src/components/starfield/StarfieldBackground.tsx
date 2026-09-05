import { useEffect, useRef } from "react";
import { drawPlanet, type PlanetSpec } from "./Planet";
import { drawPortal, type PortalSpec } from "./PortalArt";
import { drawFlyingSaucer, type FlyingSaucerSpec } from "./FlyingSaucer";

/**
 * 动态星空背景（Rick and Morty 原画风，research-rm-art §六）。
 *
 * - canvas 单层实现：
 *   · 深蓝近黑渐变底（#0b1226 → #04060c，替换纯黑/紫底）
 *   · 静态星点：90% 冷白 #e8ecf8 + 10% 暖黄 #fff4c2（稍大）+ 少量 4 芒十字星光
 *   · 星云层：紫 #7b4ea8 / 青绿 #2f5d5f / 橙 #c96f4a 低透明径向渐变（粉紫橙 + 青绿两类）
 *   · 流星：每 6-15s 一颗，斜线拖尾渐隐（保持现状）
 *   · cel-shade 星球（Planet）：1-2 颗，角落偏置，极缓自转，暖金土星式环
 *   · 螺旋涡旋传送门（PortalArt）：1 个，右下视觉锚点，阿基米德螺旋臂旋转
 *   · 巡航飞船（FlyingSaucer）：1-2 艘在背景飞行（舱内 Rick & Morty 剪影）
 * - 性能：动画帧率限 ~30fps；document 不可见时暂停（visibilitychange）
 * - prefers-reduced-motion: reduce → 静态星点 + 静态星球/传送门/飞船（固定相位）
 */
interface Star {
  x: number; // 归一化 0..1
  y: number;
  r: number; // 半径 px
  base: number; // 基础亮度
  phase: number;
  speed: number;
  warm: boolean; // 暖黄星（R&M 暖色点缀）
  cross: boolean; // 4 芒十字星光
}

interface Meteor {
  x: number; // 归一化 0..1
  y: number;
  vx: number; // 归一化单位/秒
  vy: number;
  life: number; // 1 → 0
}

interface StarfieldBackgroundProps {
  /** 基准星点密度（1280x720 视口下的目标数量，实际按面积缩放） */
  density?: number;
  className?: string;
}

const FRAME_MS = 1000 / 30; // ~30fps
const METEOR_MIN_MS = 6000;
const METEOR_MAX_MS = 15000;
const METEOR_TAIL_PX = 90;

// 背景星球（cel-shade 双段）：一颗大（右上带暖金环）+ 一颗小（左下）
const PLANETS: PlanetSpec[] = [
  { x: 0.84, y: 0.14, r: 0.14, alpha: 0.5, ring: true, phase: 0.6, speed: 0.05 },
  { x: 0.08, y: 0.86, r: 0.07, alpha: 0.4, ring: false, phase: 2.1, speed: 0.04 },
];

// 背景传送门（螺旋涡旋盘面，偏右下视觉锚点；大尺寸、醒目的青柠绿涡旋）
const PORTAL: PortalSpec = { x: 0.8, y: 0.72, r: 0.26, alpha: 0.6, speed: 0.9 };

// 背景巡航飞船（1-2 艘，舱内 Rick & Morty）
const FLYING_SAUCERS: FlyingSaucerSpec[] = [
  { x: 0.15, y: 0.3, vx: 0.008, vy: 0.0012, size: 0.09, alpha: 0.4, phase: 0.2, flipped: true },
  { x: 0.7, y: 0.55, vx: -0.005, vy: 0.0008, size: 0.06, alpha: 0.32, phase: 2.6 },
];

// 星云层（低透明径向渐变；R&M 粉紫橙 + 青绿两类）
const NEBULAE = [
  { x: 0.2, y: 0.32, r: 0.5, color: "123, 78, 168", alpha: 0.13 }, // 紫 #7b4ea8
  { x: 0.78, y: 0.6, r: 0.45, color: "47, 93, 95", alpha: 0.1 }, // 青绿 #2f5d5f
  { x: 0.55, y: 0.2, r: 0.3, color: "201, 111, 74", alpha: 0.07 }, // 橙 #c96f4a 点缀
];

const BG_TOP = "#0b1226"; // 藏青近黑（顶）
const BG_BOTTOM = "#04060c"; // 近黑（底）
const STAR_COLD = "#e8ecf8";
const STAR_WARM = "#fff4c2";

export default function StarfieldBackground({
  density = 120,
  className = "",
}: StarfieldBackgroundProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const reduced = window.matchMedia(
      "(prefers-reduced-motion: reduce)"
    ).matches;
    let stars: Star[] = [];
    let meteors: Meteor[] = [];
    let raf = 0;
    let visible = true;
    let lastFrame = 0;
    let nextMeteorAt = 0;
    let dpr = 1;
    let bgGradient: CanvasGradient | null = null;

    const seedStars = () => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      const count = Math.max(
        40,
        Math.round((density * w * h) / (1280 * 720))
      );
      stars = [];
      for (let i = 0; i < count; i++) {
        // 约 8-10% 暖黄星（大一点），约 4% 十字星光
        const warm = i % 11 === 5;
        const cross = i % 25 === 0;
        stars.push({
          x: Math.random(),
          y: Math.random(),
          r: (warm ? 0.8 : 0.4) + Math.random() * 1.4,
          base: 0.25 + Math.random() * 0.65,
          phase: Math.random() * Math.PI * 2,
          speed: 0.4 + Math.random() * 1.2,
          warm,
          cross,
        });
      }
    };

    const buildBg = () => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      bgGradient = ctx.createLinearGradient(0, 0, 0, h);
      bgGradient.addColorStop(0, BG_TOP);
      bgGradient.addColorStop(1, BG_BOTTOM);
      ctx.fillStyle = bgGradient;
      ctx.fillRect(0, 0, w, h);
    };

    const drawNebulae = (staticMode: boolean) => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      // 极缓漂移（背景氛围，reduced-motion 固定）
      const driftX = staticMode ? 0 : Math.sin(performance.now() / 200000) * 0.02;
      for (const n of NEBULAE) {
        const r = n.r * Math.min(w, h);
        const cx = (n.x + driftX) * w;
        const cy = n.y * h;
        const g = ctx.createRadialGradient(cx, cy, r * 0.1, cx, cy, r);
        g.addColorStop(0, `rgba(${n.color}, ${n.alpha})`);
        g.addColorStop(1, `rgba(${n.color}, 0)`);
        ctx.fillStyle = g;
        ctx.beginPath();
        ctx.arc(cx, cy, r, 0, Math.PI * 2);
        ctx.fill();
      }
    };

    const drawStars = (timeSec: number, animated: boolean) => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);
      if (bgGradient) {
        ctx.fillStyle = bgGradient;
        ctx.fillRect(0, 0, w, h);
      }
      for (const s of stars) {
        const twinkle = animated
          ? 0.55 + 0.45 * Math.sin(s.phase + timeSec * s.speed)
          : 1;
        ctx.globalAlpha = Math.min(1, s.base * twinkle);
        ctx.fillStyle = s.warm ? STAR_WARM : STAR_COLD;
        ctx.beginPath();
        ctx.arc(s.x * w, s.y * h, s.r, 0, Math.PI * 2);
        ctx.fill();
        if (s.cross) {
          // 4 芒十字星光（R&M 常见大星）
          const cr = s.r * 3.2;
          const cx = s.x * w;
          const cy = s.y * h;
          ctx.strokeStyle = "#ffffff";
          ctx.globalAlpha = Math.min(1, s.base * twinkle * 0.7);
          ctx.lineWidth = Math.max(0.6, s.r * 0.5);
          ctx.beginPath();
          ctx.moveTo(cx - cr, cy);
          ctx.lineTo(cx + cr, cy);
          ctx.moveTo(cx, cy - cr);
          ctx.lineTo(cx, cy + cr);
          ctx.stroke();
        }
      }
      ctx.globalAlpha = 1;
    };

    const drawMeteor = (m: Meteor) => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      const headX = m.x * w;
      const headY = m.y * h;
      const len = Math.hypot(m.vx, m.vy) || 1;
      const dirX = m.vx / len;
      const dirY = m.vy / len;
      const tailX = headX - dirX * METEOR_TAIL_PX;
      const tailY = headY - dirY * METEOR_TAIL_PX;
      const alpha = Math.max(0, Math.min(1, m.life));
      const grad = ctx.createLinearGradient(headX, headY, tailX, tailY);
      grad.addColorStop(0, `rgba(168, 216, 255, ${alpha})`);
      grad.addColorStop(1, "rgba(168, 216, 255, 0)");
      ctx.strokeStyle = grad;
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.moveTo(headX, headY);
      ctx.lineTo(tailX, tailY);
      ctx.stroke();
    };

    // 背景装饰层（星球 + 传送门 + 巡航飞船）——叠加在星点/流星之上
    const drawDecor = (timeSec: number, staticMode: boolean) => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      for (const p of PLANETS) {
        drawPlanet(ctx, p, w, h, timeSec, staticMode);
      }
      drawPortal(ctx, PORTAL, w, h, timeSec, staticMode);
      // 巡航飞船：位置由 timeSec 推进（越界环绕）；staticMode 静置起点
      for (const s of FLYING_SAUCERS) {
        const x = staticMode ? s.x : (s.x + s.vx * timeSec) % 1;
        let y = staticMode ? s.y : (s.y + s.vy * timeSec) % 1;
        if (y < 0) y = 1 - y;
        drawFlyingSaucer(ctx, { ...s, x, y }, w, h, timeSec, staticMode);
      }
    };

    const resize = () => {
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.floor(canvas.clientWidth * dpr);
      canvas.height = Math.floor(canvas.clientHeight * dpr);
      buildBg();
      seedStars();
      drawStars(0, false);
      drawNebulae(true);
      drawDecor(0, true);
    };

    const frame = (ts: number) => {
      raf = requestAnimationFrame(frame);
      if (!visible) return;
      if (ts - lastFrame < FRAME_MS) return;
      const dt = Math.min((ts - lastFrame) / 1000, 0.1);
      lastFrame = ts;

      drawStars(ts / 1000, true);
      drawNebulae(false);

      // 偶发流星
      if (ts >= nextMeteorAt) {
        const m: Meteor = {
          x: 0.05 + Math.random() * 0.9,
          y: -0.05,
          vx: 0.2 + Math.random() * 0.25,
          vy: 0.18 + Math.random() * 0.15,
          life: 1,
        };
        if (Math.random() < 0.5) m.vx = -m.vx; // 随机左右方向
        meteors.push(m);
        nextMeteorAt =
          ts + METEOR_MIN_MS + Math.random() * (METEOR_MAX_MS - METEOR_MIN_MS);
      }

      meteors = meteors.filter((m) => m.life > 0);
      for (const m of meteors) {
        drawMeteor(m);
        m.x += m.vx * dt;
        m.y += m.vy * dt;
        m.life -= dt * 0.9;
      }

      // 背景装饰（星球 + 传送门 + 飞船）——叠加在星点/流星之上，低帧率自转
      drawDecor(ts / 1000, false);
    };

    const onVisibility = () => {
      visible = document.visibilityState === "visible";
      if (visible) lastFrame = performance.now();
    };

    resize();
    window.addEventListener("resize", resize);
    document.addEventListener("visibilitychange", onVisibility);

    if (!reduced) {
      nextMeteorAt =
        performance.now() + 2000 + Math.random() * 4000; // 首颗流星早一些
      raf = requestAnimationFrame(frame);
    }

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [density]);

  return (
    <canvas
      ref={canvasRef}
      className={`pointer-events-none fixed inset-0 -z-10 h-screen w-screen ${className}`}
      aria-hidden="true"
    />
  );
}
