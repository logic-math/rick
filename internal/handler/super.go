// super.go — `rick super` 模式的 handler 编排（job_47/task11）。
//
// 职责边界（spec）：Go 侧只做编排与监督——工作区生命周期（init/super.json/干预
// 队列）、PySupervisor 进程监督（task10 包）、状态聚合与中控会话。**不实现任何
// 进化逻辑**（采样/门禁/档案/淘汰全部在 py 调度器）。
//
// 跨语言契约（super.json 是 Go↔py 的共享真相）：
//   - 顶层：state(idle|running|stopped|failed) / iteration_cursor / env_path /
//     pid / worker_pid / daemon / preverify_red / heartbeat_path / budget / config
//   - config 段与 py super/superopt/config.py DEFAULTS 同值（双语言一致性由
//     super_test.go 与 gate2/gate4 断言）
//   - preverify_red：start 时 Go 写 true，py 调度器启动先跑门先证红、失败即
//     退出非 0（task7 spec 的 --preverify-red 语义经由配置文件承载——task10
//     的 PySupervisor.Start 只支持固定 argv，无额外 flag 通道）
//
// 生命周期：SuperStart 建骨架 + super.json → spawn → 启动窗口（等首心跳，
// 调度器即退 = 门先证红失败）→ running → SIGTERM/SIGINT 或 SuperStop 优雅停
// → stopped；崩溃（首心跳后死亡）按 OnDead 重启（上限 MaxRestarts，超限 failed）。
package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sunquan/rick/internal/config"
	superpkg "github.com/sunquan/rick/internal/super"
	"github.com/sunquan/rick/internal/runtime"
)

// --- 配置与工作区解析 ---

// SuperStartConfig 汇总一次 start 的全部可注入参数（CLI 从 config+flags 组装，
// 测试直接构造短心跳节奏）。
type SuperStartConfig struct {
	// Root 是 super 工作区根目录（空 = DefaultSuperDir 解析）。
	Root string
	// HeartbeatInterval/HeartbeatTimeout/TermGrace/PythonPath 透传 PySupervisor。
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	TermGrace         time.Duration
	PythonPath        string
	// MaxRestarts 是首心跳后崩溃的自动重启上限（0 = 用默认 3）。
	MaxRestarts int
	// StartWindow 是启动窗口（等首心跳/捕获启动即退）；0 = timeout+2×interval。
	StartWindow time.Duration
}

// SuperStartOverrides 收集 start 的 CLI 覆盖项（0 值 = 不覆盖，落 DEFAULTS）。
type SuperStartOverrides struct {
	MaxWallTimeH   float64
	MaxIterations  int
	AdjustInterval int
}

// superWsRoot 解析 super 工作区根：cfg 覆盖 > ~/.rick/super。
func superWsRoot() string {
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Super.DefaultSuperDir != "" {
		return cfg.Super.DefaultSuperDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".rick", "super")
	}
	return filepath.Join(home, ".rick", "super")
}

// superWsPath 是 <root>/<name> 工作区路径。
func superWsPath(root, name string) string {
	return filepath.Join(root, name)
}

// resolveSuperStartConfig 组装 SuperStartConfig：显式值 > ~/.rick/config.json
// super 段 > DefaultSuperConfig。
func resolveSuperStartConfig() SuperStartConfig {
	def := config.DefaultSuperConfig()
	cfg := SuperStartConfig{
		HeartbeatInterval: time.Duration(def.HeartbeatIntervalMs) * time.Millisecond,
		HeartbeatTimeout:  time.Duration(def.HeartbeatTimeoutMs) * time.Millisecond,
		TermGrace:         time.Duration(def.TermGraceMs) * time.Millisecond,
		PythonPath:        def.PythonPath,
		MaxRestarts:       3,
	}
	if c, err := config.LoadConfig(); err == nil {
		if c.Super.HeartbeatIntervalMs > 0 {
			cfg.HeartbeatInterval = time.Duration(c.Super.HeartbeatIntervalMs) * time.Millisecond
		}
		if c.Super.HeartbeatTimeoutMs > 0 {
			cfg.HeartbeatTimeout = time.Duration(c.Super.HeartbeatTimeoutMs) * time.Millisecond
		}
		if c.Super.TermGraceMs > 0 {
			cfg.TermGrace = time.Duration(c.Super.TermGraceMs) * time.Millisecond
		}
		if c.Super.PythonPath != "" {
			cfg.PythonPath = c.Super.PythonPath
		}
	}
	return cfg
}

