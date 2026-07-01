<template>
  <div class="duplicates-page">
    <div class="dp-header">
      <h2>重复文件</h2>
      <div class="dp-summary" v-if="totalWasted > 0">
        发现 {{ totalGroups }} 组重复文件，共可释放
        <strong>{{ formatSize(totalWasted) }}</strong> 空间
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadData" size="small" round>重新扫描</el-button>
    </div>

    <div v-if="loading" class="dp-loading">
      <el-icon class="is-loading" :size="32" color="var(--cb-primary)"><Loading /></el-icon>
      <p>正在扫描重复文件...</p>
    </div>

    <EmptyState v-else-if="groups.length === 0" :icon="Files" title="没有重复文件" description="所有文件都是唯一的" />

    <div v-else class="dp-groups">
      <div v-for="(group, gi) in groups" :key="group.fileHash" class="dp-group">
        <div class="dpg-head">
          <div class="dpg-head-left">
            <span class="dpg-tag">{{ group.count }} 个相同文件</span>
            <span>{{ group.sizeStr }} / 个</span>
            <span class="dpg-waste">可节省 {{ group.wastedStr }}</span>
          </div>
          <el-button size="small" type="danger" @click="deleteDuplicates(group)" :loading="deleting === gi">
            保留最新，删除其余
          </el-button>
        </div>
        <div class="dpg-files">
          <div v-for="f in group.files" :key="f.id" class="dpg-file">
            <el-icon :size="18" :color="getFileIconColor(f.name, false)">
              <component :is="getFileIcon(f.name, false)" />
            </el-icon>
            <span class="dpg-fname" :title="f.id">{{ f.name }}</span>
            <span class="dpg-fsize">{{ f.size_str }}</span>
            <span class="dpg-fdate">{{ f.created_at }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, Refresh, Files } from '@element-plus/icons-vue'
import { getDuplicateFiles, deleteFile } from '@/api/file'
import { getFileIcon, getFileIconColor } from '@/utils/fileIcon'
import EmptyState from '@/components/EmptyState.vue'
import { useFileActions } from '@/composables/useFileActions'

const { formatSize } = useFileActions()

const loading = ref(false)
const deleting = ref(-1)
const groups = ref([])
const totalWasted = ref(0)

const totalGroups = computed(() => groups.value.length)

async function loadData() {
  loading.value = true
  try {
    const res = await getDuplicateFiles()
    groups.value = res?.groups || []
    totalWasted.value = res?.totalWasted || 0
  } catch {
    ElMessage.error('扫描重复文件失败')
  } finally {
    loading.value = false
  }
}

async function deleteDuplicates(group) {
  const idx = groups.value.indexOf(group)
  if (idx === -1) return

  try {
    await ElMessageBox.confirm(
      `将删除 ${group.count - 1} 个重复文件，保留最新的一个。可释放 ${group.wastedStr} 空间。`,
      '确认删除',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch { return }

  deleting.value = idx

  const sorted = [...group.files].sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
  const toDelete = sorted.slice(1)

  let success = 0
  for (const f of toDelete) {
    try {
      await deleteFile(f.id)
      success++
    } catch (e) { console.error('Failed to delete duplicate file', f.id, e) }
  }

  if (success > 0) {
    ElMessage.success(`已删除 ${success} 个重复文件`)
  }
  deleting.value = -1
  loadData()
}

onMounted(loadData)
</script>

<style scoped>
.duplicates-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--cb-bg);
}

.dp-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px 28px 16px;
  flex-shrink: 0;
}
.dp-header h2 {
  font-size: 20px;
  font-weight: 800;
  color: var(--cb-text);
  margin: 0;
}
.dp-summary {
  flex: 1;
  font-size: 13px;
  color: var(--cb-text-secondary);
}
.dp-summary strong {
  color: var(--cb-danger);
  font-weight: 700;
}

.dp-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--cb-text-muted);
}

.dp-groups {
  flex: 1;
  overflow-y: auto;
  padding: 0 28px 32px;
}

.dp-group {
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius);
  background: var(--cb-surface);
  margin-bottom: 12px;
  overflow: hidden;
}

.dpg-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--cb-border-light);
  background: var(--cb-bg-alt);
}
.dpg-head-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--cb-text-secondary);
}
.dpg-tag {
  font-weight: 700;
  color: var(--cb-warning);
  background: #FEF3C7;
  padding: 2px 10px;
  border-radius: 99px;
  font-size: 12px;
}
.dpg-waste {
  color: var(--cb-danger);
  font-weight: 600;
}

.dpg-files {
  padding: 4px 8px;
}
.dpg-file {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 13px;
  transition: background var(--cb-transition-fast);
}
.dpg-file:hover { background: var(--cb-bg-alt); }
.dpg-fname {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--cb-text);
  font-weight: 500;
}
.dpg-fsize {
  color: var(--cb-text-muted);
  font-size: 12px;
  width: 70px;
  text-align: right;
}
.dpg-fdate {
  color: var(--cb-text-muted);
  font-size: 12px;
  width: 120px;
  text-align: right;
}

.cb-empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--cb-text-muted);
}
.cb-empty-state h3 { font-size: 16px; font-weight: 600; color: var(--cb-text); margin: 0; }
.cb-empty-state p { font-size: 13px; margin: 0; }
</style>
