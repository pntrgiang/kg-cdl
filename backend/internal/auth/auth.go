// Package auth: mật khẩu (bcrypt), JWT access token, refresh token, RBAC.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// SubjectType phân biệt chủ thể của token.
const (
	SubjectUser     = "user"     // nhân viên/quản lý/dev
	SubjectCustomer = "customer" // khách hàng
)

// Roles của nhân viên.
const (
	RoleDev     = "dev"
	RoleManager = "manager"
	RoleStaff   = "staff"
)

// Identity là chủ thể đã xác thực, lấy từ access token.
type Identity struct {
	SubjectType string `json:"subject_type"`
	SubjectID   int64  `json:"subject_id"`
	Role        string `json:"role"` // chỉ có ý nghĩa với user; customer = ""
}

// ── mật khẩu ─────────────────────────────────────────────────────

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// ── JWT access token ─────────────────────────────────────────────

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

type accessClaims struct {
	SubjectType string `json:"styp"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}

// IssueAccessToken tạo access token ngắn hạn.
func (m *Manager) IssueAccessToken(id Identity) (string, time.Time, error) {
	exp := nowUTC().Add(m.accessTTL)
	claims := accessClaims{
		SubjectType: id.SubjectType,
		Role:        id.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", id.SubjectID),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(nowUTC()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(m.secret)
	return signed, exp, err
}

var ErrInvalidToken = errors.New("invalid token")

// ParseAccessToken xác thực và trả về Identity.
func (m *Manager) ParseAccessToken(tokenStr string) (Identity, error) {
	claims := &accessClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !tok.Valid {
		return Identity{}, ErrInvalidToken
	}
	var sid int64
	if _, err := fmt.Sscanf(claims.Subject, "%d", &sid); err != nil {
		return Identity{}, ErrInvalidToken
	}
	return Identity{SubjectType: claims.SubjectType, SubjectID: sid, Role: claims.Role}, nil
}

// ── refresh token ────────────────────────────────────────────────

// NewRefreshToken sinh token thô (trả cho client) + hash (lưu DB).
func NewRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken băm token bằng SHA-256 (refresh token là ngẫu nhiên đủ mạnh).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// nowUTC tách ra để dễ test; dùng thời gian thực.
var nowUTC = func() time.Time { return time.Now().UTC() }
