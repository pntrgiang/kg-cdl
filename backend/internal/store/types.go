package store

import (
	"encoding/json"
	"time"
)

type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	NationalID  *string   `json:"national_id"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type Customer struct {
	ID            int64      `json:"id"`
	Username      *string    `json:"username"`
	FullName      string     `json:"full_name"`
	Phone         string     `json:"phone"`
	NationalID    string     `json:"national_id"`
	Gender        *string    `json:"gender"`     // 'male' | 'female' | 'other' | null
	BirthDate     *string    `json:"birth_date"` // 'YYYY-MM-DD' | null
	Rank          string     `json:"rank"`
	TotalSpent    float64    `json:"total_spent"`
	LastPurchase  *time.Time `json:"last_purchase_at"`
	ClaimedAt     *time.Time `json:"claimed_at"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
}

type CatalogVehicle struct {
	ID          int64     `json:"id"`
	ModelCode   *string   `json:"model_code"`
	Name        string    `json:"name"`
	Brand       string    `json:"brand"`
	Class       string    `json:"class"`
	ImageURL    string    `json:"image_url"`
	Description string    `json:"description"`
	Model3D     string    `json:"model_3d"`
	Seats        *int     `json:"seats"`
	TrunkKg      int      `json:"trunk_kg"`
	RateSpeed    int      `json:"rate_speed"`
	RateAccel    int      `json:"rate_accel"`
	RateBraking  int      `json:"rate_braking"`
	RateTraction int      `json:"rate_traction"`
	IsMod       bool      `json:"is_mod"`
	CreatedAt   time.Time `json:"created_at"`
}

// InventoryItem là một dòng kho kèm thông tin catalog + khuyến mãi đã tính.
type InventoryItem struct {
	ID              int64     `json:"id"`
	CatalogID       int64     `json:"catalog_id"`
	Name            string    `json:"name"`
	Brand           string    `json:"brand"`
	Class           string    `json:"class"`
	ImageURL        string    `json:"image_url"`
	Description     string    `json:"description"`
	Model3D         string    `json:"model_3d"`
	Seats           *int      `json:"seats"`
	TrunkKg         int       `json:"trunk_kg"`
	RateSpeed       int       `json:"rate_speed"`
	RateAccel       int       `json:"rate_accel"`
	RateBraking     int       `json:"rate_braking"`
	RateTraction    int       `json:"rate_traction"`
	BasePrice       float64   `json:"base_price"`
	Quantity        int       `json:"quantity"`        // tồn hiện tại
	TotalImported   int       `json:"total_imported"`  // tổng số xe đã nhập (tồn + đã bán chưa hoàn)
	Status          string    `json:"status"`
	OnSaleAt        *time.Time `json:"on_sale_at"`
	Note            string    `json:"note"`
	DiscountPercent float64   `json:"discount_percent"` // 0 nếu không có
	FinalPrice      float64   `json:"final_price"`
	BookingOpen     bool      `json:"booking_open"` // có nhận đặt lịch xem/mua không
	CreatedAt       time.Time `json:"created_at"`
}

// Booking: lịch đặt xem/mua xe của khách.
type Booking struct {
	ID          int64     `json:"id"`
	InventoryID int64     `json:"inventory_id"`
	CustomerID  int64     `json:"customer_id"`
	VehicleName string    `json:"vehicle_name"`
	VisitDate   string    `json:"visit_date"` // YYYY-MM-DD
	Note        string    `json:"note"`
	Status      string    `json:"status"` // pending | accepted | rejected
	HandledAt   *time.Time `json:"handled_at"`
	CreatedAt   time.Time `json:"created_at"`
	// chỉ dùng cho danh sách nhân viên/quản lý
	CustomerName       string `json:"customer_name,omitempty"`
	CustomerNationalID string `json:"customer_national_id,omitempty"`
	CustomerPhone      string `json:"customer_phone,omitempty"`
	HandledByName      string `json:"handled_by_name,omitempty"`
}

