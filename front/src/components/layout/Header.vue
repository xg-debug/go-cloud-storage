<template>
  <div class="header-container">
    <!-- 左侧：面包屑导航和搜索 -->
    <div class="header-left">
      <el-breadcrumb separator="/" class="breadcrumb">
        <el-breadcrumb-item :to="{ path: '/' }">我的网盘</el-breadcrumb-item>
        <el-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="index">
          {{ item }}
        </el-breadcrumb-item>
      </el-breadcrumb>

      <el-input 
        v-model="searchQuery" 
        placeholder="搜索文件/文件夹" 
        class="search-input" 
        clearable 
        @keyup.enter="handleSearch"
      >
        <template #prefix>
          <el-icon class="search-icon">
            <Search />
          </el-icon>
        </template>
      </el-input>
    </div>

    <!-- 右侧：操作按钮和用户菜单 -->
    <div class="header-right">
      <div class="action-buttons">
        <el-button type="primary" @click="handleUpload" class="action-btn">
          <el-icon>
            <Upload />
          </el-icon>
          <span>上传</span>
        </el-button>
        <el-button @click="handleNewFolder" class="action-btn secondary">
          <el-icon>
            <FolderAdd />
          </el-icon>
          <span>新建文件夹</span>
        </el-button>
      </div>

      <el-dropdown class="user-dropdown">
        <div class="user-avatar">
          <el-avatar :size="40" :src="user?.avatar" class="avatar" />
          <div class="user-info">
            <span class="username" v-if="user?.username">{{ user.username }}</span>
          </div>
          <el-icon class="dropdown-icon">
            <ArrowDown />
          </el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu class="user-menu">
            <el-dropdown-item @click="goToProfile" class="menu-item">
              <el-icon><User /></el-icon>
              <span>个人中心</span>
            </el-dropdown-item>
            <el-dropdown-item @click="goToSettings" class="menu-item">
              <el-icon><Setting /></el-icon>
              <span>设置</span>
            </el-dropdown-item>
            <el-dropdown-item divided @click="handleLogout" class="menu-item logout">
              <el-icon><SwitchButton /></el-icon>
              <span>退出登录</span>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { logout } from '@/api/auth';
import { useStore } from 'vuex'
import { computed } from 'vue';

const router = useRouter()
const searchQuery = ref('')
const breadcrumbs = ref(['文档', '工作资料']) // 动态生成
const loading = ref(false) // 加载状态
const store = useStore()

const user = computed(() => store.state.userInfo)

const handleSearch = () => {
  if (!searchQuery.value.trim()) return
  router.push({ name: 'Search', query: { q: searchQuery.value.trim() } })
}

const handleUpload = () => {
  console.log('上传文件')
}

const handleNewFolder = () => {
  console.log('新建文件夹')
}

const goToProfile = () => {
  router.push({ name: 'UserProfile' })
}

const goToSettings = () => {
  router.push({ name: 'Settings' })
}

const handleLogout = async () => {
    try {
        await logout()

        store.commit('clearAuth')

        ElMessage.success('退出成功')
        router.push('/login')
    } catch (error) {
        console.error('退出失败:', error)
        store.commit('clearAuth')
        ElMessage.error('退出异常')
        router.push('/login')
    }
}
</script>

<style scoped>
.header-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  background: transparent;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 32px;
  flex: 1;
}

.breadcrumb {
  font-size: 14px;
  color: #64748b;
}

.breadcrumb :deep(.el-breadcrumb__item) {
  color: #64748b;
}

.breadcrumb :deep(.el-breadcrumb__item.is-link:hover) {
  color: #3b82f6;
}

.search-input {
  width: 320px;
  transition: all 0.3s ease;
}

.search-input :deep(.el-input__wrapper) {
  background: #f1f5f9;
  border-radius: 12px;
  padding: 0 16px;
  border: 1px solid transparent;
  transition: all 0.3s ease;
  box-shadow: none;
}

.search-input :deep(.el-input__wrapper:hover) {
  background: #e2e8f0;
  border-color: #cbd5e1;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  background: #ffffff;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-icon {
  color: #94a3b8;
  font-size: 16px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 24px;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

.action-btn {
  border-radius: 12px;
  padding: 10px 20px;
  font-weight: 500;
  transition: all 0.3s ease;
  border: none;
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn.primary {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  color: white;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.action-btn.primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.action-btn.secondary {
  background: #ffffff;
  color: #64748b;
  border: 1px solid #e2e8f0;
}

.action-btn.secondary:hover {
  background: #f8fafc;
  color: #3b82f6;
  border-color: #3b82f6;
  transform: translateY(-1px);
}

.user-dropdown {
  cursor: pointer;
}

.user-avatar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  border-radius: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  transition: all 0.3s ease;
}

.user-avatar:hover {
  background: #f1f5f9;
  border-color: #cbd5e1;
  transform: translateY(-1px);
}

.avatar {
  border: 2px solid #ffffff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  line-height: 1;
}

.user-role {
  font-size: 12px;
  color: #64748b;
  line-height: 1;
}

.dropdown-icon {
  font-size: 12px;
  color: #94a3b8;
  transition: transform 0.3s ease;
}

.user-dropdown:hover .dropdown-icon {
  transform: rotate(180deg);
}

.user-menu {
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  border: 1px solid #e2e8f0;
  padding: 8px 0;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  font-size: 14px;
  color: #64748b;
  transition: all 0.3s ease;
}

.menu-item:hover {
  background: #f8fafc;
  color: #3b82f6;
}

.menu-item.logout:hover {
  background: #fef2f2;
  color: #dc2626;
}

.menu-item .el-icon {
  font-size: 16px;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .header-container {
    padding: 0 24px;
  }
  
  .header-left {
    gap: 24px;
  }
  
  .search-input {
    width: 280px;
  }
}

@media (max-width: 768px) {
  .header-container {
    padding: 0 16px;
  }
  
  .header-left {
    gap: 16px;
  }
  
  .search-input {
    width: 200px;
  }
  
  .action-buttons {
    gap: 8px;
  }
  
  .action-btn {
    padding: 8px 16px;
  }
  
  .action-btn span {
    display: none;
  }
  
  .user-info {
    display: none;
  }
  
  .user-avatar {
    padding: 6px 12px;
  }
}
</style>