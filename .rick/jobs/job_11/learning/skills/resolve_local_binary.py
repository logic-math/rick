# Description: 解析本地构建的二进制路径，优先用 ./bin/<name>，fallback 到系统安装版

import argparse
import json
import os
import shutil
import sys


def resolve_binary(name, project_root=None):
    """
    Resolve the path to a binary, preferring the locally built version.

    Priority order:
    1. {project_root}/bin/{name}  (local build, contains latest code changes)
    2. shutil.which(name)         (system-installed version, may be stale)

    Returns (path, source) where source is "local" or "system".
    Raises FileNotFoundError if neither is found.
    """
    if project_root is None:
        project_root = _find_project_root()

    if project_root:
        local_path = os.path.join(project_root, "bin", name)
        if os.path.isfile(local_path) and os.access(local_path, os.X_OK):
            return local_path, "local"

    system_path = shutil.which(name)
    if system_path:
        return system_path, "system"

    raise FileNotFoundError(
        f"Binary '{name}' not found. "
        f"Expected at {os.path.join(project_root or '.', 'bin', name)} or in PATH."
    )


def _find_project_root(start=None):
    """Walk up from start (or cwd) to find the directory containing .rick/."""
    current = os.path.abspath(start or os.getcwd())
    while True:
        if os.path.isdir(os.path.join(current, ".rick")):
            return current
        parent = os.path.dirname(current)
        if parent == current:
            return None
        current = parent


def run_tests():
    """Built-in self-tests."""
    errors = []

    # Test 1: resolve_binary falls back to system if no local build
    try:
        path, source = resolve_binary("python3")
        if not path:
            errors.append("Test 1 failed: resolve_binary returned empty path for python3")
        # python3 should always be found via system
        if source not in ("local", "system"):
            errors.append(f"Test 1 failed: unexpected source '{source}'")
    except FileNotFoundError as e:
        errors.append(f"Test 1 failed: {e}")

    # Test 2: nonexistent binary raises FileNotFoundError
    try:
        resolve_binary("__nonexistent_binary_xyz__")
        errors.append("Test 2 failed: should have raised FileNotFoundError")
    except FileNotFoundError:
        pass  # expected

    # Test 3: local binary takes priority when it exists
    import tempfile
    with tempfile.TemporaryDirectory() as tmpdir:
        bin_dir = os.path.join(tmpdir, "bin")
        os.makedirs(bin_dir)
        fake_bin = os.path.join(bin_dir, "fake_tool")
        with open(fake_bin, "w") as f:
            f.write("#!/bin/sh\necho fake\n")
        os.chmod(fake_bin, 0o755)
        # Also create .rick/ so _find_project_root works
        os.makedirs(os.path.join(tmpdir, ".rick"))

        path, source = resolve_binary("fake_tool", project_root=tmpdir)
        if source != "local":
            errors.append(f"Test 3 failed: expected 'local', got '{source}'")
        if path != fake_bin:
            errors.append(f"Test 3 failed: expected {fake_bin}, got {path}")

    return {"pass": len(errors) == 0, "errors": errors}


def main():
    parser = argparse.ArgumentParser(
        description="Resolve local-build binary path, falling back to system version"
    )
    parser.add_argument("--test", action="store_true", help="Run built-in tests")
    parser.add_argument("name", nargs="?", help="Binary name to resolve (e.g. rick)")
    parser.add_argument(
        "--project-root",
        default=None,
        help="Project root directory (default: auto-detect via .rick/)",
    )
    args = parser.parse_args()

    if args.test:
        result = run_tests()
        print(json.dumps(result))
        sys.exit(0 if result["pass"] else 1)

    if not args.name:
        parser.print_help()
        sys.exit(1)

    try:
        path, source = resolve_binary(args.name, project_root=args.project_root)
        result = {"pass": True, "path": path, "source": source, "errors": []}
        print(json.dumps(result))
    except FileNotFoundError as e:
        result = {"pass": False, "path": None, "source": None, "errors": [str(e)]}
        print(json.dumps(result))
        sys.exit(1)


if __name__ == "__main__":
    main()
