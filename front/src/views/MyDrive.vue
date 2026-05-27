<template>
  <div class="drive">
    <!-- Breadcrumb bar (when navigating folders or searching) -->
    <div v-if="isInFolder || isSearching" class="drive-bar">
      <div class="drive-bar-left">
        <el-button v-show="pathIdStack.length > 1" :icon="ArrowLeft" circle size="small" @click="goBack" />
        <el-breadcrumb separator="/">
          <el-breadcrumb-item @click="goRoot"><el-icon :size="14"><HomeFilled /></el-icon> 我的文件</el-breadcrumb-item>
          <el-breadcrumb-item v-for="(name, idx) in currentPath" :key="idx" @click="goToBreadcrumb(idx)">
            {{ name }}
          </el-breadcrumb-item>
        </el-breadcrumb>
        <span v-if="!isSearching" class="drive-count">{{ total }} 项</span>
      </div>
      <div class="drive-bar-right">
        <el-button-group size="small">
          <el-button :type="viewMode === 'grid' ? 'primary' : ''" @click="viewMode = 'grid'">
            <el-icon><Grid /></el-icon>
          </el-button>
          <el-button :type="viewMode === 'list' ? 'primary' : ''" @click="viewMode = 'list'">
            <el-icon><List /></el-icon>
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- Search banner -->
    <div v-if="isSearching" class="drive-search-banner">
      搜索 "{{ searchKeyword }}" 找到 {{ fileList.length }} 个结果
      <el-button type="primary" link size="small" @click="clearSearch(); loadFiles()">清除搜索</el-button>
    </div>

    <!-- Welcome area (only at root, not searching) -->
    <template v-if="!isInFolder && !isSearching">
      <div class="drive-welcome">
        <div class="welcome-greeting">
          <h1>你好{{ greetingEmoji }}，今天想找点什么呢？</h1>
          <p>{{ greetingHint }}</p>
        </div>
        <div class="welcome-actions">
          <el-button type="primary" :icon="Upload" size="large" round @click="uploadDialogVisible = true">上传文件</el-button>
          <FileUploadDialog v-model="uploadDialogVisible" :parent-id="currentParentId" @success="handleUploadSuccess" />
          <el-button :icon="FolderAdd" size="large" round @click="handleNewFolder">新建文件夹</el-button>
        </div>
      </div>

      <!-- Quick Access -->
      <div v-if="quickFolders.length > 0" class="drive-section">
        <h2 class="section-head">快速访问</h2>
        <div class="quick-scroll">
          <div
            v-for="f in quickFolders" :key="f.id"
            class="quick-card"
            :style="{ '--qc-color': f.color }"
            @click="navigateTo(f)"
          >
            <div class="qc-icon">
              <el-icon :size="22"><Folder /></el-icon>
            </div>
            <div class="qc-info">
              <strong>{{ f.name }}</strong>
              <span>{{ f.fileCount || 0 }} 个文件</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent files section label -->
      <div class="drive-section" v-if="!isInFolder">
        <div class="section-head-row">
          <h2 class="section-head">全部文件</h2>
          <div class="section-actions">
            <el-button-group size="small">
              <el-button :type="viewMode === 'grid' ? 'primary' : ''" @click="viewMode = 'grid'">
                <el-icon><Grid /></el-icon>
              </el-button>
              <el-button :type="viewMode === 'list' ? 'primary' : ''" @click="viewMode = 'list'">
                <el-icon><List /></el-icon>
              </el-button>
            </el-button-group>
          </div>
        </div>
      </div>
    </template>

    <!-- File area -->
    <div class="drive-files" :class="{ 'has-welcome': !isInFolder && !isSearching }">
      <!-- Grid -->
      <div v-if="viewMode === 'grid'" class="file-grid">
        <article
          v-for="item in fileList" :key="item.id"
          class="file-card"
          :class="{ selected: selectedIds.includes(item.id) }"
          @dblclick="handleOpen(item)"
          @click.exact="onCardClick($event, item)"
          @contextmenu.prevent="showCtxMenu($event, item)"
        >
          <!-- Checkbox -->
          <div class="fc-check" :class="{ show: selectedIds.includes(item.id) || hoveredId === item.id }">
            <el-checkbox
              :model-value="selectedIds.includes(item.id)"
              @change="toggleSelect(item)"
              @click.stop
            />
          </div>
          <!-- More menu -->
          <div class="fc-menu" :class="{ show: hoveredId === item.id }">
            <el-dropdown trigger="click" @command="cmd => handleAction(item, cmd)" placement="bottom-end">
              <button class="fc-menu-btn" @click.stop>
                <el-icon :size="15"><MoreFilled /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="rename"><el-icon><Edit /></el-icon>重命名</el-dropdown-item>
                  <el-dropdown-item command="star"><el-icon><Star /></el-icon>收藏</el-dropdown-item>
                  <el-dropdown-item command="download" v-if="!item.is_dir"><el-icon><Download /></el-icon>下载</el-dropdown-item>
                  <el-dropdown-item command="preview" v-if="!item.is_dir"><el-icon><View /></el-icon>预览</el-dropdown-item>
                  <el-dropdown-item command="share"><el-icon><Share /></el-icon>分享</el-dropdown-item>
                  <el-dropdown-item command="move"><el-icon><FolderOpened /></el-icon>移动</el-dropdown-item>
                  <el-dropdown-item divided command="delete"><el-icon><Delete /></el-icon>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <!-- Thumbnail -->
          <div class="fc-thumb" :class="{ folder: item.is_dir }">
            <template v-if="item.is_dir">
              <el-icon :size="40"><Folder /></el-icon>
            </template>
            <template v-else-if="item.thumbnail_url">
              <img :src="item.thumbnail_url" :alt="item.name" />
            </template>
            <template v-else>
              <el-icon :size="40" :color="getFileIconColor(item.name, false)">
                <component :is="getFileIcon(item.name, false)" />
              </el-icon>
            </template>
          </div>

          <!-- Info -->
          <div class="fc-info">
            <div class="fc-name" :title="item.name">{{ item.name }}</div>
            <div class="fc-meta" @mouseenter="hoveredId = item.id" @mouseleave="hoveredId = null">
              <span>{{ item.size_str || '-' }}</span>
              <span class="fc-dot">·</span>
              <span>{{ formatTime(item.updated_at || item.created_at) }}</span>
            </div>
          </div>
        </article>
      </div>

      <!-- List -->
      <div v-else class="cb-table-wrap">
        <el-table :data="fileList" row-key="id" @row-dblclick="handleOpen" @selection-change="onSelectionChange">
          <el-table-column type="selection" width="44" />
          <el-table-column width="48">
            <template #default="{ row }">
              <el-icon :size="22" :color="getFileIconColor(row.name, row.is_dir)">
                <component :is="getFileIcon(row.name, row.is_dir)" />
              </el-icon>
            </template>
          </el-table-column>
          <el-table-column label="名称" min-width="300" show-overflow-tooltip>
            <template #default="{ row }"><span class="file-link">{{ row.name }}</span></template>
          </el-table-column>
          <el-table-column label="大小" width="100">
            <template #default="{ row }">{{ row.size_str || '-' }}</template>
          </el-table-column>
          <el-table-column label="修改日期" width="170">
            <template #default="{ row }">{{ row.updated_at || row.created_at }}</template>
          </el-table-column>
          <el-table-column label="操作" width="260" fixed="right">
            <template #default="{ row }">
              <div class="action-row">
                <el-button size="small" link @click="handleRename(row)"><el-icon><Edit /></el-icon>重命名</el-button>
                <el-button size="small" link @click="handleAction(row, 'download')" v-if="!row.is_dir"><el-icon><Download /></el-icon>下载</el-button>
                <el-button size="small" link @click="handleAction(row, 'preview')" v-if="!row.is_dir"><el-icon><View /></el-icon>预览</el-button>
                <el-dropdown @command="cmd => handleAction(row, cmd)">
                  <el-button size="small" link>更多<el-icon style="margin-left:2px;"><ArrowDown /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="share"><el-icon><Share /></el-icon>分享</el-dropdown-item>
                      <el-dropdown-item command="star"><el-icon><Star /></el-icon>收藏</el-dropdown-item>
                      <el-dropdown-item command="move"><el-icon><FolderOpened /></el-icon>移动</el-dropdown-item>
                      <el-dropdown-item divided command="delete"><el-icon><Delete /></el-icon>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Empty -->
      <div v-if="!loading && fileList.length === 0 && !isSearching" class="cb-empty-state">
        <div class="empty-icon"><el-icon :size="36"><Folder /></el-icon></div>
        <h3>此文件夹为空</h3>
        <p>拖拽文件到此处或点击上传按钮</p>
      </div>
    </div>

    <!-- Batch actions bar -->
    <transition name="slide-up">
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <span class="batch-count">已选择 {{ selectedIds.length }} 项</span>
        <div class="batch-actions">
          <el-button size="small" round @click="handleBatchAction('download')"><el-icon><Download /></el-icon>下载</el-button>
          <el-button size="small" round @click="handleBatchAction('move')"><el-icon><FolderOpened /></el-icon>移动</el-button>
          <el-button size="small" round @click="handleBatchAction('share')"><el-icon><Share /></el-icon>分享</el-button>
          <el-button size="small" type="danger" round @click="handleBatchAction('delete')"><el-icon><Delete /></el-icon>删除</el-button>
        </div>
        <el-button size="small" link @click="selectedIds = []"><el-icon><Close /></el-icon></el-button>
      </div>
    </transition>

    <!-- Context menu -->
    <div
      v-if="ctxMenu.visible"
      class="ctx-menu"
      :style="{ top: ctxMenu.y + 'px', left: ctxMenu.x + 'px' }"
    >
      <button v-for="a in ctxActions" :key="a.cmd" @click="runCtxAction(a.cmd)" :class="{ danger: a.danger }">
        <el-icon :size="14"><component :is="a.icon" /></el-icon>{{ a.label }}
      </button>
    </div>

    <!-- Dialogs (same as before) -->
    <el-dialog v-model="renameDialogVisible" title="重命名" width="400px">
      <el-input v-model="renameForm.name" placeholder="输入新名称" @keyup.enter="confirmRename" />
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmRename">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="deleteDialogVisible" title="确认删除" width="420px">
      <div style="text-align:center;padding:16px 0;">
        <div style="width:56px;height:56px;border-radius:50%;background:#FEF2F2;display:inline-flex;align-items:center;justify-content:center;margin-bottom:16px;">
          <el-icon :size="28" color="#EF4444"><Warning /></el-icon>
        </div>
        <p style="font-size:15px;font-weight:600;color:var(--cb-text);">确定删除 <strong>{{ deleteTarget.name }}</strong>？</p>
        <p style="font-size:13px;color:var(--cb-text-muted);margin-top:6px;">删除后可在回收站保留 7 天</p>
      </div>
      <template #footer>
        <el-button @click="deleteDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmDelete" :loading="deleting">确定删除</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="moveDialogVisible" title="移动到" width="480px">
      <p style="margin-bottom:14px;color:var(--cb-text-secondary);">将 <strong>{{ moveTarget.name }}</strong> 移动到：</p>
      <div style="border:1px solid var(--cb-border);border-radius:8px;max-height:260px;overflow:auto;padding:8px;">
        <el-tree :data="folderTree" node-key="id" :props="{ label: 'name', children: 'children' }"
          highlight-current :expand-on-click-node="false" @node-click="onFolderSelect" />
      </div>
      <template #footer>
        <el-button @click="moveDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmMove" :loading="moving" :disabled="!selectedFolder">移动</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="newFolderVisible" title="新建文件夹" width="400px">
      <el-input v-model="newFolderName" placeholder="文件夹名称" maxlength="50" show-word-limit @keyup.enter="confirmNewFolder" />
      <template #footer>
        <el-button @click="newFolderVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmNewFolder" :loading="creatingFolder">确定</el-button>
      </template>
    </el-dialog>

    <CreateShareDialog v-model="shareDialogVisible" :file-info="shareFileInfo" />

    <el-dialog v-model="previewVisible" :title="previewData?.name || '预览'" width="900px" top="5vh" destroy-on-close>
      <div v-loading="previewLoading" class="preview-body">
        <template v-if="previewData">
          <img v-if="previewData.preview_type === 'image'" :src="previewData.file_url" class="preview-img" />
          <video v-else-if="previewData.preview_type === 'video'" :src="previewData.file_url" controls class="preview-video" />
          <audio v-else-if="previewData.preview_type === 'audio'" :src="previewData.file_url" controls class="preview-audio" />
          <iframe v-else-if="previewData.preview_type === 'pdf'" :src="previewData.file_url" class="preview-frame" />
          <iframe v-else-if="previewData.preview_type === 'office'" :src="previewData.office_preview_url" class="preview-frame" />
          <iframe v-else-if="previewData.preview_type === 'text'" :src="previewData.file_url" class="preview-frame" />
          <div v-else class="preview-unsupported">此文件类型暂不支持在线预览</div>
        </template>
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button type="primary" @click="handleDownload(previewData)">下载</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useStore } from 'vuex'
import { ElMessage } from 'element-plus'
import {
  ArrowDown, ArrowLeft, Close, Delete, Download, Edit, Folder, FolderAdd, FolderOpened,
  Grid, HomeFilled, List, MoreFilled, Search, Share, Star, Upload, View, Warning
} from '@element-plus/icons-vue'
import { listFiles, createFolder, deleteFile, renameFile, previewFile, downloadFile, searchFiles, getFolderTree, moveFile } from '@/api/file'
import { addFavorite } from '@/api/favorite'
import { getFileIcon, getFileIconColor } from '@/utils/fileIcon'
import CreateShareDialog from '@/components/CreateShareDialog.vue'
import FileUploadDialog from '@/components/FileUploadDialog.vue'

