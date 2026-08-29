import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";

// rick web 前端构建配置。
// - build.outDir=dist：产物提交仓库（node 不进 rick 构建链——AdGuardHome 模式）
// - build.manifest：输出 .vite/manifest.json 供后端 asset 映射（ygncode 方案）
// - dev proxy：开发态 /api 代理到 rick web 服务（默认 127.0.0.1:6137）
// - PWA（task12 激活）：manifest + autoUpdate service worker；
//   workbox 只缓存 /assets/（hashed 长缓存产物），API/SSE 永不缓存。
//   注意：service worker 需安全上下文（localhost / HTTPS）——LAN IP over http
//   下自动降级为普通网页（注册失败静默，不影响功能）。
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["icon-192.png", "icon-512.png"],
      manifest: {
        name: "rick web",
        short_name: "rick web",
        description: "rick web ui —— 对抗上下文熵增的 AI Coding 控制台",
        theme_color: "#0b0e14",
        background_color: "#0b0e14",
        display: "standalone",
        start_url: "/",
        icons: [
          { src: "icon-192.png", sizes: "192x192", type: "image/png" },
          { src: "icon-512.png", sizes: "512x512", type: "image/png" },
          {
            src: "icon-maskable-512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
      },
      workbox: {
        // 只缓存 hashed 静态产物（不可变资源——长缓存安全）
        globPatterns: ["assets/**"],
        navigateFallbackDenylist: [/^\/api\//],
        // 不预缓存 index.html 之外的 HTML；导航请求交给网络优先（no-cache 语义
        // 由服务端响应头保证；SW 层面 navigateFallback 保持默认 index.html）
        maximumFileSizeToCacheInBytes: 4 * 1024 * 1024,
      },
      devOptions: {
        enabled: false,
      },
    }),
  ],
  build: {
    outDir: "dist",
    manifest: true,
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:6137",
    },
  },
});
