<template>
  <div class="drive">
    <header class="workspace-header">
      <div class="workspace-heading">
        <div class="workspace-title-row">
          <el-button v-show="pathIdStack.length > 1" :icon="ArrowLeft" circle size="small" title="返回上一级" @click="goBack" />
          <h1 v-if="!isInFolder">我的文件</h1>
          <el-breadcrumb v-else separator="/">
            <el-breadcrumb-item @click="goRoot"><el-icon :size="14"><HomeFilled /></el-icon> 我的文件</el-breadcrumb-item>
            <el-breadcrumb-item v-for="(name, idx) in currentPath" :key="idx" @click="goToBreadcrumb(idx)">
              {{ name }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <p>{{ isSearching ? `正在搜索“${searchKeyword}”` : `${total} 项内容` }}</p>
      </div>
      <div class="workspace-primary-actions">
        <el-button :icon="FolderAdd" @click="handleNewFolder">新建文件夹</el-button>
        <el-button type="primary" :icon="Upload" @click="uploadDialogVisible = true">上传文件</el-button>
      </div>
    </header>

    <div class="workspace-toolbar">
      <div class="workspace-toolbar-left">
        <strong>{{ isSearching ? '搜索结果' : (isInFolder ? currentPath[currentPath.length - 1] : '全部文件') }}</strong>
        <span v-if="!isSearching && fileList.length > 0" class="drive-sort-inline">
          <el-radio-group v-model="sortBy" size="small" @change="loadFiles">
            <el-radio-button value="created_at">时间</el-radio-button>
            <el-radio-button value="name">名称</el-radio-button>
            <el-radio-button value="size">大小</el-radio-button>
          </el-radio-group>
          <el-button size="small" link :icon="sortOrder === 'asc' ? SortUp : SortDown" :title="sortOrder === 'asc' ? '升序' : '降序'" @click="toggleSortOrder" />
        </span>
      </div>
      <el-button class="view-toggle-btn" size="small" @click="viewMode = viewMode === 'grid' ? 'list' : 'grid'" :title="viewMode === 'grid' ? '切换列表视图' : '切换网格视图'">
        <el-icon :size="16"><component :is="viewMode === 'grid' ? List : Grid" /></el-icon>
      </el-button>
    </div>

    <FileUploadDialog v-model="uploadDialogVisible" :parent-id="currentParentId" @success="handleUploadSuccess" />

    <!-- Search banner -->
    <div v-if="isSearching" class="drive-search-banner">
      搜索 "{{ searchKeyword }}" 找到 {{ fileList.length }} 个结果
      <el-button type="primary" link size="small" @click="clearSearch(); loadFiles()">清除搜索</el-button>
    </div>

    <div class="drive-workspace">
      <!-- File area -->
      <div class="drive-files">
      <!-- Grid -->
      <div v-if="viewMode === 'grid'" class="file-grid">
        <article
          v-for="item in fileList" :key="item.id"
          class="file-card"
          :class="{ selected: selectedIds.includes(item.id), 'drag-over': dragOverId === item.id && item.is_dir, 'dragging': dragId === item.id }"
          :draggable="!item.is_dir"
          @dblclick="handleOpen(item)"
          @click.exact="onCardClick($event, item)"
          @contextmenu.prevent="showCtxMenu($event, item)"
          @dragstart="onDragStart($event, item)"
          @dragover.prevent="item.is_dir && onDragOver($event, item)"
          @dragleave="onDragLeave(item)"
          @drop="item.is_dir && onDrop(item)"
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
        <el-table :data="fileList" row-key="id" @row-click="onTableRowClick" @row-dblclick="handleOpen" @selection-change="onSelectionChange">
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

      <!-- Pagination -->
      <div v-if="total > pageSize && !isSearching" class="drive-pagination">
        <el-pagination
          background layout="prev, pager, next, total"
          :total="total" :page-size="pageSize" v-model:current-page="currentPage"
          @current-change="loadFiles"
        />
      </div>

      <!-- Empty -->
      <EmptyState v-if="!loading && fileList.length === 0 && !isSearching" :icon="Folder" title="此文件夹为空" description="拖拽文件到此处或点击上传按钮" />
      </div>

      <aside v-if="selectedFile" class="file-details" aria-label="文件详情">
        <div class="details-head">
          <strong>详情</strong>
          <button type="button" title="关闭详情" @click="selectedIds = []"><el-icon><Close /></el-icon></button>
        </div>
        <div class="details-preview" :class="{ folder: selectedFile.is_dir }">
          <img v-if="!selectedFile.is_dir && selectedFile.thumbnail_url" :src="selectedFile.thumbnail_url" :alt="selectedFile.name" />
          <el-icon v-else :size="52" :color="getFileIconColor(selectedFile.name, selectedFile.is_dir)">
            <component :is="getFileIcon(selectedFile.name, selectedFile.is_dir)" />
          </el-icon>
        </div>
        <h2 :title="selectedFile.name">{{ selectedFile.name }}</h2>
        <div class="details-actions">
          <el-button v-if="!selectedFile.is_dir" type="primary" :icon="View" @click="handleAction(selectedFile, 'preview')">预览</el-button>
          <el-button v-if="!selectedFile.is_dir" :icon="Download" @click="handleAction(selectedFile, 'download')">下载</el-button>
          <el-button v-if="!selectedFile.is_dir" :icon="Share" @click="handleAction(selectedFile, 'share')">分享</el-button>
        </div>
        <dl class="details-meta">
          <div><dt>类型</dt><dd>{{ selectedFile.is_dir ? '文件夹' : selectedFileType }}</dd></div>
          <div><dt>大小</dt><dd>{{ selectedFile.size_str || '-' }}</dd></div>
          <div><dt>修改时间</dt><dd>{{ formatTime(selectedFile.updated_at || selectedFile.created_at) }}</dd></div>
          <div><dt>所在位置</dt><dd>{{ currentPath.length ? `我的文件 / ${currentPath.join(' / ')}` : '我的文件' }}</dd></div>
        </dl>
      </aside>
    </div>

    <!-- Batch actions bar -->
    <transition name="slide-up">
      <div v-if="selectedIds.length > 1" class="batch-bar">
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
        <p style="font-size:15px;font-weight:600;color:var(--cb-text);">确定删除 <strong>{{ deleteTarget.name || deleteTarget.ids?.length + ' 个文件' }}</strong>？</p>
        <p style="font-size:13px;color:var(--cb-text-muted);margin-top:6px;">删除后可在回收站保留 7 天</p>
      </div>
      <template #footer>
        <el-button @click="deleteDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmDelete" :loading="deleting">确定删除</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="moveDialogVisible" title="移动到" width="480px">
      <p style="margin-bottom:14px;color:var(--cb-text-secondary);">将 <strong>{{ moveTarget.ids ? moveTarget.ids.length + ' 个文件' : moveTarget.name }}</strong> 移动到：</p>
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

    <!-- Shortcuts help -->
    <el-dialog v-model="shortcutsHelpVisible" title="键盘快捷键" width="440px">
      <div class="shortcuts-grid">
        <div class="sc-item"><kbd>?</kbd><span>显示快捷键帮助</span></div>
        <div class="sc-item"><kbd>Ctrl</kbd>+<kbd>U</kbd><span>上传文件</span></div>
        <div class="sc-item"><kbd>Ctrl</kbd>+<kbd>F</kbd><span>聚焦搜索框</span></div>
        <div class="sc-item"><kbd>Ctrl</kbd>+<kbd>A</kbd><span>全选文件</span></div>
        <div class="sc-item"><kbd>Delete</kbd><span>删除选中文件</span></div>
        <div class="sc-item"><kbd>F2</kbd><span>重命名选中文件</span></div>
        <div class="sc-item"><kbd>Enter</kbd><span>打开文件/文件夹</span></div>
        <div class="sc-item"><kbd>Esc</kbd><span>取消选择</span></div>
        <div class="sc-item"><kbd>←</kbd><span>返回上级目录</span></div>
      </div>
      <p style="font-size:12px;color:var(--cb-text-muted);margin-top:12px;">macOS 上使用 Cmd 代替 Ctrl</p>
      <template #footer>
        <el-button @click="shortcutsHelpVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="previewVisible" :title="previewTitle" width="900px" top="5vh" destroy-on-close>
      <div v-loading="previewLoading" class="preview-body">
        <template v-if="previewData">
          <div v-if="previewData.preview_type === 'image'" class="preview-img-wrap">
            <button v-if="imageNav.hasPrev" class="preview-nav preview-nav-prev" @click="navImage(-1)"><el-icon :size="28"><ArrowLeft /></el-icon></button>
            <img :src="previewData.file_url" class="preview-img" />
            <button v-if="imageNav.hasNext" class="preview-nav preview-nav-next" @click="navImage(1)"><el-icon :size="28"><ArrowRight /></el-icon></button>
          </div>
          <video v-else-if="previewData.preview_type === 'video'" :src="previewData.file_url" controls class="preview-video" />
          <audio v-else-if="previewData.preview_type === 'audio'" :src="previewData.file_url" controls class="preview-audio" />
          <iframe v-else-if="previewData.preview_type === 'pdf'" :src="previewData.file_url" class="preview-frame" />
          <iframe v-else-if="previewData.preview_type === 'office'" :src="previewData.office_preview_url" class="preview-frame" />
          <iframe v-else-if="previewData.preview_type === 'text'" :src="previewData.file_url" class="preview-frame" />
          <div v-else-if="previewData.preview_type === 'markdown'" class="preview-markdown" v-html="markdownHtml"></div>
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
  ArrowDown, ArrowLeft, ArrowRight, Close, Delete, Download, Edit, Folder, FolderAdd, FolderOpened,
  Grid, HomeFilled, List, MoreFilled, Search, Share, SortDown, SortUp, Star, Upload, View, Warning
} from '@element-plus/icons-vue'
import { listFiles, createFolder, deleteFile, renameFile, previewFile, searchFiles, getFolderTree, moveFile, createBatchDownload } from '@/api/file'
import { addFavorite } from '@/api/favorite'
import { getFileIcon, getFileIconColor } from '@/utils/fileIcon'
import CreateShareDialog from '@/components/CreateShareDialog.vue'
import FileUploadDialog from '@/components/FileUploadDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useFileActions } from '@/composables/useFileActions'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const { download: doDownload, getFileType, isImage, formatTime } = useFileActions()

