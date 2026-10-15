#!/bin/bash
# relocate_e2e.sh - End-to-end sandbox test for `rick tools dump` / `rick tools import`
# (job_46 workspace relocation feature).
#
# Simulates a cross-machine migration entirely inside /tmp: a fake workspace
# with real content + exclusion garbage, a fake rick state dir (web.json /
# sessions.json / archived.json / job-names.json / config.json /
# start-web.sh), and a fake pi agent dir with a real-shaped session jsonl.
# Then: dump → assert archive structure / manifest / source backfill →
# import to a different path+state → assert remapping (paths, marker ids,
# pi session dir re-escape + header cwd) → re-import (merge idempotency).
#
# Nothing outside the sandbox is touched (RICK_STATE_DIR / RICK_PI_AGENT_DIR
# are pinned to sandbox dirs for both commands).
#
# Usage: bash tests/relocate_e2e.sh
# Exit code: 0 if all tests pass, 1 if any test fails.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

PASSED=0
FAILED=0
TMPDIR_BASE=$(mktemp -d -t rick_relocate_e2e_XXXXXX)

cleanup() {
    rm -rf "$TMPDIR_BASE"
}
trap cleanup EXIT

pass() {
    echo -e "  ${GREEN}✓ PASS${NC}: $1"
    ((PASSED++))
}

fail() {
    echo -e "  ${RED}✗ FAIL${NC}: $1"
    echo "         $2"
    ((FAILED++))
}

# assert_file_exists <desc> <path>
assert_file_exists() {
    if [ -e "$2" ]; then pass "$1"; else fail "$1" "missing: $2"; fi
}

# assert_file_missing <desc> <path>
assert_file_missing() {
    if [ ! -e "$2" ]; then pass "$1"; else fail "$1" "should not exist: $2"; fi
}

# assert_contains <desc> <haystack-string> <needle>
assert_contains() {
    if echo "$2" | grep -qF "$3"; then pass "$1"; else fail "$1" "'$2' does not contain '$3'"; fi
}

# assert_not_contains <desc> <haystack-string> <needle>
assert_not_contains() {
    if echo "$2" | grep -qF "$3"; then fail "$1" "'$2' contains '$3'"; else pass "$1"; fi
}

# assert_json <desc> <json-file> <python-expr on d>
assert_json() {
    local desc="$1" file="$2" expr="$3"
    if python3 -c "import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if ($expr) else 1)" "$file" 2>/dev/null; then
        pass "$desc"
    else
        fail "$desc" "$file: $expr is false"
    fi
}

# escape_dir <abs-path> — mirror relocate.EscapeDir (pi session dir naming):
# strip leading /, replace / \ : with -, wrap in --
escape_dir() {
    local p="${1#/}"
    p="$(printf '%s' "$p" | tr '/\:' '---')"
    echo "--${p}--"
}

# sha1_8 <string> — legacy workspace id: first 8 hex of sha1
sha1_8() {
    printf '%s' "$1" | sha1sum | cut -c1-8
}

# ─── build rick once ────────────────────────────────────────────────────────

NATIVE_GOOS="$(go env GOOS)"
NATIVE_GOARCH="$(go env GOARCH)"
NATIVE="$NATIVE_GOOS/$NATIVE_GOARCH"

echo "== build rick (native: $NATIVE) =="
RICK="$TMPDIR_BASE/rick"
if ! go build -o "$RICK" ./cmd/rick; then
    echo "FATAL: cannot build rick" >&2
    exit 1
fi
pass "build rick binary"

# ─── fixtures ───────────────────────────────────────────────────────────────

SBX="$TMPDIR_BASE/sandbox"
SRC_WS="$SBX/home/proj"
SRC_STATE="$SBX/home/.rick"
SRC_AGENT="$SRC_STATE/pi/agent"
DUMP_OUT="$SBX/dump/out.tar.gz"

