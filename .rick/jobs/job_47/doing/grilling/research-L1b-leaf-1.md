# research-L1b-leaf-1：CLI 注册模式 + 配置系统（信源=代码直读，置信度高）

## A. CLI 子命令注册
- 框架=cobra（root.go:8 import spf13/cobra）。NewRootCmd(version)（root.go:46）内 rootCmd.AddCommand 注册 9 个子命令（root.go:67-75：plan/doing/learning/dream/tools/human-loop/ctrl/easy/web）。
- 全局 flag 用 PersistentFlags：verbose/dry-run/job/resume（root.go:57-60），包级 getter GetVerbose/GetDryRun/GetJobID/GetResume（root.go:90-118）供子命令读取。
- 新增子命令标准模式（easy.go 56L 全文）：internal/cmd/easy.go 定义 NewEasyCmd() *cobra.Command；局部 flag 用 cmd.Flags().StringVarP（easy.go:51-53）；RunE 内组 handler.Options{Verbose,DryRun,JobID}（easy.go:20-24）再调 handler.Easy/EasyDryRun/ResumeEasy（easy.go:27/47）——cmd 只做 flag 解析+分发。
- 组合根注入先例：doing.go:93-98 在 cmd 层 LoadConfig + runtime.NewPiRuntime(cfg.PiPath, cfg.PiExtraArgs...) 注入 handler.Doing(jobID, opts, rt)，handler 只依赖 Runtime 接口（DIP）。
- cmd/handler 分工：handler=调度聚合层，禁 import internal/cmd（options.go:1-9）；flag 值经 Options 显式透传（options.go:11-15）。
- v4.5 core 重构：完整逻辑落 xxxCore(rickDir,...)/XxxIn(rickDir,...)，路径全显式禁 cwd 函数；CLI 包装经 rickDirFromCwd()（core.go:31-35，全包唯一 cwd 解析点）；web 入口走 XxxIn（core.go:1-26）。
- web.go 是组合根例外（web.go:1-10 注释，rick-spec 例外三）：internal/web 装配后注入 handler.Web。

## B. 配置系统
- Config 结构（config.go:4-34）：MaxRetries/Runtime/PiPath/PiExtraArgs/DefaultWorkspace/Git{user_name,user_email}/HumanLoop{max_retries,research_source_weights,think_top_n,sense_max_backflows,think_min_assumptions}/WebToken/WebMaxActive。
- 加载：仅 ~/.rick/config.json 单一 JSON（loader.go:71-104），无 env/viper/flag 绑定；二进制 _dev 后缀→.rick_dev 目录（loader.go:22-26,42-46）。
- 优先级：文件不存在→自动写默认 GetDefaultConfig（loader.go:35-68）；Runtime 空归一 "pi"（loader.go:117-121，Unmarshal 不回填默认值）。
- 新增配置先例：嵌套结构 HumanLoopConfig（config.go:11,29-34）；顶层 omitempty 字段 WebToken/WebMaxActive（config.go:17-23，注释明示 json round-trip 对 omitempty 透明、loader 零改动）。
- config.example.json 仅示例（max_retries/pi_path/pi_extra_args/default_workspace/git），非 schema，未含 human_loop。

## 结论
1. rick super 照 easy.go 模式建 internal/cmd/super.go，root.go 加一行 AddCommand 即可。
2. super 配置最自然：Config 顶层加 SuperConfig（json:"super,omitempty"），仿 HumanLoopConfig+WebToken 先例。
3. cmd 层做组合根注入 runtime；handler 侧走 Core(rickDir) 显式路径，super 应复用同规。
