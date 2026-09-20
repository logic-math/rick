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
 *   · cel-shade 星球（Planet）：右上暖金环 + 左下紫色小星，极缓自转
 *   · 液态涡旋传送门（PortalArt v2）：**左上 + 右下**对角线各一个
 *   · 传送之旅飞船：右下门 B → 左上门 A 对角穿越，飞入渐隐 → 从 B 渐显（~12s 循环）
 *   · 副对角线巡航飞船：左下紫色星球 ↔ 右上暖金星球往返（~14s，舱内 Rick & Morty）
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

// ============ 背景编排（v3.1：对角线布局） ============
// 用户设计（更正）：传送门在**左上 + 右下**（对角线）；两艘飞船在**左下 ↔ 右上**
// （另一条对角线）之间通行。
//
// 布局（两条对角线交叉但不重叠——传送门占主对角线，飞船走副对角线）：
//   左上传送门 A(0.10, 0.14) ───────── 右上暖金星球(0.84, 0.14)
//        ╲                              ╱
//         ╲  飞船巡航（左下→右上往返） ╱
//          ╲                        ╱
//           ╲─── 传送之旅（右下B→左上A 对角穿越，飞入A渐隐→从B渐显）
//          ╱                        ╲
//   左下紫色星球(0.08, 0.86) ──── 右下传送门 B(0.87, 0.80)

// 星球：右上暖金环 + 左下紫色
const PLANETS: PlanetSpec[] = [
  { x: 0.84, y: 0.14, r: 0.12, alpha: 0.5, ring: true, phase: 0.6, speed: 0.05 },
  // 左下小星球：紫色系（用户要求）——亮 #c4b5fd / 主 #8b5cf6 / 暗 #4c1d95
  {
    x: 0.08, y: 0.86, r: 0.07, alpha: 0.4, ring: false, phase: 2.1, speed: 0.04,
    baseColor: "#8b5cf6", highlightColor: "#c4b5fd", shadowColor: "#4c1d95",
  },
];

// 双传送门（液态涡旋）：**左上 A + 右下 B**（用户更正）
const PORTALS: PortalSpec[] = [
  { x: 0.10, y: 0.14, r: 0.10, alpha: 0.85, speed: 1.0 },   // A：左上
  { x: 0.87, y: 0.80, r: 0.10, alpha: 0.85, speed: -0.8 },  // B：右下（反向转）
];

// 副对角线巡航飞船：左下紫色星球(0.08,0.86) ↔ 右上暖金星球(0.84,0.14)
const SAUCER_CRUISE: FlyingSaucerSpec = {
  x: 0.46, y: 0.5, vx: 0, vy: 0, size: 0.038, alpha: 0.35, phase: 1.2,
};

