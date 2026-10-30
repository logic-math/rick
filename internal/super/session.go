// session.go — super 中控会话 prompt 构建（job_47/task11 + job_48/task5 目录模型 v2）。
//
// BuildSuperPrompt 组装 `rick super`（默认 session 子命令）的交互提示词：
//   - 本次进化使命（requirement：参数优先，缺省回读 <ws>/requirement.md；
//     两者皆无 → 降级提示「先与 human 对齐后落盘」）
//   - 工作流程三步：grilling 需求追问（skill 全文 inline，工作目录 <ws>/grilling）
//     → gen_env 生成 env → 运行/观察/干预
//   - super-loop 全文 inline（super 会话默认加载的工作法，.rick/loops/super-loop.md）
//   - gen_env skill 全文 inline（env 建设引导，.rick/skills/gen_env_skill/skill.md）
//   - grilling skill 全文 inline（embedded 模板，编译期内嵌永不缺失；
//     {{grilling_workdir}} 替换为 <ws>/grilling）
//   - Loops / Skills 目录（LoadLoopsContext / LoadSkillsContext）
//   - super 方法 SOP + CLI 速查
//   - 工作区上下文（job_48/task4：工作区根/super 路径/env 生成约定/观察与干预手段）
//   - 工作区速查（ws 路径/super.json/events/干预队列）
//
// 产物落 <ws>/sessions/<uuid>/{session_id,method.md,prompt.md}：session_id 持久化
// 供 `rick super --resume <name>` 经 pi `--session <id>` 恢复（bugs.md：--session
// 加载已有语义）。与 easy（jobs/<id>/doing/prompts/）不同，super 会话锚定在
// super 工作区（<ws>/.rick/super/env_N）而非 job 目录——进化系统的运行状态本就以 ws 为家。
package super

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunquan/rick/internal/prompt"
)

// superMethodSOP 是 super 会话的方法层浓缩（commandMethod 的 super 侧对应物——
// internal/builder 不在 super 包写域内，故在本包自持文案）。
const superMethodSOP = `Super 方法（super-loop 中控）：env 建设与升级（gen_env skill 现场生成/演进 env，门禁系统设计是 env 范畴）→ 启动前门先证红确认（exploit 拒识 100% 才放行）→ rick super start/stop/status 启停监控后台进化 → rick super event 投递 human 干预（经干预队列，绝不直写 events.jsonl/archive.json——单写者纪律）。进化逻辑（采样/门禁/档案/淘汰）全部在后台 py 调度器，中控只做编排与人机接口。`

// superCLICheatsheet 是五子命令速查（与 internal/cmd/super.go 对齐）。
const superCLICheatsheet = `## CLI 速查

- rick super --name <n>                    # 中控会话（本会话）
- rick super start --name <n> --env <path> # 初始化工作区 + 拉起 py 调度器（--daemon 后台）
- rick super status --name <n> [--json]    # 状态 + 心跳 + 北极星（迭代/入格/token）
- rick super event --name <n> --type <t> --payload '<json>'  # 干预入队
- rick super stop --name <n>               # 优雅停（SIGTERM 落完当前事件 → SIGKILL 兜底）`

// workspaceCheatsheet 渲染工作区速查段。
func workspaceCheatsheet(ws string) string {
	var b strings.Builder
	b.WriteString("## 工作区速查\n\n")
	b.WriteString(fmt.Sprintf("- 工作区（ws）：%s\n", ws))
	b.WriteString(fmt.Sprintf("- 运行元信息：%s\n", filepath.Join(ws, "super.json")))
	b.WriteString(fmt.Sprintf("- 事件流（唯一真相源，只读）：%s\n", filepath.Join(ws, "events.jsonl")))
	b.WriteString(fmt.Sprintf("- 精英档案（只读）：%s\n", filepath.Join(ws, "archive.json")))
	b.WriteString(fmt.Sprintf("- 干预队列（human 写入口）：%s\n", filepath.Join(ws, "interventions")))
	b.WriteString(fmt.Sprintf("- env（gen_env 产出）：%s\n", filepath.Join(ws, "env")))
	b.WriteString(fmt.Sprintf("- 后台日志（--daemon）：%s\n", filepath.Join(ws, "super.log")))
	b.WriteString("\n事件类型速查与目录协议详见仓库 super/README.md。\n")
	return b.String()
}

