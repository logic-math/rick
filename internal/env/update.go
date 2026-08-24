package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/runtime"
)

// UpdateTarget 标识 update-pi 要更新的对象（env 职责 1+2 的更新侧收口）。
type UpdateTarget int

const (
	// UpdateAll 更新 pi runtime + 全部扩展 + 模型目录（默认）。
	UpdateAll UpdateTarget = iota
	// UpdateSelf 仅更新 pi runtime。
	UpdateSelf
	// UpdateExtensions 仅更新全部已注册扩展。
	UpdateExtensions
	// UpdateModels 仅刷新模型目录。
	UpdateModels
	// UpdateOne 仅更新指定名称的单个扩展。
	UpdateOne
)

// ParseUpdateTarget 把 CLI 位置参数解析为更新目标。空串 → All；
// "all" → All；"pi"/"self" → Self；"extensions"/"ext" → Extensions；
// "models" → Models；其他任意非空串视为扩展名 → One（返回原始名）。
func ParseUpdateTarget(arg string) (UpdateTarget, string) {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "", "all":
		return UpdateAll, ""
	case "pi", "self":
		return UpdateSelf, ""
	case "extensions", "ext":
		return UpdateExtensions, ""
	case "models":
		return UpdateModels, ""
	default:
		return UpdateOne, strings.TrimSpace(arg)
	}
}

// updateStep 是更新计划中的单个动作。
type updateStep struct {
	kind   string // "pi" | "extensions" | "one" | "models"
	source string // 仅 "one" 使用（已解析的注册源名）
}

// buildUpdatePlan 生成有序更新步骤（纯函数，可单测）。
// All 顺序 = pi → extensions → models：先换 runtime 再更新其上安装的扩展。
func buildUpdatePlan(target UpdateTarget, one string) []updateStep {
	switch target {
	case UpdateAll:
		return []updateStep{
			{kind: "pi"},
			{kind: "extensions"},
			{kind: "models"},
		}
	case UpdateSelf:
		return []updateStep{{kind: "pi"}}
	case UpdateExtensions:
		return []updateStep{{kind: "extensions"}}
	case UpdateModels:
		return []updateStep{{kind: "models"}}
	case UpdateOne:
		return []updateStep{{kind: "one", source: one}}
	}
	return nil
}

// UpdateResult 汇报 update-pi 实际发生的事。
type UpdateResult struct {
	PiBefore          string   // 更新前 pi 版本（探测失败为 ""）
	PiAfter           string   // 更新后 pi 版本
	PiUpdated         bool     // pi runtime 是否执行了更新
	ExtensionsUpdated bool     // 是否更新了全部扩展
	OneUpdated        string   // 单扩展更新时实际使用的源名（"" = 未执行）
	ModelsRefreshed   bool     // 是否刷新了模型目录
	ManagedRuntime    bool     // 更新的是 rick 托管 runtime（true）还是 PATH pi（false）
	Warnings          []string // 非致命告警（如定制重部署失败）
}

// UpdatePi 执行 pi runtime / 扩展 / 模型目录的更新（env 职责 1+2）。
//
// pi runtime：托管 runtime（AgentDir/runtime）存在时用 rick 自己的
// InstallManagedPI 重装最新版（npm --prefix，语义与 init-pi 一致，绕开
// `pi update --self` 对非全局 npm 安装的 guard）；无托管 runtime（用户 PATH
// pi）时委托 `pi update --self`，由 pi 自行判断可行性。
//
// 扩展：`pi update --extensions`（全部）或 `pi update <source>`（单个），
// 均经 PiCommand 注入 PI_CODING_AGENT_DIR，作用于 rick 托管的 agent 目录。
//
// 模型目录：`pi update --models`。
//
// 更新完成后尽力重部署 rick 定制（agents/hooks/skills，幂等）；失败仅告警。
func UpdatePi(target UpdateTarget, one string) (*UpdateResult, error) {
	// 单扩展更新前先解析注册源名（带 npm: 前缀的注册形态），未注册直接报错。
	if target == UpdateOne {
		resolved, err := resolveExtensionSource(one)
		if err != nil {
			return nil, err
		}
		one = resolved
	}

	res := &UpdateResult{}
	res.PiBefore = PiVersion("")

	for _, step := range buildUpdatePlan(target, one) {
		switch step.kind {
		case "pi":
			managed := runtime.FileExists(runtime.RuntimeBin())
			res.ManagedRuntime = managed
			if managed {
				fmt.Println("→ updating rick's managed pi runtime (npm --prefix latest) ...")
				if err := InstallManagedPI(""); err != nil {
					return res, fmt.Errorf("update managed pi runtime: %w", err)
				}
			} else {
				if _, err := exec.LookPath("pi"); err != nil {
					return res, fmt.Errorf("no managed pi runtime and no pi on PATH — run `rick tools init-pi` first")
				}
				fmt.Println("→ no managed runtime — delegating to `pi update --self` (PATH pi) ...")
				cmd := PiCommand("update", "--self")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return res, fmt.Errorf("pi update --self: %w", err)
				}
			}
			res.PiUpdated = true
		case "extensions":
			fmt.Println("→ updating all registered extensions (`pi update --extensions`) ...")
			if err := runPiUpdate("--extensions"); err != nil {
				return res, fmt.Errorf("pi update --extensions: %w", err)
			}
			res.ExtensionsUpdated = true
		case "one":
			fmt.Printf("→ updating extension %s (`pi update %s`) ...\n", step.source, step.source)
			if err := runPiUpdate(step.source); err != nil {
				return res, fmt.Errorf("pi update %s: %w", step.source, err)
			}
			res.OneUpdated = step.source
		case "models":
			fmt.Println("→ refreshing model catalogs (`pi update --models`) ...")
			if err := runPiUpdate("--models"); err != nil {
				return res, fmt.Errorf("pi update --models: %w", err)
			}
			res.ModelsRefreshed = true
		}
	}

	// 更新后重部署 rick 自有定制（agents/hooks/skills，幂等；cwd 不在 rick
	// 仓库时 skills 源不可得，仅告警——init-pi 会补齐）。
	if err := DeployRickCustomizations(); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("redeploy rick customizations failed (run `rick tools init-pi` in the rick repo to fix): %v", err))
	}

	res.PiAfter = PiVersion("")
	return res, nil
}