// 传送之旅飞船：从右下门 B(0.87,0.80) 飞向左上门 A(0.10,0.14)，飞入渐隐 → 从 B 渐显
const SAUCER_PORTAL: FlyingSaucerSpec = {
  x: 0.5, y: 0.5, vx: 0, vy: 0, size: 0.042, alpha: 0.42, phase: 0.0,
};

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
    // 缓存的 CSS 像素尺寸：**只在 resize/ResizeObserver 时读取 clientWidth/Height**。
    // 旧版在每个绘制函数里逐帧读 canvas.clientWidth/clientHeight——任何 DOM 变更后
    // 该读取都会强制同步重排（forced synchronous layout）；长会话（2.5 万 DOM 节点）
    // 下每帧 ~1.3ms×10 次 → 流式期间掉帧（实测 12s/600 帧）。详见 debug/bug4。
    let cssW = 0;
    let cssH = 0;

    const seedStars = () => {
      const w = cssW;
      const h = cssH;
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
      const w = cssW;
      const h = cssH;
      bgGradient = ctx.createLinearGradient(0, 0, 0, h);
      bgGradient.addColorStop(0, BG_TOP);
      bgGradient.addColorStop(1, BG_BOTTOM);
      ctx.fillStyle = bgGradient;
      ctx.fillRect(0, 0, w, h);
    };

    const drawNebulae = (staticMode: boolean) => {
      const w = cssW;
      const h = cssH;
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
      const w = cssW;
      const h = cssH;
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
      const w = cssW;
      const h = cssH;
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

    // 背景装饰层（星球 + 双传送门 + 两种飞船）——叠加在星点/流星之上
    const drawDecor = (timeSec: number, staticMode: boolean) => {
      const w = cssW;
      const h = cssH;

      // 星球
      for (const p of PLANETS) {
        drawPlanet(ctx, p, w, h, timeSec, staticMode);
      }

      // 双传送门（先画门，飞船叠在上面）
      for (const portal of PORTALS) {
        drawPortal(ctx, portal, w, h, timeSec, staticMode);
      }

      // ---- 副对角线巡航飞船：左下紫色星球(0.08,0.86) ↔ 右上暖金星球(0.84,0.14) ----
      // 三角波插值往返；周期 ~14s
      if (!staticMode) {
        const CYCLE_S = 14;
        const t = (timeSec % CYCLE_S) / CYCLE_S; // 0..1
        const tri = t < 0.5 ? t * 2 : (1 - t) * 2; // 0→1→0 三角波
        // 左下(0.08,0.86) → 右上(0.84,0.14)
        const x1 = 0.08, y1 = 0.86, x2 = 0.84, y2 = 0.14;
        const x = x1 + (x2 - x1) * tri;
        const y = y1 + (y2 - y1) * tri;
        const flip = tri > 0.5; // 返程时翻转朝向
        drawFlyingSaucer(ctx, { ...SAUCER_CRUISE, x, y, flipped: flip }, w, h, timeSec, staticMode);
      } else {
        drawFlyingSaucer(ctx, SAUCER_CRUISE, w, h, timeSec, staticMode);
      }

      // ---- 传送之旅飞船：右下门 B(0.87,0.80) → 左上门 A(0.10,0.14)，对角线穿越 ----
      // 飞入 A 渐隐 → 从 B 渐显（循环 ~12s）
      //   前 15%：从 B 渐显（alpha 0→1）
      //   中 70%：对角线匀速飞向 A
      //   后 15%：靠近 A 渐隐（alpha 1→0），飞入门内
      if (!staticMode) {
        const TRIP_S = 12;
        const t = (timeSec % TRIP_S) / TRIP_S; // 0..1
        // B(右下 0.87,0.80) → A(左上 0.10,0.14)
        const xB = 0.87, yB = 0.80, xA = 0.10, yA = 0.14;
        const x = xB + (xA - xB) * t;
        const y = yB + (yA - yB) * t;
        // 渐入渐出
        let vis = 1;
        if (t < 0.15) vis = t / 0.15;
        else if (t > 0.85) vis = (1 - t) / 0.15;
        // 轻微弧线偏移（垂直于飞行方向）
        const arc = Math.sin(t * Math.PI) * 0.04;
        if (vis > 0.02) {
          drawFlyingSaucer(
            ctx, { ...SAUCER_PORTAL, x: x + arc, y: y - arc, alpha: SAUCER_PORTAL.alpha * vis },
            w, h, timeSec, staticMode,
          );
        }
      } else {
        drawFlyingSaucer(ctx, SAUCER_PORTAL, w, h, timeSec, staticMode);
      }
    };

    const resize = () => {
      // 唯一的尺寸读取点（布局读取仅在此发生）
      cssW = canvas.clientWidth;
      cssH = canvas.clientHeight;
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.max(1, Math.floor(cssW * dpr));
      canvas.height = Math.max(1, Math.floor(cssH * dpr));
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
    // 元素盒变化（视口/缩放/DPR 变化）→ 重取尺寸；避免在动画帧内读布局
    const ro = typeof ResizeObserver === "function" ? new ResizeObserver(() => resize()) : null;
    ro?.observe(canvas);

    if (!reduced) {
      nextMeteorAt =
        performance.now() + 2000 + Math.random() * 4000; // 首颗流星早一些
      raf = requestAnimationFrame(frame);
    }

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
      document.removeEventListener("visibilitychange", onVisibility);
      ro?.disconnect();
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
