// Package handler 是 rick 四层架构第二层「调度聚合」。
//
// handler 接受 cli 路由与解析后的参数，编排 env → builder → runtime 完成 rick
// 命令的功能实现，并把 runtime 返回的 sessionID 持久化到 job 目录。它不 import
// internal/cmd（否则会形成跨包循环依赖）；cli 在调用点把 flag 值解析好，经
// Options 透传给 handler。
package handler

// Options carries flag values resolved by the CLI layer into handler
// orchestrators. Handlers must not import internal/cmd; the cmd layer passes
// these explicitly at the call site.
type Options struct {
	Verbose bool
	DryRun  bool
	JobID   string
}
