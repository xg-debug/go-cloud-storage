<template>
  <div class="page-wrap">
    <div class="page-hdr">
      <div class="page-hdr-title">
        <div class="page-hdr-icon" style="background:#FEF2F2;color:#EF4444;">
          <el-icon :size="20"><Delete /></el-icon>
        </div>
        <div><h1>回收站</h1><p>已删除文件保留 7 天后自动清理</p></div>
      </div>
      <div v-if="selectedItems.length" class="hdr-actions">
        <span class="sel-badge">{{ selectedItems.length }} 项</span>
        <el-button :icon="Refresh" type="success" plain size="small" @click="handleRestore">还原</el-button>
        <el-button :icon="Delete" type="danger" plain size="small" @click="handleBatchDelete">彻底删除</el-button>
      </div>
    </div>
    <div class="page-body">
      <div style="margin-bottom:16px;">
        <el-button type="danger" plain :icon="Delete" size="small" :disabled="!trashItems.length" @click="openClearDialog">清空回收站</el-button>
      </div>
      <div v-if="!trashItems.length" class="cb-empty-state">
        <div class="empty-icon"><el-icon :size="36"><Delete /></el-icon></div>
        <h3>回收站为空</h3><p>删除的文件会显示在这里</p>
      </div>
      <div v-else class="cb-table-wrap">
        <el-table :data="trashItems" row-key="fileId" @selection-change="s => selectedItems = s">
          <el-table-column type="selection" width="44" />
          <el-table-column width="48">
            <template #default="{ row }"><el-icon :size="22" :color="row.is_dir ? '#F59E0B' : '#2F6BFF'"><Folder v-if="row.is_dir" /><Document v-else /></el-icon></template>
          </el-table-column>
          <el-table-column label="名称" min-width="260" show-overflow-tooltip><template #default="{ row }"><span class="tbl-link">{{ row.name }}</span></template></el-table-column>
          <el-table-column label="大小" width="100"><template #default="{ row }">{{ row.size_str }}</template></el-table-column>
          <el-table-column label="删除时间" width="170"><template #default="{ row }">{{ row.deletedDate }}</template></el-table-column>
          <el-table-column label="剩余" width="90"><template #default="{ row }"><el-tag :type="row.expireDays <= 3 ? 'danger' : 'warning'" size="small" effect="light">{{ row.expireDays }}天</el-tag></template></el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="success" link :icon="Refresh" @click="handleRestoreOne(row)">还原</el-button>
              <el-button size="small" type="danger" link :icon="Delete" @click="handleDeleteOne(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog v-model="clearDialogVisible" title="清空回收站" width="420px">
      <div style="text-align:center;padding:16px 0;">
        <div style="width:56px;height:56px;border-radius:50%;background:#FEF2F2;display:inline-flex;align-items:center;justify-content:center;margin-bottom:16px;"><el-icon :size="28" color="#EF4444"><Warning /></el-icon></div>
        <p style="font-size:15px;font-weight:600;color:var(--cb-text);">确认清空回收站？</p>
        <p style="font-size:13px;color:var(--cb-text-muted);margin-top:6px;">此操作不可恢复</p>
      </div>
      <template #footer><el-button @click="clearDialogVisible = false">取消</el-button><el-button type="danger" @click="confirmClear" :loading="deleting">确认清空</el-button></template>
    </el-dialog>

    <el-dialog v-model="deleteDialogVisible" title="彻底删除" width="420px">
      <div style="text-align:center;padding:16px 0;">
        <div style="width:56px;height:56px;border-radius:50%;background:#FEF2F2;display:inline-flex;align-items:center;justify-content:center;margin-bottom:16px;"><el-icon :size="28" color="#EF4444"><Warning /></el-icon></div>
        <p style="font-size:15px;font-weight:600;color:var(--cb-text);">文件将无法恢复，确认删除？</p>
      </div>
      <template #footer><el-button @click="deleteDialogVisible = false">取消</el-button><el-button type="danger" @click="confirmDelete">确认删除</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, Document, Folder, Refresh, Warning } from '@element-plus/icons-vue'
import { loadSoftDeletedFiles, deletePermanent, deleteSelected, clearRecycleBin, restore, restoreBatch } from '@/api/recycle'

const trashItems = ref([])
const selectedItems = ref([])
const clearDialogVisible = ref(false)
const deleteDialogVisible = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)

async function fetch() { try { const r = await loadSoftDeletedFiles(); trashItems.value = r.data || [] } catch {} }
async function handleRestore() { try { await restoreBatch(selectedItems.value.map(i => i.fileId)); ElMessage.success('已还原'); fetch() } catch { ElMessage.error('失败') } }
async function handleRestoreOne(r) { try { await restore(r.fileId); ElMessage.success('已还原'); fetch() } catch {} }
async function handleBatchDelete() { try { await deleteSelected(selectedItems.value.map(i => i.fileId)); ElMessage.success('已删除'); fetch() } catch {} }
function handleDeleteOne(r) { deleteTarget.value = r; deleteDialogVisible.value = true }
async function confirmDelete() { try { await deletePermanent(deleteTarget.value.fileId); ElMessage.success('已删除'); deleteDialogVisible.value = false; fetch() } catch {} }
function openClearDialog() { clearDialogVisible.value = true }
async function confirmClear() { deleting.value = true; try { await clearRecycleBin(); ElMessage.success('已清空'); clearDialogVisible.value = false; fetch() } catch {} finally { deleting.value = false } }
onMounted(fetch)
</script>

<style scoped>
.hdr-actions { display: flex; align-items: center; gap: 8px; }
.sel-badge { font-size: 12px; font-weight: 700; color: var(--cb-primary); background: var(--cb-primary-light); padding: 4px 10px; border-radius: 99px; }
.tbl-link { font-weight: 600; color: var(--cb-text); cursor: pointer; }
</style>
