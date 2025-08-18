<template>
  <div class="starred-files">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-info">
          <div class="header-icon">
            <el-icon :size="28" color="#f59e0b">
              <Star />
            </el-icon>
          </div>
          <div class="header-text">
            <h1 class="page-title">⭐ 我的收藏</h1>
            <p class="page-description">管理您收藏的重要文件</p>
          </div>
        </div>
        <div class="header-stats">
          <div class="stat-item">
            <span class="stat-number">{{ totalCount }}</span>
            <span class="stat-label">收藏文件</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-input
          placeholder="搜索收藏文件..."
          :prefix-icon="Search"
          clearable
          style="width: 300px;"
        />
      </div>
      <div class="toolbar-right">
        <el-button :icon="Refresh" @click="fetchFavorites">
          刷新
        </el-button>
      </div>
    </div>

    <!-- 收藏文件内容 -->
    <div class="starred-content">
      <!-- 收藏列表 -->
      <el-table
        :data="starredItems"
        v-loading="loading"
        style="width: 100%"
        empty-text="暂无收藏内容"
        class="starred-table"
      >
        <el-table-column label="名称" min-width="300">
          <template #default="{ row }">
            <div class="file-name-cell">
              <div class="file-icon">
                <el-icon :size="20" :color="row.is_dir ? '#FFB800' : getFileIconColor(row.name)">
                  <component :is="row.is_dir ? Folder : getFileIcon(row.name)" />
                </el-icon>
              </div>
              <span class="file-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="path" label="所在目录" min-width="200" show-overflow-tooltip />
        
        <el-table-column prop="size_str" label="大小" width="120" />
        
        <el-table-column prop="created_at" label="收藏时间" width="180" />
        
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="previewFile(row)">
              <el-icon><View /></el-icon>
              预览
            </el-button>
            <el-button size="small" type="primary" link @click="downloadFile(row)" :disabled="row.is_dir">
              <el-icon><Download /></el-icon>
              下载
            </el-button>
            <el-button size="small" type="danger" link @click="unfavorite(row)">
              <el-icon><Delete /></el-icon>
              取消收藏
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页器 -->
      <div class="pagination" v-if="totalCount > pageSize">
        <el-pagination
          background
          layout="prev, pager, next, total"
          :total="totalCount"
          :page-size="pageSize"
          v-model:current-page="currentPage"
          @current-change="onPageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Document, 
  Folder, 
  Search, 
  Star,
  StarFilled,
  View,
  Download,
  Location,
  Refresh,
  Picture,
  VideoCamera,
  Headset,
  Files
} from '@element-plus/icons-vue'
import { getFavorites, cancelFavorite } from '@/api/favorite'
import { ElMessage } from 'element-plus'

const router = useRouter()

// 加载状态
const loading = ref(false)

// 收藏列表
const starredItems = ref([])
const totalCount = ref(0)

// 分页参数
const currentPage = ref(1)
const pageSize = 10

// 获取收藏列表 - 保持原有函数
const fetchFavorites = async () => {
  loading.value = true
  try {
    const res = await getFavorites({ page: currentPage.value, pageSize })
    starredItems.value = res.favoriteList
    totalCount.value = res.total
  } catch (err) {
    console.error('获取收藏列表失败', err)
  } finally {
    loading.value = false
  }
}

// 页码变化回调 - 保持原有函数
const onPageChange = (page) => {
  currentPage.value = page
  fetchFavorites()
}

// 操作方法 - 保持原有函数
const openFile = (row) => {
  if (row.is_dir) {
    // 跳转到目录页面
    router.push({ name: 'FileList', query: { parentId: row.id } })
  } else {
    // 文件预览，可以在新窗口打开或者使用内置预览组件
    window.open(row.file_url, '_blank')
  }
}

const downloadFile = (row) => {
  if (!row.is_dir && row.fileURL) {
    const a = document.createElement('a')
    a.href = row.fileURL
    a.download = row.name
    a.click()
  }
}