const route = useRoute()
const store = useStore()

// ── State ──
const viewMode = ref('grid')
const fileList = ref([])
const total = ref(0)
const loading = ref(false)
const currentParentId = ref('')
const currentPath = ref([])
const pathIdStack = ref([])
const searchKeyword = ref('')
const isSearching = ref(false)
const hoveredId = ref(null)
const selectedIds = ref([])
let searchTimer = null

// ── Computed ──
const isInFolder = computed(() => pathIdStack.value.length > 1)
const quickFolders = computed(() => {
  if (isInFolder.value || isSearching.value) return []
  return fileList.value.filter(f => f.is_dir).slice(0, 6).map((f, i) => ({
    ...f,
    color: ['#F59E0B','#8B5CF6','#EC4899','#10B981','#2F6BFF','#F97316'][i % 6],
    fileCount: Math.floor(Math.random() * 50) + 1
  }))
})

const greetingEmoji = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return ' ☀️'
  if (h < 18) return ' 👋'
  return ' 🌙'
})
const greetingHint = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return '早上好！文件已为你整理好，随时开始工作。'
  if (h < 18) return '下午好！你的云端文件随身携带，随时访问。'
  return '晚上好！查看今天的工作成果，或为明天做准备。'
})

// ── Context menu ──
const ctxMenu = ref({ visible: false, x: 0, y: 0, item: null })
const ctxActions = [
  { cmd: 'rename', label: '重命名', icon: Edit },
  { cmd: 'star', label: '收藏', icon: Star },
  { cmd: 'download', label: '下载', icon: Download },
  { cmd: 'preview', label: '预览', icon: View },
  { cmd: 'share', label: '分享', icon: Share },
  { cmd: 'move', label: '移动', icon: FolderOpened },
  { cmd: 'delete', label: '删除', icon: Delete, danger: true },
]

