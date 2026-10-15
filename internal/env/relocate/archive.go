package relocate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Item maps one on-disk tree into the archive: SourcePath is walked and
// stored under ArchivePath (every entry becomes <ArchivePath>/<rel>).
//
// Match is the per-root exclude hook with KEEP semantics — true keeps the
// path, false skips it; skipping a directory prunes its whole subtree. It
// receives paths relative to SourcePath (slash-separated, "" is never
// passed). nil falls back to the package-level match argument of
// WriteArchive. Together these let one call archive heterogeneous roots
// (workspaces / state / pi agent) with root-scoped rules — build one via
// Matcher.MatchFunc(Root...).
type Item struct {
	SourcePath  string
	ArchivePath string // slash-separated archive prefix, no leading slash
	Match       func(relPath string) bool
	// OnUnreadable, when set, is invoked for entries the walk cannot read
	// (permission, vanished mid-walk, …) with the slash-rel path and the
	// error. Returning true skips the entry (a directory skips its whole
	// subtree) instead of failing the dump — real machines have permission
	// holes inside otherwise packable trees. Returning false (or a nil
	// callback) preserves the hard-fail behavior.
	OnUnreadable func(relPath string, err error) bool
}

// WriteArchive streams items into w as a PAX tar.gz, hashing the compressed
// bytes with SHA-256 as they are written. It returns the hex digest, the
// number of regular files archived, and the payload byte count.
//
// Symlinks are stored as symlinks (never followed), file modes are
// preserved, empty directories are kept, and non-regular files (sockets,
// fifos, devices) are skipped — they are not portable across machines.
func WriteArchive(w io.Writer, items []Item, match func(relPath string) bool) (sha256Hex string, files int, bytes int64, err error) {
	h := sha256.New()
	mw := io.MultiWriter(w, h)
	gz := gzip.NewWriter(mw)
	tw := tar.NewWriter(gz)
	var payload int64
	for _, it := range items {
		n, b, err := writeItem(tw, it, match)
		if err != nil {
			return "", 0, 0, fmt.Errorf("archive %s: %w", it.SourcePath, err)
		}
		files += n
		payload += b
	}
	if err := tw.Close(); err != nil {
		return "", 0, 0, fmt.Errorf("close tar stream: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", 0, 0, fmt.Errorf("close gzip stream: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), files, payload, nil
}

func writeItem(tw *tar.Writer, it Item, fallback func(relPath string) bool) (files int, payload int64, err error) {
	abs, err := filepath.Abs(it.SourcePath)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve source path: %w", err)
	}
	match := it.Match
	if match == nil {
		match = fallback
	}
	err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if it.OnUnreadable != nil {
				rel, rerr := filepath.Rel(abs, p)
				if rerr == nil && it.OnUnreadable(filepath.ToSlash(rel), err) {
					if d != nil && d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}
			}
			return err
		}
		rel, err := filepath.Rel(abs, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		name := it.ArchivePath
		if rel != "." {
			if match != nil && !match(rel) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			name = name + "/" + rel
		}
		fi, err := d.Info()
		if err != nil {
			if it.OnUnreadable != nil && it.OnUnreadable(rel, err) {
				return nil
			}
			return err
		}
		link := ""
		if fi.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(p)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", p, err)
			}
		} else if !fi.Mode().IsRegular() && !fi.IsDir() {
			// Sockets, fifos, devices: not portable, skip silently.
			return nil
		}
		hdr, err := tar.FileInfoHeader(fi, link)
		if err != nil {
			return fmt.Errorf("header %s: %w", p, err)
		}
		hdr.Name = name
		hdr.Format = tar.FormatPAX
		if fi.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}
		// Open regular files BEFORE writing the header so an unreadable
		// file (permission hole) can be skipped without leaving an empty
		// tar entry behind — OnUnreadable decides skip-vs-fail.
		var src *os.File
		if fi.Mode().IsRegular() {
			src, err = os.Open(p)
			if err != nil {
				if it.OnUnreadable != nil && it.OnUnreadable(rel, err) {
					return nil
				}
				return fmt.Errorf("open %s: %w", p, err)
			}
			defer src.Close()
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write header %s: %w", name, err)
		}
		if src != nil {
			n, err := io.Copy(tw, src)
			if err != nil {
				return fmt.Errorf("copy %s: %w", p, err)
			}
			files++
			payload += n
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return files, payload, nil
}

