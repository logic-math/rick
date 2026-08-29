// static.go 服务 rick web 前端静态资源（api-contract.md 静态资源节）：
//
//	GET /                → index.html（覆盖层 ~/.rick/web/dist 优先；否则 embed baseline）
//	GET /assets/*        → hashed 产物（长缓存 immutable）
//	SPA fallback          → 未知 GET 非 /api/ → index.html（前端路由）
//
// 缓存策略：
//   - index.html / manifest / sw.js / service-worker.js → no-cache（HTML 引用
//     hash 化 assets，必须每次校验新鲜度——自迭代 reload 依赖它）
//   - /assets/* → Cache-Control: public, max-age=31536000, immutable（Vite
//     产物文件名带内容 hash，内容变名字变，天然可永久缓存）
//
// 安全面（HTTP 暴露的文件服务）：
//   - 目录穿越清洗：filepath.Clean + 前缀复核 + 显式拒绝 ".." 元素与绝对路径
//   - 覆盖层路径不信任符号链接（Lstat 复核）
//   - embed FS 天然只读且编译期定界
//
// 覆盖优先级（自迭代机制，design-tree L4 终判）：overrideDir/index.html
// 存在 → 整个覆盖层生效（dist 树）；否则 embed baseline。逐文件混合会产
// 生「半个新半个旧」的脏状态——all-or-nothing 是刻意选择。
package web

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// staticNoCache 是必须每次校验新鲜度的文件（HTML 入口 + PWA 壳文件）。
// 自迭代的 frontend_reload → location.reload() 依赖 index.html no-cache：
// 新构建的 hash 化 assets 只有经由新 index.html 才被发现。
var staticNoCache = map[string]bool{
	"index.html":         true,
	"manifest.webmanifest": true,
	"manifest.json":      true,
	"sw.js":              true,
	"service-worker.js":  true,
	"registerSW.js":      true,
	"favicon.ico":        true, // 无 hash 的根级杂项
}

// StaticHandler serves the frontend: overlay (overrideDir) takes priority
// when its index.html exists; otherwise the embedded baseline (embedFS —
// pass the dist-rooted view, e.g. webassets.DistWeb()). An empty overrideDir
// or a missing overlay index.html falls back to embed.
//
// overrideDir semantics: files are served from the directory tree with
// traversal sanitized; symlinks are not followed (Lstat per component).
func StaticHandler(embedFS fs.FS, overrideDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, `{"error":{"code":"method_not_allowed","message":"static files are GET/HEAD only"}}`, http.StatusMethodNotAllowed)
			return
		}

		// Sanitize the request path into a clean relative slash path.
		rel := sanitizeStaticPath(r.URL.Path)
		if rel == "" {
			// Bare "/" → index.html.
			rel = "index.html"
		}

		useOverlay := false
		if overrideDir != "" {
			if idx, err := os.Lstat(filepath.Join(overrideDir, "index.html")); err == nil && !idx.IsDir() {
				useOverlay = true
			}
		}

		if useOverlay {
			if serveOverlay(w, r, overrideDir, rel) {
				return
			}
			// Overlay is missing the requested file → fall through to embed?
			// No: all-or-nothing by design (半个新半个旧的脏状态更糟).
			// Serve index.html (SPA fallback) or 404.
			if looksLikeAsset(rel) {
				http.NotFound(w, r)
				return
			}
			serveOverlay(w, r, overrideDir, "index.html")
			return
		}

		serveEmbed(w, r, embedFS, rel)
	})
}

// serveOverlay serves one file from the overlay dir (returns false when the
// file does not exist / is not a regular file).
func serveOverlay(w http.ResponseWriter, r *http.Request, dir, rel string) bool {
	full, ok := resolveUnder(dir, rel)
	if !ok {
		return false
	}
	fi, err := os.Lstat(full)
	if err != nil || fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return false
	}
	f, err := os.Open(full)
	if err != nil {
		return false
	}
	defer f.Close()
	setStaticCaching(w, rel)
	http.ServeContent(w, r, path.Base(rel), fi.ModTime(), f)
	return true
}

// serveEmbed serves one file from the embedded FS with SPA fallback: a
// missing non-asset path falls back to index.html (frontend routing); a
// missing asset is a genuine 404.
func serveEmbed(w http.ResponseWriter, r *http.Request, fsys fs.FS, rel string) {
	data, err := fs.ReadFile(fsys, rel)
	if err != nil {
		if looksLikeAsset(rel) {
			http.NotFound(w, r)
			return
		}
		// SPA fallback: unknown route → index.html.
		if idx, ierr := fs.ReadFile(fsys, "index.html"); ierr == nil {
			setStaticCaching(w, "index.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(idx)
			return
		}
		http.NotFound(w, r)
		return
	}
	setStaticCaching(w, rel)
	// Content-Type via extension (ServeContent-free path for byte slices).
	if ct := mimeByExt(rel); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// setStaticCaching applies the caching policy: hashed assets immutable for
// a year; entry documents no-cache.
func setStaticCaching(w http.ResponseWriter, rel string) {
	if looksLikeAsset(rel) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	if staticNoCache[rel] {
		w.Header().Set("Cache-Control", "no-cache")
	}
}

// looksLikeAsset reports whether the path is a hashed build artifact
// (assets/<name> 或根级 hash 文件——Vite 产物约定).
func looksLikeAsset(rel string) bool {
	if strings.HasPrefix(rel, "assets/") {
		return true
	}
	// Root-level hashed artifacts (e.g. workbox-<hash>.js from vite-plugin-pwa).
	base := path.Base(rel)
	if strings.HasPrefix(base, "workbox-") && strings.HasSuffix(base, ".js") {
		return true
	}
	return false
}

// sanitizeStaticPath cleans a URL path into a safe relative slash path:
// absolute → cleaned of leading "/", ".." elements rejected outright.
// Returns "" for the root ("/").
func sanitizeStaticPath(urlPath string) string {
	if urlPath == "" || urlPath == "/" {
		return ""
	}
	// Reject any traversal attempt before cleaning (predictability).
	for _, part := range strings.Split(urlPath, "/") {
		if part == ".." {
			return "" // treated as root → index.html (harmless default)
		}
	}
	cleaned := path.Clean("/" + urlPath) // forces a leading "/" then strips it
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." || cleaned == "/" {
		return ""
	}
	return cleaned
}

// resolveUnder joins dir + rel after re-validating that the result stays
// under dir (defense in depth: rel came from sanitizeStaticPath, but the
// filesystem join is re-checked).
func resolveUnder(dir, rel string) (string, bool) {
	if rel == "" {
		return "", false
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", false
	}
	if absFull != absDir && !strings.HasPrefix(absFull, absDir+string(filepath.Separator)) {
		return "", false
	}
	return absFull, true
}

// mimeByExt maps the extensions the dist actually carries. The standard
// library mime.TypeByExtension covers most; this keeps the fallback explicit
// for the SPA's own files.
func mimeByExt(rel string) string {
	switch strings.ToLower(path.Ext(rel)) {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json"
	case ".webmanifest":
		return "application/manifest+json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".woff2":
		return "font/woff2"
	case ".map":
		return "application/json"
	default:
		return ""
	}
}
