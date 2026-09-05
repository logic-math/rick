/**
 * 时间线展开控制（chat/monitor 共用）。
 *
 * 三态 mode（默认/全部展开/全部折叠）+ 单项 override（用户手动点击的块
 * 不再随 mode 批量变化）。子折叠块用 useExpandState(key, defaultOpen) 接入。
 */

import { createContext, useContext } from "react";

export type ExpandMode = "default" | "all-open" | "all-closed";

export interface ExpandCtrl {
  mode: ExpandMode;
  /** 单项覆盖（key → open） */
  overrides: ReadonlyMap<string, boolean>;
  setOverride: (key: string, open: boolean) => void;
}

export const ExpandCtx = createContext<ExpandCtrl>({
  mode: "default",
  overrides: new Map(),
  setOverride: () => {},
});

/** 子折叠块解析自己的 open 态：override > mode > defaultOpen */
export function useExpandState(key: string, defaultOpen: boolean): {
  open: boolean;
  onToggle: (open: boolean) => void;
} {
  const { mode, overrides, setOverride } = useContext(ExpandCtx);
  const open = overrides.has(key)
    ? (overrides.get(key) as boolean)
    : mode === "all-open"
      ? true
      : mode === "all-closed"
        ? false
        : defaultOpen;
  return {
    open,
    onToggle: (next: boolean) => setOverride(key, next),
  };
}