// --- super.json 读写（Go 侧权威写者：与 py 的 EventWriter 单写者无冲突——
// super.json 是运行元信息，不属事件流） ---

// superJSON 是 Go 侧读写的 super.json 视图（宽松：未知字段保留在 raw 里原样写回）。
type superJSON struct {
	State          string                 `json:"state"`
	IterationCursor int                   `json:"iteration_cursor"`
	EnvPath        string                 `json:"env_path"`
	Pid            int                    `json:"pid"`
	WorkerPid      int                    `json:"worker_pid"`
	Daemon         bool                   `json:"daemon"`
	PreverifyRed   bool                   `json:"preverify_red"`
	HeartbeatPath  string                 `json:"heartbeat_path"`
	Budget         map[string]interface{} `json:"budget"`
	Config         map[string]interface{} `json:"config"`
	UpdatedAt      string                 `json:"updated_at"`
}

// writeSuperJSONAtomic 原子写 super.json（tmp+rename，崩溃安全）。
func writeSuperJSONAtomic(ws string, sj *superJSON) error {
	sj.UpdatedAt = nowRFC3339Local()
	data, err := json.MarshalIndent(sj, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(ws, ".super.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(ws, "super.json"))
}

// readSuperJSON 读取 super.json（缺文件返回 error——调用方区分「从未 start」）。
func readSuperJSON(ws string) (*superJSON, error) {
	data, err := os.ReadFile(filepath.Join(ws, "super.json"))
	if err != nil {
		return nil, err
	}
	var sj superJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return nil, fmt.Errorf("super.json 解析失败: %w", err)
	}
	return &sj, nil
}

// nowRFC3339Local 与 py now_iso() 同型（带本地时区偏移，bugs.md 坑位）。
func nowRFC3339Local() string {
	return time.Now().Format(time.RFC3339)
}

// goSuperDefaults 是 py DEFAULTS 的 Go 侧镜像（写进 super.json config 段；
// 一致性断言见 super_test.go）。
func goSuperDefaults() map[string]interface{} {
	d := config.DefaultSuperConfig()
	return map[string]interface{}{
		"adjust_interval":     d.AdjustInterval,
		"debug_max_attempts":  d.DebugMaxAttempts,
		"soft_gate_k":         3,
		"heartbeat_interval_s": d.HeartbeatIntervalMs / 1000,
		"heartbeat_timeout_s": d.HeartbeatTimeoutMs / 1000,
		"term_grace_s":        d.TermGraceMs / 1000,
		"max_wall_time_h":     24.0,
		"max_iterations":      0,
		"python_path":         d.PythonPath,
		"pi_path":             "",
	}
}

// initSuperWorkspace 建目录骨架 + super.json（Go 直做，spec 裁决；与 py
// config.init_ws 幂等语义一致——已存在不删除、只补缺）。
//
// 骨架：interventions/ codes/ prompts/ sessions/ + events.jsonl（空） +
// archive.json（初始） + super.json。
func initSuperWorkspace(ws, envPath string, ov SuperStartOverrides) (*superJSON, error) {
	for _, d := range []string{"interventions", "codes", "prompts", "sessions"} {
		if err := os.MkdirAll(filepath.Join(ws, d), 0o755); err != nil {
			return nil, err
		}
	}
	eventsPath := filepath.Join(ws, "events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		if err := os.WriteFile(eventsPath, []byte(""), 0o644); err != nil {
			return nil, err
		}
	}
	archivePath := filepath.Join(ws, "archive.json")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		initArchive := `{"cells": {}, "pending": []}`
		if err := os.WriteFile(archivePath, []byte(initArchive), 0o644); err != nil {
			return nil, err
		}
	}

	sj, err := readSuperJSON(ws)
	if err != nil {
		sj = &superJSON{State: "idle"}
	}
	if sj.Config == nil {
		sj.Config = goSuperDefaults()
	}
	// CLI 覆盖（0 值不动）
	if ov.MaxWallTimeH > 0 {
		sj.Config["max_wall_time_h"] = ov.MaxWallTimeH
	}
	if ov.MaxIterations > 0 {
		sj.Config["max_iterations"] = ov.MaxIterations
	}
	if ov.AdjustInterval > 0 {
		sj.Config["adjust_interval"] = ov.AdjustInterval
	}
	if sj.Budget == nil {
		sj.Budget = map[string]interface{}{"wall_start": nil, "max_wall_time_h": sj.Config["max_wall_time_h"]}
	}
	if envPath != "" {
		if abs, err := filepath.Abs(envPath); err == nil {
			envPath = abs
		}
		sj.EnvPath = envPath
	}
	sj.HeartbeatPath = filepath.Join(ws, "heartbeat.json")
	sj.PreverifyRed = true // 启动契约：py 调度器启动先跑门先证红
	// init 落盘（幂等：已存在 super.json 时保留 state/cursor，仅补缺与覆盖项）
	if err := writeSuperJSONAtomic(ws, sj); err != nil {
		return nil, err
	}
	return sj, nil
}

