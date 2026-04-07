# Description: 构建 rick 并返回本地二进制路径，测试脚本用此代替系统安装版

import json
import os
import subprocess
import sys


def build_and_get_rick_bin(project_root=None):
    """
    Run `go build -o ./bin/rick ./cmd/rick` and return the path to the binary.
    Always builds first to ensure the binary contains the latest code changes.

    Returns (bin_path, error_message).
    error_message is None on success.
    """
    if project_root is None:
        # Walk up to find project root (.rick/ dir)
        current = os.path.abspath(os.getcwd())
        while True:
            if os.path.isdir(os.path.join(current, ".rick")):
                project_root = current
                break
            parent = os.path.dirname(current)
            if parent == current:
                return None, "could not find project root (.rick/ directory)"
            current = parent

    bin_path = os.path.join(project_root, "bin", "rick")
    os.makedirs(os.path.join(project_root, "bin"), exist_ok=True)

    result = subprocess.run(
        ["go", "build", "-o", bin_path, "./cmd/rick"],
        cwd=project_root,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        return None, f"go build failed:\n{result.stderr}"

    return bin_path, None


def main():
    import argparse
    parser = argparse.ArgumentParser(
        description="Build rick and return the path to the local binary"
    )
    parser.add_argument("--project-root", default=None)
    args = parser.parse_args()

    bin_path, err = build_and_get_rick_bin(args.project_root)
    if err:
        print(json.dumps({"pass": False, "bin_path": None, "errors": [err]}))
        sys.exit(1)
    print(json.dumps({"pass": True, "bin_path": bin_path, "errors": []}))


if __name__ == "__main__":
    main()
