package server

import (
	"fmt"
	"net/http"
	"time"

	"kg-cdl/backend/internal/store"
)

// ── popup: tải ảnh từ máy lên (ghi đè tên cố định release-popup.jpg) ─

func (s *Server) handlePopupUpload(w http.ResponseWriter, r *http.Request) {
	if err := s.saveUpload(r, "file", "release-popup.jpg"); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	url := "/api/uploads/release-popup.jpg"
	// giữ nguyên target hiện tại, chỉ đổi ảnh sang file vừa tải.
	cur, _ := s.store.GetModalConfig(r.Context())
	if err := s.store.SetModalConfig(r.Context(), store.ModalConfig{Image: url, Target: cur.Target}); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "popup.upload", "settings", 0, nil)
	writeJSON(w, http.StatusOK, map[string]string{"image": url})
}

// ── banner trang chủ ───────────────────────────────────────────────

func (s *Server) handleListBannersPublic(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListBanners(r.Context(), true)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListBanners(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListBanners(r.Context(), false)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateBanner(w http.ResponseWriter, r *http.Request) {
	name := fmt.Sprintf("banner-%d.jpg", time.Now().UnixNano())
	if err := s.saveUpload(r, "file", name); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	actor, _ := identity(r)
	b, err := s.store.CreateBanner(r.Context(), name, actor.SubjectID)
	if err != nil {
		s.deleteUpload(name)
		handleStoreErr(w, err)
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "banner.create", "banner", b.ID, nil)
	writeJSON(w, http.StatusCreated, b)
}

type toggleBannerReq struct {
	Active bool `json:"active"`
}

func (s *Server) handleToggleBanner(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req toggleBannerReq
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.store.SetBannerActive(r.Context(), id, req.Active); err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"is_active": req.Active})
}

// handleDeleteBanner (chỉ dev): xoá banner + file ảnh.
func (s *Server) handleDeleteBanner(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	image, err := s.store.DeleteBanner(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	s.deleteUpload(image)
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "banner.delete", "banner", id, nil)
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
