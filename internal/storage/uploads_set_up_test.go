// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.

package storage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"donetick.com/core/config"
	"github.com/gin-gonic/gin"
)

func newTestS3Storage(t *testing.T, storageConfig config.StorageConfig) *S3Storage {
	t.Helper()
	// The S3 client also reads a region from the environment; keep it out of the test.
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	s, err := NewS3Storage(&config.Config{Storage: storageConfig})
	if err != nil {
		t.Fatalf("NewS3Storage: %v", err)
	}
	return s
}

func TestUploadsSetUp(t *testing.T) {
	cases := []struct {
		name            string
		storage         config.StorageConfig
		files, profiles bool
	}{
		{"no settings", config.StorageConfig{}, false, false},
		{"bucket without a region", config.StorageConfig{BucketName: "files"}, false, false},
		{"region without a bucket", config.StorageConfig{Region: "auto"}, false, false},
		{"bucket and region", config.StorageConfig{BucketName: "files", Region: "auto"}, true, true},
		{"photo bucket alone", config.StorageConfig{PublicBucketName: "photos", Region: "auto"}, false, true},
	}
	for _, tc := range cases {
		s := newTestS3Storage(t, tc.storage)
		if got := UploadsSetUp(s, false); got != tc.files {
			t.Errorf("%s: UploadsSetUp(files) = %v, want %v", tc.name, got, tc.files)
		}
		if got := UploadsSetUp(s, true); got != tc.profiles {
			t.Errorf("%s: UploadsSetUp(profile photos) = %v, want %v", tc.name, got, tc.profiles)
		}
	}
	local := NewLocalStorage(&config.Config{Storage: config.StorageConfig{StorageType: "local"}})
	if !UploadsSetUp(local, false) || !UploadsSetUp(local, true) {
		t.Error("local storage needs no bucket")
	}
}

func TestChoreUploadWithoutBucketAnswersTheSentence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{storage: newTestS3Storage(t, config.StorageConfig{}), maxFileSize: 1 << 20}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/assets/chore", nil)

	h.ChoreUploadHandler(c)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q: %v", recorder.Body.String(), err)
	}
	if body["error"] != UploadsNotSetUpMessage || body["code"] != UploadsNotSetUpCode {
		t.Errorf("body = %v", body)
	}
}
