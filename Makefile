.PHONY: web-dist web-dev

# 构建前端并产出 web/dist/（产物提交仓库——node 不进 rick 构建链）。
# 前置：web/package-lock.json 已入库（首次开发用 npm install 生成后提交）。
web-dist:
	cd web && npm ci && npm run build

# 前端开发态：vite dev server（/api 代理到 rick web 服务 127.0.0.1:6137）。
web-dev:
	cd web && npm run dev
