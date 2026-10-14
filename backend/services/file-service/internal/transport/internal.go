package transport

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	repository "go-cloud-storage/backend/services/file-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

func RegisterFileInternal(r *gin.Engine, token string, db *gorm.DB) {
	repo := repository.NewFileRepository(db)
	r.POST(rpc.Prefix+"files/:action", rpc.Protect(token), func(c *gin.Context) {
		var in contracts.FileRequest
		if !rpc.Bind(c, &in) {
			return
		}
		switch c.Param("action") {
		case "get":
			f, err := repo.GetUserFileByID(in.UserID, in.ID)
			if err != nil || f == nil || f.IsDeleted {
				c.Status(404)
				return
			}
			c.JSON(200, fileRecord(*f))
		case "batch":
			if len(in.IDs) > 1000 {
				c.Status(400)
				return
			}
			files, err := repo.GetFileByIds(in.IDs)
			if err != nil {
				c.Status(503)
				return
			}
			out := make([]contracts.FileMetadata, 0, len(files))
			for _, f := range files {
				if !f.IsDeleted {
					out = append(out, fileRecord(f))
				}
			}
			c.JSON(200, out)
		default:
			c.Status(404)
		}
	})
}

func fileRecord(f domain.File) contracts.FileMetadata {
	return contracts.FileMetadata{ShareVersion: f.ShareVersion, Id: f.Id, UserId: f.UserId, Name: f.Name, Size: f.Size, SizeStr: f.SizeStr, IsDir: f.IsDir, FileExtension: f.FileExtension, FileURL: f.FileURL, ThumbnailURL: f.ThumbnailURL, OssObjectKey: f.OssObjectKey, FileHash: f.FileHash, ParentId: contracts.NullableString{String: f.ParentId.String, Valid: f.ParentId.Valid}, IsDeleted: f.IsDeleted, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt}
}
