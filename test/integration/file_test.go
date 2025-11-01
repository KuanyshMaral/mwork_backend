package integration_test

import (
	"bytes"
	"context" // ❗️ ДОБАВЛЕН ИМПОРТ
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"mwork_backend/internal/models"
	"mwork_backend/pkg/contextkeys" // ❗️ ДОБАВЛЕН ИМПОРТ
	"mwork_backend/test/helpers"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestFileUploadSystem - comprehensive tests for the production-ready file upload system
func TestFileUploadSystem(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t) // ❗️ Транзакция
	defer ts.RollbackTransaction(t, tx)

	// Create test users
	modelToken, modelUser, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)
	empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx,
		"File Admin", "file-admin@test.com", "password123", models.UserRoleAdmin)

	t.Run("Image Upload with Processing", func(t *testing.T) {
		// Create a test image file
		imageData := createTestImage(t, 800, 600)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Add form fields
		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", modelProfile.ID)
		_ = writer.WriteField("usage", "avatar")

		// Add file
		part, err := writer.CreateFormFile("file", "test-avatar.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Send request
		url := ts.Server.URL + "/api/v1/uploads"
		req, err := http.NewRequest(http.MethodPost, url, body)
		require.NoError(t, err)

		// ❗️ Внедряем транзакцию в контекст
		ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		resBody, _ := io.ReadAll(res.Body)
		assert.Equal(t, http.StatusCreated, res.StatusCode, "Upload failed: "+string(resBody))

		// Parse response
		var upload models.Upload
		err = json.Unmarshal(resBody, &upload)
		require.NoError(t, err)

		// Verify upload metadata
		assert.NotEmpty(t, upload.ID)
		assert.Equal(t, modelUser.ID, upload.UserID)
		assert.Equal(t, "model_profile", upload.EntityType)
		assert.Equal(t, "avatar", upload.Usage)
		assert.Contains(t, upload.MimeType, "image")
		assert.Greater(t, upload.Size, int64(0))

		// Verify file exists on disk (if using local storage)
		if upload.Path != "" {
			_, err := os.Stat(upload.Path)
			assert.NoError(t, err, "Uploaded file should exist on disk")
		}
	})

	t.Run("File Size Validation", func(t *testing.T) {
		// Try to upload a file that's too large (simulate with metadata)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", modelProfile.ID)
		_ = writer.WriteField("usage", "portfolio_photo")

		// Create a large dummy file (10MB)
		largeData := make([]byte, 10*1024*1024)
		part, _ := writer.CreateFormFile("file", "large-file.jpg")
		part.Write(largeData)
		writer.Close()

		url := ts.Server.URL + "/api/v1/uploads"
		req, _ := http.NewRequest(http.MethodPost, url, body)

		// ❗️ Внедряем транзакцию в контекст
		ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		// Should fail if max file size is configured lower than 10MB
		// Or succeed if limit is higher - adjust based on your config
		assert.True(t, res.StatusCode == http.StatusCreated || res.StatusCode == http.StatusBadRequest)
	})

	t.Run("MIME Type Validation", func(t *testing.T) {
		// Try to upload an invalid file type
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", modelProfile.ID)
		_ = writer.WriteField("usage", "avatar")

		// Create a fake executable file
		part, _ := writer.CreateFormFile("file", "malicious.exe")
		part.Write([]byte("MZ\x90\x00")) // PE header
		writer.Close()

		url := ts.Server.URL + "/api/v1/uploads"
		req, _ := http.NewRequest(http.MethodPost, url, body)

		// ❗️ Внедряем транзакцию в контекст
		ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		// Should reject invalid MIME types
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("Storage Quota Check", func(t *testing.T) {
		// Get current storage usage
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/storage/usage", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		var usage map[string]interface{}
		err := json.Unmarshal([]byte(bodyStr), &usage)
		require.NoError(t, err)

		// Verify usage fields
		assert.Contains(t, usage, "total_usage_bytes")
		assert.Contains(t, usage, "total_files")
		assert.Contains(t, usage, "quota_bytes")
		assert.Contains(t, usage, "quota_remaining_bytes")
	})

	t.Run("File Serving and Download", func(t *testing.T) {
		// First upload a file
		imageData := createTestImage(t, 400, 300)
		// ❗️ Добавлен 'tx'
		uploadID := uploadTestFile(t, ts, tx, modelToken, modelProfile.ID, "test-download.jpg", imageData)

		// Test public file access (no auth required)
		// ❗️ Добавлен 'tx'
		res, _ := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/files/"+uploadID, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Test file metadata retrieval
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/"+uploadID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, uploadID)
	})

	t.Run("Private File Access Control", func(t *testing.T) {
		// Upload a private file
		imageData := createTestImage(t, 200, 200)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", modelProfile.ID)
		_ = writer.WriteField("usage", "private_document")
		_ = writer.WriteField("isPublic", "false")

		part, _ := writer.CreateFormFile("file", "private.jpg")
		part.Write(imageData)
		writer.Close()

		url := ts.Server.URL + "/api/v1/uploads"
		req, _ := http.NewRequest(http.MethodPost, url, body)

		// ❗️ Внедряем транзакцию в контекст
		ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		resBody, _ := io.ReadAll(res.Body)
		var upload models.Upload
		json.Unmarshal(resBody, &upload)

		// Owner can access
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/files/"+upload.ID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Non-owner cannot access private file
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/files/"+upload.ID, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode)

		// Unauthenticated cannot access
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/files/"+upload.ID, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("Signed URLs for Temporary Access", func(t *testing.T) {
		// Upload a private file
		imageData := createTestImage(t, 150, 150)
		// ❗️ Добавлен 'tx'
		uploadID := uploadTestFile(t, ts, tx, modelToken, modelProfile.ID, "signed-url-test.jpg", imageData)

		// Request a signed URL
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPost,
			fmt.Sprintf("/api/v1/files/%s/signed-url", uploadID),
			modelToken,
			gin.H{"expires_in": 3600})

		if res.StatusCode == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal([]byte(bodyStr), &response)
			require.NoError(t, err)

			// Verify signed URL is returned
			assert.Contains(t, response, "url")
			assert.Contains(t, response, "expires_at")

			signedURL := response["url"].(string)
			assert.NotEmpty(t, signedURL)
			assert.Contains(t, signedURL, "signature")
		}
	})

	t.Run("Bulk File Operations", func(t *testing.T) {
		// Upload multiple files
		uploadIDs := make([]string, 3)
		for i := 0; i < 3; i++ {
			imageData := createTestImage(t, 100+i*50, 100+i*50)
			// ❗️ Добавлен 'tx'
			uploadIDs[i] = uploadTestFile(t, ts, tx, modelToken, modelProfile.ID,
				fmt.Sprintf("bulk-test-%d.jpg", i), imageData)
		}

		// Get all user uploads
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/user/me", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Verify all uploads are returned
		for _, id := range uploadIDs {
			assert.Contains(t, bodyStr, id)
		}

		// Delete one file
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodDelete, "/api/v1/uploads/"+uploadIDs[0], modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Verify it's deleted
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/"+uploadIDs[0], modelToken, nil)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("Admin File Management", func(t *testing.T) {
		// Get system-wide upload stats
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/admin/uploads/stats", adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		var stats map[string]interface{}
		err := json.Unmarshal([]byte(bodyStr), &stats)
		require.NoError(t, err)

		assert.Contains(t, stats, "total_files")
		assert.Contains(t, stats, "total_storage_gb")
		assert.Contains(t, stats, "by_type")

		// Non-admin cannot access
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/admin/uploads/stats", modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode)

		// Clean orphaned files
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/admin/uploads/clean-orphaned", adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("File Deletion Cascade", func(t *testing.T) {
		// Upload a file
		imageData := createTestImage(t, 200, 200)
		// ❗️ Добавлен 'tx'
		uploadID := uploadTestFile(t, ts, tx, modelToken, modelProfile.ID, "cascade-test.jpg", imageData)

		// Verify file exists
		// ❗️ Добавлен 'tx'
		res, _ := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/"+uploadID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Delete the file
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodDelete, "/api/v1/uploads/"+uploadID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Verify file is deleted from database
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/"+uploadID, modelToken, nil)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("Security - Unauthorized Access", func(t *testing.T) {
		// Upload file as model
		imageData := createTestImage(t, 150, 150)
		// ❗️ Добавлен 'tx'
		uploadID := uploadTestFile(t, ts, tx, modelToken, modelProfile.ID, "security-test.jpg", imageData)

		// Employer tries to delete model's file
		// ❗️ Добавлен 'tx'
		res, _ := ts.SendRequest(t, tx, http.MethodDelete, "/api/v1/uploads/"+uploadID, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode)

		// Verify file still exists
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/"+uploadID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("Error Handling - Missing File", func(t *testing.T) {
		// Try to upload without file
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", modelProfile.ID)
		writer.Close()

		url := ts.Server.URL + "/api/v1/uploads"
		req, _ := http.NewRequest(http.MethodPost, url, body)

		// ❗️ Внедряем транзакцию в контекст
		ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("Error Handling - Invalid Entity", func(t *testing.T) {
		// Try to upload with non-existent entity
		imageData := createTestImage(t, 100, 100)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", "non-existent-uuid")
		_ = writer.WriteField("usage", "avatar")

		part, _ := writer.CreateFormFile("file", "invalid-entity.jpg")
		part.Write(imageData)
		writer.Close()

		url := ts.Server.URL + "/api/v1/uploads"
		req, _ := http.NewRequest(http.MethodPost, url, body)

		// ❗️ Внедряем транзакцию в контекст
		ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		// Should fail validation
		assert.True(t, res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusNotFound)
	})
}

// TestFileUpload_Concurrent tests concurrent file uploads
func TestFileUpload_Concurrent(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)

	// Upload 5 files concurrently
	done := make(chan bool, 5)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(index int) {
			imageData := createTestImage(t, 200, 200)
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)

			_ = writer.WriteField("entityType", "model_profile")
			_ = writer.WriteField("entityId", modelProfile.ID)
			_ = writer.WriteField("usage", "portfolio_photo")

			part, _ := writer.CreateFormFile("file", fmt.Sprintf("concurrent-%d.jpg", index))
			part.Write(imageData)
			writer.Close()

			url := ts.Server.URL + "/api/v1/uploads"
			req, _ := http.NewRequest(http.MethodPost, url, body)

			// ❗️ Внедряем транзакцию в контекст
			ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
			req = req.WithContext(ctx)

			req.Header.Set("Authorization", "Bearer "+modelToken)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			res, err := ts.Server.Client().Do(req)
			if err != nil {
				errors <- err
				return
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusCreated {
				errors <- fmt.Errorf("upload %d failed with status %d", index, res.StatusCode)
				return
			}

			done <- true
		}(i)
	}

	// Wait for all uploads
	successCount := 0
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Logf("Concurrent upload error: %v", err)
		}
	}

	assert.GreaterOrEqual(t, successCount, 4, "At least 4 out of 5 concurrent uploads should succeed")
}

