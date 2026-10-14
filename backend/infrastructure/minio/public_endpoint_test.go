package minio

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	sdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestPublicEndpointSignsForBrowserWithoutContactingInternalHost(t *testing.T) {
	signer, err := sdk.New("storage.example.test:9000", &sdk.Options{Creds: credentials.NewStaticV4("access", "secret-secret", ""), Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	service := &MinioService{signer: signer, bucket: "private", endpoint: "storage.example.test:9000"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	value, err := service.PresignedDownloadURL(ctx, "files/7/test.txt", "test.txt", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "storage.example.test:9000" || u.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("invalid public signature URL", u.Host)
	}
	if !strings.HasPrefix(service.GenerateObjectURL("files/7/test.txt"), "http://storage.example.test:9000/private/") {
		t.Fatal("internal endpoint escaped into metadata")
	}
}