mkdir -p "$SRC_WS/.rick/jobs/job_1/plan" "$SRC_WS/scripts" "$SRC_WS/web/node_modules/x" "$SRC_WS/bin" "$SRC_WS/.pi" "$SRC_WS/.morty"
echo "# proj" > "$SRC_WS/README.md"
echo "PROMPT" > "$SRC_WS/.rick/jobs/job_1/plan/prompt.md"
printf '#!/bin/sh\necho hi\n' > "$SRC_WS/scripts/run.sh"
chmod 755 "$SRC_WS/scripts/run.sh"
ln -s ../README.md "$SRC_WS/scripts/link"
echo "// huge" > "$SRC_WS/web/node_modules/x/index.js"
echo "elf-junk" > "$SRC_WS/bin/rick"
echo '{}' > "$SRC_WS/.pi/cache.json"
echo x > "$SRC_WS/.morty/log.txt"

LEGACY_ID="$(sha1_8 "$SRC_WS")"
WS_ESCAPE="$(escape_dir "$SRC_WS")"

mkdir -p "$SRC_STATE/web" "$SRC_AGENT/sessions/$WS_ESCAPE" "$SRC_AGENT/sessions/$(escape_dir /nonexistent/other)" "$SRC_AGENT/extensions" "$SRC_AGENT/bin" "$SRC_AGENT/runtime/node_modules/.bin"
cat > "$SRC_STATE/web.json" <<EOF
{
  "version": 1,
  "workspaces": [
    {"id": "$LEGACY_ID", "path": "$SRC_WS", "name": "proj", "added_at": "2026-01-01T00:00:00Z"}
  ]
}
EOF
cat > "$SRC_STATE/web/sessions.json" <<EOF
{
  "version": 1,
  "sessions": [
    {"id": "sess-1", "workspace_id": "$LEGACY_ID", "type": "easy", "status": "closed",
     "pi_session_id": "pi-0001",
     "params": {"_prompt_file": "$SRC_WS/.rick/jobs/job_1/plan/prompt.md"},
     "created_at": "2026-01-01T00:00:00Z"}
  ]
}
EOF
cat > "$SRC_STATE/web/archived.json" <<EOF
{"version": 1, "archived": {"$LEGACY_ID": ["job_9"]}}
EOF
cat > "$SRC_STATE/web/job-names.json" <<EOF
{"version": 1, "names": {"$LEGACY_ID/job_9": "旧名"}}
EOF
cat > "$SRC_STATE/config.json" <<EOF
{"default_workspace": "$SRC_STATE", "web_token": "e2e-token-123", "runtime": "pi"}
EOF
cat > "$SRC_STATE/start-web.sh" <<EOF
#!/bin/bash
cd $SRC_WS
exec ./bin/rick web --port 8413
EOF
echo "12345" > "$SRC_STATE/web.pid"
echo "log" > "$SRC_STATE/web.log"
echo '{"theme":"rick"}' > "$SRC_AGENT/settings.json"
echo '{"deepseek":{"key":"sk-e2e"}}' > "$SRC_AGENT/auth.json"
cat > "$SRC_AGENT/sessions/$WS_ESCAPE/2026-01-01T00-00-00-000Z_pi-0001.jsonl" <<EOF
{"type":"session","version":3,"id":"pi-0001","timestamp":"2026-01-01T00:00:00Z","cwd":"$SRC_WS"}
{"type":"message","role":"user","content":"hello"}
EOF
echo '{"type":"session","version":3,"id":"pi-0002","cwd":"/nonexistent/other"}' > "$SRC_AGENT/sessions/$(escape_dir /nonexistent/other)/2026-01-02T00-00-00-000Z_pi-0002.jsonl"
echo "rg-elf" > "$SRC_AGENT/bin/rg"
ln -s /workdir/nowhere/catpaw.ts "$SRC_AGENT/extensions/catpaw-proxy.ts"
echo "#!/bin/sh" > "$SRC_AGENT/runtime/node_modules/.bin/pi"
pass "create sandbox fixtures"

# ─── dump ───────────────────────────────────────────────────────────────────

