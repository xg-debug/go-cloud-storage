<template>
  <div class="sidebar-container">

    <!-- 主菜单 -->
    <nav class="sidebar-menu">
      <div class="menu-section">
        <el-menu
          active-text-color="#6366f1"
          background-color="transparent"
          text-color="#64748b"
          :default-active="activeMenu"
          router
          class="menu-list"
        >
          <el-menu-item index="/" class="menu-item">
            <div class="menu-icon-wrapper">
              <el-icon><House /></el-icon>
            </div>
            <span>全部文件</span>
          </el-menu-item>

          <el-menu-item index="/recent" class="menu-item">
            <div class="menu-icon-wrapper recent">
              <el-icon><Clock /></el-icon>
            </div>
            <span>最近访问</span>
          </el-menu-item>

          <el-menu-item index="/starred" class="menu-item">
            <div class="menu-icon-wrapper starred">
              <el-icon><Star /></el-icon>
            </div>
            <span>我的收藏</span>
          </el-menu-item>

          <el-sub-menu index="file-category" class="menu-item sub-menu">
            <template #title>
              <div class="menu-icon-wrapper category">
                <el-icon><FolderOpened /></el-icon>
              </div>
              <span>文件分类</span>
            </template>
            <el-menu-item index="/file/image" class="sub-menu-item">
              <div class="sub-icon image">
                <el-icon><Picture /></el-icon>
              </div>
              <span>图片</span>
            </el-menu-item>
            <el-menu-item index="/file/video" class="sub-menu-item">
              <div class="sub-icon video">
                <el-icon><VideoCamera /></el-icon>
              </div>
              <span>视频</span>
            </el-menu-item>
            <el-menu-item index="/file/audio" class="sub-menu-item">
              <div class="sub-icon audio">
                <el-icon><Headset /></el-icon>
              </div>
              <span>音乐</span>
            </el-menu-item>
            <el-menu-item index="/file/document" class="sub-menu-item">
              <div class="sub-icon document">
                <el-icon><Document /></el-icon>
              </div>
              <span>文档</span>
            </el-menu-item>
          </el-sub-menu>

          <el-menu-item index="/shared" class="menu-item">
            <div class="menu-icon-wrapper shared">
              <el-icon><Share /></el-icon>
            </div>
            <span>我的分享</span>
          </el-menu-item>

          <el-menu-item index="/recycle" class="menu-item">
            <div class="menu-icon-wrapper recycle">
              <el-icon><Delete /></el-icon>
            </div>
            <span>回收站</span>
          </el-menu-item>
        </el-menu>
      </div>
    </nav>

    <!-- 存储空间指示器 -->
    <div class="storage-section">
      <div class="storage-card">
        <div class="storage-header">
          <div class="storage-icon">
            <el-icon><Coin /></el-icon>
          </div>
          <div class="storage-title">
            <span class="title">存储空间</span>
            <span class="subtitle">{{ usedStorage }}GB / {{ totalStorage }}GB</span>
          </div>
        </div>
        
        <div class="storage-progress-bar">
          <div 
            class="progress-fill" 
            :style="{ width: storagePercentage + '%', background: storageGradient }"
          ></div>
        </div>
        
        <div class="storage-footer">
          <span class="storage-percent">已使用 {{ storagePercentage }}%</span>
          <el-button type="primary" size="small" class="upgrade-btn" link @click="handleUpgrade">
            <el-icon><ArrowRight /></el-icon>
            升级容量
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getUserStorageQuota } from '@/api/user'
import { Plus, Coin, ArrowRight } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const activeMenu = ref(route.path)

// 存储配额数据
const storageQuota = ref({
    Used: 0,
    Total: 10737418240,
    UsedPercent: 0,
    UsedGB: 0,
    TotalGB: 10
})

// 计算已使用存储（GB）
const usedStorage = computed(() => {
    return storageQuota.value.UsedGB.toFixed(2)
})

// 计算总存储（GB）
const totalStorage = computed(() => {
    return storageQuota.value.TotalGB.toFixed(2)
})

// 计算存储百分比
const storagePercentage = computed(() => {
    return storageQuota.value.UsedPercent
})

// 动态渐变色
const storageGradient = computed(() => {
    const percent = storagePercentage.value
    if (percent > 90) return 'linear-gradient(90deg, #ef4444 0%, #f87171 100%)'
    if (percent > 70) return 'linear-gradient(90deg, #f59e0b 0%, #fbbf24 100%)'
    return 'linear-gradient(90deg, #6366f1 0%, #8b5cf6 100%)'
})

const loadStorageQuota = async () => {
    try {
        const res = await getUserStorageQuota()
        if (res) {
            storageQuota.value = {
                Used: res.used || 0,
                Total: res.total || 10737418240,
                UsedPercent: res.used_percent || 0,
                UsedGB: res.used_gb || 0,
                TotalGB: res.total_gb || 10
            }
        }
    } catch (error) {
        console.error('加载存储配额失败:', error)
    }
}

const handleUpgrade = () => {
    console.log('升级存储空间')
}

const handleUpload = () => {
    router.push('/')
}

onMounted(() => {
    loadStorageQuota()
})
</script>

<style scoped>
.sidebar-container {
  height: 100%;
  padding: 20px 0;
  display: flex;
  flex-direction: column;
  background: transparent;
  overflow-y: auto;
}

/* 上传按钮区域 */
.upload-section {
  padding: 0 16px 20px;
}

