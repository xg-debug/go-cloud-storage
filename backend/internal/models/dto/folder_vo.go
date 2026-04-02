package dto

import "time"

type FolderVO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentId  string    `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}
