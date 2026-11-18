/**
 * 多次刷新 RSI 会话——验证游标恢复一致性（human 反馈：每次反应不同）
 */
import { chromium } from "playwright";

const BASE = "http://127.0.0.1:8414";
const TOKEN = "devtok-b8512037";
const SID = "4d9763c0-2360-4523-bcc8-29e885a16556";

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await (await browser.newContext({ viewport: { width: 1280, height: 800 } })).newPage();

  const logs = [];
  page.on("console", (msg) => {
    const t = msg.text();
    if (t.includes("[scroll-cursor]") || t.includes("error")) logs.push(t);
  });

  await page.goto(BASE);
  await page.evaluate((t) => localStorage.setItem("rick-web-token", t), TOKEN);
  await page.reload();
  await page.waitForTimeout(2000);

  // 查会话 entries 数
  const entryCount = await page.evaluate(async ({ t, sid }) => {
    const r = await fetch("http://127.0.0.1:8414/api/sessions/" + sid + "/entries", { headers: { Authorization: `Bearer ${t}` } });
    return (await r.json()).entries?.length || 0;
  }, { t: TOKEN, sid: SID });
  console.log(`RSI 会话 entries: ${entryCount}`);

  // 进入会话
  await page.evaluate(async ({ t, sid }) => {
    await fetch("http://127.0.0.1:8414/api/drafts/ui-state", {
      method: "PUT", headers: { Authorization: `Bearer ${t}`, "Content-Type": "application/json" },
      body: JSON.stringify({ content: `/session/${sid}` }),
    });
  }, { t: TOKEN, sid: SID });

  // 测 5 轮刷新：每轮记录游标恢复结果
  for (let round = 1; round <= 5; round++) {
    logs.length = 0;
    await page.reload();
    await page.waitForTimeout(4000); // 等加载 + 定位

    // 读滚动状态
    const pos = await page.evaluate(() => {
      const all = Array.from(document.querySelectorAll("div"));
      const s = all.filter((d) => {
        const st = getComputedStyle(d);
        return (st.overflowY === "auto" || st.overflowY === "scroll") && d.scrollHeight > d.clientHeight + 100;
      }).sort((a, b) => b.scrollHeight - a.scrollHeight)[0];
      if (!s) return null;
      return { st: s.scrollTop, sh: s.scrollHeight, ch: s.clientHeight };
    });

    // 读服务端锚
    const serverAnchor = await page.evaluate(async ({ t, sid }) => {
      const r = await fetch("http://127.0.0.1:8414/api/drafts/scroll:" + sid, { headers: { Authorization: `Bearer ${t}` } });
      return (await r.json()).content?.slice(0, 12);
    }, { t: TOKEN, sid: SID });

    const savedLog = [...logs].reverse().find((l) => l.includes("saved:"));
    const hasSaved = !!savedLog;
    const anchorFromLog = savedLog?.match(/saved: (\S+)/)?.[1]?.slice(0, 12);

    if (pos) {
      const pct = (pos.st / Math.max(pos.sh, 1) * 100).toFixed(1);
      const atTop = pos.st < 50;
      const atBottom = pos.st + pos.ch >= pos.sh - 50;
      console.log(`轮次 ${round}: scrollTop=${pos.st}/${pos.sh} (${pct}%) ${atTop ? "📍顶" : atBottom ? "📍底" : "✅中间"} | 服务端锚=${serverAnchor} | 本轮saved=${anchorFromLog ?? "无"}`);
    } else {
      console.log(`轮次 ${round}: ❌ 未找到滚动容器 | 服务端锚=${serverAnchor}`);
    }

    // 滚动到 1/3（每轮都滚——模拟用户持续阅读）
    await page.evaluate(() => {
      const all = Array.from(document.querySelectorAll("div"));
      const s = all.filter((d) => {
        const st = getComputedStyle(d);
        return (st.overflowY === "auto" || st.overflowY === "scroll") && d.scrollHeight > d.clientHeight + 100;
      }).sort((a, b) => b.scrollHeight - a.scrollHeight)[0];
      if (s) s.scrollTop = Math.floor(s.scrollHeight / 3);
    });
    await page.waitForTimeout(2000); // 等 debounce 保存
  }

  await browser.close();
}

main().catch(console.error);