// --- start ---

// SuperStart 是 CLI 入口：daemon 形态 re-exec 自身（--daemon-child 内部 flag，
// Setsid 脱离终端），前台形态直接跑监控循环。
func SuperStart(opts Options, name, env string, daemon bool, ov SuperStartOverrides) error {
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	root := superWsRoot()
	ws := superWsPath(root, name)

	if daemon {
		return superStartDaemon(name, env, ov)
	}

	cfg := resolveSuperStartConfig()
	stopCh := signalHandler()
	return SuperStartMonitored(cfg, name, ws, env, ov, stopCh)
}

// superStartDaemon re-exec 自身跑 daemon-child（web.go --daemon 先例：Setsid +
// 日志文件 + RICK_SUPER_DAEMON=1；daemon-child 经环境变量识别自身形态）。
func superStartDaemon(name, env string, ov SuperStartOverrides) error {
	root := superWsRoot()
	ws := superWsPath(root, name)
	if err := os.MkdirAll(ws, 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(ws, "super.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open super log: %w", err)
	}
	defer f.Close()

	self, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"super", "start", "--name", name}
	if env != "" {
		args = append(args, "--env", env)
	}
	if ov.MaxWallTimeH > 0 {
		args = append(args, "--max-wall-time", fmt.Sprintf("%g", ov.MaxWallTimeH))
	}
	if ov.MaxIterations > 0 {
		args = append(args, "--max-iterations", strconv.Itoa(ov.MaxIterations))
	}
	if ov.AdjustInterval > 0 {
		args = append(args, "--adjust-interval", strconv.Itoa(ov.AdjustInterval))
	}
	child := exec.Command(self, args...)
	child.Env = append(os.Environ(), "RICK_SUPER_DAEMON=1")
	child.Stdin = nil
	child.Stdout = f
	child.Stderr = f
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		return fmt.Errorf("start super daemon: %w", err)
	}
	fmt.Printf("[rick super] daemon started: name=%s pid=%d log=%s\n", name, child.Process.Pid, logPath)
	fmt.Printf("[rick super] stop: rick super stop --name %s\n", name)
	return nil
}

// signalHandler 返回一个在 SIGTERM/SIGINT 时 close 的通道（优雅停信号源）。
func signalHandler() chan struct{} {
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		close(stopCh)
	}()
	return stopCh
}

