# Git Warnings Bug Fix

## Issue Description

During `rick doing job_0` execution, two git-related warnings appeared:

1. **First Warning** (after task1 execution):
   ```
   WARN: Failed to get commit hash: failed to run git rev-parse: exit status 128
   ```

2. **Second Warning** (after job completion):
   ```
   [WARN] Failed to commit results: failed to commit changes: failed to add files: failed to add files: exit status 128
   ```

## Root Cause Analysis

### Warning 1: Failed to get commit hash
- **Cause**: When task1 executed, the git repository was just initialized (no commits yet)
- **Command**: `git rev-parse HEAD` fails with exit code 128 when there are no commits
- **Location**: `internal/executor/executor.go:314-332`

### Warning 2: Failed to commit results
- **Cause**: After job completion, all files were already committed by individual tasks
- **Command**: `git add` with empty file list fails with exit code 128
- **Location**: `internal/git/git.go:47-61` and `internal/git/commit.go:176-201`

## Solution

### Fix 1: Better error handling for empty repository
**File**: `internal/executor/executor.go`

Changed `getCurrentCommitHash()` to:
- Use `CombinedOutput()` instead of `Output()` to capture stderr
- Detect "no commits yet" scenario with specific error message
- Provide clearer error messages with output details

```go
output, err := cmd.CombinedOutput()
if err != nil {
    // Check if this is because there are no commits yet
    if strings.Contains(string(output), "unknown revision") ||
       strings.Contains(string(output), "bad revision") ||
       strings.Contains(string(output), "ambiguous argument") {
        return "", fmt.Errorf("no commits yet in repository")
    }
    return "", fmt.Errorf("failed to run git rev-parse: %w (output: %s)", err, string(output))
}
```

### Fix 2: Handle empty file list gracefully
**File**: `internal/git/git.go`

Changed `AddFiles()` to:
- Return `nil` (not error) when paths list is empty
- This is not an error condition - just nothing to add
- Improved error messages with output details

```go
func (gm *GitManager) AddFiles(paths []string) error {
    if len(paths) == 0 {
        // Not an error - just no files to add
        return nil
    }
    // ... rest of implementation
}
```

**File**: `internal/git/commit.go`

Changed `AutoAddAndCommitJob()` to:
- Return early if no files to commit
- Avoid calling `AddFiles()` and `CommitJob()` when nothing changed

```go
// If no files to add, nothing to commit
if len(allFiles) == 0 {
    return nil
}
```

### Fix 3: Update test expectations
**File**: `internal/git/git_test.go`

Updated `TestAddFilesEmpty()` to reflect new behavior:
- Empty file list should NOT fail
- Changed assertion from "should fail" to "should not fail"

## Verification

### Unit Tests
All 59 git package tests pass:
```bash
go test -v ./internal/git/...
# PASS: 59/59 tests
```

### Manual Testing
1. Create test directory with task file
2. Run `rick plan` and `rick doing`
3. Verify no warnings appear

## Impact

- **Behavior Change**: `AddFiles([])` now returns `nil` instead of error
- **Breaking Change**: No (internal API only)
- **User Impact**: Cleaner output, no confusing warnings
- **Test Updates**: 1 test updated to reflect new behavior

## Commit

```
commit f547d06
bugfix: fix git warnings in doing command

Fixed two git-related warnings that appeared during job execution:
1. "Failed to get commit hash" warning when no commits exist yet
2. "Failed to commit results" warning when no changes to commit

All git package tests pass (59/59).
```

## Related Files

- `internal/executor/executor.go` - getCurrentCommitHash()
- `internal/git/git.go` - AddFiles()
- `internal/git/commit.go` - AutoAddAndCommitJob()
- `internal/git/git_test.go` - TestAddFilesEmpty()
