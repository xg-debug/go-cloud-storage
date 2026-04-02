package models

// FileTypeStat 文件类型统计信息
type FileTypeStat struct {
	Type  string `json:"type"`  // 文件类型标识 (image, video, audio, document, other)
	Name  string `json:"name"`  // 文件类型中文名称 (图片, 视频, 音频, 文档, 其他)
	Count int    `json:"count"` // 该类型文件数量
	//Size       int64   `json:"size"`       // 该类型文件总大小(字节)
	//SizeGB     float64 `json:"size_gb"`    // 该类型文件总大小(GB)
	Percentage float64 `json:"percentage"` // 占总文件数量的百分比
}

// UserDashboardStatsResponse 用户仪表板统计信息响应
type UserDashboardStatsResponse struct {
	StorageQuota  StorageQuotaInfo `json:"storage_quota"`   // 存储配额信息
	FileStats     FileStatsInfo    `json:"file_stats"`      // 文件统计信息
	FileTypeStats []FileTypeStat   `json:"file_type_stats"` // 文件类型统计
}

// StorageQuotaInfo 存储配额信息
type StorageQuotaInfo struct {
	Total       int64   `json:"total"`        // 总配额(字节)
	Used        int64   `json:"used"`         // 已使用(字节)
	UsedPercent float64 `json:"used_percent"` // 使用百分比
	TotalGB     float64 `json:"total_gb"`     // 总配额(GB)
	UsedGB      float64 `json:"used_gb"`      // 已使用(GB)
}

// FileStatsInfo 文件统计信息
type FileStatsInfo struct {
	TotalFiles  int64 `json:"total_files"`  // 文件总数
	Folders     int64 `json:"folders"`      // 文件夹数量
	SharedFiles int64 `json:"shared_files"` // 共享文件数量
}
