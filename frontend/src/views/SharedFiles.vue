<template>
  <div class="page-wrap">
    <div class="page-hdr">
      <div class="page-hdr-title">
        <div class="page-hdr-icon" style="background:#ECFDF5;color:#059669;">
          <el-icon :size="20"><Share /></el-icon>
        </div>
        <div><h1>我的分享</h1><p>{{ activeShares }} 个有效，共 {{ totalShares }} 个</p></div>
      </div>
      <el-button :icon="Refresh" size="small" round @click="refreshData">刷新</el-button>
    </div>
    <div class="page-body">
      <div v-if="loading" class="cb-empty-state"><el-icon class="is-loading" :size="36"><Loading /></el-icon><p style="margin-top:16px;">加载中...</p></div>
      <template v-else-if="filteredShares.length">
        <div class="cb-table-wrap">
          <el-table :data="filteredShares" row-key="id">
            <el-table-column width="48"><template #default="{ row }"><el-icon :size="22" :color="iconColor(row.fileType)"><component :is="iconComp(row.fileType)" /></el-icon></template></el-table-column>
            <el-table-column label="文件" min-width="240" show-overflow-tooltip><template #default="{ row }"><span class="tbl-link">{{ row.fileName }}</span></template></el-table-column>
            <el-table-column label="有效期" width="140"><template #default="{ row }"><el-tag :type="row.status==='active'?'success':'warning'" size="small" effect="light" round>{{ row.expireAt || '永久' }}</el-tag></template></el-table-column>
            <el-table-column label="访问 / 下载" width="120"><template #default="{ row }"><b>{{ row.accessCount||0 }}</b><span style="color:#9CA3AF;"> / </span><b>{{ row.downloadCount||0 }}</b></template></el-table-column>
            <el-table-column label="分享时间" width="170"><template #default="{ row }">{{ fmtDate(row.createdAt) }}</template></el-table-column>
            <el-table-column label="操作" width="250" fixed="right">
              <template #default="{ row }">
                <el-button size="small" link @click="copyLink(row)"><el-icon><Link /></el-icon>复制链接</el-button>
                <el-button size="small" link @click="showDetail(row)"><el-icon><View /></el-icon>详情</el-button>
                <el-button size="small" type="danger" link @click="cancelShare(row)" :disabled="row.status!=='active'"><el-icon><Close /></el-icon>取消</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
      <div v-else class="cb-empty-state"><div class="empty-icon"><el-icon :size="36"><Share /></el-icon></div><h3>暂无分享</h3><p>在文件列表中点击分享按钮即可创建</p></div>
    </div>

    <el-dialog v-model="detailVisible" title="分享详情" width="500px">
      <div v-if="currentShare" class="d-grid">
        <div class="d-r"><span class="d-l">文件名</span><span class="d-v">{{ currentShare.fileName }}</span></div>
        <div class="d-r"><span class="d-l">大小</span><span class="d-v">{{ fmtSize(currentShare.fileSize) }}</span></div>
        <div class="d-r"><span class="d-l">提取码</span>
          <span v-if="currentShare.extractCode" class="d-code-group">
            <span class="d-code">{{ currentShare.extractCode }}</span>
            <el-button size="small" text @click="copyText(currentShare.extractCode)"><el-icon><CopyDocument /></el-icon></el-button>
          </span>
          <span v-else class="d-code">无</span>
        </div>
        <div class="d-r"><span class="d-l">链接</span><div class="d-link"><el-input :model-value="buildLink(currentShare)" readonly size="small" /><el-button size="small" @click="copyLink(currentShare)"><el-icon><CopyDocument /></el-icon>复制</el-button></div></div>
        <div class="d-r"><span class="d-l">访问/下载</span><span class="d-v">{{ currentShare.accessCount||0 }} / {{ currentShare.downloadCount||0 }}</span></div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button v-if="currentShare?.extractCode" type="primary" @click="copyFullShare(currentShare)"><el-icon><CopyDocument /></el-icon>复制链接和提取码</el-button>
        <el-button v-else type="primary" @click="copyLink(currentShare)"><el-icon><CopyDocument /></el-icon>复制链接</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" title="编辑分享" width="440px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="提取码"><el-input v-model="editForm.extractionCode" placeholder="留空不设置" maxlength="4" /></el-form-item>
        <el-form-item label="有效期"><el-select v-model="editForm.expireDays"><el-option label="永久有效" :value="0" /><el-option label="7天" :value="7" /><el-option label="30天" :value="30" /><el-option label="1年" :value="365" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button @click="editVisible = false">取消</el-button><el-button type="primary" @click="confirmEdit" :loading="editing">保存</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Close, CopyDocument, Link, Loading, Refresh, Share, View } from '@element-plus/icons-vue'
