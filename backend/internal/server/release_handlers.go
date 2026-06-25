package server

import (
	"context"
	"fmt"
	"log"
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

// handlePromoteRelease (chỉ quản lý): CHẠY NGAY việc chuyển các xe 'sắp mở bán' đủ điều kiện
// sang 'đang mở bán'. Dùng cho trường hợp bổ sung xe mới sau mốc tự động (thứ Bảy 21:00).
func (s *Server) handlePromoteRelease(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.PromoteDueInventory(r.Context(), true) // thủ công -> chạy bất kể đang tạm dừng
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "release.promote", "inventory", 0, jsonObj("promoted", fmt.Sprintf("%d", n)))
	writeJSON(w, http.StatusOK, map[string]any{"promoted": n})
}

type pauseReleaseReq struct {
	Paused bool `json:"paused"`
}

// handleSetReleasePause (chỉ quản lý): bật/tắt tạm dừng TỰ ĐỘNG mở bán cho hôm nay.
// Tạm dừng chỉ ảnh hưởng tiến trình tự động (scheduler + lúc tải danh sách); nút "cập nhật mở bán" vẫn chạy được.
func (s *Server) handleSetReleasePause(w http.ResponseWriter, r *http.Request) {
	var req pauseReleaseReq
	if !readJSON(w, r, &req) {
		return
	}
	var err error
	if req.Paused {
		err = s.store.SetReleasePauseToday(r.Context())
	} else {
		err = s.store.ClearReleasePause(r.Context())
	}
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	action := "release.resume"
	if req.Paused {
		action = "release.pause"
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), action, "settings", 0, nil)
	writeJSON(w, http.StatusOK, map[string]any{"paused": req.Paused})
}

// handleReleaseStatus (chỉ quản lý): trạng thái tự động mở bán (đang tạm dừng trong hôm nay?).
func (s *Server) handleReleaseStatus(w http.ResponseWriter, r *http.Request) {
	paused, err := s.store.IsReleasePausedToday(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"paused": paused})
}

// StartReleaseScheduler chạy nền: mỗi thứ Bảy 21:00 (giờ cấu hình) tự chuyển xe 'sắp mở bán'
// đủ điều kiện sang 'đang mở bán'. Chạy 1 lần lúc khởi động để bù các mốc đã lỡ.
func (s *Server) StartReleaseScheduler(ctx context.Context) {
	loc := s.releaseLoc()
	if n, err := s.store.PromoteDueInventory(ctx, false); err != nil {
		log.Printf("release scheduler: lỗi chạy bù lúc khởi động: %v", err)
	} else if n > 0 {
		log.Printf("release scheduler: đã mở bán %d xe (chạy bù lúc khởi động)", n)
	}
	for {
		next := nextSaturday21(time.Now(), loc)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if n, err := s.store.PromoteDueInventory(ctx, false); err != nil {
				log.Printf("release scheduler: lỗi mở bán tự động: %v", err)
			} else {
				log.Printf("release scheduler: đã mở bán %d xe lúc %s", n, time.Now().In(loc).Format(time.RFC3339))
			}
		}
	}
}
