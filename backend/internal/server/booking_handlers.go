package server

import (
	"net/http"
	"strings"
	"time"

	"kg-cdl/backend/internal/store"
)

// ── quản lý: mở/đóng nhận đặt lịch cho 1 mục kho ───────────────────

type setBookingReq struct {
	Open bool `json:"open"`
}

func (s *Server) handleSetBookingOpen(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req setBookingReq
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.store.SetBookingOpen(r.Context(), id, req.Open); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	action := "booking.open"
	if !req.Open {
		action = "booking.close"
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), action, "inventory", id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"booking_open": req.Open})
}

// ── khách hàng: tạo lịch + xem lịch của mình ───────────────────────

type createBookingReq struct {
	InventoryID int64  `json:"inventory_id"`
	VisitDate   string `json:"visit_date"` // YYYY-MM-DD
	Note        string `json:"note"`
}

func (s *Server) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	var req createBookingReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.InventoryID <= 0 {
		writeErr(w, http.StatusBadRequest, "thiếu thông tin xe")
		return
	}
	req.VisitDate = strings.TrimSpace(req.VisitDate)
	t, err := time.Parse("2006-01-02", req.VisitDate)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "ngày hẹn không hợp lệ")
		return
	}
	loc, _ := time.LoadLocation(s.cfg.Timezone)
	if loc == nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
	if t.Format("2006-01-02") < today {
		writeErr(w, http.StatusBadRequest, "ngày hẹn phải từ hôm nay trở đi")
		return
	}
	actor, _ := identity(r)
	b, err := s.store.CreateBooking(r.Context(), req.InventoryID, actor.SubjectID, req.VisitDate, strings.TrimSpace(req.Note))
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleMyBookings(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity(r)
	items, err := s.store.ListCustomerBookings(r.Context(), actor.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bookings": items})
}

// ── nhân viên/quản lý: danh sách + xử lý lịch ──────────────────────

func (s *Server) handleListBookings(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", "pending", "accepted", "rejected":
		// ok
	default:
		writeErr(w, http.StatusBadRequest, "trạng thái không hợp lệ")
		return
	}
	items, err := s.store.ListBookings(r.Context(), status)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.Booking{}
	}
	writeJSON(w, http.StatusOK, items)
}

type handleBookingReq struct {
	Status string `json:"status"` // accepted | rejected
}

func (s *Server) handleHandleBooking(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req handleBookingReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.Status != "accepted" && req.Status != "rejected" {
		writeErr(w, http.StatusBadRequest, "trạng thái phải là accepted hoặc rejected")
		return
	}
	actor, _ := identity(r)
	if err := s.store.HandleBooking(r.Context(), id, req.Status, actor.SubjectID); err != nil {
		handleStoreErr(w, err)
		return
	}
	action := "booking.accept"
	if req.Status == "rejected" {
		action = "booking.reject"
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), action, "booking", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}
