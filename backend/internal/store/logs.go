package store

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// LogFilter các tiêu chí lọc log (đều tùy chọn).
type LogFilter struct {
	Action  string
	ActorID int64
	From    *time.Time
	To      *time.Time
	Limit   int
	Offset  int
}

// InsertLog ghi một dòng log (dùng cho các hành động ngoài transaction nghiệp vụ).
func (s *Store) InsertLog(ctx context.Context, actorID int64, actorName, action, targetType string, targetID int64, detail []byte) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO activity_logs (actor_id, actor_name, action, target_type, target_id, detail)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		actorID, actorName, action, nullStr(targetType), nullID(targetID), nullJSON(detail))
	return err
}

// ListLogs trả về log theo trang (tối đa 100/trang) + tổng số bản ghi khớp lọc.
func (s *Store) ListLogs(ctx context.Context, f LogFilter) ([]ActivityLog, int, error) {
	var where []string
	var args []any
	i := 1
	if f.Action != "" {
		where = append(where, "action = $"+strconv.Itoa(i))
		args = append(args, f.Action)
		i++
	}
	if f.ActorID > 0 {
		where = append(where, "actor_id = $"+strconv.Itoa(i))
		args = append(args, f.ActorID)
		i++
	}
	if f.From != nil {
		where = append(where, "created_at >= $"+strconv.Itoa(i))
		args = append(args, *f.From)
		i++
	}
	if f.To != nil {
		where = append(where, "created_at <= $"+strconv.Itoa(i))
		args = append(args, *f.To)
		i++
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM activity_logs`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	q := `SELECT id, actor_id, COALESCE(actor_name,''), action, target_type, target_id, COALESCE(detail,'{}'::jsonb), created_at
	      FROM activity_logs` + whereSQL +
		" ORDER BY created_at DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []ActivityLog{}
	for rows.Next() {
		var l ActivityLog
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorName, &l.Action, &l.TargetType, &l.TargetID, &l.Detail, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

// DistinctActions trả về danh sách action có trong log (cho dropdown filter).
func (s *Store) DistinctActions(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT action FROM activity_logs ORDER BY action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}
func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
