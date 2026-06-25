package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kg-cdl/backend/internal/auth"
	"kg-cdl/backend/internal/store"
)

// ── catalog ──────────────────────────────────────────────────────

func (s *Server) handleListCatalog(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.store.ListCatalog(r.Context(), search, limit)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.CatalogVehicle{}
	}
	writeJSON(w, http.StatusOK, items)
}

type specsReq struct {
	Seats        *int `json:"seats"`
	TrunkKg      int  `json:"trunk_kg"`
	RateSpeed    int  `json:"rate_speed"`
	RateAccel    int  `json:"rate_accel"`
	RateBraking  int  `json:"rate_braking"`
	RateTraction int  `json:"rate_traction"`
}

// toSpecs chuẩn hóa & kẹp điểm hiệu năng trong [0,100].
func (sp specsReq) toSpecs() store.CatalogSpecs {
	clamp := func(n int) int {
		if n < 0 {
			return 0
		}
		if n > 100 {
			return 100
		}
		return n
	}
	trunk := sp.TrunkKg
	if trunk < 0 {
		trunk = 0
	}
	return store.CatalogSpecs{
		Seats:        sp.Seats,
		TrunkKg:      trunk,
		RateSpeed:    clamp(sp.RateSpeed),
		RateAccel:    clamp(sp.RateAccel),
		RateBraking:  clamp(sp.RateBraking),
		RateTraction: clamp(sp.RateTraction),
	}
}

