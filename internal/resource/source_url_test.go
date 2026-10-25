// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream
// Donetick. It is licensed under the AGPL-3.0, like the rest of this server.

package resource

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"donetick.com/core/config"
	"github.com/gin-gonic/gin"
)

// TestResourceExposesSourceURL verifies the resource endpoint names the source
// code address the web app links to, and leaves the field out when none is set.
func TestResourceExposesSourceURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, want := range []string{"https://github.com/tofu-gallery/donetick/tree/pin/example", ""} {
		h := NewHandler(&config.Config{SourceURL: want})
		r := gin.New()
		r.GET("api/v1/resource", h.getResource)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got %d", w.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		got, present := body["source_url"]
		if want == "" {
			if present {
				t.Errorf("source_url = %v, want it left out", got)
			}
			continue
		}
		if got != want {
			t.Errorf("source_url = %v, want %q", got, want)
		}
	}
}
