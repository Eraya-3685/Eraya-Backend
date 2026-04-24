package storage

import (
	"bytes"
	"eraya/config"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type StorageService struct {
	baseURL string
	key     string
	bucket  string
}

func NewStorageService(supabaseConfig config.SupabaseConfig) *StorageService {
	return &StorageService{
		baseURL: supabaseConfig.URL,
		key:     supabaseConfig.Key,
		bucket:  supabaseConfig.Bucket,
	}
}

func (s *StorageService) UploadFile(folder, filename string, content io.Reader, contentType string) (string, error) {
	// Generate unique filename with timestamp
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	fullPath := filepath.Join(folder, uniqueFilename)

	// API URL for upload
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, fullPath)

	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("apikey", s.key)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload to supabase: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Return relative path with leading slash
	return "/" + strings.ReplaceAll(fullPath, "\\", "/"), nil
}

func (s *StorageService) DeleteFile(input string) error {
	var fullPath string

	// Check if input is a full URL or a relative path
	if strings.HasPrefix(input, "http") {
		// Public URL looks like: {URL}/storage/v1/object/public/{bucket}/{path}
		searchStr := fmt.Sprintf("/storage/v1/object/public/%s/", s.bucket)
		idx := strings.Index(input, searchStr)
		if idx == -1 {
			// Not a Supabase public URL we recognize, maybe it's just the filename?
			// Best effort: just take the last part
			parts := strings.Split(input, "/")
			fullPath = parts[len(parts)-1]
		} else {
			fullPath = input[idx+len(searchStr):]
		}
	} else {
		// It's a relative path, remove leading slash
		fullPath = strings.TrimPrefix(input, "/")
	}

	// Delete URL format: {URL}/storage/v1/object/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, fullPath)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("apikey", s.key)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete from supabase: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
