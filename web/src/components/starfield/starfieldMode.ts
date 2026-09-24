/**
 * 星空背景模式（job_43 KR4a/KR4d）——共享读写 + 语义解析。
 *
 * 背景：移动端「空闲也发烫」主根因是全屏 canvas 30fps 常驻（叠加
 * backdrop-blur 每帧 GPU 重合成）。human 裁决（2026-09-24）：
 * - auto（默认）：移动端（<768px）静态一帧定格 / 桌面动画；
 * - animated：强制动画（想要完整视觉的用户可手动开）；
 * - static：强制静态（省电优先）。
 *
 * 该模块是 localStorage 的唯一读写点：
 * - StarfieldBackground 初始化时读取，并监听 STARFIELD_MODE_CHANGE_EVENT
 *   （同页签 Settings 切换）与 storage 事件（跨页签同步）实时切换；
 * - Settings 页写入时派发自定义事件。
 *
 * prefers-reduced-motion 语义保持：auto/animated 模式下若系统要求
 * 减少动效，仍静态化（无障碍优先级高于用户设置——用户可显式选 animated
 * 时也尊重 reduced-motion，与旧行为一致）。
 */

export type StarfieldMode = "auto" | "animated" | "static";

/** localStorage key（与 rick.chat.enterMode 各自独立，互不冲突） */
export const STARFIELD_MODE_KEY = "rick.starfield.mode";

/** 同页签实时切换事件（Settings 写入后派发；StarfieldBackground 监听） */
export const STARFIELD_MODE_CHANGE_EVENT = "rick:starfield-mode-change";

/** 读 localStorage 星空模式（缺省/非法值回退 auto；隐私模式 localStorage 不可用） */
export function readStarfieldMode(): StarfieldMode {
  try {
    const v = localStorage.getItem(STARFIELD_MODE_KEY);
    return v === "animated" || v === "static" ? v : "auto";
  } catch {
    return "auto";
  }
}

/** 写 localStorage 星空模式并派发同页签事件（让已挂载的背景立即切换） */
export function writeStarfieldMode(mode: StarfieldMode): void {
  try {
    localStorage.setItem(STARFIELD_MODE_KEY, mode);
  } catch {
    // localStorage 不可用（隐私模式等）：仅本次事件生效，不持久化
  }
  try {
    window.dispatchEvent(new CustomEvent(STARFIELD_MODE_CHANGE_EVENT, { detail: mode }));
  } catch {
    // 无 window（SSR/测试）：无事件可派发
  }
}

/**
 * 解析「是否运行动画」：
 * - static → 恒 false（省电定格）；
 * - animated → true（用户显式要动画；reduced-motion 仍压制——无障碍优先）；
 * - auto → 桌面（非窄屏）且系统未要求减少动效时动画，否则静态。
 */
export function resolveShouldAnimate(
  mode: StarfieldMode,
  isNarrowViewport: boolean,
  prefersReducedMotion: boolean,
): boolean {
  if (mode === "static") return false;
  if (prefersReducedMotion) return false;
  if (mode === "animated") return true;
  return !isNarrowViewport; // auto：移动端静态、桌面动画
}
