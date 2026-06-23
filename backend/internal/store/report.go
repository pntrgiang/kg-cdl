package store

import (
	"context"
	"time"
)

type RevenueSummary struct {
	Revenue      float64 `json:"revenue"`
	Sales        int     `json:"sales"`
	Gifts        int     `json:"gifts"`
	VoucherUses  int     `json:"voucher_uses"`
	VoucherTotal float64 `json:"voucher_total"`
	AvgSale      float64 `json:"avg_sale"`
}

type NamedRevenue struct {
	Name    string  `json:"name"`
	Sales   int     `json:"sales"`
	Revenue float64 `json:"revenue"`
}

// SaleDetail là chi tiết một xe đã bán.
type SaleDetail struct {
	ID              int64     `json:"id"`
	VehicleName     string    `json:"vehicle_name"`
	CustomerName    string    `json:"customer_name"`
	SoldByName      string    `json:"sold_by_name"`
	OriginalPrice   float64   `json:"original_price"`
	DiscountPercent float64   `json:"discount_percent"`
	VoucherDiscount float64   `json:"voucher_discount"`
	FinalPrice      float64   `json:"final_price"`
	IsGift          bool      `json:"is_gift"`
	Refunded        bool      `json:"refunded"`
	RefundReason    string    `json:"refund_reason"`
	CreatedAt       time.Time `json:"created_at"`
}

// WeekReport doanh thu + chi tiết các xe đã bán trong một tuần.
type WeekReport struct {
	WeekStart string       `json:"week_start"` // thứ Hai (YYYY-MM-DD)
	WeekEnd   string       `json:"week_end"`   // Chủ Nhật
	Sales     int          `json:"sales"`
	Revenue   float64      `json:"revenue"`
	Items     []SaleDetail `json:"items"`
}

type RevenueReport struct {
	Summary      RevenueSummary `json:"summary"`
	Weeks        []WeekReport   `json:"weeks"`
	TopVehicles  []NamedRevenue `json:"top_vehicles"`
	TopCustomers []NamedRevenue `json:"top_customers"`
}

// RevenueReport: tổng quan + doanh thu theo TUẦN kèm chi tiết từng xe đã bán + top xe/khách.
func (s *Store) RevenueReport(ctx context.Context) (RevenueReport, error) {
	r := RevenueReport{Weeks: []WeekReport{}, TopVehicles: []NamedRevenue{}, TopCustomers: []NamedRevenue{}}

	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_price),0), COUNT(*),
		       COALESCE(SUM(CASE WHEN is_gift THEN 1 ELSE 0 END),0),
		       COALESCE(SUM(CASE WHEN voucher_id IS NOT NULL THEN 1 ELSE 0 END),0),
		       COALESCE(SUM(voucher_discount),0), COALESCE(AVG(final_price),0)
		FROM sales WHERE refunded_at IS NULL`).Scan(
		&r.Summary.Revenue, &r.Summary.Sales, &r.Summary.Gifts,
		&r.Summary.VoucherUses, &r.Summary.VoucherTotal, &r.Summary.AvgSale)
	if err != nil {
		return r, err
	}

	// tất cả giao dịch kèm mốc tuần (date_trunc 'week' = thứ Hai), mới nhất trước.
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.vehicle_name, c.full_name, COALESCE(u.display_name,'(đã xoá)'),
		       s.original_price, s.discount_percent, s.voucher_discount, s.final_price, s.is_gift,
		       (s.refunded_at IS NOT NULL), COALESCE(s.refund_reason,''), s.created_at,
		       to_char((s.created_at::date - ((extract(dow from s.created_at)::int - 6 + 7) % 7)), 'YYYY-MM-DD')
		FROM sales s
		JOIN customers c ON c.id = s.customer_id
		LEFT JOIN users u ON u.id = s.sold_by
		ORDER BY s.created_at DESC`)
	if err != nil {
		return r, err
	}
	defer rows.Close()

	idx := map[string]int{} // week_start -> vị trí trong r.Weeks
	for rows.Next() {
		var d SaleDetail
		var weekStart string
		if err := rows.Scan(&d.ID, &d.VehicleName, &d.CustomerName, &d.SoldByName,
			&d.OriginalPrice, &d.DiscountPercent, &d.VoucherDiscount, &d.FinalPrice, &d.IsGift,
			&d.Refunded, &d.RefundReason, &d.CreatedAt, &weekStart); err != nil {
			return r, err
		}
		i, ok := idx[weekStart]
		if !ok {
			i = len(r.Weeks)
			idx[weekStart] = i
			r.Weeks = append(r.Weeks, WeekReport{WeekStart: weekStart, WeekEnd: weekEnd(weekStart), Items: []SaleDetail{}})
		}
		r.Weeks[i].Items = append(r.Weeks[i].Items, d)
		if !d.Refunded { // giao dịch đã hoàn không tính vào doanh thu
			r.Weeks[i].Sales++
			r.Weeks[i].Revenue = round2(r.Weeks[i].Revenue + d.FinalPrice)
		}
	}
	if err := rows.Err(); err != nil {
		return r, err
	}

	if r.TopVehicles, err = s.topNamed(ctx, `
		SELECT vehicle_name, COUNT(*), COALESCE(SUM(final_price),0)
		FROM sales WHERE refunded_at IS NULL GROUP BY vehicle_name ORDER BY SUM(final_price) DESC, COUNT(*) DESC LIMIT 10`); err != nil {
		return r, err
	}
	if r.TopCustomers, err = s.topNamed(ctx, `
		SELECT c.full_name, COUNT(*), COALESCE(SUM(s.final_price),0)
		FROM sales s JOIN customers c ON c.id = s.customer_id
		WHERE s.refunded_at IS NULL
		GROUP BY c.id, c.full_name ORDER BY SUM(s.final_price) DESC, COUNT(*) DESC LIMIT 10`); err != nil {
		return r, err
	}
	return r, nil
}

// weekEnd cộng 6 ngày vào mốc thứ Hai để ra Chủ Nhật (YYYY-MM-DD).
func weekEnd(weekStart string) string {
	t, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		return weekStart
	}
	return t.AddDate(0, 0, 6).Format("2006-01-02")
}

func (s *Store) topNamed(ctx context.Context, q string) ([]NamedRevenue, error) {
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NamedRevenue{}
	for rows.Next() {
		var n NamedRevenue
		if err := rows.Scan(&n.Name, &n.Sales, &n.Revenue); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
