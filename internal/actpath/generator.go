package actpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/agent"
)

const maxFinalMessageLen = 200

// Generate creates an act-path.md report at outputFile from the given session.
// The output directory is created automatically if it does not exist.
func Generate(session agent.AgentSession, outputFile string) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}

	toolCalls := session.ToolCalls()
	errorCount := 0
	for _, tc := range toolCalls {
		if tc.IsError {
			errorCount++
		}
	}

	rawLogBase := filepath.Base(session.RawLogPath())

	var sb strings.Builder

	fmt.Fprintf(&sb, "# act-path\n\n")

	fmt.Fprintf(&sb, "## 执行摘要\n\n")
	fmt.Fprintf(&sb, "- Session ID: %s\n", session.ID())
	fmt.Fprintf(&sb, "- 耗时: %s\n", session.Duration())
	fmt.Fprintf(&sb, "- 工具调用次数: %d\n", len(toolCalls))
	fmt.Fprintf(&sb, "- 报错次数: %d\n", errorCount)
	fmt.Fprintf(&sb, "- 完整日志: [%s](%s)\n\n", rawLogBase, session.RawLogPath())

	fmt.Fprintf(&sb, "## 行为轨迹\n\n")
	fmt.Fprintf(&sb, "| 行号 | 工具 | 输入 | 错误 |\n")
	fmt.Fprintf(&sb, "|------|------|------|------|\n")
	for _, tc := range toolCalls {
		errStr := ""
		if tc.IsError {
			errStr = "✗"
		}
		fmt.Fprintf(&sb, "| [L%d](%s:%d) | %s | %s | %s |\n",
			tc.Line, session.RawLogPath(), tc.Line, tc.Name, tc.Input, errStr)
	}
	fmt.Fprintf(&sb, "\n")

	fmt.Fprintf(&sb, "## Agent 最终输出\n\n")
	msg := session.FinalMessage()
	runes := []rune(msg)
	if len(runes) > maxFinalMessageLen {
		msg = string(runes[:maxFinalMessageLen])
	}
	fmt.Fprintf(&sb, "%s\n\n", msg)
	fmt.Fprintf(&sb, "> [%s:%d](%s)\n", rawLogBase, session.FinalMessageLine(), session.RawLogPath())

	return os.WriteFile(outputFile, []byte(sb.String()), 0644)
}
