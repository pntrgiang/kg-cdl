package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const customerCols = `id, username, full_name, phone, national_id, rank, total_spent,
	last_purchase_at, claimed_at, is_active, created_at`

func scanCustomer(row pgx.Row) (Customer, error) {
	var c Customer
	err := row.Scan(&c.ID, &c.Username, &c.FullName, &c.Phone, &c.NationalID, &c.Rank,
		&c.TotalSpent, &c.LastPurchase, &c.ClaimedAt, &c.IsActive, &c.CreatedAt)
	return c, err
}

// CreateCustomerByStaff tạo khách hàng từ nhân viên (không có tài khoản đăng nhập).
func (s *Store) CreateCustomerByStaff(ctx context.Context, fullName, phone, nationalID string, createdBy int64) (Customer, error) {
	return scanCustomer(s.pool.QueryRow(ctx, `
		INSERT INTO customers (full_name, phone, national_id, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING `+customerCols, fullName, phone, nationalID, createdBy))
}

// GetCustomerByNationalID dùng cho luồng claim.
func (s *Store) GetCustomerByNationalID(ctx context.Context, nationalID string) (Customer, error) {
	c, err := scanCustomer(s.pool.QueryRow(ctx,
		`SELECT `+customerCols+` FROM customers WHERE national_id = $1`, nationalID))
	return c, mapNotFound(err)
}

func (s *Store) GetCustomerByID(ctx context.Context, id int64) (Customer, error) {
	c, err := scanCustomer(s.pool.QueryRow(ctx,
		`SELECT `+customerCols+` FROM customers WHERE id = $1`, id))
	return c, mapNotFound(err)
}

// GetCustomerByUsername trả về customer + password_hash để xác thực đăng nhập.
func (s *Store) GetCustomerByUsername(ctx context.Context, username string) (Customer, string, error) {
	var c Customer
	var hash *string
	err := s.pool.QueryRow(ctx, `
		SELECT `+customerCols+`, password_hash FROM customers WHERE username = $1`, username,
	).Scan(&c.ID, &c.Username, &c.FullName, &c.Phone, &c.NationalID, &c.Rank, &c.TotalSpent,
		&c.LastPurchase, &c.ClaimedAt, &c.IsActive, &c.CreatedAt, &hash)
	h := ""
	if hash != nil {
		h = *hash
	}
	return c, h, mapNotFound(err)
}

// ClaimExisting gắn tài khoản (username/password) vào khách đã được nhân viên tạo trước.
func (s *Store) ClaimExisting(ctx context.Context, id int64, username, passwordHash string) (Customer, error) {
	c, err := scanCustomer(s.pool.QueryRow(ctx, `
		UPDATE customers SET username = $2, password_hash = $3, claimed_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING `+customerCols, id, username, passwordHash))
	return c, mapNotFound(err)
}

// RegisterNewCustomer tạo khách tự đăng ký (chưa từng tồn tại national_id).
func (s *Store) RegisterNewCustomer(ctx context.Context, username, passwordHash, nationalID, fullName, phone string) (Customer, error) {
	return scanCustomer(s.pool.QueryRow(ctx, `
		INSERT INTO customers (username, password_hash, national_id, full_name, phone, claimed_at)
		VALUES ($1, $2, $3, $4, $5, now())
		RETURNING `+customerCols, username, passwordHash, nationalID, fullName, phone))
}

func (s *Store) ListCustomers(ctx context.Context, search string) ([]Customer, error) {
	q := `SELECT ` + customerCols + ` FROM customers WHERE is_active`
	args := []any{}
	if search != "" {
		q += ` AND (full_name ILIKE $1 OR phone ILIKE $1 OR national_id ILIKE $1)`
		args = append(args, "%"+search+"%")
	}
	q += ` ORDER BY total_spent DESC, full_name`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCustomerPasswordHash lấy hash mật khẩu theo id (để xác thực khi đổi mật khẩu).
func (s *Store) GetCustomerPasswordHash(ctx context.Context, id int64) (string, error) {
	var hash *string
	err := s.pool.QueryRow(ctx, `SELECT password_hash FROM customers WHERE id = $1`, id).Scan(&hash)
	if err != nil {
		return "", mapNotFound(err)
	}
	if hash == nil {
		return "", nil
	}
	return *hash, nil
}

// UpdateCustomerPassword đổi mật khẩu của chính khách hàng.
func (s *Store) UpdateCustomerPassword(ctx context.Context, id int64, passwordHash string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE customers SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ResetCustomerLogin (chỉ dev) đặt lại đăng nhập của khách về số căn cước:
// username = căn cước, password = hash(căn cước), và đảm bảo tài khoản đã kích hoạt.
func (s *Store) ResetCustomerLogin(ctx context.Context, id int64, nationalID, passwordHash string) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE customers
		SET password_hash = $3, username = $2, claimed_at = COALESCE(claimed_at, now()), updated_at = now()
		WHERE id = $1`, id, nationalID, passwordHash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateCustomer cập nhật thông tin (chỉ manager — enforce ở handler).
func (s *Store) UpdateCustomer(ctx context.Context, id int64, fullName, phone, nationalID string) (Customer, error) {
	c, err := scanCustomer(s.pool.QueryRow(ctx, `
		UPDATE customers SET full_name = $2, phone = $3, national_id = $4, updated_at = now()
		WHERE id = $1
		RETURNING `+customerCols, id, fullName, phone, nationalID))
	return c, mapNotFound(err)
}

// DeleteCustomer xoá khách (chỉ dev — enforce ở handler).
// Nếu khách ĐÃ có giao dịch bán xe (FK sales NO ACTION) -> chỉ ngưng hoạt động để giữ lịch sử.
// Nếu chưa có giao dịch -> xoá hẳn (các bảng đăng ký/voucher/lượt quay... tự cascade).
func (s *Store) DeleteCustomer(ctx context.Context, id int64) (hard bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// thu hồi mọi refresh token của khách này.
	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now()
		WHERE subject_type = 'customer' AND subject_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return false, err
	}

	// có giao dịch bán xe -> không xoá hẳn được (giữ lịch sử doanh thu).
	var sales int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM sales WHERE customer_id = $1`, id).Scan(&sales); err != nil {
		return false, err
	}
	if sales > 0 {
		ct, err := tx.Exec(ctx, `UPDATE customers SET is_active = false, updated_at = now() WHERE id = $1`, id)
		if err != nil {
			return false, err
		}
		if ct.RowsAffected() == 0 {
			return false, ErrNotFound
		}
		return false, tx.Commit(ctx)
	}

	// chưa có giao dịch -> xoá hẳn (cascade: rank_history, event_registrations, event_spins,
	// event_winners, customer_vouchers).
	ct, err := tx.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	if ct.RowsAffected() == 0 {
		return false, ErrNotFound
	}
	return true, tx.Commit(ctx)
}