function showCtxMenu(e, item) {
  ctxMenu.value = { visible: true, x: e.clientX, y: e.clientY, item }
}
function hideCtxMenu() { ctxMenu.value.visible = false }
function runCtxAction(cmd) {
  if (ctxMenu.value.item) handleAction(ctxMenu.value.item, cmd)
  hideCtxMenu()
}
function onDocumentClick() { hideCtxMenu() }
onMounted(() => document.addEventListener('click', onDocumentClick))
onUnmounted(() => document.removeEventListener('click', onDocumentClick))

// ── Navigation ──
function navigateTo(f) { handleOpen(f) }

// ── Dialogs ──
const renameDialogVisible = ref(false)
const renameForm = ref({ id: '', name: '' })
const deleteDialogVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)
const moveDialogVisible = ref(false)
const moveTarget = ref({})
const folderTree = ref([])
const selectedFolder = ref(null)
const moving = ref(false)
const newFolderVisible = ref(false)
const newFolderName = ref('')
const creatingFolder = ref(false)
const shareDialogVisible = ref(false)
const shareFileInfo = ref({})
const uploadDialogVisible = ref(false)
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewData = ref(null)

// ── Load files ──
async function loadFiles() {
  loading.value = true
  try {
    const res = await listFiles({ parentId: currentParentId.value })
    fileList.value = res.list || []
    total.value = res.total || 0
  } catch { ElMessage.error('加载文件列表失败') }
  finally { loading.value = false }
}

