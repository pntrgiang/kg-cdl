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
	if req.Username == "" || strings.TrimSpace(req.DisplayName) == "" {
		writeErr(w, http.StatusBadRequest, "cần username và tên hiển thị")
		return
	}
	if !validRole(req.Role) {
		writeErr(w, http.StatusBadRequest, "role không hợp lệ")
		return
	}
	nidStr := strings.TrimSpace(req.NationalID)
	var nid *string
	if nidStr != "" {
		nid = &nidStr
	}

	// Nguồn mật khẩu cho tài khoản nhân viên:
	//  - Nhập mật khẩu mới (>=6) -> dùng mật khẩu đó.
	//  - Để TRỐNG + có số căn cước trùng KHÁCH ĐÃ CÓ TÀI KHOẢN -> DÙNG LẠI mật khẩu của khách,
	//    để khách thăng cấp đăng nhập bằng đúng tài khoản/mật khẩu đã đăng ký.
	var hash string
	var promotedCustomerID int64 // != 0 nếu đây là thăng cấp từ khách đã có tài khoản
	if req.Password == "" && nidStr != "" {
		if cid, ch, err := s.store.GetCustomerAuthByNationalID(r.Context(), nidStr); err == nil && ch != "" {
			hash = ch
			promotedCustomerID = cid
		}
	}
	if hash == "" {
		if len(req.Password) < 6 {
			writeErr(w, http.StatusBadRequest, "cần mật khẩu (>=6); hoặc chọn khách đã có tài khoản và để trống mật khẩu để dùng lại mật khẩu của khách")
			return
		}
		h, err := auth.HashPassword(req.Password)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		hash = h
	}

	u, err := s.store.CreateUser(r.Context(), req.Username, hash, req.DisplayName, req.Role, nid)
	if err != nil {
		writeErr(w, http.StatusConflict, "username hoặc số căn cước đã tồn tại")
		return
	}
	// thăng cấp: vô hiệu phiên khách hiện tại -> họ bị đăng xuất ngay, đăng nhập lại sẽ vào với tư cách nhân viên;
	// đồng thời xếp lại hạng để loại tài khoản khách này khỏi bảng xếp hạng thành viên (nhân viên không được xếp hạng).
	if promotedCustomerID != 0 {
		_ = s.store.InvalidateSessions(r.Context(), auth.SubjectCustomer, promotedCustomerID)
		limits, _ := s.store.GetRankLimits(r.Context())
		_ = s.store.RecomputeRanks(r.Context(), limits.SVIP, limits.VIP)
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
	// đổi quyền -> vô hiệu phiên cũ (token mang role cũ) để buộc đăng nhập lại
	_ = s.store.InvalidateSessions(r.Context(), auth.SubjectUser, id)
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
	// xoá/ngưng nhân viên -> vô hiệu mọi phiên hiện tại của họ (đăng xuất ngay)
	_ = s.store.InvalidateSessions(r.Context(), auth.SubjectUser, id)
	// loại bỏ nhân viên -> trở lại làm khách hàng: tài khoản khách (cùng CCCD) BẮT ĐẦU LẠI TỪ PHỔ THÔNG
	// (chi tiêu = 0, xem như chưa từng mua xe), rồi xếp lại hạng toàn bộ.
	if u.NationalID != nil && strings.TrimSpace(*u.NationalID) != "" {
		_ = s.store.ResetCustomerSpendingByNationalID(r.Context(), strings.TrimSpace(*u.NationalID))
		limits, _ := s.store.GetRankLimits(r.Context())
		_ = s.store.RecomputeRanks(r.Context(), limits.SVIP, limits.VIP)
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