const route = useRoute()
const store = useStore()

// ── State ──
const viewMode = ref(localStorage.getItem('driveViewMode') || 'grid')
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
const sortBy = ref('created_at')
const sortOrder = ref('desc')
const currentPage = ref(1)
const pageSize = ref(60)
const dragId = ref(null)
const dragOverId = ref(null)
let searchTimer = null

// ── Computed ──
const isInFolder = computed(() => pathIdStack.value.length > 1)
const selectedFile = computed(() => {
  if (selectedIds.value.length !== 1) return null
  return fileList.value.find(file => file.id === selectedIds.value[0]) || null
})
const selectedFileType = computed(() => {
  if (!selectedFile.value) return '-'
  const extension = selectedFile.value.file_extension || selectedFile.value.name.split('.').pop()
  return {
    image: '图片', video: '视频', audio: '音频', document: '文档', other: '其他文件'
  }[getFileType(extension)]
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

function onDragStart(e, item) {
  if (item.is_dir) return
  dragId.value = item.id
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', item.id)
}

function onDragOver(e, item) {
  if (!item.is_dir) return
  e.dataTransfer.dropEffect = 'move'
  dragOverId.value = item.id
}

function onDragLeave(item) {
  if (dragOverId.value === item.id) dragOverId.value = null
}

async function onDrop(folder) {
  dragOverId.value = null
  const fileId = dragId.value
  dragId.value = null
  if (!fileId || fileId === folder.id) return
  try {
    await moveFile({ fileId, targetFolderId: folder.id })
    ElMessage.success(`已移动到 ${folder.name}`)
    loadFiles()
  } catch { ElMessage.error('移动失败') }
}

function onKeydown(e) {
  // 忽略在 input/textarea 中的按键
  if (['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)) return
  if (e.key === 'Delete' && selectedIds.value.length > 0) {
    e.preventDefault()
    handleBatchAction('delete')
  } else if (e.key === 'F2' && selectedIds.value.length === 1) {
    e.preventDefault()
    const item = fileList.value.find(f => f.id === selectedIds.value[0])
    if (item) handleRename(item)
  } else if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
    e.preventDefault()
    selectedIds.value = fileList.value.map(f => f.id)
  } else if ((e.ctrlKey || e.metaKey) && e.key === 'u') {
    e.preventDefault()
    uploadDialogVisible.value = true
  } else if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
    e.preventDefault()
    document.querySelector('.tb-search-input')?.focus()
  } else if (e.key === 'Escape') {
    selectedIds.value = []
    hideCtxMenu()
  } else if (e.key === 'Enter' && selectedIds.value.length === 1) {
    const item = fileList.value.find(f => f.id === selectedIds.value[0])
    if (item) handleOpen(item)
  } else if (e.key === '?' && !e.ctrlKey && !e.metaKey) {
    e.preventDefault()
    shortcutsHelpVisible.value = true
  }
}

onMounted(() => document.addEventListener('click', onDocumentClick))
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('click', onDocumentClick))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

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
const shortcutsHelpVisible = ref(false)
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewData = ref(null)
const previewIndex = ref(-1)
const markdownHtml = ref('')
const imageNav = computed(() => {
  const images = fileList.value.filter(f => !f.is_dir && isImage(f.name || f.extension || ''))
  const idx = previewIndex.value
  return {
    images,
    total: images.length,
    hasPrev: idx > 0,
    hasNext: idx < images.length - 1,
    current: idx + 1
  }
})
const previewTitle = computed(() => {
  if (!previewData.value) return '预览'
  const idx = imageNav.value.current
  const total = imageNav.value.total
  return total > 1 ? `${previewData.value.name} (${idx}/${total})` : previewData.value.name
})

// ── Load files ──
async function loadFiles() {
  loading.value = true
	try {
		const res = await listFiles({ parentId: currentParentId.value, sortBy: sortBy.value, sortOrder: sortOrder.value, page: currentPage.value, pageSize: pageSize.value })
		fileList.value = res.list || []
		total.value = res.total || 0
		selectedIds.value = []
  } catch { ElMessage.error('加载文件列表失败') }
  finally { loading.value = false }
}

function handleOpen(item) {
  if (item.is_dir) {
    searchKeyword.value = ''; isSearching.value = false
    currentParentId.value = item.id
    currentPath.value = [...currentPath.value, item.name]
    pathIdStack.value = [...pathIdStack.value, item.id]
    currentPage.value = 1
    loadFiles()
  } else { handlePreview(item) }
}

function toggleSortOrder() {
  sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  loadFiles()
}

function goRoot() {
  clearSearch(); currentPage.value = 1; currentParentId.value = store.state.userInfo?.rootFolderId || ''
  currentPath.value = []; pathIdStack.value = [currentParentId.value]; loadFiles()
}
function goToBreadcrumb(idx) {
  clearSearch(); currentPage.value = 1; currentPath.value = currentPath.value.slice(0, idx + 1)
  pathIdStack.value = pathIdStack.value.slice(0, idx + 2)
  currentParentId.value = pathIdStack.value[pathIdStack.value.length - 1]; loadFiles()
}
function goBack() {
  if (pathIdStack.value.length <= 1) return
  clearSearch(); currentPage.value = 1; pathIdStack.value.pop(); currentPath.value.pop()
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
	} else selectedIds.value = [item.id]
}
function toggleSelect(item) {
  const idx = selectedIds.value.indexOf(item.id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(item.id)
}
function onSelectionChange(sel) { selectedIds.value = sel.map(s => s.id) }
function onTableRowClick(row, column) {
  if (column?.type !== 'selection') selectedIds.value = [row.id]
}

// ── Actions ──
function handleAction(item, cmd) {
  const m = {
    rename: () => { renameForm.value = { id: item.id, name: item.name }; renameDialogVisible.value = true },
    download: () => handleDownload(item),
    preview: () => handlePreview(item),
    share: () => {
      if (item.is_dir) { ElMessage.warning('暂不支持分享文件夹'); return }
      const extension = item.file_extension || item.name.split('.').pop()
      shareFileInfo.value = { id: item.id, name: item.name, size: item.size, fileType: getFileType(extension) }
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


async function handlePreview(item, idx) {
  if (!item || item.is_dir) return
  loading.value = true

  previewLoading.value = true; previewVisible.value = true; previewData.value = null; markdownHtml.value = ''

  // Track image index for carousel
  const images = fileList.value.filter(f => !f.is_dir && isImage(f.name || f.extension || ''))
  previewIndex.value = idx !== undefined ? idx : images.findIndex(f => f.id === item.id)

  try {
    const d = await previewFile(item.id); previewData.value = d
    if (!d.can_preview) ElMessage.warning('不支持在线预览')
    if (d.preview_type === 'markdown' && d.file_url) {
      const res = await fetch(d.file_url)
      const text = await res.text()
      markdownHtml.value = DOMPurify.sanitize(marked(text), {
        USE_PROFILES: { html: true },
        FORBID_TAGS: ['style', 'form'],
        FORBID_ATTR: ['style']
      })
    }
  } catch { previewVisible.value = false; ElMessage.error('预览失败') }
  finally { previewLoading.value = false; loading.value = false }
}

async function navImage(dir) {
  const images = imageNav.value.images
  const newIdx = previewIndex.value + dir
  if (newIdx < 0 || newIdx >= images.length) return
  const item = images[newIdx]
  await handlePreview(item, newIdx)
}

function handleDownload(item) { doDownload(item) }

async function confirmDelete() {
  deleting.value = true
  try {
    if (deleteTarget.value.ids) {
      // Batch delete
      for (const fid of deleteTarget.value.ids) {
        try { await deleteFile(fid) } catch (e) { console.error('Failed to delete file', fid, e) }
      }
      ElMessage.success(`已删除 ${deleteTarget.value.ids.length} 个文件`)
      selectedIds.value = []
    } else {
      await deleteFile(deleteTarget.value.id)
      ElMessage.success('已移至回收站')
    }
    deleteDialogVisible.value = false; loadFiles()
  }
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
  try {
    // Batch move: iterate selected files
    if (moveTarget.value.ids) {
      for (const fid of moveTarget.value.ids) {
        try { await moveFile({ fileId: fid, targetFolderId: selectedFolder.value.id }) } catch (e) { console.error('Failed to move file', fid, e) }
      }
      ElMessage.success(`已移动 ${moveTarget.value.ids.length} 个文件`)
      selectedIds.value = []
    } else {
      await moveFile({ fileId: moveTarget.value.id, targetFolderId: selectedFolder.value.id })
      ElMessage.success('已移动')
    }
    moveDialogVisible.value = false; loadFiles()
  }
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

async function handleBatchAction(cmd) {
  if (selectedIds.value.length === 0) return
  if (cmd === 'download') {
    await batchDownloadHandler()
  } else if (cmd === 'delete') {
    deleteTarget.value = { name: `${selectedIds.value.length} 个文件`, ids: selectedIds.value.slice() }
    deleteDialogVisible.value = true
  } else if (cmd === 'move') {
    batchMoveHandler()
  } else if (cmd === 'share') {
    ElMessage.info('暂不支持批量分享，请逐个文件分享')
  }
}

async function batchDownloadHandler() {
  const ids = selectedIds.value.filter(id => {
    const f = fileList.value.find(x => x.id === id)
    return f && !f.is_dir
  })
  if (ids.length === 0) {
    ElMessage.warning('请选择文件（不支持下载文件夹）')
    return
  }
  try {
    const task = await createBatchDownload(ids)
    if (!task?.downloadUrl) {
      throw new Error('missing download url')
    }
    const a = document.createElement('a')
    a.href = task.downloadUrl
    a.download = task.fileName || 'files.zip'
    a.rel = 'noopener'
    a.click()
    ElMessage.success('批量下载已开始')
  } catch {
    ElMessage.error('批量下载失败')
  }
}

function batchMoveHandler() {
  if (selectedIds.value.length === 0) return
  moveTarget.value = { name: `${selectedIds.value.length} 个文件`, id: selectedIds.value[0] }
  moveDialogVisible.value = true
  selectedFolder.value = null
  getFolderTree().then(res => {
    folderTree.value = (res.list || []).map(n => ({
      ...n,
      disabled: selectedIds.value.includes(n.id)
    }))
  }).catch(() => ElMessage.error('加载文件夹失败'))
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
function handleUploadSuccess() { loadFiles(); store.commit('file/setNeedRefreshStorage', true) }

// ── Helpers ──

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

// 监听路由搜索参数变化（Header 搜索时已在本页面不会重新 mounted）
watch(() => route.query.search, (kw) => {
  if (kw) { searchKeyword.value = kw; performSearch(kw) }
  else { clearSearch(); loadFiles() }
})

// 视图模式持久化
watch(viewMode, val => { localStorage.setItem('driveViewMode', val) })

// 同步当前文件夹到 store，Header 上传按钮使用
watch(currentParentId, val => { store.commit('file/setCurrentParentId', val) })

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

/* ── Desktop workspace header ── */
.workspace-header {
  min-height: 84px;
  padding: 18px 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  flex-shrink: 0;
  background: var(--cb-surface);
  border-bottom: 1px solid var(--cb-border-light);
}
.workspace-heading { min-width: 0; }
.workspace-title-row { display: flex; align-items: center; gap: 10px; min-height: 30px; }
.workspace-title-row h1 {
  margin: 0;
  color: var(--cb-text);
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -.35px;
}
.workspace-heading p {
  margin: 4px 0 0;
  color: var(--cb-text-muted);
  font-size: 12px;
  font-weight: 600;
}
.workspace-primary-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.workspace-toolbar {
  min-height: 54px;
  padding: 10px 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-shrink: 0;
  background: var(--cb-bg);
  border-bottom: 1px solid var(--cb-border-light);
}
.workspace-toolbar-left { display: flex; align-items: center; min-width: 0; }
.workspace-toolbar-left > strong { color: var(--cb-text); font-size: 14px; white-space: nowrap; }
.drive-sort-inline {
  display: inline-flex; align-items: center; gap: 4px;
  margin-left: 12px;
}
.drive-sort-inline .el-radio-group { --el-radio-button-bg-color: var(--cb-bg-alt); }
.view-toggle-btn {
  width: 32px; height: 32px;
  border: 1px solid var(--cb-border);
  border-radius: 8px;
  background: var(--cb-surface);
  display: flex; align-items: center; justify-content: center;
}

/* ── Search banner ── */
.drive-search-banner {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 28px;
  background: var(--cb-primary-light);
  font-size: 13px; color: var(--cb-primary); font-weight: 600;
  border-bottom: 1px solid var(--cb-primary-soft);
}

/* ── Files ── */
.drive-workspace { flex: 1; min-height: 0; display: flex; overflow: hidden; }
.drive-files { flex: 1; min-width: 0; overflow: auto; padding: 20px 28px 32px; }

.file-details {
  width: clamp(280px, 24vw, 340px);
  flex-shrink: 0;
  padding: 18px 20px 28px;
  overflow-y: auto;
  background: var(--cb-surface);
  border-left: 1px solid var(--cb-border);
  box-shadow: -10px 0 24px rgba(15, 23, 42, .025);
}
.details-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
.details-head strong { color: var(--cb-text); font-size: 14px; }
.details-head button {
  width: 30px; height: 30px; padding: 0; border: 0; border-radius: 7px;
  display: grid; place-items: center; background: transparent; color: var(--cb-text-muted); cursor: pointer;
}
.details-head button:hover { background: var(--cb-bg-alt); color: var(--cb-text); }
.details-preview {
  height: 180px;
  display: grid;
  place-items: center;
  overflow: hidden;
  background: var(--cb-bg-alt);
  border: 1px solid var(--cb-border-light);
  border-radius: var(--cb-radius);
}
.details-preview.folder { background: #fffbf0; }
.details-preview img { width: 100%; height: 100%; object-fit: contain; }
.file-details h2 {
  margin: 16px 0;
  overflow: hidden;
  color: var(--cb-text);
  font-size: 16px;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.details-actions { display: flex; flex-wrap: wrap; gap: 8px; padding-bottom: 20px; border-bottom: 1px solid var(--cb-border-light); }
.details-actions .el-button + .el-button { margin-left: 0; }
.details-meta { margin: 18px 0 0; }
.details-meta > div { display: grid; grid-template-columns: 74px minmax(0, 1fr); gap: 12px; padding: 8px 0; }
.details-meta dt { color: var(--cb-text-muted); font-size: 12px; }
.details-meta dd { margin: 0; color: var(--cb-text-secondary); font-size: 12px; line-height: 1.5; word-break: break-word; }

.drive-pagination {
  display: flex;
  justify-content: center;
  padding: 8px 0 4px;
}

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
.file-card.dragging { opacity: 0.4; }
.file-card.drag-over {
  border-color: var(--cb-primary);
  background: var(--cb-primary-light);
  box-shadow: inset 0 0 0 2px var(--cb-primary);
  transform: scale(1.03);
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
.preview-img-wrap { position: relative; display: flex; align-items: center; justify-content: center; }
.preview-img { width: 100%; max-height: 520px; object-fit: contain; display: block; }
.preview-nav {
  position: absolute; top: 50%; transform: translateY(-50%);
  width: 44px; height: 44px;
  border: 0; border-radius: 50%;
  background: var(--cb-surface);
  color: var(--cb-text-secondary);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  box-shadow: var(--cb-shadow-md);
  transition: all var(--cb-transition-fast);
  z-index: 2;
}
.preview-nav:hover { background: var(--cb-bg-alt); color: var(--cb-text); box-shadow: var(--cb-shadow-lg); }
.preview-nav-prev { left: 8px; }
.preview-nav-next { right: 8px; }
.preview-video { width: 100%; max-height: 520px; background: #000; }
.preview-audio { width: 100%; margin: 16px 0; }
.preview-frame { width: 100%; height: 520px; border: 0; }
.preview-unsupported { text-align: center; padding: 60px 0; color: var(--cb-text-muted); }
.preview-markdown {
  max-height: 520px;
  overflow-y: auto;
  padding: 24px 32px;
  background: var(--cb-surface);
  border: 1px solid var(--cb-border-light);
  border-radius: var(--cb-radius);
  font-size: 14px;
  line-height: 1.8;
  color: var(--cb-text);
}
.preview-markdown :deep(h1) { font-size: 1.8em; font-weight: 700; margin: 0.8em 0 0.5em; border-bottom: 1px solid var(--cb-border-light); padding-bottom: 0.3em; }
.preview-markdown :deep(h2) { font-size: 1.5em; font-weight: 700; margin: 0.8em 0 0.4em; }
.preview-markdown :deep(h3) { font-size: 1.25em; font-weight: 600; margin: 0.7em 0 0.3em; }
.preview-markdown :deep(p) { margin: 0.5em 0; }
.preview-markdown :deep(code) {
  background: var(--cb-bg-alt);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.9em;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
}
.preview-markdown :deep(pre) {
  background: var(--cb-bg);
  border: 1px solid var(--cb-border-light);
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
  margin: 0.8em 0;
}
.preview-markdown :deep(pre code) { background: transparent; padding: 0; }
.preview-markdown :deep(blockquote) {
  border-left: 4px solid var(--cb-primary);
  margin: 0.8em 0;
  padding: 4px 16px;
  color: var(--cb-text-secondary);
  background: var(--cb-bg-alt);
  border-radius: 0 8px 8px 0;
}
.preview-markdown :deep(table) { border-collapse: collapse; width: 100%; margin: 0.8em 0; }
.preview-markdown :deep(th), .preview-markdown :deep(td) { border: 1px solid var(--cb-border); padding: 8px 12px; text-align: left; }
.preview-markdown :deep(th) { background: var(--cb-bg-alt); font-weight: 600; }
.preview-markdown :deep(ul), .preview-markdown :deep(ol) { padding-left: 1.5em; margin: 0.5em 0; }
.preview-markdown :deep(li) { margin: 0.2em 0; }
.preview-markdown :deep(img) { max-width: 100%; border-radius: var(--cb-radius); }
.preview-markdown :deep(a) { color: var(--cb-primary); }
.preview-markdown :deep(hr) { border: 0; border-top: 1px solid var(--cb-border-light); margin: 1em 0; }

/* ── Shortcuts help ── */
.shortcuts-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sc-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--cb-bg-alt);
  font-size: 13px;
}
.sc-item kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 6px;
  border: 1px solid var(--cb-border);
  border-radius: 5px;
  background: var(--cb-surface);
  font-size: 12px;
  font-weight: 600;
  color: var(--cb-text-secondary);
  font-family: 'SFMono-Regular', Consolas, monospace;
  box-shadow: 0 1px 0 var(--cb-border);
}
.sc-item span {
  color: var(--cb-text);
  font-weight: 500;
}
</style>
