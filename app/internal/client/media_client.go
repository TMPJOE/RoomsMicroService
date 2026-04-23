// Package client provides external service clients for the Rooms MicroService.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// MediaClient defines the interface for interacting with the Media Service
type MediaClient interface {
	UploadFile(ctx context.Context, file io.Reader, filename, assetType, assetID, contentType string) (*UploadResponse, error)
}

// UploadResponse represents the response from the media service upload
type UploadResponse struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// ErrorResponse represents an error response from the media service
type ErrorResponse struct {
	Error string `json:"error"`
}

// mediaClient implements the MediaClient interface
type mediaClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMediaClient creates a new media client with the given base URL
func NewMediaClient(baseURL string) MediaClient {
	return &mediaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadFile uploads a file to the media service
func (m *mediaClient) UploadFile(ctx context.Context, file io.Reader, filename, assetType, assetID, contentType string) (*UploadResponse, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		if err := writer.WriteField("asset_type", assetType); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := writer.WriteField("asset_id", assetID); err != nil {
			pw.CloseWithError(err)
			return
		}

		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		if _, err := io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}

		if err := writer.Close(); err != nil {
			pw.CloseWithError(err)
		}
	}()

	url := fmt.Sprintf("%s/upload", m.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("media service error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("media service returned status %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &uploadResp, nil
}

// NewMediaClientFromEnv creates a media client using environment variables
func NewMediaClientFromEnv() MediaClient {
	baseURL := os.Getenv("MEDIA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://media-service:8080"
	}
	return NewMediaClient(baseURL)
}

// NewMediaClientWithConfig creates a media client with the given base URL or default
func NewMediaClientWithConfig(baseURL string) MediaClient {
	if baseURL == "" {
		baseURL = "http://media-service:8080"
	}
	return NewMediaClient(baseURL)
}
