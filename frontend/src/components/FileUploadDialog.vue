<template>
  <el-dialog
    v-model="dialogVisible"
    title="文件上传"
    width="520px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <!-- Drop zone -->
    <div
      class="drop-zone"
      :class="{ dragging: isDragging }"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="onDrop"
    >
      <el-icon :size="48"><Upload /></el-icon>
      <p>将文件拖拽到此处 或</p>
      <el-button type="primary" link @click="triggerSelect">选择本地文件</el-button>
      <span class="drop-hint">支持多选，大文件自动分片上传</span>
      <input
        type="file"
        ref="uploadInputRef"
        multiple
        style="display: none"
        @change="onSelectFiles"
      />
    </div>

    <!-- Queue preview -->
    <div v-if="pendingFiles.length > 0" class="pending-preview">
      <div class="pp-head">待添加文件 ({{ pendingFiles.length }})</div>
      <div class="pp-list">
        <div v-for="(f, i) in pendingFiles" :key="i" class="pp-item">
          <el-icon :size="18" color="var(--cb-text-muted)"><Document /></el-icon>
          <span class="pp-name">{{ f.name }}</span>
          <span class="pp-size">{{ formatSize(f.size) }}</span>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
      <el-button type="primary" @click="confirmUpload" :disabled="pendingFiles.length === 0">
        添加到上传队列 ({{ pendingFiles.length }})
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useStore } from 'vuex'
import { ElMessage } from 'element-plus'
import { Upload, Document } from '@element-plus/icons-vue'
import { formatSize } from '@/utils/format'

const store = useStore()

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  parentId: { type: String, required: true, default: '' }
})

const currentParentId = ref(props.parentId)
watch(() => props.parentId, (v) => { currentParentId.value = v })

const emit = defineEmits(['update:modelValue', 'success'])

const dialogVisible = computed({
  get: () => props.modelValue,
  set: val => emit('update:modelValue', val)
})

const uploadInputRef = ref(null)
const pendingFiles = ref([])
const isDragging = ref(false)

function triggerSelect() { uploadInputRef.value.click() }

function onSelectFiles(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  addFiles(files)
}

function onDrop(e) {
  isDragging.value = false
  const files = Array.from(e.dataTransfer.files || [])
  addFiles(files)
}

function addFiles(files) {
  for (const f of files) {
    if (!f || f.size === 0) continue
    const exists = pendingFiles.value.find(p => p.name === f.name && p.size === f.size)
    if (!exists) pendingFiles.value.push(f)
  }
}

function confirmUpload() {
  if (pendingFiles.value.length === 0) return
  store.dispatch('upload/addToQueue', {
    files: pendingFiles.value,
    parentId: currentParentId.value
  })
  ElMessage.success(`已添加 ${pendingFiles.value.length} 个文件到上传队列`)
  emit('success')
  dialogVisible.value = false
  pendingFiles.value = []
}

function handleClose() {
  if (pendingFiles.value.length > 0) {
    pendingFiles.value = []
  }
  dialogVisible.value = false
}

</script>

<style scoped>
.drop-zone {
  border: 2px dashed var(--cb-line);
  padding: 48px 24px;
  text-align: center;
  cursor: pointer;
  transition: all var(--cb-transition);
  border-radius: var(--cb-radius-lg);
  background: var(--cb-surface-muted);
}
.drop-zone:hover,
.drop-zone.dragging {
  border-color: var(--cb-primary);
  background: var(--cb-primary-light);
}
.drop-zone .el-icon { color: var(--cb-primary); }
.drop-zone p { margin: 16px 0 4px; color: var(--cb-text-soft); font-size: 15px; }
.drop-hint { font-size: 12px; color: var(--cb-text-muted); display: block; margin-top: 8px; }

.pending-preview {
  margin-top: 16px;
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius);
  max-height: 200px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.pp-head {
  font-size: 12px;
  font-weight: 600;
  color: var(--cb-text-secondary);
  padding: 8px 12px;
  border-bottom: 1px solid var(--cb-border-light);
  background: var(--cb-bg-alt);
}
.pp-list { flex: 1; overflow-y: auto; padding: 4px; }
.pp-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 13px;
}
.pp-item:hover { background: var(--cb-bg-alt); }
.pp-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--cb-text); }
.pp-size { font-size: 11px; color: var(--cb-text-muted); flex-shrink: 0; }
</style>
