package dto

type FavoriteDTO struct {
	Id        string `json:"id"`
	FileId    string `json:"file_id"`
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	Path      string `json:"path"` // 计算出来的完整路径
	Size      int64  `json:"size"`
	SizeStr   string `json:"size_str"`
	Extension string `json:"extension"`
	FileURL   string `json:"file_url"`
	CreatedAt string `json:"created_at"` // 收藏时间
}
