-- Banner trang chủ (giao diện khách): nhiều banner, chọn banner nào hiển thị slide.
CREATE TABLE banners (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    image      text   NOT NULL,                       -- tên file trong UploadDir
    is_active  boolean NOT NULL DEFAULT true,          -- có dùng để slide không
    sort       integer NOT NULL DEFAULT 0,             -- thứ tự slide
    created_by bigint REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_banners_active ON banners(is_active, sort, id);
