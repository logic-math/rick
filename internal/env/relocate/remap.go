package relocate

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// WorkspaceMapping is one workspace's origin→target relocation. LegacyID is
// the pre-marker id (sha1(path)[:8]) the origin machine's state files
// reference; MarkerID is the stable marker id (may be empty when the
// workspace never got a marker — then remaps fall back to the legacy hash
// of the NEW path).
type WorkspaceMapping struct {
	Name       string
	OriginPath string
	NewPath    string
	MarkerID   string
	LegacyID   string
}

// PathMap is the origin→target mapping applied when importing a dump: the
// longest matching prefix (workspace or state) wins, path-segment aware.
type PathMap struct {
	Workspaces  []WorkspaceMapping
	StateOrigin string
	StateNew    string
}

// MapPath rewrites absolute path p through the longest matching workspace
// or state prefix. Matching is path-segment aware ("/a/b" matches "/a/b/c"
// but not "/a/bc"). Returns the rewritten path and whether a mapping hit.
func (pm *PathMap) MapPath(p string) (string, bool) {
	if pm == nil || p == "" {
		return p, false
	}
	best := ""
	repl := ""
	consider := func(origin, target string) {
		if origin == "" || target == "" {
			return
		}
		if p == origin || strings.HasPrefix(p, origin+"/") {
			if len(origin) > len(best) {
				best, repl = origin, target
			}
		}
	}
	for _, w := range pm.Workspaces {
		consider(w.OriginPath, w.NewPath)
	}
	consider(pm.StateOrigin, pm.StateNew)
	if best == "" {
		return p, false
	}
	return repl + strings.TrimPrefix(p, best), true
}

// LegacyWorkspaceID is the pre-marker workspace id scheme: first 8 hex
// chars of sha1(path). It must stay stable forever — existing web.json /
// sessions.json / archived.json / job-names.json entries reference it.
// (Mirrors internal/web's registry fallback; exported here so dump can
// record legacy ids in the manifest without importing web.)
func LegacyWorkspaceID(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])[:8]
}

// decodeJSONPreserving unmarshals JSON into map[string]any keeping number
// literals intact (json.Number), so rewriting never perturbs unrelated
// numeric fields (job_num, timestamps, ...) on re-marshal.
func decodeJSONPreserving(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root map[string]any
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}
	return root, nil
}

// encodeJSONIndent re-marshals with the 2-space indent rick's own state
// writers use, keeping diffs human-readable.
func encodeJSONIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// RewriteWebJSON rewrites a web.json payload under pm: workspace paths are
// remapped and their ids switch to the mapping's MarkerID (or, without a
// marker, to LegacyWorkspaceID of the NEW path — matching what
// registry.Add would derive there). Unmatched entries pass through
// unchanged; unknown fields and number literals are preserved.
func RewriteWebJSON(data []byte, pm *PathMap) ([]byte, error) {
	root, err := decodeJSONPreserving(data)
	if err != nil {
		return nil, fmt.Errorf("parse web.json: %w", err)
	}
	wsAny, ok := root["workspaces"]
	if !ok {
		return encodeJSONIndent(root)
	}
	wsList, ok := wsAny.([]any)
	if !ok {
		return nil, fmt.Errorf("web.json workspaces: expected array, got %T", wsAny)
	}
	for i, e := range wsList {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		path, _ := entry["path"].(string)
		if path == "" {
			continue
		}
		newPath, hit := pm.MapPath(path)
		if !hit {
			continue
		}
		// Exact workspace match → stable marker id. A prefix hit that is NOT
		// an exact workspace (e.g. a registered sub-workspace inside a dumped
		// tree) keeps its ORIGINAL id: the id is the opaque join key between
		// web.json and sessions.json/archived/job-names, and neither side is
		// rewritten for sub-workspaces — recomputing it here would orphan
		// every session anchored to the sub-workspace.
		id := ""
		exact := false
		for _, w := range pm.Workspaces {
			if w.OriginPath == path {
				exact = true
				if w.MarkerID != "" {
					id = w.MarkerID
				}
				break
			}
		}
		if id == "" && exact && newPath != "" {
			// Exact workspace without a marker (should not happen — dump
			// ensures markers) — fall back to the legacy hash of the new path.
			id = LegacyWorkspaceID(newPath)
		}
		if id == "" {
			id, _ = entry["id"].(string) // sub-workspace: keep original id
		}
		entry["path"] = newPath
		entry["id"] = id
		wsList[i] = entry
	}
	return encodeJSONIndent(root)
}

