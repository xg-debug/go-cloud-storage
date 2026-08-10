import { ElMessage } from 'element-plus'
import { downloadFile as downloadApi, getDownloadInfo, previewFile as previewApi } from '@/api/file'
import { addFavorite, cancelFavorite } from '@/api/favorite'
import { formatSize, formatTime } from '@/utils/format'

export function useFileActions() {
  async function download(item) {
    try {
      const info = await getDownloadInfo(item.id)
      if (info?.directDownloadUrl) {
        triggerDownload(info.directDownloadUrl, info.fileName || item.name)
        return
      }

      if (item.size && item.size > 10 * 1024 * 1024) {
        ElMessage.error('下载链接生成失败，请稍后重试')
        return
      }

      const blob = await downloadApi(item.id)
      const url = URL.createObjectURL(blob)
      triggerDownload(url, item.name)
      URL.revokeObjectURL(url)
    } catch {
      ElMessage.error('下载失败')
    }
  }

  async function preview(item) {
    try {
      const res = await previewApi(item.id)
      if (res?.file_url) {
        window.open(res.file_url, '_blank')
      } else {
        ElMessage.info('该文件不支持在线预览')
      }
    } catch {
      ElMessage.info('暂不支持预览')
    }
  }

  async function toggleStar(fileId, isStarred) {
    try {
      if (isStarred) {
        await cancelFavorite(fileId)
        ElMessage.success('已取消收藏')
      } else {
        await addFavorite(fileId)
        ElMessage.success('已收藏')
      }
    } catch {
      ElMessage.error('操作失败')
    }
  }

  function getFileType(extension) {
    if (!extension) return 'other'
    const ext = extension.toLowerCase()
    if (['jpg','jpeg','png','gif','bmp','webp','svg'].includes(ext)) return 'image'
    if (['mp4','avi','mov','wmv','flv','mkv','webm'].includes(ext)) return 'video'
    if (['mp3','wav','flac','aac','ogg'].includes(ext)) return 'audio'
    if (['pdf','doc','docx','xls','xlsx','ppt','pptx','txt'].includes(ext)) return 'document'
    return 'other'
  }

  function isImage(filename) {
    return /\.(jpg|jpeg|png|gif|bmp|webp|svg)$/i.test(filename)
  }

  return { download, preview, toggleStar, getFileType, isImage, formatSize, formatTime }
}

function triggerDownload(url, fileName) {
  const a = document.createElement('a')
  a.href = url
  if (fileName) a.download = fileName
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}
