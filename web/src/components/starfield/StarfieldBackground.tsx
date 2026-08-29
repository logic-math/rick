import { useEffect, useRef } from "react";

/**
 * 动态星空背景（Rick and Morty 深空）。
 *
 * - canvas 单层实现：静态星点（数量随视口面积缩放，基准 ~120 @1280x720）
 * - 闪烁：每星随机相位/速度的正弦亮度
 * - 流星：每 6-15s 一颗，斜线拖尾渐隐
 * - 性能：动画帧率限 ~30fps；document 不可见时暂停（visibilitychange）
 * - prefers-reduced-motion: reduce → 降级为静态星点，无动画
 */
interface Star {
  x: number; // 归一化 0..1
  y: number;
  r: number; // 半径 px
  base: number; // 基础亮度
  phase: number;
  speed: number;
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

    const seedStars = () => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      const count = Math.max(
        40,
        Math.round((density * w * h) / (1280 * 720))
      );
      stars = [];
      for (let i = 0; i < count; i++) {
        stars.push({
          x: Math.random(),
          y: Math.random(),
          r: 0.4 + Math.random() * 1.4,
          base: 0.25 + Math.random() * 0.65,
          phase: Math.random() * Math.PI * 2,
          speed: 0.4 + Math.random() * 1.2,
        });
      }
    };

    const drawStars = (timeSec: number, animated: boolean) => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);
      for (const s of stars) {
        const twinkle = animated
          ? 0.55 + 0.45 * Math.sin(s.phase + timeSec * s.speed)
          : 1;
        ctx.globalAlpha = Math.min(1, s.base * twinkle);
        ctx.fillStyle = "#e8ecf8";
        ctx.beginPath();
        ctx.arc(s.x * w, s.y * h, s.r, 0, Math.PI * 2);
        ctx.fill();
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

    const resize = () => {
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.floor(canvas.clientWidth * dpr);
      canvas.height = Math.floor(canvas.clientHeight * dpr);
      seedStars();
      drawStars(0, false);
    };

    const frame = (ts: number) => {
      raf = requestAnimationFrame(frame);
      if (!visible) return;
      if (ts - lastFrame < FRAME_MS) return;
      const dt = Math.min((ts - lastFrame) / 1000, 0.1);
      lastFrame = ts;

      drawStars(ts / 1000, true);

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
      className={`pointer-events-none fixed inset-0 -z-10 ${className}`}
      aria-hidden="true"
    />
  );
}
