<template>
  <div class="page-wrap">
    <div class="page-hdr">
      <div class="page-hdr-title">
        <div class="page-hdr-icon" style="background:#FDF2F8;color:#DB2777;">
          <el-icon :size="20"><StarFilled /></el-icon>
        </div>
        <div><h1>收藏夹</h1><p>{{ totalCount }} 个收藏</p></div>
      </div>
    </div>
    <div class="page-body">
      <div v-if="loading" class="cb-empty-state"><el-icon class="is-loading" :size="36"><Loading /></el-icon><p style="margin-top:16px;">加载中...</p></div>
      <template v-else-if="starredItems.length > 0">
        <div class="cb-table-wrap">
          <el-table :data="starredItems" row-key="file_id" @row-dblclick="openFile">
            <el-table-column width="48">
              <template #default="{ row }"><el-icon :size="22" :color="getFileIconColor(row.name, row.is_dir)"><component :is="getFileIcon(row.name, row.is_dir)" /></el-icon></template>
            </el-table-column>
            <el-table-column label="名称" min-width="280" show-overflow-tooltip>
              <template #default="{ row }"><span class="tbl-link">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="大小" width="100"><template #default="{ row }">{{ row.size_str || '-' }}</template></el-table-column>
            <el-table-column label="收藏时间" width="170"><template #default="{ row }">{{ row.created_at }}</template></el-table-column>
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button size="small" link @click="previewFile(row)" v-if="!row.is_dir"><el-icon><View /></el-icon>预览</el-button>
                <el-button size="small" link @click="downloadFile(row)" v-if="!row.is_dir"><el-icon><Download /></el-icon>下载</el-button>
                <el-button size="small" type="danger" link @click="unFavorite(row)"><el-icon><StarFilled /></el-icon>取消</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <div style="margin-top:24px;display:flex;justify-content:center;" v-if="totalCount > pageSize">
          <el-pagination background layout="prev, pager, next, total" :total="totalCount" :page-size="pageSize" v-model:current-page="currentPage" @current-change="fetchFavorites" />
        </div>
      </template>
      <div v-else class="cb-empty-state">
        <div class="empty-icon"><el-icon :size="36"><Star /></el-icon></div>
        <h3>暂无收藏</h3><p>在文件列表中点击收藏按钮即可添加</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Download, Loading, Star, StarFilled, View } from '@element-plus/icons-vue'
import { getFavorites, cancelFavorite } from '@/api/favorite'
import { previewFile as previewApi, downloadFile as dlFile } from '@/api/file'
import { getFileIcon, getFileIconColor } from '@/utils/fileIcon'

const router = useRouter()
const loading = ref(false)
const starredItems = ref([])
const totalCount = ref(0)
const currentPage = ref(1)
const pageSize = 10

async function fetchFavorites() {
  loading.value = true
  try { const r = await getFavorites(currentPage.value, pageSize); starredItems.value = r.favoriteList || []; totalCount.value = r.total || 0 }
  catch {} finally { loading.value = false }
}
async function unFavorite(row) {
  try { await cancelFavorite(row.file_id); starredItems.value = starredItems.value.filter(i => i.file_id !== row.file_id); totalCount.value--; ElMessage.success('已取消') }
  catch { ElMessage.error('操作失败') }
}
function openFile(row) {
  if (row.is_dir) router.push({ name: 'MyDrive', query: { parentId: row.file_id } })
  else window.open(row.file_url, '_blank')
}
async function previewFile(row) {
  try { const r = await previewApi(row.file_id); if (r?.file_url) window.open(r.file_url, '_blank'); else ElMessage.info('暂不支持预览') }
  catch { ElMessage.info('暂不支持预览') }
}
async function downloadFile(row) {
  try { const b = await dlFile(row.file_id); const u = URL.createObjectURL(b); const a = document.createElement('a'); a.href = u; a.download = row.name; a.click(); URL.revokeObjectURL(u) }
  catch { ElMessage.error('下载失败') }
}
onMounted(fetchFavorites)
</script>

<style scoped>
.tbl-link { font-weight: 600; color: var(--cb-text); cursor: pointer; }
.tbl-link:hover { color: var(--cb-primary); }
</style>
