package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
)

func formatOKRContent(okrInfo *parser.ContextInfo) string {
	if okrInfo == nil || (len(okrInfo.Objectives) == 0 && len(okrInfo.KeyResults) == 0) {
		return "暂无项目 OKR 信息"
	}
	var b strings.Builder
	if len(okrInfo.Objectives) > 0 {
		b.WriteString("**Objectives**:\n")
		for _, obj := range okrInfo.Objectives {
			b.WriteString(fmt.Sprintf("- %s\n", obj))
		}
		b.WriteString("\n")
	}
	if len(okrInfo.KeyResults) > 0 {
		b.WriteString("**Key Results**:\n")
		for _, kr := range okrInfo.KeyResults {
			b.WriteString(fmt.Sprintf("- %s\n", kr))
		}
	}
	return b.String()
}

func formatSPECContent(specInfo *parser.ContextInfo) string {
	if specInfo == nil || len(specInfo.Specifications) == 0 {
		return "暂无项目 SPEC 信息"
	}
	var b strings.Builder
	b.WriteString("**Specifications**:\n")
	for _, spec := range specInfo.Specifications {
		b.WriteString(fmt.Sprintf("- %s\n", spec))
	}
	return b.String()
}

func formatCompletedWork(history []string) string {
	if len(history) == 0 {
		return "这是项目的第一阶段规划"
	}
	var b strings.Builder
	b.WriteString("**已完成的工作:**\n")
	for _, item := range history {
		b.WriteString(fmt.Sprintf("- %s\n", item))
	}
	return b.String()
}
