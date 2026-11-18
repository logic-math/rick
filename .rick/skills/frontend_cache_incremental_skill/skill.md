# skill:frontend-cache-incremental（前端历史数据缓存 + 增量更新模式）

## 触发场景

给 React 组件（因 key 重挂或路由切换销毁 state）加「历史数据客户端缓存」或「增量拉取更新」时使用。信号词：
- 「每次切回来都要重新加载」「会话/列表切换慢」
- 缓存命中后某功能（如上拉分页）失效
- 增量合并导致切换卡顿（巨型数据被全量重建）

## 预期效果

- 切换回来 0ms 首屏（缓存直渲染）+ 后台增量拉取保持 timeline 一致
- 缓存恢复全部下游功能（含分页上拉），无回归
- 巨型数据（8MB/626 条）增量路径不卡顿（无全量重建）

## 核心内容

### 缓存契约 = 消费方全量 state 切片

缓存渲染产物时必须枚举组件**全部下游 state 依赖**：items + 分页（earliestId/hasMore）+ flags，缺一即回归。job_53 实测：只存 items 漏分页 → 切回 hasMore=false → 上拉加载断裂，一轮 human 报障返工。

### 增量拉取

```javascript
const resp = await api.getEntries(id, { since: leafId }); // since 语义含 leaf 本身
const delta = resp.entries.filter((e) => e.id !== leafId);
if (delta.length === 0) { touchCache(id); return; } // 无新条目 → 滑动续期
```

### delta-build-append（增量路径禁全量重建）

只 build delta、拼接既有 items——全量重建巨型数据 = 300ms+ 主线程阻塞 + items 引用全换 → React 全量重渲染（job_53 巨型会话切换卡顿根因）：

```javascript
const deltaBuilt = build(delta); // 只 build 新条目（成本 O(delta)）
merged.items = [...baseItems, ...deltaBuilt.items]; // 既有引用稳定 → memo 不重挂
merged.fingerprints = new Set([...base.fingerprints, ...deltaBuilt.fingerprints]);
```

- **跨缓存边界配对**（toolCall 在缓存尾/toolResult 在 delta 首）：delta 里的独立 result 卡（args=null）回填到缓存中同 toolCallId 且空 output 的项（用**新对象引用**触发该项重渲染），独立卡丢弃
- **缓存写回含补拉页时**：组合循环内已构建结果（flatMap items/fingerprints），零二次全量构建

### hasMore 判定

窗口内条数 < PAGE_SIZE ≠ 没有更早条目（窗口只装 36 条但之前还有 44 条）——hasMore 默认 true，API 返回空页时自动置 false。

### 设计检查（30 秒成本模型）

增量路径**按最坏规模设计**：已知极端数据在手（8MB 会话）就该按 8MB 设计，别按平均规模——「build(全量) 每次切换跑一遍值不值？」30 秒检查省一轮返工（job_53 task50 全量重建 → task51 才改 delta 拼接）。

### 附带模式

- 配置（TTL 天数/最大条目数）localStorage 动态读取即时生效（Settings 页可改，无需刷新）
- 渲染门控（如「恢复位置」等待）必须有超时放行（500ms）——门控无限等 = 用户感知「加载不出来」
