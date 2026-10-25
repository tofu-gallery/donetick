// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.

package user

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"donetick.com/core/config"
	storage "donetick.com/core/internal/storage"
	"github.com/gin-gonic/gin"
)

// TestProfilePhotoWithoutBucketAnswersTheSentence: a copy with no storage bucket answers
// a profile photo upload with 503 and the sentence that says the owner has to add one
// (internal/storage/uploads_set_up.go), instead of 500 "Failed to save file".
func TestProfilePhotoWithoutBucketAnswersTheSentence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	s, err := storage.NewS3Storage(&config.Config{})
	if err != nil {
		t.Fatalf("NewS3Storage: %v", err)
	}
	h := &Handler{storage: s}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users/profile_photo", nil)

	h.updateProfilePhoto(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), storage.UploadsNotSetUpCode) {
		t.Errorf("expected the %q code, got %s", storage.UploadsNotSetUpCode, w.Body.String())
	}
}
