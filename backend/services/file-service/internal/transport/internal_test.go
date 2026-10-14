package transport

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"go-cloud-storage/backend/pkg/contracts"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

func TestFileRPCPreservesVersionAndObjectWithoutExposingPublicKey(t *testing.T) {
	file := domain.File{Id: "file", UserId: 7, ShareVersion: 3, OssObjectKey: "files/7/key", ParentId: sql.NullString{String: "root", Valid: true}}
	data, err := json.Marshal(fileRecord(file))
	if err != nil {
		t.Fatal(err)
	}
	var remote contracts.FileMetadata
	if err = json.Unmarshal(data, &remote); err != nil {
		t.Fatal(err)
	}
	if remote.Id != file.Id || remote.UserId != 7 || remote.ShareVersion != 3 || remote.OssObjectKey != file.OssObjectKey || remote.ParentId.String != "root" || !remote.ParentId.Valid {
		t.Fatalf("wire fields lost: %+v", remote)
	}
	public, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(public), file.OssObjectKey) {
		t.Fatal("internal object key leaked in public file JSON")
	}
}
