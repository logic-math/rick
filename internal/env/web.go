// web.go 承载 env 职责 3 的 web 扩展：前端基座部署（自迭代机制的 env 侧落地）。
//
// 「embed 混合分发 + 自迭代」（design-tree L4 终判 J4.1）：
//   - baseline：web 包 embed 的 SrcFS（前端源码 + 根构建配置）与 DistFS（构建产物）
//   - 覆盖层：~/.rick/web/（src/ + dist/ + 根配置文件）——agent 会话可改造，
//     构建后由 rick web 的静态资源服务优先接管（task13 static.go）
//   - 复位：删覆盖层回 baseline
//
// DeployWebScaffold 把 SrcFS 幂等抽取到 ~/.rick/web/（deployRickAgents 同款
// rick-managed 标记语义——但这里是目录级标记而非文件内 frontmatter：一个
// .rick-managed marker 文件位于 ~/.rick/web/.rick-managed）。
package env

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	webembed "github.com/sunquan/rick/web"
)

// webScaffoldMarker is the directory-level rick-managed marker: its presence
// in ~/.rick/web/ means the scaffold was deployed (and is ours to overwrite
// on reset). Absent marker = no scaffold deployed yet (first run).
const webScaffoldMarker = ".rick-managed"

// WebStateDir returns the machine-level web state directory (~/.rick/web).
// It follows HOME (not RICK_PI_AGENT_DIR) — same isolation contract as
// internal/web.WebStateDir; duplicated here to keep env self-contained
// (importing internal/web from env would invert the layer direction).
func WebStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick", "web")
}

// DeployWebScaffold extracts the embedded frontend source baseline
// (webembed.SrcFS: src/** + package.json/vite.config.ts/tsconfig.json/
// index.html) into ~/.rick/web/, creating the writable customization layer.
// It is idempotent: if the scaffold marker already exists it is a no-op and
// returns (false, nil) — user customizations are never clobbered. On a fresh
// deploy it returns (true, nil).
//
// The extracted tree is what an agent session edits; running
// `npm ci && npm run build` there (node is already a rick env dependency)
// produces ~/.rick/web/dist which the static handler serves with priority
// over the embedded baseline, and the watcher broadcasts frontend_reload.
func DeployWebScaffold() (bool, error) {
	dir := WebStateDir()
	if dir == "" {
		return false, fmt.Errorf("resolve web state dir: home unavailable")
	}

	marker := filepath.Join(dir, webScaffoldMarker)
	if _, err := os.Stat(marker); err == nil {
		return false, nil // already deployed; never clobber customizations
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat scaffold marker: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("create web state dir: %w", err)
	}

	if err := extractFS(webembed.SrcFS, ".", dir); err != nil {
		return false, fmt.Errorf("extract web scaffold: %w", err)
	}

	if err := os.WriteFile(marker, []byte("rick-managed web scaffold (job_36)\n"), 0644); err != nil {
		return false, fmt.Errorf("write scaffold marker: %w", err)
	}
	return true, nil
}

// ResetWebCustomization removes the customization overlay
// (~/.rick/web/src, ~/.rick/web/dist and the root config files) so rick web
// serves the embedded baseline again. The machine-level registries
// (~/.rick/web.json, ~/.rick/web/sessions.json) are preserved — reset only
// affects the frontend customization layer, never user data.
func ResetWebCustomization() error {
	dir := WebStateDir()
	if dir == "" {
		return fmt.Errorf("resolve web state dir: home unavailable")
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // nothing to reset
	}

	// Remove scaffold tree except sessions.json (registry state).
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read web state dir: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == "sessions.json" {
			continue // machine-level session registry: preserved by contract
		}
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	return nil
}

// CheckWebEmbed verifies the embedded web assets are present and readable
// (compile-time go:embed guarantees presence; this is a smoke check for
// tooling like `rick tools` wiring — missing FS means a broken build).
func CheckWebEmbed() error {
	src, err := fs.ReadDir(webembed.SrcFS, "src")
	if err != nil {
		return fmt.Errorf("embedded web src missing: %w", err)
	}
	if len(src) == 0 {
		return fmt.Errorf("embedded web src is empty")
	}
	if _, err := fs.Stat(webembed.SrcFS, "package.json"); err != nil {
		return fmt.Errorf("embedded web package.json missing: %w", err)
	}
	if _, err := fs.Stat(webembed.DistFS, "dist/index.html"); err != nil {
		return fmt.Errorf("embedded web dist missing: %w", err)
	}
	return nil
}

// extractFS walks fsys from root prefix and writes every file into dst,
// preserving relative paths (dirs created as needed). Files only — the
// embedded FS has no symlinks.
func extractFS(fsys fs.FS, root, dst string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		target := filepath.Join(dst, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
		return nil
	})
}