function handleOpen(item) {
  if (item.is_dir) {
    searchKeyword.value = ''; isSearching.value = false
    currentParentId.value = item.id
    currentPath.value = [...currentPath.value, item.name]
    pathIdStack.value = [...pathIdStack.value, item.id]
    loadFiles()
  } else { handlePreview(item) }
}

function goRoot() {
  clearSearch(); currentParentId.value = store.state.userInfo?.rootFolderId || ''
  currentPath.value = []; pathIdStack.value = [currentParentId.value]; loadFiles()
}
function goToBreadcrumb(idx) {
  clearSearch(); currentPath.value = currentPath.value.slice(0, idx + 1)
  pathIdStack.value = pathIdStack.value.slice(0, idx + 2)
  currentParentId.value = pathIdStack.value[pathIdStack.value.length - 1]; loadFiles()
}
function goBack() {
  if (pathIdStack.value.length <= 1) return
  clearSearch(); pathIdStack.value.pop(); currentPath.value.pop()
  currentParentId.value = pathIdStack.value[pathIdStack.value.length - 1]; loadFiles()
}

// ── Select ──
function onCardClick(e, item) {
  if (e.ctrlKey || e.metaKey) toggleSelect(item)
  else if (e.shiftKey && selectedIds.value.length > 0) {
    const last = fileList.value.findIndex(f => f.id === selectedIds.value.at(-1))
    const cur = fileList.value.findIndex(f => f.id === item.id)
    const range = fileList.value.slice(Math.min(last, cur), Math.max(last, cur) + 1)
    selectedIds.value = [...new Set([...selectedIds.value, ...range.map(f => f.id)])]
  }
}
function toggleSelect(item) {
  const idx = selectedIds.value.indexOf(item.id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(item.id)
}
function onSelectionChange(sel) { selectedIds.value = sel.map(s => s.id) }

// ── Actions ──
function handleAction(item, cmd) {
  const m = {
    rename: () => { renameForm.value = { id: item.id, name: item.name }; renameDialogVisible.value = true },
    download: () => handleDownload(item),
    preview: () => handlePreview(item),
    share: () => {
      if (item.is_dir) { ElMessage.warning('暂不支持分享文件夹'); return }
      shareFileInfo.value = { id: item.id, name: item.name, size: item.size, fileType: getType(item.extension) }
      shareDialogVisible.value = true
    },
    star: () => { addFavorite(item.id).then(() => ElMessage.success('已收藏')).catch(() => ElMessage.error('收藏失败')) },
    move: () => handleMove(item),
    delete: () => { deleteTarget.value = item; deleteDialogVisible.value = true }
  }
  if (m[cmd]) m[cmd]()
}

function handleRename(row) { renameForm.value = { id: row.id, name: row.name }; renameDialogVisible.value = true }

async function confirmRename() {
  if (!renameForm.value.name.trim()) return
  try { await renameFile(renameForm.value.id, renameForm.value.name.trim()); ElMessage.success('已重命名'); renameDialogVisible.value = false; loadFiles() }
  catch { ElMessage.error('重命名失败') }
}

async function handlePreview(item) {
  if (!item || item.is_dir) return
  previewLoading.value = true; previewVisible.value = true; previewData.value = null
  try { const d = await previewFile(item.id); previewData.value = d; if (!d.can_preview) ElMessage.warning('不支持在线预览') }
  catch { previewVisible.value = false; ElMessage.error('预览失败') }
  finally { previewLoading.value = false }
}

async function handleDownload(item) {
  try {
    const blob = await downloadFile(item.id)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = item.name; a.click()
    URL.revokeObjectURL(url)
  } catch { ElMessage.error('下载失败') }
}

async function confirmDelete() {
  deleting.value = true
  try { await deleteFile(deleteTarget.value.id); ElMessage.success('已移至回收站'); deleteDialogVisible.value = false; loadFiles() }
  catch { ElMessage.error('删除失败') }
  finally { deleting.value = false }
}

async function handleMove(item) {
  moveTarget.value = item; moveDialogVisible.value = true; selectedFolder.value = null
  try {
    const res = await getFolderTree()
    folderTree.value = (res.list || []).map(n => ({ ...n, disabled: n.id === item.id || n.id === (item.parent_id || currentParentId.value) }))
  } catch { ElMessage.error('加载文件夹失败') }
}
function onFolderSelect(node) { if (!node.disabled) selectedFolder.value = node }
async function confirmMove() {
  if (!selectedFolder.value) return; moving.value = true
  try { await moveFile({ fileId: moveTarget.value.id, targetFolderId: selectedFolder.value.id }); ElMessage.success('已移动'); moveDialogVisible.value = false; loadFiles() }
  catch { ElMessage.error('移动失败') }
  finally { moving.value = false }
}

function handleNewFolder() { newFolderName.value = ''; newFolderVisible.value = true }
async function confirmNewFolder() {
  if (!newFolderName.value.trim()) { ElMessage.warning('请输入名称'); return }
  creatingFolder.value = true
  try { await createFolder({ name: newFolderName.value.trim(), parentId: currentParentId.value }); ElMessage.success('已创建'); newFolderVisible.value = false; loadFiles() }
  catch { ElMessage.error('创建失败') }
  finally { creatingFolder.value = false }
}

function handleBatchAction(cmd) {
  if (cmd === 'delete') {
    deleteTarget.value = { name: `${selectedIds.value.length} 个文件` }
    deleteDialogVisible.value = true
  }
}

// ── Search ──
function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    const kw = searchKeyword.value.trim()
    if (!kw) { clearSearch(); loadFiles(); return }
    performSearch(kw)
  }, 300)
}
async function performSearch(kw) {
  isSearching.value = true
  try { const res = await searchFiles({ keyword: kw, parentId: currentParentId.value, page: 1, pageSize: 100 }); fileList.value = res.list || []; total.value = fileList.value.length }
  catch { ElMessage.error('搜索失败') }
}
function clearSearch() { searchKeyword.value = ''; isSearching.value = false }
function handleUploadSuccess() { loadFiles() }

