package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotDrawEvent       = errors.New("not a draw event")
	ErrBadDrawState       = errors.New("invalid draw state")
	ErrNoEligible         = errors.New("no eligible participants")
	ErrRegistrationClosed = errors.New("registration closed")
	ErrInviteOnly         = errors.New("event is invite-only")
	ErrEventNotCancelable = errors.New("event cannot be cancelled")
)

// CancelEvent (quản lý) huỷ một sự kiện CHƯA quay số: vô hiệu hoá + ghi log lý do.
// Chỉ huỷ được khi draw_status='open' (chưa quay) và còn hoạt động.
func (s *Store) CancelEvent(ctx context.Context, eventID, cancelledBy int64, actorName, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var drawStatus *string
	var title string
	var cancelledAt *time.Time
	err = tx.QueryRow(ctx, `SELECT draw_status, title, cancelled_at FROM events WHERE id = $1 FOR UPDATE`, eventID).
		Scan(&drawStatus, &title, &cancelledAt)
	if err != nil {
		return mapNotFound(err)
	}
	if drawStatus == nil {
		return ErrNotDrawEvent
	}
	if cancelledAt != nil || *drawStatus != "open" {
		return ErrEventNotCancelable // đã quay số / đã huỷ trước đó
	}

	if _, err := tx.Exec(ctx, `
		UPDATE events SET cancelled_at = now(), cancelled_by = $2, cancel_reason = $3 WHERE id = $1`,
		eventID, cancelledBy, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'event.cancel','event',$3, jsonb_build_object('title',$4::text,'reason',$5::text))`,
		cancelledBy, actorName, eventID, title, reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CreateDrawEvent tạo sự kiện quay số trúng thưởng (chỉ manager — enforce ở handler).
func (s *Store) CreateDrawEvent(ctx context.Context, title, description string, deadline time.Time, prizeType string, voucherID, vehicleCatalogID *int64, winnersCount int, inviteCustomerIDs []int64, createdBy int64) (Event, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Event{}, err
	}
	defer tx.Rollback(ctx)

	inviteOnly := len(inviteCustomerIDs) > 0
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO events (title, description, type, created_by,
		  register_deadline, prize_type, voucher_id, prize_vehicle_catalog_id, winners_count, draw_status, invite_only)
		VALUES ($1,$2,'discount_campaign',$3,$4,$5,$6,$7,$8,'open',$9)
		RETURNING id`,
		title, description, createdBy, deadline, prizeType, voucherID, vehicleCatalogID, winnersCount, inviteOnly,
	).Scan(&id)
	if err != nil {
		return Event{}, err
	}
	// sự kiện chỉ định: đưa sẵn người được chọn vào danh sách quay số (không cần đăng ký).
	if inviteOnly {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_registrations (event_id, customer_id)
			SELECT $1, cid FROM unnest($2::bigint[]) AS cid
			ON CONFLICT (event_id, customer_id) DO NOTHING`, id, inviteCustomerIDs); err != nil {
			return Event{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, action, target_type, target_id, detail)
		VALUES ($1,'event.create','event',$2, jsonb_build_object('title',$3::text,'prize_type',$4::text,'winners',$5::int))`,
		createdBy, id, title, prizeType, winnersCount); err != nil {
		return Event{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Event{}, err
	}
	return s.GetEvent(ctx, id)
}

// CountRegistrations đếm số khách đã ĐĂNG KÝ tham gia sự kiện.
func (s *Store) CountRegistrations(ctx context.Context, eventID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM event_registrations WHERE event_id = $1`, eventID).Scan(&n)
	return n, err
}

// EventEntrant: khách đã đăng ký tham gia (để hiện lên vòng quay).
type EventEntrant struct {
	CustomerID   int64  `json:"customer_id"`
	CustomerName string `json:"customer_name"`
}

// ListEventEntrants danh sách khách đã đăng ký tham gia sự kiện (theo thứ tự đăng ký).
func (s *Store) ListEventEntrants(ctx context.Context, eventID int64) ([]EventEntrant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.customer_id, c.full_name
		FROM event_registrations r JOIN customers c ON c.id = r.customer_id
		WHERE r.event_id = $1 ORDER BY r.created_at, r.customer_id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EventEntrant{}
	for rows.Next() {
		var e EventEntrant
		if err := rows.Scan(&e.CustomerID, &e.CustomerName); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// IsRegistered kiểm tra một khách đã đăng ký tham gia sự kiện chưa.
func (s *Store) IsRegistered(ctx context.Context, eventID, customerID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM event_registrations WHERE event_id=$1 AND customer_id=$2)`,
		eventID, customerID).Scan(&ok)
	return ok, err
}

