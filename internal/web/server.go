// server.go 组装 rick web 的 http.Server 生命周期（task13）：
//
//	NewServer(cfg, deps) → *Server{http.Server}
//	Server.Serve(ctx)    → 阻塞服务；ctx cancel → Shutdown(10s) 优雅关闭
//
// 启动打印（stdout，单用户本地工具的人类接口）：监听地址、token 提示、
// 工作区数。task14 的 handler.Web 负责 pid/singleton/config token 编排；
// 本文件只关心 HTTP 生命周期与路由组装。
package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"time"
)

// ServerConfig carries everything NewServer needs to assemble the service.
type ServerConfig struct {
	// Addr is the listen address (host:port; default 127.0.0.1:6137).
	Addr string
	// Token is the single-user auth token ("" disables auth).
	Token string
	// WebFS is the embedded baseline dist view (webassets.DistWeb()).
	WebFS fs.FS
	// StateDir is the machine-level web state root (~/.rick/web) — the
	// overlay source for the static handler. May not exist yet (H4).
	StateDir string
	// Version is the rick binary version (→ /api/config rick_version).
	Version string
}

// DefaultWebPort is the default listen port (C-137 easter egg, design-tree J4.2).
const DefaultWebPort = 6137

// DefaultWebAddr is the default listen address (loopback-only by default;
// opening to the network is a deliberate --listen flag, design-tree J2.4).
const DefaultWebAddr = "127.0.0.1:6137"

// shutdownGrace bounds graceful shutdown (in-flight SSE streams included).
const shutdownGrace = 10 * time.Second

// Server wraps the assembled http.Server with rick web's lifecycle helpers.
type Server struct {
	httpServer *http.Server
	cfg        ServerConfig
	deps       Deps
}

// NewServer assembles the mux (RegisterRoutes), the static handler
// (overlay priority over embed), and the http.Server. deps supplies the
// session manager/hub/registries; deps.Version is overridden by cfg.Version
// when non-empty (single source: cfg wins — task14 passes one value).
func NewServer(cfg ServerConfig, deps Deps) (*Server, error) {
	if deps.Hub == nil {
		deps.Hub = NewHub(0)
	}
	if cfg.Version != "" {
		deps.Version = cfg.Version
	}
	if deps.Static == nil {
		overlayDist := ""
		if cfg.StateDir != "" {
			overlayDist = joinPath(cfg.StateDir, "dist")
		}
		if cfg.WebFS == nil {
			return nil, errors.New("web: NewServer needs WebFS (embed baseline) or a Static handler")
		}
		deps.Static = StaticHandler(cfg.WebFS, overlayDist)
	}

	addr := cfg.Addr
	if addr == "" {
		addr = DefaultWebAddr
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, deps)

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 30 * time.Second,
		},
		cfg:  cfg,
		deps: deps,
	}, nil
}

// Serve starts listening and blocks until ctx is cancelled or the listener
// fails. Startup prints go to stdout (listening address, token mode,
// workspace count). On ctx cancellation the server shuts down gracefully
// (up to shutdownGrace), draining in-flight requests and SSE streams.
func (s *Server) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.httpServer.Addr, err)
	}

	// Startup banner (human interface, single-user local tool).
	addr := ln.Addr().String()
	fmt.Printf("[rick-web] listening on http://%s\n", addr)
	if s.cfg.Token != "" {
		fmt.Printf("[rick-web] auth: token required (Authorization: Bearer or ?token=)\n")
	} else {
		fmt.Printf("[rick-web] auth: DISABLED (empty token — local development mode)\n")
	}
	if ws := s.deps.Workspaces; ws != nil {
		fmt.Printf("[rick-web] workspaces: %d registered\n", len(ws.List()))
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// 先收掉 pi worker（子进程自成进程组，父进程退出不会连带杀死）——否则会留下
	// 孤儿 pi 进程占着会话文件，下次启动 resume 时「新 worker 起来就死」。
	// 放在 HTTP drain 之前：worker 的收尸有 TermGrace 预算，不能和 SSE drain 抢时间。
	if s.deps.Sessions != nil {
		s.deps.Sessions.ShutdownWorkers()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		// Graceful drain failed (SSE streams stuck): force close.
		_ = s.httpServer.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return <-errCh
}

// StartWatchers wires the polling watchers to this server's hub and state
// dir (frontend overlay + registered workspaces' tasks.json). Convenience
// for the composition root (task14): watchers must run for the same ctx
// lifetime as Serve.
func (s *Server) StartWatchers(ctx context.Context) {
	overlayDist := ""
	if s.cfg.StateDir != "" {
		overlayDist = joinPath(s.cfg.StateDir, "dist")
	}
	StartWatchers(ctx, overlayDist, s.deps.Hub, s.deps.Workspaces)
}

// joinPath is filepath.Join for two segments with empty-tolerance.
func joinPath(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + string(os.PathSeparator) + b
}
