// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestShippedSettingsFileAndStorageSettings loads the settings file Tofu's image ships
// (/config/selfhosted.yaml, a copy of selfhosted.yaml.example) the way the server does,
// with the DT_STORAGE_* settings TOFU.md lists for uploads.
func TestShippedSettingsFileAndStorageSettings(t *testing.T) {
	example, err := os.ReadFile("selfhosted.yaml.example")
	if err != nil {
		t.Fatalf("read selfhosted.yaml.example: %v", err)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config", "selfhosted.yaml"), example, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("DT_ENV", "selfhosted")
	t.Setenv("DT_JWT_SECRET", "a-test-only-secret-that-is-long-enough-0123456789")
	settings := map[string]string{
		"DT_STORAGE_BUCKET_NAME":        "donetick-files",
		"DT_STORAGE_REGION":             "auto",
		"DT_STORAGE_ENDPOINT":           "https://storage.example.test",
		"DT_STORAGE_ACCESS_KEY":         "test-access-key",
		"DT_STORAGE_SECRET_KEY":         "test-secret-key",
		"DT_STORAGE_BASE_PATH":          "donetick",
		"DT_STORAGE_PATH_STYLE":         "true",
		"DT_STORAGE_PUBLIC_BUCKET_NAME": "donetick-photos",
		"DT_STORAGE_PUBLIC_BUCKET_HOST": "photos.example.test",
		"DT_STORAGE_MAX_USER_STORAGE":   "2147483648",
	}
	for key, value := range settings {
		t.Setenv(key, value)
	}

	cfg := LoadConfig()

	s := cfg.Storage
	if s.MaxFileSize != 4194304 {
		t.Errorf("max_file_size = %d, want the file's 4194304", s.MaxFileSize)
	}
	if s.MaxUserStorage != 2147483648 {
		t.Errorf("max_user_storage = %d, want DT_STORAGE_MAX_USER_STORAGE's 2147483648", s.MaxUserStorage)
	}
	got := map[string]string{
		"DT_STORAGE_BUCKET_NAME":        s.BucketName,
		"DT_STORAGE_REGION":             s.Region,
		"DT_STORAGE_ENDPOINT":           s.Endpoint,
		"DT_STORAGE_ACCESS_KEY":         s.AccessKey,
		"DT_STORAGE_SECRET_KEY":         s.SecretKey,
		"DT_STORAGE_BASE_PATH":          s.BasePath,
		"DT_STORAGE_PUBLIC_BUCKET_NAME": s.PublicBucketName,
		"DT_STORAGE_PUBLIC_BUCKET_HOST": s.PublicBucketHost,
	}
	for key, value := range got {
		if value != settings[key] {
			t.Errorf("%s: got %q, want %q", key, value, settings[key])
		}
	}
	if !s.PathStyle {
		t.Error("DT_STORAGE_PATH_STYLE: got false, want true")
	}
}
