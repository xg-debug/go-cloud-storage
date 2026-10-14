package config

import (
	"os"
	"testing"
)

func TestRemoteStorageEnvironmentWithoutMinioCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("go-cloud-storage.dev.yaml", []byte(`server:
  storagePath: /tmp/upload
  allowedOrigins: [http://localhost:8080]
database:
  host: mysql
  dbname: file-store
`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GCS_STORAGE_URL", "http://storage-service:8082")
	t.Setenv("GCS_STORAGE_TOKEN", "test-storage-token-at-least-32-bytes")
	t.Setenv("GCS_TRUSTED_PROXIES", "172.30.0.2")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StorageService.URL != "http://storage-service:8082" || cfg.StorageService.Token != "test-storage-token-at-least-32-bytes" {
		t.Fatal("remote storage environment not bound")
	}
	if len(cfg.Server.TrustedProxies) != 1 || cfg.Server.TrustedProxies[0] != "172.30.0.2" {
		t.Fatal("proxy trust not bound")
	}
	t.Setenv("GCS_TRUSTED_PROXIES", "invalid-proxy")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid proxy accepted")
	}
}
