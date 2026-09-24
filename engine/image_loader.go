package engine

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadImage loads an image from:
// 1. Data URI base64 (e.g. "data:image/png;base64,iVBORw...")
// 2. Pure base64 string
// 3. HTTP / HTTPS URL
// 4. Local file path (checked in current dir, template/, assets/, or executable dir)
func LoadImage(src string) (image.Image, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, fmt.Errorf("sumber gambar (src) kosong")
	}

	// 1. Data URI (e.g. data:image/png;base64,...)
	if strings.HasPrefix(src, "data:image/") {
		commaIdx := strings.Index(src, ",")
		if commaIdx != -1 {
			rawBase64 := src[commaIdx+1:]
			decoded, err := base64.StdEncoding.DecodeString(rawBase64)
			if err != nil {
				return nil, fmt.Errorf("gagal decode base64 image: %w", err)
			}
			img, _, err := image.Decode(bytes.NewReader(decoded))
			if err != nil {
				return nil, fmt.Errorf("gagal decode format image dari base64: %w", err)
			}
			return img, nil
		}
	}

	// 2. Pure base64 string (starts with base64 without prefix)
	if len(src) > 100 && !strings.ContainsAny(src, "/\\") && !strings.Contains(src, ".") {
		if decoded, err := base64.StdEncoding.DecodeString(src); err == nil {
			if img, _, err := image.Decode(bytes.NewReader(decoded)); err == nil {
				return img, nil
			}
		}
	}

	// 3. HTTP / HTTPS URL
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(src)
		if err != nil {
			return nil, fmt.Errorf("gagal mendownload gambar dari %s: %w", src, err)
		}
		defer resp.Body.Close()
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gagal decode format gambar dari URL %s: %w", src, err)
		}
		return img, nil
	}

	// 4. Local file path search
	candidates := []string{
		src,
		filepath.Join("assets", src),
		filepath.Join("template", src),
		filepath.Join("..", "assets", src),
		filepath.Join("..", "template", src),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, src),
			filepath.Join(exeDir, "assets", src),
			filepath.Join(exeDir, "template", src),
		)
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			f, err := os.Open(p)
			if err != nil {
				continue
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				return nil, fmt.Errorf("gagal membaca format gambar %s: %w", p, err)
			}
			return img, nil
		}
	}

	return nil, fmt.Errorf("gambar tidak ditemukan pada path: %s", src)
}
