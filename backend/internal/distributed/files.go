package distributed

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/infrastructure/eventing"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/internal/repositories"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type FileRecord struct {
	models.File
	ObjectKey string `json:"objectKey"`
}
type FileRequest struct {
	UserID int      `json:"userId"`
	ID     string   `json:"id"`
	IDs    []string `json:"ids"`
}
type Files struct{ Client *Client }

func (f *Files) GetUserFileByID(user int, id string) (*models.File, error) {
	var out FileRecord
	err := f.Client.Call(context.Background(), "files/get", FileRequest{UserID: user, ID: id}, &out)
	out.File.OssObjectKey = out.ObjectKey
	return &out.File, err
}
func (f *Files) GetFileByIds(ids []string) ([]models.File, error) {
	var out []FileRecord
	if err := f.Client.Call(context.Background(), "files/batch", FileRequest{IDs: ids}, &out); err != nil {
		return nil, err
	}
	files := make([]models.File, 0, len(out))
	for _, f := range out {
		f.File.OssObjectKey = f.ObjectKey
		files = append(files, f.File)
	}
	return files, nil
}
func RegisterFileInternal(r *gin.Engine, token string, db *gorm.DB) {
	repo := repositories.NewFileRepository(db)
	r.POST(Prefix+"files/:action", Protect(token), func(c *gin.Context) {
		var in FileRequest
		if !Bind(c, &in) {
			return
		}
		switch c.Param("action") {
		case "get":
			f, err := repo.GetUserFileByID(in.UserID, in.ID)
			if err != nil || f == nil || f.IsDeleted {
				c.Status(404)
				return
			}
			c.JSON(200, FileRecord{File: *f, ObjectKey: f.OssObjectKey})
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
			out := make([]FileRecord, 0, len(files))
			for _, f := range files {
				if !f.IsDeleted {
					out = append(out, FileRecord{File: f, ObjectKey: f.OssObjectKey})
				}
			}
			c.JSON(200, out)
		default:
			c.Status(404)
		}
	})
}

// EnsureOwner provisions only file-owned data and is safe to repeat after crashes.
func EnsureOwner(db *gorm.DB, quotaGB int) func(context.Context, Identity) error {
	return func(ctx context.Context, id Identity) error {
		if id.UserID <= 0 || id.RootID == "" {
			return errors.New("invalid owner")
		}
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			q := models.StorageQuota{UserID: id.UserID, Total: int64(quotaGB) * 1024 * 1024 * 1024, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&q).Error; err != nil {
				return err
			}
			root := models.File{Id: id.RootID, UserId: id.UserID, Name: "/", IsDir: true, ParentId: sql.NullString{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&root).Error
		})
	}
}

type Shares struct{ Client *Client }

func (s *Shares) CountSharedFiles(user int) (int64, error) {
	var out struct {
		Count int64 `json:"count"`
	}
	err := s.Client.Call(context.Background(), "shares/count", FileRequest{UserID: user}, &out)
	return out.Count, err
}
func (s *Shares) DeleteBatch(tx *gorm.DB, ids []string) error {
	if tx == nil {
		return errors.New("share invalidation requires file transaction")
	}
	var files []models.File
	if err := tx.Where("id IN ?", ids).Find(&files).Error; err != nil {
		return err
	}
	versions := map[string]int64{}
	for _, file := range files {
		versions[file.Id] = file.ShareVersion
	}
	return eventing.Enqueue(tx, ports.Event{Type: "file.deleted.v1", Data: versions})
}

type EventShares struct {
	repositories.ShareRepository
	DB *gorm.DB
}

func NewEventShares(db *gorm.DB) *EventShares {
	return &EventShares{ShareRepository: repositories.NewShareRepository(db), DB: db}
}
func (s *EventShares) CreateShareAndEvent(share *models.Share, name string) (*models.Share, error) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(share).Error; err != nil {
			return err
		}
		return eventing.Enqueue(tx, ports.Event{Type: "file.shared.v1", UserID: share.UserId, AggregateID: share.FileId, Data: map[string]string{"fileName": name}})
	})
	return share, err
}
func RegisterShareInternal(r *gin.Engine, token string, repo repositories.ShareRepository) {
	r.POST(Prefix+"shares/count", Protect(token), func(c *gin.Context) {
		var in FileRequest
		if !Bind(c, &in) {
			return
		}
		count, err := repo.CountSharedFiles(in.UserID)
		if err != nil {
			c.Status(503)
			return
		}
		c.JSON(200, gin.H{"count": count})
	})
}
