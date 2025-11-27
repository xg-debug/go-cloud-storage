<template>
  <el-container class="layout-container">
    <el-header class="main-header">
      <LayoutHeader />
    </el-header>

    <el-container class="main-container">
      <el-aside class="main-sidebar">
        <LayoutSidebar />
      </el-aside>

      <el-main class="main-content">
        <div class="content-wrapper">
          <router-view />
        </div>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import LayoutHeader from '@/components/layout/Header.vue'
import LayoutSidebar from '@/components/layout/Sidebar.vue'
</script>

<style scoped>
.layout-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: transparent;
  position: relative;
  overflow: hidden;
}

.main-header {
  height: 72px;
  padding: 12px 20px !important;
  z-index: 1000;
  background: #ffffff;
  border-bottom: 1px solid var(--border-light);
  position: relative;
}

.main-container {
  flex: 1;
  overflow: hidden;
  display: flex;
  padding: 0;
  gap: 0;
  position: relative;
  z-index: 1;
}

.main-sidebar {
  width: 280px !important;
  background: #ffffff;
  border-radius: 0;
  border-right: 1px solid var(--border-light);
  z-index: 900;
  flex-shrink: 0;
  overflow: hidden;
}

.main-content {
  flex: 1;
  padding: 0 !important;
  background: transparent;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.content-wrapper {
  flex: 1;
  background: #ffffff;
  border-radius: 0;
  overflow: hidden;
  position: relative;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .main-sidebar {
    width: 240px !important;
  }
  
  .main-container {
    padding: 0;
    gap: 0;
  }
}

@media (max-width: 768px) {
  .main-container {
    flex-direction: column;
    padding: 0;
    gap: 0;
  }
  
  .main-sidebar {
    width: 100% !important;
    height: auto;
    max-height: 200px;
  }
  
  .main-header {
    height: 64px;
    padding: 8px 16px !important;
  }
}
</style>