import { getUserShares, cancelShare as cancelApi, updateShare } from '@/api/share'

const loading = ref(false); const shares = ref([])
const detailVisible = ref(false); const editVisible = ref(false)
const currentShare = ref(null); const editing = ref(false)
const editForm = ref({ extractionCode: '', expireDays: 0 })

const totalShares = computed(() => shares.value.length)
const activeShares = computed(() => shares.value.filter(s => s.status === 'active').length)
const filteredShares = computed(() => [...shares.value].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt)))

async function load() { loading.value = true; try { const r = await getUserShares(); shares.value = (Array.isArray(r) ? r : []).map(s => ({ ...s, createdAt: s.createdAt ? new Date(s.createdAt) : null })) } catch { ElMessage.error('加载失败') } finally { loading.value = false } }
const iconComp = t => ({ image: 'Picture', video: 'VideoCamera', audio: 'Headset', document: 'Document' }[t] || 'Files')
const iconColor = t => ({ image: '#EC4899', video: '#EF4444', audio: '#8B5CF6', document: '#2F6BFF' }[t] || '#6B7280')
function buildLink(s) { if (!s) return ''; if (s.shareUrl?.startsWith('http')) return s.shareUrl; if (s.shareToken) return `${location.origin}/s/${s.shareToken}`; return '' }
async function copyLink(s) { try { await navigator.clipboard.writeText(buildLink(s)); ElMessage.success('链接已复制') } catch { ElMessage.error('复制失败') } }
async function copyText(t) { try { await navigator.clipboard.writeText(t); ElMessage.success('已复制') } catch { ElMessage.error('复制失败') } }
async function copyFullShare(s) { try { const t = `${buildLink(s)}\n提取码：${s.extractCode}`; await navigator.clipboard.writeText(t); ElMessage.success('已复制链接和提取码') } catch { ElMessage.error('复制失败') } }
function showDetail(s) { currentShare.value = s; detailVisible.value = true }
async function cancelShare(s) { try { await cancelApi(s.id); ElMessage.success('已取消'); load() } catch {} }
async function confirmEdit() { editing.value = true; try { await updateShare(currentShare.value.id, { extraction_code: editForm.value.extractionCode, expire_days: editForm.value.expireDays }); ElMessage.success('已更新'); editVisible.value = false; load() } catch {} finally { editing.value = false } }
function fmtSize(b) { if (!b) return '-'; const u = ['B','KB','MB','GB']; const k = 1024; const i = Math.floor(Math.log(b) / Math.log(k)); return parseFloat((b / Math.pow(k, i)).toFixed(2)) + ' ' + u[i] }
function fmtDate(d) { if (!d) return '-'; return new Date(d).toLocaleDateString('zh-CN', { year:'numeric', month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit' }) }
function refreshData() { load() }
onMounted(load)
</script>

<style scoped>
.tbl-link { font-weight: 600; color: var(--cb-text); cursor: pointer; }
.tbl-link:hover { color: var(--cb-primary); }
.d-grid { display: grid; gap: 14px; }
.d-r { display: flex; align-items: flex-start; gap: 12px; }
.d-l { width: 64px; flex-shrink: 0; font-size: 13px; color: var(--cb-text-muted); padding-top: 6px; }
.d-v { font-size: 14px; color: var(--cb-text); font-weight: 600; padding-top: 6px; }
.d-code { font-family: 'SF Mono', 'Fira Code', monospace; background: var(--cb-bg-alt); padding: 4px 10px; border-radius: 6px; font-size: 13px; font-weight: 600; }
.d-link { flex: 1; display: flex; gap: 8px; }
.d-code-group { display: inline-flex; align-items: center; gap: 4px; }
</style>
