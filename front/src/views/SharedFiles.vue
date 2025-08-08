<template>
    <div class="shared-container">
      <div class="header">
        <h2>共享文件</h2>
        <el-tabs v-model="activeTab" class="tabs">
          <el-tab-pane label="我共享的" name="sharedByMe" />
          <el-tab-pane label="共享给我的" name="sharedWithMe" />
        </el-tabs>
      </div>
  
      <el-divider />
  
      <div v-if="activeTab === 'sharedByMe'">
        <el-table
          :data="sharedByMe"
          style="width: 100%"
          stripe
          border
          :row-key="row => row.id"
          empty-text="暂无我共享的文件"
        >
          <el-table-column label="文件名" min-width="280">
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
  
          <el-table-column label="共享给" min-width="220">
            <template #default="{ row }">
              <div class="shared-users">
                <el-tooltip
                  v-for="user in row.sharedWith"
                  :key="user.id"
                  :content="user.name"
                  placement="top"
                >
                  <el-avatar
                    size="small"
                    :style="{ backgroundColor: stringToColor(user.name) }"
                    class="user-avatar"
                  >
                    {{ user.name.slice(-2) }}
                  </el-avatar>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
  
          <el-table-column prop="sharedDate" label="共享时间" width="160" />
          <el-table-column prop="permission" label="权限" width="120" />
          <el-table-column label="操作" fixed="right" width="180">
            <template #default="{ row }">
              <el-button
                size="small"
                type="primary"
                text
                @click="handleManageShare(row)"
                title="管理共享"
              >
                管理
              </el-button>
              <el-button
                size="small"
                type="danger"
                text
                @click="handleCancelShare(row)"
                title="取消共享"
              >
                取消共享
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
  
      <div v-else>
        <el-table
          :data="sharedWithMe"
          style="width: 100%"
          stripe
          border
          :row-key="row => row.id"
          empty-text="暂无共享给我的文件"
        >
          <el-table-column label="文件名" min-width="280">
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
  
          <el-table-column prop="sharedBy" label="共享者" width="140" />
          <el-table-column prop="sharedDate" label="共享时间" width="160" />
          <el-table-column prop="permission" label="权限" width="120" />
          <el-table-column label="操作" fixed="right" width="180">
            <template #default="{ row }">
              <el-button
                size="small"
                type="primary"
                text
                @click="handleRequestEdit(row)"
                v-if="row.permission === '只读'"
                title="申请编辑权限"
              >
                申请编辑
              </el-button>
              <el-button
                size="small"
                type="danger"
                text
                @click="handleLeaveShare(row)"
                title="取消共享"
              >
                取消共享
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </template>
  
  <script setup>
  import { ref } from 'vue'
  import { Folder, Document } from '@element-plus/icons-vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  
  const activeTab = ref('sharedByMe')
  
  const sharedByMe = ref([
    {
      id: 1,
      name: '项目需求文档.pdf',
      type: 'file',
      sharedWith: [
        { id: 2, name: '李四' },
        { id: 3, name: '王五' },
        { id: 4, name: '赵六' }
      ],
      sharedDate: '2025-08-01',
      permission: '可编辑'
    },
    {
      id: 2,
      name: '设计资源',
      type: 'folder',
      sharedWith: [{ id: 5, name: '钱七' }],
      sharedDate: '2025-07-25',
      permission: '只读'
    }
  ])
  
  const sharedWithMe = ref([
    {
      id: 11,
      name: '公司报表.xlsx',
      type: 'file',
      sharedBy: '张三',
      sharedDate: '2025-08-02',
      permission: '只读'
    },
    {
      id: 12,
      name: '项目资料',
      type: 'folder',
      sharedBy: '李四',
      sharedDate: '2025-07-30',
      permission: '可编辑'
    }
  ])
  
  // 点击打开文件或文件夹
  const handleOpen = (item) => {
    ElMessage.info(`打开文件：${item.name}`)
    // 这里可接入路由跳转或文件预览功能
  }
  
  // 管理共享 — 弹窗示例
  const handleManageShare = (item) => {
    ElMessageBox.alert(
      `管理共享功能待开发，当前文件：${item.name}`,
      '提示',
      { confirmButtonText: '知道了' }
    )
  }
  
  // 取消共享 — 确认弹窗
  const handleCancelShare = (item) => {
    ElMessageBox.confirm(
      `确定要取消共享《${item.name}》吗？`,
      '操作确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
      .then(() => {
        sharedByMe.value = sharedByMe.value.filter((f) => f.id !== item.id)
        ElMessage.success('已取消共享')
      })
      .catch(() => {
        ElMessage.info('已取消操作')
      })
  }
  
  // 共享给我的申请编辑权限
  const handleRequestEdit = (item) => {
    ElMessageBox.prompt(
      `向共享者申请编辑权限，文件：${item.name}`,
      '申请编辑权限',
      {
        confirmButtonText: '发送申请',
        cancelButtonText: '取消',
        inputPlaceholder: '请输入申请理由',
        inputValidator: (val) => val && val.trim() !== '',
        inputErrorMessage: '申请理由不能为空'
      }
    )
      .then(({ value }) => {
        // 模拟申请提交
        ElMessage.success(`申请已发送：${value}`)
      })
      .catch(() => {
        ElMessage.info('申请已取消')
      })
  }
  
  // 共享给我的文件取消共享/退出共享
  const handleLeaveShare = (item) => {
    ElMessageBox.confirm(
      `确定取消共享，退出《${item.name}》的访问吗？`,
      '操作确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
      .then(() => {
        sharedWithMe.value = sharedWithMe.value.filter((f) => f.id !== item.id)
        ElMessage.success('已取消访问')
      })
      .catch(() => {
        ElMessage.info('已取消操作')
      })
  }
  
  // 颜色生成工具，字符串转颜色（用于头像背景）
  const stringToColor = (str) => {
    let hash = 0
    for (let i = 0; i < str.length; i++) {
      hash = str.charCodeAt(i) + ((hash << 5) - hash)
    }
    let color = '#'
    for (let i = 0; i < 3; i++) {
      const value = (hash >> (i * 8)) & 0xff
      color += ('00' + value.toString(16)).slice(-2)
    }
    return color
  }
  </script>
  
  <style scoped>
  .shared-container {
    padding: 24px;
    background-color: #fff;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgb(0 0 0 / 0.05);
    height: 100%;
    overflow: auto;
  }
  
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 12px;
    margin-bottom: 16px;
  }
  
  .header h2 {
    font-size: 24px;
    font-weight: 700;
    margin: 0;
  }
  
  .tabs {
    flex: 1;
    max-width: 400px;
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
    color: #409eff;
    transition: color 0.3s;
  }
  .file-name:hover {
    color: #1869c7;
    text-decoration: underline;
  }
  
  .shared-users {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
  
  .user-avatar {
    font-size: 12px;
    color: #fff;
    font-weight: 600;
    user-select: none;
  }
  
  .el-table th,
  .el-table td {
    vertical-align: middle !important;
  }
  
  /* 操作按钮文字更紧凑 */
  .el-button--text {
    padding: 2px 6px;
    font-weight: 600;
  }
  
  /* 取消共享按钮颜色 */
  .el-button--danger {
    color: #f56c6c;
  }
  
  @media (max-width: 768px) {
    .header {
      flex-direction: column;
      align-items: flex-start;
    }
  
    .tabs {
      max-width: 100%;
      width: 100%;
    }
  }
  </style>
  