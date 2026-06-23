package store

import (
	"context"
	"time"
)

// Banner: ảnh banner trang chủ.
type Banner struct {
	ID        int64     `json:"id"`
	Image     string    `json:"image"`     // tên file trong UploadDir
	ImageURL  string    `json:"image_url"` // đường dẫn phục vụ /api/uploads/<image>
	IsActive  bool      `json:"is_active"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func scanBanner(rows interface{ Scan(...any) error }) (Banner, error) {
	var b Banner
	err := rows.Scan(&b.ID, &b.Image, &b.IsActive, &b.Sort, &b.CreatedAt)
	b.ImageURL = "/api/uploads/" + b.Image
	return b, err
}

// ListBanners tất cả banner (cho quản lý). active=true -> chỉ banner đang bật (cho trang khách).
func (s *Store) ListBanners(ctx context.Context, onlyActive bool) ([]Banner, error) {
	q := `SELECT id, image, is_active, sort, created_at FROM banners`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY sort, id`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Banner{}
	for rows.Next() {
		b, err := scanBanner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateBanner thêm banner mới (file đã lưu sẵn vào UploadDir).
func (s *Store) CreateBanner(ctx context.Context, image string, createdBy int64) (Banner, error) {
	b, err := scanBanner(s.pool.QueryRow(ctx, `
		INSERT INTO banners (image, created_by, sort)
		VALUES ($1, $2, COALESCE((SELECT max(sort)+1 FROM banners), 0))
		RETURNING id, image, is_active, sort, created_at`, image, createdBy))
	return b, err
}

// SetBannerActive bật/tắt một banner.
func (s *Store) SetBannerActive(ctx context.Context, id int64, active bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE banners SET is_active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBanner xoá banner, trả về tên file để xoá ảnh kèm theo.
func (s *Store) DeleteBanner(ctx context.Context, id int64) (string, error) {
	var image string
	err := s.pool.QueryRow(ctx, `DELETE FROM banners WHERE id = $1 RETURNING image`, id).Scan(&image)
	if err != nil {
		return "", mapNotFound(err)
	}
	return image, nil
}