echo "== rick tools dump =="
mkdir -p "$SBX/dump"
if "$RICK" tools dump \
    --workspace "$SRC_WS" \
    -o "$DUMP_OUT" \
    --state-dir "$SRC_STATE" \
    --sessions workspaces \
    --goos "$NATIVE" \
    2>&1 | tee "$TMPDIR_BASE/dump.log"; then
    pass "dump command exits 0"
else
    fail "dump command exits 0" "$(cat "$TMPDIR_BASE/dump.log")"
fi

assert_file_exists "archive written" "$DUMP_OUT"
assert_file_exists "sha256 sidecar written" "$DUMP_OUT.sha256"

TAR_NAMES="$(tar -tzf "$DUMP_OUT")"
assert_contains "archive has manifest.json" "$TAR_NAMES" "manifest.json"
assert_contains "archive has workspace tree" "$TAR_NAMES" "workspaces/proj/README.md"
assert_contains "archive keeps executable script" "$TAR_NAMES" "workspaces/proj/scripts/run.sh"
assert_contains "archive has state web.json" "$TAR_NAMES" "state/web.json"
assert_contains "archive has sessions.json" "$TAR_NAMES" "state/web/sessions.json"
assert_contains "archive has pi settings" "$TAR_NAMES" "state/pi-agent/settings.json"
assert_contains "archive has session history" "$TAR_NAMES" "state/pi-agent/sessions/$WS_ESCAPE/2026-01-01T00-00-00-000Z_pi-0001.jsonl"
assert_contains "archive has embedded binary" "$TAR_NAMES" "bin/$NATIVE_GOOS-$NATIVE_GOARCH/rick"

assert_not_contains "node_modules excluded" "$TAR_NAMES" "web/node_modules"
assert_not_contains "workspace bin excluded" "$TAR_NAMES" "workspaces/proj/bin/rick"
assert_not_contains ".pi excluded" "$TAR_NAMES" "workspaces/proj/.pi/cache.json"
assert_not_contains "web.pid excluded" "$TAR_NAMES" "state/web.pid"
assert_not_contains "web.log excluded" "$TAR_NAMES" "state/web.log"
assert_not_contains "rg ELF excluded" "$TAR_NAMES" "state/pi-agent/bin/rg"
assert_not_contains "catpaw excluded" "$TAR_NAMES" "catpaw-proxy.ts"
assert_not_contains "other-cwd sessions excluded (policy=workspaces)" "$TAR_NAMES" "pi-0002.jsonl"

# marker created in the source workspace + manifest identity
MARKER_ID="$(python3 -c "import json;print(json.load(open('$SRC_WS/.rick-workspace.json'))['id'])" 2>/dev/null)"
if [ -n "$MARKER_ID" ]; then
    pass "marker created in source workspace ($MARKER_ID)"
else
    fail "marker created in source workspace" "cannot read $SRC_WS/.rick-workspace.json"
fi

# sidecar sha matches
ARCHIVE_SHA="$(sha256sum "$DUMP_OUT" | cut -d' ' -f1)"
SIDECAR_SHA="$(cut -d' ' -f1 "$DUMP_OUT.sha256")"
[ "$ARCHIVE_SHA" = "$SIDECAR_SHA" ] && pass "sidecar sha256 matches archive" || fail "sidecar sha256 matches archive" "$SIDECAR_SHA != $ARCHIVE_SHA"

# manifest content
python3 - "$DUMP_OUT" <<'PYEOF'
import json, subprocess, sys
archive = sys.argv[1]
names = subprocess.run(["tar", "-tzf", archive], capture_output=True, text=True).stdout
print("MANIFEST_OK")  # marker line consumed below
PYEOF
tar -xzf "$DUMP_OUT" -O manifest.json > "$TMPDIR_BASE/manifest.json" 2>/dev/null
assert_json "manifest format is rick-dump v1" "$TMPDIR_BASE/manifest.json" "d['format']=='rick-dump' and d['version']==1"
assert_json "manifest workspace origin path" "$TMPDIR_BASE/manifest.json" "d['workspaces'][0]['origin_path']=='$SRC_WS'"
assert_json "manifest marker id == on-disk marker" "$TMPDIR_BASE/manifest.json" "d['workspaces'][0]['marker_id']=='$MARKER_ID'"
assert_json "manifest legacy id recorded" "$TMPDIR_BASE/manifest.json" "d['workspaces'][0]['legacy_id']=='$LEGACY_ID'"
assert_json "manifest records exclusions" "$TMPDIR_BASE/manifest.json" "len(d['excluded'])>0"
assert_json "manifest records native binary" "$TMPDIR_BASE/manifest.json" "any(b['goos']=='$NATIVE_GOOS' and b['goarch']=='$NATIVE_GOARCH' for b in d['binaries'])"
assert_json "manifest sessions policy" "$TMPDIR_BASE/manifest.json" "d['sessions_policy']=='workspaces'"