// ── Helpers ──
function getType(ext) {
  if (!ext) return 'other'
  const e = ext.toLowerCase()
  if (['jpg','jpeg','png','gif','bmp','webp','svg'].includes(e)) return 'image'
  if (['mp4','avi','mov','wmv','flv','mkv','webm'].includes(e)) return 'video'
  if (['mp3','wav','flac','aac','ogg'].includes(e)) return 'audio'
  if (['pdf','doc','docx','xls','xlsx','ppt','pptx','txt'].includes(e)) return 'document'
  return 'other'
}
function formatTime(d) {
  if (!d) return ''
  const now = Date.now(), t = new Date(d).getTime(), diff = now - t
  const min = Math.floor(diff / 6e4)
  if (min < 1) return '刚刚'
  if (min < 60) return min + '分钟前'
  const hrs = Math.floor(min / 60)
  if (hrs < 24) return hrs + '小时前'
  return new Date(d).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

// ── Init ──
onMounted(() => {
  const rootId = store.state.userInfo?.rootFolderId || ''
  if (!rootId) { ElMessage.error('用户数据加载中，请刷新'); return }
  currentParentId.value = rootId; pathIdStack.value = [rootId]

  const sq = route.query.search
  if (sq) { searchKeyword.value = sq; performSearch(sq); return }
  const pid = route.query.parentId
  if (pid) { currentParentId.value = pid; pathIdStack.value = [pid] }
  loadFiles()
})

watch(() => store.state.file.needRefresh, val => {
  if (val) { loadFiles(); store.commit('file/setNeedRefresh', false) }
})
</script>

<style scoped>
.drive {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--cb-bg);
}