.upload-btn {
  width: 100%;
  height: 48px;
  border-radius: var(--radius-lg);
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--primary-gradient) !important;
  border: none !important;
  box-shadow: 0 4px 15px rgba(99, 102, 241, 0.35);
  transition: all var(--transition-normal);
}

.upload-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.45);
}

.upload-btn:active {
  transform: translateY(0);
}

.upload-icon {
  font-size: 18px;
}

/* 菜单区域 */
.sidebar-menu {
  flex: 1;
  padding: 0 12px;
  overflow-y: auto;
}

.menu-section {
  margin-bottom: 0;
}

.menu-list {
  border: none !important;
  background: transparent !important;
}

/* 菜单图标包装器 */
.menu-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  background: var(--primary-light);
  color: var(--primary-color);
  transition: all var(--transition-fast);
}

.menu-icon-wrapper.recent {
  background: #fef3c7;
  color: #f59e0b;
}

.menu-icon-wrapper.starred {
  background: #fce7f3;
  color: #ec4899;
}

.menu-icon-wrapper.category {
  background: #dbeafe;
  color: #3b82f6;
}

.menu-icon-wrapper.shared {
  background: #d1fae5;
  color: #10b981;
}

.menu-icon-wrapper.recycle {
  background: #fee2e2;
  color: #ef4444;
}

/* 主菜单项样式 */
.menu-list :deep(.el-menu-item) {
  margin-bottom: 4px;
  border-radius: var(--radius-lg);
  height: 52px;
  padding: 0 12px !important;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
}

.menu-list :deep(.el-menu-item span) {
  font-size: 14px;
  font-weight: 500;
}

.menu-list :deep(.el-menu-item:hover) {
  background: rgba(99, 102, 241, 0.08) !important;
}

.menu-list :deep(.el-menu-item:hover) .menu-icon-wrapper {
  transform: scale(1.05);
}

.menu-list :deep(.el-menu-item.is-active) {
  background: #f1f5f9 !important;
  color: var(--primary-color) !important;
}

.menu-list :deep(.el-menu-item.is-active) .menu-icon-wrapper {
  background: var(--primary-light);
  color: var(--primary-color);
}

/* 子菜单样式 */
.menu-list :deep(.el-sub-menu) {
  margin-bottom: 4px;
}

.menu-list :deep(.el-sub-menu .el-sub-menu__title) {
  border-radius: var(--radius-lg);
  height: 52px;
  padding: 0 12px !important;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
}

.menu-list :deep(.el-sub-menu .el-sub-menu__title span) {
  font-size: 14px;
  font-weight: 500;
}

.menu-list :deep(.el-sub-menu .el-sub-menu__title:hover) {
  background: rgba(99, 102, 241, 0.08) !important;
}

.menu-list :deep(.el-sub-menu .el-menu) {
  background: transparent !important;
  padding: 4px 0 4px 20px;
}

/* 子菜单项图标 */
.sub-icon {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 10px;
  font-size: 14px;
  transition: all var(--transition-fast);
}

.sub-icon.image {
  background: #dbeafe;
  color: #3b82f6;
}

.sub-icon.video {
  background: #fce7f3;
  color: #ec4899;
}

.sub-icon.audio {
  background: #d1fae5;
  color: #10b981;
}

.sub-icon.document {
  background: #fef3c7;
  color: #f59e0b;
}

.menu-list :deep(.el-sub-menu .el-menu-item) {
  margin: 2px 0;
  border-radius: var(--radius-md);
  height: 42px;
  padding: 0 12px !important;
  transition: all var(--transition-fast);
}

.menu-list :deep(.el-sub-menu .el-menu-item span) {
  font-size: 13px;
}

.menu-list :deep(.el-sub-menu .el-menu-item:hover) {
  background: rgba(99, 102, 241, 0.08) !important;
}

.menu-list :deep(.el-sub-menu .el-menu-item.is-active) {
  background: #f1f5f9 !important;
  color: var(--primary-color) !important;
}

.menu-list :deep(.el-sub-menu .el-menu-item.is-active) .sub-icon {
  background: var(--primary-light);
  color: var(--primary-color);
}

/* 存储空间区域 */
.storage-section {
  padding: 16px;
  margin-top: auto;
}

.storage-card {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-radius: var(--radius-lg);
  padding: 16px;
  border: 1px solid var(--border-light);
}

.storage-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

.storage-icon {
  width: 40px;
  height: 40px;
  background: var(--primary-gradient);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 18px;
}

.storage-title {
  display: flex;
  flex-direction: column;
}

.storage-title .title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.storage-title .subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

.storage-progress-bar {
  height: 8px;
  background: var(--border-light);
  border-radius: var(--radius-full);
  overflow: hidden;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.5s ease;
}

.storage-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.storage-percent {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.upgrade-btn {
  font-size: 12px !important;
  font-weight: 500 !important;
  color: var(--primary-color) !important;
  padding: 0 !important;
  height: auto !important;
  background: transparent !important;
  box-shadow: none !important;
}

.upgrade-btn:hover {
  transform: none !important;
  box-shadow: none !important;
  color: var(--primary-dark) !important;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .sidebar-container {
    padding: 12px 0;
  }

  .upload-section {
    padding: 0 12px 16px;
  }

  .sidebar-menu {
    padding: 0 8px;
  }

  .storage-section {
    padding: 12px;
  }
}
</style>