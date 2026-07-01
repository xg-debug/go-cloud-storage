package filetypes

import "strings"

// Category returns the standard category for a file extension.
// Extensions may or may not have a leading dot.
func Category(ext string) string {
	switch Normalize(ext) {
	case "jpg", "jpeg", "png", "gif", "bmp", "webp", "svg":
		return "image"
	case "mp4", "avi", "mov", "wmv", "flv", "webm", "mkv":
		return "video"
	case "mp3", "wav", "flac", "aac", "ogg", "m4a":
		return "audio"
	case "doc", "docx", "xls", "xlsx", "ppt", "pptx", "pdf", "txt", "md", "csv", "json", "xml":
		return "document"
	default:
		return "other"
	}
}

// ImageExtensions returns all image extensions.
func ImageExtensions() []string {
	return []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"}
}

// VideoExtensions returns all video extensions.
func VideoExtensions() []string {
	return []string{"mp4", "avi", "mov", "wmv", "flv", "webm", "mkv"}
}

// AudioExtensions returns all audio extensions.
func AudioExtensions() []string {
	return []string{"mp3", "wav", "flac", "aac", "ogg", "m4a"}
}

// DocumentExtensions returns all document/text extensions.
func DocumentExtensions() []string {
	return []string{"doc", "docx", "xls", "xlsx", "ppt", "pptx", "pdf", "txt", "md", "csv", "json", "xml"}
}

// Previewable returns whether the extension can be previewed inline and its preview type.
func Previewable(ext string) (bool, string) {
	e := Normalize(ext)
	if e == "" {
		return false, "other"
	}
	for _, v := range ImageExtensions() {
		if e == v {
			return true, "image"
		}
	}
	for _, v := range VideoExtensions() {
		if e == v {
			return true, "video"
		}
	}
	for _, v := range AudioExtensions() {
		if e == v {
			return true, "audio"
		}
	}
	if e == "md" {
		return true, "markdown"
	}
	textExts := []string{"txt", "json", "xml", "csv", "log", "js", "css", "html", "go", "java", "py", "c", "cpp"}
	for _, v := range textExts {
		if e == v {
			return true, "text"
		}
	}
	if e == "pdf" {
		return true, "pdf"
	}
	officeExts := []string{"doc", "docx", "xls", "xlsx", "ppt", "pptx"}
	for _, v := range officeExts {
		if e == v {
			return false, "office"
		}
	}
	return false, "other"
}

// ExtensionsForCategory returns all extensions belonging to a category for SQL queries.
func ExtensionsForCategory(category string) []string {
	switch category {
	case "image":
		return ImageExtensions()
	case "video":
		return VideoExtensions()
	case "audio":
		return AudioExtensions()
	case "document":
		return DocumentExtensions()
	default:
		return nil
	}
}

// Normalize trims whitespace and a leading dot, then lowercases.
func Normalize(ext string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
}