/* ── Bar ── */
.drive-bar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 12px 28px;
  background: var(--cb-surface);
  border-bottom: 1px solid var(--cb-border-light);
  flex-shrink: 0;
}
.drive-bar-left { display: flex; align-items: center; gap: 10px; }
.drive-count {
  font-size: 12px; color: var(--cb-text-muted); font-weight: 600;
  background: var(--cb-bg-alt); padding: 2px 8px; border-radius: 99px;
}
.drive-bar-right { flex-shrink: 0; }

/* ── Search banner ── */
.drive-search-banner {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 28px;
  background: var(--cb-primary-light);
  font-size: 13px; color: var(--cb-primary); font-weight: 600;
  border-bottom: 1px solid var(--cb-primary-soft);
}

/* ── Welcome ── */
.drive-welcome {
  padding: 32px 28px 24px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
}
.welcome-greeting h1 {
  font-size: 26px;
  font-weight: 800;
  color: var(--cb-text);
  letter-spacing: -0.5px;
  margin: 0 0 8px;
}
.welcome-greeting p {
  font-size: 14px;
  color: var(--cb-text-muted);
  margin: 0;
}
.welcome-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

/* ── Section ── */
.drive-section { padding: 8px 28px 0; }
.section-head {
  font-size: 16px;
  font-weight: 700;
  color: var(--cb-text);
  letter-spacing: -0.2px;
  margin: 0 0 14px;
}
.section-head-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.section-head-row .section-head { margin-bottom: 0; }

