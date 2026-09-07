# 依赖关系
无

# 写域
web/src/components/starfield/StarfieldBackground.tsx
web/src/components/starfield/Planet.tsx（新建）
web/src/components/starfield/PortalArt.tsx（新建）

# 任务目标
星空背景增强（用户反馈：背景太素，加 rick and morty 风格的紫色星球与绿色传送门）。

# 关键结果

1. **紫色星球**（Planet.tsx）：canvas 绘制 1-2 颗紫色系星球（深紫 #5d3fd3 渐变 + 表面环形纹理/暗斑 + 大气辉光；可选：细环（土星式，R&M 星云紫）；位置：角落偏置（如左上/右下），随视口缩放；缓慢自转（纹理偏移动画）；低透明度融入背景（不抢内容可读性）
2. **绿色传送门**（PortalArt.tsx）：背景大传送门（绿色旋涡：同心椭圆旋转 + 高光扫过 + 边缘辉光），1 个，位置：偏右下或居中偏上作为视觉锚点；SMIL/CSS 旋转动画（与 favicon 同款风格）；尺寸大（≥300px）但透明度低（背景氛围用）；可选：传送门旁 1-2 个漂浮小碎片（绿色光点沿旋涡轨迹）
3. **整合**：StarfieldBackground 在现有星点/流星层之上叠加 Planet + PortalArt 层（canvas 分层绘制或叠加 DOM/SVG 层）；prefers-reduced-motion 时全部静态（星球/传送门不旋转）
4. 性能：星球/传送门动画帧率低（10-20fps 即可——大图形旋转，视觉上流畅）；页面不可见（visibilitychange）暂停
5. 视觉一致性：R&M 配色 token（--rm-nebula #5d3fd3 紫 / --rm-portal #39ff88 绿 / 深空 #0b0e14）；不影响前景内容对比度（背景元素 opacity 0.3-0.5 + 模糊边缘）

# 测试方法
npx tsc --noEmit + npm run build；playwright 冒烟：截图对比（截图存 /tmp/starfield_v2.png——目检星球/传送门可见）；canvas 元素存在 + 无 JS 错误；reduced-motion 降级不报错

# 上下文提示
- 现有 StarfieldBackground.tsx：canvas 星点 + 流星（requestAnimationFrame 30fps + visibilitychange 暂停）
- 传送门视觉参考：绿色同心椭圆旋涡（Rick and Morty 标志性传送门）——favicon.svg 已有简化版可参考
- 星球视觉参考：R&M 片头星云/紫色行星
