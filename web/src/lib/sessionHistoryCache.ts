/**
 * Session History Cache（会话历史客户端缓存）
 *
 * 多会话切换加速：ChatView 因 key={session.id} 每次切换都重挂——全部历史
 * state（useState/useRef）销毁重建，REST getEntries 重新拉取（500+ entries
 * 300-800ms）→ 用户感知「每次切回来都要重新加载」。
 *
 * 本模块提供模块级（跨组件生命周期）的会话历史缓存：
 * - 切换回来：cache 命中 → 立即渲染（0ms）→ 后台增量拉取更新（保持 timeline
 *   一致性：只拉缓存 leaf 之后的新条目，合并重建，无闪烁）
 * - 首次打开：cache miss → 正常拉取 → 写入 cache
 * - 过期策略：TTL（默认 7 天，Settings 可调）内有效；增量拉取无新条目时
 *   滑动续期；超期未访问 → 下次全量拉取并覆盖
 * - 内存上限：最多缓存 maxSessions 个会话（默认 64，Settings 可调），LRU 淘汰
 *
 * 不缓存（不安全/不可复用）：
 * - live SSE envelopes（已由 useSessionEventsStore 全局管理）
 * - optimistic user message（会话切换时已消解）
 * - scroll cursor（服务端 drafts 管理，跨设备）
 */

import type { HistoryBuildResult } from "../components/chat/viewModel";

export interface CachedHistory {
  result: HistoryBuildResult;
  /** 已加载的 entries 数组（raw，供重放 buildHistoryItems / 增量合并） */
  entries: unknown[];
  /** 分页状态（上拉加载更早——缓存命中时恢复，否则 hasMore=false 无法上拉） */
  earliestId: string | null;
  hasMore: boolean;
  fetchedAt: number;
}

/** 设置（localStorage 持久化；Settings 页可改，动态读取即时生效） */
export const HISTORY_CACHE_CONFIG_KEY = "rick-web:history-cache";
export interface HistoryCacheConfig {
  /** 缓存时长（天）——超期未访问的会话缓存过期（下次全量拉取） */
  ttlDays: number;
  /** 最大缓存会话数（LRU 淘汰最旧） */
  maxSessions: number;
}
export const DEFAULT_HISTORY_CACHE_CONFIG: HistoryCacheConfig = { ttlDays: 7, maxSessions: 64 };

export function readHistoryCacheConfig(): HistoryCacheConfig {
  try {
    const raw = localStorage.getItem(HISTORY_CACHE_CONFIG_KEY);
    if (raw) {
      const v = JSON.parse(raw) as Partial<HistoryCacheConfig> | null;
      if (v && typeof v.ttlDays === "number" && typeof v.maxSessions === "number") {
        return {
          ttlDays: Math.min(365, Math.max(1, Math.floor(v.ttlDays))),
          maxSessions: Math.min(512, Math.max(1, Math.floor(v.maxSessions))),
        };
      }
    }
  } catch {
    // localStorage 不可用 / JSON 损坏——默认
  }
  return DEFAULT_HISTORY_CACHE_CONFIG;
}

const CACHE = new Map<string, CachedHistory>();

export function getCachedHistory(sessionId: string): CachedHistory | null {
  return CACHE.get(sessionId) ?? null;
}

export function isFresh(cached: CachedHistory): boolean {
  const ttlMs = readHistoryCacheConfig().ttlDays * 24 * 60 * 60 * 1000;
  return Date.now() - cached.fetchedAt < ttlMs;
}

export function setCachedHistory(sessionId: string, result: HistoryBuildResult, entries: unknown[], earliestId: string | null, hasMore: boolean): void {
  CACHE.set(sessionId, { result, entries, earliestId, hasMore, fetchedAt: Date.now() });
}

/** 滑动续期（增量拉取无新条目时刷新时间戳——重置 TTL 窗口） */
export function touchCachedHistory(sessionId: string): void {
  const c = CACHE.get(sessionId);
  if (c) c.fetchedAt = Date.now();
}

export function evictCachedHistory(sessionId: string): void {
  CACHE.delete(sessionId);
}

/** 内存上限（防长跑膨胀）：最多缓存 maxSessions 个会话（Settings 可调），LRU 淘汰最旧 */
export function pruneCache(): void {
  const max = readHistoryCacheConfig().maxSessions;
  if (CACHE.size <= max) return;
  const sorted = [...CACHE.entries()].sort((a, b) => a[1].fetchedAt - b[1].fetchedAt);
  const toEvict = sorted.slice(0, CACHE.size - max);
  for (const [id] of toEvict) CACHE.delete(id);
}
