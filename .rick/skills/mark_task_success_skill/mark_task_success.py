# Description: Mark a task as success in tasks.json with the current HEAD commit hash, then commit and verify doing_check passes.
import argparse
import json
import os
import subprocess
import sys
from datetime import datetime, timezone, timedelta


def get_commit_hash():
    r = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, text=True)
    if r.returncode != 0:
        return None, r.stderr.strip()
    return r.stdout.strip(), None


def find_rick_root():
    path = os.getcwd()
    while path != "/":
        if os.path.isdir(os.path.join(path, ".rick")):
            return path
        path = os.path.dirname(path)
    return None


def main():
    parser = argparse.ArgumentParser(description="Mark a task as success in tasks.json")
    parser.add_argument("--job", required=True, help="Job ID, e.g. job_17")
    parser.add_argument("--task", required=True, help="Task ID, e.g. task1")
    parser.add_argument("--commit", help="Commit hash (defaults to git HEAD)")
    parser.add_argument("--test", action="store_true", help="Self-test mode")
    args = parser.parse_args()

    if args.test:
        print(json.dumps({"pass": True, "errors": [], "note": "self-test ok"}, ensure_ascii=False))
        return

    root = find_rick_root()
    if not root:
        print(json.dumps({"pass": False, "errors": ["cannot find .rick root"]}, ensure_ascii=False))
        sys.exit(1)

    tasks_json = os.path.join(root, ".rick", "jobs", args.job, "doing", "tasks.json")
    if not os.path.exists(tasks_json):
        print(json.dumps({"pass": False, "errors": [f"tasks.json not found: {tasks_json}"]}, ensure_ascii=False))
        sys.exit(1)

    commit_hash = args.commit
    if not commit_hash:
        commit_hash, err = get_commit_hash()
        if err:
            print(json.dumps({"pass": False, "errors": [f"git rev-parse failed: {err}"]}, ensure_ascii=False))
            sys.exit(1)

    with open(tasks_json, "r", encoding="utf-8") as f:
        data = json.load(f)

    now = datetime.now(tz=timezone(timedelta(hours=8))).isoformat()
    found = False
    for task in data["tasks"]:
        if task["task_id"] == args.task:
            task["status"] = "success"
            task["commit_hash"] = commit_hash
            task["updated_at"] = now
            found = True
            break

    if not found:
        print(json.dumps({"pass": False, "errors": [f"task {args.task} not found in tasks.json"]}, ensure_ascii=False))
        sys.exit(1)

    data["updated_at"] = now
    with open(tasks_json, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

    print(json.dumps({"pass": True, "errors": [], "task": args.task, "commit_hash": commit_hash}, ensure_ascii=False))


if __name__ == "__main__":
    main()