// workspaceContext 渲染工作区上下文块（job_48/task4：中控会话开箱即知全部操作面）。
// rickDir 是 .rick 目录（workspaceRoot = 其上层）；ws 是 super 工作区绝对路径
// （<workspaceRoot>/.rick/super/env_N——目录模型 v2）。保持 SOP 级简洁——不是文档。
func workspaceContext(rickDir, ws string) string {
	workspaceRoot := filepath.Dir(rickDir)
	var b strings.Builder
	b.WriteString("## 工作区上下文\n\n")
	b.WriteString(fmt.Sprintf("- 当前 rick 工作区根：%s\n", workspaceRoot))
	b.WriteString(fmt.Sprintf("- super 工作区（本会话的 env 目录）：%s\n", ws))
	b.WriteString("- env 生成约定：按 gen_env skill 引导现场生成/演进，产出落 ")
	b.WriteString(fmt.Sprintf("%s\n", filepath.Join(ws, "env")))
	b.WriteString("\n### 观察手段（当前运行状态，任意信息可读）\n\n")
	b.WriteString("- CLI：`rick super status --json`（心跳/迭代游标/北极星）；事件尾随可读事件流文件\n")
	b.WriteString(fmt.Sprintf("- 直读文件（真相源）：`%s`（事件流）\n", filepath.Join(ws, "events.jsonl")))
	b.WriteString(fmt.Sprintf("  `%s`（精英格档案）；`%s/`（每代代码快照 + prompt + agent 产物）\n",
		filepath.Join(ws, "archive.json"), filepath.Join(ws, "codes")))
	b.WriteString(fmt.Sprintf("  `%s`（心跳）\n", filepath.Join(ws, "heartbeat.json")))
	b.WriteString("- pi agent 的执行 session 文件同样可读——只要能描绘运行状态，读任何信息\n")
	b.WriteString("\n### 干预手段\n\n")
	b.WriteString("- `rick super event --type <t> --payload '<json>'`（写干预队列，由调度器在迭代边界摄取；单写者保证——绝不直写 events.jsonl / archive.json）\n")
	return b.String()
}

// newSessionUUID 生成 v4 UUID（与 handler.generateUUID 同型；本包不 import handler）。
func newSessionUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// missionBlock 渲染本次进化使命段：requirement 非空 → 使命全文 + requirement.md
// 锚点；空 → 降级提示（第一步先与 human 对齐落盘 requirement.md）。
func missionBlock(ws, requirement string) string {
	reqPath := filepath.Join(ws, "requirement.md")
	var b strings.Builder
	b.WriteString("## 本次进化使命（requirement）\n\n")
	if requirement != "" {
		b.WriteString(requirement)
		b.WriteString("\n\n需求原文锚点：")
		b.WriteString(reqPath)
		b.WriteString("（会话全程以此为锚；human 修订后同步写回该文件）\n\n---\n\n")
	} else {
		b.WriteString("（本工作区暂无 requirement.md——第一步先与 human 对齐进化目标，落盘 ")
		b.WriteString(reqPath)
		b.WriteString("）\n\n---\n\n")
	}
	return b.String()
}

// workflowBlock 渲染三步工作流（grilling 需求追问 → gen_env 生成 env → 运行/观察/干预）。
func workflowBlock(ws string) string {
	var b strings.Builder
	b.WriteString("## 工作流程（三步，按序推进）\n\n")
	b.WriteString("**第一步：需求追问（grilling）**——对本次进化使命跑设计树追问循环（skill: grilling 全文见下），")
	b.WriteString("直到需求树落盘达标（")
	b.WriteString(filepath.Join(ws, "grilling", "design-tree.md"))
	b.WriteString("，经 grilling_gate 校验）。需求未理清前不进入 env 创建。\n\n")
	b.WriteString("**第二步：生成 env（gen_env）**——按 gen_env skill 引导现场生成/演进 env，产出落 ")
	b.WriteString(filepath.Join(ws, "env"))
	b.WriteString("（含门禁系统设计引导——门禁/容差/评分属 env 范畴）。\n\n")
	b.WriteString("**第三步：运行/观察/干预**——启动前门先证红确认（exploit 拒识 100% 才放行）→ `rick super start` 拉起后台进化 → 观察（status / 直读文件，见工作区上下文）→ `rick super event` 投递干预。\n\n---\n\n")
	return b.String()
}

// grillingSkillInline 返回 embedded grilling skill 全文，{{grilling_workdir}}
// 替换为 <ws>/grilling（embedded 模板编译期内嵌，永不缺失——不需运行时预检）。
func grillingSkillInline(ws string) (string, error) {
	content, err := prompt.ReadEmbeddedSkill("grilling")
	if err != nil {
		return "", fmt.Errorf("super: embedded grilling skill: %w", err)
	}
	return strings.ReplaceAll(content, "{{grilling_workdir}}", filepath.Join(ws, "grilling")), nil
}

