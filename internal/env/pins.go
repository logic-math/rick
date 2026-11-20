package env

import (
	"strings"
)

// 外部依赖版本固定表（job_55，2026-11-20 服务器迁移实证）。
//
// 为什么固定：pi 生态版本必须配套——pi-subagents 0.89.1 依赖 pi ≥0.86 的
// toToolDeclaration API，与 pi 0.84.2 组合时扩展加载失败、pi worker 启动即
// exit 1（实测于 10.99.19.123 迁移）。pi 包管理器对不带版本号的 spec 一律
// 解析为 latest（pi install / pi update --extensions 同理），等于随机引入
// 不兼容组合。
//
// 升级纪律：改本表 = 显式决策；pi 与扩展必须一起升并全量回归
// （go build + go test + init-pi 冒烟）。init-pi / update-pi 只按本表安装，
// 绝不装 latest。
const (
	// PinnedPiVersion 是 rick 托管 pi runtime（AgentDir/runtime）的固定版本。
	PinnedPiVersion = "0.84.2"
)

// PinnedExtension 是一个 rick 必需扩展及其固定版本（pi install npm:<pkg>@<version>）。
type PinnedExtension struct {
	Pkg     string // npm 包名（如 pi-subagents）
	Version string // 固定版本号
}

// PinnedExtensions 是 rick 必需扩展的固定版本表（RequiredExtensions 由此派生）。
var PinnedExtensions = []PinnedExtension{
	{Pkg: "pi-subagents", Version: "0.51.0"},
	{Pkg: "pi-web-access", Version: "0.24.0"},
}

// PinnedExtensionSpec 返回扩展的固定安装 spec（"npm:pi-subagents@0.51.0"
// 形态，直接用于 pi install）；未知扩展返回 ""。
func PinnedExtensionSpec(pkg string) string {
	for _, p := range PinnedExtensions {
		if p.Pkg == pkg {
			return "npm:" + p.Pkg + "@" + p.Version
		}
	}
	return ""
}

// PinnedExtensionVersion 返回扩展的固定版本号；未知扩展返回 ""。
func PinnedExtensionVersion(pkg string) string {
	for _, p := range PinnedExtensions {
		if p.Pkg == pkg {
			return p.Version
		}
	}
	return ""
}

// pinnedExtensionNames 返回固定版本表的包名清单（RequiredExtensions 的派生源）。
func pinnedExtensionNames() []string {
	names := make([]string, 0, len(PinnedExtensions))
	for _, p := range PinnedExtensions {
		names = append(names, p.Pkg)
	}
	return names
}

// packageNameFromSource 从注册源名（pi list 形态）提取裸包名：
// 去掉 "npm:" 前缀与 "@version" 后缀（"npm:pi-subagents@0.51.0" → "pi-subagents"）。
func packageNameFromSource(source string) string {
	s := source
	if len(s) > 4 && s[:4] == "npm:" {
		s = s[4:]
	}
	if i := strings.LastIndex(s, "@"); i >= 0 {
		s = s[:i]
	}
	return s
}
