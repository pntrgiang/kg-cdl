package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const catalogCols = `id, model_code, name, COALESCE(brand,''), COALESCE(class,''),
	COALESCE(image_url,''), COALESCE(description,''), COALESCE(model_3d,''),
	seats, COALESCE(trunk_kg,10), COALESCE(rate_speed,0), COALESCE(rate_accel,0), COALESCE(rate_braking,0), COALESCE(rate_traction,0),
	is_mod, created_at`

func scanCatalog(row pgx.Row) (CatalogVehicle, error) {
	var c CatalogVehicle
	err := row.Scan(&c.ID, &c.ModelCode, &c.Name, &c.Brand, &c.Class,
		&c.ImageURL, &c.Description, &c.Model3D,
		&c.Seats, &c.TrunkKg, &c.RateSpeed, &c.RateAccel, &c.RateBraking, &c.RateTraction,
		&c.IsMod, &c.CreatedAt)
	return c, err
}

// ListCatalog tìm kiếm danh mục mẫu xe.
func (s *Store) ListCatalog(ctx context.Context, search string, limit int) ([]CatalogVehicle, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT ` + catalogCols + ` FROM vehicle_catalog`
	args := []any{}
	if search != "" {
		q += ` WHERE name ILIKE $1 OR brand ILIKE $1 OR model_code ILIKE $1`
		args = append(args, "%"+search+"%")
	}
	q += ` ORDER BY name LIMIT ` + itoa(limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatalogVehicle
	for rows.Next() {
		c, err := scanCatalog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCatalog(ctx context.Context, id int64) (CatalogVehicle, error) {
	c, err := scanCatalog(s.pool.QueryRow(ctx, `SELECT `+catalogCols+` FROM vehicle_catalog WHERE id = $1`, id))
	return c, mapNotFound(err)
}

// CatalogSpecs gói thông số kỹ thuật (số chỗ + cốp xe + điểm hiệu năng 0-100).
type CatalogSpecs struct {
	Seats        *int
	TrunkKg      int
	RateSpeed    int
	RateAccel    int
	RateBraking  int
	RateTraction int
}

// CreateCatalog tạo mẫu xe mới (thường là xe mod) kèm thông số.
func (s *Store) CreateCatalog(ctx context.Context, modelCode *string, name, brand, class, imageURL, description string, sp CatalogSpecs, isMod bool, createdBy int64) (CatalogVehicle, error) {
	return scanCatalog(s.pool.QueryRow(ctx, `
		INSERT INTO vehicle_catalog (model_code, name, brand, class, image_url, description,
		  seats, trunk_kg, rate_speed, rate_accel, rate_braking, rate_traction, is_mod, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING `+catalogCols,
		modelCode, name, brand, class, imageURL, description,
		sp.Seats, sp.TrunkKg, sp.RateSpeed, sp.RateAccel, sp.RateBraking, sp.RateTraction, isMod, createdBy))
}

// UpdateCatalogInfo cập nhật giới thiệu + thông số (chỉ manager — enforce ở handler).
func (s *Store) UpdateCatalogInfo(ctx context.Context, id int64, description string, sp CatalogSpecs) (CatalogVehicle, error) {
	c, err := scanCatalog(s.pool.QueryRow(ctx, `
		UPDATE vehicle_catalog SET description = $2, seats = $3, trunk_kg = $4,
		  rate_speed = $5, rate_accel = $6, rate_braking = $7, rate_traction = $8
		WHERE id = $1
		RETURNING `+catalogCols,
		id, description, sp.Seats, sp.TrunkKg, sp.RateSpeed, sp.RateAccel, sp.RateBraking, sp.RateTraction))
	return c, mapNotFound(err)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
