package relocate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadMarkerIDMissingDir(t *testing.T) {
	if id := ReadMarkerID(t.TempDir()); id != "" {
		t.Fatalf("absent marker must yield \"\", got %q", id)
	}
}

func TestReadMarkerIDMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, MarkerFile), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if id := ReadMarkerID(dir); id != "" {
		t.Fatalf("malformed marker must yield \"\", got %q", id)
	}
}

func TestReadMarkerIDEmptyID(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, MarkerFile), []byte(`{"id":"","created_at":"2026-10-13T00:00:00Z"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if id := ReadMarkerID(dir); id != "" {
		t.Fatalf("empty marker id must yield \"\", got %q", id)
	}
}

func TestEnsureMarkerCreatesStableID(t *testing.T) {
	dir := t.TempDir()
	id1, created, err := EnsureMarker(dir, "5.0.15")
	if err != nil {
		t.Fatalf("EnsureMarker: %v", err)
	}
	if !created {
		t.Fatal("first EnsureMarker must report created=true")
	}
	if len(id1) != 32 || strings.ContainsAny(id1, "gh-") {
		t.Fatalf("marker id must be 32 hex chars, got %q", id1)
	}
	// Idempotent: second call returns the same id, created=false.
	id2, created2, err := EnsureMarker(dir, "9.9.9")
	if err != nil {
		t.Fatalf("EnsureMarker second: %v", err)
	}
	if created2 {
		t.Fatal("second EnsureMarker must report created=false")
	}
	if id1 != id2 {
		t.Fatalf("marker id must be stable: %q vs %q", id1, id2)
	}
	// The version passed on the second call must not overwrite the original.
	data, err := os.ReadFile(filepath.Join(dir, MarkerFile))
	if err != nil {
		t.Fatal(err)
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("marker file must be valid JSON: %v", err)
	}
	if m.RickVersion != "5.0.15" {
		t.Fatalf("existing marker must keep its rick_version, got %q", m.RickVersion)
	}
	if m.ID != id1 {
		t.Fatalf("marker file id %q != returned %q", m.ID, id1)
	}
}

func TestEnsureMarkerReadback(t *testing.T) {
	dir := t.TempDir()
	id, _, err := EnsureMarker(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := ReadMarkerID(dir); got != id {
		t.Fatalf("ReadMarkerID after EnsureMarker: got %q want %q", got, id)
	}
}

func TestEnsureMarkerUnwritableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits do not block writes")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "ws")
	if err := os.Mkdir(sub, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0755) })
	if _, _, err := EnsureMarker(sub, ""); err == nil {
		t.Fatal("EnsureMarker on read-only dir must fail (caller degrades to legacy id)")
	}
}
