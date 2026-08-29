import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// rick web 前端构建配置。
// - build.outDir=dist：产物提交仓库（node 不进 rick 构建链——AdGuardHome 模式）
// - build.manifest：输出 .vite/manifest.json 供后端 asset 映射（ygncode 方案）
// - dev proxy：开发态 /api 代理到 rick web 服务（默认 127.0.0.1:6137）
// - PWA（vite-plugin-pwa）依赖已安装但未激活——task12 统一启用
export default defineConfig({
  plugins: [react(), tailwindcss()],
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
