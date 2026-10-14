// Package storagehttp implements the internal, versioned storage service protocol.
// User authorization stays with the calling application; this endpoint must only
// be accessible to trusted services, over a private network or TLS.
package storagehttp

import (
	"go-cloud-storage/backend/internal/ports"
	"time"
)

const Prefix = "/internal/storage/v1/"

type Request struct {
	Key         string                `json:"key"`
	Keys        []string              `json:"keys,omitempty"`
	Destination string                `json:"destination,omitempty"`
	UploadID    string                `json:"uploadId,omitempty"`
	Parts       []ports.CompletedPart `json:"parts,omitempty"`
	FileName    string                `json:"fileName,omitempty"`
	StoredURL   string                `json:"storedUrl,omitempty"`
	Expiry      time.Duration         `json:"expiry,omitempty"`
}

type Response struct {
	Value  string              `json:"value,omitempty"`
	Size   int64               `json:"size,omitempty"`
	Object *ports.StoredObject `json:"object,omitempty"`
	Part   *ports.UploadedPart `json:"part,omitempty"`
	Hash   string              `json:"hash,omitempty"`
}
