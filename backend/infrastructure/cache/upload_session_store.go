package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-cloud-storage/backend/internal/ports"

	"github.com/go-redis/redis/v8"
)

const uploadSessionSet = "upload:sessions"

type RedisUploadSessionStore struct {
	client      *redis.Client
	activeTTL   time.Duration
	metadataTTL time.Duration
}

func NewRedisUploadSessionStore(client *redis.Client, activeTTL, metadataTTL time.Duration) *RedisUploadSessionStore {
	return &RedisUploadSessionStore{client: client, activeTTL: activeTTL, metadataTTL: metadataTTL}
}

func (s *RedisUploadSessionStore) Available() bool {
	return s != nil && s.client != nil
}

func (s *RedisUploadSessionStore) Get(ctx context.Context, userID int, fileHash string) (*ports.UploadSession, error) {
	if !s.Available() {
		return nil, ports.ErrUploadSessionUnavailable
	}
	fields, err := s.client.HGetAll(ctx, uploadSessionKey(userID, fileHash)).Result()
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, ports.ErrUploadSessionNotFound
	}

	session, err := decodeUploadSession(userID, fileHash, fields)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *RedisUploadSessionStore) Save(ctx context.Context, session *ports.UploadSession) error {
	if !s.Available() {
		return ports.ErrUploadSessionUnavailable
	}
	if session == nil {
		return fmt.Errorf("upload session is nil")
	}
	if session.Parts == nil {
		session.Parts = make(map[int]ports.UploadPartState)
	}

	key := uploadSessionKey(session.UserID, session.FileHash)
	values := []interface{}{
		"id", session.UploadID,
		"key", session.ObjectKey,
		"fileName", session.FileName,
		"parentId", session.ParentID,
		"fileSize", strconv.FormatInt(session.FileSize, 10),
		"chunkSize", strconv.FormatInt(session.ChunkSize, 10),
		"totalChunks", strconv.Itoa(session.TotalChunks),
	}
	for idx, part := range session.Parts {
		values = append(values,
			strconv.Itoa(idx), part.ETag,
			strconv.Itoa(idx)+"_hash", part.Hash,
			strconv.Itoa(idx)+"_size", strconv.FormatInt(part.Size, 10),
		)
	}
	if err := s.client.HSet(ctx, key, values...).Err(); err != nil {
		return err
	}
	return s.Refresh(ctx, session.UserID, session.FileHash)
}

func (s *RedisUploadSessionStore) SavePart(ctx context.Context, userID int, fileHash string, part ports.UploadPartState) error {
	if !s.Available() {
		return ports.ErrUploadSessionUnavailable
	}
	key := uploadSessionKey(userID, fileHash)
	if err := s.client.HSet(ctx, key,
		strconv.Itoa(part.Index), part.ETag,
		strconv.Itoa(part.Index)+"_hash", part.Hash,
		strconv.Itoa(part.Index)+"_size", strconv.FormatInt(part.Size, 10),
	).Err(); err != nil {
		return err
	}
	return s.Refresh(ctx, userID, fileHash)
}

func (s *RedisUploadSessionStore) Refresh(ctx context.Context, userID int, fileHash string) error {
	if !s.Available() {
		return ports.ErrUploadSessionUnavailable
	}
	if s.activeTTL <= 0 {
		s.activeTTL = 24 * time.Hour
	}
	if s.metadataTTL <= 0 {
		s.metadataTTL = 48 * time.Hour
	}

	key := uploadSessionKey(userID, fileHash)
	expiresAt := time.Now().Add(s.activeTTL)
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, "expiresAt", strconv.FormatInt(expiresAt.Unix(), 10))
	pipe.Expire(ctx, key, s.metadataTTL)
	pipe.ZAdd(ctx, uploadSessionSet, &redis.Z{Score: float64(expiresAt.Unix()), Member: key})
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisUploadSessionStore) Delete(ctx context.Context, userID int, fileHash string) error {
	if !s.Available() {
		return ports.ErrUploadSessionUnavailable
	}
	key := uploadSessionKey(userID, fileHash)
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.ZRem(ctx, uploadSessionSet, key)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisUploadSessionStore) AcquireMergeLock(ctx context.Context, userID int, fileHash string, ttl time.Duration) (bool, error) {
	if !s.Available() {
		return false, ports.ErrUploadSessionUnavailable
	}
	return s.client.SetNX(ctx, uploadSessionLockKey(userID, fileHash), "1", ttl).Result()
}

