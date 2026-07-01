<template>
  <transition name="slide-right">
    <div v-if="visible" class="upload-queue-panel">
      <div class="uq-header">
        <h3>上传任务 <span class="uq-badge">{{ tasks.length }}</span></h3>
        <div class="uq-header-actions">
          <el-button v-if="hasCompleted" size="small" link @click="store.dispatch('upload/clearCompleted')">清除已完成</el-button>
          <el-button size="small" link @click="visible = false"><el-icon :size="16"><Close /></el-icon></el-button>
        </div>
      </div>

      <div class="uq-list" v-if="tasks.length > 0">
        <div v-for="task in tasks" :key="task.id" class="uq-task" :class="'status-' + task.status">
          <div class="uq-task-icon">
            <el-icon v-if="task.status === 'uploading'" class="is-loading" :size="18"><Loading /></el-icon>
            <el-icon v-else-if="task.status === 'completed'" :size="18" color="#10B981"><CircleCheckFilled /></el-icon>
            <el-icon v-else-if="task.status === 'failed'" :size="18" color="#EF4444"><CircleCloseFilled /></el-icon>
            <el-icon v-else-if="task.status === 'paused'" :size="18" color="#F59E0B"><VideoPause /></el-icon>
            <el-icon v-else :size="18" color="#9CA3AF"><Clock /></el-icon>
          </div>
          <div class="uq-task-body">
            <div class="uq-task-name" :title="task.fileName">{{ task.fileName }}</div>
            <div class="uq-task-meta">
              <span>{{ formatSize(task.fileSize) }}</span>
              <span v-if="task.status === 'uploading'">{{ task.progress }}%</span>
              <span v-else-if="task.status === 'failed'" class="uq-error">{{ task.error || '上传失败' }}</span>
              <span v-else-if="task.status === 'completed'">已完成</span>
              <span v-else-if="task.status === 'paused'">已暂停</span>
              <span v-else-if="task.status === 'pending'">等待中</span>
              <span v-else-if="task.status === 'cancelled'">已取消</span>
            </div>
            <el-progress
              v-if="task.status === 'uploading'"
              :percentage="task.progress"
              :stroke-width="4"
              :show-text="false"
              :color="task.progress === 100 ? '#10B981' : '#2F6BFF'"
            />
          </div>
          <div class="uq-task-actions">
            <el-button
              v-if="task.status === 'uploading'"
              size="small" circle :icon="VideoPause"
              @click="store.dispatch('upload/pauseTask', task.id)"
              title="暂停"
            />
            <el-button
              v-if="task.status === 'paused'"
              size="small" circle type="primary" :icon="VideoPlay"
              @click="store.dispatch('upload/resumeTask', task.id)"
              title="继续"
            />
            <el-button
              v-if="task.status === 'failed'"
              size="small" circle :icon="Refresh"
              @click="store.dispatch('upload/retryTask', task.id)"
              title="重试"
            />
            <el-button
              v-if="task.status !== 'completed' && task.status !== 'cancelled'"
              size="small" circle :icon="Close"
              @click="store.dispatch('upload/cancelTask', task.id)"
              title="取消"
            />
            <el-button
              v-if="task.status === 'completed' || task.status === 'cancelled'"
              size="small" circle :icon="Delete"
              @click="store.dispatch('upload/removeTask', task.id)"
              title="移除"
            />
          </div>
        </div>
      </div>
      <div v-else class="uq-empty">暂无上传任务</div>
    </div>
  </transition>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useStore } from 'vuex'
import { Close, Loading, CircleCheckFilled, CircleCloseFilled, VideoPause, VideoPlay, Clock, Refresh, Delete } from '@element-plus/icons-vue'
import { formatSize } from '@/utils/format'

const store = useStore()

const visible = ref(false)

const tasks = computed(() => store.state.upload.tasks)
const hasCompleted = computed(() => tasks.value.some(t => t.status === 'completed' || t.status === 'cancelled'))
const hasActive = computed(() => tasks.value.some(t => t.status === 'uploading' || t.status === 'pending' || t.status === 'paused'))
const activeTasks = computed(() => tasks.value.filter(t => t.status !== 'completed' && t.status !== 'cancelled'))

function show() { visible.value = true }
function hide() { visible.value = false }
function toggle() { visible.value = !visible.value }

defineExpose({ show, hide, toggle, visible: computed(() => visible.value), hasActive, activeTasks })
</script>

<style scoped>
.upload-queue-panel {
  position: fixed;
  right: 0;
  top: 56px;
  bottom: 0;
  width: 380px;
  background: var(--cb-surface);
  border-left: 1px solid var(--cb-border);
  box-shadow: var(--cb-shadow-lg);
  z-index: 500;
  display: flex;
  flex-direction: column;
}

.uq-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--cb-border-light);
  flex-shrink: 0;
}
.uq-header h3 {
  font-size: 15px;
  font-weight: 700;
  color: var(--cb-text);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.uq-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 99px;
  background: var(--cb-primary);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
}
.uq-header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.uq-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
}
.uq-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--cb-text-muted);
  font-size: 14px;
}

.uq-task {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  border-radius: 10px;
  margin-bottom: 6px;
  border: 1px solid var(--cb-border-light);
  background: var(--cb-surface);
  transition: all var(--cb-transition-fast);
}
.uq-task:hover { border-color: var(--cb-border-strong); }
.uq-task.status-completed { background: #F0FDF4; border-color: #BBF7D0; }
.uq-task.status-failed { background: #FEF2F2; border-color: #FECACA; }

.uq-task-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 2px;
}

.uq-task-body {
  flex: 1;
  min-width: 0;
}
.uq-task-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--cb-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4px;
}
.uq-task-meta {
  font-size: 11px;
  color: var(--cb-text-muted);
  margin-bottom: 6px;
  display: flex;
  gap: 8px;
}
.uq-error { color: #EF4444; }

.uq-task-actions {
  flex-shrink: 0;
  display: flex;
  gap: 4px;
  margin-top: 2px;
}

.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform .25s var(--cb-ease), opacity .25s var(--cb-ease);
}
.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