type Sale struct {
	ID              int64     `json:"id"`
	InventoryID     int64     `json:"inventory_id"`
	CatalogID       int64     `json:"catalog_id"`
	CustomerID      int64     `json:"customer_id"`
	CustomerName    string    `json:"customer_name"`
	SoldBy          int64     `json:"sold_by"`
	SoldByName      string    `json:"sold_by_name"`
	OriginalPrice   float64   `json:"original_price"`
	DiscountPercent float64   `json:"discount_percent"`
	FinalPrice      float64   `json:"final_price"`
	VehicleName     string    `json:"vehicle_name"`
	VoucherDiscount float64   `json:"voucher_discount"`
	Refunded        bool      `json:"refunded"`
	RefundReason    string    `json:"refund_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Event struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	StartsAt    *time.Time  `json:"starts_at"`
	EndsAt      *time.Time  `json:"ends_at"`
	IsActive    bool        `json:"is_active"`
	CreatedBy   int64       `json:"created_by"`
	CreatedAt   time.Time   `json:"created_at"`
	Prizes      []EventPrize `json:"prizes,omitempty"`

	// Sự kiện quay số trúng thưởng (draw_status != null)
	RegisterDeadline *time.Time `json:"register_deadline"`
	PrizeType        *string    `json:"prize_type"`  // 'voucher' | 'vehicle'
	VoucherID        *int64     `json:"voucher_id"`
	PrizeVehicleID   *int64     `json:"prize_vehicle_catalog_id"`
	WinnersCount     *int       `json:"winners_count"`
	DrawStatus       *string    `json:"draw_status"` // 'open' | 'drawn' | 'published'
	CancelledAt      *time.Time `json:"cancelled_at"`
	CancelReason     string     `json:"cancel_reason"`
	PrizeName        string     `json:"prize_name"`
	EligibleCount    int        `json:"eligible_count"`
	Winners          []EventWinner `json:"winners,omitempty"`
}

type Voucher struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	DiscountPercent float64    `json:"discount_percent"`
	MaxAmount       float64    `json:"max_amount"` // 0 = tối đa = toàn bộ giá trị xe
	Quantity        int        `json:"quantity"`   // tổng số lượt được dùng
	UsedCount       int        `json:"used_count"` // số lượt đã dùng
	Remaining       int        `json:"remaining"`  // còn lại = quantity - used_count
	ExpiresAt       *time.Time `json:"expires_at"`
	AppliesToAll    bool       `json:"applies_to_all"`
	MinRank         string     `json:"min_rank"`             // 'regular' | 'vip' | 'svip'
	Vehicles        []VoucherVehicle `json:"vehicles,omitempty"` // khi applies_to_all=false
	IsActive        bool       `json:"is_active"`
	CancelledAt     *time.Time `json:"cancelled_at"`
	CancelReason    string     `json:"cancel_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VoucherVehicle: xe cụ thể mà voucher áp dụng.
type VoucherVehicle struct {
	CatalogID int64  `json:"catalog_id"`
	Name      string `json:"name"`
}

type EventWinner struct {
	ID           int64      `json:"id"`
	CustomerID   int64      `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	Status       string     `json:"status"`
	FulfilledAt  *time.Time `json:"fulfilled_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CustomerVoucher: voucher khả dụng của khách (dùng khi mua xe).
type CustomerVoucher struct {
	ID              int64   `json:"id"`
	VoucherID       int64   `json:"voucher_id"`
	Name            string  `json:"name"`
	DiscountPercent float64 `json:"discount_percent"`
	MaxAmount       float64 `json:"max_amount"`
}

// VehiclePrize: xe tặng trúng thưởng chưa giao của khách.
type VehiclePrize struct {
	WinID       int64  `json:"win_id"`
	EventID     int64  `json:"event_id"`
	EventTitle  string `json:"event_title"`
	VehicleName string `json:"vehicle_name"`
}

// CustomerVoucherFull: voucher của khách kèm trạng thái dùng + nhân viên đã áp dụng (cho trang tài khoản).
type CustomerVoucherFull struct {
	ID              int64            `json:"id"`
	VoucherID       int64            `json:"voucher_id"`
	Name            string           `json:"name"`
	DiscountPercent float64          `json:"discount_percent"`
	MaxAmount       float64          `json:"max_amount"`
	Status          string           `json:"status"` // 'available' | 'used' | 'cancelled'
	UsedAt          *time.Time       `json:"used_at"`
	ExpiresAt       *time.Time       `json:"expires_at"`
	AppliesToAll    bool             `json:"applies_to_all"`
	MinRank         string           `json:"min_rank"`
	Vehicles        []VoucherVehicle `json:"vehicles,omitempty"`
	SellerName      *string          `json:"seller_name"`
	CancelReason    string           `json:"cancel_reason"` // khi status='cancelled'
}

// VehiclePrizeFull: xe trúng thưởng kèm trạng thái giao + nhân viên đã giao.
type VehiclePrizeFull struct {
	WinID       int64      `json:"win_id"`
	EventID     int64      `json:"event_id"`
	EventTitle  string     `json:"event_title"`
	VehicleName string     `json:"vehicle_name"`
	Fulfilled   bool       `json:"fulfilled"`
	FulfilledAt *time.Time `json:"fulfilled_at"`
	SellerName  *string    `json:"seller_name"`
}

type EventPrize struct {
	ID       int64   `json:"id"`
	EventID  int64   `json:"event_id"`
	Name     string  `json:"name"`
	ImageURL string  `json:"image_url"`
	Weight   int     `json:"weight"`
	Stock    *int    `json:"stock"`
	IsActive bool    `json:"is_active"`
}

type Spin struct {
	ID         int64     `json:"id"`
	EventID    int64     `json:"event_id"`
	CustomerID int64     `json:"customer_id"`
	PrizeID    *int64    `json:"prize_id"`
	PrizeName  string    `json:"prize_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type Discount struct {
	ID        int64      `json:"id"`
	Percent   float64    `json:"percent"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

type ActivityLog struct {
	ID         int64     `json:"id"`
	ActorID    *int64    `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	Action     string    `json:"action"`
	TargetType *string         `json:"target_type"`
	TargetID   *int64          `json:"target_id"`
	Detail     json.RawMessage `json:"detail"`
	CreatedAt  time.Time       `json:"created_at"`
}