func (s *RedisUploadSessionStore) ReleaseMergeLock(ctx context.Context, userID int, fileHash string) error {
	if !s.Available() {
		return ports.ErrUploadSessionUnavailable
	}
	return s.client.Del(ctx, uploadSessionLockKey(userID, fileHash)).Err()
}

func (s *RedisUploadSessionStore) ListExpired(ctx context.Context, before time.Time, limit int64) ([]ports.UploadSessionRef, error) {
	if !s.Available() {
		return nil, ports.ErrUploadSessionUnavailable
	}
	if limit <= 0 {
		limit = 100
	}
	keys, err := s.client.ZRangeByScore(ctx, uploadSessionSet, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(before.Unix(), 10),
		Offset: 0,
		Count: limit,
	}).Result()
	if err != nil {
		return nil, err
	}

	refs := make([]ports.UploadSessionRef, 0, len(keys))
	for _, key := range keys {
		userID, fileHash, ok := parseUploadSessionKey(key)
		if !ok {
			continue
		}
		refs = append(refs, ports.UploadSessionRef{UserID: userID, FileHash: fileHash})
	}
	return refs, nil
}

func uploadSessionKey(userID int, fileHash string) string {
	return fmt.Sprintf("upload:%d:%s", userID, fileHash)
}

func uploadSessionLockKey(userID int, fileHash string) string {
	return uploadSessionKey(userID, fileHash) + ":lock"
}

func parseUploadSessionKey(key string) (int, string, bool) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) != 3 || parts[0] != "upload" || parts[2] == "" {
		return 0, "", false
	}
	userID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, "", false
	}
	return userID, parts[2], true
}

func decodeUploadSession(userID int, fileHash string, fields map[string]string) (*ports.UploadSession, error) {
	fileSize, err := strconv.ParseInt(fields["fileSize"], 10, 64)
	if err != nil || fileSize <= 0 {
		return nil, fmt.Errorf("invalid upload session file size")
	}
	chunkSize, err := strconv.ParseInt(fields["chunkSize"], 10, 64)
	if err != nil || chunkSize <= 0 {
		return nil, fmt.Errorf("invalid upload session chunk size")
	}
	totalChunks, err := strconv.Atoi(fields["totalChunks"])
	if err != nil || totalChunks <= 0 {
		return nil, fmt.Errorf("invalid upload session chunk count")
	}

	session := &ports.UploadSession{
		UserID: userID,
		FileHash: fileHash,
		UploadID: fields["id"],
		ObjectKey: fields["key"],
		FileName: fields["fileName"],
		ParentID: fields["parentId"],
		FileSize: fileSize,
		ChunkSize: chunkSize,
		TotalChunks: totalChunks,
		Parts: make(map[int]ports.UploadPartState),
	}
	if expiresAt, parseErr := strconv.ParseInt(fields["expiresAt"], 10, 64); parseErr == nil && expiresAt > 0 {
		session.ExpiresAt = time.Unix(expiresAt, 0)
	}

	for key, etag := range fields {
		idx, convErr := strconv.Atoi(key)
		if convErr != nil {
			continue
		}
		part := ports.UploadPartState{Index: idx, ETag: etag, Hash: fields[key+"_hash"]}
		if size, sizeErr := strconv.ParseInt(fields[key+"_size"], 10, 64); sizeErr == nil {
			part.Size = size
		}
		session.Parts[idx] = part
	}
	return session, nil
}

var _ ports.UploadSessionStore = (*RedisUploadSessionStore)(nil)
