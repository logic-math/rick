// Package web 内嵌 rick web 前端（源码 + 构建产物）。
//
// 两个导出 FS 是「embed 混合分发 + 自迭代」机制的基座：
//   - SrcFS：前端源码全树 + public/（PWA 图标与 manifest 源）+ 根构建配置
//     （package.json/vite.config.ts/tsconfig.json/index.html）——`rick web customize`
//     （env.DeployWebScaffold）把它幂等抽取到 ~/.rick/web/ 供 agent 会话改造后
//     npm build（customize 链路依赖根配置与 public/ 都在 SrcFS 内，多 directive
//     累积缺一不可——vite build 会把 public/ 拷进 dist，缺它则覆盖层构建无 PWA 图标）
//   - DistFS：构建产物 baseline——无覆盖层（~/.rick/web/dist）时由静态资源服务兜底
//
// 路径契约（消费方按此取用）：
//   - SrcFS 的路径 = web/ 仓库布局（src/main.tsx、public/icon-192.png、
//     package.json、vite.config.ts…）
//   - DistFS 的路径带 dist/ 前缀（dist/index.html、dist/assets/…）——用 DistWeb() 拿到
//     以 dist/ 为根的干净视图
//
// dist 提交仓库是 rick 哲学：node 不进 rick 构建链（AdGuardHome 模式），
// 普通 `go build` 即得完整可服务的二进制。
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:src
//go:embed all:public
//go:embed package.json vite.config.ts tsconfig.json index.html
var SrcFS embed.FS

//go:embed all:dist
var DistFS embed.FS

// DistWeb 返回以 dist/ 为根的构建产物视图（index.html 在根，assets/…）。
// task13 静态资源服务 embed baseline 时用它；出错意味着 embed 指令被破坏（编译期保证）。
func DistWeb() fs.FS {
	sub, err := fs.Sub(DistFS, "dist")
	if err != nil {
		panic("web: embedded dist missing: " + err.Error())
	}
	return sub
}
