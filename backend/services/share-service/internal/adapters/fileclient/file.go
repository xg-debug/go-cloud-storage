package fileclient

import (
	"context"

	"go-cloud-storage/backend/pkg/contracts"
	"go-cloud-storage/backend/pkg/rpc"
)

type Files struct{ Client *rpc.Client }

func (f *Files) GetUserFileByID(user int, id string) (*contracts.FileMetadata, error) {
	var out contracts.FileMetadata
	err := f.Client.Call(context.Background(), "files/get", contracts.FileRequest{UserID: user, ID: id}, &out)
	return &out, err
}
func (f *Files) GetFileByIds(ids []string) ([]contracts.FileMetadata, error) {
	var out []contracts.FileMetadata
	err := f.Client.Call(context.Background(), "files/batch", contracts.FileRequest{IDs: ids}, &out)
	return out, err
}
