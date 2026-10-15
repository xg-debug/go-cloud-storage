package application

import (
	"context"
	"go-cloud-storage/backend/pkg/contracts"
	ports "go-cloud-storage/backend/pkg/ports"
	domain "go-cloud-storage/backend/services/share-service/internal/domain"
	"testing"
	"time"
)

type auditShareRepo struct {
	shareListRepo
	share *domain.Share
}

func (r auditShareRepo) GetShareByToken(string) (*domain.Share, error) { return r.share, nil }
func (r auditShareRepo) IncrementDownloadCount(int) error              { return nil }

type auditSigningStorage struct {
	ports.Storage
	ttl time.Duration
}

func (s *auditSigningStorage) PresignedDownloadURL(_ context.Context, _, _ string, ttl time.Duration) (string, error) {
	s.ttl = ttl
	return "signed", nil
}
func TestRegressionSignedURLMustNotOutliveShare(t *testing.T) {
	expires := time.Now().Add(time.Minute)
	st := &auditSigningStorage{}
	app := NewShareService(auditShareRepo{share: &domain.Share{Id: 1, UserId: 1, FileId: "f", FileVersion: 2, ExpireTime: &expires}}, shareFiles{}, st, nil)
	if _, err := app.DownloadSharedFile("token", ""); err != nil {
		t.Fatal(err)
	}
	if st.ttl > time.Until(expires) {
		t.Fatalf("share expires in 1 minute, signed URL TTL=%v", st.ttl)
	}
}

func (r auditShareRepo) IncrementAccessCount(int) error { return nil }

type previewShareFiles struct{ shareFiles }

func (previewShareFiles) GetUserFileByID(int, string) (*contracts.FileMetadata, error) {
	return &contracts.FileMetadata{Id: "f", UserId: 1, ShareVersion: 2, OssObjectKey: "key", FileURL: "object", ThumbnailURL: "thumb", FileExtension: "png"}, nil
}

type previewSigningStorage struct {
	auditSigningStorage
	previewTTL, thumbTTL time.Duration
}

func (s *previewSigningStorage) PresignedGetPreviewURL(_ context.Context, _ string, ttl time.Duration) (string, error) {
	s.previewTTL = ttl
	return "preview", nil
}
func (s *previewSigningStorage) PresignStoredObjectURL(_ context.Context, _ string, ttl time.Duration) string {
	s.thumbTTL = ttl
	return "thumbnail"
}
func TestSharePreviewAndThumbnailRespectExpiry(t *testing.T) {
	expiry := time.Now().Add(time.Minute)
	storage := &previewSigningStorage{}
	app := NewShareService(auditShareRepo{share: &domain.Share{Id: 1, UserId: 1, FileId: "f", FileVersion: 2, ExpireTime: &expiry}}, previewShareFiles{}, storage, nil)
	response, err := app.AccessShare("token", "")
	if err != nil {
		t.Fatal(err)
	}
	if response.FileURL == "" || response.ThumbnailURL == "" {
		t.Fatal("URLs missing")
	}
	for _, ttl := range []time.Duration{storage.previewTTL, storage.thumbTTL} {
		if ttl < time.Second || ttl > time.Until(expiry) {
			t.Fatalf("invalid signing TTL %v", ttl)
		}
	}
}
func TestShareURLTTLRejectsExpiredAndSubsecondShares(t *testing.T) {
	for _, delta := range []time.Duration{-time.Minute, 500 * time.Millisecond} {
		expiry := time.Now().Add(delta)
		if _, err := shareURLTTL(&expiry, time.Hour); err == nil {
			t.Fatal("expired/subsecond share allowed")
		}
	}
	if ttl, err := shareURLTTL(nil, time.Hour); err != nil || ttl != time.Hour {
		t.Fatal("unlimited share changed")
	}
}
