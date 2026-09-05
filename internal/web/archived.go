package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sunquan/rick/internal/workspace"
)

// ArchiveFileVersion is the current schema version of archived.json.
const ArchiveFileVersion = 1

// archivedFile is the on-disk shape of the archive registry.
type archivedFile struct {
	Version  int                 `json:"version"`
	Archived map[string][]string `json:"archived"`
}

// ArchivedStore is the machine-level "soft archive" registry: per workspace
// the list of job ids hidden from the default jobs listing. It is a pure
// web-layer view state — rick's job files are never touched, so dream /
// learning keep scanning every job regardless of archive status.
//
// Soft-archive rationale (job_36): completed jobs accumulate and bury the
// active ones; archiving hides them from the default listing while keeping
// full data recoverable (unarchive). Files are never deleted.
type ArchivedStore struct {
	mu   sync.Mutex
	path string
	data archivedFile
}

// LoadArchived loads the archive registry from path. A missing file yields
// an empty store (no error); corrupt JSON surfaces as an error — a
// malformed state file should not silently reset the registry.
func LoadArchived(path string) (*ArchivedStore, error) {
	s := &ArchivedStore{
		path: path,
		data: archivedFile{Version: ArchiveFileVersion, Archived: map[string][]string{}},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read archived registry %s: %w", path, err)
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s.data); err != nil {
		return nil, fmt.Errorf("parse archived registry %s: %w", path, err)
	}
	if s.data.Archived == nil {
		s.data.Archived = map[string][]string{}
	}
	return s, nil
}

// Archive marks jobID archived for wsID (idempotent) and persists.
func (s *ArchivedStore) Archive(wsID, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !containsString(s.data.Archived[wsID], jobID) {
		s.data.Archived[wsID] = append(s.data.Archived[wsID], jobID)
	}
	return s.persist()
}

// Unarchive removes jobID from wsID's archived list (idempotent) and
// persists. Unknown wsID/jobID is a no-op.
func (s *ArchivedStore) Unarchive(wsID, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.data.Archived[wsID]
	for i, id := range list {
		if id == jobID {
			list = append(list[:i], list[i+1:]...)
			if len(list) == 0 {
				delete(s.data.Archived, wsID)
			} else {
				s.data.Archived[wsID] = list
			}
			return s.persist()
		}
	}
	return nil
}

// IsArchived reports whether jobID is archived for wsID.
func (s *ArchivedStore) IsArchived(wsID, jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return containsString(s.data.Archived[wsID], jobID)
}

// List returns the archived job ids for wsID (copy, insertion order).
func (s *ArchivedStore) List(wsID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.data.Archived[wsID]...)
}

func (s *ArchivedStore) persist() error {
	if s.path == "" {
		return nil // in-memory only (tests)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create archived registry dir: %w", err)
	}
	data, err := json.MarshalIndent(&s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal archived registry: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write archived registry tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("persist archived registry: %w", err)
	}
	return nil
}

func containsString(list []string, id string) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}

// DreamArchivedJobs returns the set of job IDs that dream has already
// processed (a dream_run_{job_id}_log.md file exists under
// <rickDir>/dream/). Per the job_36 archive semantics upgrade, such jobs
// are considered automatically archived: they are hidden from the default
// jobs listing and surfaced in the Jobs 归档区 with an "已由 dream 学习"
// badge. Manual archiving remains a supplementary mechanism.
//
// Reuses workspace.GetDreamProcessedJobs (exported) — same scan semantics:
// prefix dream_run_ + suffix _log.md, job_id shape job_N.
func DreamArchivedJobs(rickDir string) map[string]bool {
	return workspace.GetDreamProcessedJobs(rickDir)
}