// SuperStartMonitored 是前台/daemon-child 共享的监控循环：
// init → spawn → 启动窗口（首心跳 or 启动即退=门先证红失败）→ running →
// 等待（stop 信号 / worker 死亡重启 / 重启超限）→ 收尾落 state。
//
// 导出供测试注入短心跳节奏与 stopCh（CLI 两条路径都收敛到这里）。
func SuperStartMonitored(cfg SuperStartConfig, name, ws, envPath string, ov SuperStartOverrides, stopCh chan struct{}) error {
	if cfg.HeartbeatInterval <= 0 || cfg.HeartbeatTimeout <= 0 || cfg.TermGrace <= 0 {
		return fmt.Errorf("super: invalid supervisor cadence (interval=%v timeout=%v grace=%v)",
			cfg.HeartbeatInterval, cfg.HeartbeatTimeout, cfg.TermGrace)
	}
	if cfg.PythonPath == "" {
		cfg.PythonPath = "python3"
	}
	maxRestarts := cfg.MaxRestarts
	if maxRestarts <= 0 {
		maxRestarts = 3
	}
	startWindow := cfg.StartWindow
	if startWindow <= 0 {
		startWindow = cfg.HeartbeatTimeout + 2*cfg.HeartbeatInterval
	}

	// init（骨架 + super.json；daemon-child 形态经 RICK_SUPER_DAEMON 识别并落盘）
	sj, err := initSuperWorkspace(ws, envPath, ov)
	if err != nil {
		return fmt.Errorf("init super workspace: %w", err)
	}
	if os.Getenv("RICK_SUPER_DAEMON") == "1" {
		sj.Daemon = true
	}

	// supervisor + worker：OnDead 只处理「首心跳后死亡」的重启；首心跳前的死亡
	// 留给启动窗口逻辑（门先证红失败路径）。优雅停（shutdown 置位）不重启。
	// 自然完成（exit 0，如迭代上限跑满）不重启——走 completed 收尾。
	var mu sync.Mutex
	shutdown := false
	restartCh := make(chan struct{}, maxRestarts+1)
	srvDead := make(chan struct{}, 1)
	doneCh := make(chan struct{}, 1)
	var workerRef *superpkg.PyWorker

	supCfg := superpkg.PySupervisorConfig{
		HeartbeatInterval: cfg.HeartbeatInterval,
		HeartbeatTimeout:  cfg.HeartbeatTimeout,
		TermGrace:         cfg.TermGrace,
		PythonPath:        cfg.PythonPath,
		OnDead: func(_ string, lastSeq int64) {
			mu.Lock()
			isShutdown := shutdown
			w := workerRef
			mu.Unlock()
			if isShutdown {
				return // 优雅停路径：不重启
			}
			if w != nil && w.LastHeartbeat() == nil {
				// 首心跳前死亡：启动窗口逻辑接管（门先证红失败路径）
				select {
				case srvDead <- struct{}{}:
				default:
				}
				return
			}
			// 自然完成（exit 0：如 max_iterations 跑满/预算墙到点自然收尾）
			// ≠ 崩溃：不重启，走 completed 收尾（重启只会空转——py 重新拉起后发现
			// 无迭代可做立即再退，烧完重启配额后误标 failed；job_47 首跑实测）。
			if w != nil && w.Reason() == "exited: 0" {
				select {
				case doneCh <- struct{}{}:
				default:
				}
				return
			}
			select {
			case restartCh <- struct{}{}:
			default:
			}
		},
	}
	srv := superpkg.NewPySupervisor(supCfg)
	defer srv.StopAll()

	worker, err := srv.Start(ws)
	if err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}
	mu.Lock()
	workerRef = worker
	mu.Unlock()

	// pid 落盘 + state=starting（首心跳通过后转 running）
	sj.State = "starting"
	sj.Pid = os.Getpid()
	sj.WorkerPid = worker.Pid()
	if err := writeSuperJSONAtomic(ws, sj); err != nil {
		_ = worker.Close()
		return err
	}

	// --- 启动窗口：首心跳（转 running）或启动即退（门先证红失败） ---
	window := time.NewTimer(startWindow)
	defer window.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-window.C:
			_ = worker.Close()
			return fmt.Errorf("super: 启动窗口内未见首心跳（%v）——调度器未起活；stderr 尾部：%s",
				startWindow, tailOf(worker.StderrTail(400)))
		case <-srvDead:
			_ = worker.Close()
			sj.State = "failed"
			_ = writeSuperJSONAtomic(ws, sj)
			return fmt.Errorf("super: 门先证红失败——调度器启动即退出（%s）；stderr 尾部：%s",
				worker.Reason(), tailOf(worker.StderrTail(400)))
		case <-tick.C:
			if worker.LastHeartbeat() != nil {
				sj.State = "running"
				if err := writeSuperJSONAtomic(ws, sj); err != nil {
					_ = worker.Close()
					return err
				}
				goto Running
			}
			if worker.IsDead() {
				// OnDead 竞态兜底：直接按启动失败处理
				_ = worker.Close()
				sj.State = "failed"
				_ = writeSuperJSONAtomic(ws, sj)
				return fmt.Errorf("super: 门先证红失败——调度器启动即退出（%s）；stderr 尾部：%s",
					worker.Reason(), tailOf(worker.StderrTail(400)))
			}
		}
	}

	// --- running：等 stop / 自然完成 / 崩溃重启 / 重启超限 ---
