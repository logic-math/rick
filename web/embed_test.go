package web

import (
	"io/fs"
	"testing"
)

// TestEmbedSrcFSContents 锁定 SrcFS 路径契约：web/ 仓库布局（src 树 + 根构建配置）。
// customize 抽取链路（env.DeployWebScaffold）依赖根配置文件在 SrcFS 内。
func TestEmbedSrcFSContents(t *testing.T) {
	must := []string{
		"src/main.tsx",
		"src/App.tsx",
		"src/styles/theme.css",
		"src/components/starfield/StarfieldBackground.tsx",
		"src/components/starfield/Portal.tsx",
		"src/components/starfield/Saucer.tsx",
		"public/icon-192.png",
		"public/icon-512.png",
		"public/icon-maskable-512.png",
		"package.json",
		"vite.config.ts",
		"tsconfig.json",
		"index.html",
	}
	for _, p := range must {
		if _, err := fs.Stat(SrcFS, p); err != nil {
			t.Errorf("SrcFS 缺 %s: %v", p, err)
		}
	}
}

// TestEmbedDistWebContents 锁定 DistWeb() 路径契约：以 dist/ 为根的构建产物视图。
// task13 静态资源服务 embed baseline 依赖 index.html 在根、assets/ 与 .vite/manifest.json 就位。
func TestEmbedDistWebContents(t *testing.T) {
	sub := DistWeb()
	must := []string{
		"index.html",
		"assets",
		".vite/manifest.json",
	}
	for _, p := range must {
		if _, err := fs.Stat(sub, p); err != nil {
			t.Errorf("DistWeb() 缺 %s: %v", p, err)
		}
	}
}
