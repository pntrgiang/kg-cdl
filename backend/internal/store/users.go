package store

import "context"

const userCols = `id, username, display_name, role, national_id, is_active, created_at`

// CreateUser tạo nhân viên mới (role do dev set).
func (s *Store) CreateUser(ctx context.Context, username, passwordHash, displayName, role string, nationalID *string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, display_name, role, national_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userCols,
		username, passwordHash, displayName, role, nationalID,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.NationalID, &u.IsActive, &u.CreatedAt)
	return u, err
}

// GetUserByUsername trả về user + password_hash để xác thực.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, string, error) {
	var u User
	var hash string
	err := s.pool.QueryRow(ctx, `
		SELECT `+userCols+`, password_hash
		FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.NationalID, &u.IsActive, &u.CreatedAt, &hash)
	return u, hash, mapNotFound(err)
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT `+userCols+` FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.NationalID, &u.IsActive, &u.CreatedAt)
	return u, mapNotFound(err)
}

// GetUserPasswordHash lấy hash mật khẩu theo id (để xác thực khi đổi mật khẩu).
func (s *Store) GetUserPasswordHash(ctx context.Context, id int64) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash)
	return hash, mapNotFound(err)
}

// UpdateUserPassword đổi mật khẩu của chính nhân viên.
func (s *Store) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userCols+` FROM users WHERE is_active ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.NationalID, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateUserRole đổi role (chỉ dev được gọi — enforce ở handler).
func (s *Store) UpdateUserRole(ctx context.Context, id int64, role string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		UPDATE users SET role = $2, updated_at = now()
		WHERE id = $1
		RETURNING `+userCols, id, role,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.NationalID, &u.IsActive, &u.CreatedAt)
	return u, mapNotFound(err)
}

// DeleteUser xoá nhân viên. Nếu đã có giao dịch/sự kiện -> vô hiệu hoá (giữ lịch sử),
// còn không -> xoá hẳn. Trả về hard=true nếu xoá hẳn.
func (s *Store) DeleteUser(ctx context.Context, id int64) (hard bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// thu hồi mọi refresh token của user này.
	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now()
		WHERE subject_type = 'user' AND subject_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return false, err
	}

	// có giao dịch bán xe hoặc tạo sự kiện -> không xoá hẳn được (NOT NULL FK).
	var refs int
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM sales WHERE sold_by = $1)
		     + (SELECT count(*) FROM events WHERE created_by = $1)`, id).Scan(&refs); err != nil {
		return false, err
	}

	if refs > 0 {
		ct, err := tx.Exec(ctx, `UPDATE users SET is_active = false, updated_at = now() WHERE id = $1`, id)
		if err != nil {
			return false, err
		}
		if ct.RowsAffected() == 0 {
			return false, ErrNotFound
		}
		return false, tx.Commit(ctx)
	}

	// gỡ các tham chiếu cho phép NULL trước khi xoá.
	for _, q := range []string{
		`UPDATE activity_logs   SET actor_id   = NULL WHERE actor_id   = $1`,
		`UPDATE customers       SET created_by = NULL WHERE created_by = $1`,
		`UPDATE vehicle_catalog SET created_by = NULL WHERE created_by = $1`,
		`UPDATE inventory       SET created_by = NULL WHERE created_by = $1`,
		`UPDATE discounts       SET created_by = NULL WHERE created_by = $1`,
		`UPDATE vouchers        SET created_by = NULL WHERE created_by = $1`,
	} {
		if _, err := tx.Exec(ctx, q, id); err != nil {
			return false, err
		}
	}
	ct, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	if ct.RowsAffected() == 0 {
		return false, ErrNotFound
	}
	return true, tx.Commit(ctx)
}

// CountUsers đếm số nhân viên (dùng để seed dev đầu tiên).
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}
