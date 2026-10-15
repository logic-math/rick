package relocate

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to path atomically (create parent dirs, write
// to a sibling tmp file, rename over the target) so a crash never leaves a
// half-written state file behind. Mirrors internal/web's writeFileAtomic —
// duplicated here because the web package cannot be imported (cycle: web
// already imports relocate for marker ids).
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create parent dir %s: %w", dir, err)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