// DrawWinners chọn ngẫu nhiên người trúng (nháp, chờ xác nhận). Chỉ khi draw_status='open'.
func (s *Store) DrawWinners(ctx context.Context, eventID, actorID int64, actorName string) ([]EventWinner, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var drawStatus *string
	var winnersCount *int
	err = tx.QueryRow(ctx, `
		SELECT draw_status, winners_count FROM events WHERE id = $1 FOR UPDATE`, eventID,
	).Scan(&drawStatus, &winnersCount)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if drawStatus == nil {
		return nil, ErrNotDrawEvent
	}
	if *drawStatus != "open" {
		return nil, ErrBadDrawState
	}
	n := 1
	if winnersCount != nil {
		n = *winnersCount
	}

	// chọn ngẫu nhiên trong số khách ĐÃ ĐĂNG KÝ tham gia
	rows, err := tx.Query(ctx, `
		SELECT customer_id FROM event_registrations
		WHERE event_id = $2
		ORDER BY random() LIMIT $1`, n, eventID)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var cid int64
		if err := rows.Scan(&cid); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, cid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNoEligible
	}

	for _, cid := range ids {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_winners (event_id, customer_id, status) VALUES ($1,$2,'pending')
			ON CONFLICT (event_id, customer_id) DO NOTHING`, eventID, cid); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE events SET draw_status='drawn' WHERE id=$1`, eventID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'draw.run','event',$3, jsonb_build_object('winners',$4::int))`,
		actorID, actorName, eventID, len(ids)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListWinners(ctx, eventID)
}

// RedrawWinners quay lại: chỉ khi đang ở trạng thái 'drawn' (đã quay, chưa công bố).
// Xoá kết quả nháp cũ, chọn lại ngẫu nhiên, GHI LẠI LÝ DO vào nhật ký.
func (s *Store) RedrawWinners(ctx context.Context, eventID, actorID int64, actorName, reason string) ([]EventWinner, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var drawStatus *string
	var winnersCount *int
	if err := tx.QueryRow(ctx,
		`SELECT draw_status, winners_count FROM events WHERE id=$1 FOR UPDATE`, eventID).
		Scan(&drawStatus, &winnersCount); err != nil {
		return nil, mapNotFound(err)
	}
	if drawStatus == nil {
		return nil, ErrNotDrawEvent
	}
	if *drawStatus != "drawn" {
		return nil, ErrBadDrawState // chỉ quay lại khi đã quay & chưa công bố
	}
	n := 1
	if winnersCount != nil {
		n = *winnersCount
	}

	// xoá kết quả nháp cũ
	if _, err := tx.Exec(ctx, `DELETE FROM event_winners WHERE event_id=$1`, eventID); err != nil {
		return nil, err
	}
	// chọn lại từ danh sách đã đăng ký
	rows, err := tx.Query(ctx, `
		SELECT customer_id FROM event_registrations WHERE event_id=$1 ORDER BY random() LIMIT $2`, eventID, n)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var cid int64
		if err := rows.Scan(&cid); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, cid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNoEligible
	}
	for _, cid := range ids {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_winners (event_id, customer_id, status) VALUES ($1,$2,'pending')`, eventID, cid); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'draw.redraw','event',$3, jsonb_build_object('reason',$4::text,'winners',$5::int))`,
		actorID, actorName, eventID, reason, len(ids)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListWinners(ctx, eventID)
}

// ConfirmDraw quản lý xác nhận → công bố kết quả + phát voucher cho người trúng.
func (s *Store) ConfirmDraw(ctx context.Context, eventID, actorID int64, actorName string) ([]EventWinner, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var drawStatus, prizeType *string
	var voucherID *int64
	err = tx.QueryRow(ctx, `
		SELECT draw_status, prize_type, voucher_id FROM events WHERE id=$1 FOR UPDATE`, eventID,
	).Scan(&drawStatus, &prizeType, &voucherID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if drawStatus == nil {
		return nil, ErrNotDrawEvent
	}
	if *drawStatus != "drawn" {
		return nil, ErrBadDrawState
	}

	if _, err := tx.Exec(ctx, `
		UPDATE event_winners SET status='confirmed', confirmed_at=now()
		WHERE event_id=$1 AND status='pending'`, eventID); err != nil {
		return nil, err
	}

	// phát voucher cho người trúng nếu thưởng là voucher
	if prizeType != nil && *prizeType == "voucher" && voucherID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO customer_vouchers (customer_id, voucher_id, event_id, status)
			SELECT w.customer_id, $2, $1, 'available' FROM event_winners w
			WHERE w.event_id=$1 AND w.status='confirmed'`, eventID, *voucherID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE events SET draw_status='published' WHERE id=$1`, eventID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,'draw.confirm','event',$3, jsonb_build_object('prize_type', COALESCE($4::text,'')))`,
		actorID, actorName, eventID, prizeType); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListWinners(ctx, eventID)
}

// ListWinners danh sách người trúng kèm tên khách.
func (s *Store) ListWinners(ctx context.Context, eventID int64) ([]EventWinner, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT w.id, w.customer_id, c.full_name, w.status, w.fulfilled_at, w.created_at
		FROM event_winners w JOIN customers c ON c.id = w.customer_id
		WHERE w.event_id = $1 ORDER BY w.created_at`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventWinner
	for rows.Next() {
		var win EventWinner
		if err := rows.Scan(&win.ID, &win.CustomerID, &win.CustomerName, &win.Status, &win.FulfilledAt, &win.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, win)
	}
	return out, rows.Err()
}
