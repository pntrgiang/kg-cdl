package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// rankOrder thứ tự hạng để so sánh (regular < vip < svip).
func rankOrder(rank string) int {
	switch rank {
	case "svip":
		return 2
	case "vip":
		return 1
	default:
		return 0
	}
}

// SellResult là kết quả của một giao dịch bán xe.
type SellResult struct {
	Sale          Sale
	NewRank       string // rank của khách sau giao dịch
	RankChangedTo string // != "" nếu rank thay đổi
}

// SellOptions: tùy chọn cho một giao dịch bán.
type SellOptions struct {
	CustomerVoucherID *int64   // id trong customer_vouchers (voucher khả dụng của khách)
	PrizeWinID        *int64   // (đã ngừng dùng) xe tặng trúng thưởng
	OverridePrice     *float64 // giá bán tuỳ chỉnh CHO PHIÊN NÀY (bỏ KM %), không đổi giá gốc xe
}

var (
	// ErrOutOfStock khi xe đã hết hàng.
	ErrOutOfStock = fmt.Errorf("out of stock")
	// ErrPrizeInvalid khi voucher/giải thưởng không hợp lệ với khách.
	ErrPrizeInvalid = fmt.Errorf("prize or voucher invalid")
	// ErrVoucherDepleted khi voucher đã hết số lượng.
	ErrVoucherDepleted = fmt.Errorf("voucher depleted")
	// ErrVoucherAlreadyUsed khi khách đã dùng voucher này rồi.
	ErrVoucherAlreadyUsed = fmt.Errorf("voucher already used by customer")
	// ErrVoucherExpired khi voucher đã hết hạn.
	ErrVoucherExpired = fmt.Errorf("voucher expired")
	// ErrVoucherRank khi hạng khách không đủ điều kiện dùng voucher.
	ErrVoucherRank = fmt.Errorf("voucher rank not met")
	// ErrVoucherNotApplicable khi voucher không áp dụng cho xe đang bán.
	ErrVoucherNotApplicable = fmt.Errorf("voucher not applicable to this vehicle")
)

