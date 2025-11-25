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
  background-color: #f1f5f9;
}

.main-header {
  height: 64px;
  padding: 0 !important;
  z-index: 1000;
  background: white;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  position: relative;
}

.main-container {
  flex: 1;
  overflow: hidden;
  display: flex;
}

.main-sidebar {
  width: 240px !important;
  background: white;
  z-index: 900;
  flex-shrink: 0;
}

.main-content {
  flex: 1;
  padding: 16px;
  background-color: #f1f5f9;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.content-wrapper {
  flex: 1;
  background: white;
  border-radius: 16px;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  position: relative;
}
</style>