package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const voucherCols = `id, name, discount_percent, max_amount, quantity, used_count,
	greatest(quantity - used_count, 0) AS remaining, expires_at, applies_to_all, min_rank, is_active,
	cancelled_at, COALESCE(cancel_reason,''), created_at`

func scanVoucher(row pgx.Row) (Voucher, error) {
	var v Voucher
	err := row.Scan(&v.ID, &v.Name, &v.DiscountPercent, &v.MaxAmount, &v.Quantity, &v.UsedCount, &v.Remaining,
		&v.ExpiresAt, &v.AppliesToAll, &v.MinRank, &v.IsActive, &v.CancelledAt, &v.CancelReason, &v.CreatedAt)
	return v, err
}

// ErrAlreadyCancelled khi voucher đã bị huỷ trước đó.
var ErrAlreadyCancelled = fmt.Errorf("voucher already cancelled")

// CancelVoucher (quản lý) huỷ một voucher đã tạo: vô hiệu hoá voucher và huỷ luôn
// các bản đã phát cho khách mà CHƯA dùng (status='available' -> 'cancelled'), ghi log lý do.
func (s *Store) CancelVoucher(ctx context.Context, voucherID, cancelledBy int64, actorName, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var name string
	var cancelledAt *time.Time
	err = tx.QueryRow(ctx, `SELECT name, cancelled_at FROM vouchers WHERE id = $1 FOR UPDATE`, voucherID).Scan(&name, &cancelledAt)
	if err != nil {
		return mapNotFound(err)
	}
	if cancelledAt != nil {
		return ErrAlreadyCancelled
	}

	// 1. vô hiệu hoá voucher (không còn dùng được khi bán).
	if _, err := tx.Exec(ctx, `
		UPDATE vouchers SET is_active = false, cancelled_at = now(), cancelled_by = $2, cancel_reason = $3
		WHERE id = $1`, voucherID, cancelledBy, reason); err != nil {
		return err
	}

	// 2. huỷ các bản voucher khách đang sở hữu nhưng chưa dùng.
	affected := int64(0)
	ct, err := tx.Exec(ctx, `
		UPDATE customer_vouchers SET status = 'cancelled'
		WHERE voucher_id = $1 AND status = 'available'`, voucherID)
	if err != nil {
		return err
	}
	affected = ct.RowsAffected()

	// 3. ghi log lý do.
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'voucher.cancel','voucher',$3, jsonb_build_object('name',$4::text,'reason',$5::text,'revoked_from_customers',$6::int))`,
		cancelledBy, actorName, voucherID, name, reason, affected); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// VoucherInput gói dữ liệu tạo voucher.
type VoucherInput struct {
	Name            string
	DiscountPercent float64
	MaxAmount       float64 // 0 = tối đa = toàn bộ giá trị xe
	Quantity        int
	ExpiresAt       time.Time
	AppliesToAll    bool
	MinRank         string
	VehicleIDs      []int64 // khi AppliesToAll=false
}

// CreateVoucher tạo voucher (chỉ manager — enforce ở handler).
func (s *Store) CreateVoucher(ctx context.Context, in VoucherInput, createdBy int64) (Voucher, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Voucher{}, err
	}
	defer tx.Rollback(ctx)

	v, err := scanVoucher(tx.QueryRow(ctx, `
		INSERT INTO vouchers (name, discount_percent, max_amount, quantity, expires_at, applies_to_all, min_rank, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING `+voucherCols,
		in.Name, in.DiscountPercent, in.MaxAmount, in.Quantity, in.ExpiresAt, in.AppliesToAll, in.MinRank, createdBy))
	if err != nil {
		return Voucher{}, err
	}
	if !in.AppliesToAll {
		for _, cid := range in.VehicleIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO voucher_vehicles (voucher_id, catalog_id) VALUES ($1,$2)
				ON CONFLICT DO NOTHING`, v.ID, cid); err != nil {
				return Voucher{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Voucher{}, err
	}
	v.Vehicles, _ = s.listVoucherVehicles(ctx, v.ID)
	return v, nil
}

