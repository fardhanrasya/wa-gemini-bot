package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryService(url string) (*CloudinaryService, error) {
	if url == "" {
		return nil, nil // Return nil jika tidak dikonfigurasi
	}
	cld, err := cloudinary.NewFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi cloudinary: %w", err)
	}
	return &CloudinaryService{cld: cld}, nil
}

// UpscaleImage mengupload gambar ke Cloudinary, memberikan efek e_upscale, lalu mendownload dan mengembalikan hasilnya.
func (s *CloudinaryService) UpscaleImage(ctx context.Context, imgData []byte) ([]byte, error) {
	reader := bytes.NewReader(imgData)
	resp, err := s.cld.Upload.Upload(ctx, reader, uploader.UploadParams{
		Folder: "wa_bot_upscale",
	})
	if err != nil {
		return nil, fmt.Errorf("gagal upload ke cloudinary: %w", err)
	}

	// Gunakan trik string manipulation untuk menyisipkan e_upscale ke URL SecureURL.
	// Contoh: https://res.cloudinary.com/demo/image/upload/v123/folder/img.jpg
	// Menjadi: https://res.cloudinary.com/demo/image/upload/e_upscale/v123/folder/img.jpg
	parts := strings.SplitN(resp.SecureURL, "/upload/", 2)
	var upscaleURL string
	if len(parts) == 2 {
		upscaleURL = parts[0] + "/upload/e_upscale/" + parts[1]
	} else {
		// Fallback jika format tidak dikenali
		upscaleURL = resp.SecureURL
	}

	// Download hasil upscale
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upscaleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request download: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal mendownload hasil upscale: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status download gagal: %d", res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca body: %w", err)
	}

	return bodyBytes, nil
}