const unfavorite = async (row) => {
  try {
    await cancelFavorite(row.id)
    starredItems.value = starredItems.value.filter(i => i.id !== row.id)
    totalCount.value = totalCount.value - 1
  } catch (err) {
    console.error('取消收藏失败', err)
  }
}

const locateFile = (row) => {
  // 跳转到所在目录，并传 fileId 用于高亮
  router.push({
    name: 'FileList',
    query: { parentId: row.parentId, highlightFileId: row.id }
  })
}

// 预览文件功能
const previewFile = (row) => {
  if (row.is_dir) {
    // 文件夹直接打开
    openFile(row)
  } else {
    // 文件预览 - 需要后端提供预览接口
    const fileName = row.name.toLowerCase()
    const ext = fileName.split('.').pop()
    
    // 可预览的文件类型
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'mp4', 'avi', 'mov', 'mp3', 'wav', 'pdf', 'txt', 'doc', 'docx'].includes(ext)) {
      // 构建预览URL - 需要后端提供 /api/files/preview/:id 接口
      const previewUrl = `/api/files/preview/${row.id}`
      window.open(previewUrl, '_blank')
    } else {
      // 不支持预览的文件类型提示用户
      ElMessage.info('此文件类型不支持预览，请下载后查看')
    }
  }
}

// 获取文件图标
const getFileIcon = (fileName) => {
  const ext = fileName.split('.').pop()?.toLowerCase()
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext)) {
    return Picture
  } else if (['mp4', 'avi', 'mov', 'wmv', 'mkv'].includes(ext)) {
    return VideoCamera
  } else if (['mp3', 'wav', 'flac', 'aac', 'ogg'].includes(ext)) {
    return Headset
  } else if (['pdf', 'doc', 'docx', 'txt', 'xls', 'xlsx', 'ppt', 'pptx'].includes(ext)) {
    return Document
  }
  return Files
}

// 获取文件图标颜色
const getFileIconColor = (fileName) => {
  const ext = fileName.split('.').pop()?.toLowerCase()
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext)) {
    return '#f59e0b'
  } else if (['mp4', 'avi', 'mov', 'wmv', 'mkv'].includes(ext)) {
    return '#ef4444'
  } else if (['mp3', 'wav', 'flac', 'aac', 'ogg'].includes(ext)) {
    return '#8b5cf6'
  } else if (['pdf', 'doc', 'docx', 'txt', 'xls', 'xlsx', 'ppt', 'pptx'].includes(ext)) {
    return '#06b6d4'
  }
  return '#6b7280'
}

// 页面加载时获取 - 保持原有逻辑
onMounted(fetchFavorites)
</script>

<style scoped>
.starred-files {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f8fafc;
}

/* 页面头部 */
.page-header {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
  padding: 14px 24px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-icon {
  width: 48px;
  height: 48px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 4px 0;
}

.page-description {
  font-size: 14px;
  opacity: 0.9;
  margin: 0;
}

.header-stats {
  display: flex;
  gap: 32px;
}

.stat-item {
  text-align: center;
}

.stat-number {
  display: block;
  font-size: 24px;
  font-weight: 600;
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  opacity: 0.8;
}

/* 工具栏 */
.toolbar {
  background: white;
  padding: 20px 24px;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* 收藏内容 */
.starred-content {
  flex: 1;
  background: white;
  padding: 24px;
  overflow: auto;
}

/* 表格样式 */
.starred-table {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.starred-table :deep(.el-table__header) {
  background: #f8fafc;
}

.starred-table :deep(.el-table__row:hover) {
  background: #f0f9ff;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-icon {
  flex-shrink: 0;
}

.file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
  color: #1f2937;
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.action-buttons .el-button {
  margin: 0;
}

/* 分页器 */
.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: center;
  padding: 20px 0;
  border-top: 1px solid #e5e7eb;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    padding: 20px 16px;
  }
  
  .header-content {
    flex-direction: column;
    gap: 20px;
    text-align: center;
  }
  
  .toolbar {
    padding: 16px;
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
  }
  
  .starred-content {
    padding: 16px;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 4px;
  }
  
  .action-buttons .el-button {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>