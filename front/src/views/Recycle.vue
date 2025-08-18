<template>
    <div class="recycle-bin">
        <!-- 页面头部 -->
        <div class="page-header">
            <div class="header-content">
                <div class="header-info">
                    <div class="header-icon">
                        <el-icon :size="28" color="#ffffff">
                            <Delete />
                        </el-icon>
                    </div>
                    <div class="header-text">
                        <h1 class="page-title">回收站</h1>
                        <p class="page-description">已删除的文件将在此保留10天</p>
                    </div>
                </div>
                <div class="header-stats">
                    <div class="stat-item">
                        <span class="stat-number">{{ trashItems.length }}</span>
                        <span class="stat-label">文件数量</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-number">{{ selectedItems.length }}</span>
                        <span class="stat-label">已选择</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- 工具栏 -->
        <div class="toolbar">
            <div class="toolbar-left">
                <span class="selection-info" v-if="selectedItems.length > 0">
                    已选择 {{ selectedItems.length }} 个文件
                </span>
            </div>
            <div class="toolbar-right">
                <el-button 
                    :icon="Refresh" 
                    :disabled="selectedItems.length === 0"
                    @click="handleRestore"
                    type="success"
                >
                    还原选中
                </el-button>
                <el-button 
                    :icon="Delete" 
                    :disabled="selectedItems.length === 0"
                    @click="handleBatchDelete"
                    type="danger"
                >
                    彻底删除
                </el-button>
                <el-button 
                    :icon="Delete" 
                    @click="openClearDialog"
                    type="danger"
                    plain
                >
                    清空回收站
                </el-button>
            </div>
        </div>

        <!-- 文件内容 -->
        <div class="file-content">
            <!-- 空状态 -->
            <div v-if="trashItems.length === 0" class="empty-state">
                <div class="empty-icon">
                    <el-icon :size="80" color="#c0c4cc">
                        <Delete />
                    </el-icon>
                </div>
                <h3>回收站为空</h3>
                <p>删除的文件会出现在这里，并保留10天</p>
            </div>

            <!-- 文件表格 -->
            <div v-else class="table-container">
                <el-table
                    :data="trashItems"
                    class="recycle-table"
                    @selection-change="handleSelectionChange"
                    :row-key="row => row.fileId"
                    empty-text="回收站空空如也"
                >
                    <el-table-column type="selection" width="55" />

                    <el-table-column width="60">
                        <template #default="{ row }">
                            <el-icon :size="20" :color="row.is_dir === true ? '#FFB800' : '#3a86ff'">
                                <component :is="row.is_dir === true ? Folder : Document"/>
                            </el-icon>
                        </template>
                    </el-table-column>

                    <el-table-column label="名称" min-width="200" show-overflow-tooltip>
                        <template #default="{ row }">
                            <span class="file-name" @click="handleOpen(row)" :title="row.name">
                                {{ row.name }}
                            </span>
                        </template>
                    </el-table-column>

                    <el-table-column label="大小" width="120" align="center">
                        <template #default="{ row }">
                            <span class="file-size">{{ row.size_str }}</span>
                        </template>
                    </el-table-column>

                    <el-table-column label="删除时间" width="180" align="center">
                        <template #default="{ row }">
                            <span class="delete-time">{{ row.deletedDate }}</span>
                        </template>
                    </el-table-column>

                    <el-table-column label="剩余天数" width="120" align="center">
                        <template #default="{ row }">
                            <el-tag 
                                :type="row.expireDays <= 3 ? 'danger' : row.expireDays <= 7 ? 'warning' : 'success'"
                                size="small"
                            >
                                {{ row.expireDays }}天
                            </el-tag>
                        </template>
                    </el-table-column>

                    <el-table-column label="操作" fixed="right" width="180" align="center">
                        <template #default="{ row }">
                            <el-button
                                size="small"
                                type="success"
                                link
                                @click="handleRestoreOne(row)"
                                :icon="Refresh"
                            >
                                还原
                            </el-button>
                            <el-button
                                size="small"
                                type="danger"
                                link
                                @click="openDeleteDialog(row)"
                                :icon="Delete"
                            >
                                彻底删除
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </div>

        <!-- 清空回收站弹窗 -->
        <el-dialog
            v-model="clearDialogVisible"
            title="清空回收站"
            width="400px"
            :before-close="handleClearDialogClose"
        >
            <div class="delete-confirm-text">
                <div>确认清空回收站？</div>
            </div>
            <template #footer>
                <el-button @click="clearDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmClear" :loading="deleting">确定</el-button>
            </template>
        </el-dialog>

        <!-- 彻底删除弹窗 -->
        <el-dialog
            v-model="deleteDialogVisible"
            title="彻底删除"
            width="400px"
            :before-close="handleDeleteDialogClose"
        >
            <div class="delete-confirm-text">
                <div>文件删除后将无法恢复，您确认要彻底删除所选文件吗？</div>
            </div>
            <template #footer>
                <el-button @click="deleteDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Delete, Document, Folder, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
    loadSoftDeletedFiles,
    deletePermanent,
    deleteSelected,
    clearRecycleBin,
    restore,
    restoreBatch,
} from '@/api/recycle'

const deleteDialogVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)

const clearDialogVisible = ref(false)

const selectedItems = ref([])
const trashItems = ref([])

// 页面初始化加载回收站数据
const fetchTrashItems = async () => {
    try {
        const res = await loadSoftDeletedFiles()
        trashItems.value = res.data || []
    } catch (error) {
        ElMessage.error('加载回收站数据失败')
    }
}

onMounted(() => {
    fetchTrashItems()
})

// 多选变化
const handleSelectionChange = (selection) => {
    selectedItems.value = selection
}

// 打开文件/文件夹
const handleOpen = (item) => {
    ElMessage.info(`打开文件: ${item.name}`)
}

const openClearDialog = () => {
    clearDialogVisible.value = true
}

// 清空回收站
const confirmClear = async () => {
    try {
        await clearRecycleBin()
        trashItems.value = []
        selectedItems.value = []
        ElMessage.success('回收站已清空')
        clearDialogVisible.value = false
    } catch (error) {
        if (error !== 'cancel') {
            ElMessage.error('清空失败')
        }
    }
}

const handleClearDialogClose = () => {
    clearDialogVisible.value = false
}

// 批量删除
const handleBatchDelete = async () => {
    if (selectedItems.value.length === 0) return
    try {
        const deleteIds = selectedItems.value.map((item) => item.fileId)
        await deleteSelected(deleteIds)
        trashItems.value = trashItems.value.filter((item) => !deleteIds.includes(item.fileId))
        selectedItems.value = []
        ElMessage.success('选中文件已彻底删除')
    } catch (error) {
        ElMessage.error('删除失败')
    }
}

// 批量还原
const handleRestore = async () => {
    if (selectedItems.value.length === 0) return
    try {
        const restoredIds = selectedItems.value.map((item) => item.fileId)
        await restoreBatch(restoredIds)
        trashItems.value = trashItems.value.filter((item) => !restoredIds.includes(item.fileId))
        selectedItems.value = []
        ElMessage.success('选中文件已还原')
    } catch (error) {
        ElMessage.error('还原失败')
    }
}

// 单个还原
const handleRestoreOne = async (row) => {
    try {
        await restore(row.fileId)
        trashItems.value = trashItems.value.filter((item) => item.fileId !== row.fileId)
        ElMessage.success(`已还原 ${row.name}`)
    } catch (error) {
        ElMessage.error('还原失败')
    }
}

// 彻底删除 - 打开确认弹窗
const openDeleteDialog = (row) => {
    deleteTarget.value = row
    deleteDialogVisible.value = true
}

// 确认彻底删除
const confirmDelete = async () => {
    deleting.value = true
    try {
        await deletePermanent(deleteTarget.value.fileId)
        ElMessage.success('删除成功')
        deleteDialogVisible.value = false
        fetchTrashItems()
    } catch (error) {
        ElMessage.error('删除失败')
    } finally {
        deleting.value = false
        deleteTarget.value = {}
    }
}

// 关闭弹窗
const handleDeleteDialogClose = () => {
    deleteDialogVisible.value = false
    deleteTarget.value = {}
    deleting.value = false
}
</script>

<style scoped>
.recycle-bin {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f8fafc;
}

/* 页面头部 */
.page-header {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
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
}

.selection-info {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
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

/* 表格容器 */
.table-container {
  padding: 24px;
}

.recycle-table {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.recycle-table :deep(.el-table__header) {
  background: #f8fafc;
}

.recycle-table :deep(.el-table__row:hover) {
  background: #fef2f2;
}

.file-name {
  font-weight: 500;
  color: #1f2937;
  cursor: pointer;
  transition: color 0.2s;
}

.file-name:hover {
  color: #ef4444;
  text-decoration: underline;
}

.file-size {
  font-size: 13px;
  color: #6b7280;
}

.delete-time {
  font-size: 13px;
  color: #6b7280;
}

/* 对话框样式 */
.delete-confirm-text {
  text-align: center;
  font-size: 14px;
  line-height: 1.8;
  user-select: none;
}

.delete-confirm-text strong {
  color: #ef4444;
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
  
  .header-stats {
    gap: 24px;
  }
  
  .toolbar {
    padding: 16px;
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
  }
  
  .toolbar-left,
  .toolbar-right {
    justify-content: center;
  }
  
  .table-container {
    padding: 16px;
  }
  
  .recycle-table :deep(.el-table__cell) {
    padding: 8px 4px;
  }
}
</style>
