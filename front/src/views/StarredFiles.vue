<template>
    <div class="starred-container">
      <!-- 顶部搜索栏 -->
      <div class="header">
        <h2>⭐ 我的收藏</h2>
        <el-input
          v-model="searchQuery"
          placeholder="搜索收藏内容"
          clearable
          class="search-box"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
  
      <el-divider />
  
      <!-- 收藏列表 -->
      <el-table
        :data="paginatedStarred"
        v-loading="loading"
        style="width: 100%"
        empty-text="暂无收藏内容"
        stripe
        border
      >
        <el-table-column label="名称" min-width="250">
          <template #default="{ row }">
            <div class="file-name-cell">
              <el-icon :color="row.type === 'folder' ? '#FFB800' : '#3a86ff'">
                <component :is="row.type === 'folder' ? Folder : Document" />
              </el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
  
        <el-table-column prop="path" label="所在路径" min-width="200" />
        <el-table-column prop="starredDate" label="收藏时间" width="160" />
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button type="primary" text size="small" @click="handleOpen(row)">
              打开
            </el-button>
            <el-button
              type="danger"
              text
              size="small"
              @click="handleUnstar(row)"
            >
              取消收藏
            </el-button>
          </template>
        </el-table-column>
      </el-table>
  
      <!-- 分页器 -->
      <div class="pagination">
        <el-pagination
          background
          layout="prev, pager, next"
          :total="filteredStarred.length"
          :page-size="pageSize"
          v-model:current-page="currentPage"
          hide-on-single-page
        />
      </div>
    </div>
  </template>
  
  <script setup>
  import { ref, computed, onMounted } from 'vue'
  import { Search, Folder, Document } from '@element-plus/icons-vue'
  
  // 模拟加载状态
  const loading = ref(false)
  
  // 搜索关键词
  const searchQuery = ref('')
  
  // 当前分页信息
  const currentPage = ref(1)
  const pageSize = 5
  
  // 假数据（后期可替换）
  const starredItems = ref([
    {
      id: 1,
      name: '产品设计文档.pdf',
      type: 'file',
      path: '/设计/产品文档',
      starredDate: '2025-08-05'
    },
    {
      id: 2,
      name: '需求分析文件夹',
      type: 'folder',
      path: '/项目资料',
      starredDate: '2025-08-03'
    },
    {
      id: 3,
      name: '总结报告.docx',
      type: 'file',
      path: '/工作总结',
      starredDate: '2025-07-29'
    },
    {
      id: 4,
      name: '用户调研',
      type: 'folder',
      path: '/调研',
      starredDate: '2025-07-25'
    },
    {
      id: 5,
      name: '系统架构图.png',
      type: 'file',
      path: '/技术资料',
      starredDate: '2025-07-20'
    },
    {
      id: 6,
      name: '开发日志',
      type: 'folder',
      path: '/开发',
      starredDate: '2025-07-18'
    }
  ])
  
  // 搜索过滤
  const filteredStarred = computed(() => {
    return starredItems.value.filter((item) =>
      item.name.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  })
  
  // 当前页数据
  const paginatedStarred = computed(() => {
    const start = (currentPage.value - 1) * pageSize
    return filteredStarred.value.slice(start, start + pageSize)
  })
  
  // 操作：打开文件夹或文件
  const handleOpen = (item) => {
    console.log('打开项目：', item)
    // 可跳转至对应目录或预览文件
  }
  
  // 操作：取消收藏
  const handleUnstar = (item) => {
    starredItems.value = starredItems.value.filter((i) => i.id !== item.id)
  }
  </script>
  
  <style scoped>
  .starred-container {
    padding: 24px;
    background-color: #fff;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }
  
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 12px;
  }
  
  .header h2 {
    font-size: 20px;
    margin: 0;
  }
  
  .search-box {
    width: 300px;
  }
  
  .file-name-cell {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .pagination {
    margin-top: 16px;
    display: flex;
    justify-content: center;
  }
  </style>
  