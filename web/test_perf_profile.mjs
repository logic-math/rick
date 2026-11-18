/**
 * 刷新流畅度 profile：测量刷新各阶段的耗时（网络/渲染/交互就绪）
 */
import { chromium } from "playwright";

const BASE = "http://127.0.0.1:8414";
const TOKEN = "devtok-b8512037";

async function main() {
  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await ctx.newPage();

  // 收集网络请求耗时
  const netReqs = [];
  page.on("request", (req) => {
    if (req.url().includes("/api/")) netReqs.push({ url: req.url().split("/api/")[1]?.slice(0, 50), start: Date.now() });
  });
  page.on("response", (resp) => {
    if (resp.url().includes("/api/")) {
      const r = netReqs.find((n) => resp.url().includes(n.url));
      if (r) r.ms = Date.now() - r.start;
    }
  });

  // 收集长任务（>50ms blocking）
  const longTasks = [];
  await page.addInitScript(() => {
    window.__longTasks = [];
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        window.__longTasks.push({ name: entry.name.slice(0, 50), duration: Math.round(entry.duration), start: Math.round(entry.startTime) });
      }
    }).observe({ entryTypes: ["longtask"] });
  });

  // ① 首次加载（预热 + 设置 token）
  await page.goto(BASE);
  await page.evaluate((t) => localStorage.setItem("rick-web-token", t), TOKEN);
  await page.reload();
  await page.waitForTimeout(3000);

  // 找 entry 最多的会话
  const sessions = await page.evaluate(async (t) => {
    const r = await fetch('http://127.0.0.1:8414/api/sessions', { headers: { Authorization: `Bearer ${t}` } });
    const d = await r.json();
    const list = Array.isArray(d) ? d : d.sessions || [];
    return list.filter((s) => s.status !== "active").slice(0, 8);
  }, TOKEN);

  let best = null, bestCount = 0;
  for (const s of sessions) {
    const count = await page.evaluate(async ({ t, sid }) => {
      const r = await fetch('http://127.0.0.1:8414/api/sessions/' + sid + '/entries', { headers: { Authorization: `Bearer ${t}` } });
      const d = await r.json();
      return (d.entries || []).length;
    }, { t: TOKEN, sid: s.id });
    if (count > bestCount) { best = s; bestCount = count; }
  }
  console.log("目标会话:", best?.title, "| entries:", bestCount);

  // ② 进入会话页面（设 ui-state）
  await page.evaluate(async ({ t, sid }) => {
    await fetch('http://127.0.0.1:8414/api/drafts/ui-state', {
      method: "PUT",
      headers: { Authorization: `Bearer ${t}`, "Content-Type": "application/json" },
      body: JSON.stringify({ content: `/session/${sid}` }),
    });
  }, { t: TOKEN, sid: best.id });

  // ③ 测量刷新（关键指标）
  netReqs.length = 0;
  const t0 = Date.now();
  await page.reload();

  // 等页面基本可用（主内容渲染）
  await page.waitForSelector("text=" + (best.title?.slice(0, 10) || "session"), { timeout: 15000 }).catch(() => {});
  const tInteractive = Date.now() - t0;
  console.log(`\n📡 首内容可见: ${tInteractive}ms`);

  // 等所有 API 请求完成
  await page.waitForTimeout(3000);
  const tSettled = Date.now() - t0;
  console.log(`📡 完全稳定: ${tSettled}ms`);

  // ④ 网络请求分析
  const apiReqs = netReqs.filter((n) => n.ms !== undefined);
  const totalNetMs = apiReqs.reduce((a, b) => a + b.ms, 0);
  console.log(`\n🌐 API 请求: ${apiReqs.length} 个, 总耗时 ${totalNetMs}ms`);
  const slowReqs = apiReqs.filter((n) => n.ms > 100).sort((a, b) => b.ms - a.ms);
  if (slowReqs.length > 0) {
    console.log("慢请求 (>100ms):");
    slowReqs.slice(0, 5).forEach((r) => console.log(`  ${r.url} → ${r.ms}ms`));
  }

  // ⑤ 长任务分析（CPU 阻塞）
  const tasks = await page.evaluate(() => window.__longTasks || []);
  const totalBlockMs = tasks.reduce((a, b) => a + b.duration, 0);
  console.log(`\n🔒 长任务 (>${50}ms): ${tasks.length} 个, 总阻塞 ${totalBlockMs}ms`);
  if (tasks.length > 0) {
    console.log("最长的阻塞任务:");
    tasks.sort((a, b) => b.duration - a.duration).slice(0, 5).forEach((t) =>
      console.log(`  ${t.duration}ms @ ${t.start}ms: ${t.name}`)
    );
  }

  // ⑥ DOM 复杂度
  const domStats = await page.evaluate(() => {
    const items = document.querySelectorAll("[class*=my-1], [class*=my-2], [class*=rounded-md]");
    return {
      totalElements: document.querySelectorAll("*").length,
      itemLike: items.length,
      textNodes: document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT).root ? "yes" : "no",
    };
  });
  console.log(`\n📦 DOM: ${domStats.totalElements} 元素`);

  // ⑦ FPS 测试（滚动流畅度）
  await page.evaluate(() => {
    window.__fps = [];
    let last = performance.now();
    let frames = 0;
    function tick(now) {
      frames++;
      if (now - last >= 1000) {
        window.__fps.push(frames);
        frames = 0;
        last = now;
      }
      requestAnimationFrame(tick);
    }
    requestAnimationFrame(tick);
  });

  // 滚动测试
  await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll("div"));
    const scrollable = all.filter((d) => {
      const st = getComputedStyle(d);
      return (st.overflowY === "auto" || st.overflowY === "scroll") && d.scrollHeight > d.clientHeight + 200;
    });
    const target = scrollable.sort((a, b) => b.scrollHeight - a.scrollHeight)[0];
    if (target) {
      let y = 0;
      const interval = setInterval(() => {
        y += 50;
        target.scrollTop = y;
        if (y >= target.scrollHeight) clearInterval(interval);
      }, 16);
    }
  });
  await page.waitForTimeout(3000);
  const fps = await page.evaluate(() => window.__fps || []);
  console.log(`\n🎬 滚动 FPS: ${fps.join(", ")} (目标 60)`);

  await browser.close();
}

main().catch((e) => { console.error(e); process.exit(1); });
