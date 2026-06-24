package server

import (
	"net/http"

	"kg-cdl/backend/internal/store"
)

// handleListVehicles trả về kho theo trạng thái. status=on_sale|upcoming (mặc định on_sale).
// Công khai cho khách xem; đã tính sẵn giá sau giảm + % giảm.
func (s *Server) handleListVehicles(w http.ResponseWriter, r *http.Request) {
	_ = s.store.PromoteDueInventory(r.Context())
	status := r.URL.Query().Get("status")
	switch status {
	case "", "on_sale":
		status = "on_sale"
	case "upcoming", "sold_out":
		// giữ nguyên
	default:
		writeErr(w, http.StatusBadRequest, "status không hợp lệ")
		return
	}
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

func (s *Server) handleGetVehicle(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	it, err := s.store.GetInventory(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// handleSimilarVehicles gợi ý xe đang bán cùng dòng/hãng.
func (s *Server) handleSimilarVehicles(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	items, err := s.store.SimilarOnSale(r.Context(), id, 4)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.InventoryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

// handleVehicleDiscounts lịch sử khuyến mãi của xe.
func (s *Server) handleVehicleDiscounts(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	items, err := s.store.ListDiscounts(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.Discount{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.ListEvents(r.Context(), true)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if events == nil {
		events = []store.Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	e, err := s.store.GetEvent(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	// sự kiện đã huỷ -> coi như không tồn tại (khách không được xem).
	if e.CancelledAt != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, e)
}
