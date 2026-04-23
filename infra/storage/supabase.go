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
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().Unix(), filename)
	fullPath := filepath.Join(folder, uniqueFilename)

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

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, s.bucket, fullPath)
	return publicURL, nil
}

// DeleteFile deletes a file from the bucket using its public URL
func (s *StorageService) DeleteFile(publicURL string) error {
	// Extract the path from public URL
	// publicURL looks like: {URL}/storage/v1/object/public/{bucket}/{path}
	searchStr := fmt.Sprintf("/storage/v1/object/public/%s/", s.bucket)
	idx := strings.Index(publicURL, searchStr)
	if idx == -1 {
		return fmt.Errorf("invalid public url")
	}

	fullPath := publicURL[idx+len(searchStr):]

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
