package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"kg-cdl/backend/internal/auth"
	"kg-cdl/backend/internal/store"
)

// nationalIDPattern: căn cước khách hàng = "LUX" (in hoa) + đúng 5 chữ số (vd LUX12345).
var nationalIDPattern = regexp.MustCompile(`^LUX[0-9]{5}$`)

func validNationalID(s string) bool { return nationalIDPattern.MatchString(s) }

// normalizeNationalID chuẩn hóa căn cước: bỏ khoảng trắng + in hoa toàn bộ (đảm bảo "LUX...").
func normalizeNationalID(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// requireExistingSubject từ chối (401) khi token còn hạn nhưng tài khoản đã bị xoá/khoá,
// tránh lỗi 500 do khóa ngoại khi thao tác. Frontend sẽ tự đăng xuất khi gặp 401.
func (s *Server) requireExistingSubject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := auth.FromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		exists, active, validAfter, err := s.store.SubjectAuthState(r.Context(), id.SubjectType, id.SubjectID)
		if err != nil || !exists || !active {
			// chủ thể đã bị xoá hoặc khoá/ngưng -> đăng xuất ngay
			writeErr(w, http.StatusUnauthorized, "tài khoản không tồn tại hoặc đã bị khoá, vui lòng đăng nhập lại")
			return
		}
		if validAfter != nil && id.IssuedAt.Before(*validAfter) {
			// phiên đã bị vô hiệu (vd: bị thăng cấp/đổi quyền/ngưng) -> buộc đăng nhập lại
			writeErr(w, http.StatusUnauthorized, "phiên đăng nhập đã hết hiệu lực, vui lòng đăng nhập lại")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// readJSON đọc body JSON vào dst; trả false (và đã ghi lỗi) nếu hỏng.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func urlID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func identity(r *http.Request) (auth.Identity, bool) {
	return auth.FromContext(r.Context())
}

// actorName lấy tên hiển thị của nhân viên hiện tại (cho log).
func (s *Server) actorName(ctx context.Context, id auth.Identity) string {
	if id.SubjectType != auth.SubjectUser {
		return ""
	}
	u, err := s.store.GetUserByID(ctx, id.SubjectID)
	if err != nil {
		return ""
	}
	return u.DisplayName
}

// handleStoreErr ánh xạ lỗi store sang HTTP code.
func handleStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrOutOfStock):
		writeErr(w, http.StatusConflict, "out of stock")
	case errors.Is(err, store.ErrAlreadyRefunded):
		writeErr(w, http.StatusConflict, "giao dịch này đã được hoàn trước đó")
	case errors.Is(err, store.ErrAlreadyCancelled):
		writeErr(w, http.StatusConflict, "voucher này đã bị huỷ trước đó")
	case errors.Is(err, store.ErrBookingClosed):
		writeErr(w, http.StatusConflict, "xe này hiện không nhận đặt lịch")
	case errors.Is(err, store.ErrBookingDuplicate):
		writeErr(w, http.StatusConflict, "bạn đã có lịch hẹn cho xe này (chưa tới ngày xem), vui lòng chờ qua ngày hẹn rồi đặt lại")
	case errors.Is(err, store.ErrBookingHandled):
		writeErr(w, http.StatusConflict, "lịch đặt này đã được xử lý trước đó")
	case errors.Is(err, store.ErrNoSpins):
		writeErr(w, http.StatusConflict, "no spins remaining")
	case errors.Is(err, store.ErrNotRegistered):
		writeErr(w, http.StatusForbidden, "not registered for this event")
	case errors.Is(err, store.ErrNotDrawEvent):
		writeErr(w, http.StatusBadRequest, "không phải sự kiện quay số")
	case errors.Is(err, store.ErrBadDrawState):
		writeErr(w, http.StatusConflict, "trạng thái quay số không hợp lệ (đã quay/đã công bố?)")
	case errors.Is(err, store.ErrEventNotCancelable):
		writeErr(w, http.StatusConflict, "chỉ có thể huỷ sự kiện CHƯA quay số")
	case errors.Is(err, store.ErrNoEligible):
		writeErr(w, http.StatusConflict, "chưa có khách nào đăng ký tham gia")
	case errors.Is(err, store.ErrRegistrationClosed):
		writeErr(w, http.StatusConflict, "đã hết hạn đăng ký tham gia")
	case errors.Is(err, store.ErrInviteOnly):
		writeErr(w, http.StatusForbidden, "sự kiện này chỉ dành cho người được chỉ định, không nhận đăng ký")
	case errors.Is(err, store.ErrPrizeInvalid):
		writeErr(w, http.StatusConflict, "voucher hoặc giải thưởng không hợp lệ với khách này")
	case errors.Is(err, store.ErrVoucherDepleted):
		writeErr(w, http.StatusConflict, "voucher đã hết số lượng")
	case errors.Is(err, store.ErrVoucherAlreadyUsed):
		writeErr(w, http.StatusConflict, "khách đã dùng voucher này rồi, không thể dùng lại")
	case errors.Is(err, store.ErrVoucherExpired):
		writeErr(w, http.StatusConflict, "voucher đã hết hạn sử dụng")
	case errors.Is(err, store.ErrVoucherRank):
		writeErr(w, http.StatusConflict, "hạng khách hàng chưa đủ điều kiện dùng voucher này")
	case errors.Is(err, store.ErrVoucherNotApplicable):
		writeErr(w, http.StatusConflict, "voucher không áp dụng cho mẫu xe này")
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}