# source-side backfill: registries upgraded legacy → marker
assert_json "source web.json id backfilled to marker" "$SRC_STATE/web.json" "d['workspaces'][0]['id']=='$MARKER_ID'"
assert_json "source sessions.json workspace_id backfilled" "$SRC_STATE/web/sessions.json" "d['sessions'][0]['workspace_id']=='$MARKER_ID'"
assert_json "source archived.json key backfilled" "$SRC_STATE/web/archived.json" "'$MARKER_ID' in d['archived']"
assert_json "source job-names key backfilled" "$SRC_STATE/web/job-names.json" "'$MARKER_ID/job_9' in d['names']"

# ─── import (the "other machine") ───────────────────────────────────────────

echo "== rick tools import (cross-path) =="
NEW_HOME="$SBX/new-machine"
NEW_WS="$NEW_HOME/proj"
NEW_STATE="$NEW_HOME/.rick"
mkdir -p "$NEW_HOME"

if "$RICK" tools import "$DUMP_OUT" \
    --target "$NEW_WS" \
    --state-dir "$NEW_STATE" \
    2>&1 | tee "$TMPDIR_BASE/import.log"; then
    pass "import command exits 0"
else
    fail "import command exits 0" "$(cat "$TMPDIR_BASE/import.log")"
fi

# workspace landed at target with content/mode/symlink intact
assert_file_exists "workspace placed at target" "$NEW_WS/README.md"
[ "$(cat "$NEW_WS/README.md")" = "# proj" ] && pass "file content intact" || fail "file content intact" "$(cat "$NEW_WS/README.md")"
[ -x "$NEW_WS/scripts/run.sh" ] && pass "executable bit preserved" || fail "executable bit preserved" "$NEW_WS/scripts/run.sh"
[ -L "$NEW_WS/scripts/link" ] && pass "symlink preserved" || fail "symlink preserved" "$NEW_WS/scripts/link"
assert_file_missing "node_modules not imported" "$NEW_WS/web/node_modules/x/index.js"
assert_file_missing "workspace bin not imported" "$NEW_WS/bin/rick"
assert_file_missing "web.pid not imported" "$NEW_STATE/web.pid"

# marker id stable across machines
NEW_MARKER="$(python3 -c "import json;print(json.load(open('$NEW_WS/.rick-workspace.json'))['id'])" 2>/dev/null)"
[ "$NEW_MARKER" = "$MARKER_ID" ] && pass "marker id stable across machines" || fail "marker id stable across machines" "$NEW_MARKER != $MARKER_ID"

# state remapping
assert_json "web.json path remapped" "$NEW_STATE/web.json" "d['workspaces'][0]['path']=='$NEW_WS'"
assert_json "web.json id is marker (stable)" "$NEW_STATE/web.json" "d['workspaces'][0]['id']=='$MARKER_ID'"
assert_json "sessions.json workspace_id is marker" "$NEW_STATE/web/sessions.json" "d['sessions'][0]['workspace_id']=='$MARKER_ID'"
assert_json "sessions.json _prompt_file remapped" "$NEW_STATE/web/sessions.json" "d['sessions'][0]['params']['_prompt_file']=='$NEW_WS/.rick/jobs/job_1/plan/prompt.md'"
assert_json "archived.json key remapped" "$NEW_STATE/web/archived.json" "'$MARKER_ID' in d['archived']"
assert_json "job-names key remapped" "$NEW_STATE/web/job-names.json" "'$MARKER_ID/job_9' in d['names']"
assert_json "config default_workspace remapped" "$NEW_STATE/config.json" "d['default_workspace']=='$NEW_STATE'"
assert_json "config web_token survived" "$NEW_STATE/config.json" "d['web_token']=='e2e-token-123'"

