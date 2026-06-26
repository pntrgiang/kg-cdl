package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrBookingClosed khi xe không nhận đặt lịch.
	ErrBookingClosed = fmt.Errorf("booking not open for this vehicle")
	// ErrBookingDuplicate khi khách còn lịch chưa qua ngày hẹn cho xe này.
	ErrBookingDuplicate = fmt.Errorf("already have an active booking for this vehicle")
	// ErrBookingHandled khi lịch đã được xử lý trước đó.
	ErrBookingHandled = fmt.Errorf("booking already handled")
)

// CountNewBookings: số lịch đặt được tạo SAU thời điểm nhân viên xem gần nhất (badge thông báo).
func (s *Store) CountNewBookings(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM bookings
		WHERE created_at > (SELECT bookings_seen_at FROM users WHERE id = $1)`, userID).Scan(&n)
	return n, err
}

// MarkBookingsSeen: đánh dấu nhân viên đã xem danh sách lịch (badge về 0).
func (s *Store) MarkBookingsSeen(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET bookings_seen_at = now() WHERE id = $1`, userID)
	return err
}

// SetBookingOpen bật/tắt nhận đặt lịch cho 1 mục kho (chỉ quản lý — enforce ở handler).
func (s *Store) SetBookingOpen(ctx context.Context, inventoryID int64, open bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE inventory SET booking_open = $2, updated_at = now() WHERE id = $1`, inventoryID, open)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateBooking tạo lịch đặt của khách (chỉ khi xe đang nhận đặt lịch, không trùng lịch chờ).
func (s *Store) CreateBooking(ctx context.Context, inventoryID, customerID int64, visitDate, note string) (Booking, error) {
	var b Booking
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return b, err
	}
	defer tx.Rollback(ctx)

	var bookingOpen bool
	var vehicleName string
	err = tx.QueryRow(ctx, `
		SELECT i.booking_open, c.name FROM inventory i
		JOIN vehicle_catalog c ON c.id = i.catalog_id WHERE i.id = $1`, inventoryID).Scan(&bookingOpen, &vehicleName)
	if err != nil {
		return b, mapNotFound(err)
	}
	if !bookingOpen {
		return b, ErrBookingClosed
	}

	// chống spam: chặn đặt lại nếu khách còn lịch cho xe này CHƯA qua ngày hẹn
	// (đang chờ hoặc đã được nhận). Lịch bị từ chối hoặc đã quá ngày xem thì cho đặt lại.
	var dup int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM bookings
		WHERE inventory_id = $1 AND customer_id = $2
		  AND status <> 'rejected' AND visit_date >= current_date`, inventoryID, customerID).Scan(&dup); err != nil {
		return b, err
	}
	if dup > 0 {
		return b, ErrBookingDuplicate
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO bookings (inventory_id, customer_id, vehicle_name, visit_date, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, inventory_id, customer_id, vehicle_name, visit_date::text, COALESCE(note,''), status, handled_at, created_at`,
		inventoryID, customerID, vehicleName, visitDate, note,
	).Scan(&b.ID, &b.InventoryID, &b.CustomerID, &b.VehicleName, &b.VisitDate, &b.Note, &b.Status, &b.HandledAt, &b.CreatedAt)
	if err != nil {
		// hai request đồng thời -> unique index uq_booking_pending chặn cái thứ hai.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return b, ErrBookingDuplicate
		}
		return b, err
	}
	return b, tx.Commit(ctx)
}

// ListCustomerBookings danh sách lịch của 1 khách (cho trang Tài khoản).
func (s *Store) ListCustomerBookings(ctx context.Context, customerID int64) ([]Booking, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, inventory_id, customer_id, vehicle_name, visit_date::text, COALESCE(note,''), status, handled_at, created_at
		FROM bookings WHERE customer_id = $1
		ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Booking{}
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.InventoryID, &b.CustomerID, &b.VehicleName, &b.VisitDate, &b.Note, &b.Status, &b.HandledAt, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListBookings danh sách lịch cho nhân viên/quản lý. status="" = tất cả.
// Ưu tiên "pending" lên đầu theo thứ tự đặt trước (FIFO), sau đó tới các lịch đã xử lý (mới nhất trước).
func (s *Store) ListBookings(ctx context.Context, status string) ([]Booking, error) {
	q := `
		SELECT b.id, b.inventory_id, b.customer_id, b.vehicle_name, b.visit_date::text, COALESCE(b.note,''),
		       b.status, b.handled_at, b.created_at,
		       c.full_name, c.national_id, COALESCE(c.phone,''), COALESCE(u.display_name,'')
		FROM bookings b
		JOIN customers c ON c.id = b.customer_id
		LEFT JOIN users u ON u.id = b.handled_by`
	args := []any{}
	if status != "" {
		q += ` WHERE b.status = $1`
		args = append(args, status)
	}
	q += `
		ORDER BY (b.status = 'pending') DESC,
		         CASE WHEN b.status = 'pending' THEN b.created_at END ASC,
		         b.created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Booking{}
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.InventoryID, &b.CustomerID, &b.VehicleName, &b.VisitDate, &b.Note,
			&b.Status, &b.HandledAt, &b.CreatedAt,
			&b.CustomerName, &b.CustomerNationalID, &b.CustomerPhone, &b.HandledByName); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// HandleBooking nhận (accepted) hoặc từ chối (rejected) một lịch đang chờ.
func (s *Store) HandleBooking(ctx context.Context, bookingID int64, status string, handledBy int64) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE bookings SET status = $2, handled_by = $3, handled_at = now()
		WHERE id = $1 AND status = 'pending'`, bookingID, status, handledBy)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		// hoặc không tồn tại, hoặc đã xử lý rồi.
		var exists int
		_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM bookings WHERE id = $1`, bookingID).Scan(&exists)
		if exists == 0 {
			return ErrNotFound
		}
		return ErrBookingHandled
	}
	return nil
}
