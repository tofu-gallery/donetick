// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.

package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"donetick.com/core/config"
)

// TestS3DeleteRemovesTheObjectSaveWrote: Delete names the same object Save wrote, with
// or without a base path, against a stand-in S3 endpoint.
func TestS3DeleteRemovesTheObjectSaveWrote(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.EscapedPath())
		mu.Unlock()
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer endpoint.Close()

	for _, basePath := range []string{"donetick", ""} {
		seen = nil
		s := newTestS3Storage(t, config.StorageConfig{
			BucketName: "files", Region: "auto", Endpoint: endpoint.URL, PathStyle: true,
			AccessKey: "test-access-key", SecretKey: "test-secret-key", BasePath: basePath,
		})
		ctx := context.Background()
		if err := s.Save(ctx, "users/1/file.png", strings.NewReader("png")); err != nil {
			t.Fatalf("base path %q: Save: %v", basePath, err)
		}
		if err := s.Delete(ctx, []string{"users/1/file.png"}); err != nil {
			t.Fatalf("base path %q: Delete: %v", basePath, err)
		}
		if len(seen) != 2 || strings.TrimPrefix(seen[0], "PUT ") != strings.TrimPrefix(seen[1], "DELETE ") {
			t.Errorf("base path %q: requests %q, want a PUT and a DELETE of the same object", basePath, seen)
		}
	}
}
