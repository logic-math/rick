# task12：工具箱图标语义化（表示「路径」）

## 任务目标

顶栏「打开文件」入口的图标从扳手换成**表示路径**的图标——用户一眼看出「输入路径打开文件」。

## 关键结果

- KR13a：App.tsx 顶栏图标替换为路径语义图标（推荐：文档图标 + 内嵌斜杠路径线 `/_…/` 风格，或 folder + 斜杠组合；R&M 手绘简约线稿风保持——stroke 风格与现有图标一致）
- KR13b：aria-label/title 同步精化（如「打开文件（输入路径）」）；hover title 提示「输入路径打开文件」
- KR13c：尺寸/位置不变（齿轮旁、ConnectionDot 与 Settings 之间）；点击行为不变（同一 Dialog）

# 写域

- web/src/App.tsx

# 依赖关系

- task11

## 实现要点

- 现状：App.tsx:246 附近 ToolboxIcon（扳手 SVG）
- 图标语义参考：路径 = 斜杠分隔——文档(file-text)轮廓 + 中部一条 `/…/` 线条，视觉直接传达「文件路径」
- 保持 24px 线稿风格（stroke 1.5、currentColor）

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate8 断言：新图标标记（如 path 图形 + 「路径/openPath」标识）、扳手标记移除
- 手动：dev UI 顶栏看图标观感

## 禁止事项

- 只改 App.tsx 图标与文案；不动 Dialog/Provider 逻辑；不做 git 操作；只在 dev 树
