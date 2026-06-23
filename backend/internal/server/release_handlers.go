package server

import (
	"net/http"
	"strings"
	"time"

	"kg-cdl/backend/internal/store"
)

// nextSaturday21 trả về mốc thứ Bảy 21:00 (9 giờ tối) kế tiếp theo múi giờ loc.
// Nếu hiện tại là thứ Bảy và chưa tới 21:00 -> chính hôm nay; nếu đã qua -> thứ Bảy tuần sau.
func nextSaturday21(now time.Time, loc *time.Location) time.Time {
	n := now.In(loc)
	days := (int(time.Saturday) - int(n.Weekday()) + 7) % 7
	cand := time.Date(n.Year(), n.Month(), n.Day(), 21, 0, 0, 0, loc).AddDate(0, 0, days)
	if !cand.After(n) {
		cand = cand.AddDate(0, 0, 7)
	}
	return cand
}

func (s *Server) releaseLoc() *time.Location {
	loc, err := time.LoadLocation(s.cfg.Timezone)
	if err != nil || loc == nil {
		return time.UTC
	}
	return loc
}

// handleReleaseInfo (công khai): trả mốc mở bán xe kế tiếp cho countdown.
// release_at = override của quản lý nếu còn hiệu lực (chưa qua), ngược lại = mặc định thứ 7 21:00.
func (s *Server) handleReleaseInfo(w http.ResponseWriter, r *http.Request) {
	loc := s.releaseLoc()
	now := time.Now()
	def := nextSaturday21(now, loc)

	releaseAt := def
	overridden := false
	if ov, _ := s.store.GetReleaseOverride(r.Context()); ov != nil && ov.After(now) {
		releaseAt = ov.In(loc)
		overridden = true
	}
	mc, _ := s.store.GetModalConfig(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"release_at":   releaseAt,
		"default_at":   def,
		"overridden":   overridden,
		"modal_image":  mc.Image,
		"modal_target": mc.Target,
	})
}

type setReleaseModalReq struct {
	Image  string `json:"image"`
	Target string `json:"target"`
}

// handleSetReleaseModal (chỉ quản lý): cập nhật ảnh + đích chuyển hướng của popup mở bán.
func (s *Server) handleSetReleaseModal(w http.ResponseWriter, r *http.Request) {
	var req setReleaseModalReq
	if !readJSON(w, r, &req) {
		return
	}
	cfg := store.ModalConfig{Image: strings.TrimSpace(req.Image), Target: strings.TrimSpace(req.Target)}
	if err := s.store.SetModalConfig(r.Context(), cfg); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "release.modal", "settings", 0, jsonObj2("image", cfg.Image, "target", cfg.Target))
	writeJSON(w, http.StatusOK, map[string]string{"image": cfg.Image, "target": cfg.Target})
}

type setReleaseReq struct {
	ReleaseAt string `json:"release_at"` // RFC3339, vd 2026-06-27T21:00:00+07:00
	Reset     bool   `json:"reset"`      // true -> xoá override, về mặc định
}

// handleSetReleaseOverride (chỉ quản lý): đặt/đổi mốc countdown cho tuần hiện tại, hoặc reset về mặc định.
func (s *Server) handleSetReleaseOverride(w http.ResponseWriter, r *http.Request) {
	var req setReleaseReq
	if !readJSON(w, r, &req) {
		return
	}
	actor, _ := identity(r)
	if req.Reset {
		if err := s.store.ClearReleaseOverride(r.Context()); err != nil {
			handleStoreErr(w, err)
			return
		}
		_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "release.reset", "settings", 0, nil)
		writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
		return
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ReleaseAt))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "thời điểm không hợp lệ")
		return
	}
	if !t.After(time.Now()) {
		writeErr(w, http.StatusBadRequest, "thời điểm mở bán phải ở tương lai")
		return
	}
	if err := s.store.SetReleaseOverride(r.Context(), t); err != nil {
		handleStoreErr(w, err)
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "release.override", "settings", 0, jsonObj("release_at", t.Format(time.RFC3339)))
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "release_at": t})
}
