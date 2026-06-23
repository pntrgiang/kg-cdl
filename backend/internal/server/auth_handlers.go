package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"kg-cdl/backend/internal/auth"
	"kg-cdl/backend/internal/store"
)

type tokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// issueTokens tạo access + refresh và lưu refresh hash.
func (s *Server) issueTokens(ctx context.Context, r *http.Request, id auth.Identity) (tokenPair, error) {
	access, exp, err := s.authm.IssueAccessToken(id)
	if err != nil {
		return tokenPair{}, err
	}
	raw, hash, err := auth.NewRefreshToken()
	if err != nil {
		return tokenPair{}, err
	}
	expiresAt := time.Now().UTC().Add(s.authm.RefreshTTL())
	ip := r.RemoteAddr
	if host, _, splitErr := net.SplitHostPort(ip); splitErr == nil {
		ip = host
	}
	if err := s.store.SaveRefreshToken(ctx, id.SubjectType, id.SubjectID, hash, expiresAt, r.UserAgent(), ip); err != nil {
		return tokenPair{}, err
	}
	return tokenPair{AccessToken: access, RefreshToken: raw, ExpiresAt: exp}, nil
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleStaffLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !readJSON(w, r, &req) {
		return
	}
	u, hash, err := s.store.GetUserByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil || !u.IsActive || !auth.CheckPassword(hash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "sai tài khoản hoặc mật khẩu")
		return
	}
	id := auth.Identity{SubjectType: auth.SubjectUser, SubjectID: u.ID, Role: u.Role}
	tp, err := s.issueTokens(r.Context(), r, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tp, "user": u})
}

func (s *Server) handleCustomerLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !readJSON(w, r, &req) {
		return
	}
	c, hash, err := s.store.GetCustomerByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil || hash == "" || !auth.CheckPassword(hash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "sai tài khoản hoặc mật khẩu")
		return
	}
	id := auth.Identity{SubjectType: auth.SubjectCustomer, SubjectID: c.ID}
	tp, err := s.issueTokens(r.Context(), r, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tp, "customer": c})
}

// handleCustomerLookup tra cứu thông tin theo số căn cước (cho màn đăng ký tự điền).
// Chỉ trả tên/SĐT khi bản ghi CHƯA có tài khoản (do nhân viên tạo trước).
func (s *Server) handleCustomerLookup(w http.ResponseWriter, r *http.Request) {
	nid := normalizeNationalID(r.URL.Query().Get("national_id"))
	if nid == "" {
		writeErr(w, http.StatusBadRequest, "cần số căn cước")
		return
	}
	c, err := s.store.GetCustomerByNationalID(r.Context(), nid)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"found": false})
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if c.ClaimedAt != nil {
		// đã có tài khoản -> không tiết lộ thông tin, hướng dẫn đăng nhập
		writeJSON(w, http.StatusOK, map[string]any{"found": true, "claimed": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"found": true, "claimed": false,
		"full_name": c.FullName, "phone": c.Phone,
	})
}