/* ── Quick Access ── */
.quick-scroll {
  display: flex;
  gap: 12px;
  overflow-x: auto;
  padding-bottom: 4px;
}
.quick-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius);
  background: var(--cb-surface);
  cursor: pointer;
  flex-shrink: 0;
  min-width: 190px;
  transition: all var(--cb-transition-fast);
}
.quick-card:hover {
  border-color: var(--cb-border-strong);
  transform: translateY(-2px);
  box-shadow: var(--cb-shadow-sm);
}
.qc-icon {
  width: 42px; height: 42px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--qc-color) 12%, transparent);
  color: var(--qc-color);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.qc-info strong { display: block; font-size: 13px; font-weight: 600; color: var(--cb-text); margin-bottom: 2px; }
.qc-info span { font-size: 11px; color: var(--cb-text-muted); }

/* ── Files ── */
.drive-files { flex: 1; overflow: auto; padding: 16px 28px 32px; }
.drive-files.has-welcome { padding-top: 8px; }

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 12px;
}

/* ── File card ── */
.file-card {
  position: relative;
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius);
  background: var(--cb-surface);
  cursor: pointer;
  transition: all var(--cb-transition-fast);
  overflow: hidden;
}
.file-card:hover {
  border-color: var(--cb-border-strong);
  transform: translateY(-2px);
  box-shadow: var(--cb-shadow);
}
.file-card.selected {
  border-color: var(--cb-primary);
  background: var(--cb-primary-light);
  box-shadow: 0 0 0 2px rgba(47,107,255,.12);
}