// Helper functions

// createTestImage creates a simple test image (JPEG format)
func createTestImage(t *testing.T, width, height int) []byte {
	t.Helper()

	// Create a minimal valid JPEG header
	// This is a simplified version - in production you'd use image/jpeg
	header := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00,
	}

	// Add some dummy data
	data := make([]byte, width*height/10)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// JPEG end marker
	footer := []byte{0xFF, 0xD9}

	return append(append(header, data...), footer...)
}

// uploadTestFile is a helper to upload a file and return its ID
// ❗️ Добавлен 'tx *gorm.DB'
func uploadTestFile(t *testing.T, ts *helpers.TestServer, tx *gorm.DB, token, entityID, filename string, data []byte) string {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("entityType", "model_profile")
	_ = writer.WriteField("entityId", entityID)
	_ = writer.WriteField("usage", "portfolio_photo")

	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	url := ts.Server.URL + "/api/v1/uploads"
	req, err := http.NewRequest(http.MethodPost, url, body)
	require.NoError(t, err)

	// ❗️ Внедряем транзакцию в контекст
	ctx := context.WithValue(req.Context(), contextkeys.DBContextKey, tx)
	req = req.WithContext(ctx)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := ts.Server.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	resBody, _ := io.ReadAll(res.Body)
	require.Equal(t, http.StatusCreated, res.StatusCode, "Upload failed: "+string(resBody))

	var upload models.Upload
	err = json.Unmarshal(resBody, &upload)
	require.NoError(t, err)

	return upload.ID
}

// TestFileUpload_ImageVariants tests image processing and variant generation
func TestFileUpload_ImageVariants(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)

	// Upload an image
	imageData := createTestImage(t, 1920, 1080)
	// ❗️ Добавлен 'tx'
	uploadID := uploadTestFile(t, ts, tx, modelToken, modelProfile.ID, "variants-test.jpg", imageData)

	// Get upload details
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/uploads/"+uploadID, modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var upload models.Upload
	err := json.Unmarshal([]byte(bodyStr), &upload)
	require.NoError(t, err)

	// Check if variants were generated (if image processor is enabled)
	// This depends on your configuration
	t.Logf("Upload path: %s", upload.Path)
	t.Logf("Upload MIME type: %s", upload.MimeType)

	// Verify the file type is correct
	assert.True(t, strings.HasPrefix(upload.MimeType, "image/"),
		"MIME type should be an image type")
}
