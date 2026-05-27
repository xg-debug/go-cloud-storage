<template>
  <div class="page-wrap">
    <div class="page-hdr">
      <div class="page-hdr-title">
        <div class="page-hdr-icon" :style="{ background: catBg(selectedCategory), color: catColor(selectedCategory) }">
          <el-icon :size="20"><component :is="catIcon(selectedCategory)" /></el-icon>
        </div>
        <div>
          <h1>{{ selectedCategory ? catName(selectedCategory) : '文件分类' }}</h1>
          <p>{{ selectedCategory ? total + ' 个文件' : '按类型浏览文件' }}</p>
        </div>
      </div>
      <el-button-group size="small" v-if="selectedCategory">
        <el-button :type="viewMode === 'list' ? 'primary' : ''" @click="viewMode = 'list'"><el-icon><List /></el-icon></el-button>
        <el-button :type="viewMode === 'grid' ? 'primary' : ''" @click="viewMode = 'grid'"><el-icon><Grid /></el-icon></el-button>
      </el-button-group>
    </div>

    <div class="page-body">
      <div v-if="!selectedCategory" class="cat-picker">
        <h2 class="picker-title">选择文件分类</h2>
        <div class="cat-grid">
          <button v-for="c in categories" :key="c.type" class="cat-btn" @click="$router.push(`/file/${c.type}`)">
            <div class="cat-btn-icon" :style="{ background: catBg(c.type), color: catColor(c.type) }">
              <el-icon :size="32"><component :is="c.icon" /></el-icon>
            </div>
            <strong>{{ c.name }}</strong>
            <span>{{ c.extensions.slice(0, 4).join('、') }}</span>
          </button>
        </div>
      </div>

      <template v-else>
        <div v-if="loading" class="cb-empty-state"><el-icon class="is-loading" :size="36"><Loading /></el-icon><p style="margin-top:16px;">加载中...</p></div>
        <div v-else-if="viewMode === 'grid'" class="cat-file-grid">
          <div v-for="item in files" :key="item.id" class="cat-gc" @dblclick="previewHandler(item)">
            <div class="cat-gc-thumb">
              <img v-if="item.thumbnail_url" :src="item.thumbnail_url" :alt="item.name" />
              <el-icon v-else :size="44" :color="catColor(selectedCategory)"><component :is="catIcon(selectedCategory)" /></el-icon>
            </div>
            <div class="cat-gc-name" :title="item.name">{{ item.name }}</div>
            <div class="cat-gc-size">{{ item.size_str }}</div>
          </div>
        </div>
        <div v-else class="cb-table-wrap">
          <el-table :data="files" row-key="id" @row-dblclick="previewHandler">
            <el-table-column width="48"><template #default><el-icon :size="22" :color="catColor(selectedCategory)"><component :is="catIcon(selectedCategory)" /></el-icon></template></el-table-column>
            <el-table-column label="名称" min-width="320" show-overflow-tooltip><template #default="{ row }"><span class="tbl-link">{{ row.name }}</span></template></el-table-column>
            <el-table-column label="大小" width="100"><template #default="{ row }">{{ row.size_str }}</template></el-table-column>
            <el-table-column label="日期" width="170"><template #default="{ row }">{{ row.created_at }}</template></el-table-column>
            <el-table-column label="操作" width="160" fixed="right">
              <template #default="{ row }">
                <el-button size="small" link @click="doDownload(row)"><el-icon><Download /></el-icon>下载</el-button>
                <el-button size="small" type="danger" link @click="doDelete(row)"><el-icon><Delete /></el-icon>删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <div v-if="!loading && files.length === 0" class="cb-empty-state">
          <div class="empty-icon"><el-icon :size="36"><component :is="catIcon(selectedCategory)" /></el-icon></div>
          <h3>暂无文件</h3><p>该分类下还没有文件</p>
        </div>
      </template>
    </div>

    <el-dialog v-model="deleteVisible" title="确认删除" width="420px">
      <div style="text-align:center;padding:16px 0;">
        <div style="width:56px;height:56px;border-radius:50%;background:#FEF2F2;display:inline-flex;align-items:center;justify-content:center;margin-bottom:16px;"><el-icon :size="28" color="#EF4444"><Warning /></el-icon></div>
        <p style="font-size:15px;font-weight:600;color:var(--cb-text);">确定删除 <strong>{{ deleteTarget.name }}</strong>？</p>
        <p style="font-size:13px;color:var(--cb-text-muted);margin-top:6px;">删除后可在回收站保留 7 天</p>
      </div>
      <template #footer><el-button @click="deleteVisible = false">取消</el-button><el-button type="danger" @click="confirmDelete" :loading="deleting">确认删除</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Delete, Download, Grid, List, Loading, Warning } from '@element-plus/icons-vue'
