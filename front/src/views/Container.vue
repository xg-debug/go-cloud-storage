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
        <router-view />
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
}

.main-header {
  height: 64px;
  padding: 0 !important;
  /* 强制取消内边距 */
  z-index: 1000;
  background: white;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  position: relative;
  /* 确保header在最上层 */
}

.main-container {
  flex: 1;
  overflow: hidden;
}

.main-sidebar {
  width: 220px !important;
  /* 强制宽度 */
  background: white;
  box-shadow: 1px 0 4px rgba(0, 0, 0, 0.05);
  z-index: 900;
}

.main-content {
  padding: 20px;
  background-color: #f5f7fa;
  overflow: hidden;
}
</style>