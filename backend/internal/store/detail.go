package store

import "context"

// SimilarOnSale gợi ý các xe đang mở bán cùng dòng (class) hoặc cùng hãng (brand),
// trừ chính nó. Ưu tiên cùng dòng trước.
func (s *Store) SimilarOnSale(ctx context.Context, inventoryID int64, limit int) ([]InventoryItem, error) {
	if limit <= 0 || limit > 12 {
		limit = 4
	}
	const tClass = `(SELECT c2.class FROM inventory i2 JOIN vehicle_catalog c2 ON c2.id = i2.catalog_id WHERE i2.id = $1)`
	const tBrand = `(SELECT c3.brand FROM inventory i3 JOIN vehicle_catalog c3 ON c3.id = i3.catalog_id WHERE i3.id = $1)`
	q := inventorySelect + `
		WHERE i.status = 'on_sale' AND i.quantity > 0 AND i.id <> $1
		  AND (c.class = ` + tClass + ` OR c.brand = ` + tBrand + `)
		ORDER BY (c.class = ` + tClass + `) DESC, i.created_at DESC
		LIMIT ` + itoa(limit)

	rows, err := s.pool.Query(ctx, q, inventoryID)
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

// ListDiscounts lịch sử khuyến mãi của một dòng kho (mới nhất trước).
func (s *Store) ListDiscounts(ctx context.Context, inventoryID int64) ([]Discount, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, percent, starts_at, ends_at, is_active, created_at
		FROM discounts WHERE inventory_id = $1 ORDER BY created_at DESC`, inventoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Discount
	for rows.Next() {
		var d Discount
		if err := rows.Scan(&d.ID, &d.Percent, &d.StartsAt, &d.EndsAt, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