// SellVehicle thực hiện trọn vẹn trong 1 transaction:
//  1. khóa & kiểm tra tồn kho, giảm số lượng
//  2. lấy giá + khuyến mãi đang áp dụng, tạo bản ghi sale (snapshot giá)
//  3. cộng total_spent cho khách
//  4. recompute toàn bộ rank khách theo giới hạn svip/vip
//  5. ghi activity log
func (s *Store) SellVehicle(ctx context.Context, inventoryID, customerID, soldBy int64, actorName string, svipLimit, vipLimit int, opts SellOptions) (SellResult, error) {
	var res SellResult
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	// Serialize mọi thao tác xếp lại hạng (bán/hoàn/đổi giới hạn) bằng advisory lock
	// theo transaction -> tránh deadlock & giữ rank nhất quán khi có giao dịch đồng thời.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, rankLockKey); err != nil {
		return res, err
	}

	// 1. khóa dòng kho.
	var qty int
	var catalogID int64
	var basePrice float64
	var vehicleName string
	err = tx.QueryRow(ctx, `
		SELECT i.quantity, i.catalog_id, i.base_price, c.name
		FROM inventory i JOIN vehicle_catalog c ON c.id = i.catalog_id
		WHERE i.id = $1 FOR UPDATE OF i`, inventoryID,
	).Scan(&qty, &catalogID, &basePrice, &vehicleName)
	if err != nil {
		return res, mapNotFound(err)
	}
	if qty <= 0 {
		return res, ErrOutOfStock
	}

	// giá bán: nếu có giá tuỳ chỉnh cho phiên này -> dùng giá đó (bỏ KM %),
	// không thay đổi giá gốc của xe. Ngược lại dùng giá gốc + khuyến mãi đang áp dụng.
	var percent float64
	if opts.OverridePrice != nil && *opts.OverridePrice > 0 {
		basePrice = round2(*opts.OverridePrice)
		percent = 0
	} else {
		_ = tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(percent),0) FROM discounts
			WHERE inventory_id = $1 AND is_active AND starts_at <= now()
			  AND (ends_at IS NULL OR ends_at > now())`, inventoryID).Scan(&percent)
	}
	finalPrice := round2(basePrice * (1 - percent/100))

	// 2. giảm tồn kho, set sold_out nếu hết.
	if _, err := tx.Exec(ctx, `
		UPDATE inventory SET quantity = quantity - 1,
		  status = CASE WHEN quantity - 1 = 0 THEN 'sold_out'::vehicle_status ELSE status END,
		  updated_at = now()
		WHERE id = $1`, inventoryID); err != nil {
		return res, err
	}

	// khách + tên.
	var customerName, oldRank string
	if err := tx.QueryRow(ctx, `SELECT full_name, rank FROM customers WHERE id = $1 FOR UPDATE`, customerID).
		Scan(&customerName, &oldRank); err != nil {
		return res, mapNotFound(err)
	}

	// tùy chọn: dùng voucher (kiểm tra hạn, hạng, phạm vi xe).
	var voucherID *int64
	var voucherDiscount float64
	if opts.CustomerVoucherID != nil {
		var vid int64
		var vpercent, vmax float64
		var vqty, vused int
		var vexpires *time.Time
		var vAll bool
		var vMinRank string
		if err := tx.QueryRow(ctx, `
			SELECT v.id, v.discount_percent, v.max_amount, v.quantity, v.used_count,
			       v.expires_at, v.applies_to_all, v.min_rank
			FROM customer_vouchers cv JOIN vouchers v ON v.id=cv.voucher_id
			WHERE cv.id=$1 AND cv.customer_id=$2 AND cv.status='available' AND v.is_active
			FOR UPDATE OF cv, v`,
			*opts.CustomerVoucherID, customerID).Scan(&vid, &vpercent, &vmax, &vqty, &vused, &vexpires, &vAll, &vMinRank); err != nil {
			return res, ErrPrizeInvalid
		}
		if vqty > 0 && vused >= vqty {
			return res, ErrVoucherDepleted
		}
		// hết hạn
		if vexpires != nil && time.Now().After(*vexpires) {
			return res, ErrVoucherExpired
		}
		// hạng tối thiểu
		if rankOrder(oldRank) < rankOrder(vMinRank) {
			return res, ErrVoucherRank
		}
		// áp dụng cho xe đang bán
		if !vAll {
			var ok bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM voucher_vehicles WHERE voucher_id=$1 AND catalog_id=$2)`,
				vid, catalogID).Scan(&ok); err != nil {
				return res, err
			}
			if !ok {
				return res, ErrVoucherNotApplicable
			}
		}
		// 1 khách không được dùng cùng 1 voucher quá 1 lần
		var already bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM customer_vouchers
			  WHERE customer_id=$1 AND voucher_id=$2 AND status='used')`,
			customerID, vid).Scan(&already); err != nil {
			return res, err
		}
		if already {
			return res, ErrVoucherAlreadyUsed
		}
		voucherDiscount = round2(finalPrice * vpercent / 100)
		if vmax > 0 && voucherDiscount > vmax { // vmax=0 => tối đa = giá trị xe (không giới hạn thêm)
			voucherDiscount = vmax
		}
		finalPrice = round2(finalPrice - voucherDiscount)
		if finalPrice < 0 {
			finalPrice = 0
		}
		voucherID = &vid
	}
	var prizeWinID *int64
	isGift := false

	// 3. tạo sale.
	var sale Sale
	err = tx.QueryRow(ctx, `
		INSERT INTO sales (inventory_id, catalog_id, customer_id, sold_by,
		  original_price, discount_percent, final_price, vehicle_name,
		  voucher_id, voucher_discount, prize_win_id, is_gift)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, inventory_id, catalog_id, customer_id, sold_by,
		  original_price, discount_percent, final_price, vehicle_name, created_at`,
		inventoryID, catalogID, customerID, soldBy, basePrice, percent, finalPrice, vehicleName,
		voucherID, voucherDiscount, prizeWinID, isGift,
	).Scan(&sale.ID, &sale.InventoryID, &sale.CatalogID, &sale.CustomerID, &sale.SoldBy,
		&sale.OriginalPrice, &sale.DiscountPercent, &sale.FinalPrice, &sale.VehicleName, &sale.CreatedAt)
	if err != nil {
		return res, err
	}
	sale.CustomerName = customerName
	sale.SoldByName = actorName

	// đánh dấu voucher đã dùng / giải xe đã giao.
	if voucherID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE customer_vouchers SET status='used', used_sale_id=$2, used_at=now() WHERE id=$1`,
			*opts.CustomerVoucherID, sale.ID); err != nil {
			return res, err
		}
		// trừ số lượng voucher còn lại
		if _, err := tx.Exec(ctx, `UPDATE vouchers SET used_count = used_count + 1 WHERE id=$1`, *voucherID); err != nil {
			return res, err
		}
	}
	if prizeWinID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE event_winners SET fulfilled_at=now(), fulfilled_sale_id=$2 WHERE id=$1`,
			*prizeWinID, sale.ID); err != nil {
			return res, err
		}
	}

	// 4. cộng total_spent.
	if _, err := tx.Exec(ctx, `
		UPDATE customers SET total_spent = total_spent + $2, last_purchase_at = now(), updated_at = now()
		WHERE id = $1`, customerID, finalPrice); err != nil {
		return res, err
	}

	// recompute rank cho toàn bộ khách active.
	newRank, err := recomputeRanksTx(ctx, tx, svipLimit, vipLimit)
	if err != nil {
		return res, err
	}

	// 5. activity log.
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'sale.create','sale',$3, jsonb_build_object(
		  'vehicle', $4::text, 'customer', $5::text, 'final_price', $6::numeric, 'discount_percent', $7::numeric))`,
		soldBy, actorName, sale.ID, vehicleName, customerName, finalPrice, percent); err != nil {
		return res, err
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Sale = sale
	res.NewRank = newRank[customerID]
	if res.NewRank != oldRank {
		res.RankChangedTo = res.NewRank
	}
	return res, nil
}