// ExtractArchive unpacks the tar.gz from r under dest, restoring modes and
// symlinks. It is safe to re-run over an existing tree (regular files are
// overwritten — import is idempotent). Defense against malicious archives:
// entry names are validated (no absolute paths, no ".."), no entry is ever
// written through a symlink ancestor, and an existing symlink at a file
// target is removed rather than followed. Absolute and ".." symlink
// LINKNAMEs are allowed (legit state: npm .bin links, pinned-release links)
// because no write ever follows them.
func ExtractArchive(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		name := path.Clean(strings.TrimSuffix(filepath.ToSlash(hdr.Name), "/"))
		if err := safeEntryName(name); err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := ensureNoSymlinkAncestor(dest, name); err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			if err := chmodIfMode(target, hdr.Mode); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := ensureNoSymlinkAncestor(dest, name); err != nil {
				return err
			}
			// Never follow an existing symlink at the target itself.
			if fi, err := os.Lstat(target); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(target); err != nil {
					return fmt.Errorf("remove symlink %s: %w", target, err)
				}
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent of %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("write %s: %w", target, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("close %s: %w", target, err)
			}
			if err := chmodIfMode(target, hdr.Mode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := ensureNoSymlinkAncestor(dest, name); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent of %s: %w", target, err)
			}
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				// A directory may live there from an earlier run; a symlink
				// can't replace a non-empty dir, which is a real conflict.
				return fmt.Errorf("clear %s for symlink: %w", target, err)
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", target, hdr.Linkname, err)
			}
		case tar.TypeLink:
			linkName := path.Clean(strings.TrimSuffix(filepath.ToSlash(hdr.Linkname), "/"))
			if err := safeEntryName(linkName); err != nil {
				return err
			}
			source := filepath.Join(dest, filepath.FromSlash(linkName))
			if err := ensureNoSymlinkAncestor(dest, name); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Link(source, target); err != nil {
				return fmt.Errorf("hardlink %s -> %s: %w", target, source, err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type %c for %s", hdr.Typeflag, name)
		}
	}
}

// safeEntryName rejects absolute paths, ".." segments, and empty artifacts
// after cleaning.
func safeEntryName(name string) error {
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, "/") {
		return fmt.Errorf("refusing unsafe tar entry name %q", name)
	}
	if strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
		return fmt.Errorf("refusing tar entry name escaping dest: %q", name)
	}
	return nil
}

// ensureNoSymlinkAncestor walks every proper ancestor of name under dest and
// fails if any exists as a symlink — this blocks "symlink then write through
// it" traversal while still allowing symlink entries themselves (whose
// linknames may legitimately point outside, e.g. npm .bin or pinned
// releases; nothing ever follows those links during extraction).
func ensureNoSymlinkAncestor(dest, name string) error {
	segs := strings.Split(name, "/")
	for i := 1; i < len(segs); i++ {
		ancestor := filepath.Join(dest, filepath.FromSlash(strings.Join(segs[:i], "/")))
		if fi, err := os.Lstat(ancestor); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink %s", ancestor)
		}
	}
	return nil
}

// chmodIfMode applies the recorded permission bits (tar headers from other
// writers may carry Mode 0 — keep the created default then).
func chmodIfMode(target string, mode int64) error {
	perm := os.FileMode(mode).Perm()
	if perm == 0 {
		return nil
	}
	if err := os.Chmod(target, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", target, err)
	}
	return nil
}