import { getFilesByCategory, deleteFile, downloadFile } from '@/api/file'

const route = useRoute()
const viewMode = ref('list')
const files = ref([])
const total = ref(0)
const loading = ref(false)
const selectedCategory = ref(route.params.type || null)
const deleteVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)

const categories = [
  { type: 'image', name: '图片', icon: 'Picture', color: '#EC4899', extensions: ['jpg','jpeg','png','gif','bmp','webp','svg'] },
  { type: 'video', name: '视频', icon: 'VideoCamera', color: '#EF4444', extensions: ['mp4','avi','mov','wmv','flv','mkv','webm'] },
  { type: 'audio', name: '音频', icon: 'Headset', color: '#8B5CF6', extensions: ['mp3','wav','flac','aac','ogg'] },
  { type: 'document', name: '文档', icon: 'Document', color: '#2F6BFF', extensions: ['pdf','doc','docx','xls','xlsx','ppt','pptx','txt'] }
]
const catMap = Object.fromEntries(categories.map(c => [c.type, c]))
const catName = t => catMap[t]?.name || ''
const catIcon = t => catMap[t]?.icon || 'FolderOpened'
const catColor = t => catMap[t]?.color || '#2F6BFF'
const catBg = t => ({ image: '#FDF2F8', video: '#FEF2F2', audio: '#F5F3FF', document: '#EEF4FF' }[t] || '#F8F9FB')

async function loadFiles(type) {
  loading.value = true
  try { const r = await getFilesByCategory({ fileType: type, sortBy: 'created_at', sortOrder: 'desc', page: 1, pageSize: 100 }); files.value = r.list || []; total.value = r.total || 0 }
  catch {} finally { loading.value = false }
}
watch(() => route.params.type, t => { selectedCategory.value = t; if (t) loadFiles(t) }, { immediate: true })
function previewHandler(f) { if (f.file_url) window.open(f.file_url, '_blank'); else ElMessage.info('暂不支持预览') }
async function doDownload(f) { try { const b = await downloadFile(f.id); const u = URL.createObjectURL(b); const a = document.createElement('a'); a.href = u; a.download = f.name; a.click(); URL.revokeObjectURL(u) } catch { ElMessage.error('下载失败') } }
function doDelete(f) { deleteTarget.value = f; deleteVisible.value = true }
async function confirmDelete() { deleting.value = true; try { await deleteFile(deleteTarget.value.id); ElMessage.success('已删除'); deleteVisible.value = false; loadFiles(selectedCategory.value) } catch { ElMessage.error('删除失败') } finally { deleting.value = false } }
</script>

<style scoped>
.cat-picker { padding: 12px 0; }
.picker-title { font-size: 18px; font-weight: 800; margin: 0 0 20px; color: var(--cb-text); }
.cat-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.cat-btn { display: flex; flex-direction: column; align-items: center; gap: 12px; padding: 32px 20px; border: 1px solid var(--cb-border); border-radius: var(--cb-radius-lg); background: var(--cb-surface); cursor: pointer; transition: all var(--cb-transition-fast); }
.cat-btn:hover { border-color: var(--cb-border-strong); transform: translateY(-2px); box-shadow: var(--cb-shadow); }
.cat-btn-icon { width: 72px; height: 72px; border-radius: 18px; display: flex; align-items: center; justify-content: center; transition: transform .25s var(--cb-ease-spring); }
.cat-btn:hover .cat-btn-icon { transform: scale(1.08); }
.cat-btn strong { font-size: 15px; font-weight: 700; color: var(--cb-text); }
.cat-btn span { font-size: 12px; color: var(--cb-text-muted); }
.cat-file-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(176px, 1fr)); gap: 14px; }
.cat-gc { padding: 20px 14px; text-align: center; border: 1px solid var(--cb-border); border-radius: var(--cb-radius); background: var(--cb-surface); cursor: pointer; transition: all var(--cb-transition-fast); }
.cat-gc:hover { border-color: var(--cb-border-strong); transform: translateY(-2px); box-shadow: var(--cb-shadow-sm); }
.cat-gc-thumb { height: 72px; display: flex; align-items: center; justify-content: center; margin-bottom: 12px; }
.cat-gc-thumb img { max-width: 100%; max-height: 100%; object-fit: cover; border-radius: 6px; }
.cat-gc-name { font-size: 13px; font-weight: 600; color: var(--cb-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-bottom: 4px; }
.cat-gc-size { font-size: 11px; color: var(--cb-text-muted); }
.tbl-link { font-weight: 600; color: var(--cb-text); cursor: pointer; }
@media (max-width: 640px) { .cat-grid { grid-template-columns: repeat(2, 1fr); } }
</style>
