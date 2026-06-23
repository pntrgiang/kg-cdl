package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey int

const identityKey ctxKey = 0

// Middleware xác thực access token từ header Authorization: Bearer <token>.
// Nếu không có/không hợp lệ → 401. Gắn Identity vào context.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			unauthorized(w)
			return
		}
		id, err := m.ParseAccessToken(raw)
		if err != nil {
			unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), identityKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext lấy Identity đã xác thực.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// RequireUser chỉ cho phép chủ thể là nhân viên (user), không phải customer.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := FromContext(r.Context())
		if !ok || id.SubjectType != SubjectUser {
			forbidden(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole cho phép user có một trong các role chỉ định.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := FromContext(r.Context())
			if !ok || id.SubjectType != SubjectUser || !allowed[id.Role] {
				forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireCustomer chỉ cho phép chủ thể là khách hàng.
func RequireCustomer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := FromContext(r.Context())
		if !ok || id.SubjectType != SubjectCustomer {
			forbidden(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
}

func forbidden(w http.ResponseWriter) {
	http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
}
