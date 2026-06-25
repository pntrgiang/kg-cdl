package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// inventorySelect lấy kho kèm catalog + khuyến mãi đang active (tính sẵn giá sau giảm).
const inventorySelect = `
SELECT i.id, i.catalog_id, c.name, COALESCE(c.brand,''), COALESCE(c.class,''),
       COALESCE(c.image_url,''), COALESCE(c.description,''), COALESCE(c.model_3d,''),
       c.seats, COALESCE(c.trunk_kg,10), COALESCE(c.rate_speed,0), COALESCE(c.rate_accel,0), COALESCE(c.rate_braking,0), COALESCE(c.rate_traction,0),
       i.base_price, i.quantity,
       i.quantity + COALESCE((SELECT count(*) FROM sales s WHERE s.inventory_id = i.id AND s.refunded_at IS NULL), 0) AS total_imported,
       i.status, i.on_sale_at, COALESCE(i.note,''),
       COALESCE(d.percent, 0) AS discount_percent,
       i.booking_open,
       i.created_at
FROM inventory i
JOIN vehicle_catalog c ON c.id = i.catalog_id
LEFT JOIN LATERAL (
    SELECT percent FROM discounts d
    WHERE d.inventory_id = i.id AND d.is_active
      AND d.starts_at <= now() AND (d.ends_at IS NULL OR d.ends_at > now())
    ORDER BY percent DESC LIMIT 1
) d ON true`

func scanInventory(row pgx.Row) (InventoryItem, error) {
	var it InventoryItem
	err := row.Scan(&it.ID, &it.CatalogID, &it.Name, &it.Brand, &it.Class,
		&it.ImageURL, &it.Description, &it.Model3D,
		&it.Seats, &it.TrunkKg, &it.RateSpeed, &it.RateAccel, &it.RateBraking, &it.RateTraction,
		&it.BasePrice, &it.Quantity, &it.TotalImported, &it.Status,
		&it.OnSaleAt, &it.Note, &it.DiscountPercent, &it.BookingOpen, &it.CreatedAt)
	if err != nil {
		return it, err
	}
	it.FinalPrice = round2(it.BasePrice * (1 - it.DiscountPercent/100))
	return it, nil
}

// ListInventory trả về kho. status="" = tất cả; "on_sale"/"upcoming"/... lọc theo.
func (s *Store) ListInventory(ctx context.Context, status string) ([]InventoryItem, error) {
	q := inventorySelect
	args := []any{}
	if status != "" {
		q += ` WHERE i.status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY i.created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryItem
	for rows.Next() {
		it, err := scanInventory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) GetInventory(ctx context.Context, id int64) (InventoryItem, error) {
	q := inventorySelect + ` WHERE i.id = $1`
	it, err := scanInventory(s.pool.QueryRow(ctx, q, id))
	return it, mapNotFound(err)
}

// CreateInventory nhập kho từ một mẫu catalog (đường cũ: trạng thái thủ công).
func (s *Store) CreateInventory(ctx context.Context, catalogID int64, basePrice float64, quantity int, status string, onSaleAt *time.Time, note string, createdBy int64) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO inventory (catalog_id, base_price, quantity, status, on_sale_at, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		catalogID, basePrice, quantity, status, onSaleAt, note, createdBy,
	).Scan(&id)
	return id, err
}

// CreateInventoryForWeek nhập kho theo tuần mở bán: trạng thái tự suy
// (đang mở bán nếu tuần đang diễn ra, sắp mở bán nếu là tuần tương lai), on_sale_at = đầu tuần.
func (s *Store) CreateInventoryForWeek(ctx context.Context, catalogID int64, basePrice float64, quantity int, note string, salesWeekID, createdBy int64) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO inventory (catalog_id, base_price, quantity, status, on_sale_at, note, sales_week_id, created_by)
		SELECT $1, $2, $3,
		       CASE WHEN w.week_start <= current_date THEN 'on_sale' ELSE 'upcoming' END::vehicle_status,
		       w.week_start::timestamptz, $4, w.id, $5
		FROM sales_weeks w WHERE w.id = $6
		RETURNING id`,
		catalogID, basePrice, quantity, note, createdBy, salesWeekID,
	).Scan(&id)
	if err != nil {
		return 0, mapNotFound(err)
	}
	return id, nil
}

// SetDiscount tắt khuyến mãi cũ và tạo khuyến mãi mới cho 1 dòng kho.
func (s *Store) SetDiscount(ctx context.Context, inventoryID int64, percent float64, endsAt *time.Time, createdBy int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE discounts SET is_active = false WHERE inventory_id = $1 AND is_active`, inventoryID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO discounts (inventory_id, percent, ends_at, created_by)
		VALUES ($1, $2, $3, $4)`, inventoryID, percent, endsAt, createdBy); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateInventoryStatus đổi trạng thái (vd upcoming -> on_sale).
func (s *Store) UpdateInventoryStatus(ctx context.Context, id int64, status string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE inventory SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateInventoryPrice đổi giá bán gốc (base_price) của một dòng kho.
func (s *Store) UpdateInventoryPrice(ctx context.Context, id int64, basePrice float64) error {
	ct, err := s.pool.Exec(ctx, `UPDATE inventory SET base_price = $2, updated_at = now() WHERE id = $1`, id, basePrice)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
