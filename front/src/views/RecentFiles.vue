<template>
    <div class="recent-files">
        <!-- 页面头部 -->
        <div class="page-header">
            <div class="header-content">
                <div class="header-info">
                    <div class="header-icon">
                        <el-icon :size="28" color="#ffffff">
                            <Clock />
                        </el-icon>
                    </div>
                    <div class="header-text">
                        <h1 class="page-title">最近文件</h1>
                        <p class="page-description">查看您最近访问的文件</p>
                    </div>
                </div>
                <div class="header-actions">
                    <el-select v-model="timeRange" placeholder="时间范围" class="time-select">
                        <el-option label="今天" value="today"/>
                        <el-option label="本周" value="week"/>
                        <el-option label="本月" value="month"/>
                    </el-select>
                </div>
            </div>
        </div>

        <!-- 文件内容 -->
        <div class="file-content">
            <!-- 空状态 -->
            <div v-if="!filteredFiles || filteredFiles.length === 0" class="empty-state">
                <div class="empty-icon">
                    <el-icon :size="80" color="#c0c4cc">
                        <Clock />
                    </el-icon>
                </div>
                <h3>暂无最近文件</h3>
                <p>您在所选时间范围内没有访问任何文件</p>
            </div>

            <!-- 时间线视图 -->
            <div v-else class="timeline-container">
                <el-timeline class="file-timeline">
                    <el-timeline-item
                        v-for="(day, index) in filteredFiles"
                        :key="index"
                        :timestamp="day.date"
                        placement="top"
                        size="large"
                        type="primary"
                    >
                        <div class="day-files">
                            <!-- <div class="day-header">
                                <span class="file-count">{{ day.files.length }} 个文件</span>
                            </div> -->
                            
                            <div class="files-grid">
                                <div 
                                    class="file-card" 
                                    v-for="file in day.files" 
                                    :key="file.id"
                                >
                                    <!-- 文件图标 -->
                                    <div class="file-icon">
                                        <el-icon :size="32" :color="getIconColor(file)">
                                            <component :is="getIconComponent(file)"/>
                                        </el-icon>
                                    </div>
                                    
                                    <!-- 文件信息 -->
                                    <div class="file-info">
                                        <div class="file-name" :title="file.name">{{ file.name }}</div>
                                        <div class="file-meta">
                                            <span class="file-size">{{ file.size_str }}</span>
                                            <span class="file-time">{{ file.modified }}</span>
                                        </div>
                                    </div>
                                    
                                    <!-- 操作按钮 -->
                                    <div class="file-actions">
                                        <el-button size="small" type="primary" link @click="handleOpen(file)">
                                            <el-icon><View /></el-icon>
                                            打开
                                        </el-button>
                                        <el-button size="small" type="primary" link @click="handleLocate(file)">
                                            <el-icon><Location /></el-icon>
                                            定位
                                        </el-button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </el-timeline-item>
                </el-timeline>
            </div>
        </div>
    </div>
</template>

<script setup>
import {computed, ref, watch, onMounted} from 'vue'
import {Document, Folder, Clock, View, Location} from '@element-plus/icons-vue'
import {getRecentFiles} from "@/api/file";

// 时间范围
const timeRange = ref('week')
const allFiles = ref([])

// 图标选择逻辑
function getIconComponent(file) {
    if (file.type === 'folder') return Folder
    const ext = file.name.split('.').pop().toLowerCase()
    if (['doc', 'docx'].includes(ext)) return Document
    if (['xlsx', 'xls', 'csv'].includes(ext)) return Document
    if (['ppt', 'pptx'].includes(ext)) return Document
    if (['pdf'].includes(ext)) return Document
    return Document
}

function getIconColor(file) {
    if (file.type === 'folder') return '#FFB800'
    const ext = file.name.split('.').pop().toLowerCase()
    if (['doc', 'docx'].includes(ext)) return '#1E90FF'
    if (['xlsx', 'xls', 'csv'].includes(ext)) return '#27ae60'
    if (['ppt', 'pptx'].includes(ext)) return '#e67e22'
    if (['pdf'].includes(ext)) return '#e74c3c'
    return '#3a86ff'
}

// 拉取数据方法
async function fetchRecentFiles() {
    try {
        const res = await getRecentFiles(timeRange.value)
        allFiles.value = res || []
    } catch (err) {
        console.error('获取最近文件失败', err)
        allFiles.value = []
    }
}

// 监听时间范围变化
watch(timeRange, () => {
    fetchRecentFiles()
})

// 首次加载
onMounted(() => {
    fetchRecentFiles()
})

const filteredFiles = computed(() => allFiles.value) // 后端已经按 timeRange 过滤好

// 操作按钮方法
function handleOpen(file) {
    console.log('打开文件:', file)
    // TODO: 实现文件打开逻辑
}

function handleLocate(file) {
    console.log('定位文件:', file)
    // TODO: 实现文件定位逻辑
}
</script>

<style scoped>
.recent-files {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f8fafc;
}

/* 页面头部 */
.page-header {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
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

.header-actions {
  display: flex;
  align-items: center;
}

.time-select {
  width: 140px;
}

.time-select :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: white;
}

.time-select :deep(.el-input__inner) {
  color: white;
}

.time-select :deep(.el-select__caret) {
  color: white;
}

/* 文件内容 */
.file-content {
  flex: 1;
  background: white;
  overflow: auto;
}

/* 空状态 */
.empty-state {
  padding: 80px 24px;
  text-align: center;
  color: #909399;
}

.empty-icon {
  margin-bottom: 24px;
}

.empty-state h3 {
  font-size: 20px;
  color: #4a5568;
  margin: 0 0 12px 0;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
}

/* 时间线容器 */
.timeline-container {
  padding: 24px 24px 48px 24px;
}

.file-timeline {
  padding-left: 0;
}

.file-timeline :deep(.el-timeline-item__wrapper) {
  padding-left: 32px;
  margin-bottom: 24px;
}

.file-timeline :deep(.el-timeline-item:last-child .el-timeline-item__wrapper) {
  margin-bottom: 48px;
}

.file-timeline :deep(.el-timeline-item__tail) {
  border-left: 2px solid #e2e8f0;
}

.file-timeline :deep(.el-timeline-item__node) {
  background: #8b5cf6;
  border-color: #8b5cf6;
  width: 16px;
  height: 16px;
}

.file-timeline :deep(.el-timeline-item__timestamp) {
  font-size: 16px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 16px;
}

/* 每日文件 */
.day-files {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #e2e8f0;
}

.day-header {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f1f5f9;
}

.file-count {
  font-size: 12px;
  color: #6b7280;
  background: #f3f4f6;
  padding: 4px 8px;
  border-radius: 12px;
}

/* 文件网格 */
.files-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.file-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  transition: all 0.2s ease;
  cursor: pointer;
}

.file-card:hover {
  border-color: #8b5cf6;
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.15);
  transform: translateY(-1px);
}

.file-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  background: #f9fafb;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-info {
  flex: 1;
  min-width: 0;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #6b7280;
}

.file-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.file-actions .el-button {
  padding: 4px 8px;
  font-size: 12px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    padding: 20px 16px;
  }
  
  .header-content {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  
  .timeline-container {
    padding: 16px;
  }
  
  .day-files {
    padding: 16px;
  }
  
  .files-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  
  .file-card {
    padding: 10px;
  }
  
  .file-actions {
    flex-direction: column;
    gap: 4px;
  }
}
</style>
  