Running:
	restarts := 0
	for {
		select {
		case <-doneCh:
			// 调度器自然完成（exit 0）：不是失败，置 completed 收尾
			mu.Lock()
			shutdown = true
			mu.Unlock()
			sj.State = "completed"
			_ = writeSuperJSONAtomic(ws, sj)
			return nil
		case <-stopCh:
			mu.Lock()
			shutdown = true
			mu.Unlock()
			sj.State = "stopped"
			_ = writeSuperJSONAtomic(ws, sj)
			_ = worker.Close()
			return nil
		case <-restartCh:
			restarts++
			if restarts > maxRestarts {
				mu.Lock()
				shutdown = true
				mu.Unlock()
				sj.State = "failed"
				_ = writeSuperJSONAtomic(ws, sj)
				return fmt.Errorf("super: 调度器崩溃重启超限（%d 次）——state=failed；最后死因：%s stderr：%s",
					maxRestarts, worker.Reason(), tailOf(worker.StderrTail(400)))
			}
			fmt.Fprintf(os.Stderr, "[rick super] worker died (%s), restarting %d/%d\n",
				worker.Reason(), restarts, maxRestarts)
			w2, err := srv.Start(ws)
			if err != nil {
				sj.State = "failed"
				_ = writeSuperJSONAtomic(ws, sj)
				return fmt.Errorf("super: 重启失败: %w", err)
			}
			mu.Lock()
			workerRef = w2
			mu.Unlock()
			worker = w2
			sj.WorkerPid = w2.Pid()
			_ = writeSuperJSONAtomic(ws, sj)
		}
	}
}

// tailOf 截断尾部用于错误信息（空安全）。
func tailOf(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(空)"
	}
	return s
}

// --- stop ---

// SuperStop 优雅停：读 super.json.pid → SIGTERM（daemon 形态 kill 进程组）→
// 等退出（grace 预算）→ SIGKILL 兜底 → state=stopped。
func SuperStop(opts Options, name string) error {
	root := superWsRoot()
	return SuperStopIn(superWsPath(root, name))
}

