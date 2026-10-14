package shareclient

import (
	"context"
	"errors"

	"gorm.io/gorm"

	contracts "go-cloud-storage/backend/pkg/contracts"
	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	ports "go-cloud-storage/backend/pkg/ports"
	rpc "go-cloud-storage/backend/pkg/rpc"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

type Shares struct{ Client *rpc.Client }

func (s *Shares) CountSharedFiles(user int) (int64, error) {
	var out struct {
		Count int64 `json:"count"`
	}
	err := s.Client.Call(context.Background(), "shares/count", contracts.FileRequest{UserID: user}, &out)
	return out.Count, err
}
func (s *Shares) DeleteBatch(tx *gorm.DB, ids []string) error {
	if tx == nil {
		return errors.New("share invalidation requires file transaction")
	}
	var files []domain.File
	if err := tx.Where("id IN ?", ids).Find(&files).Error; err != nil {
		return err
	}
	versions := map[string]int64{}
	for _, file := range files {
		versions[file.Id] = file.ShareVersion
	}
	return eventing.Enqueue(tx, ports.Event{Type: "file.deleted.v1", Data: versions})
}
