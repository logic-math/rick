package builder

// Method is one command-specific method-layer fragment (rick 方法描述). Methods
// are injected into the runtime's system prompt so they survive compaction,
// unlike instance context which rides in the user prompt.
type Method struct {
	// Name is the command the method belongs to, e.g. "plan", "doing", "easy".
	Name string
	// Content is the method-layer text (system prompt fragment).
	Content string
}

// AgentDef describes a runtime-specific custom agent built from methods. pi 的
// think/research/exporter 等自定义 agent 在后续 task（env 职责 3 / task9）落盘，
// 本类型为 RuntimeBuilder.BuildAgents 的返回载体预留。
type AgentDef struct {
	Name         string
	Description  string
	SystemPrompt string
}

// RuntimeBuilder is the escaping-layer seam for agent runtimes (builder 三件之
// xxxxbuilder). pi is the only implementation today (PIBuilder). Adding a new
// runtime (e.g. dsh) only means implementing this interface and registering a
// 对应的 RuntimeBuilder — cli/handler/env 不改；与新 runtime 更好适配的信息封装在此层。
type RuntimeBuilder interface {
	// Name returns the runtime identifier ("pi").
	Name() string

	// BuildAgents turns method-layer fragments into runtime-specific agent
	// definitions (预留：pi 当前不注册额外 agent，返回空列表).
	BuildAgents(method []Method) ([]AgentDef, error)

	// BuildPrompt renders the instance (user prompt) for a command using the
	// runtime-specific escaping rules. cmd is one of plan/doing/easy/human-loop/
	// ctrl/dream/learning; params carries the command's per-invocation inputs.
	BuildPrompt(cmd string, params map[string]string) (string, error)
}
