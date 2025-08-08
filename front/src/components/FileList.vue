<template>
  <div class="files-container">
    <div class="files-toolbar">
      <el-button type="primary" @click="showUploadDialog">
        <el-icon><i-ep-Upload /></el-icon>
        上传文件
      </el-button>

      <div class="view-options">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button label="grid">
            <el-icon><i-ep-Grid /></el-icon>
          </el-radio-button>
          <el-radio-button label="list">
            <el-icon><i-ep-Menu /></el-icon>
          </el-radio-button>
        </el-radio-group>
      </div>

      <el-select v-model="sortBy" size="small" style="width: 120px">
        <el-option label="按名称" value="name" />
        <el-option label="按日期" value="date" />
        <el-option label="按大小" value="size" />
      </el-select>
    </div>

    <!-- 上传对话框 -->
    <el-dialog v-model="uploadVisible" title="上传文件" width="600px">
      <file-uploader @close="uploadVisible = false" />
    </el-dialog>

    <!-- 列表视图 -->
    <el-table v-if="viewMode === 'list'" :data="filteredFiles" style="width: 100%" row-class-name="file-row"
      @row-contextmenu="handleContextMenu">
      <el-table-column width="40">
        <template #default="{ row }">
          <el-checkbox v-model="row.selected" />
        </template>
      </el-table-column>
      <el-table-column label="文件名" min-width="200">
        <template #default="{ row }">
          <div class="file-name-cell">
            <el-icon v-if="row.isFolder" color="var(--el-color-warning)">
              <i-ep-Folder />
            </el-icon>
            <el-icon v-else>
              <component :is="`i-ep-${getFileIcon(row.type)}`" />
            </el-icon>
            <span class="name">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="modified" label="修改日期" width="180" />
      <el-table-column prop="size" label="大小" width="120">
        <template #default="{ row }">
          {{ formatSize(row.size) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button size="small" @click="handleDownload(row)">
            下载
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 网格视图 -->
    <div v-else class="file-grid">
      <el-card v-for="file in filteredFiles" :key="file.id" shadow="hover" class="file-card" @click="handleSelect(file)"
        @contextmenu.prevent="handleContextMenu($event, file)">
        <div class="card-content">
          <el-icon v-if="file.isFolder" size="48" color="var(--el-color-warning)">
            <i-ep-Folder />
          </el-icon>
          <el-icon v-else size="48">
            <component :is="`i-ep-${getFileIcon(file.type)}`" />
          </el-icon>
          <div class="file-name">{{ file.name }}</div>
          <div class="file-meta">
            <span>{{ formatSize(file.size) }}</span>
            <span>{{ formatDate(file.modified) }}</span>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 右键菜单 -->
    <el-dropdown ref="contextMenu" trigger="contextmenu" v-model:visible="contextMenuVisible"
      :style="{ left: `${contextMenuX}px`, top: `${contextMenuY}px` }">
      <el-dropdown-menu>
        <el-dropdown-item @click="handleFileAction('open')">
          <el-icon><i-ep-FolderOpened /></el-icon>
          打开
        </el-dropdown-item>
        <el-dropdown-item @click="handleFileAction('download')">
          <el-icon><i-ep-Download /></el-icon>
          下载
        </el-dropdown-item>
        <el-dropdown-item @click="handleFileAction('rename')">
          <el-icon><i-ep-Edit /></el-icon>
          重命名
        </el-dropdown-item>
        <el-dropdown-item @click="handleFileAction('share')">
          <el-icon><i-ep-Share /></el-icon>
          分享
        </el-dropdown-item>
        <el-dropdown-divider />
        <el-dropdown-item @click="handleFileAction('delete')" style="color: var(--el-color-error)">
          <el-icon><i-ep-Delete /></el-icon>
          删除
        </el-dropdown-item>
      </el-dropdown-menu>
    </el-dropdown>
  </div>
</template>
  
<script setup>
import { ref, computed } from 'vue'
import { useFileStore } from '@/stores/files'
import FileUploader from '@/components/files/FileUploader.vue'

const fileStore = useFileStore()

const viewMode = ref('grid')
const sortBy = ref('name')
const uploadVisible = ref(false)
const contextMenuVisible = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const selectedFile = ref(null)

  // 其他逻辑与之前类似...
</script>
  
<style scoped>
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
  padding: 10px 0;
}

.file-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.file-card:hover {
  transform: translateY(-3px);
}

.card-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 10px;
}

.file-name {
  text-align: center;
  word-break: break-all;
  font-size: 14px;
}

.file-meta {
  display: flex;
  justify-content: space-between;
  width: 100%;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-name-cell .name {
  margin-left: 5px;
}

.files-toolbar {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 20px;
}

.view-options {
  margin-left: auto;
}
</style>