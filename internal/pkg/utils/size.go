package utils

import "fmt"

// FormatFileSize 将字节数转换为可读大小，如 B, KB, MB, GB
func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// BytesToKB 将字节数转换为 KB（浮点数）
func BytesToKB(size uint64) float64 {
	return float64(size) / 1024
}

// BytesToMB 将字节数转换为 MB（浮点数）
func BytesToMB(size uint64) float64 {
	return float64(size) / (1024 * 1024)
}

// BytesToGB 将字节数转换为 GB（浮点数）
func BytesToGB(size uint64) float64 {
	return float64(size) / (1024 * 1024 * 1024)
}

// KBToBytes KB 转字节
func KBToBytes(kb float64) uint64 {
	return uint64(kb * 1024)
}

// MBToBytes MB 转字节
func MBToBytes(mb float64) uint64 {
	return uint64(mb * 1024 * 1024)
}

// GBToBytes GB 转字节
func GBToBytes(gb float64) uint64 {
	return uint64(gb * 1024 * 1024 * 1024)
}

// GetFileExtension 获取文件扩展名 (带点)
func GetFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
