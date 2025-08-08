<template>
    <div class="recent-container">
      <div class="header">
        <h2>最近文件</h2>
        <el-select v-model="timeRange" placeholder="时间范围" size="small" style="width: 120px">
          <el-option label="今天" value="today" />
          <el-option label="本周" value="week" />
          <el-option label="本月" value="month" />
        </el-select>
      </div>
  
      <el-divider />
  
      <el-empty v-if="filteredFiles.length === 0" description="暂无最近文件" />
  
      <el-timeline v-else>
        <el-timeline-item
          v-for="(day, index) in filteredFiles"
          :key="index"
          :timestamp="day.date"
          placement="top"
        >
          <el-table :data="day.files" style="width: 100%" :fit="true" border>
            <el-table-column prop="name" label="文件名" min-width="200">
              <template #default="{ row }">
                <div class="file-name-cell">
                  <el-icon :color="getIconColor(row)">
                    <component :is="getIconComponent(row)" />
                  </el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
  
            <el-table-column prop="path" label="位置" min-width="140" />
            <el-table-column prop="modified" label="修改时间" min-width="140" />
            <el-table-column label="大小" min-width="100">
              <template #default="{ row }">
                {{ formatSize(row.size) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="120">
              <template #default="{ row }">
                <el-button size="small" type="text" @click="handleOpen(row)">打开</el-button>
                <el-button size="small" type="text" @click="handleLocate(row)">定位</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-timeline-item>
      </el-timeline>
    </div>
  </template>
  
  <script setup>
  import { ref, computed, watch } from 'vue'
  import { Folder, Document } from '@element-plus/icons-vue'
  
  // 时间范围
  const timeRange = ref('week')
  
  // 模拟数据（你可以替换成后端返回的数据）
  const allFiles = ref([
    {
      date: '2023-05-15 今天',
      range: 'today',
      files: [
        {
          name: '项目进度报告.docx',
          type: 'file',
          path: '/工作资料/项目A',
          modified: '10:30',
          size: 1258291 // 1.2 MB
        },
        {
          name: '用户调研数据.xlsx',
          type: 'file',
          path: '/工作资料/项目B',
          modified: '09:15',
          size: 3670016 // 3.5 MB
        }
      ]
    },
    {
      date: '2023-05-14 昨天',
      range: 'week',
      files: [
        {
          name: '产品规划.pptx',
          type: 'file',
          path: '/产品/规划',
          modified: '16:45',
          size: 2097152
        }
      ]
    },
    {
      date: '2023-05-01 本月早期',
      range: 'month',
      files: [
        {
          name: '总结.pdf',
          type: 'file',
          path: '/汇报材料',
          modified: '08:00',
          size: 1048576
        }
      ]
    }
  ])
  
  // 过滤逻辑
  const filteredFiles = computed(() =>
    allFiles.value.filter(item => {
      if (timeRange.value === 'today') return item.range === 'today'
      if (timeRange.value === 'week') return item.range === 'today' || item.range === 'week'
      return true
    })
  )
  
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
  
  // 大小格式化
  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B'
    const kb = bytes / 1024
    if (kb < 1024) return kb.toFixed(1) + ' KB'
    const mb = kb / 1024
    return mb.toFixed(1) + ' MB'
  }
  
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
  .recent-container {
    padding: 20px;
    background: #fff;
    border-radius: 12px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  }
  
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  
  .header h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 600;
    color: #333;
  }
  
  .file-name-cell {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .el-icon {
    font-size: 18px;
  }
  
  .el-timeline {
    padding-left: 20px;
  }
  
  .el-table {
    margin-top: 16px;
    font-size: 14px;
    border-radius: 8px;
    overflow: hidden;
  }
  
  /* 按钮样式美化 */
  :deep(.el-table__cell .cell) {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  
  .el-button {
    padding: 0 6px;
    font-size: 13px;
  }
  </style>
  