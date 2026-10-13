package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/env/relocate"
)

// legacyRegistryJSON writes a web.json exactly like pre-marker rick did:
// id = sha1(path)[:8], no marker file on disk.
func legacyRegistryJSON(t *testing.T, path string, wsPath, name string) {
	t.Helper()
	entry := WorkspaceEntry{ID: relocate.LegacyWorkspaceID(wsPath), Path: wsPath, Name: name, AddedAt: time.Now()}
	reg := WorkspaceRegistry{Version: workspaceRegistryVersion, Items: []WorkspaceEntry{entry}}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// TestWorkspaceIDMarkerPrecedence locks the id derivation order: marker
// wins; without a marker the legacy path-hash applies.
func TestWorkspaceIDMarkerPrecedence(t *testing.T) {
	ws := newTestWorkspace(t)
	legacy := relocate.LegacyWorkspaceID(ws)
	if got := workspaceID(ws); got != legacy {
		t.Fatalf("no marker: workspaceID=%q want legacy %q", got, legacy)
	}
	id, created, err := relocate.EnsureMarker(ws, "")
	if err != nil || !created {
		t.Fatalf("EnsureMarker: %v created=%v", err, created)
	}
	if got := workspaceID(ws); got != id {
		t.Fatalf("with marker: workspaceID=%q want marker %q", got, id)
	}
	if got := relocate.ReadMarkerID(ws); got != id || len(id) != 32 {
		t.Fatalf("marker readback: %q", got)
	}
}

// TestWorkspaceIDMarkerMalformedFallsBack locks graceful degradation: a
// corrupt marker must yield the legacy id, never an error path.
func TestWorkspaceIDMarkerMalformedFallsBack(t *testing.T) {
	ws := newTestWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, relocate.MarkerFile), []byte("{broken"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := workspaceID(ws); got != relocate.LegacyWorkspaceID(ws) {
		t.Fatalf("malformed marker must fall back to legacy id, got %q", got)
	}
}

// TestRegistryAddCreatesMarker: adding a NEW workspace writes its marker
// and registers the stable id.
func TestRegistryAddCreatesMarker(t *testing.T) {
	path := registryPathUnder(t)
	ws := newTestWorkspace(t)
	r, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, created, err := r.Add(ws, "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !created {
		t.Fatal("expected created=true")
	}
	if entry.ID != relocate.ReadMarkerID(ws) {
		t.Fatalf("entry.ID=%q must equal the fresh marker id %q", entry.ID, relocate.ReadMarkerID(ws))
	}
	if entry.ID == relocate.LegacyWorkspaceID(ws) {
		t.Fatal("a marker-backed add must NOT use the legacy path hash")
	}
	if len(entry.ID) != 32 {
		t.Fatalf("marker id must be 32 hex chars, got %q", entry.ID)
	}
	// Re-add is idempotent and returns the SAME id.
	again, created2, err := r.Add(ws, "")
	if err != nil {
		t.Fatal(err)
	}
	if created2 || again.ID != entry.ID {
		t.Fatalf("idempotent re-add: created=%v id=%q want %q", created2, again.ID, entry.ID)
	}
}

// TestRegistryIdempotentAddKeepsLegacyID: re-adding an ALREADY-registered
// legacy entry must return it untouched and must NOT create a marker
// behind its back — upgrading an id without rewriting sessions.json refs
// would orphan them (the ghost-id failure mode, research-L2 B2). The
// dump/import flow owns the upgrade (backfill), not a routine re-add.
func TestRegistryIdempotentAddKeepsLegacyID(t *testing.T) {
	path := registryPathUnder(t)
	ws := newTestWorkspace(t)
	legacyRegistryJSON(t, path, ws, "old")
	r, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, created, err := r.Add(ws, "ignored")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if created {
		t.Fatal("re-add must be idempotent (created=false)")
	}
	if entry.ID != relocate.LegacyWorkspaceID(ws) {
		t.Fatalf("legacy id must survive re-add: got %q", entry.ID)
	}
	if relocate.ReadMarkerID(ws) != "" {
		t.Fatal("idempotent re-add must not silently create a marker")
	}
	// And the file on disk is untouched.
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), entry.ID) {
		t.Fatal("registry file must still reference the legacy id")
	}
}

// TestRegistryLoadTrustsStoredIDs — THE backward-compat lock: loading an
// old web.json must trust the stored id even when it disagrees with
// sha1(path) and no marker exists. If load ever recomputes ids, every
// existing session reference orphans (research-L2 B2).
func TestRegistryLoadTrustsStoredIDs(t *testing.T) {
	path := registryPathUnder(t)
	ws := newTestWorkspace(t)
	fake := "deadbeef"
	if fake == relocate.LegacyWorkspaceID(ws) {
		t.Skip("unlucky collision, pick another fake id")
	}
	reg := WorkspaceRegistry{Version: workspaceRegistryVersion, Items: []WorkspaceEntry{
		{ID: fake, Path: ws, Name: "x", AddedAt: time.Now()},
	}}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	r, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := r.Get(fake)
	if !ok || got.ID != fake {
		t.Fatalf("load must trust stored id: got %+v ok=%v", got, ok)
	}
	if _, ok := r.Get(relocate.LegacyWorkspaceID(ws)); ok {
		t.Fatal("load must NOT invent the sha1 id")
	}
}

// TestRegistryRemoveAddCycleWithMarker: Remove + re-Add keeps the id stable
// thanks to the marker (previously the id was path-derived, so it stayed
// stable too — marker must not regress this).
func TestRegistryRemoveAddCycleWithMarker(t *testing.T) {
	path := registryPathUnder(t)
	ws := newTestWorkspace(t)
	r, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := r.Add(ws, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Remove(first.ID); err != nil {
		t.Fatal(err)
	}
	second, _, err := r.Add(ws, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("marker id must survive remove/add cycle: %q vs %q", first.ID, second.ID)
	}
}
