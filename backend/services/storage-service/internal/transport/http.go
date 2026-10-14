package transport

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-cloud-storage/backend/infrastructure/storagehttp"
	"go-cloud-storage/backend/internal/ports"
)

// NewHandler exposes only object operations. It has no access to MySQL, Redis,
// user sessions or file metadata. Callers must authorize a file before using it.
func NewHandler(storage ports.Storage, token string, maxBody int64) (http.Handler, error) {
	if storage == nil || len(token) < 32 || maxBody <= 0 {
		return nil, errors.New("storage, a 32-byte token and a positive body limit are required")
	}
	secret := sha256.Sum256([]byte("Bearer " + token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		supplied := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		if subtle.ConstantTimeCompare(secret[:], supplied[:]) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(r.URL.Path, storagehttp.Prefix) {
			http.NotFound(w, r)
			return
		}
		operation := strings.TrimPrefix(r.URL.Path, storagehttp.Prefix)
		if operation == "object" && r.Method == http.MethodGet {
			download(storage, w, r)
			return
		}
		if (operation == "object" || operation == "part") && r.Method == http.MethodPut {
			upload(storage, maxBody, operation, w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
		var input storagehttp.Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if !validRequest(operation, input) {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		var out storagehttp.Response
		var err error
		switch operation {
		case "url":
			out.Value = storage.GenerateObjectURL(input.Key)
		case "stat":
			out.Size, err = storage.GetObjectInfo(ctx, input.Key)
		case "hash":
			out.Value, err = storage.ComputeObjectSHA256(ctx, input.Key)
		case "multipart/init":
			out.Value, err = storage.InitiateMultipartUpload(ctx, input.Key)
		case "multipart/complete":
			out.Object, err = storage.CompleteMultipartUpload(ctx, input.Key, input.UploadID, input.Parts)
		case "multipart/abort":
			err = storage.AbortMultipartUpload(ctx, input.Key, input.UploadID)
		case "presign/preview":
			out.Value, err = storage.PresignedGetPreviewURL(ctx, input.Key, input.Expiry)
		case "presign/download":
			out.Value, err = storage.PresignedDownloadURL(ctx, input.Key, input.FileName, input.Expiry)
		case "presign/stored":
			out.Value = storage.PresignStoredObjectURL(ctx, input.StoredURL, input.Expiry)
		case "delete":
			err = storage.DeleteFile(ctx, input.Key)
		case "delete-with-thumbnail":
			err = storage.DeleteObjectWithThumbnail(ctx, input.Key)
		case "delete-thumbnail":
			err = storage.DeleteThumbnailForObject(ctx, input.Key)
		case "delete-many":
			err = storage.DeleteFiles(ctx, input.Keys)
		case "copy":
			err = storage.CopyObject(ctx, input.Key, input.Destination)
		case "thumbnail":
			out.Value, err = storage.GenerateThumbnailForObject(ctx, input.Key)
		default:
			http.NotFound(w, r)
			return
		}
		respond(w, out, err)
	}), nil
}

func validKey(key string) bool {
	if key == "" || len(key) > 1024 || strings.HasPrefix(key, "/") || strings.ContainsAny(key, "\x00\r\n\\") {
		return false
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == ".." || segment == "." {
			return false
		}
	}
	return true
}
func validRequest(op string, in storagehttp.Request) bool {
	if strings.HasPrefix(op, "presign/") && (in.Expiry < time.Second || in.Expiry > 7*24*time.Hour) {
		return false
	}
	if op == "presign/stored" {
		return len(in.StoredURL) <= 8192
	}
	if op == "delete-many" {
		if len(in.Keys) > 10000 {
			return false
		}
		for _, key := range in.Keys {
			if !validKey(key) {
				return false
			}
		}
		return true
	}
	if !validKey(in.Key) {
		return false
	}
	if op == "copy" && !validKey(in.Destination) {
		return false
	}
	if (op == "multipart/complete" || op == "multipart/abort") && in.UploadID == "" {
		return false
	}
	if op == "multipart/complete" {
		if len(in.Parts) == 0 || len(in.Parts) > 10000 {
			return false
		}
		previous := 0
		for _, part := range in.Parts {
			if part.PartNumber <= previous || part.PartNumber > 10000 || part.ETag == "" {
				return false
			}
			previous = part.PartNumber
		}
	}
	return true
}
func respond(w http.ResponseWriter, output storagehttp.Response, err error) {
	if err != nil {
		slog.Error("storage operation failed", "error", err)
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "storage operation failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(output)
}
func upload(storage ports.Storage, limit int64, operation string, w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key := q.Get("key")
	if !validKey(key) {
		http.Error(w, "invalid key", http.StatusBadRequest)
		return
	}
	if r.ContentLength > limit {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()
	var out storagehttp.Response
	var err error
	if operation == "part" {
		part, e := strconv.Atoi(q.Get("part"))
		if e != nil || part < 1 || part > 10000 || q.Get("uploadId") == "" || r.ContentLength < 0 {
			http.Error(w, "invalid part", http.StatusBadRequest)
			return
		}
		out.Part, out.Hash, err = storage.UploadPart(r.Context(), key, q.Get("uploadId"), part, r.Body, r.ContentLength, q.Get("hash"))
	} else {
		out.Size, err = storage.PutObjectStream(r.Context(), key, r.Body, r.ContentLength, r.Header.Get("Content-Type"))
		if err == nil {
			objectURL := storage.GenerateObjectURL(key)
			out.Object = &ports.StoredObject{ObjectKey: key, Size: out.Size, URL: objectURL, ThumbnailURL: objectURL}
		}
	}
	respond(w, out, err)
}
func download(storage ports.Storage, w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key := q.Get("key")
	if !validKey(key) {
		http.Error(w, "invalid key", http.StatusBadRequest)
		return
	}
	var reader io.ReadCloser
	var err error
	if q.Has("start") || q.Has("end") {
		start, e1 := strconv.ParseInt(q.Get("start"), 10, 64)
		end, e2 := strconv.ParseInt(q.Get("end"), 10, 64)
		if e1 != nil || e2 != nil || start < 0 || end < start {
			http.Error(w, "invalid range", http.StatusBadRequest)
			return
		}
		reader, err = storage.DownloadFileRange(r.Context(), key, start, end)
	} else {
		reader, err = storage.DownloadFile(r.Context(), key)
	}
	if err != nil {
		respond(w, storagehttp.Response{}, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err = io.Copy(w, reader); err != nil {
		panic(http.ErrAbortHandler)
	}
}
