# skill:virtualization-component-protocol（虚拟化库自定义组件 props 协议）

## 触发场景

在 react-virtuoso / react-window 等虚拟化库中自定义 `components`（Scroller/Header/Footer/Item）或接管其渲染容器时使用。信号词：
- 换虚拟化后列表/消息区**整体空白**但数据完好（控制台 `Cannot read properties of null (reading 'dataset')`）
- 自定义组件只解构了 `{ style, children }` 这类「看起来够用」的 props
- 虚拟化库文档提到 measurement/observer 依赖 DOM 标记（如 `data-virtuoso-scroller`）

## 预期效果

- 自定义组件不静默破坏虚拟化库内部协议，避免上线即 P0（job_43 task7 实测：gate 静态断言全绿但生产全空白，靠用户报告才发现）
- 引用稳定（模块级常量/useMemo），不因 components 引用变化触发整树重挂

## 核心内容

1. **全量转发**：自定义组件签名一律 `({ style, children, ...rest }, ref)`，DOM 上 `{...rest}` 展开——虚拟化库把测量标记/回调藏在 props 里（Virtuoso 沿父链找 `data-virtuoso-scroller`，找不到 → 测量循环 null 崩溃 → 尺寸永不就绪 → 渲染空白）
2. **引用稳定**：`components={{ Scroller, Header, Footer }}` 的成员必须模块级常量或 useMemo——每次渲染新函数引用会导致滚动容器/列表重挂（闪烁+滚动位置丢失）
3. **间距走 Footer**：底部 padding 不进容器 className（虚拟化库不把容器 padding 计入贴底计算）——用 `components.Footer` 返回固定高度 spacer，计入 footerHeight 后 followOutput/scrollToIndex 都带着它算
4. **贴底双通道**：`followOutput` 只管「追加」，流式原地增高需 `scrollToIndex(LAST, align:end)` 补充；前置插入用 `firstItemIndex` 递减防跳
5. **验收**：typecheck/build 全绿≠可用——必须真渲染验收（打开页面看列表渲染，见 skill:render-layer-acceptance）
