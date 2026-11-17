/**
 * Session History Cache（会话历史客户端缓存）
 *
 * 多会话切换加速：ChatView 因 key={session.id} 每次切换都重挂——全部历史
 * state（useState/useRef）销毁重建，REST getEntries 重新拉取（500+ entries
 * 300-800ms）→ 用户感知「每次切回来都要重新加载」。
 *
 * 本模块提供模块级（跨组件生命周期）的会话历史缓存：
 * - 切换回来：cache 命中 → 立即渲染（0ms）→ 后台刷新更新
 * - 首次打开：cache miss → 正常拉取 → 写入 cache
 * - 过期策略：TTL 60s 内直接用（不拉网络）；超 TTL 先渲染旧数据再后台刷新
 *
 * 不缓存（不安全/不可复用）：
 * - live SSE envelopes（已由 useSessionEventsStore 全局管理）
 * - optimistic user message（会话切换时已消解）
 * - scroll cursor（服务端 drafts 管理，跨设备）
 */

import type { HistoryBuildResult } from "../components/chat/viewModel";

export interface CachedHistory {
  result: HistoryBuildResult;
  /** 已加载的 entries 数组（raw，供重放 buildHistoryItems） */
  entries: unknown[];
  fetchedAt: number;
}

const CACHE = new Map<string, CachedHistory>();
const TTL_MS = 60_000; // 60s 内直接用

export function getCachedHistory(sessionId: string): CachedHistory | null {
  return CACHE.get(sessionId) ?? null;
}

export function isFresh(cached: CachedHistory): boolean {
  return Date.now() - cached.fetchedAt < TTL_MS;
}

export function setCachedHistory(sessionId: string, result: HistoryBuildResult, entries: unknown[]): void {
  CACHE.set(sessionId, { result, entries, fetchedAt: Date.now() });
}

export function evictCachedHistory(sessionId: string): void {
  CACHE.delete(sessionId);
}

/** 内存上限（防长跑膨胀）：最多缓存 20 个会话，LRU 淘汰最旧 */
const MAX_CACHE = 20;
export function pruneCache(): void {
  if (CACHE.size <= MAX_CACHE) return;
  const sorted = [...CACHE.entries()].sort((a, b) => a[1].fetchedAt - b[1].fetchedAt);
  const toEvict = sorted.slice(0, CACHE.size - MAX_CACHE);
  for (const [id] of toEvict) CACHE.delete(id);
}
