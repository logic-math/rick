package env

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/sunquan/rick/internal/runtime"
)

// requiredRickAgents 是 rick 自定义 agent 的清单（对应 AgentDir()/agents/<name>.md）。
// human-loop v7.x 四 agent（重生型）由 env 职责 3 落盘（task9 机制）并在此注册
// 为就绪 check 的必需项：任一文件缺失即报未就绪。
// v7.2（job_50）：交互型环节（费曼演绎/ZPD 档案/探索流）收回 parent 会话内联
// 执行，agent 收缩为 4 个；zpd-assessor/feynman/originality-sparker 移除。
var requiredRickAgents = []string{
	"research",
	"syllabus-builder",
	"lesson-writer",
	"recall-review",
}

// IsPIReady 汇总所有功能点就绪判定，返回 (ok, missing)。ok=true 且 missing 为空
// 表示 pi 环境就绪。纯「功能点就绪」，不含任何 session 校验（session 归 runtime）。
func IsPIReady() (bool, []string) {
	missing := CheckReady()
	return len(missing) == 0, missing
}

// CheckReady 汇总四职责的功能点就绪，返回未就绪点清单（空 = 全部就绪）。
func CheckReady() []string {
	var missing []string
	missing = append(missing, CheckPIInstalled()...)
	missing = append(missing, CheckEcosystemExtensions()...)
	missing = append(missing, CheckRickAgents()...)
	missing = append(missing, CheckRickHooks()...)
	return missing
}

// CheckPIInstalled 检查 pi agent 是否可用（托管 runtime 二进制或 PATH pi）。
// 就绪返回 nil，否则返回缺失描述。
func CheckPIInstalled() []string {
	if runtime.FileExists(runtime.RuntimeBin()) {
		return nil
	}
	if _, err := exec.LookPath("pi"); err == nil {
		return nil
	}
	return []string{"pi runtime not installed"}
}

// CheckEcosystemExtensions 检查 pi 生态扩展（pi-subagents/pi-web-access）是否
// 注册。就绪返回 nil，缺失时返回未注册扩展名。
func CheckEcosystemExtensions() []string {
	return VerifyExtensions()
}

// CheckExtensionVersionDrift 核对已注册的固定扩展版本是否与 pin 表一致
// （job_55：pi-subagents 0.89.1 vs pinned 0.51.0 的不兼容即此类漂移）。
// 返回漂移描述清单（空 = 全部一致）。未注册的扩展不报（那是
// MissingExtensions 的事）；installed 版本读不到时不报（无法判定，容总）。
func CheckExtensionVersionDrift() []string {
	var drift []string
	for _, p := range PinnedExtensions {
		got := InstalledExtensionVersion(p.Pkg)
		if got == "" || got == p.Version {
			continue
		}
		drift = append(drift, fmt.Sprintf("%s installed %s ≠ pinned %s", p.Pkg, got, p.Version))
	}
	return drift
}

// CheckRickAgents 检查 rick 自定义 agent（v7.x 四 agent：research + 三个重生型
// 教学 agent）是否落盘到 AgentDir()/agents/<name>.md。就绪返回 nil，缺失时返回
// 缺失 agent 名。
func CheckRickAgents() []string {
	var missing []string
	for _, name := range requiredRickAgents {
		if !runtime.FileExists(filepath.Join(runtime.AgentDir(), "agents", name+".md")) {
			missing = append(missing, name)
		}
	}
	return missing
}

// CheckRickHooks 检查 rick-gates hook 扩展是否落盘
// （AgentDir()/extensions/rick-gates/helper.py）。就绪返回 nil。
func CheckRickHooks() []string {
	p := filepath.Join(runtime.AgentDir(), "extensions", "rick-gates", "helper.py")
	if !runtime.FileExists(p) {
		return []string{"rick-gates"}
	}
	return nil
}
