package storagekey

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func New(userID int, parentID, fileName string) string {
	prefix := fmt.Sprintf("files/%d", userID)
	if parentID != "" {
		prefix += "/" + parentID
	}
	return prefix + "/" + uuid.NewString() + filepath.Ext(fileName)
}

func Thumbnail(key string) string {
	return strings.TrimSuffix(key, filepath.Ext(key)) + "_thumb.jpg"
}
