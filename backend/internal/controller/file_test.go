package controller

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/internal/services"
)

type previewStub struct{ services.FileService }

func (previewStub) PreviewFile(int, string) (*services.FilePreview, error) {
	return &services.FilePreview{PreviewType: "pdf", FileURL: "https://object.example/file"}, nil
}
func TestPDFPreviewReturnsProxyPathWithoutTrustingHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := &FileController{fileService: previewStub{}}
	router.GET("/file/preview/:fileId", ctrl.PreviewFile)
	req := httptest.NewRequest("GET", "http://untrusted.example/file/preview/abc", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	var payload struct{ Data services.FilePreview }
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || payload.Data.ProxyPath != "/file/preview-stream/abc" {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}
