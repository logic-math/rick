package relocate

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Archive format identity — recorded in every manifest and validated on
// read, so an import never silently consumes a foreign archive.
const (
	DumpFormat  = "rick-dump"
	DumpVersion = 1
)

// Sessions policy values (dump --sessions=...): which pi session
// directories travel with the archive.
const (
	// SessionsPolicyNone leaves all pi session history behind.
	SessionsPolicyNone = "none"
	// SessionsPolicyWorkspaces packs only the escaped session dirs of the
	// packed workspaces — what web session resume needs.
	SessionsPolicyWorkspaces = "workspaces"
	// SessionsPolicyAll packs the whole sessions tree (1.9G on the origin
	// machine as of job_46 — expensive but complete).
	SessionsPolicyAll = "all"
)

// OriginInfo describes the machine that produced the dump. Paths recorded
// here (Home, and per-workspace OriginPath) are the remap sources at import
// time.
type OriginInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	User     string `json:"user"`
	Home     string `json:"home"`
}

// WorkspaceInfo records one packed workspace: where it lived, and both of
// its identity spellings (MarkerID = stable marker id, LegacyID =
// sha1(OriginPath)[:8] as referenced by the origin machine's state files).
type WorkspaceInfo struct {
	Name       string `json:"name"`
	OriginPath string `json:"origin_path"`
	MarkerID   string `json:"marker_id,omitempty"`
	LegacyID   string `json:"legacy_id,omitempty"`
}

// ExcludedEntry is one exclusion hit recorded for auditability — the dump
// stays honest about what it left behind and why.
type ExcludedEntry struct {
	Path   string `json:"path"`
	Root   string `json:"root,omitempty"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

// BinaryInfo records one cross-compiled rick binary embedded in the
// archive under bin/<goos>-<goarch>/rick.
type BinaryInfo struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Manifest is the dump's self-description, stored as manifest.json at the
// archive root. Field names are a format contract across rick versions.
type Manifest struct {
	Format         string          `json:"format"`
	Version        int             `json:"version"`
	CreatedAt      time.Time       `json:"created_at"`
	Origin         OriginInfo      `json:"origin"`
	StateDirOrigin string          `json:"state_dir_origin"`
	Workspaces     []WorkspaceInfo `json:"workspaces"`
	SessionsPolicy string          `json:"sessions_policy"`
	Excluded       []ExcludedEntry `json:"excluded"`
	Binaries       []BinaryInfo    `json:"binaries"`
	RickVersion    string          `json:"rick_version"`
}

// WriteManifest writes m as indented JSON (human-auditable, like rick's
// other state files).
func WriteManifest(w io.Writer, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// ReadManifest parses a manifest and enforces the format identity — a
// foreign "format" value is rejected rather than best-effort parsed.
func ReadManifest(r io.Reader) (*Manifest, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Format != DumpFormat {
		return nil, fmt.Errorf("not a %s archive (format=%q)", DumpFormat, m.Format)
	}
	return &m, nil
}
