package store

import (
	"context"
	"time"
)

// SaveRefreshToken lưu hash của refresh token.
func (s *Store) SaveRefreshToken(ctx context.Context, subjectType string, subjectID int64, tokenHash string, expiresAt time.Time, userAgent, ip string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (subject_type, subject_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		subjectType, subjectID, tokenHash, expiresAt, nullStr(userAgent), nullIP(ip))
	return err
}

// RefreshSubject là chủ thể sở hữu refresh token còn hiệu lực.
type RefreshSubject struct {
	SubjectType string
	SubjectID   int64
}

// LookupRefreshToken trả về chủ thể nếu token còn hiệu lực (chưa thu hồi, chưa hết hạn).
func (s *Store) LookupRefreshToken(ctx context.Context, tokenHash string) (RefreshSubject, error) {
	var r RefreshSubject
	err := s.pool.QueryRow(ctx, `
		SELECT subject_type, subject_id FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`, tokenHash,
	).Scan(&r.SubjectType, &r.SubjectID)
	return r, mapNotFound(err)
}

// RevokeRefreshToken thu hồi 1 token (logout).
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func nullIP(ip string) any {
	if ip == "" {
		return nil
	}
	return ip
}
