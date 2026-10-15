package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	fileports "go-cloud-storage/backend/services/file-service/internal/ports"
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

func (s *RedisUploadSessionStore) Get(ctx context.Context, userID int, fileHash string) (*fileports.UploadSession, error) {
	if !s.Available() {
		return nil, fileports.ErrUploadSessionUnavailable
	}
	fields, err := s.client.HGetAll(ctx, uploadSessionKey(userID, fileHash)).Result()
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fileports.ErrUploadSessionNotFound
	}

	session, err := decodeUploadSession(userID, fileHash, fields)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *RedisUploadSessionStore) Save(ctx context.Context, session *fileports.UploadSession) error {
	if !s.Available() {
		return fileports.ErrUploadSessionUnavailable
	}
	if session == nil {
		return fmt.Errorf("upload session is nil")
	}
	if session.Parts == nil {
		session.Parts = make(map[int]fileports.UploadPartState)
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

func (s *RedisUploadSessionStore) SavePart(ctx context.Context, userID int, fileHash string, part fileports.UploadPartState) error {
	if !s.Available() {
		return fileports.ErrUploadSessionUnavailable
	}
	key := uploadSessionKey(userID, fileHash)
	active, metadata := s.activeTTL, s.metadataTTL
	if active <= 0 {
		active = 24 * time.Hour
	}
	if metadata <= 0 {
		metadata = 48 * time.Hour
	}
	expires := time.Now().Add(active).Unix()
	n, err := s.client.Eval(ctx, `if redis.call('HGET',KEYS[1],'id') ~= ARGV[1] then return 0 end
 redis.call('HSET',KEYS[1],ARGV[2],ARGV[3],ARGV[2]..'_hash',ARGV[4],ARGV[2]..'_size',ARGV[5],'expiresAt',ARGV[6])
 redis.call('PEXPIRE',KEYS[1],ARGV[7]);redis.call('ZADD',KEYS[2],ARGV[6],KEYS[1]);return 1`, []string{key, uploadSessionSet}, part.UploadID, part.Index, part.ETag, part.Hash, part.Size, expires, metadata.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return fileports.ErrUploadSessionNotFound
	}
	return nil
}

func (s *RedisUploadSessionStore) Refresh(ctx context.Context, userID int, fileHash string) error {
	if !s.Available() {
		return fileports.ErrUploadSessionUnavailable
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

func (s *RedisUploadSessionStore) Delete(ctx context.Context, userID int, fileHash string, uploadID string) error {
	if !s.Available() {
		return fileports.ErrUploadSessionUnavailable
	}
	key := uploadSessionKey(userID, fileHash)
	return s.client.Eval(ctx, `local id=redis.call('HGET',KEYS[1],'id')
 if id and id ~= ARGV[1] then return 0 end
 redis.call('DEL',KEYS[1]);redis.call('ZREM',KEYS[2],KEYS[1]);return 1`, []string{key, uploadSessionSet}, uploadID).Err()
}

func (s *RedisUploadSessionStore) AcquireMergeLock(ctx context.Context, userID int, fileHash string, owner string, ttl time.Duration) (bool, error) {
	if !s.Available() {
		return false, fileports.ErrUploadSessionUnavailable
	}
	return s.client.SetNX(ctx, uploadSessionLockKey(userID, fileHash), owner, ttl).Result()
}

func (s *RedisUploadSessionStore) ReleaseMergeLock(ctx context.Context, userID int, fileHash string, owner string) error {
	if !s.Available() {
		return fileports.ErrUploadSessionUnavailable
	}
	return s.client.Eval(ctx, `if redis.call('GET',KEYS[1]) == ARGV[1] then return redis.call('DEL',KEYS[1]) end return 0`, []string{uploadSessionLockKey(userID, fileHash)}, owner).Err()
}

func (s *RedisUploadSessionStore) ListExpired(ctx context.Context, before time.Time, limit int64) ([]fileports.UploadSessionRef, error) {
	if !s.Available() {
		return nil, fileports.ErrUploadSessionUnavailable
	}
	if limit <= 0 {
		limit = 100
	}
	keys, err := s.client.ZRangeByScore(ctx, uploadSessionSet, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(before.Unix(), 10),
		Offset: 0,
		Count:  limit,
	}).Result()
	if err != nil {
		return nil, err
	}

	refs := make([]fileports.UploadSessionRef, 0, len(keys))
	for _, key := range keys {
		userID, fileHash, ok := parseUploadSessionKey(key)
		if !ok {
			continue
		}
		refs = append(refs, fileports.UploadSessionRef{UserID: userID, FileHash: fileHash})
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

func decodeUploadSession(userID int, fileHash string, fields map[string]string) (*fileports.UploadSession, error) {
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

	session := &fileports.UploadSession{
		UserID:      userID,
		FileHash:    fileHash,
		UploadID:    fields["id"],
		ObjectKey:   fields["key"],
		FileName:    fields["fileName"],
		ParentID:    fields["parentId"],
		FileSize:    fileSize,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		Parts:       make(map[int]fileports.UploadPartState),
	}
	if expiresAt, parseErr := strconv.ParseInt(fields["expiresAt"], 10, 64); parseErr == nil && expiresAt > 0 {
		session.ExpiresAt = time.Unix(expiresAt, 0)
	}

	for key, etag := range fields {
		idx, convErr := strconv.Atoi(key)
		if convErr != nil {
			continue
		}
		part := fileports.UploadPartState{Index: idx, ETag: etag, Hash: fields[key+"_hash"]}
		if size, sizeErr := strconv.ParseInt(fields[key+"_size"], 10, 64); sizeErr == nil {
			part.Size = size
		}
		session.Parts[idx] = part
	}
	return session, nil
}

var _ fileports.UploadSessionStore = (*RedisUploadSessionStore)(nil)
