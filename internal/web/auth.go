package web

import (
	"net/http"
	"strings"
)

// unauthorizedBody is the contract error body for a failed token check
// (api-contract.md 通用: 401 {"error":{"code":"unauthorized","message":"..."}}).
func unauthorizedBody(hint string) string {
	if hint == "" {
		hint = "missing or invalid token"
	}
	return `{"error":{"code":"unauthorized","message":"` + hint + `"}}`
}

// writeUnauthorized emits the 401 error body with the right content type.
func writeUnauthorized(w http.ResponseWriter, hint string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(unauthorizedBody(hint)))
}

// TokenAuth returns an HTTP middleware enforcing the single-user token:
//
//   - `Authorization: Bearer <token>` header (REST clients)
//   - `?token=<token>` query parameter (EventSource/SSE and any GET that
//     cannot set headers)
//
// An empty configured token disables authentication entirely (local
// development mode — loopback default makes this a deliberate convenience,
// not a security hole the operator did not opt into).
//
// Failure responses use the contract error body with status 401.
func TokenAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			if tokenMatches(r, token) {
				next.ServeHTTP(w, r)
				return
			}

			writeUnauthorized(w, "missing or invalid token (use Authorization: Bearer or ?token=)")
		})
	}
}

// tokenMatches checks both credential channels. The Bearer header wins when
// present and non-empty (so a wrong header is a failure even if the query
// param happens to be right — explicit credentials must not be silently
// overridden); the query parameter is the fallback channel.
func tokenMatches(r *http.Request, token string) bool {
	if h := r.Header.Get("Authorization"); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return false
		}
		return strings.TrimSpace(parts[1]) == token
	}
	if q := r.URL.Query().Get("token"); q != "" {
		return q == token
	}
	return false
}
