package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// SubjectAuthState: trạng thái xác thực của chủ thể (cho mỗi request).
// exists=false nếu đã bị xoá; active=false nếu bị khoá/ngưng; validAfter = mốc vô hiệu phiên (nil = không).
func (s *Store) SubjectAuthState(ctx context.Context, subjectType string, subjectID int64) (exists, active bool, validAfter *time.Time, err error) {
	table := "users"
	if subjectType == "customer" {
		table = "customers"
	}
	err = s.pool.QueryRow(ctx, `SELECT is_active, tokens_valid_after FROM `+table+` WHERE id = $1`, subjectID).Scan(&active, &validAfter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil, nil
		}
		return false, false, nil, err
	}
	return true, active, validAfter, nil
}

// InvalidateSessions vô hiệu MỌI phiên hiện tại của chủ thể: đặt mốc tokens_valid_after = now()
// (chặn ngay access token cũ ở requireExistingSubject) và thu hồi toàn bộ refresh token.
func (s *Store) InvalidateSessions(ctx context.Context, subjectType string, subjectID int64) error {
	table := "users"
	if subjectType == "customer" {
		table = "customers"
	}
	if _, err := s.pool.Exec(ctx, `UPDATE `+table+` SET tokens_valid_after = now() WHERE id = $1`, subjectID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE subject_type = $1 AND subject_id = $2 AND revoked_at IS NULL`, subjectType, subjectID)
	return err
}

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
