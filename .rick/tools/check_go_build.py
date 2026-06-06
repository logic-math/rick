# Description: 检查 Go 项目是否能成功编译，输出 JSON 结果

import subprocess, json, sys

result = subprocess.run(
    ["go", "build", "./..."],
    capture_output=True, text=True
)

output = {
    "pass": result.returncode == 0,
    "errors": [result.stderr.strip()] if result.returncode != 0 else []
}
print(json.dumps(output, ensure_ascii=False))
sys.exit(0 if output["pass"] else 1)
