<template>
  <div class="app-root">
    <!-- Left sidebar -->
    <aside class="app-sidebar" :class="{ collapsed: sidebarCollapsed }">
      <LayoutSidebar :collapsed="sidebarCollapsed" @toggle="sidebarCollapsed = !sidebarCollapsed" />
    </aside>

    <!-- Main area -->
    <div class="app-main">
      <!-- Top header -->
      <header class="app-header">
        <LayoutHeader @toggle-sidebar="sidebarCollapsed = !sidebarCollapsed" />
      </header>

      <!-- Content + right panel -->
      <div class="app-body">
        <main class="app-content">
          <router-view />
        </main>

        <!-- Right info panel (shown on file pages) -->
        <aside v-if="showRightPanel" class="app-right-panel">
          <RightPanel />
        </aside>
      </div>
    </div>

    <!-- Drop zone overlay -->
    <div class="drop-overlay" :class="{ active: isDragging }">
      <div class="drop-overlay-inner">
        <div class="drop-icon">
          <el-icon :size="36"><Upload /></el-icon>
        </div>
        <h2>松开即可上传</h2>
        <p>支持任意文件类型，单文件最大 5GB</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, provide, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import LayoutHeader from '@/components/layout/Header.vue'
import LayoutSidebar from '@/components/layout/Sidebar.vue'
import RightPanel from '@/components/layout/RightPanel.vue'
import { Upload } from '@element-plus/icons-vue'

const route = useRoute()
const sidebarCollapsed = ref(false)
const isDragging = ref(false)

// Only show right panel on main file pages
const showRightPanel = computed(() => {
  const name = route.name
  return ['MyDrive', 'Recent', 'Starred', 'FileCategory', 'FileCategoryType'].includes(name)
})

let dragCounter = 0

function onDragEnter(e) {
  e.preventDefault()
  dragCounter++
  if (dragCounter === 1) isDragging.value = true
}
function onDragLeave(e) {
  e.preventDefault()
  dragCounter--
  if (dragCounter === 0) isDragging.value = false
}
function onDragOver(e) { e.preventDefault() }
function onDrop(e) {
  e.preventDefault()
  dragCounter = 0
  isDragging.value = false
}

onMounted(() => {
  document.addEventListener('dragenter', onDragEnter)
  document.addEventListener('dragleave', onDragLeave)
  document.addEventListener('dragover', onDragOver)
  document.addEventListener('drop', onDrop)
})
onUnmounted(() => {
  document.removeEventListener('dragenter', onDragEnter)
  document.removeEventListener('dragleave', onDragLeave)
  document.removeEventListener('dragover', onDragOver)
  document.removeEventListener('drop', onDrop)
})
</script>

<style scoped>
.app-root {
  height: 100vh;
  display: flex;
  background: var(--cb-bg);
  overflow: hidden;
}

.app-sidebar {
  width: var(--cb-sidebar-w);
  flex-shrink: 0;
  transition: width var(--cb-transition);
  overflow: hidden;
}
.app-sidebar.collapsed { width: 72px; }

.app-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.app-header {
  height: var(--cb-header-h);
  flex-shrink: 0;
  background: var(--cb-surface);
  border-bottom: 1px solid var(--cb-border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  z-index: 20;
}

.app-body {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.app-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  background: var(--cb-bg);
}

.app-right-panel {
  width: var(--cb-right-panel-w);
  flex-shrink: 0;
  border-left: 1px solid var(--cb-border);
  background: var(--cb-surface);
  overflow-y: auto;
}

@media (max-width: 1200px) {
  .app-right-panel { display: none; }
}
@media (max-width: 768px) {
  .app-sidebar:not(.collapsed) { width: var(--cb-sidebar-w); }
}
</style>
