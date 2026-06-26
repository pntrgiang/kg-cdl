package server

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif" // đăng ký decoder gif
	_ "image/png" // đăng ký decoder png

	"github.com/go-chi/chi/v5"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // đăng ký decoder webp
)

const (
	maxUploadBytes = 10 << 20 // 10MB (giới hạn file tải lên)
	maxImageDim    = 1600     // cạnh dài tối đa (px) sau khi nén
	jpegQuality    = 82       // chất lượng JPEG khi nén lại
)

// compressImage giải mã ảnh, thu nhỏ nếu cạnh dài > maxImageDim, dán lên nền trắng (xử lý alpha),
// rồi mã hóa lại JPEG (jpegQuality). Giải mã lỗi hoặc nén ra lớn hơn -> trả về dữ liệu GỐC.
func compressImage(data []byte) []byte {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data // không giải mã được -> giữ nguyên (không phá upload)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return data
	}
	nw, nh := w, h
	if w > maxImageDim || h > maxImageDim {
		if w >= h {
			nw, nh = maxImageDim, h*maxImageDim/w
		} else {
			nw, nh = w*maxImageDim/h, maxImageDim
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src) // nền trắng -> loại alpha
	if nw == w && nh == h {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	} else {
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil) // scale chất lượng cao
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return data
	}
	// nếu không thu nhỏ kích thước mà file nén lại còn lớn hơn (ảnh đã tối ưu sẵn) -> giữ gốc
	if nw == w && nh == h && out.Len() >= len(data) {
		return data
	}
	return out.Bytes()
}

// saveUpload đọc file ảnh từ multipart form (field), NÉN/RESIZE lại, rồi lưu vào UploadDir với tên destName.
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
	raw, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	data := compressImage(raw) // nén lại trước khi lưu (lỗi giải mã -> giữ nguyên)

	if err := os.MkdirAll(s.cfg.UploadDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(s.cfg.UploadDir, destName)
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
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
