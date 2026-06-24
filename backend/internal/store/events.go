package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNoSpins       = errors.New("no spins remaining")
	ErrNotRegistered = errors.New("not registered")
)

// CreateEvent tạo sự kiện + danh sách ô thưởng (chỉ manager — enforce ở handler).
func (s *Store) CreateEvent(ctx context.Context, title, description, typ string, prizes []EventPrize, createdBy int64) (Event, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Event{}, err
	}
	defer tx.Rollback(ctx)

	var e Event
	err = tx.QueryRow(ctx, `
		INSERT INTO events (title, description, type, created_by)
		VALUES ($1,$2,$3,$4)
		RETURNING id, title, COALESCE(description,''), type, starts_at, ends_at, is_active, created_by, created_at`,
		title, description, typ, createdBy,
	).Scan(&e.ID, &e.Title, &e.Description, &e.Type, &e.StartsAt, &e.EndsAt, &e.IsActive, &e.CreatedBy, &e.CreatedAt)
	if err != nil {
		return Event{}, err
	}

	for _, p := range prizes {
		var pr EventPrize
		err = tx.QueryRow(ctx, `
			INSERT INTO event_prizes (event_id, name, image_url, weight, stock)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id, event_id, name, COALESCE(image_url,''), weight, stock, is_active`,
			e.ID, p.Name, p.ImageURL, p.Weight, p.Stock,
		).Scan(&pr.ID, &pr.EventID, &pr.Name, &pr.ImageURL, &pr.Weight, &pr.Stock, &pr.IsActive)
		if err != nil {
			return Event{}, err
		}
		e.Prizes = append(e.Prizes, pr)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, action, target_type, target_id, detail)
		VALUES ($1,'event.create','event',$2, jsonb_build_object('title',$3::text))`,
		createdBy, e.ID, title); err != nil {
		return Event{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Event{}, err
	}
	return e, nil
}

const eventSelect = `
SELECT e.id, e.title, COALESCE(e.description,''), e.type, e.starts_at, e.ends_at, e.is_active, e.created_by, e.created_at,
       e.register_deadline, e.prize_type, e.voucher_id, e.prize_vehicle_catalog_id, e.winners_count, e.draw_status,
       e.cancelled_at, COALESCE(e.cancel_reason,''),
       COALESCE(vc.name, vh.name, '') AS prize_name
FROM events e
LEFT JOIN vouchers vc ON vc.id = e.voucher_id
LEFT JOIN vehicle_catalog vh ON vh.id = e.prize_vehicle_catalog_id`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.Title, &e.Description, &e.Type, &e.StartsAt, &e.EndsAt, &e.IsActive, &e.CreatedBy, &e.CreatedAt,
		&e.RegisterDeadline, &e.PrizeType, &e.VoucherID, &e.PrizeVehicleID, &e.WinnersCount, &e.DrawStatus,
		&e.CancelledAt, &e.CancelReason, &e.PrizeName)
	return e, err
}

// ListEvents: onlyActive=true cho KHÁCH (loại sự kiện đã huỷ); false cho QUẢN LÝ (xem tất cả kể cả đã huỷ).
func (s *Store) ListEvents(ctx context.Context, onlyActive bool) ([]Event, error) {
	q := eventSelect
	if onlyActive {
		q += ` WHERE e.is_active AND e.cancelled_at IS NULL`
	}
	q += ` ORDER BY e.created_at DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// tính số khách đã đăng ký cho từng sự kiện quay số.
	for i := range out {
		if out[i].DrawStatus != nil {
			out[i].EligibleCount, _ = s.CountRegistrations(ctx, out[i].ID)
		}
	}
	return out, nil
}

func (s *Store) GetEvent(ctx context.Context, id int64) (Event, error) {
	e, err := scanEvent(s.pool.QueryRow(ctx, eventSelect+` WHERE e.id = $1`, id))
	if err != nil {
		return e, mapNotFound(err)
	}
	if e.DrawStatus != nil {
		// sự kiện quay số: số người đã đăng ký + danh sách trúng
		if e.EligibleCount, err = s.CountRegistrations(ctx, id); err != nil {
			return e, err
		}
		if e.Winners, err = s.ListWinners(ctx, id); err != nil {
			return e, err
		}
	} else {
		// sự kiện vòng quay cũ
		if e.Prizes, err = s.listPrizes(ctx, id); err != nil {
			return e, err
		}
	}
	return e, nil
}