// RewriteSessionRefs rewrites a sessions.json payload: workspace_id values
// matching a mapping's LegacyID switch to MarkerID (already-marker ids stay
// put — idempotent), and params._prompt_file / params._method_file absolute
// paths are remapped through pm. Sessions whose workspace_id matches no
// mapping pass through unchanged — ghost ids from earlier moves are left
// alone rather than guessed at.
func RewriteSessionRefs(data []byte, pm *PathMap) ([]byte, error) {
	root, err := decodeJSONPreserving(data)
	if err != nil {
		return nil, fmt.Errorf("parse sessions.json: %w", err)
	}
	idRemap := pm.IDRemap()
	sessionsAny, ok := root["sessions"]
	if !ok {
		return encodeJSONIndent(root)
	}
	sessions, ok := sessionsAny.([]any)
	if !ok {
		return nil, fmt.Errorf("sessions.json sessions: expected array, got %T", sessionsAny)
	}
	for i, s := range sessions {
		entry, ok := s.(map[string]any)
		if !ok {
			continue
		}
		if wsID, _ := entry["workspace_id"].(string); wsID != "" {
			if to, hit := idRemap[wsID]; hit {
				entry["workspace_id"] = to
			}
		}
		if paramsAny, ok := entry["params"].(map[string]any); ok {
			for _, key := range []string{"_prompt_file", "_method_file"} {
				if p, _ := paramsAny[key].(string); p != "" {
					if mapped, hit := pm.MapPath(p); hit {
						paramsAny[key] = mapped
					}
				}
			}
		}
		sessions[i] = entry
	}
	return encodeJSONIndent(root)
}

// IDRemap builds the workspace-id translation table: LegacyID→MarkerID and
// MarkerID→MarkerID (idempotent). Mappings without a marker id are skipped.
func (pm *PathMap) IDRemap() map[string]string {
	out := map[string]string{}
	if pm == nil {
		return out
	}
	for _, w := range pm.Workspaces {
		if w.MarkerID == "" {
			continue
		}
		if w.LegacyID != "" {
			out[w.LegacyID] = w.MarkerID
		}
		out[w.MarkerID] = w.MarkerID
	}
	return out
}

// RewriteArchivedIDs rewrites an archived.json payload whose map is keyed
// by workspace id: keys matching a mapping's LegacyID are renamed to
// MarkerID. Ghost keys pass through.
func RewriteArchivedIDs(data []byte, pm *PathMap) ([]byte, error) {
	root, err := decodeJSONPreserving(data)
	if err != nil {
		return nil, fmt.Errorf("parse archived.json: %w", err)
	}
	archAny, ok := root["archived"]
	if !ok {
		return encodeJSONIndent(root)
	}
	arch, ok := archAny.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("archived.json archived: expected object, got %T", archAny)
	}
	idRemap := pm.IDRemap()
	for from, to := range idRemap {
		if from == to {
			continue
		}
		if v, exists := arch[from]; exists {
			delete(arch, from)
			arch[to] = v
		}
	}
	return encodeJSONIndent(root)
}

// RewriteJobNameKeys rewrites a job-names.json payload whose keys are
// "<workspaceID>/<jobID>": the workspace-id prefix is remapped
// LegacyID→MarkerID. Job ids never contain "/", so splitting on the first
// separator is unambiguous.
func RewriteJobNameKeys(data []byte, pm *PathMap) ([]byte, error) {
	root, err := decodeJSONPreserving(data)
	if err != nil {
		return nil, fmt.Errorf("parse job-names.json: %w", err)
	}
	namesAny, ok := root["names"]
	if !ok {
		return encodeJSONIndent(root)
	}
	names, ok := namesAny.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("job-names.json names: expected object, got %T", namesAny)
	}
	idRemap := pm.IDRemap()
	for key, v := range names {
		wsID, _, found := strings.Cut(key, "/")
		if !found {
			continue
		}
		if to, hit := idRemap[wsID]; hit && to != wsID {
			delete(names, key)
			names[to+"/"+key[len(wsID)+1:]] = v
		}
	}
	return encodeJSONIndent(root)
}

// RewriteConfigDefaultWorkspace remaps config.json's default_workspace
// through pm (typically the state-dir mapping). Every other field —
// including credentials like web_token and pi_extra_args — passes through
// untouched.
func RewriteConfigDefaultWorkspace(data []byte, pm *PathMap) ([]byte, error) {
	root, err := decodeJSONPreserving(data)
	if err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}
	if ws, _ := root["default_workspace"].(string); ws != "" {
		if mapped, hit := pm.MapPath(ws); hit {
			root["default_workspace"] = mapped
		}
	}
	return encodeJSONIndent(root)
}

// cdLine matches a shell `cd <path>` line, tolerating surrounding
// whitespace, double quotes around the path, and a trailing comment.
var cdLine = regexp.MustCompile(`^(\s*cd\s+)(?:"([^"]+)"|'([^']+)'|([^\s#;]+))(.*)$`)

// RewriteStartScript remaps `cd <path>` lines of a start-web.sh style
// script through pm. Proxy exports, exec lines, and everything else are
// left untouched — network topology is the target machine's business.
func RewriteStartScript(data []byte, pm *PathMap) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		m := cdLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		path := m[2]
		quote := `"`
		if path == "" {
			if m[3] != "" {
				path, quote = m[3], `'`
			} else {
				path = m[4]
				quote = ""
			}
		}
		mapped, hit := pm.MapPath(path)
		if !hit {
			continue
		}
		lines[i] = m[1] + quote + mapped + quote + m[5]
	}
	return []byte(strings.Join(lines, "\n")), nil
}