NEW_SW="$(cat "$NEW_STATE/start-web.sh")"
assert_contains "start-web.sh cd remapped" "$NEW_SW" "cd $NEW_WS"
assert_contains "start-web.sh keeps exec line" "$NEW_SW" "exec ./bin/rick web --port 8413"

# pi sessions: dir re-escaped to the new cwd + header rewritten
NEW_ESCAPE="$(escape_dir "$NEW_WS")"
NEW_SESS_FILE="$NEW_STATE/pi/agent/sessions/$NEW_ESCAPE/2026-01-01T00-00-00-000Z_pi-0001.jsonl"
assert_file_exists "session dir re-escaped to new cwd" "$NEW_SESS_FILE"
assert_file_missing "origin-escaped session dir absent" "$NEW_STATE/pi/agent/sessions/$WS_ESCAPE"
FIRST_LINE="$(head -1 "$NEW_SESS_FILE")"
assert_contains "session header cwd rewritten" "$FIRST_LINE" "\"cwd\":\"$NEW_WS\""
SECOND_LINE="$(sed -n 2p "$NEW_SESS_FILE")"
assert_contains "session payload untouched" "$SECOND_LINE" "hello"

# pi agent config traveled
assert_file_exists "pi settings placed" "$NEW_STATE/pi/agent/settings.json"
assert_file_exists "pi auth placed" "$NEW_STATE/pi/agent/auth.json"
assert_file_exists "pi runtime placed (non-slim)" "$NEW_STATE/pi/agent/runtime/node_modules/.bin/pi"

# embedded native binary staged + install hint printed
assert_file_exists "native rick staged for install" "$NEW_STATE/bin/rick-import-$NATIVE_GOOS-$NATIVE_GOARCH"
assert_contains "import output has install hint" "$(cat "$TMPDIR_BASE/import.log")" "rick-import-$NATIVE_GOOS-$NATIVE_GOARCH"

# ─── re-import: merge idempotency ───────────────────────────────────────────

echo "== re-import same archive (merge semantics) =="
echo "# LOCAL EDIT" > "$NEW_WS/README.md"
SESS_SHA_BEFORE="$(sha256sum "$NEW_STATE/web/sessions.json" | cut -d' ' -f1)"

if "$RICK" tools import "$DUMP_OUT" \
    --target "$NEW_WS" \
    --state-dir "$NEW_STATE" \
    2>&1 | tee "$TMPDIR_BASE/reimport.log"; then
    pass "re-import exits 0"
else
    fail "re-import exits 0" "$(cat "$TMPDIR_BASE/reimport.log")"
fi

[ "$(cat "$NEW_WS/README.md")" = "# LOCAL EDIT" ] && pass "existing file NOT overwritten" || fail "existing file NOT overwritten" "$(cat "$NEW_WS/README.md")"
SESS_SHA_AFTER="$(sha256sum "$NEW_STATE/web/sessions.json" | cut -d' ' -f1)"
[ "$SESS_SHA_BEFORE" = "$SESS_SHA_AFTER" ] && pass "registry not rewritten when nothing new" || fail "registry not rewritten when nothing new" "$SESS_SHA_BEFORE != $SESS_SHA_AFTER"
assert_json "web.json still single entry" "$NEW_STATE/web.json" "len(d['workspaces'])==1"
assert_contains "re-import reports skipped files" "$(cat "$TMPDIR_BASE/reimport.log")" "跳过"

# ─── summary ────────────────────────────────────────────────────────────────

echo ""
echo "=========================================="
echo "relocate e2e: $PASSED passed, $FAILED failed"
echo "=========================================="
[ "$FAILED" -eq 0 ] || exit 1
exit 0
