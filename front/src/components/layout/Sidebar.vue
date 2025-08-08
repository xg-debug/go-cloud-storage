<template>
  <div class="sidebar-container">
    <!-- 存储空间指示器 -->
    <div class="storage-info">
      <div class="storage-header">
        <h3>存储空间</h3>
        <el-button type="primary" size="small" class="upgrade-btn" @click="handleUpgrade">
          升级
        </el-button>
      </div>
      
      <div class="storage-progress">
        <el-progress 
          type="dashboard" 
          :percentage="storagePercentage" 
          :color="storageColor" 
          :width="80"
        >
          <template #default>
            <div class="progress-content">
              <div class="percentage">{{ storagePercentage }}%</div>
              <div class="size">{{ usedStorage }}GB</div>
            </div>
          </template>
        </el-progress>
      </div>

      <div class="storage-details">
        <div class="storage-item">
          <span class="label">已使用</span>
          <span class="value">{{ usedStorage }}GB</span>
        </div>
        <div class="storage-item">
          <span class="label">总容量</span>
          <span class="value">{{ totalStorage }}GB</span>
        </div>
      </div>
    </div>

    <!-- 主菜单 -->
    <nav class="sidebar-menu">
      <div class="menu-section">
        <h4 class="section-title">主要功能</h4>
        <el-menu 
          active-text-color="#3b82f6" 
          background-color="transparent" 
          text-color="#64748b"
          :default-active="activeMenu" 
          router
          class="menu-list"
        >
          <el-menu-item index="/" class="menu-item">
            <el-icon><House /></el-icon>
            <span>我的网盘</span>
          </el-menu-item>

          <el-menu-item index="/recent" class="menu-item">
            <el-icon><Clock /></el-icon>
            <span>最近文件</span>
          </el-menu-item>

          <el-menu-item index="/starred" class="menu-item">
            <el-icon><Star /></el-icon>
            <span>收藏夹</span>
          </el-menu-item>

          <el-menu-item index="/file-category" class="menu-item">
            <el-icon><FolderOpened /></el-icon>
            <span>分类文件</span>
          </el-menu-item>

          <el-menu-item index="/shared" class="menu-item">
            <el-icon><Share /></el-icon>
            <span>共享文件</span>
          </el-menu-item>

          <el-menu-item index="/trash" class="menu-item">
            <el-icon><Delete /></el-icon>
            <span>回收站</span>
          </el-menu-item>
        </el-menu>
      </div>
    </nav>

    <!-- 快捷标签 -->
    <div class="quick-tags">
      <div class="section-title">快捷访问</div>
      <div class="tags-container">
        <el-tag 
          v-for="tag in quickTags" 
          :key="tag.id" 
          :type="tag.type" 
          effect="plain" 
          round 
          class="quick-tag"
          @click="handleTagClick(tag)"
        >
          {{ tag.name }}
        </el-tag>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const activeMenu = ref(route.path)

const usedStorage = 45.8
const totalStorage = 100
const storagePercentage = computed(() => Math.round((usedStorage / totalStorage) * 100))
const storageColor = computed(() => {
  const percent = storagePercentage.value
  return percent > 90 ? '#ef4444'
    : percent > 70 ? '#f59e0b'
      : '#10b981'
})

const quickTags = ref([
  { id: 1, name: '工作文档', type: '' },
  { id: 2, name: '项目资料', type: 'success' },
  { id: 3, name: '个人照片', type: 'warning' },
  { id: 4, name: '学习资料', type: 'danger' }
])

const handleUpgrade = () => {
  console.log('升级存储空间')
}

const handleTagClick = (tag) => {
  console.log('跳转到标签:', tag.name)
}
</script>

<style scoped>
.sidebar-container {
  height: 100%;
  padding: 24px 0;
  display: flex;
  flex-direction: column;
  background: transparent;
}

.storage-info {
  padding: 0 24px 24px;
  border-bottom: 1px solid #e2e8f0;
  margin-bottom: 24px;
}

.storage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.storage-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.upgrade-btn {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  border: none;
  border-radius: 8px;
  font-size: 12px;
  padding: 6px 12px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.upgrade-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.storage-progress {
  display: flex;
  justify-content: center;
  margin-bottom: 20px;
}

.progress-content {
  text-align: center;
}

.percentage {
  font-size: 18px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1;
}

.size {
  font-size: 12px;
  color: #64748b;
  margin-top: 2px;
}

.storage-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.storage-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.storage-item:hover {
  background: #f1f5f9;
}

.storage-item .label {
  font-size: 12px;
  color: #64748b;
}

.storage-item .value {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}

.sidebar-menu {
  flex: 1;
  padding: 0 16px;
}

.menu-section {
  margin-bottom: 32px;
}

.section-title {
  margin: 0 0 16px 8px;
  font-size: 14px;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.menu-list {
  border: none;
  background: transparent;
}

.menu-item {
  margin-bottom: 4px;
  border-radius: 12px;
  height: 48px;
  line-height: 48px;
  transition: all 0.3s ease;
}

.menu-item:hover {
  background: #f1f5f9 !important;
  color: #3b82f6 !important;
}

.menu-item.is-active {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%) !important;
  color: white !important;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.menu-item .el-icon {
  font-size: 18px;
  margin-right: 12px;
}

.quick-tags {
  padding: 0 24px;
  margin-top: auto;
}

.quick-tags .section-title {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-tag {
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 16px;
}

.quick-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .sidebar-container {
    padding: 16px 0;
  }
  
  .storage-info {
    padding: 0 16px 16px;
    margin-bottom: 16px;
  }
  
  .sidebar-menu {
    padding: 0 8px;
  }
  
  .quick-tags {
    padding: 0 16px;
  }
  
  .menu-item {
    height: 44px;
    line-height: 44px;
  }
}
</style>