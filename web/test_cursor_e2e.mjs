/**
 * 端到端无头浏览器测试：滚动游标保存 + 刷新恢复定位
 */
import { chromium } from "playwright";

const BASE = "http://127.0.0.1:8414";
const TOKEN = "devtok-b8512037";

async function main() {
  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await ctx.newPage();

  const logs = [];
  page.on("console", (msg) => {
    const text = msg.text();
    if (text.includes("[scroll-cursor]")) logs.push(text);
  });

  await page.goto(BASE);
  await page.evaluate((t) => localStorage.setItem("rick-web-token", t), TOKEN);
  await page.reload();
  await page.waitForTimeout(3000);

  // 找 entry 最多的会话（通过 API）
  const sessions = await page.evaluate(async (t) => {
    const r = await fetch("http://127.0.0.1:8414/api/sessions", {
      headers: { Authorization: `Bearer ${t}` },
    });
    const d = await r.json();
    const list = Array.isArray(d) ? d : d.sessions || [];
    return list.filter((s) => s.status !== "active").slice(0, 8);
  }, TOKEN);

  // 拉每个会话的 entries 数，选最多的
  let best = null;
  let bestCount = 0;
  for (const s of sessions) {
    const count = await page.evaluate(async ({ t, sid }) => {
      const r = await fetch(`http://127.0.0.1:8414/api/sessions/${sid}/entries`, {
        headers: { Authorization: `Bearer ${t}` },
      });
      const d = await r.json();
      return (d.entries || []).length;
    }, { t: TOKEN, sid: s.id });
    if (count > bestCount) { best = s; bestCount = count; }
  }
  console.log("目标会话:", best?.id?.slice(0, 8), best?.title, "| entries:", bestCount);
  if (!best || bestCount < 10) { console.error("无足够长的会话"); await browser.close(); return; }

  // 进入会话页
  await page.evaluate(async ({ t, sid }) => {
    await fetch("http://127.0.0.1:8414/api/drafts/ui-state", {
      method: "PUT",
      headers: { Authorization: `Bearer ${t}`, "Content-Type": "application/json" },
      body: JSON.stringify({ content: `/session/${sid}` }),
    });
  }, { t: TOKEN, sid: best.id });
  await page.reload();
  await page.waitForTimeout(4000); // 等历史加载

  // 找 Virtuoso 滚动容器（不用 data-testid——不存在的属性，改为找 overflow-y:auto 且子元素多的）
  const scrollerFound = await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll("div"));
    const scrollable = all.filter((d) => {
      const st = getComputedStyle(d);
      return (st.overflowY === "auto" || st.overflowY === "scroll") && d.scrollHeight > d.clientHeight + 200;
    });
    if (scrollable.length === 0) return { found: false, count: 0 };
    // 最大的那个就是消息列表
    const target = scrollable.sort((a, b) => b.scrollHeight - a.scrollHeight)[0];
    return { found: true, count: scrollable.length, sh: target.scrollHeight, ch: target.clientHeight, cls: target.className.slice(0, 60) };
  });
  console.log("滚动容器:", scrollerFound);
  if (!scrollerFound.found) {
    await page.screenshot({ path: "/tmp/cursor_debug.png" });
    console.error("❌ 未找到消息滚动容器");
    await browser.close();
    return;
  }

  // 滚动到 1/3 处
  console.log("--- 滚动到 1/3 ---");
  await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll("div"));
    const scrollable = all.filter((d) => {
      const st = getComputedStyle(d);
      return (st.overflowY === "auto" || st.overflowY === "scroll") && d.scrollHeight > d.clientHeight + 200;
    });
    const target = scrollable.sort((a, b) => b.scrollHeight - a.scrollHeight)[0];
    target.scrollTop = Math.floor(target.scrollHeight / 3);
  });
  await page.waitForTimeout(6000); // 等 auto-compensation 完成 + rangeChanged + debounce + saveDraft

  console.log("滚动后全部日志:", logs);
  const saved = [...logs].reverse().find((l) => l.includes("saved:"));
  if (!saved) { console.error("❌ 保存链路断"); await browser.close(); return; }
  console.log("✅ 保存:", saved);

  // 直接查服务端（确认保存到位）
  const serverVal = await page.evaluate(async ({ t, sid }) => {
    const r = await fetch(`http://127.0.0.1:8414/api/drafts/scroll:${sid}`, {
      headers: { Authorization: `Bearer ${t}` },
    });
    const d = await r.json();
    return d.content;
  }, { t: TOKEN, sid: best.id });
  console.log("服务端锚:", serverVal, "| 期望:", saved.match(/saved: (\S+)/)?.[1]);
  console.log(serverVal === saved.match(/saved: (\S+)/)?.[1] ? "✅ 一致" : "❌ 不一致——保存被覆盖或失败");

  // 刷新
  console.log("--- 刷新 ---");
  logs.length = 0;
  await page.reload();
  await page.waitForTimeout(6000); // 等页面加载 + 历史恢复 + 定位

  console.log("刷新后日志:", logs.slice(-8));

  // 检查位置
  const pos = await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll("div"));
    const scrollable = all.filter((d) => {
      const st = getComputedStyle(d);
      return (st.overflowY === "auto" || st.overflowY === "scroll") && d.scrollHeight > d.clientHeight + 200;
    });
    if (scrollable.length === 0) return { error: "no scroller" };
    const target = scrollable.sort((a, b) => b.scrollHeight - a.scrollHeight)[0];
    return { st: target.scrollTop, sh: target.scrollHeight, ch: target.clientHeight };
  });
  if (pos.error) { console.error("❌", pos.error); await browser.close(); return; }
  const pct = pos.st / Math.max(pos.sh, 1) * 100;
  console.log(`滚动位置: ${pos.st}/${pos.sh} (${pct.toFixed(1)}%)`);
  const nearThird = Math.abs(pct - 33.3) < 15;
  console.log(nearThird ? `✅ 定位在 1/3 附近（${pct.toFixed(0)}%）` : pct > 80 ? `❌ 贴底（${pct.toFixed(0)}%）` : `⚠️ 位置偏移（${pct.toFixed(0)}%，预期 ~33%）`);

  await page.screenshot({ path: "/tmp/cursor_e2e_result.png" });
  await browser.close();
}

main().catch((e) => { console.error(e); process.exit(1); });