func (s *Store) listPrizes(ctx context.Context, eventID int64) ([]EventPrize, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_id, name, COALESCE(image_url,''), weight, stock, is_active
		FROM event_prizes WHERE event_id = $1 AND is_active ORDER BY id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventPrize
	for rows.Next() {
		var p EventPrize
		if err := rows.Scan(&p.ID, &p.EventID, &p.Name, &p.ImageURL, &p.Weight, &p.Stock, &p.IsActive); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Register cho khách đăng ký tham gia sự kiện quay số. Idempotent.
// Chỉ cho đăng ký khi sự kiện đang mở và còn trong hạn đăng ký.
func (s *Store) Register(ctx context.Context, eventID, customerID int64) error {
	var drawStatus *string
	var deadline *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT draw_status, register_deadline FROM events WHERE id = $1`, eventID).Scan(&drawStatus, &deadline)
	if err != nil {
		return mapNotFound(err)
	}
	if drawStatus == nil {
		return ErrNotDrawEvent
	}
	if *drawStatus != "open" {
		return ErrBadDrawState // đã quay/công bố -> không nhận đăng ký nữa
	}
	if deadline != nil && time.Now().After(*deadline) {
		return ErrRegistrationClosed
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO event_registrations (event_id, customer_id)
		VALUES ($1,$2) ON CONFLICT (event_id, customer_id) DO NOTHING`, eventID, customerID)
	return err
}

// Spin thực hiện 1 lượt quay theo trọng số, loại ô đã hết stock.
// rng là số ngẫu nhiên [0,1) do caller cung cấp (để dễ test & tránh phụ thuộc).
func (s *Store) Spin(ctx context.Context, eventID, customerID int64, rng float64) (Spin, error) {
	var sp Spin
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return sp, err
	}
	defer tx.Rollback(ctx)

	// kiểm tra lượt quay còn lại (khóa dòng).
	var remaining int
	err = tx.QueryRow(ctx, `
		SELECT spins_remaining FROM event_registrations
		WHERE event_id = $1 AND customer_id = $2 FOR UPDATE`, eventID, customerID).Scan(&remaining)
	if err != nil {
		return sp, ErrNotRegistered
	}
	if remaining <= 0 {
		return sp, ErrNoSpins
	}

	// lấy các ô còn quay được (stock null = vô hạn, hoặc stock > 0).
	rows, err := tx.Query(ctx, `
		SELECT id, name, weight, stock FROM event_prizes
		WHERE event_id = $1 AND is_active AND weight > 0
		  AND (stock IS NULL OR stock > 0) ORDER BY id FOR UPDATE`, eventID)
	if err != nil {
		return sp, err
	}
	type prize struct {
		id     int64
		name   string
		weight int
		stock  *int
	}
	var prizes []prize
	total := 0
	for rows.Next() {
		var p prize
		if err := rows.Scan(&p.id, &p.name, &p.weight, &p.stock); err != nil {
			rows.Close()
			return sp, err
		}
		prizes = append(prizes, p)
		total += p.weight
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sp, err
	}

	// chọn ô theo trọng số.
	var chosenID *int64
	chosenName := "Chúc may mắn lần sau"
	if total > 0 {
		target := int(rng * float64(total))
		if target >= total {
			target = total - 1
		}
		acc := 0
		for _, p := range prizes {
			acc += p.weight
			if target < acc {
				id := p.id
				chosenID = &id
				chosenName = p.name
				if p.stock != nil {
					if _, err := tx.Exec(ctx, `UPDATE event_prizes SET stock = stock - 1 WHERE id = $1`, p.id); err != nil {
						return sp, err
					}
				}
				break
			}
		}
	}

	// giảm lượt quay.
	if _, err := tx.Exec(ctx, `
		UPDATE event_registrations SET spins_remaining = spins_remaining - 1
		WHERE event_id = $1 AND customer_id = $2`, eventID, customerID); err != nil {
		return sp, err
	}

	// ghi lượt quay.
	err = tx.QueryRow(ctx, `
		INSERT INTO event_spins (event_id, customer_id, prize_id, prize_name)
		VALUES ($1,$2,$3,$4)
		RETURNING id, event_id, customer_id, prize_id, prize_name, created_at`,
		eventID, customerID, chosenID, chosenName,
	).Scan(&sp.ID, &sp.EventID, &sp.CustomerID, &sp.PrizeID, &sp.PrizeName, &sp.CreatedAt)
	if err != nil {
		return sp, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sp, err
	}
	return sp, nil
}