// runPiUpdate 运行 `pi update <args...>`（继承 stdio，作用于托管 agent 目录）。
func runPiUpdate(args ...string) error {
	cmd := PiCommand(append([]string{"update"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// resolveExtensionSource 把用户输入的扩展名解析为 pi list 中的注册源名。
// 接受裸名（pi-subagents）或注册形态（npm:pi-subagents）。注意：裸名是注册行
// （npm:pi-subagents）的子串，所以必须**前缀形态优先**判断，否则裸名会被
// PiListContains 子串误命中而不加前缀地传给 pi update。
func resolveExtensionSource(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty extension name")
	}
	if prefixed := "npm:" + name; PiListContains(prefixed) {
		return prefixed, nil // 注册形态是 npm:<name>，用前缀形态调用
	}
	if PiListContains(name) {
		return name, nil // 直接命中（如 local: 源或恰好是完整注册行）
	}
	registered := registeredExtensionSources()
	return "", fmt.Errorf("extension %q is not registered (registered: %s)",
		name, strings.Join(registered, ", "))
}

// registeredExtensionSources 返回 `pi list` 中已注册扩展的源名清单（尽力而为，
// 解析失败返回提示而非报错）。
func registeredExtensionSources() []string {
	out, err := PiCommand("list").Output()
	if err != nil {
		return []string{"<pi list failed>"}
	}
	var sources []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "npm:") || strings.HasPrefix(line, "local:") {
			sources = append(sources, line)
		}
	}
	if len(sources) == 0 {
		sources = append(sources, "<none>")
	}
	return sources
}

// QuickCheckResult 是 update-pi 完成后的快速功能自检结果（职责 4 的更新侧
// 变体：只检「更新可能破坏的东西」——pi 可用性、扩展注册、定制落盘、
// rick-gates hook 可解析）。
type QuickCheckResult struct {
	PiVersion         string   // 更新后 pi 版本（探测失败为 ""）
	MissingExtensions []string // 未注册的必需扩展（空 = 全部就绪）
	NotReady          []string // CheckReady 未就绪项（空 = 全部就绪）
	GatesHelperOK     bool     // rick-gates helper.py 可被 python3 解析
	GatesHelperNote   string   // 不可解析/无法检查的原因
}

// QuickCheck 在 update-pi 后对 rick 的 pi 环境做快速功能检查。纯检查、无副作用。
func QuickCheck() QuickCheckResult {
	res := QuickCheckResult{PiVersion: PiVersion("")}
	res.MissingExtensions = VerifyExtensions()
	res.NotReady = CheckReady()

	helper := filepath.Join(runtime.AgentDir(), "extensions", "rick-gates", "helper.py")
	if !runtime.FileExists(helper) {
		res.GatesHelperNote = fmt.Sprintf("helper.py not found at %s", helper)
		return res
	}
	if _, err := exec.LookPath("python3"); err != nil {
		res.GatesHelperNote = "python3 not on PATH — skipped syntax check"
		return res
	}
	cmd := exec.Command("python3", "-c",
		`import ast,sys; ast.parse(open(sys.argv[1],encoding="utf-8").read())`, helper)
	if out, err := cmd.CombinedOutput(); err != nil {
		res.GatesHelperNote = fmt.Sprintf("python3 syntax check failed: %v: %s",
			err, strings.TrimSpace(string(out)))
		return res
	}
	res.GatesHelperOK = true
	return res
}
