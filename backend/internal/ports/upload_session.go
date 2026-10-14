package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUploadSessionNotFound    = errors.New("upload session not found")
	ErrUploadSessionUnavailable = errors.New("upload session store unavailable")
)

// UploadPartState is the storage-neutral state persisted for one uploaded part.
type UploadPartState struct {
	Index int
	ETag  string
	Hash  string
	Size  int64
}

// UploadSession is the application representation of a resumable upload.
// Redis key/field names deliberately do not leak into the application layer.
type UploadSession struct {
	UserID      int
	FileHash    string
	UploadID    string
	ObjectKey   string
	FileName    string
	ParentID    string
	FileSize    int64
	ChunkSize   int64
	TotalChunks int
	ExpiresAt   time.Time
	Parts       map[int]UploadPartState
}

func (s *UploadSession) Expired(now time.Time) bool {
	return s != nil && !s.ExpiresAt.IsZero() && !s.ExpiresAt.After(now)
}

// UploadSessionRef identifies a session without exposing infrastructure keys.
type UploadSessionRef struct {
	UserID   int
	FileHash string
}

// UploadSessionStore owns resumable-upload state, expiry indexing and the merge
// lock. Implementations may use Redis, a database, or another shared store.
type UploadSessionStore interface {
	Available() bool
	Get(ctx context.Context, userID int, fileHash string) (*UploadSession, error)
	Save(ctx context.Context, session *UploadSession) error
	SavePart(ctx context.Context, userID int, fileHash string, part UploadPartState) error
	Refresh(ctx context.Context, userID int, fileHash string) error
	Delete(ctx context.Context, userID int, fileHash string) error

	AcquireMergeLock(ctx context.Context, userID int, fileHash string, ttl time.Duration) (bool, error)
	ReleaseMergeLock(ctx context.Context, userID int, fileHash string) error

	ListExpired(ctx context.Context, before time.Time, limit int64) ([]UploadSessionRef, error)
}
