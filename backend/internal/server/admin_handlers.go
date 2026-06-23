package server

import (
	"net/http"
	"strings"

	"kg-cdl/backend/internal/auth"
	"kg-cdl/backend/internal/store"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if users == nil {
		users = []store.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

type createUserReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	NationalID  string `json:"national_id"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if !readJSON(w, r, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 || strings.TrimSpace(req.DisplayName) == "" {
		writeErr(w, http.StatusBadRequest, "cần username, mật khẩu (>=6) và tên hiển thị")
		return
	}
	if !validRole(req.Role) {
		writeErr(w, http.StatusBadRequest, "role không hợp lệ")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	var nid *string
	if v := strings.TrimSpace(req.NationalID); v != "" {
		nid = &v
	}
	u, err := s.store.CreateUser(r.Context(), req.Username, hash, req.DisplayName, req.Role, nid)
	if err != nil {
		writeErr(w, http.StatusConflict, "username hoặc số căn cước đã tồn tại")
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "user.create", "user", u.ID, jsonObj2("username", u.Username, "role", u.Role))
	writeJSON(w, http.StatusCreated, u)
}

type updateRoleReq struct {
	Role string `json:"role"`
}

func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	var req updateRoleReq
	if !readJSON(w, r, &req) {
		return
	}
	if !validRole(req.Role) {
		writeErr(w, http.StatusBadRequest, "role không hợp lệ")
		return
	}
	actor, _ := identity(r)
	if actor.SubjectID == id {
		writeErr(w, http.StatusBadRequest, "không thể tự đổi role của chính mình")
		return
	}
	u, err := s.store.UpdateUserRole(r.Context(), id, req.Role)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "user.role", "user", u.ID, jsonObj2("username", u.Username, "role", u.Role))
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	actor, _ := identity(r)
	if actor.SubjectID == id {
		writeErr(w, http.StatusBadRequest, "không thể tự xoá chính mình")
		return
	}
	u, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	hard, err := s.store.DeleteUser(r.Context(), id)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	action := "user.delete"
	if !hard {
		action = "user.deactivate"
	}
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), action, "user", id, jsonObj("username", u.Username))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "hard": hard})
}

// handleGetRankLimits trả giới hạn số lượng svip/vip hiện tại (nhân viên xem được).
func (s *Server) handleGetRankLimits(w http.ResponseWriter, r *http.Request) {
	limits, err := s.store.GetRankLimits(r.Context())
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, limits)
}

type rankLimitsReq struct {
	SVIP int `json:"svip"`
	VIP  int `json:"vip"`
}

// handleSetRankLimits (chỉ dev): đặt giới hạn svip/vip + xếp lại rank toàn bộ khách.
func (s *Server) handleSetRankLimits(w http.ResponseWriter, r *http.Request) {
	var req rankLimitsReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.SVIP < 0 || req.VIP < 0 {
		writeErr(w, http.StatusBadRequest, "giới hạn phải ≥ 0")
		return
	}
	if err := s.store.SetRankLimits(r.Context(), store.RankLimits{SVIP: req.SVIP, VIP: req.VIP}); err != nil {
		handleStoreErr(w, err)
		return
	}
	if err := s.store.RecomputeRanks(r.Context(), req.SVIP, req.VIP); err != nil {
		handleStoreErr(w, err)
		return
	}
	actor, _ := identity(r)
	_ = s.store.InsertLog(r.Context(), actor.SubjectID, s.actorName(r.Context(), actor), "settings.rank_limits", "settings", 0, jsonObj2("svip", req.SVIP, "vip", req.VIP))
	writeJSON(w, http.StatusOK, store.RankLimits{SVIP: req.SVIP, VIP: req.VIP})
}

func validRole(role string) bool {
	switch role {
	case auth.RoleDev, auth.RoleManager, auth.RoleStaff:
		return true
	}
	return false
}