func (s *Store) listVoucherVehicles(ctx context.Context, voucherID int64) ([]VoucherVehicle, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name FROM voucher_vehicles vv JOIN vehicle_catalog c ON c.id = vv.catalog_id
		WHERE vv.voucher_id = $1 ORDER BY c.name`, voucherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VoucherVehicle{}
	for rows.Next() {
		var vv VoucherVehicle
		if err := rows.Scan(&vv.CatalogID, &vv.Name); err != nil {
			return nil, err
		}
		out = append(out, vv)
	}
	return out, rows.Err()
}

func (s *Store) ListVouchers(ctx context.Context) ([]Voucher, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+voucherCols+` FROM vouchers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Voucher
	for rows.Next() {
		v, err := scanVoucher(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if !out[i].AppliesToAll {
			out[i].Vehicles, _ = s.listVoucherVehicles(ctx, out[i].ID)
		}
	}
	return out, nil
}

func (s *Store) GetVoucher(ctx context.Context, id int64) (Voucher, error) {
	v, err := scanVoucher(s.pool.QueryRow(ctx, `SELECT `+voucherCols+` FROM vouchers WHERE id = $1`, id))
	if err != nil {
		return v, mapNotFound(err)
	}
	if !v.AppliesToAll {
		v.Vehicles, _ = s.listVoucherVehicles(ctx, id)
	}
	return v, nil
}

// ListCustomerVouchers voucher khả dụng của khách CHO XE đang chọn (catalogID):
// còn hạn, còn lượt, chưa dùng, đúng hạng, áp dụng được cho xe đó.
func (s *Store) ListCustomerVouchers(ctx context.Context, customerID, catalogID int64) ([]CustomerVoucher, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cv.id, v.id, v.name, v.discount_percent, v.max_amount
		FROM customer_vouchers cv
		JOIN vouchers v ON v.id = cv.voucher_id
		JOIN customers cu ON cu.id = cv.customer_id
		WHERE cv.customer_id = $1 AND cv.status = 'available' AND v.is_active
		  AND v.used_count < v.quantity
		  AND (v.expires_at IS NULL OR v.expires_at > now())
		  AND cu.rank >= v.min_rank
		  AND ($2 = 0 OR v.applies_to_all OR EXISTS (
		        SELECT 1 FROM voucher_vehicles vv WHERE vv.voucher_id = v.id AND vv.catalog_id = $2))
		  AND NOT EXISTS (SELECT 1 FROM customer_vouchers u
		    WHERE u.customer_id = cv.customer_id AND u.voucher_id = cv.voucher_id AND u.status = 'used')
		ORDER BY cv.created_at`, customerID, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CustomerVoucher{}
	for rows.Next() {
		var c CustomerVoucher
		if err := rows.Scan(&c.ID, &c.VoucherID, &c.Name, &c.DiscountPercent, &c.MaxAmount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListAllCustomerVouchers TẤT CẢ voucher của khách (cả đã dùng) kèm chi tiết + nhân viên đã áp dụng.
func (s *Store) ListAllCustomerVouchers(ctx context.Context, customerID int64) ([]CustomerVoucherFull, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cv.id, v.id, v.name, v.discount_percent, v.max_amount, cv.status, cv.used_at,
		       v.expires_at, v.applies_to_all, v.min_rank, u.display_name, COALESCE(v.cancel_reason,'')
		FROM customer_vouchers cv
		JOIN vouchers v ON v.id = cv.voucher_id
		LEFT JOIN sales sa ON sa.id = cv.used_sale_id
		LEFT JOIN users u ON u.id = sa.sold_by
		WHERE cv.customer_id = $1
		ORDER BY cv.created_at DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CustomerVoucherFull{}
	var ids []int64
	for rows.Next() {
		var c CustomerVoucherFull
		if err := rows.Scan(&c.ID, &c.VoucherID, &c.Name, &c.DiscountPercent, &c.MaxAmount,
			&c.Status, &c.UsedAt, &c.ExpiresAt, &c.AppliesToAll, &c.MinRank, &c.SellerName, &c.CancelReason); err != nil {
			return nil, err
		}
		out = append(out, c)
		ids = append(ids, c.VoucherID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if !out[i].AppliesToAll {
			out[i].Vehicles, _ = s.listVoucherVehicles(ctx, out[i].VoucherID)
		}
	}
	return out, nil
}
