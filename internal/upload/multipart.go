package upload

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxConcurrentUploads = 2
	maxRetries           = 8
	retryDelayMS         = 2000
	bodyTimeoutMS        = 300000
)

// PresignedURL represents a presigned URL for a part upload
type PresignedURL struct {
	PartNumber int    `json:"partNumber"`
	URL        string `json:"url"`
}

// CompletedPart represents a completed part upload
type CompletedPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"etag"`
}

// ProgressCallback is called during upload with progress updates
type ProgressCallback func(completed, total int, completedBytes, totalBytes int64)

// UploadParts uploads file parts to S3 using presigned URLs
func UploadParts(presignedUrls []PresignedURL, fileBuffer []byte, partSize int, onProgress ProgressCallback) ([]CompletedPart, error) {
	totalParts := len(presignedUrls)
	totalBytes := int64(len(fileBuffer))
	completedParts := make([]CompletedPart, 0, totalParts)
	var completedBytes int64

	// Process uploads in batches to limit concurrency
	for i := 0; i < len(presignedUrls); i += maxConcurrentUploads {
		end := i + maxConcurrentUploads
		if end > len(presignedUrls) {
			end = len(presignedUrls)
		}
		batch := presignedUrls[i:end]

		// Upload batch in parallel
		results := make(chan struct {
			part CompletedPart
			size int64
			err  error
		}, len(batch))

		for idx, pu := range batch {
			go func(pu PresignedURL, idx int) {
				start := (pu.PartNumber - 1) * partSize
				endIdx := start + partSize
				if endIdx > len(fileBuffer) {
					endIdx = len(fileBuffer)
				}
				partData := fileBuffer[start:endIdx]

				// Small delay between starting concurrent uploads
				if idx > 0 {
					time.Sleep(100 * time.Millisecond * time.Duration(idx))
				}

				etag, err := uploadPart(pu.URL, partData, pu.PartNumber)
				results <- struct {
					part CompletedPart
					size int64
					err  error
				}{
					part: CompletedPart{PartNumber: pu.PartNumber, ETag: etag},
					size: int64(len(partData)),
					err:  err,
				}
			}(pu, idx)
		}

		// Collect batch results
		for range batch {
			result := <-results
			if result.err != nil {
				return nil, result.err
			}
			completedParts = append(completedParts, result.part)
			completedBytes += result.size

			if onProgress != nil {
				onProgress(len(completedParts), totalParts, completedBytes, totalBytes)
			}
		}
	}

	// Sort by part number
	sortParts(completedParts)

	return completedParts, nil
}

func uploadPart(presignedURL string, data []byte, partNumber int) (string, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		client := &http.Client{
			Timeout: time.Duration(bodyTimeoutMS) * time.Millisecond,
		}

		req, err := http.NewRequest("PUT", presignedURL, bytes.NewReader(data))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
		req.ContentLength = int64(len(data))

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			isConnectionError := strings.Contains(err.Error(), "EPIPE") ||
				strings.Contains(err.Error(), "ECONNRESET") ||
				strings.Contains(err.Error(), "timeout")

			baseDelay := time.Duration(retryDelayMS) * time.Millisecond
			if isConnectionError {
				baseDelay *= 3
			}

			if attempt < maxRetries-1 {
				time.Sleep(baseDelay * time.Duration(attempt+1))
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))

			if attempt < maxRetries-1 {
				time.Sleep(time.Duration(retryDelayMS) * time.Millisecond * time.Duration(attempt+1))
			}
			continue
		}

		// Get ETag from response headers
		etag := resp.Header.Get("ETag")
		if etag == "" {
			lastErr = fmt.Errorf("no ETag in response headers")
			continue
		}

		return etag, nil
	}

	return "", fmt.Errorf("failed to upload part %d after %d attempts: %v", partNumber, maxRetries, lastErr)
}

func sortParts(parts []CompletedPart) {
	// Simple insertion sort for small arrays
	for i := 1; i < len(parts); i++ {
		key := parts[i]
		j := i - 1
		for j >= 0 && parts[j].PartNumber > key.PartNumber {
			parts[j+1] = parts[j]
			j--
		}
		parts[j+1] = key
	}
}

// GetMimeType returns the MIME type for a file based on its extension
func GetMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	// Remove charset suffix if present
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	return mimeType
}
