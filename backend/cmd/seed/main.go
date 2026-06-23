// Command seed: tạo tài khoản dev đầu tiên + nạp danh mục xe GTA5.
// Idempotent: chạy lại không tạo trùng (ON CONFLICT DO NOTHING theo model_code).
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path"

	"kg-cdl/backend/internal/auth"
	"kg-cdl/backend/internal/config"
	"kg-cdl/backend/internal/db"
	"kg-cdl/backend/internal/store"
)

type seedVehicle struct {
	ModelCode    string `json:"model_code"`
	Name         string `json:"name"`
	Brand        string `json:"brand"`
	Class        string `json:"class"`
	ClassID      string `json:"class_id"`
	DefaultPrice int64  `json:"default_price"`
	Description  string `json:"description"`
	Seats        *int   `json:"seats"`
	RateSpeed    int    `json:"rate_speed"`
	RateAccel    int    `json:"rate_accel"`
	RateBraking  int    `json:"rate_braking"`
	RateTraction int    `json:"rate_traction"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.Timezone)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	seedDevUser(ctx, st)
	seedVehicles(ctx, st)
	log.Println("seed: done")
}

func seedDevUser(ctx context.Context, st *store.Store) {
	n, err := st.CountUsers(ctx)
	if err != nil {
		log.Fatalf("count users: %v", err)
	}
	if n > 0 {
		log.Printf("seed: đã có %d nhân viên, bỏ qua tạo dev", n)
		return
	}
	username := envOr("SEED_DEV_USER", "admin")
	password := envOr("SEED_DEV_PASS", "admin123")
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	u, err := st.CreateUser(ctx, username, hash, "Dev Admin", auth.RoleDev, nil)
	if err != nil {
		log.Fatalf("create dev: %v", err)
	}
	log.Printf("seed: tạo tài khoản dev '%s' (mật khẩu mặc định '%s') id=%d", u.Username, password, u.ID)
}

func seedVehicles(ctx context.Context, st *store.Store) {
	exec := st.Pool()
	vehPath := envOr("SEED_VEHICLES_PATH", "../db/seed/gta_vehicles.json")
	raw, err := os.ReadFile(vehPath)
	if err != nil {
		log.Printf("seed: bỏ qua nạp xe (không đọc được %s: %v)", vehPath, err)
		return
	}
	var list []seedVehicle
	if err := json.Unmarshal(raw, &list); err != nil {
		log.Fatalf("parse vehicles: %v", err)
	}
	// thư mục ảnh thật đã tải (nếu có file <code>.png thì dùng ảnh thật).
	imgDir := envOr("SEED_IMAGE_DIR", "../frontend/public/vehicles/img")
	inserted := 0
	for _, v := range list {
		image := "/vehicles/class/" + v.ClassID + ".svg"
		if st, err := os.Stat(path.Join(imgDir, v.ModelCode+".webp")); err == nil && st.Size() > 0 {
			image = "/vehicles/img/" + v.ModelCode + ".webp"
		}
		ct, err := exec.Exec(ctx, `
			INSERT INTO vehicle_catalog (model_code, name, brand, class, image_url, description,
			  seats, rate_speed, rate_accel, rate_braking, rate_traction, is_mod)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,false)
			ON CONFLICT (model_code) DO NOTHING`,
			v.ModelCode, v.Name, v.Brand, v.Class, image, v.Description,
			v.Seats, v.RateSpeed, v.RateAccel, v.RateBraking, v.RateTraction)
		if err != nil {
			log.Fatalf("insert vehicle %s: %v", v.ModelCode, err)
		}
		inserted += int(ct.RowsAffected())
	}
	log.Printf("seed: nạp %d xe mới (tổng %d trong file)", inserted, len(list))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
