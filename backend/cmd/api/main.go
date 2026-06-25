package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kg-cdl/backend/internal/config"
	"kg-cdl/backend/internal/db"
	"kg-cdl/backend/internal/server"
)

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

	app := server.New(cfg, pool)
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Tự động mở bán xe 'sắp mở bán' đủ điều kiện vào 21:00 thứ Bảy hằng tuần.
	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()
	go app.StartReleaseScheduler(schedCtx)

	go func() {
		log.Printf("API listening on %s (env=%s)", srv.Addr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
