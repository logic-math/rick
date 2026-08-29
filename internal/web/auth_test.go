package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// guarded builds a tiny pipeline: TokenAuth(token) wrapping a 200-OK handler.
func guarded(token string) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	return TokenAuth(token)(ok)
}

func doReq(t *testing.T, h http.Handler, headerToken, queryToken string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/config?token="+queryToken, nil)
	if headerToken != "" {
		req.Header.Set("Authorization", "Bearer "+headerToken)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestAuthNoTokenRejected：配置了 token 但请求不带 → 401 统一错误体。
func TestAuthNoTokenRejected(t *testing.T) {
	rec := doReq(t, guarded("secret"), "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code: want 401 got %d", rec.Code)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body unmarshal: %v (%s)", err, rec.Body.String())
	}
	if body.Error.Code != "unauthorized" {
		t.Fatalf("error code: %q", body.Error.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type: %q", rec.Header().Get("Content-Type"))
	}
}

// TestAuthBearerAccepted：Bearer header 通过（REST 通道）。
func TestAuthBearerAccepted(t *testing.T) {
	rec := doReq(t, guarded("secret"), "secret", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code: want 200 got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

// TestAuthQueryTokenAccepted：?token= 通过（SSE/EventSource 通道）。
func TestAuthQueryTokenAccepted(t *testing.T) {
	rec := doReq(t, guarded("secret"), "", "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("code: want 200 got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

// TestAuthWrongTokenRejected：错 token（header/query）均 401。
func TestAuthWrongTokenRejected(t *testing.T) {
	cases := []struct {
		name   string
		header string
		query  string
	}{
		{"wrong-header", "nope", ""},
		{"wrong-query", "", "nope"},
	}
	for _, c := range cases {
		rec := doReq(t, guarded("secret"), c.header, c.query)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: code want 401 got %d", c.name, rec.Code)
		}
	}
}

// TestAuthEmptyTokenDisablesAuth：空 token 配置=不鉴权（本地开发模式）。
func TestAuthEmptyTokenDisablesAuth(t *testing.T) {
	rec := doReq(t, guarded(""), "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code: want 200 got %d", rec.Code)
	}
}

// TestAuthHeaderWinsOverQuery：错 header 即使 query 对了也 401（显式凭据不被静默覆盖）。
func TestAuthHeaderWinsOverQuery(t *testing.T) {
	rec := doReq(t, guarded("secret"), "nope", "secret")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code: want 401 got %d（错误 Bearer 必须失败，即使 query token 正确）", rec.Code)
	}
}

// TestAuthNonBearerSchemeRejected：Authorization 非 Bearer 方案 → 401。
func TestAuthNonBearerSchemeRejected(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Basic c2VjcmV0")
	rec := httptest.NewRecorder()
	guarded("secret").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code: want 401 got %d", rec.Code)
	}
}

// TestAuthPostMethod：非 GET 请求同样走双通道校验（REST POST 带 query token 也应通过）。
func TestAuthPostMethod(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/workspaces?token=secret", nil)
	rec := httptest.NewRecorder()
	guarded("secret").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code: want 200 got %d", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/workspaces", nil)
	rec2 := httptest.NewRecorder()
	guarded("secret").ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("no-credential POST: want 401 got %d", rec2.Code)
	}
}
