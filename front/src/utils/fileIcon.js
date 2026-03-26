import {
  Document,
  Files,
  Folder,
  Headset,
  Picture,
  VideoCamera,
} from '@element-plus/icons-vue'

function extOf(fileName = '') {
  const ext = fileName.split('.').pop()
  return (ext || '').toLowerCase()
}

export function getFileIcon(fileName = '', isDir = false) {
  if (isDir) return Folder
  const ext = extOf(fileName)
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext)) return Picture
  if (['mp4', 'avi', 'mov', 'wmv', 'mkv', 'webm'].includes(ext)) return VideoCamera
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a'].includes(ext)) return Headset
  if (['pdf', 'doc', 'docx', 'txt', 'xls', 'xlsx', 'ppt', 'pptx', 'md'].includes(ext)) return Document
  return Files
}

export function getFileIconColor(fileName = '', isDir = false) {
  if (isDir) return '#FFB800'
  const ext = extOf(fileName)
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext)) return '#f59e0b'
  if (['mp4', 'avi', 'mov', 'wmv', 'mkv', 'webm'].includes(ext)) return '#ef4444'
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a'].includes(ext)) return '#8b5cf6'
  if (['pdf', 'doc', 'docx', 'txt', 'xls', 'xlsx', 'ppt', 'pptx', 'md'].includes(ext)) return '#06b6d4'
  return '#6b7280'
}
