# bug2：MessageList 虚拟化自定义 Scroller 丢弃 Virtuoso props → 会话消息区全空白（P0）

**发现**：2026-09-29（release 3e35ec8 后用户立刻报告「会话消息没有了」） | **修复**：0d26c68（其他 agent，2026-09-29 12:46，已发布生产 0d26c68-260929124639） | **状态**：✅ 已修复

## 症状

release 提升后所有会话的消息区全空白（消息数据完好，纯渲染层故障）。

## 根因

task7（eeda69d）引入：MessageList 换 react-virtuoso 时自定义 Scroller 组件只解构 `{ style, children }`，**丢弃了 Virtuoso 通过 props 传入的其余 props**——尤其 `data-virtuoso-scroller` 标记属性。Virtuoso 测量回调沿父链找标记节点：

```
for (; g.dataset.virtuosoScroller === void 0;) g = g.parentElement
```

找不到 → 遍历到 null → `TypeError: Cannot read properties of null (reading 'dataset')` → 尺寸状态永不就绪 → 消息区渲染空白。

## 修复

Scroller 改为 `{ style, children, ...rest }` 全量转发（`<div {...rest}>`）。

## 教训（防再犯）

1. **自定义 Scroller/components 必须转发全部 props**——Virtuoso 等虚拟化库把关键标记/回调藏在 props 里，挑着解构等于静默破坏其内部协议
2. **gate4 静态断言的盲区**：gate4 断言了 Virtuoso/followOutput/computeItemKey 存在，但测不到「props 转发缺失」这类运行时协议破坏——前端无运行时测试时，**虚拟化这类重写渲染层的改动必须先在 dev UI 做真渲染验收再 release**（本次 dev 8414 只验证了 bundle 构建，没做打开会话页看消息的验收）
3. 应急响应良好实践：用户报告 → 立即定位（虚拟化刚上线为首要嫌疑）→ 回滚点在（8dd1307）但选择了热修复（0d26c68）→ 当轮 release，无需回滚

## 关联

- job_43 task7（eeda69d）引入；gate4 未能拦截（见教训 2）
- research-I2 O6 风险预警「虚拟化与贴底/展开态交互复杂」成真——复杂度不在 API 调用而在组件协议细节