// BuildSuperPrompt 组装 super 中控会话的 method/prompt 两件产物。
//
// rickDir 是 .rick 目录（loops/skills 加载源）；ws 是 super 工作区绝对路径；
// requirement 是本次进化使命（参数优先；空则回读 <ws>/requirement.md；
// 两者皆无 → prompt 里降级提示先对齐落盘——CLI 旧工作区兼容）。
// 每次调用新建 <ws>/sessions/<uuid>/ 会话目录（session_id + method.md +
// prompt.md），返回两个产物的绝对路径。失败原子：任一写失败返回 err
// （已建目录保留无害）。
func BuildSuperPrompt(rickDir, ws, requirement string) (methodFile, promptFile string, err error) {
	if rickDir == "" || ws == "" {
		return "", "", fmt.Errorf("super: BuildSuperPrompt requires rickDir and ws")
	}

	// requirement 解析：参数空 → 回读 <ws>/requirement.md（web prep 先落盘再调；
	// CLI resume 同一工作区时由此取回使命）
	if requirement == "" {
		if data, rerr := os.ReadFile(filepath.Join(ws, "requirement.md")); rerr == nil {
			requirement = strings.TrimSpace(string(data))
		}
	}

	grilling, gerr := grillingSkillInline(ws)
	if gerr != nil {
		return "", "", gerr
	}

	uuid, err := newSessionUUID()
	if err != nil {
		return "", "", fmt.Errorf("super: session uuid: %w", err)
	}
	sessDir := filepath.Join(ws, "sessions", uuid)
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		return "", "", fmt.Errorf("super: create session dir: %w", err)
	}

	// session_id（resume 契约：rick super --resume 经 pi --session 加载）
	if err := os.WriteFile(filepath.Join(sessDir, "session_id"), []byte(uuid), 0o644); err != nil {
		return "", "", fmt.Errorf("super: persist session id: %w", err)
	}

	// method.md：方法层（rick 前缀 + super SOP）
	method := "你是 rick 的 coding agent。rick = 方法（loops/skills/domain 体系），你 = 实现。工作前先读 domain 事实，执行中遵循 loop 流程，按需加载 skills。\n\n" + superMethodSOP + "\n"
	methodFile = filepath.Join(sessDir, "method.md")
	if err := os.WriteFile(methodFile, []byte(method), 0o644); err != nil {
		return "", "", fmt.Errorf("super: write method.md: %w", err)
	}

	// inline 材料：super-loop 全文 + gen_env skill 全文（缺失降级为显式提示，
	// 不静默——中控会话缺工作法必须可见）。
	superLoop := readInline(filepath.Join(rickDir, "loops", "super-loop.md"),
		"（super-loop.md 缺失：请先按 .rick/loops 规范落盘 super-loop）")
	genEnvSkill := readInline(filepath.Join(rickDir, "skills", "gen_env_skill", "skill.md"),
		"（gen_env_skill/skill.md 缺失：请先按三段式规范落盘）")

	var b strings.Builder
	b.WriteString("# super 中控会话\n\n")
	b.WriteString("你是 rick super 模式的**中控会话**：env 的建设与升级、后台进化引擎的启停与监控、human 干预的投递。进化本身由后台执行（Go PySupervisor 监督 → py 调度器四状态循环 → pi agent 池），**你不执行进化逻辑**。\n\n")
	b.WriteString(missionBlock(ws, requirement))
	b.WriteString("## 单写者纪律（不可协商）\n\n")
	b.WriteString("events.jsonl / archive.json 只有后台 py 调度器一个写者。你的任何干预一律经 `rick super event` 写干预队列，由调度器在迭代边界摄取——**永远不直写** events.jsonl / archive.json。\n\n")
	b.WriteString(workflowBlock(ws))
	b.WriteString("---\n\n# Loop: super-loop（默认加载，先完整读取再动手）\n\n")
	b.WriteString(superLoop)
	b.WriteString("\n\n---\n\n# Skill: grilling（第一步：需求追问，设计树协议）\n\n")
	b.WriteString(grilling)
	b.WriteString("\n\n---\n\n# Skill: gen-env（第二步：env 建设与升级时加载）\n\n")
	b.WriteString(genEnvSkill)
	b.WriteString("\n\n---\n\n")
	b.WriteString(prompt.LoadLoopsContext(filepath.Join(rickDir, "loops")))
	b.WriteString("\n")
	b.WriteString(prompt.LoadSkillsContext(filepath.Join(rickDir, "skills")))
	b.WriteString("\n---\n\n")
	b.WriteString(superCLICheatsheet)
	b.WriteString("\n\n---\n\n")
	b.WriteString(workspaceContext(rickDir, ws))
	b.WriteString("\n\n---\n\n")
	b.WriteString(workspaceCheatsheet(ws))

	promptFile = filepath.Join(sessDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte(b.String()), 0o644); err != nil {
		return "", "", fmt.Errorf("super: write prompt.md: %w", err)
	}
	return methodFile, promptFile, nil
}

// readInline 读取 inline 材料；缺失时返回 fallback 提示（带时间戳的缺席说明）。
func readInline(path, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback + fmt.Sprintf("\n（路径：%s，探测时间 %s）\n", path,
			time.Now().Format(time.RFC3339))
	}
	return string(data)
}
