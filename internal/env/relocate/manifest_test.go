package relocate

import (
	"bytes"

	"strings"
	"testing"
	"time"
)

func testManifest() *Manifest {
	return &Manifest{
		Format:         DumpFormat,
		Version:        DumpVersion,
		CreatedAt:      time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
		Origin:         OriginInfo{Hostname: "srv", OS: "linux", Arch: "amd64", User: "hadoop-recsys", Home: "/home/hadoop-recsys"},
		StateDirOrigin: "/home/hadoop-recsys/.rick",
		Workspaces: []WorkspaceInfo{
			{Name: "rick", OriginPath: "/workdir/sunquan20/AI_CODING/rick", MarkerID: "0123456789abcdef0123456789abcdef", LegacyID: "a8ceb938"},
		},
		SessionsPolicy: SessionsPolicyWorkspaces,
		Excluded: []ExcludedEntry{
			{Path: "web/node_modules", Root: string(RootWorkspace), Kind: KindRegenerable, Reason: "npm ci"},
			{Path: "bin/rg", Root: string(RootPIAgent), Kind: KindPlatformELF, Reason: "Linux ELF"},
		},
		Binaries: []BinaryInfo{
			{GOOS: "darwin", GOARCH: "arm64", SHA256: strings.Repeat("a", 64), Size: 17203618},
			{GOOS: "linux", GOARCH: "amd64", SHA256: strings.Repeat("b", 64), Size: 24413978},
		},
		RickVersion: "5.0.15",
	}
}

func TestManifestRoundTrip(t *testing.T) {
	m := testManifest()
	var buf bytes.Buffer
	if err := WriteManifest(&buf, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	got, err := ReadManifest(&buf)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.Format != DumpFormat || got.Version != DumpVersion {
		t.Errorf("format/version: %+v", got)
	}
	if got.StateDirOrigin != m.StateDirOrigin {
		t.Errorf("state dir origin: %q", got.StateDirOrigin)
	}
	if len(got.Workspaces) != 1 || got.Workspaces[0].LegacyID != "a8ceb938" {
		t.Errorf("workspaces: %+v", got.Workspaces)
	}
	if len(got.Excluded) != 2 || got.Excluded[1].Root != string(RootPIAgent) {
		t.Errorf("excluded: %+v", got.Excluded)
	}
	if len(got.Binaries) != 2 || got.Binaries[0].GOARCH != "arm64" {
		t.Errorf("binaries: %+v", got.Binaries)
	}
	if got.RickVersion != "5.0.15" {
		t.Errorf("rick version: %q", got.RickVersion)
	}
}

func TestManifestJSONShape(t *testing.T) {
	// Field names are part of the archive format contract (R3's dump writes
	// it, import reads it — potentially by a different rick version).
	var buf bytes.Buffer
	if err := WriteManifest(&buf, testManifest()); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{
		`"format": "rick-dump"`,
		`"version": 1`,
		`"state_dir_origin"`,
		`"origin_path"`,
		`"marker_id"`,
		`"legacy_id"`,
		`"sessions_policy"`,
		`"goos"`,
		`"goarch"`,
		`"sha256"`,
		`"rick_version"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("manifest JSON must contain %s:\n%s", want, s)
		}
	}
}

func TestManifestRejectsForeignFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteManifest(&buf, testManifest()); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(&buf)
	if err != nil {
		t.Fatalf("same-format read must succeed: %v", err)
	}
	got.Format = "someone-elses-dump"
	var buf2 bytes.Buffer
	if err := WriteManifest(&buf2, got); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(&buf2); err == nil {
		t.Fatal("ReadManifest must reject a foreign format value")
	}
	if _, err := ReadManifest(bytes.NewReader([]byte("not json"))); err == nil {
		t.Fatal("ReadManifest must reject invalid JSON")
	}
}

func TestSessionsPolicyConstants(t *testing.T) {
	// The three policy values are CLI flag values (dump --sessions=...).
	if SessionsPolicyNone != "none" || SessionsPolicyWorkspaces != "workspaces" || SessionsPolicyAll != "all" {
		t.Fatal("sessions policy constants drifted")
	}
}
