package store

import (
	"context"
	"errors"
)

// ErrWeekPast khi cố đăng ký/chọn tuần đã qua.
var ErrWeekPast = errors.New("week is in the past")

type SalesWeek struct {
	ID        int64  `json:"id"`
	WeekStart string `json:"week_start"` // thứ Hai (YYYY-MM-DD)
	WeekEnd   string `json:"week_end"`   // Chủ Nhật
	Label     string `json:"label"`
	IsCurrent bool   `json:"is_current"` // đang diễn ra (week_start <= hôm nay <= week_end)
}

// ListSalesWeeks trả các tuần CHƯA kết thúc (tuần hiện tại + tương lai) để chọn khi nhập kho.
func (s *Store) ListSalesWeeks(ctx context.Context) ([]SalesWeek, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, to_char(week_start,'YYYY-MM-DD'), to_char(week_end,'YYYY-MM-DD'), COALESCE(label,''),
		       (week_start <= current_date AND current_date <= week_end)
		FROM sales_weeks
		WHERE week_end >= current_date
		ORDER BY week_start`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SalesWeek{}
	for rows.Next() {
		var w SalesWeek
		if err := rows.Scan(&w.ID, &w.WeekStart, &w.WeekEnd, &w.Label, &w.IsCurrent); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// CreateSalesWeek đăng ký tuần mở bán mới từ một ngày bất kỳ (tự quy về thứ Hai).
// Không cho đăng ký tuần đã kết thúc. Nếu tuần đã tồn tại thì trả về tuần đó.
func (s *Store) CreateSalesWeek(ctx context.Context, dateStr string, createdBy int64) (SalesWeek, error) {
	var ws, we string
	// quy ngày về thứ Bảy đầu tuần (dow 6): week_start = d - ((dow(d) - 6 + 7) % 7)
	err := s.pool.QueryRow(ctx, `
		WITH w AS (SELECT ($1::date - ((extract(dow from $1::date)::int - 6 + 7) % 7))::date AS ws)
		SELECT to_char(ws,'YYYY-MM-DD'), to_char(ws + 6,'YYYY-MM-DD') FROM w`, dateStr).Scan(&ws, &we)
	if err != nil {
		return SalesWeek{}, err
	}
	var past bool
	if err := s.pool.QueryRow(ctx, `SELECT $1::date < current_date`, we).Scan(&past); err != nil {
		return SalesWeek{}, err
	}
	if past {
		return SalesWeek{}, ErrWeekPast
	}

	var w SalesWeek
	err = s.pool.QueryRow(ctx, `
		INSERT INTO sales_weeks (week_start, week_end, label, created_by)
		VALUES ($1::date, $2::date, 'Tuần ' || to_char($1::date,'DD/MM') || ' – ' || to_char($2::date,'DD/MM/YYYY'), $3)
		ON CONFLICT (week_start) DO UPDATE SET week_start = EXCLUDED.week_start
		RETURNING id, to_char(week_start,'YYYY-MM-DD'), to_char(week_end,'YYYY-MM-DD'), COALESCE(label,''),
		          (week_start <= current_date AND current_date <= week_end)`,
		ws, we, createdBy).Scan(&w.ID, &w.WeekStart, &w.WeekEnd, &w.Label, &w.IsCurrent)
	return w, err
}

// PromoteDueInventory chuyển xe 'upcoming' sang 'on_sale' khi tuần mở bán đã bắt đầu.
func (s *Store) PromoteDueInventory(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE inventory i SET status = 'on_sale', updated_at = now()
		FROM sales_weeks w
		WHERE i.sales_week_id = w.id AND i.status = 'upcoming'
		  AND w.week_start <= current_date AND i.quantity > 0`)
	return err
}
