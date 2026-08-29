/**
 * 路由层共享 hooks（task12）。
 *
 * useMediaQuery：JS 断点感知——必须与 Tailwind 断点对齐（768/1024 两档，
 * 对应 Tailwind md:/lg:）。SSR/测试安全（无 window 时返回 false）。
 * 预演坑位（LibreChat bug 教训）：Tailwind class 断点与 JS 断点数值不一致
 * 会导致桌面布局与 JS 逻辑分叉——此处单一常量源保证对齐。
 */

import { useEffect, useState } from "react";

/** Tailwind 断点对齐常量（px） */
export const BREAKPOINTS = {
  /** md（移动端 ↔ 平板/桌面分界） */
  md: 768,
  /** lg（平板 ↔ 桌面分界） */
  lg: 1024,
} as const;

export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false;
    return window.matchMedia(query).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return;
    const mql = window.matchMedia(query);
    const onChange = (ev: MediaQueryListEvent) => setMatches(ev.matches);
    setMatches(mql.matches);
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [query]);

  return matches;
}

/** <768px（移动端布局：drawer 侧栏 / 粘性输入 / 堆叠视图） */
export function useIsMobile(): boolean {
  return useMediaQuery(`(max-width: ${BREAKPOINTS.md - 1}px)`);
}

/** ≥1024px（大桌面：可用双栏宽布局） */
export function useIsDesktop(): boolean {
  return useMediaQuery(`(min-width: ${BREAKPOINTS.lg}px)`);
}
