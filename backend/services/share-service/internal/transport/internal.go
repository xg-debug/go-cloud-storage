package transport

import (
	"github.com/gin-gonic/gin"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	repository "go-cloud-storage/backend/services/share-service/internal/adapters/repository"
)

func RegisterShareInternal(r *gin.Engine, token string, repo repository.ShareRepository) {
	r.POST(rpc.Prefix+"shares/count", rpc.Protect(token), func(c *gin.Context) {
		var in contracts.FileRequest
		if !rpc.Bind(c, &in) {
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
