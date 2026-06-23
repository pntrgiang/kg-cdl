.PHONY: help setup demo db-up db-down migrate-up migrate-down seed vehicles-data api frontend-install frontend dev

help:
	@echo "KG Car Dealer — lệnh thường dùng:"
	@echo "  make setup            # db-up + migrate-up + seed (chuẩn bị lần đầu)"
	@echo "  make demo             # tạo dữ liệu demo (cần API đang chạy)"
	@echo "  make db-up            # khởi động Postgres + Adminer (docker compose)"
	@echo "  make db-down          # tắt database"
	@echo "  make migrate-up       # chạy migration (tạo bảng)"
	@echo "  make migrate-down     # rollback migration"
	@echo "  make seed             # tạo dev account + nạp 881 xe GTA5"
	@echo "  make vehicles-data    # tải lại & chuẩn hóa data xe GTA5"
	@echo "  make vehicle-images   # tải ~705 ảnh xe thật + cập nhật image_url"
	@echo "  make batch1           # đặt danh sách xe đang bán = đợt 1"
	@echo "  make api              # chạy backend Go (cổng 8080)"
	@echo "  make frontend-install # cài dependency frontend"
	@echo "  make frontend         # chạy Nuxt dev (cổng 3000)"

setup: db-up
	@sleep 2
	$(MAKE) migrate-up
	$(MAKE) seed

demo:
	bash backend/scripts/demo_seed.sh

vehicles-data:
	curl -sL -o db/seed/gta_vehicles_raw.json https://raw.githubusercontent.com/DurtyFree/gta-v-data-dumps/master/vehicles.json
	node db/seed/transform.mjs

vehicle-images:
	node db/seed/download_images.mjs
	docker exec -i kg_cdl_db psql -U kg -d kg_cdl < db/seed/update_images.sql

batch1:
	docker exec -i kg_cdl_db psql -U kg -d kg_cdl < db/seed/batch1.sql

descriptions:
	node db/seed/descriptions.mjs
	docker exec -i kg_cdl_db psql -U kg -d kg_cdl < db/seed/update_descriptions.sql

stats:
	node db/seed/stats.mjs
	docker exec -i kg_cdl_db psql -U kg -d kg_cdl < db/seed/update_stats.sql

db-up:
	docker compose up -d db adminer

db-down:
	docker compose down

migrate-up:
	cd backend && go run ./cmd/migrate up

migrate-down:
	cd backend && go run ./cmd/migrate down

seed:
	cd backend && go run ./cmd/seed

api:
	cd backend && go run ./cmd/api

frontend-install:
	cd frontend && npm install

frontend:
	cd frontend && npm run dev