type customerRegisterReq struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	NationalID string `json:"national_id"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
}

// handleCustomerRegister: nếu national_id đã được nhân viên tạo trước -> claim bản ghi đó
// (giữ nguyên thông tin & lịch sử mua). Nếu chưa có -> tạo mới, các trường để trống nếu khách bỏ trống.
func (s *Server) handleCustomerRegister(w http.ResponseWriter, r *http.Request) {
	var req customerRegisterReq
	if !readJSON(w, r, &req) {
		return
	}
	req.NationalID = normalizeNationalID(req.NationalID)
	if len(req.Password) < 6 || req.NationalID == "" {
		writeErr(w, http.StatusBadRequest, "cần mật khẩu (>=6 ký tự) và số căn cước")
		return
	}
	if !validNationalID(req.NationalID) {
		writeErr(w, http.StatusBadRequest, "số căn cước phải có dạng LUX + 5 chữ số (vd LUX12345)")
		return
	}
	// Tài khoản đăng nhập CHÍNH LÀ số căn cước (không cho client tự đặt).
	req.Username = req.NationalID
	passHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	var c store.Customer
	existing, err := s.store.GetCustomerByNationalID(r.Context(), req.NationalID)
	switch {
	case err == nil:
		if existing.ClaimedAt != nil {
			writeErr(w, http.StatusConflict, "số căn cước này đã có tài khoản")
			return
		}
		c, err = s.store.ClaimExisting(r.Context(), existing.ID, req.Username, passHash)
	case errors.Is(err, store.ErrNotFound):
		c, err = s.store.RegisterNewCustomer(r.Context(), req.Username, passHash, req.NationalID, req.FullName, req.Phone)
	}
	if err != nil {
		// trùng username sẽ rơi vào đây (unique violation)
		writeErr(w, http.StatusConflict, "không tạo được tài khoản (username có thể đã tồn tại)")
		return
	}

	id := auth.Identity{SubjectType: auth.SubjectCustomer, SubjectID: c.ID}
	tp, err := s.issueTokens(r.Context(), r, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": tp, "customer": c})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !readJSON(w, r, &req) {
		return
	}
	hash := auth.HashToken(strings.TrimSpace(req.RefreshToken))
	sub, err := s.store.LookupRefreshToken(r.Context(), hash)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "refresh token không hợp lệ")
		return
	}
	id := auth.Identity{SubjectType: sub.SubjectType, SubjectID: sub.SubjectID}
	if sub.SubjectType == auth.SubjectUser {
		u, err := s.store.GetUserByID(r.Context(), sub.SubjectID)
		if err != nil || !u.IsActive {
			writeErr(w, http.StatusUnauthorized, "tài khoản không khả dụng")
			return
		}
		id.Role = u.Role
	} else {
		if _, err := s.store.GetCustomerByID(r.Context(), sub.SubjectID); err != nil {
			writeErr(w, http.StatusUnauthorized, "tài khoản không khả dụng")
			return
		}
	}
	// rotate: thu hồi refresh cũ, cấp cặp mới.
	_ = s.store.RevokeRefreshToken(r.Context(), hash)
	tp, err := s.issueTokens(r.Context(), r, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tp})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.RefreshToken != "" {
		_ = s.store.RevokeRefreshToken(r.Context(), auth.HashToken(strings.TrimSpace(req.RefreshToken)))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// handleChangePassword cho chủ thể đang đăng nhập (nhân viên hoặc khách) tự đổi mật khẩu.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordReq
	if !readJSON(w, r, &req) {
		return
	}
	if len(req.NewPassword) < 6 {
		writeErr(w, http.StatusBadRequest, "mật khẩu mới cần tối thiểu 6 ký tự")
		return
	}
	id, _ := identity(r)

	var oldHash string
	var err error
	if id.SubjectType == auth.SubjectUser {
		oldHash, err = s.store.GetUserPasswordHash(r.Context(), id.SubjectID)
	} else {
		oldHash, err = s.store.GetCustomerPasswordHash(r.Context(), id.SubjectID)
	}
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	if oldHash == "" || !auth.CheckPassword(oldHash, req.OldPassword) {
		writeErr(w, http.StatusBadRequest, "mật khẩu hiện tại không đúng")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if id.SubjectType == auth.SubjectUser {
		err = s.store.UpdateUserPassword(r.Context(), id.SubjectID, newHash)
		_ = s.store.InsertLog(r.Context(), id.SubjectID, s.actorName(r.Context(), id), "user.password", "user", id.SubjectID, nil)
	} else {
		err = s.store.UpdateCustomerPassword(r.Context(), id.SubjectID, newHash)
	}
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMe trả về thông tin chủ thể hiện tại.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	id, _ := identity(r)
	if id.SubjectType == auth.SubjectUser {
		u, err := s.store.GetUserByID(r.Context(), id.SubjectID)
		if err != nil {
			handleStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"type": "user", "user": u})
		return
	}
	c, err := s.store.GetCustomerByID(r.Context(), id.SubjectID)
	if err != nil {
		handleStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": "customer", "customer": c})
}