// ListCustomerSales lịch sử mua xe của 1 khách (mới nhất trước), kèm voucher & trạng thái hoàn.
func (s *Store) ListCustomerSales(ctx context.Context, customerID int64) ([]Sale, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.inventory_id, s.catalog_id, s.customer_id, c.full_name,
		       s.sold_by, COALESCE(u.display_name,'(đã xoá)'), s.original_price, s.discount_percent,
		       s.final_price, s.vehicle_name, COALESCE(s.voucher_discount,0),
		       (s.refunded_at IS NOT NULL), COALESCE(s.refund_reason,''), s.created_at
		FROM sales s
		JOIN customers c ON c.id = s.customer_id
		LEFT JOIN users u ON u.id = s.sold_by
		WHERE s.customer_id = $1
		ORDER BY s.created_at DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Sale{}
	for rows.Next() {
		var sl Sale
		if err := rows.Scan(&sl.ID, &sl.InventoryID, &sl.CatalogID, &sl.CustomerID, &sl.CustomerName,
			&sl.SoldBy, &sl.SoldByName, &sl.OriginalPrice, &sl.DiscountPercent,
			&sl.FinalPrice, &sl.VehicleName, &sl.VoucherDiscount,
			&sl.Refunded, &sl.RefundReason, &sl.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}

// RecomputeRanks xếp lại rank toàn bộ khách theo giới hạn mới (transaction riêng).
func (s *Store) RecomputeRanks(ctx context.Context, svipLimit, vipLimit int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, rankLockKey); err != nil {
		return err
	}
	if _, err := recomputeRanksTx(ctx, tx, svipLimit, vipLimit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ErrAlreadyRefunded khi giao dịch đã hoàn trước đó.
var ErrAlreadyRefunded = fmt.Errorf("sale already refunded")

// Lỗi khi chuyển giao dịch (chỉ áp dụng cho tài khoản tạm).
var ErrNotTransferable = fmt.Errorf("sale not transferable")     // nguồn không phải tài khoản tạm / đã hoàn
var ErrInvalidTransferTarget = fmt.Errorf("invalid transfer target") // đích không hợp lệ

// TransferSale chuyển 1 giao dịch CHƯA hoàn từ TÀI KHOẢN TẠM (exclude_from_rank) sang khách thật:
// dời customer_id, dịch chuyển total_spent (trừ nguồn, cộng đích), cập nhật last_purchase, xếp lại hạng.
func (s *Store) TransferSale(ctx context.Context, saleID, toCustomerID, actorID int64, actorName string, svipLimit, vipLimit int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, rankLockKey); err != nil {
		return err
	}

	var fromCustomerID int64
	var finalPrice float64
	var vehicleName string
	var refunded *time.Time
	var saleCreatedAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT customer_id, final_price, vehicle_name, refunded_at, created_at
		FROM sales WHERE id = $1 FOR UPDATE`, saleID,
	).Scan(&fromCustomerID, &finalPrice, &vehicleName, &refunded, &saleCreatedAt)
	if err != nil {
		return mapNotFound(err)
	}
	if refunded != nil {
		return ErrNotTransferable // giao dịch đã hoàn -> không chuyển
	}
	if toCustomerID == fromCustomerID {
		return ErrInvalidTransferTarget
	}

	// nguồn PHẢI là tài khoản tạm (exclude_from_rank) — tạm thời chỉ áp dụng cho loại này (vd LUX00000)
	var fromExcluded bool
	if err := tx.QueryRow(ctx, `SELECT exclude_from_rank FROM customers WHERE id = $1`, fromCustomerID).Scan(&fromExcluded); err != nil {
		return mapNotFound(err)
	}
	if !fromExcluded {
		return ErrNotTransferable
	}

	// đích phải tồn tại, đang hoạt động, KHÔNG phải tài khoản tạm
	var toActive, toExcluded bool
	if err := tx.QueryRow(ctx, `SELECT is_active, exclude_from_rank FROM customers WHERE id = $1`, toCustomerID).Scan(&toActive, &toExcluded); err != nil {
		return ErrInvalidTransferTarget
	}
	if !toActive || toExcluded {
		return ErrInvalidTransferTarget
	}

	// dời giao dịch sang khách thật
	if _, err := tx.Exec(ctx, `UPDATE sales SET customer_id = $2 WHERE id = $1`, saleID, toCustomerID); err != nil {
		return err
	}
	// trừ chi tiêu của tài khoản tạm
	if _, err := tx.Exec(ctx, `UPDATE customers SET total_spent = greatest(total_spent - $2, 0), updated_at = now() WHERE id = $1`, fromCustomerID, finalPrice); err != nil {
		return err
	}
	// cộng chi tiêu cho khách thật + cập nhật mốc mua gần nhất
	if _, err := tx.Exec(ctx, `
		UPDATE customers
		SET total_spent = total_spent + $2,
		    last_purchase_at = greatest(coalesce(last_purchase_at, $3), $3),
		    updated_at = now()
		WHERE id = $1`, toCustomerID, finalPrice, saleCreatedAt); err != nil {
		return err
	}
	// xếp lại hạng toàn bộ khách
	if _, err := recomputeRanksTx(ctx, tx, svipLimit, vipLimit); err != nil {
		return err
	}
	// log
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'sale.transfer','sale',$3, jsonb_build_object('vehicle',$4::text,'amount',$5::numeric,'from',$6::bigint,'to',$7::bigint))`,
		actorID, actorName, saleID, vehicleName, finalPrice, fromCustomerID, toCustomerID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// rankLockKey: khóa advisory dùng chung để serialize mọi thao tác xếp lại hạng khách.
const rankLockKey int64 = 91713

// RefundSale (quản lý) hoàn trả một giao dịch bán sai: trả xe vào kho, trừ lại chi tiêu,
// xếp lại hạng, hoàn voucher đã dùng (nếu có), đánh dấu đã hoàn + ghi log lý do.
func (s *Store) RefundSale(ctx context.Context, saleID, refundedBy int64, actorName, reason string, svipLimit, vipLimit int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, rankLockKey); err != nil {
		return err
	}

	var inventoryID, customerID int64
	var voucherID *int64
	var finalPrice float64
	var vehicleName string
	var refunded *time.Time
	err = tx.QueryRow(ctx, `
		SELECT inventory_id, customer_id, voucher_id, final_price, vehicle_name, refunded_at
		FROM sales WHERE id = $1 FOR UPDATE`, saleID,
	).Scan(&inventoryID, &customerID, &voucherID, &finalPrice, &vehicleName, &refunded)
	if err != nil {
		return mapNotFound(err)
	}
	if refunded != nil {
		return ErrAlreadyRefunded
	}

	// 1. trả xe vào kho (nếu đang sold_out -> on_sale lại).
	if _, err := tx.Exec(ctx, `
		UPDATE inventory SET quantity = quantity + 1,
		  status = CASE WHEN status = 'sold_out' THEN 'on_sale'::vehicle_status ELSE status END,
		  updated_at = now()
		WHERE id = $1`, inventoryID); err != nil {
		return err
	}

	// 2. trừ lại chi tiêu của khách.
	if _, err := tx.Exec(ctx, `
		UPDATE customers SET total_spent = greatest(total_spent - $2, 0), updated_at = now()
		WHERE id = $1`, customerID, finalPrice); err != nil {
		return err
	}

	// 3. hoàn voucher đã dùng trong giao dịch này (nếu có).
	if _, err := tx.Exec(ctx, `
		UPDATE customer_vouchers SET status='available', used_sale_id=NULL, used_at=NULL
		WHERE used_sale_id = $1`, saleID); err != nil {
		return err
	}
	if voucherID != nil {
		if _, err := tx.Exec(ctx, `UPDATE vouchers SET used_count = greatest(used_count - 1, 0) WHERE id = $1`, *voucherID); err != nil {
			return err
		}
	}

	// 4. xếp lại hạng toàn bộ khách.
	if _, err := recomputeRanksTx(ctx, tx, svipLimit, vipLimit); err != nil {
		return err
	}

	// 5. đánh dấu đã hoàn + log.
	if _, err := tx.Exec(ctx, `
		UPDATE sales SET refunded_at = now(), refunded_by = $2, refund_reason = $3 WHERE id = $1`,
		saleID, refundedBy, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'sale.refund','sale',$3, jsonb_build_object('vehicle',$4::text,'amount',$5::numeric,'reason',$6::text))`,
		refundedBy, actorName, saleID, vehicleName, finalPrice, reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// recomputeRanksTx xếp lại rank toàn bộ khách theo total_spent.
// Trả về map customerID -> rank mới. Ghi customer_rank_history khi có thay đổi.
func recomputeRanksTx(ctx context.Context, tx pgx.Tx, svipLimit, vipLimit int) (map[int64]string, error) {
	// Đưa về 'regular' những khách KHÔNG xếp hạng: tài khoản tạm (exclude_from_rank)
	// hoặc không còn chi tiêu (total_spent <= 0, vd sau khi hoàn trả) — tránh giữ hạng cũ.
	if _, err := tx.Exec(ctx, `
		UPDATE customers SET rank = 'regular', updated_at = now()
		WHERE rank <> 'regular' AND (exclude_from_rank OR total_spent <= 0)`); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT id, rank FROM customers
		WHERE is_active AND total_spent > 0 AND NOT exclude_from_rank
		ORDER BY total_spent DESC, last_purchase_at ASC NULLS LAST, id ASC`)
	if err != nil {
		return nil, err
	}
	type cust struct {
		id  int64
		old string
	}
	var list []cust
	for rows.Next() {
		var c cust
		if err := rows.Scan(&c.id, &c.old); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[int64]string, len(list))
	for i, c := range list {
		var newRank string
		switch {
		case i < svipLimit:
			newRank = "svip"
		case i < svipLimit+vipLimit:
			newRank = "vip"
		default:
			newRank = "regular"
		}
		result[c.id] = newRank
		if newRank != c.old {
			if _, err := tx.Exec(ctx, `UPDATE customers SET rank = $2, updated_at = now() WHERE id = $1`, c.id, newRank); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO customer_rank_history (customer_id, old_rank, new_rank, reason)
				VALUES ($1, $2, $3, 'recompute after sale')`, c.id, c.old, newRank); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// ListSales liệt kê giao dịch (mới nhất trước).
func (s *Store) ListSales(ctx context.Context, limit int) ([]Sale, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.inventory_id, s.catalog_id, s.customer_id, c.full_name,
		       s.sold_by, u.display_name, s.original_price, s.discount_percent,
		       s.final_price, s.vehicle_name, s.created_at
		FROM sales s
		JOIN customers c ON c.id = s.customer_id
		JOIN users u ON u.id = s.sold_by
		ORDER BY s.created_at DESC LIMIT `+itoa(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Sale
	for rows.Next() {
		var sl Sale
		if err := rows.Scan(&sl.ID, &sl.InventoryID, &sl.CatalogID, &sl.CustomerID, &sl.CustomerName,
			&sl.SoldBy, &sl.SoldByName, &sl.OriginalPrice, &sl.DiscountPercent,
			&sl.FinalPrice, &sl.VehicleName, &sl.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}
