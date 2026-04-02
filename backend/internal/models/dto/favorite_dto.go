package dto

type FavoriteDTO struct {
	FileId    string `json:"file_id"`
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	Path      string `json:"path"` // 计算出来的完整路径
	SizeStr   string `json:"size_str"`
	FileURL   string `json:"file_url"`
	CreatedAt string `json:"created_at"` // 收藏时间
}
