import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";

// rick web 前端构建配置。
// - build.outDir=dist：产物提交仓库（node 不进 rick 构建链——AdGuardHome 模式）
// - build.manifest：输出 .vite/manifest.json 供后端 asset 映射（ygncode 方案）
// - dev proxy：开发态 /api 代理到 rick web 服务，**目标参数化**（RICK_DEV_API）——
//   默认指向隔离的 dev 实例（8414），而不是写死某个端口（写死会导致开发态代理到
//   错误的实例：历史默认值是生产端口，dev 实例上做前后端联调时会静默打到生产）。
// - PWA（task12 激活）：manifest + autoUpdate service worker；
//   workbox 只缓存 /assets/（hashed 长缓存产物），API/SSE 永不缓存。
//   注意：service worker 需安全上下文（localhost / HTTPS）——LAN IP over http
//   下自动降级为普通网页（注册失败静默，不影响功能）。
//   RICK_DEV_NO_PWA=1 → 构建期禁用 PWA 插件（dev 树专用：service worker 的缓存
//   会掩盖"改了看不到"之外的问题，热更排障时应先排除 SW 变量；生产构建不受影响）。

/** 读环境变量（vite.config 由 esbuild 直跑、不经 tsc，故不依赖 @types/node） */
function envOf(key: string, fallback: string): string {
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
  const v = env?.[key];
  return v && v.trim() !== "" ? v.trim() : fallback;
}

// dev 联调目标：隔离 dev 实例（8414）为默认；指向别处时显式覆盖 RICK_DEV_API。
const devApiTarget = envOf("RICK_DEV_API", "http://127.0.0.1:8414");
const devVitePort = Number(envOf("RICK_DEV_VITE_PORT", "5173"));
const devViteHost = envOf("RICK_DEV_VITE_HOST", "0.0.0.0");
const disablePwa = envOf("RICK_DEV_NO_PWA", "0") === "1";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      // dev 构建（RICK_DEV_NO_PWA=1）跳过 SW 生成：避免缓存干扰前端热更排障
      disable: disablePwa,
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
    host: devViteHost,
    port: devVitePort,
    strictPort: true,
    proxy: {
      "/api": {
        target: devApiTarget,
        changeOrigin: true,
        // SSE（/api/events）必须关闭缓冲，否则事件被 vite 代理攒着不推
        ws: false,
      },
    },
  },
});
