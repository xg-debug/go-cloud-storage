export function formatSize(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i >= sizes.length) return '> 1 PB'
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

export function formatTime(date) {
  if (!date) return ''
  const now = Date.now()
  const t = new Date(date).getTime()
  const diff = now - t
  const min = Math.floor(diff / 6e4)
  if (min < 1) return '刚刚'
  if (min < 60) return min + '分钟前'
  const hrs = Math.floor(min / 60)
  if (hrs < 24) return hrs + '小时前'
  return new Date(date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
