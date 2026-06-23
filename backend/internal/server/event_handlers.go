package server

import (
	"crypto/rand"
	"encoding/binary"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kg-cdl/backend/internal/store"
)

func (s *Server) handleRegisterEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	actor, _ := identity(r)
	if err := s.store.Register(r.Context(), id, actor.SubjectID); err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

// handleMyRegistration: khách kiểm tra mình đã đăng ký sự kiện chưa.
func (s *Server) handleMyRegistration(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	actor, _ := identity(r)
	reg, err := s.store.IsRegistered(r.Context(), id, actor.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"registered": reg})
}

func (s *Server) handleSpin(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	actor, _ := identity(r)
	sp, err := s.store.Spin(r.Context(), id, actor.SubjectID, secureFloat())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sp)
}

type createEventReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Prizes      []struct {
		Name     string `json:"name"`
		ImageURL string `json:"image_url"`
		Weight   int    `json:"weight"`
		Stock    *int   `json:"stock"`
	} `json:"prizes"`
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, http.StatusBadRequest, "cần tiêu đề sự kiện")
		return
	}
	if req.Type != "lucky_wheel" && req.Type != "discount_campaign" {
		writeErr(w, http.StatusBadRequest, "loại sự kiện không hợp lệ")
		return
	}
	if req.Type == "lucky_wheel" && len(req.Prizes) == 0 {
		writeErr(w, http.StatusBadRequest, "vòng quay cần ít nhất 1 ô thưởng")
		return
	}
	prizes := make([]store.EventPrize, 0, len(req.Prizes))
	for _, p := range req.Prizes {
		if strings.TrimSpace(p.Name) == "" || p.Weight < 0 {
			writeErr(w, http.StatusBadRequest, "ô thưởng không hợp lệ")
			return
		}
		prizes = append(prizes, store.EventPrize{Name: p.Name, ImageURL: p.ImageURL, Weight: p.Weight, Stock: p.Stock})
	}
	actor, _ := identity(r)
	e, err := s.store.CreateEvent(r.Context(), req.Title, req.Description, req.Type, prizes, actor.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// ── voucher (quản lý tạo, nhân viên xem) ─────────────────────────

func (s *Server) handleListVouchers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListVouchers(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.Voucher{}
	}
	writeJSON(w, http.StatusOK, items)
}

type createVoucherReq struct {
	Name            string     `json:"name"`
	DiscountPercent float64    `json:"discount_percent"`
	MaxAmount       float64    `json:"max_amount"` // 0 = tối đa = toàn bộ giá trị xe
	Quantity        int        `json:"quantity"`
	ExpiresAt       *time.Time `json:"expires_at"`
	AppliesToAll    bool       `json:"applies_to_all"`
	MinRank         string     `json:"min_rank"`
	VehicleIDs      []int64    `json:"vehicle_ids"`
}

func (s *Server) handleCreateVoucher(w http.ResponseWriter, r *http.Request) {
	var req createVoucherReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.DiscountPercent <= 0 || req.DiscountPercent > 100 || req.MaxAmount < 0 {
		writeErr(w, http.StatusBadRequest, "cần tên, % giảm trong (0,100], mức tối đa ≥ 0")
		return
	}
	if req.Quantity < 1 {
		writeErr(w, http.StatusBadRequest, "cần số lượng voucher ≥ 1")
		return
	}
	if req.ExpiresAt == nil {
		writeErr(w, http.StatusBadRequest, "cần chọn hạn sử dụng")
		return
	}
	if !req.AppliesToAll && len(req.VehicleIDs) == 0 {
		writeErr(w, http.StatusBadRequest, "cần chọn ít nhất 1 xe áp dụng (hoặc chọn áp dụng mọi xe)")
		return
	}
	rank := req.MinRank
	if rank != "regular" && rank != "vip" && rank != "svip" {
		rank = "regular"
	}
	actor, _ := identity(r)
	v, err := s.store.CreateVoucher(r.Context(), store.VoucherInput{
		Name: req.Name, DiscountPercent: req.DiscountPercent, MaxAmount: req.MaxAmount,
		Quantity: req.Quantity, ExpiresAt: *req.ExpiresAt, AppliesToAll: req.AppliesToAll,
		MinRank: rank, VehicleIDs: req.VehicleIDs,
	}, actor.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "voucher.create", "voucher", v.ID, jsonObj("name", v.Name))
	writeJSON(w, http.StatusCreated, v)
}

