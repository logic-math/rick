package relocate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// buildArchiveFixture creates a tree exercising every entry kind: nested
// dirs, an empty dir, exec-bit file, plain file, relative symlink, absolute
// symlink, ".." symlink (npm .bin pattern), and a 5MB random file.
func buildArchiveFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "a/b"), 0755))
	must(os.MkdirAll(filepath.Join(root, "empty"), 0700))
	must(os.WriteFile(filepath.Join(root, "a/b/file.txt"), []byte("hello"), 0644))
	must(os.WriteFile(filepath.Join(root, "a/exec.sh"), []byte("#!/bin/sh\n"), 0755))
	must(os.Symlink("b/file.txt", filepath.Join(root, "a/rel-link")))
	must(os.Symlink("/etc/hostname", filepath.Join(root, "a/abs-link")))
	must(os.Symlink("../pkg/cli.js", filepath.Join(root, "a/dotdot-link")))
	big := make([]byte, 5*1024*1024)
	if _, err := rand.Read(big); err != nil {
		t.Fatal(err)
	}
	must(os.WriteFile(filepath.Join(root, "big.bin"), big, 0644))
	return root
}

// treeSnapshot walks dir and records relpath -> kind, mode perm, linkname,
// content sha (files only). Used for roundtrip equality assertions.
func treeSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		fi, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			out[rel] = "symlink->" + link
		case fi.IsDir():
			out[rel] = "dir:" + fi.Mode().Perm().String()
		default:
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			out[rel] = "file:" + fi.Mode().Perm().String() + ":" + hex.EncodeToString(sum[:8])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func snapDiff(a, b map[string]string) []string {
	var diffs []string
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		if a[k] != b[k] {
			diffs = append(diffs, k+": "+a[k]+" != "+b[k])
		}
	}
	return diffs
}

func TestArchiveRoundTrip(t *testing.T) {
	src := buildArchiveFixture(t)
	var buf bytes.Buffer
	sha, files, payload, err := WriteArchive(&buf, []Item{{SourcePath: src, ArchivePath: "workspaces/ws"}}, nil)
	if err != nil {
		t.Fatalf("WriteArchive: %v", err)
	}
	if files != 3 { // file.txt, exec.sh, big.bin (symlinks/dirs carry no payload)
		t.Errorf("files count: got %d want 3", files)
	}
	if payload != int64(len("hello")+len("#!/bin/sh\n")+5*1024*1024) {
		t.Errorf("payload bytes: got %d", payload)
	}
	// SHA must equal hashing the produced bytes independently.
	h := sha256.Sum256(buf.Bytes())
	if got := hex.EncodeToString(h[:]); got != sha {
		t.Fatalf("returned sha256 mismatch: %s vs %s", got, sha)
	}

	dest := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(buf.Bytes()), dest); err != nil {
		t.Fatalf("ExtractArchive: %v", err)
	}
	want := treeSnapshot(t, src)
	got := treeSnapshot(t, filepath.Join(dest, "workspaces/ws"))
	if diffs := snapDiff(want, got); len(diffs) > 0 {
		for _, d := range diffs {
			t.Errorf("roundtrip diff: %s", d)
		}
	}
}

