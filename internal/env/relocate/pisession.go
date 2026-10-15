package relocate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrSessionDirExists reports that a RenameSessionDir target already holds
// sessions — merging is the import orchestrator's policy decision, not this
// primitive's.
var ErrSessionDirExists = fmt.Errorf("target session directory already exists")

// EscapeDir mirrors pi's session-directory naming (session-manager.js):
// strip the leading "/", replace "/", "\", and ":" with "-", then wrap the
// result in "--". Underscores are NOT replaced and there is no truncation.
// Locked against the real tree (job_46 research-L2 leaf A: 12-case
// experiment matrix; production dirs like
// --workdir-sunquan20-AI_CODING-rick-- and --tmp-rick_accept_7YgSff--).
func EscapeDir(cwd string) string {
	s := strings.TrimPrefix(cwd, "/")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return "--" + s + "--"
}

// RewriteSessionHeader rewrites the first line (the session header) of a pi
// session .jsonl: cwd is set to newCwd, and parentSession (a fork pointer
// stored as an absolute path) is remapped through pm when it matches. Every
// other line is copied through verbatim — message payloads may embed origin
// paths in prose, but pi's resume never parses them (research-L2 A2).
//
// The rewrite is atomic (sibling tmp file + rename) and preserves the file
// mode. An empty file is a no-op; a malformed first line is an error and
// leaves the original untouched.
func RewriteSessionHeader(path, newCwd string, pm *PathMap) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat session %s: %w", path, err)
	}
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open session %s: %w", path, err)
	}
	defer src.Close()

	tmp := path + ".reloc.tmp"
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open tmp %s: %w", tmp, err)
	}
	defer func() {
		if dst != nil {
			dst.Close()
			os.Remove(tmp)
		}
	}()

	br := bufio.NewReader(src)
	first := true
	for {
		line, rerr := br.ReadBytes('\n')
		if len(line) > 0 {
			var out []byte
			if first {
				out, err = rewriteHeaderLine(line, newCwd, pm)
				if err != nil {
					return err
				}
				first = false
			} else {
				out = line
			}
			if _, werr := dst.Write(out); werr != nil {
				return fmt.Errorf("write tmp %s: %w", tmp, werr)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("read session %s: %w", path, rerr)
		}
	}
	if err := dst.Close(); err != nil {
		dst = nil
		return fmt.Errorf("close tmp %s: %w", tmp, err)
	}
	dst = nil
	if err := os.Chmod(tmp, fi.Mode().Perm()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("chmod tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// rewriteHeaderLine parses one header line, rewrites cwd/parentSession, and
// re-marshals it. Unknown fields are preserved (map round-trip with
// UseNumber keeping number literals intact); key order may differ from the
// original — pi only ever JSON.parses the line.
func rewriteHeaderLine(line []byte, newCwd string, pm *PathMap) ([]byte, error) {
	trimmed := bytes.TrimRight(line, "\r\n")
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var hdr map[string]any
	if err := dec.Decode(&hdr); err != nil {
		return nil, fmt.Errorf("parse session header: %w", err)
	}
	if hdr["type"] != "session" {
		// Defensive: pi writes the header first, but if a file ever starts
		// with something else, fail loudly instead of corrupting it.
		return nil, fmt.Errorf("first line is not a session header (type=%v)", hdr["type"])
	}
	hdr["cwd"] = newCwd
	if ps, ok := hdr["parentSession"].(string); ok && ps != "" && pm != nil {
		if mapped, hit := pm.MapPath(ps); hit {
			hdr["parentSession"] = mapped
		}
	}
	out, err := json.Marshal(hdr)
	if err != nil {
		return nil, fmt.Errorf("marshal session header: %w", err)
	}
	if bytes.HasSuffix(line, []byte("\n")) {
		out = append(out, '\n')
	}
	return out, nil
}

// RenameSessionDir renames the session directory of oldCwd to the escaped
// name of newCwd under <agentDir>/sessions — after a relocation, pi's
// --continue/--session/--session-id lookups key off the (new) cwd
// (research-L2 A3/A4: without the rename, history is unreachable).
// A missing source directory is a no-op; an existing target is
// ErrSessionDirExists (merge semantics belong to the import orchestrator).
func RenameSessionDir(agentDir, oldCwd, newCwd string) error {
	sessions := filepath.Join(agentDir, "sessions")
	oldDir := filepath.Join(sessions, EscapeDir(oldCwd))
	newDir := filepath.Join(sessions, EscapeDir(newCwd))
	if oldDir == newDir {
		return nil
	}
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Lstat(newDir); err == nil {
		return fmt.Errorf("%w: %s", ErrSessionDirExists, newDir)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("rename session dir %s -> %s: %w", oldDir, newDir, err)
	}
	return nil
}