type cancelVoucherReq struct {
	Reason string `json:"reason"`
}

// handleCancelVoucher (chỉ quản lý): huỷ voucher + thu hồi các bản chưa dùng của khách.
func (s *Server) handleCancelVoucher(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req cancelVoucherReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeErr(w, http.StatusBadRequest, "bắt buộc nhập lý do huỷ voucher")
		return
	}
	actor, _ := identity(r)
	if err := s.store.CancelVoucher(r.Context(), id, actor.SubjectID, s.actorName(r.Context(), actor), strings.TrimSpace(req.Reason)); err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ── sự kiện quay số trúng thưởng ─────────────────────────────────

type createDrawReq struct {
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	RegisterDeadline time.Time `json:"register_deadline"`
	VoucherID        *int64    `json:"voucher_id"`
	WinnersCount     int       `json:"winners_count"`
}

func (s *Server) handleCreateDrawEvent(w http.ResponseWriter, r *http.Request) {
	var req createDrawReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" || req.WinnersCount < 1 {
		writeErr(w, http.StatusBadRequest, "cần tiêu đề và số người trúng ≥ 1")
		return
	}
	if req.RegisterDeadline.IsZero() {
		writeErr(w, http.StatusBadRequest, "cần hạn đăng ký")
		return
	}
	// thưởng chỉ là voucher
	if req.VoucherID == nil {
		writeErr(w, http.StatusBadRequest, "cần chọn voucher làm phần thưởng")
		return
	}
	actor, _ := identity(r)
	e, err := s.store.CreateDrawEvent(r.Context(), req.Title, req.Description, req.RegisterDeadline,
		"voucher", req.VoucherID, nil, req.WinnersCount, actor.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// handleEventEntrants: danh sách khách đã đăng ký (cho vòng quay của quản lý).
func (s *Server) handleEventEntrants(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	items, err := s.store.ListEventEntrants(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleDrawRun(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	actor, _ := identity(r)
	winners, err := s.store.DrawWinners(r.Context(), id, actor.SubjectID, s.actorName(r.Context(), actor))
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, winners)
}

type redrawReq struct {
	Reason string `json:"reason"`
}

func (s *Server) handleDrawRedraw(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req redrawReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeErr(w, http.StatusBadRequest, "bắt buộc nhập lý do quay lại")
		return
	}
	actor, _ := identity(r)
	winners, err := s.store.RedrawWinners(r.Context(), id, actor.SubjectID, s.actorName(r.Context(), actor), strings.TrimSpace(req.Reason))
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, winners)
}

func (s *Server) handleDrawConfirm(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	actor, _ := identity(r)
	winners, err := s.store.ConfirmDraw(r.Context(), id, actor.SubjectID, s.actorName(r.Context(), actor))
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, winners)
}

// handleMyPrizes: khách xem TẤT CẢ voucher + xe tặng của CHÍNH MÌNH kèm trạng thái + nhân viên áp dụng.
func (s *Server) handleMyPrizes(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity(r)
	vouchers, err := s.store.ListAllCustomerVouchers(r.Context(), actor.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vouchers": vouchers})
}

// handleCustomerPrizes trả voucher khả dụng của 1 khách CHO XE đang chọn (cho màn bán xe).
func (s *Server) handleCustomerPrizes(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var catalogID int64
	if c, err := strconv.ParseInt(r.URL.Query().Get("catalog_id"), 10, 64); err == nil {
		catalogID = c
	}
	vouchers, err := s.store.ListCustomerVouchers(r.Context(), id, catalogID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vouchers": vouchers})
}

// secureFloat trả về số ngẫu nhiên [0,1) dùng crypto/rand (chống gian lận quay số).
func secureFloat() float64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	u := binary.BigEndian.Uint64(b[:]) >> 11 // 53 bit
	return float64(u) / float64(1<<53)
}