type createCatalogReq struct {
	ModelCode   string `json:"model_code"`
	Name        string `json:"name"`
	Brand       string `json:"brand"`
	Class       string `json:"class"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
	specsReq
}

func (s *Server) handleCreateCatalog(w http.ResponseWriter, r *http.Request) {
	var req createCatalogReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "cần tên xe")
		return
	}
	if req.TrunkKg <= 0 {
		writeErr(w, http.StatusBadRequest, "cần nhập cốp xe (kg) lớn hơn 0")
		return
	}
	id, _ := identity(r)
	var code *string
	if c := strings.TrimSpace(req.ModelCode); c != "" {
		code = &c
	}
	v, err := s.store.CreateCatalog(r.Context(), code, req.Name, req.Brand, req.Class, req.ImageURL, req.Description, req.toSpecs(), true, id.SubjectID)
	if err != nil {
		writeErr(w, http.StatusConflict, "không tạo được (model_code có thể trùng)")
		return
	}
	_ = s.store.InsertLog(r.Context(), id.SubjectID, s.actorName(r.Context(), id), "catalog.create", "catalog", v.ID, jsonObj("name", v.Name))
	writeJSON(w, http.StatusCreated, v)
}

type updateCatalogReq struct {
	Description string `json:"description"`
	specsReq
}

// handleUpdateCatalog cho quản lý sửa giới thiệu + thông số xe (chỉ manager/dev).
func (s *Server) handleUpdateCatalog(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req updateCatalogReq
	if !readJSON(w, r, &req) {
		return
	}
	v, err := s.store.UpdateCatalogInfo(r.Context(), id, req.Description, req.toSpecs())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "catalog.update", "catalog", v.ID, jsonObj("name", v.Name))
	writeJSON(w, http.StatusOK, v)
}

// ── tuần mở bán ──────────────────────────────────────────────────

func (s *Server) handleListSalesWeeks(w http.ResponseWriter, r *http.Request) {
	weeks, err := s.store.ListSalesWeeks(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, weeks)
}

type createWeekReq struct {
	Date string `json:"date"` // ngày bất kỳ trong tuần (YYYY-MM-DD)
}

func (s *Server) handleCreateSalesWeek(w http.ResponseWriter, r *http.Request) {
	var req createWeekReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Date) == "" {
		writeErr(w, http.StatusBadRequest, "cần chọn ngày của tuần")
		return
	}
	actor, _ := identity(r)
	wk, err := s.store.CreateSalesWeek(r.Context(), req.Date, actor.SubjectID)
	if err != nil {
		if err == store.ErrWeekPast {
			writeErr(w, http.StatusBadRequest, "không thể đăng ký tuần đã qua")
			return
		}
		writeErr(w, http.StatusBadRequest, "không đăng ký được tuần (ngày không hợp lệ?)")
		return
	}
	writeJSON(w, http.StatusCreated, wk)
}

// ── inventory ────────────────────────────────────────────────────

func (s *Server) handleListInventory(w http.ResponseWriter, r *http.Request) {
	_, _ = s.store.PromoteDueInventory(r.Context(), false)
	status := r.URL.Query().Get("status")
	items, err := s.store.ListInventory(r.Context(), status)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.InventoryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

type createInventoryReq struct {
	CatalogID   int64      `json:"catalog_id"`
	BasePrice   float64    `json:"base_price"`
	Quantity    int        `json:"quantity"`
	Status      string     `json:"status"`
	OnSaleAt    *time.Time `json:"on_sale_at"`
	Note        string     `json:"note"`
	SalesWeekID *int64     `json:"sales_week_id"`
}

func (s *Server) handleCreateInventory(w http.ResponseWriter, r *http.Request) {
	var req createInventoryReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.CatalogID <= 0 || req.BasePrice < 0 || req.Quantity < 0 {
		writeErr(w, http.StatusBadRequest, "dữ liệu nhập kho không hợp lệ")
		return
	}
	id, _ := identity(r)

	var invID int64
	var err error
	if req.SalesWeekID != nil {
		// nhập theo tuần mở bán: trạng thái tự suy theo tuần
		invID, err = s.store.CreateInventoryForWeek(r.Context(), req.CatalogID, req.BasePrice, req.Quantity, req.Note, *req.SalesWeekID, id.SubjectID)
	} else {
		// đường cũ: trạng thái thủ công
		if req.Status == "" {
			req.Status = "upcoming"
		}
		if !validStatus(req.Status) {
			writeErr(w, http.StatusBadRequest, "status không hợp lệ")
			return
		}
		invID, err = s.store.CreateInventory(r.Context(), req.CatalogID, req.BasePrice, req.Quantity, req.Status, req.OnSaleAt, req.Note, id.SubjectID)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, "không nhập kho được (mẫu xe hoặc tuần không tồn tại?)")
		return
	}
	_ = s.store.InsertLog(r.Context(), id.SubjectID, s.actorName(r.Context(), id), "inventory.add", "inventory", invID, jsonObj2("quantity", req.Quantity, "base_price", req.BasePrice))
	it, _ := s.store.GetInventory(r.Context(), invID)
	writeJSON(w, http.StatusCreated, it)
}

type setDiscountReq struct {
	Percent float64    `json:"percent"`
	EndsAt  *time.Time `json:"ends_at"`
}

func (s *Server) handleSetDiscount(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req setDiscountReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.Percent <= 0 || req.Percent > 100 {
		writeErr(w, http.StatusBadRequest, "phần trăm giảm phải trong (0,100]")
		return
	}
	actor, _ := identity(r)
	if err := s.store.SetDiscount(r.Context(), id, req.Percent, req.EndsAt, actor.SubjectID); err != nil {
		handleStoreErr(w, err)
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "inventory.discount", "inventory", id, jsonObj("percent", req.Percent))
	it, _ := s.store.GetInventory(r.Context(), id)
	writeJSON(w, http.StatusOK, it)
}

type updateStatusReq struct {
	Status string `json:"status"`
}

func (s *Server) handleUpdateInventoryStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req updateStatusReq
	if !readJSON(w, r, &req) {
		return
	}
	if !validStatus(req.Status) {
		writeErr(w, http.StatusBadRequest, "status không hợp lệ")
		return
	}
	if err := s.store.UpdateInventoryStatus(r.Context(), id, req.Status); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "inventory.status", "inventory", id, jsonObj("status", req.Status))
	it, _ := s.store.GetInventory(r.Context(), id)
	writeJSON(w, http.StatusOK, it)
}

type updatePriceReq struct {
	BasePrice float64 `json:"base_price"`
}

// handleUpdateInventoryPrice (chỉ quản lý): sửa giá bán gốc của một dòng kho.
func (s *Server) handleUpdateInventoryPrice(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req updatePriceReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.BasePrice <= 0 {
		writeErr(w, http.StatusBadRequest, "giá bán phải lớn hơn 0")
		return
	}
	if err := s.store.UpdateInventoryPrice(r.Context(), id, req.BasePrice); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "inventory.price", "inventory", id, jsonObj("base_price", req.BasePrice))
	it, _ := s.store.GetInventory(r.Context(), id)
	writeJSON(w, http.StatusOK, it)
}

// ── customers ────────────────────────────────────────────────────

func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListCustomers(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.Customer{}
	}
	writeJSON(w, http.StatusOK, items)
}

type createCustomerReq struct {
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	NationalID string `json:"national_id"`
	Gender     string `json:"gender"`
	BirthDate  string `json:"birth_date"` // YYYY-MM-DD
}

// cleanGender chuẩn hoá giới tính về 'male'|'female'|'other', rỗng/không hợp lệ -> nil.
func cleanGender(g string) *string {
	g = strings.ToLower(strings.TrimSpace(g))
	switch g {
	case "male", "female", "other":
		return &g
	default:
		return nil
	}
}

// cleanBirthDate kiểm tra định dạng YYYY-MM-DD -> *string; rỗng -> nil (hợp lệ); sai định dạng -> ok=false.
func cleanBirthDate(d string) (*string, bool) {
	d = strings.TrimSpace(d)
	if d == "" {
		return nil, true
	}
	if _, err := time.Parse("2006-01-02", d); err != nil {
		return nil, false
	}
	return &d, true
}

func (s *Server) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerReq
	if !readJSON(w, r, &req) {
		return
	}
	req.NationalID = normalizeNationalID(req.NationalID)
	if strings.TrimSpace(req.FullName) == "" || req.NationalID == "" {
		writeErr(w, http.StatusBadRequest, "cần tên và số căn cước")
		return
	}
	if !validNationalID(req.NationalID) {
		writeErr(w, http.StatusBadRequest, "số căn cước phải có dạng LUX + 5 chữ số (vd LUX12345)")
		return
	}
	birth, okB := cleanBirthDate(req.BirthDate)
	if !okB {
		writeErr(w, http.StatusBadRequest, "ngày sinh không hợp lệ (định dạng YYYY-MM-DD)")
		return
	}
	actor, _ := identity(r)
	c, err := s.store.CreateCustomerByStaff(r.Context(), req.FullName, req.Phone, req.NationalID, cleanGender(req.Gender), birth, actor.SubjectID)
	if err != nil {
		writeErr(w, http.StatusConflict, "số căn cước đã tồn tại")
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "customer.create", "customer", c.ID, jsonObj("name", c.FullName))
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req createCustomerReq
	if !readJSON(w, r, &req) {
		return
	}
	req.NationalID = normalizeNationalID(req.NationalID)
	if !validNationalID(req.NationalID) {
		writeErr(w, http.StatusBadRequest, "số căn cước phải có dạng LUX + 5 chữ số (vd LUX12345)")
		return
	}
	birth, okB := cleanBirthDate(req.BirthDate)
	if !okB {
		writeErr(w, http.StatusBadRequest, "ngày sinh không hợp lệ (định dạng YYYY-MM-DD)")
		return
	}
	actor, _ := identity(r)
	c, err := s.store.UpdateCustomer(r.Context(), id, req.FullName, req.Phone, req.NationalID, cleanGender(req.Gender), birth)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "customer.update", "customer", c.ID, jsonObj("name", c.FullName))
	writeJSON(w, http.StatusOK, c)
}

// handleResetCustomerPassword (chỉ dev): đặt lại mật khẩu khách về CHÍNH số căn cước của họ.
func (s *Server) handleResetCustomerPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	c, err := s.store.GetCustomerByID(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	hash, err := auth.HashPassword(c.NationalID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.store.ResetCustomerLogin(r.Context(), id, c.NationalID, hash); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "customer.reset_password", "customer", id, jsonObj("national_id", c.NationalID))
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset", "national_id": c.NationalID})
}

// handleDeleteCustomer (chỉ dev): xoá khách. Có giao dịch -> ngưng hoạt động; chưa có -> xoá hẳn.
func (s *Server) handleDeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	c, err := s.store.GetCustomerByID(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	hard, err := s.store.DeleteCustomer(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	// xoá/ngưng khách -> vô hiệu mọi phiên hiện tại của họ (đăng xuất ngay)
	_ = s.store.InvalidateSessions(r.Context(), auth.SubjectCustomer, id)
	action := "customer.delete"
	if !hard {
		action = "customer.deactivate"
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), action, "customer", id, jsonObj("name", c.FullName))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "hard": hard})
}

// handleCustomerSales: lịch sử xe khách đã mua (nhân viên/quản lý xem).
func (s *Server) handleCustomerSales(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	items, err := s.store.ListCustomerSales(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ── sales ────────────────────────────────────────────────────────

func (s *Server) handleListSales(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.store.ListSales(r.Context(), limit)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.Sale{}
	}
	writeJSON(w, http.StatusOK, items)
}

type createSaleReq struct {
	InventoryID       int64    `json:"inventory_id"`
	CustomerID        int64    `json:"customer_id"`
	CustomerVoucherID *int64   `json:"customer_voucher_id"`
	OverridePrice     *float64 `json:"override_price"` // giá bán tuỳ chỉnh cho phiên này
}

func (s *Server) handleCreateSale(w http.ResponseWriter, r *http.Request) {
	var req createSaleReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.InventoryID <= 0 || req.CustomerID <= 0 {
		writeErr(w, http.StatusBadRequest, "cần chọn xe và khách hàng")
		return
	}
	actor, _ := identity(r)
	limits, _ := s.store.GetRankLimits(r.Context())
	res, err := s.store.SellVehicle(r.Context(), req.InventoryID, req.CustomerID, actor.SubjectID, s.actorName(r.Context(), actor), limits.SVIP, limits.VIP,
		store.SellOptions{CustomerVoucherID: req.CustomerVoucherID, OverridePrice: req.OverridePrice})
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"sale":            res.Sale,
		"new_rank":        res.NewRank,
		"rank_changed_to": res.RankChangedTo,
	})
}

type refundSaleReq struct {
	Reason string `json:"reason"`
}

// handleRefundSale (chỉ quản lý): hoàn trả giao dịch bán sai.
func (s *Server) handleRefundSale(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req refundSaleReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeErr(w, http.StatusBadRequest, "bắt buộc nhập lý do hoàn trả")
		return
	}
	actor, _ := identity(r)
	limits, _ := s.store.GetRankLimits(r.Context())
	if err := s.store.RefundSale(r.Context(), id, actor.SubjectID, s.actorName(r.Context(), actor), strings.TrimSpace(req.Reason), limits.SVIP, limits.VIP); err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "refunded"})
}

type transferSaleReq struct {
	CustomerID int64 `json:"customer_id"`
}

// handleTransferSale (quản lý): chuyển 1 giao dịch của tài khoản tạm (vd LUX00000) sang khách thật.
func (s *Server) handleTransferSale(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req transferSaleReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.CustomerID <= 0 {
		writeErr(w, http.StatusBadRequest, "cần chọn khách hàng nhận")
		return
	}
	actor, _ := identity(r)
	limits, _ := s.store.GetRankLimits(r.Context())
	if err := s.store.TransferSale(r.Context(), id, req.CustomerID, actor.SubjectID, s.actorName(r.Context(), actor), limits.SVIP, limits.VIP); err != nil {
		switch err {
		case store.ErrNotTransferable:
			writeErr(w, http.StatusBadRequest, "chỉ chuyển được giao dịch CHƯA hoàn của tài khoản tạm")
		case store.ErrInvalidTransferTarget:
			writeErr(w, http.StatusBadRequest, "khách hàng nhận không hợp lệ (không tồn tại, đã ngưng, hoặc cũng là tài khoản tạm)")
		default:
			handleStoreErr(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "transferred"})
}

// ── báo cáo doanh thu ────────────────────────────────────────────

func (s *Server) handleRevenueReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.store.RevenueReport(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// ── logs ─────────────────────────────────────────────────────────

func (s *Server) handleListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.LogFilter{Action: q.Get("action")}
	if a, err := strconv.ParseInt(q.Get("actor_id"), 10, 64); err == nil {
		f.ActorID = a
	}
	if t, err := time.Parse(time.RFC3339, q.Get("from")); err == nil {
		f.From = &t
	}
	if t, err := time.Parse(time.RFC3339, q.Get("to")); err == nil {
		f.To = &t
	}
	if l, err := strconv.Atoi(q.Get("limit")); err == nil {
		f.Limit = l
	}
	if o, err := strconv.Atoi(q.Get("offset")); err == nil {
		f.Offset = o
	}
	logs, total, err := s.store.ListLogs(r.Context(), f)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": logs, "total": total})
}

func (s *Server) handleLogActions(w http.ResponseWriter, r *http.Request) {
	actions, err := s.store.DistinctActions(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if actions == nil {
		actions = []string{}
	}
	writeJSON(w, http.StatusOK, actions)
}

// ── helpers ──────────────────────────────────────────────────────

func validStatus(s string) bool {
	switch s {
	case "upcoming", "on_sale", "hidden", "sold_out":
		return true
	}
	return false
}

func jsonObj(k string, v any) []byte {
	b, _ := json.Marshal(map[string]any{k: v})
	return b
}
func jsonObj2(k1 string, v1 any, k2 string, v2 any) []byte {
	b, _ := json.Marshal(map[string]any{k1: v1, k2: v2})
	return b
}
