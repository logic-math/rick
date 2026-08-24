package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sunquan/rick/internal/config"
)

// CLIMode selects how the pi binary is invoked.
type CLIMode int

const (
	// ModeInteractive runs `pi [extraArgs] <promptFile>` with stdin/stdout/stderr
	// inherited from rick (human-facing interactive session).
	ModeInteractive CLIMode = iota
	// ModePrint runs `pi -p [extraArgs] <promptFile>` in non-interactive print
	// mode, forwarding stdout/stderr to the terminal. pi has no permission popups,
	// so no --dangerously-skip-permissions flag is needed.
	ModePrint
)

// FindBinary resolves the pi binary path: cfg.PiPath if set, otherwise rick's
// self-contained runtime (~/.rick/pi/agent/runtime) if installed, otherwise
// "pi" looked up in PATH. cfg may be nil (PATH-only lookup, used by auto-fix
// sites that have no config context). Returns an error if pi is neither
// configured, installed in the managed runtime, nor on PATH.
func FindBinary(cfg *config.Config) (string, error) {
	if cfg != nil && cfg.PiPath != "" {
		return cfg.PiPath, nil
	}
	if bin := RuntimeBin(); FileExists(bin) {
		return bin, nil
	}
	path, err := exec.LookPath("pi")
	if err != nil {
		return "", fmt.Errorf("pi binary not found (set pi_path in config, run rick tools init-pi, or install pi): %w", err)
	}
	return path, nil
}

// piPathOrDefault returns cfg.PiPath, else rick's managed runtime pi if
// installed, else "pi" without a PATH check. Used by CallCLI where a missing
// binary should surface as a natural exec error rather than a pre-flight
// failure.
func piPathOrDefault(cfg *config.Config) string {
	if cfg != nil && cfg.PiPath != "" {
		return cfg.PiPath
	}
	if bin := RuntimeBin(); FileExists(bin) {
		return bin
	}
	return "pi"
}

// bootstrapMessage is the initial user message that kicks the agent off.
// v4.4.5: 阶段提示词（plan/doing/easy/ctrl/human-loop/learning/dream）不再作为
// 初始 user 消息（长会话 compaction 会压缩掉协议细节），而是通过
// --append-system-prompt 常驻系统提示词（pi 对该参数自动检测文件路径并读取
// 内容）；user 消息只做启动触发——协议在系统提示词里，永不压缩遗忘。
const bootstrapMessage = "开始：按系统提示词中的 rick 协议立即执行你的职责。"

// buildArgs assembles the pi argument list for a given mode. It is split out
// from CallCLI so flag logic can be unit-tested without shelling out.
//
// extraArgs are pi flags passed through verbatim (e.g. "--session", "<id>" or
// "--continue"). promptFile, when non-empty, is injected via
// --append-system-prompt (system-prompt persistence across compaction) and the
// bootstrap message becomes the initial user prompt; empty promptFile keeps
// the legacy flag-only form (easy.go's resume path).
func buildArgs(mode CLIMode, promptFile string, extraArgs ...string) []string {
	args := make([]string, 0, len(extraArgs)+4)
	if mode == ModePrint {
		args = append(args, "-p")
	}
	args = append(args, extraArgs...)
	if promptFile != "" {
		args = append(args, "--append-system-prompt", promptFile, bootstrapMessage)
	}
	return args
}

// CallCLI invokes the pi binary for the direct (non-AgentSession) call sites.
// Interactive inherits stdio; Print forwards stdout/stderr to the terminal.
// verbose mirrors the previous callClaudeCodeCLI [INFO] log line.
//
// pi flags come from two sources, in this order: cfg.PiExtraArgs (global, e.g.
// --provider/--model/--api-key) then extraArgs (per-call, e.g. --session <id>).
// promptFile is always appended last.
func CallCLI(verbose bool, cfg *config.Config, promptFile string, mode CLIMode, extraArgs ...string) error {
	piBin := piPathOrDefault(cfg)
	merged := mergeExtraArgs(cfg, extraArgs)
	args := buildArgs(mode, promptFile, merged...)

	if verbose {
		fmt.Printf("[INFO] Executing: %s %s\n", piBin, strings.Join(args, " "))
	}

	cmd := exec.Command(piBin, args...)
	if mode == ModeInteractive {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// rick manages pi's config under ~/.rick/pi/agent (PI_CODING_AGENT_DIR),
	// isolated from the user's own ~/.pi — every pi subprocess must see it.
	cmd.Env = AgentEnv()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pi CLI failed: %w", err)
	}
	return nil
}

// mergeExtraArgs prepends cfg.PiExtraArgs to the caller's extraArgs (global
// config flags first, then per-call flags). cfg may be nil.
func mergeExtraArgs(cfg *config.Config, extraArgs []string) []string {
	if cfg == nil || len(cfg.PiExtraArgs) == 0 {
		return extraArgs
	}
	merged := make([]string, 0, len(cfg.PiExtraArgs)+len(extraArgs))
	merged = append(merged, cfg.PiExtraArgs...)
	merged = append(merged, extraArgs...)
	return merged
}
