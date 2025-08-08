<template>
    <div class="trash-container">
      <div class="header">
        <h2>回收站</h2>
        <div class="actions">
          <el-button
            type="danger"
            :disabled="selectedItems.length === 0"
            @click="handleEmptyTrash"
            plain
            round
          >
            <el-icon><Delete /></el-icon>
            清空回收站
          </el-button>
          <el-button
            :disabled="selectedItems.length === 0"
            @click="handleRestore"
            plain
            round
          >
            <el-icon><Refresh /></el-icon>
            还原
          </el-button>
        </div>
      </div>
  
      <el-divider />
  
      <el-table
        :data="trashItems"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        stripe
        border
        :row-key="row => row.id"
        empty-text="回收站空空如也"
      >
        <el-table-column type="selection" width="55" />
  
        <el-table-column label="名称" min-width="280">
          <template #default="{ row }">
            <div class="file-name-cell" title="点击打开">
              <el-icon
                :color="row.type === 'folder' ? '#FFB800' : '#3a86ff'"
                class="file-icon"
              >
                <component :is="row.type === 'folder' ? Folder : Document" />
              </el-icon>
              <span class="file-name" @click="handleOpen(row)">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
  
        <el-table-column
          prop="originalPath"
          label="原位置"
          min-width="200"
          show-overflow-tooltip
        />
  
        <el-table-column prop="deletedDate" label="删除时间" width="160" />
        <el-table-column prop="size" label="大小" width="120" />
  
        <el-table-column label="操作" fixed="right" width="140">
          <template #default="{ row }">
            <el-button
              size="small"
              type="danger"
              text
              @click="handleDeletePermanently(row)"
              title="彻底删除"
            >
              彻底删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </template>
  
  <script setup>
  import { ref } from 'vue'
  import { Delete, Refresh, Folder, Document } from '@element-plus/icons-vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  
  const selectedItems = ref([])
  const trashItems = ref([
    {
      id: 1,
      name: '旧版设计方案',
      type: 'folder',
      originalPath: '/设计资源/项目A',
      deletedDate: '2025-08-03',
      size: '-'
    },
    {
      id: 2,
      name: '测试数据.xlsx',
      type: 'file',
      originalPath: '/工作资料',
      deletedDate: '2025-08-01',
      size: '8.3 MB'
    }
    // 更多回收站数据...
  ])
  
  const handleSelectionChange = (selection) => {
    selectedItems.value = selection
  }
  
  // 模拟打开文件/文件夹
  const handleOpen = (item) => {
    ElMessage.info(`打开文件: ${item.name}`)
    // 这里可以添加跳转或预览逻辑
  }
  
  // 确认清空回收站
  const handleEmptyTrash = () => {
    ElMessageBox.confirm(
      '确定要清空回收站吗？此操作不可恢复！',
      '警告',
      {
        confirmButtonText: '清空',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
      .then(() => {
        trashItems.value = []
        selectedItems.value = []
        ElMessage.success('回收站已清空')
      })
      .catch(() => {
        ElMessage.info('已取消操作')
      })
  }
  
  // 批量还原选中项
  const handleRestore = () => {
    if (selectedItems.value.length === 0) return
  
    ElMessageBox.confirm(
      `确定还原选中的 ${selectedItems.value.length} 个文件吗？`,
      '确认还原',
      {
        confirmButtonText: '还原',
        cancelButtonText: '取消',
        type: 'info'
      }
    )
      .then(() => {
        const restoredIds = selectedItems.value.map((item) => item.id)
        trashItems.value = trashItems.value.filter((item) => !restoredIds.includes(item.id))
        selectedItems.value = []
        ElMessage.success('选中文件已还原')
      })
      .catch(() => {
        ElMessage.info('已取消操作')
      })
  }
  
  // 彻底删除单个文件
  const handleDeletePermanently = (item) => {
    ElMessageBox.confirm(
      `确定要彻底删除《${item.name}》吗？此操作不可恢复！`,
      '彻底删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'danger'
      }
    )
      .then(() => {
        trashItems.value = trashItems.value.filter((i) => i.id !== item.id)
        selectedItems.value = selectedItems.value.filter((i) => i.id !== item.id)
        ElMessage.success('文件已被彻底删除')
      })
      .catch(() => {
        ElMessage.info('已取消操作')
      })
  }
  </script>
  
  <style scoped>
  .trash-container {
    padding: 24px;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 8px 20px rgb(0 0 0 / 0.06);
    height: 100%;
    overflow-y: auto;
  }
  
  .header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
    gap: 12px;
  }
  
  .header h2 {
    font-size: 26px;
    font-weight: 700;
    color: #2c3e50;
    margin: 0;
  }
  
  .actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }
  
  .file-name-cell {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    user-select: none;
  }
  
  .file-icon {
    font-size: 20px;
  }
  
  .file-name {
    font-weight: 600;
    color: #409eff;
    transition: color 0.25s ease;
  }
  
  .file-name:hover {
    color: #1f63d6;
    text-decoration: underline;
  }
  
  .el-table th,
  .el-table td {
    vertical-align: middle !important;
  }
  
  .el-button--text {
    padding: 4px 8px;
    font-weight: 600;
  }
  
  .el-button--danger {
    color: #f56c6c;
  }
  
  @media (max-width: 768px) {
    .header {
      flex-direction: column;
      align-items: flex-start;
    }
  
    .actions {
      width: 100%;
      justify-content: flex-start;
    }
  }
  </style>
  