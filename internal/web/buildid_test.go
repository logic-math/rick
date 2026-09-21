package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// buildid_test.go —— task18 层验收：构建指纹（build_id）贯穿 /api/health
// （免认证探针）、/api/config 与 SSE server_info。
//
// 为什么这些断言重要：/api/config 的 rick_version 是静态常量，同版本的两次构建
// 无法区分；而自进化闭环（dev-web up / tools release）必须能回答「现在跑的是不是
// 我刚构建的那个二进制」。一条 curl 就能判定的前提是：指纹出现在**免认证**的
// /api/health 上，且 status 字段语义不变（旧前端与既有 E2E 只读 status）。

// withBuildID 设置进程级指纹并在测试结束恢复（包级变量——避免测试间串味）。
func withBuildID(t *testing.T, id string) {
	t.Helper()
	prev := BuildID()
	SetBuildID(id)
	t.Cleanup(func() { SetBuildID(prev) })
}

// TestHealthEndpointCarriesBuildID 验证 /api/health：
//   - 免认证（配置了 token 也不要求 Bearer）
//   - 携带 build_id（与注入值一致）与 started_at
//   - status 仍为 "ok"（兼容红线）
func TestHealthEndpointCarriesBuildID(t *testing.T) {
	withBuildID(t, "abc1234-20260101120000")

	mux := http.NewServeMux()
	RegisterRoutes(mux, Deps{Token: "secret-token", Version: "9.9.9"})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 不带任何 Authorization —— 探针必须免认证
	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200（免认证探针）", resp.StatusCode)
	}
	var got struct {
		Status    string `json:"status"`
		BuildID   string `json:"build_id"`
		StartedAt string `json:"started_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if got.Status != "ok" {
		t.Fatalf("status field = %q, want \"ok\"（兼容红线：旧前端只读它）", got.Status)
	}
	if got.BuildID != "abc1234-20260101120000" {
		t.Fatalf("build_id = %q, want the injected fingerprint", got.BuildID)
	}
	if got.StartedAt == "" {
		t.Fatal("started_at 缺失（NewServer 装配时应写入）")
	}
	if _, err := time.Parse(time.RFC3339, got.StartedAt); err != nil {
		t.Fatalf("started_at 非 RFC3339: %q (%v)", got.StartedAt, err)
	}
}

// TestBuildIDFallbackToDev 验证未注入时回退 "dev"，且空/纯空白都归一化——
// 让调用方不必特判（dev-web/release 的指纹比对依赖「永不空」）。
func TestBuildIDFallbackToDev(t *testing.T) {
	withBuildID(t, "")

	if got := BuildID(); got != "dev" {
		t.Fatalf("BuildID() = %q, want \"dev\"", got)
	}
	SetBuildID("   ")
	if got := BuildID(); got != "dev" {
		t.Fatalf("whitespace BuildID() = %q, want \"dev\"", got)
	}
	SetBuildID("sha7-20260101120000")
	if got := BuildID(); got != "sha7-20260101120000" {
		t.Fatalf("BuildID() = %q, want the injected value", got)
	}

	// 手写 Deps（无 BuildID）也必须回退到进程级值，而不是空串
	withBuildID(t, "proc-level")
	deps := Deps{Version: "1.0.0"}
	if got := depsBuildID(deps); got != "proc-level" {
		t.Fatalf("depsBuildID(fallback) = %q, want \"proc-level\"", got)
	}
	// 显式注入优先（测试/多实例场景）
	if got := depsBuildID(Deps{BuildID: "cfg-level"}); got != "cfg-level" {
		t.Fatalf("depsBuildID(explicit) = %q, want \"cfg-level\"", got)
	}
}

// TestConfigCarriesBuildID 验证 /api/config 带 build_id，且仍受认证保护
// （只在新增字段，不改变鉴权与既有字段语义）。
func TestConfigCarriesBuildID(t *testing.T) {
	withBuildID(t, "cfg-20260101120000")

	mux := http.NewServeMux()
	RegisterRoutes(mux, Deps{Token: "tok", Version: "4.4.15"})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	request := func(withToken bool) (*http.Response, map[string]any) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/config", nil)
		if withToken {
			req.Header.Set("Authorization", "Bearer tok")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /api/config: %v", err)
		}
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		return resp, body
	}

	if resp, _ := request(false); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no-token status = %d, want 401（config 仍需认证）", resp.StatusCode)
	}
	resp, body := request(true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if body["build_id"] != "cfg-20260101120000" {
		t.Fatalf("config build_id = %v, want the injected fingerprint", body["build_id"])
	}
	// 既有字段语义不变
	if body["rick_version"] != "4.4.15" {
		t.Fatalf("rick_version = %v, want 4.4.15（契约回归）", body["rick_version"])
	}
	if body["auth_required"] != true {
		t.Fatalf("auth_required = %v, want true", body["auth_required"])
	}
}

// TestServerInfoCarriesBuildID 验证 SSE 首帧 server_info 携带 build_id，
// 且既有字段（version/rick_version/time）保持（前端与既有测试依赖它们）。
func TestServerInfoCarriesBuildID(t *testing.T) {
	withBuildID(t, "sse-20260101120000")

	env := ServerInfoEvent(1, "4.4.15")
	if env.Type != EventTypeServerInfo {
		t.Fatalf("type = %q, want %q", env.Type, EventTypeServerInfo)
	}
	var payload map[string]any
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatalf("server_info data: %v", err)
	}
	if payload["build_id"] != "sse-20260101120000" {
		t.Fatalf("server_info build_id = %v, want the injected fingerprint", payload["build_id"])
	}
	if payload["rick_version"] != "4.4.15" {
		t.Fatalf("rick_version = %v（既有字段语义不得变）", payload["rick_version"])
	}
	if payload["version"] != float64(1) {
		t.Fatalf("version = %v, want 1", payload["version"])
	}
	if _, ok := payload["time"]; !ok {
		t.Fatal("server_info 缺 time 字段（既有契约）")
	}

	// 未注入时回退 dev（而非缺字段）
	withBuildID(t, "")
	env = ServerInfoEvent(1, "4.4.15")
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatalf("server_info data: %v", err)
	}
	if payload["build_id"] != "dev" {
		t.Fatalf("fallback build_id = %v, want \"dev\"", payload["build_id"])
	}
}

// TestNewServerWiresBuildIDAndStartedAt 验证装配层把指纹与启动时间注入 Deps：
// cfg.BuildID 优先，缺省用进程级值。
func TestNewServerWiresBuildIDAndStartedAt(t *testing.T) {
	withBuildID(t, "proc-1")

	deps := Deps{Static: http.NotFoundHandler(), Version: "1.2.3"}
	srv, err := NewServer(ServerConfig{Addr: "127.0.0.1:0", BuildID: "cfg-1"}, deps)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if srv.deps.BuildID != "cfg-1" {
		t.Fatalf("deps.BuildID = %q, want cfg-1（cfg 优先）", srv.deps.BuildID)
	}
	if srv.deps.StartedAt.IsZero() {
		t.Fatal("deps.StartedAt 未设置")
	}

	// cfg 不带 build id → 用进程级值（组合根注入的路径）
	srv2, err := NewServer(ServerConfig{Addr: "127.0.0.1:0"}, Deps{Static: http.NotFoundHandler()})
	if err != nil {
		t.Fatalf("NewServer(no cfg build id): %v", err)
	}
	if srv2.deps.BuildID != "proc-1" {
		t.Fatalf("deps.BuildID = %q, want proc-1（进程级回退）", srv2.deps.BuildID)
	}
}

// TestStaticOnlyMuxHealthStillWorks 验证最小嵌入面（StaticOnlyMux）的 health
// 仍可用（改签名为带 Deps 后不得破坏静态专用场景）。
func TestStaticOnlyMuxHealthStillWorks(t *testing.T) {
	withBuildID(t, "static-1")

	mux := StaticOnlyMux(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ok</html>")},
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("static-only health = %d, want 200", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["status"] != "ok" || got["build_id"] != "static-1" {
		t.Fatalf("static-only health body = %v", got)
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") {
		t.Fatal("health 不应返回 HTML")
	}
}
