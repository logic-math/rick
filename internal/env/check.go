package env

import (
	"os/exec"
	"path/filepath"

	"github.com/sunquan/rick/internal/runtime"
)

// requiredRickAgents 是 rick 自定义 agent 的清单（对应 AgentDir()/agents/<name>）。
// task3 保持为空：think/research/exporter 由 task9 经职责 3 落盘并在此注册。
// 空清单下 CheckRickAgents 恒为就绪（无必需 agent）。
var requiredRickAgents = []string{}

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

// CheckRickAgents 检查 rick 自定义 agent（think/research/exporter）是否落盘。
// task3 无必需 agent，恒为就绪；task9 注册后检查对应文件。
func CheckRickAgents() []string {
	var missing []string
	for _, name := range requiredRickAgents {
		if !runtime.FileExists(filepath.Join(runtime.AgentDir(), "agents", name)) {
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