func TestArchiveMultipleItemsAndNestedArchivePaths(t *testing.T) {
	src1 := buildArchiveFixture(t)
	src2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(src2, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _, _, err := WriteArchive(&buf, []Item{
		{SourcePath: src1, ArchivePath: "workspaces/one"},
		{SourcePath: src2, ArchivePath: "state"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(buf.Bytes()), dest); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(dest, "workspaces/one/a/b/file.txt"),
		filepath.Join(dest, "state/x.txt"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestArchiveExcludePrunesSubtree(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "web/node_modules/vite/index.js"), "v")
	mustWrite(t, filepath.Join(src, "web/dist/index.html"), "d")
	mustWrite(t, filepath.Join(src, "main.go"), "m")
	var buf bytes.Buffer
	keep := func(rel string) bool { return rel != "web/node_modules" }
	_, files, _, err := WriteArchive(&buf, []Item{{SourcePath: src, ArchivePath: "ws", Match: keep}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if files != 2 {
		t.Fatalf("excluded dir must prune subtree: got %d files, want 2", files)
	}
	dest := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(buf.Bytes()), dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "ws/web/node_modules")); !os.IsNotExist(err) {
		t.Error("node_modules must not be extracted")
	}
	if _, err := os.Stat(filepath.Join(dest, "ws/web/dist/index.html")); err != nil {
		t.Error("web/dist/index.html must survive")
	}
}

func TestArchiveGlobalMatchFallback(t *testing.T) {
	// Items without their own Match fall back to the WriteArchive argument.
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "skip.txt"), "s")
	mustWrite(t, filepath.Join(src, "keep.txt"), "k")
	var buf bytes.Buffer
	fallback := func(rel string) bool { return rel != "skip.txt" }
	_, files, _, err := WriteArchive(&buf, []Item{{SourcePath: src, ArchivePath: "ws"}}, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if files != 1 {
		t.Fatalf("global fallback match must apply: got %d files, want 1", files)
	}
}

// craftArchive builds a gzip tar from explicit headers/payloads for the
// traversal-attack tests.
func craftArchive(t *testing.T, entries []struct {
	hdr  tar.Header
	data []byte
}) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		h := e.hdr
		h.Size = int64(len(e.data))
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if len(e.data) > 0 {
			if _, err := tw.Write(e.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractRejectsTraversalNames(t *testing.T) {
	cases := []struct {
		name string
		hdr  tar.Header
	}{
		{"dotdot", tar.Header{Name: "../evil", Typeflag: tar.TypeReg, Mode: 0644}},
		{"nested-dotdot", tar.Header{Name: "a/../../evil", Typeflag: tar.TypeReg, Mode: 0644}},
		{"absolute", tar.Header{Name: "/tmp/evil", Typeflag: tar.TypeReg, Mode: 0644}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := craftArchive(t, []struct {
				hdr  tar.Header
				data []byte
			}{{hdr: tc.hdr, data: []byte("x")}})
			dest := t.TempDir()
			if err := ExtractArchive(bytes.NewReader(data), dest); err == nil {
				t.Fatal("expected extraction to reject malicious entry name")
			}
		})
	}
}

func TestExtractRejectsWriteThroughSymlink(t *testing.T) {
	// 1) create symlink s -> /tmp (escape), 2) write file s/evil.
	data := craftArchive(t, []struct {
		hdr  tar.Header
		data []byte
	}{
		{hdr: tar.Header{Name: "s", Typeflag: tar.TypeSymlink, Linkname: "/tmp", Mode: 0777}},
		{hdr: tar.Header{Name: "s/evil", Typeflag: tar.TypeReg, Mode: 0644}, data: []byte("x")},
	})
	dest := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(data), dest); err == nil {
		t.Fatal("write-through-symlink must be rejected")
	}
	if _, err := os.Stat(filepath.Join("/tmp/evil")); !os.IsNotExist(err) {
		t.Fatal("nothing may escape the dest dir")
	}
}

func TestExtractFileOverSymlinkDoesNotFollow(t *testing.T) {
	// A later file entry shadowing an earlier symlink entry: the link must
	// never be followed out of dest. Our semantics: the symlink is removed
	// and the regular file takes its place; nothing may be created at the
	// link's target outside dest.
	data := craftArchive(t, []struct {
		hdr  tar.Header
		data []byte
	}{
		{hdr: tar.Header{Name: "escape", Typeflag: tar.TypeSymlink, Linkname: "/tmp/rick-escape-probe", Mode: 0777}},
		{hdr: tar.Header{Name: "escape", Typeflag: tar.TypeReg, Mode: 0644}, data: []byte("x")},
	})
	dest := t.TempDir()
	_ = os.Remove("/tmp/rick-escape-probe")
	if err := ExtractArchive(bytes.NewReader(data), dest); err != nil {
		t.Fatalf("shadowing a symlink with a file must succeed safely: %v", err)
	}
	if _, err := os.Stat("/tmp/rick-escape-probe"); !os.IsNotExist(err) {
		t.Fatal("symlink target outside dest must not be created")
	}
	fi, err := os.Lstat(filepath.Join(dest, "escape"))
	if err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("dest/escape must now be a regular file: %v %+v", err, fi)
	}
	got, err := os.ReadFile(filepath.Join(dest, "escape"))
	if err != nil || string(got) != "x" {
		t.Fatalf("file content must be written at dest/escape: %q err=%v", got, err)
	}
}

func TestExtractPreservesEmptyDirAndMode(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "run.sh"), []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, _, _, err := WriteArchive(&buf, []Item{{SourcePath: src, ArchivePath: "ws"}}, nil); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(buf.Bytes()), dest); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dest, "ws/empty"))
	if err != nil {
		t.Fatalf("empty dir must survive roundtrip: %v", err)
	}
	if fi.Mode().Perm() != 0700 {
		t.Errorf("empty dir mode: got %v want -rwx------", fi.Mode().Perm())
	}
	fi, err = os.Stat(filepath.Join(dest, "ws/run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0755 {
		t.Errorf("exec mode: got %v want -rwxr-xr-x", fi.Mode().Perm())
	}
}

func TestExtractRerunOverwrites(t *testing.T) {
	// Import is expected to be idempotent/re-runnable: extracting over an
	// existing tree replaces regular files instead of failing.
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "f.txt"), "v1")
	var buf bytes.Buffer
	if _, _, _, err := WriteArchive(&buf, []Item{{SourcePath: src, ArchivePath: "ws"}}, nil); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(src, "f.txt"), "v2")
	var buf2 bytes.Buffer
	if _, _, _, err := WriteArchive(&buf2, []Item{{SourcePath: src, ArchivePath: "ws"}}, nil); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(buf.Bytes()), dest); err != nil {
		t.Fatal(err)
	}
	if err := ExtractArchive(bytes.NewReader(buf2.Bytes()), dest); err != nil {
		t.Fatalf("second extract must succeed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "ws/f.txt"))
	if err != nil || string(got) != "v2" {
		t.Fatalf("re-extract must overwrite: got %q err=%v", got, err)
	}
}

func TestArchiveLargeFileStreams(t *testing.T) {
	// 5MB in the fixture is enough to prove streaming (no buffering issues);
	// assert the writer handles payloads far beyond any line/token limit.
	src := buildArchiveFixture(t)
	var buf bytes.Buffer
	_, files, _, err := WriteArchive(&buf, []Item{{SourcePath: src, ArchivePath: "ws"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 || buf.Len() < 1024*1024 {
		t.Fatalf("5MB payload must compress below raw size but stay >1MB marker: files=%d len=%d", files, buf.Len())
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