// SuperStopIn 是显式 ws 变体（测试入口）。
func SuperStopIn(ws string) error {
	sj, err := readSuperJSON(ws)
	if err != nil {
		return fmt.Errorf("super workspace 未初始化或不可读: %w", err)
	}
	if sj.State == "stopped" {
		fmt.Printf("[rick super] already stopped\n")
		return nil
	}
	pid := sj.Pid
	if pid <= 0 {
		sj.State = "stopped"
		_ = writeSuperJSONAtomic(ws, sj)
		fmt.Printf("[rick super] no pid recorded; state=stopped\n")
		return nil
	}
	if pid == os.Getpid() {
		return fmt.Errorf("super: 前台会话归本进程所有——在其终端 Ctrl+C 停止（或用 --daemon 形态才能跨进程 stop）")
	}
	if !pidAliveHandler(pid) {
		sj.State = "stopped"
		_ = writeSuperJSONAtomic(ws, sj)
		fmt.Printf("[rick super] pid %d not alive; state=stopped\n", pid)
		return nil
	}

	// daemon 形态：进程组收割（setsid 后 pid==pgid）；前台外部形态：单点 TERM
	if sj.Daemon {
		_ = syscall.Kill(-pid, syscall.SIGTERM)
	} else {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	// 等优雅退出（TERM grace 预算 + 余量）
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !pidAliveHandler(pid) {
			sj.State = "stopped"
			_ = writeSuperJSONAtomic(ws, sj)
			fmt.Printf("[rick super] stopped (pid %d)\n", pid)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 兜底强杀
	if sj.Daemon {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	} else {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	time.Sleep(200 * time.Millisecond)
	sj.State = "stopped"
	_ = writeSuperJSONAtomic(ws, sj)
	fmt.Printf("[rick super] force-stopped (pid %d)\n", pid)
	return nil
}

func pidAliveHandler(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// --- status ---

// SuperStatus 人类可读输出。
func SuperStatus(opts Options, name string, jsonOut bool) error {
	root := superWsRoot()
	ws := superWsPath(root, name)
	if jsonOut {
		var b strings.Builder
		if err := SuperStatusJSONIn(ws, &b); err != nil {
			return err
		}
		fmt.Println(b.String())
		return nil
	}
	return SuperStatusIn(ws, false)
}

// SuperStatusIn 人类可读状态输出（ws 显式变体）。
func SuperStatusIn(ws string, _ bool) error {
	st, err := collectSuperStatus(ws)
	if err != nil {
		return err
	}
	fmt.Printf("workspace : %s\n", st.Workspace)
	fmt.Printf("state     : %s (recorded)  live_pid=%v\n", st.State, st.PidAlive)
	if st.HeartbeatSeq != nil {
		fmt.Printf("heartbeat : seq=%d state=%s age=%s\n", *st.HeartbeatSeq, st.HeartbeatState, st.HeartbeatAge)
	} else {
		fmt.Printf("heartbeat : (none yet)\n")
	}
	fmt.Printf("cursor    : last_seq=%d iterations=%d\n", st.LastSeq, st.Iterations)
	fmt.Printf("north star: elite_admits=%d cells=%d total_tokens=%.0f\n",
		st.EliteAdmits, st.Cells, st.TotalTokens)
	if st.EnvPath != "" {
		fmt.Printf("env       : %s\n", st.EnvPath)
	}
	if len(st.RecentEvents) > 0 {
		fmt.Printf("recent    :\n")
		for _, e := range st.RecentEvents {
			fmt.Printf("  - %s\n", e)
		}
	}
	return nil
}

// SuperStatusJSONIn 机器可读输出（--json）。
func SuperStatusJSONIn(ws string, w *strings.Builder) error {
	st, err := collectSuperStatus(ws)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}

// superStatus 是聚合视图（json tag 对齐测试断言）。
type superStatus struct {
	Workspace      string   `json:"workspace"`
	State          string   `json:"state"`
	Pid            int      `json:"pid"`
	PidAlive       bool     `json:"pid_alive"`
	EnvPath        string   `json:"env_path,omitempty"`
	LastSeq        int64    `json:"last_seq"`
	Iterations     int      `json:"iterations"`
	EliteAdmits    int      `json:"elite_admits"`
	EliteReplaces  int      `json:"elite_replaces"`
	Cells          int      `json:"cells"`
	TotalTokens    float64  `json:"total_tokens"`
	HeartbeatSeq   *int64   `json:"heartbeat_seq"`
	HeartbeatState string   `json:"heartbeat_state,omitempty"`
	HeartbeatAge   string   `json:"heartbeat_age,omitempty"`
	RecentEvents   []string `json:"recent_events,omitempty"`
}

// collectSuperStatus 聚合 super.json + heartbeat.json + events.jsonl 尾部。
func collectSuperStatus(ws string) (*superStatus, error) {
	st := &superStatus{Workspace: ws}
	sj, err := readSuperJSON(ws)
	if err != nil {
		return nil, fmt.Errorf("super workspace 未初始化或不可读（先 rick super start）: %w", err)
	}
	st.State = sj.State
	st.Pid = sj.Pid
	st.EnvPath = sj.EnvPath
	st.PidAlive = sj.Pid > 0 && pidAliveHandler(sj.Pid)

	// heartbeat.json（宽松解析；半写容错）
	if data, err := os.ReadFile(filepath.Join(ws, "heartbeat.json")); err == nil {
		var hb struct {
			Seq   int64  `json:"seq"`
			State string `json:"state"`
			Ts    string `json:"ts"`
		}
		if json.Unmarshal(data, &hb) == nil {
			seq := hb.Seq
			st.HeartbeatSeq = &seq
			st.HeartbeatState = hb.State
			if t, err := time.Parse(time.RFC3339, hb.Ts); err == nil {
				st.HeartbeatAge = time.Since(t).Round(time.Second).String()
			}
		}
	}

	// events.jsonl 全量轻扫（append-only 行流，逐行 parse；大文件按尾部窗口）
	agg, recent, err := aggregateEvents(filepath.Join(ws, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	st.LastSeq = agg.lastSeq
	st.Iterations = agg.iterations
	st.EliteAdmits = agg.eliteAdmits
	st.EliteReplaces = agg.eliteReplaces
	st.Cells = len(agg.cells)
	st.TotalTokens = agg.totalTokens
	st.RecentEvents = recent
	return st, nil
}

// eventsAgg 是 events.jsonl 的聚合结果。
type eventsAgg struct {
	lastSeq      int64
	iterations   int
	eliteAdmits  int
	eliteReplaces int
	cells        map[string]bool
	totalTokens  float64
}

// aggregateEvents 逐行解析 events.jsonl（信封对齐 super/superopt/events.py：
// {ts,seq,type,source,payload}）；坏行/半行跳过（读取端容错契约）。
// 返回聚合 + 最近 5 条摘要（"seq type source"）。
func aggregateEvents(path string) (*eventsAgg, []string, error) {
	agg := &eventsAgg{cells: map[string]bool{}}
	var recents []string

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return agg, nil, nil
		}
		return nil, nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for dec.More() {
		var ev struct {
			Seq     int64           `json:"seq"`
			Type    string          `json:"type"`
			Source  string          `json:"source"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := dec.Decode(&ev); err != nil {
			// 坏行/半行：jsonl 容错契约——跳过到下一行
			continue
		}
		if ev.Seq > agg.lastSeq {
			agg.lastSeq = ev.Seq
		}
		switch ev.Type {
		case "iteration_started":
			agg.iterations++
		case "elite_admitted":
			agg.eliteAdmits++
			agg.noteCell(ev.Payload, agg.cells)
		case "elite_replaced":
			agg.eliteReplaces++
			agg.noteCell(ev.Payload, agg.cells)
		case "agent_ended":
			var p struct {
				TotalTokens float64 `json:"total_tokens"`
				Usage       *struct {
					Total float64 `json:"total"`
				} `json:"usage"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil {
				if p.TotalTokens > 0 {
					agg.totalTokens += p.TotalTokens
				} else if p.Usage != nil {
					agg.totalTokens += p.Usage.Total
				}
			}
		}
		recents = append(recents, fmt.Sprintf("#%d %s %s", ev.Seq, ev.Type, ev.Source))
	}
	if len(recents) > 5 {
		recents = recents[len(recents)-5:]
	}
	return agg, recents, nil
}

// noteCell 从 elite_* 载荷提取 cell_id 记入集合。
func (a *eventsAgg) noteCell(payload json.RawMessage, cells map[string]bool) {
	var p struct {
		CellID string `json:"cell_id"`
	}
	if json.Unmarshal(payload, &p) == nil && p.CellID != "" {
		cells[p.CellID] = true
	}
}

// --- event（干预队列） ---

// SuperEvent 写干预队列（单写者纪律：human 干预一律经队列，不直写 events.jsonl）。
func SuperEvent(opts Options, name, etype, payload string) error {
	root := superWsRoot()
	return SuperEventIn(superWsPath(root, name), etype, payload)
}

// SuperEventIn 是显式 ws 变体：interventions/<ts>-<rand>.json。
func SuperEventIn(ws, etype, payload string) error {
	if etype == "" {
		return fmt.Errorf("--type is required")
	}
	if payload == "" {
		payload = "{}"
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return fmt.Errorf("--payload 必须是合法 JSON: %w", err)
	}
	qdir := filepath.Join(ws, "interventions")
	if err := os.MkdirAll(qdir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%06d.json", time.Now().Format("20060102T150405"), time.Now().Nanosecond()%1000000)
	entry := map[string]interface{}{
		"type":       etype,
		"payload":    raw,
		"created_at": nowRFC3339Local(),
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(qdir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("[rick super] intervention queued: %s\n", path)
	return nil
}

// --- session（中控会话） ---

// SuperSession 打开 super 中控交互会话（照 easy：BuildSuperPrompt → pi 交互）。
// resume=true 时加载该 ws 最近一次会话（pi --session 加载语义，bugs.md）。
func SuperSession(opts Options, name string, resume bool) error {
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	root := superWsRoot()
	ws := superWsPath(root, name)

	rickDir, err := rickDirFromCwd()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	var mainFile, sessionID string
	if resume {
		sid, pfile, err := latestSuperSession(ws)
		if err != nil {
			return fmt.Errorf("resume super session: %w", err)
		}
		sessionID, mainFile = sid, pfile
	} else {
		_, pfile, err := superpkg.BuildSuperPrompt(rickDir, ws)
		if err != nil {
			return fmt.Errorf("build super prompt: %w", err)
		}
		sid, _, err := latestSuperSession(ws) // 刚建的即最新
		if err != nil {
			return fmt.Errorf("read back session id: %w", err)
		}
		sessionID, mainFile = sid, pfile
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Workspace : %s\n", ws)
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Println("🤖 Starting super control session...")

	if resume {
		// bugs.md：--session 是加载已有会话语义（找不到报 No session found）
		if err := runtime.CallCLI(opts.Verbose, cfg, "", runtime.ModeInteractive, "--session", sessionID); err != nil {
			return fmt.Errorf("super session resume failed: %w", err)
		}
		return nil
	}
	if err := runtime.CallCLI(opts.Verbose, cfg, mainFile, runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("super session failed: %w", err)
	}
	return nil
}

// latestSuperSession 返回 ws 下最新会话目录的 (session_id, prompt.md)。
func latestSuperSession(ws string) (string, string, error) {
	sessRoot := filepath.Join(ws, "sessions")
	entries, err := os.ReadDir(sessRoot)
	if err != nil {
		return "", "", fmt.Errorf("no sessions dir in workspace: %w", err)
	}
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if latest == "" || info.ModTime().After(latestMod) {
			latest = e.Name()
			latestMod = info.ModTime()
		}
	}
	if latest == "" {
		return "", "", fmt.Errorf("workspace has no sessions yet")
	}
	sidData, err := os.ReadFile(filepath.Join(sessRoot, latest, "session_id"))
	if err != nil {
		return "", "", fmt.Errorf("session_id missing in %s: %w", latest, err)
	}
	sid := strings.TrimSpace(string(sidData))
	prompt := filepath.Join(sessRoot, latest, "prompt.md")
	if _, err := os.Stat(prompt); err != nil {
		return "", "", fmt.Errorf("prompt.md missing in %s: %w", latest, err)
	}
	return sid, prompt, nil
}