/* Check & menu overlays */
.fc-check, .fc-menu {
  position: absolute; top: 8px; z-index: 3;
  opacity: 0; transition: opacity .15s;
}
.fc-check { left: 8px; }
.fc-menu { right: 8px; }
.fc-check.show, .fc-menu.show { opacity: 1; }
.fc-menu-btn {
  width: 28px; height: 28px;
  border: 0; border-radius: 6px;
  background: var(--cb-surface);
  color: var(--cb-text-secondary);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  box-shadow: var(--cb-shadow-xs);
  transition: all var(--cb-transition-fast);
}
.fc-menu-btn:hover { background: var(--cb-bg-alt); color: var(--cb-text); }

/* Thumb */
.fc-thumb {
  height: 124px;
  display: flex; align-items: center; justify-content: center;
  background: var(--cb-bg-alt);
  border-bottom: 1px solid var(--cb-border-light);
}
.fc-thumb.folder { background: #FFFBF0; }
.fc-thumb img { width: 100%; height: 100%; object-fit: cover; }
.fc-thumb .el-icon { transition: transform .2s var(--cb-ease); }
.file-card:hover .fc-thumb .el-icon { transform: scale(1.06); }

/* Info */
.fc-info { padding: 12px 14px; }
.fc-name {
  font-size: 13px; font-weight: 600; color: var(--cb-text);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  margin-bottom: 4px;
}
.fc-meta { display: flex; gap: 4px; font-size: 11px; color: var(--cb-text-muted); }
.fc-dot { font-weight: 700; }

/* ── Batch bar ── */
.batch-bar {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;
  display: flex; align-items: center; gap: 12px;
  padding: 10px 16px;
  background: var(--cb-text);
  border-radius: 99px;
  box-shadow: var(--cb-shadow-xl);
}
.batch-count {
  font-size: 13px; font-weight: 600; color: #fff;
  white-space: nowrap;
}
.batch-actions { display: flex; gap: 6px; }
.batch-bar .el-button--small { background: rgba(255,255,255,.15); color: #fff; border: 0; }
.batch-bar .el-button--small:hover { background: rgba(255,255,255,.22); }
.batch-bar .el-button--danger { background: rgba(239,68,68,.4); }
.batch-bar .el-button--link { color: rgba(255,255,255,.6); }

.slide-up-enter-active, .slide-up-leave-active { transition: all .25s var(--cb-ease); }
.slide-up-enter-from, .slide-up-leave-to { opacity: 0; transform: translateX(-50%) translateY(12px); }

/* ── Context menu ── */
.ctx-menu {
  position: fixed;
  z-index: 2000;
  min-width: 180px;
  padding: 6px;
  background: var(--cb-surface);
  border: 1px solid var(--cb-border);
  border-radius: var(--cb-radius);
  box-shadow: var(--cb-shadow-lg);
}
.ctx-menu button {
  width: 100%;
  display: flex; align-items: center; gap: 10px;
  padding: 9px 12px;
  border: 0; border-radius: 6px;
  background: transparent;
  font-size: 13px; font-weight: 500; color: var(--cb-text-secondary);
  cursor: pointer;
  transition: all var(--cb-transition-fast);
}
.ctx-menu button:hover { background: var(--cb-bg-alt); color: var(--cb-text); }
.ctx-menu button.danger { color: var(--cb-danger); }
.ctx-menu button.danger:hover { background: var(--cb-danger-light); }

/* ── List ── */
.file-link { font-weight: 600; color: var(--cb-text); cursor: pointer; }
.file-link:hover { color: var(--cb-primary); }
.action-row { display: flex; align-items: center; }

/* ── Previews ── */
.preview-body { min-height: 280px; }
.preview-img { width: 100%; max-height: 520px; object-fit: contain; display: block; }
.preview-video { width: 100%; max-height: 520px; background: #000; }
.preview-audio { width: 100%; margin: 16px 0; }
.preview-frame { width: 100%; height: 520px; border: 0; }
.preview-unsupported { text-align: center; padding: 60px 0; color: var(--cb-text-mute); }
</style>
