package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

const maxUploadBytes = 10 << 20 // 10MB

// saveUpload đọc file từ multipart form (field "file"), kiểm tra là ảnh, lưu vào UploadDir với tên destName.
// Ghi đè nếu đã tồn tại. Trả về lỗi nếu không hợp lệ.
func (s *Server) saveUpload(r *http.Request, field, destName string) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxUploadBytes+1024)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return fmt.Errorf("file quá lớn hoặc không hợp lệ")
	}
	file, hdr, err := r.FormFile(field)
	if err != nil {
		return fmt.Errorf("thiếu file ảnh")
	}
	defer file.Close()
	if hdr.Size > maxUploadBytes {
		return fmt.Errorf("ảnh vượt quá 10MB")
	}
	// kiểm tra là ảnh: 512 byte đầu.
	head := make([]byte, 512)
	n, _ := file.Read(head)
	ctype := http.DetectContentType(head[:n])
	if !strings.HasPrefix(ctype, "image/") {
		return fmt.Errorf("file không phải ảnh")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := os.MkdirAll(s.cfg.UploadDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(s.cfg.UploadDir, destName)
	tmp := dest + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	out.Close()
	return os.Rename(tmp, dest) // ghi đè nguyên tử
}

// handleServeUpload phục vụ ảnh đã tải lên: GET /api/uploads/{name}. Công khai, chống path traversal.
func (s *Server) handleServeUpload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	name = filepath.Base(name) // bỏ mọi thành phần đường dẫn
	if name == "" || name == "." || strings.Contains(name, "..") || strings.HasPrefix(name, ".") {
		writeErr(w, http.StatusBadRequest, "tên file không hợp lệ")
		return
	}
	full := filepath.Join(s.cfg.UploadDir, name)
	f, err := os.Open(full)
	if err != nil {
		writeErr(w, http.StatusNotFound, "không tìm thấy ảnh")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		writeErr(w, http.StatusNotFound, "không tìm thấy ảnh")
		return
	}
	// sniff content-type từ nội dung (tên file cố định .jpg vẫn phục vụ đúng kiểu ảnh thật).
	head := make([]byte, 512)
	hn, _ := f.Read(head)
	w.Header().Set("Content-Type", http.DetectContentType(head[:hn]))
	_, _ = f.Seek(0, io.SeekStart)
	// banner có tên duy nhất (bất biến) -> cache dài; ảnh ghi đè cùng tên (popup) -> cache ngắn.
	if strings.HasPrefix(name, "banner-") {
		w.Header().Set("Cache-Control", "public, max-age=2592000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
	http.ServeContent(w, r, name, st.ModTime(), f)
}

// deleteUpload xoá file trong UploadDir (bỏ qua nếu không tồn tại).
func (s *Server) deleteUpload(name string) {
	if name == "" {
		return
	}
	_ = os.Remove(filepath.Join(s.cfg.UploadDir, filepath.Base(name)))
}
