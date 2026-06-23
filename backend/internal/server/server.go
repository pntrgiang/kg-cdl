package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"kg-cdl/backend/internal/auth"
	"kg-cdl/backend/internal/config"
	"kg-cdl/backend/internal/store"
)

// Server gói router + dependency dùng chung.
type Server struct {
	cfg   *config.Config
	pool  *pgxpool.Pool
	store *store.Store
	authm *auth.Manager
	mux   *chi.Mux
}

func New(cfg *config.Config, pool *pgxpool.Pool) *Server {
	s := &Server{
		cfg:   cfg,
		pool:  pool,
		store: store.New(pool),
		authm: auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL),
		mux:   chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	r := s.mux
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.CORSAllowOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", s.handleHealth)

	r.Route("/api", func(r chi.Router) {
		// ── public / auth ────────────────────────────────
		// Giới hạn tần suất theo IP cho các endpoint nhạy cảm (đăng nhập/đăng ký/tra cứu)
		// để chống brute-force mật khẩu, spam đăng ký và dò số căn cước.
		// Chỉ áp dụng cho nhóm này vì chúng luôn gọi từ trình duyệt (IP thật của khách);
		// các endpoint GET công khai được Nuxt gọi khi SSR (chung IP máy chủ) nên không giới hạn ở app,
		// flood tổng thể nên chặn ở tầng Caddy.
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(20, time.Minute))
			r.Post("/auth/login", s.handleStaffLogin)
			r.Post("/auth/customer/register", s.handleCustomerRegister)
			r.Post("/auth/customer/login", s.handleCustomerLogin)
			r.Get("/auth/customer/lookup", s.handleCustomerLookup)
		})
		r.With(httprate.LimitByIP(60, time.Minute)).Post("/auth/refresh", s.handleRefresh)
		r.Post("/auth/logout", s.handleLogout)

		// ── public catalog/events (khách xem) ────────────
		r.Get("/vehicles", s.handleListVehicles)
		r.Get("/vehicles/{id}", s.handleGetVehicle)
		r.Get("/vehicles/{id}/similar", s.handleSimilarVehicles)
		r.Get("/vehicles/{id}/discounts", s.handleVehicleDiscounts)
		r.Get("/events", s.handleListEvents)
		r.Get("/events/{id}", s.handleGetEvent)
			r.Get("/release-info", s.handleReleaseInfo)
			r.Get("/uploads/{name}", s.handleServeUpload)
			r.Get("/banners/active", s.handleListBannersPublic)

		// ── cần đăng nhập (user hoặc customer) ───────────
		r.Group(func(r chi.Router) {
			r.Use(s.authm.Middleware)
			r.Use(s.requireExistingSubject)
			r.Get("/me", s.handleMe)
			r.Post("/auth/change-password", s.handleChangePassword)

			// customer-only
			r.With(auth.RequireCustomer).Get("/me/prizes", s.handleMyPrizes)
			r.With(auth.RequireCustomer).Post("/events/{id}/register", s.handleRegisterEvent)
			r.With(auth.RequireCustomer).Get("/events/{id}/registration", s.handleMyRegistration)
			r.With(auth.RequireCustomer).Post("/events/{id}/spin", s.handleSpin)
				r.With(auth.RequireCustomer).Post("/bookings", s.handleCreateBooking)
				r.With(auth.RequireCustomer).Get("/me/bookings", s.handleMyBookings)

			// staff area (user)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireUser)

				// ── nhân viên xem được (chỉ đọc) ─────────────
				r.Get("/catalog", s.handleListCatalog)
				r.Get("/inventory", s.handleListInventory)
				r.Get("/sales-weeks", s.handleListSalesWeeks)
				r.Get("/customers", s.handleListCustomers)
				r.Get("/customers/{id}/prizes", s.handleCustomerPrizes)
				r.Get("/customers/{id}/sales", s.handleCustomerSales)
				r.Get("/vouchers", s.handleListVouchers)
				r.Get("/sales", s.handleListSales)
				r.Get("/reports/revenue", s.handleRevenueReport)
				r.Get("/settings/rank-limits", s.handleGetRankLimits)
				r.Get("/logs", s.handleListLogs)
				r.Get("/logs/actions", s.handleLogActions)

				// ── nhân viên thao tác: bán xe + tạo khách (cho phiên bán) ─
				r.Post("/customers", s.handleCreateCustomer)
				r.Post("/sales", s.handleCreateSale)

					// ── đặt lịch: nhân viên + quản lý đều xem & xử lý (nhận/từ chối) ─
					r.Get("/bookings", s.handleListBookings)
					r.Patch("/bookings/{id}", s.handleHandleBooking)

				// ── chỉ quản lý mới được chỉnh sửa kho/danh mục/khuyến mãi/hoàn trả ─
				r.Group(func(r chi.Router) {
					r.Use(auth.RequireRole(auth.RoleManager, auth.RoleDev))
					r.Post("/catalog", s.handleCreateCatalog)
					r.Patch("/catalog/{id}", s.handleUpdateCatalog)
					r.Post("/inventory", s.handleCreateInventory)
					r.Post("/sales-weeks", s.handleCreateSalesWeek)
					r.Post("/inventory/{id}/discount", s.handleSetDiscount)
					r.Patch("/inventory/{id}/status", s.handleUpdateInventoryStatus)
						r.Patch("/inventory/{id}/booking", s.handleSetBookingOpen)
						r.Put("/release-override", s.handleSetReleaseOverride)
						r.Put("/release-modal", s.handleSetReleaseModal)
						r.Post("/release-popup-upload", s.handlePopupUpload)
						r.Get("/banners", s.handleListBanners)
						r.Post("/banners", s.handleCreateBanner)
						r.Patch("/banners/{id}", s.handleToggleBanner)
					r.Post("/sales/{id}/refund", s.handleRefundSale)
					r.Put("/customers/{id}", s.handleUpdateCustomer)
					r.Post("/events", s.handleCreateEvent)
					r.Post("/vouchers", s.handleCreateVoucher)
					r.Post("/vouchers/{id}/cancel", s.handleCancelVoucher)
					r.Post("/events/draw", s.handleCreateDrawEvent)
					r.Get("/events/{id}/entrants", s.handleEventEntrants)
						r.Post("/events/{id}/draw", s.handleDrawRun)
					r.Post("/events/{id}/redraw", s.handleDrawRedraw)
					r.Post("/events/{id}/confirm", s.handleDrawConfirm)
				})

				r.With(auth.RequireRole(auth.RoleDev)).Put("/admin/rank-limits", s.handleSetRankLimits)

				// dev-only
				r.With(auth.RequireRole(auth.RoleDev)).Delete("/customers/{id}", s.handleDeleteCustomer)
					r.With(auth.RequireRole(auth.RoleDev)).Post("/customers/{id}/reset-password", s.handleResetCustomerPassword)
					r.With(auth.RequireRole(auth.RoleDev)).Delete("/banners/{id}", s.handleDeleteBanner)
				r.With(auth.RequireRole(auth.RoleDev)).Get("/admin/users", s.handleListUsers)
				r.With(auth.RequireRole(auth.RoleDev)).Post("/admin/users", s.handleCreateUser)
				r.With(auth.RequireRole(auth.RoleDev)).Put("/admin/users/{id}/role", s.handleUpdateUserRole)
				r.With(auth.RequireRole(auth.RoleDev)).Delete("/admin/users/{id}", s.handleDeleteUser)
			})
		})
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status, dbStatus := "ok", "ok"
	if err := s.pool.Ping(r.Context()); err != nil {
		status, dbStatus = "degraded", "down"
	}
	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"status": status, "db": dbStatus, "env": s.cfg.AppEnv})
}
