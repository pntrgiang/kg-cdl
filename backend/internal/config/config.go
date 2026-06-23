package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config chứa toàn bộ cấu hình runtime của backend.
type Config struct {
	AppEnv          string
	HTTPPort        string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CORSAllowOrigin string
	Timezone        string // IANA tz nghiệp vụ, vd Asia/Ho_Chi_Minh (độc lập với vị trí máy chủ)
	UploadDir       string // thư mục lưu ảnh tải lên (popup, banner) — bền vững, ngoài build frontend
}

// Load đọc cấu hình từ biến môi trường (và file .env nếu có).
func Load() (*Config, error) {
	// .env ở thư mục gốc dự án; thử từng vị trí, bỏ qua nếu không tồn tại
	// (production dùng env thật). Load riêng từng file vì godotenv.Load dừng
	// ở file đầu tiên không tồn tại.
	for _, p := range []string{".env", "../.env", "../../.env"} {
		_ = godotenv.Load(p)
	}

	cfg := &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		CORSAllowOrigin: getEnv("CORS_ALLOW_ORIGIN", "http://localhost:3000"),
		Timezone:        getEnv("APP_TIMEZONE", "Asia/Ho_Chi_Minh"),
		UploadDir:       getEnv("UPLOAD_DIR", "./uploads"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	var err error
	if cfg.AccessTokenTTL, err = parseDuration("ACCESS_TOKEN_TTL", "30m"); err != nil {
		return nil, err
	}
	if cfg.RefreshTokenTTL, err = parseDuration("REFRESH_TOKEN_TTL", "2160h"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	v := getEnv(key, fallback)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}
