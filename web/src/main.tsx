/**
 * 入口：挂载 App + 全局数据接线。
 *
 * 接线顺序（幂等 wire，见各 store 文件尾部的 wireXxx）：
 * 1. wireUiEvents       —— 401/SSE 状态 → ui store（锁页/连接点信号源）
 * 2. wireSessions       —— session_state → sessions store
 * 3. wireSessionEvents  —— session_event 批处理缓冲 → events store
 * 4.（job_53 Bug #5 已删：jobs_update watcher 随 doing 会话类型移除）
 * 5. sse.connect()      —— 建立 EventSource（token 走 query param）
 *
 * 注意：sse.connect 在无 token 时也会发起（服务端 401 → EventSource 报错 →
 * 指数退避重连；保存 token 后整页刷新重建——TokenGate 的行为契约）。
 */

import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { sse } from "./api/sse";
import { wireUiEvents } from "./stores/ui";
import { wireSessions } from "./stores/sessions";
import { wireSessionEvents } from "./stores/events";
import "./styles/theme.css";

wireUiEvents();
wireSessions();
wireSessionEvents();
sse.connect();

// Bug #8 第三批（human 实测根因）：旧 Service Worker 缓存死锁——SW 拦截 HTML
// 返回旧 bundle → 新代码永远跑不到。web 构建现禁 SW（RICK_DEV_NO_PWA=1）；
// 此处注销浏览器残留的旧 SW（新页面加载后清掉，之后请求走网络，更新即生效）。
if ("serviceWorker" in navigator) {
  void navigator.serviceWorker
    .getRegistrations()
    .then((rs) => {
      for (const r of rs) void r.unregister();
    })
    .catch(() => {});
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
