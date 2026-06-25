package store

import (
	"context"
	"encoding/json"
	"time"
)

// RankLimits giới hạn số lượng theo hạng (lưu ở settings.rank_limits).
type RankLimits struct {
	SVIP int `json:"svip"`
	VIP  int `json:"vip"`
}

func (s *Store) GetRankLimits(ctx context.Context) (RankLimits, error) {
	limits := RankLimits{SVIP: 3, VIP: 5} // mặc định
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = 'rank_limits'`).Scan(&raw)
	if err != nil {
		return limits, nil // dùng mặc định nếu thiếu
	}
	_ = json.Unmarshal(raw, &limits)
	return limits, nil
}

func (s *Store) SetRankLimits(ctx context.Context, limits RankLimits) error {
	raw, _ := json.Marshal(limits)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ('rank_limits', $1, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, raw)
	return err
}

// WeekStartDow: ngày bắt đầu tuần mở bán — CỐ ĐỊNH thứ Bảy (Postgres dow 6).
const WeekStartDow = 6

// GetReleaseOverride đọc mốc countdown mở bán do quản lý đặt riêng (nil nếu không có).
func (s *Store) GetReleaseOverride(ctx context.Context) (*time.Time, error) {
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = 'release_override'`).Scan(&raw); err != nil {
		return nil, nil // không có -> dùng mặc định
	}
	var ts time.Time
	if err := json.Unmarshal(raw, &ts); err != nil {
		return nil, nil
	}
	return &ts, nil
}

// SetReleaseOverride đặt mốc countdown tuỳ chỉnh cho tuần hiện tại.
func (s *Store) SetReleaseOverride(ctx context.Context, t time.Time) error {
	raw, _ := json.Marshal(t)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ('release_override', $1, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, raw)
	return err
}

// ClearReleaseOverride xoá mốc tuỳ chỉnh -> countdown trở về mặc định (thứ 7 21:00).
func (s *Store) ClearReleaseOverride(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM settings WHERE key = 'release_override'`)
	return err
}

// ── Tạm dừng tự động mở bán (hiệu lực trong NGÀY hôm đó) ──
// Lưu ngày (theo session TZ) vào settings.release_pause; chỉ có hiệu lực khi == hôm nay.

// SetReleasePauseToday tạm dừng tự động mở bán cho hôm nay.
func (s *Store) SetReleasePauseToday(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES ('release_pause', to_jsonb(current_date::text), now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`)
	return err
}

// ClearReleasePause bật lại tự động mở bán (xoá mốc tạm dừng).
func (s *Store) ClearReleasePause(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM settings WHERE key = 'release_pause'`)
	return err
}

// IsReleasePausedToday: true nếu đang tạm dừng tự động mở bán cho đúng hôm nay.
func (s *Store) IsReleasePausedToday(ctx context.Context) (bool, error) {
	var paused bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM settings
			WHERE key = 'release_pause' AND (value #>> '{}')::date = current_date
		)`).Scan(&paused)
	return paused, err
}

// ModalConfig: cấu hình popup thông báo mở bán (ảnh + đích chuyển hướng khi bấm ảnh).
type ModalConfig struct {
	Image  string `json:"image"`  // URL ảnh; rỗng = tắt popup
	Target string `json:"target"` // đường dẫn chuyển hướng: /upcoming hoặc /events
}

// GetModalConfig đọc cấu hình popup. Mặc định target = /upcoming.
// Tương thích ngược: nếu chưa có key mới nhưng còn ảnh cũ thì vẫn lấy ảnh đó.
func (s *Store) GetModalConfig(ctx context.Context) (ModalConfig, error) {
	cfg := ModalConfig{Target: "/upcoming"}
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = 'release_modal'`).Scan(&raw); err != nil {
		// fallback: ảnh lưu theo key cũ
		var oldRaw []byte
		if err2 := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = 'release_modal_image'`).Scan(&oldRaw); err2 == nil {
			_ = json.Unmarshal(oldRaw, &cfg.Image)
		}
		return cfg, nil
	}
	_ = json.Unmarshal(raw, &cfg)
	if cfg.Target != "/events" {
		cfg.Target = "/upcoming"
	}
	return cfg, nil
}

// SetModalConfig lưu cấu hình popup (image rỗng = tắt popup).
func (s *Store) SetModalConfig(ctx context.Context, cfg ModalConfig) error {
	if cfg.Target != "/events" {
		cfg.Target = "/upcoming"
	}
	raw, _ := json.Marshal(cfg)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ('release_modal', $1, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, raw)
	return err
}
