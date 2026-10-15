// Package relocate implements rick's workspace relocation primitives:
// stable workspace identity (marker files), a single-file tar.gz archive
// engine with root-scoped exclude rules, origin→target path remapping for
// rick's state files, and pi session portability (session-directory
// re-escaping plus session header rewriting).
//
// It is the library layer for `rick tools dump` / `rick tools import`
// (job_46): dump/import only orchestrate these primitives.
package relocate

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MarkerFile is the per-workspace identity marker, stored at the workspace
// root. Its "id" decouples the workspace registry id from the workspace's
// filesystem location: moving the workspace to another machine or path keeps
// the id — and therefore every sessions.json / archived.json / job-names.json
// reference — stable. Before markers existed the id was the first 8 hex chars
// of sha1(path) (see LegacyWorkspaceID), which orphaned sessions on every
// move (job_46 research-L2 B2: two ghost ids from earlier moves exist in
// production state).
const MarkerFile = ".rick-workspace.json"

// Marker is the on-disk shape of MarkerFile.
type Marker struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	RickVersion string    `json:"rick_version,omitempty"`
}

// ReadMarkerID returns the marker id of the workspace rooted at dir, or ""
// when the marker is absent, unreadable, or malformed. Callers treat "" as
// "no marker" and fall back to LegacyWorkspaceID.
func ReadMarkerID(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, MarkerFile))
	if err != nil {
		return ""
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return m.ID
}

// EnsureMarker returns the workspace's marker id, creating the marker file
// when absent (crypto/rand 16-byte hex, 32 chars). It is idempotent: an
// existing marker is returned unchanged with created=false, and the
// rickVersion argument is only recorded on creation. A write failure is
// returned as an error; callers may degrade to the legacy path-hash id.
func EnsureMarker(dir, rickVersion string) (id string, created bool, err error) {
	if existing := ReadMarkerID(dir); existing != "" {
		return existing, false, nil
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", false, fmt.Errorf("generate marker id: %w", err)
	}
	m := Marker{
		ID:          hex.EncodeToString(buf),
		CreatedAt:   time.Now().UTC(),
		RickVersion: rickVersion,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", false, fmt.Errorf("marshal marker: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(dir, MarkerFile), data, 0644); err != nil {
		return "", false, fmt.Errorf("write marker: %w", err)
	}
	return m.ID, true, nil
}
