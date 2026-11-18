# skill:headless-e2e-discipline（无头浏览器 E2E 验证纪律）

## 触发场景

用 Playwright/无头浏览器验证 web UI 行为或性能时使用。信号词：
- 「测一下渲染耗时」「验证新代码是否部署生效」「无头浏览器跑一遍」
- 测试脚本读数离谱（如恒等于固定等待时长）、locator 30s 超时卡死、bundle 字符串断言假阴性

## 预期效果

- 消除三类高频假结果（job_53 实测）：固定等待假读数（「3068ms」真值 117ms）、侧栏收起导致 locator 卡死（4 份脚本连死于同一假设）、bundle 截断漏检（1.7MB 只查前 50 万字符）
- 测试脚本一次跑通，读数可信，不用磨 human「又卡住了吗」

## 核心内容

### 度量纪律（最重要）

计时区间内**禁止固定 sleep**——用短间隔轮询（25ms）+ 满足条件即停，`Date.now()` 只包住轮询循环：

```javascript
const t0 = Date.now();
for (let i = 0; i < 200; i++) {
  const ok = await page.evaluate(() => /* 判定条件 */).catch(() => false);
  if (ok) { console.log(Date.now() - t0, "ms"); break; }
  await page.waitForTimeout(25);
}
```

反例：`await page.waitForTimeout(3000); console.log(Date.now() - t0)`——测的是等待时长，不是渲染时长（job_53 三份脚本全部报「~3000ms」假读数，误导出一轮错误诊断）。

### 导航假设必须事前校验

- 进入会话页后侧栏**自动收起**——`a[href^='/session/']` locator 直接超时卡死。切会话前先点工作区行（如 `text=RICK-DEV`）展开侧栏再点链接
- 任何 locator 假设先用 `count()` 验证存在再 click；找不到时优先换导航方式而不是加超时

### bundle 断言必须全量 fetch

```javascript
const el = document.querySelector('script[src*="index-"]');
const full = await (await fetch(el.src)).text(); // 全量 1.7MB——不能 .slice(0, 500000)（尾段字符串全漏）
full.includes("目标字符串")
```

minifier 会重命名局部变量——断言**属性名/字符串字面量**（如 `.args===null`、localStorage key），不断言变量名（如 `deltaBuilt`）。

### 共享 helper（防脚本各自从零实现）

沉淀 `helpers.mjs` 三函数：`waitForMainContent(page)`（轮询等 main 有内容）、`switchToSession(page, sid)`（先点工作区行展开再点会话链接，吸收侧栏收起事实）、`bundleContains(page, needle)`（全量 fetch）。新测试脚本**禁止**自写等待/导航/bundle 断言，必须 import helper——四份死掉脚本的死因全部会被这一个文件吸收。

### 失败即截图

测试包 try/finally + 失败时 `page.screenshot({ path })`——避免「TimeoutError 一行日志 → 猜」的循环。
