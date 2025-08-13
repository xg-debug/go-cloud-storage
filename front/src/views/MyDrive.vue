<template>
    <div class="drive-container">
        <div class="toolbar">
            <el-upload
                :http-request="uploadRequest"
                :data="{ parentId: currentParentId }"
                :multiple="true"
                :show-file-list="false"
                :before-upload="beforeUpload"
                :on-progress="handleUploadProgress"
                :on-success="handleUploadSuccess"
                :on-error="handleUploadError"
            >
                <el-button type="primary">
                    <el-icon><Upload /></el-icon>
                    <span>上传文件</span>
                </el-button>
            </el-upload>
            <el-button @click="handleNewFolder">
                <el-icon><FolderAdd /></el-icon>
                <span>新建文件夹</span>
            </el-button>
            <div class="view-switch">
                <el-radio-group v-model="viewMode">
                    <el-radio-button label="grid">
                        <el-icon><Grid /></el-icon>
                    </el-radio-button>
                    <el-radio-button label="list">
                        <el-icon><List /></el-icon>
                    </el-radio-button>
                </el-radio-group>
            </div>
        </div>

        <el-divider />

        <div class="path-breadcrumb">
            <el-breadcrumb separator="/">
                <el-breadcrumb-item @click="goRoot">全部文件</el-breadcrumb-item>
                <el-breadcrumb-item
                    v-for="(item, index) in currentPath"
                    :key="index"
                    @click="handleBreadcrumbClick(index)"
                    style="cursor: pointer"
                >
                    {{ item }}
                </el-breadcrumb-item>
            </el-breadcrumb>
        </div>

        <!-- 网格视图 -->
        <div v-if="viewMode === 'grid'" class="file-grid">
            <div
                class="file-item"
                v-for="item in fileList"
                :key="item.id"
                @dblclick="handleOpenFolder(item)"
                @mouseenter="onFileItemEnter(item.id)"
                :class="{ 'file-item-hover': hoveredId === item.id }"
            >
                <el-dropdown
                    v-if="hoveredId === item.id"
                    class="file-item-more"
                    trigger="hover"
                    @command="cmd => handleGridMenuCommand(item, cmd)"
                >
                    <el-button link size="small" class="file-item-more-btn" tabindex="-1">
                        <el-icon><MoreFilled /></el-icon>
                    </el-button>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item command="rename">重命名</el-dropdown-item>
                            <el-dropdown-item command="delete">删除</el-dropdown-item>
                            <el-dropdown-item command="share">分享</el-dropdown-item>
                            <el-dropdown-item command="download">下载</el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>

                <div class="file-icon">
                    <el-icon v-if="item.is_dir === true" :size="48" color="#FFB800">
                        <Folder />
                    </el-icon>
                    <el-image
                        v-else-if="['jpg', 'png', 'gif'].includes(item.extension)"
                        :src="item.thumbnail_url"
                        fit="cover"
                    />
                    <el-icon v-else :size="48" color="#3a86ff">
                        <Document />
                    </el-icon>
                </div>
                <div class="file-name">
                    <template v-if="item.isTemp">
                        <span style="color:#aaa">（新建文件夹）</span>
                    </template>
                    {{ item.name || newFolderName }}
                </div>
                <div class="file-meta">{{ item.modified }}</div>
            </div>
        </div>

        <!-- 列表视图 -->
        <el-table v-else :data="fileList" style="width: 100%" :fit="true">
            <el-table-column prop="name" label="名称" min-width="200" show-overflow-tooltip>
                <template #default="{ row }">
                    <div class="file-name-cell">
                        <el-icon v-if="row.is_dir === true" color="#FFB800">
                            <Folder />
                        </el-icon>
                        <el-icon v-else color="#3a86ff">
                            <Document />
                        </el-icon>
                        <span>{{ row.name }}</span>
                    </div>
                </template>
            </el-table-column>
            <el-table-column prop="modified" label="修改日期" min-width="150" />
            <el-table-column prop="size" label="大小(KB)" min-width="100"/>
            <el-table-column label="操作" min-width="180" align="center">
                <template #default="{ row }">
                    <el-button size="small" type="text" @click="handleRename(row)">重命名</el-button>
                    <el-button size="small" type="text" @click="openDeleteDialog(row)">删除</el-button>
                    <el-dropdown>
                        <el-button size="small" type="text">
                            更多<el-icon><ArrowDown /></el-icon>
                        </el-button>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item @click="handleDownload(row)">下载</el-dropdown-item>
                                <el-dropdown-item @click="handleShare(row)">分享</el-dropdown-item>
                                <el-dropdown-item @click="handleMove(row)">移动</el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </template>
            </el-table-column>
        </el-table>

        <!-- 确定删除弹窗 -->
        <el-dialog
            v-model="deleteDialogVisible"
            title="确定删除"
            width="400px"
            :before-close="handleDeleteDialogClose"
        >
            <div class="delete-confirm-text">
                <div>确定要删除所选的文件 <strong>{{ deleteTarget.name }}</strong> 吗？</div>
                <div>删除的文件可在 10天 内通过回收站还原</div>
            </div>
            <template #footer>
                <el-button @click="deleteDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmDelete" :loading="deleting">确定</el-button>
            </template>
        </el-dialog>

        <!-- 重命名弹窗 -->
        <el-dialog v-model="renameDialogVisible" title="重命名">
            <el-input v-model="renameForm.name" />
            <template #footer>
                <el-button @click="renameDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmRename">确定</el-button>
            </template>
        </el-dialog>

        <el-dialog
            v-model="newFolderDialogVisible"
            title="新建文件夹"
            width="400px"
            :close-on-click-modal="false"
            @close="cancelNewFolder"
        >
            <el-form @submit.prevent>
                <el-form-item label="文件夹名称" required>
                    <el-input
                        v-model="newFolderName"
                        placeholder="请输入文件夹名称"
                        maxlength="50"
                        show-word-limit
                        autofocus
                    />
                </el-form-item>
                <el-form-item label="创建时间">
                    <div>{{ newFolderTime }}</div>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="cancelNewFolder">取消</el-button>
                <el-button type="primary" :loading="creatingFolder" @click="confirmNewFolder">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listFiles, createFolder, renameFile, deleteFile, previewFile, uploadFile } from '@/api/file'
import { ElMessage, ElNotification } from 'element-plus'
import {
    ArrowDown,
    MoreFilled,
    Upload,
    FolderAdd,
    Grid,
    List,
    Folder,
    Document
} from '@element-plus/icons-vue'
import { useStore } from 'vuex'

const store = useStore()
const viewMode = ref('grid')
const currentPath = ref([])
const currentParentId = ref('')
const pathIdStack = ref([])
const fileList = ref([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const newFolderDialogVisible = ref(false)
const newFolderName = ref('')
const creatingFolder = ref(false)
const newFolderTime = ref('')
const renameDialogVisible = ref(false)
const renameForm = ref({ id: null, name: '' })
const hoveredId = ref(null)

const deleteDialogVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)
let tempFolderId = null
let hoverTimeout = null

const uploadRequest = (options) => {
    const { file, onProgress, onSuccess, onError, data } = options
    const formData = new FormData()
    formData.append('file', file)
    for (const key in data) {
        formData.append(key, data[key])
    }
    uploadFile(formData, (event) => {
        const percent = Math.floor((event.loaded / event.total) * 100)
        onProgress({ percent })
    })
        .then((res) => {
            onSuccess(res)
        })
        .catch((err) => {
            onError(err)
        })
}

const loadFiles = async () => {
    const res = await listFiles({
        parentId: currentParentId.value,
        page: currentPage.value,
        pageSize: pageSize.value
    })
    fileList.value = res.list
    total.value = res.total
}

const handleOpenFolder = (item) => {
    if (!item.is_dir) {
        previewFile(item.id)
        return
    }
    currentParentId.value = item.id
    currentPath.value.push(item.name)
    pathIdStack.value.push(item.id)
    loadFiles()
}

const goRoot = () => {
    const rootId = store.state.userInfo.rootFolderId
    currentParentId.value = rootId
    currentPath.value = []
    pathIdStack.value = [rootId]
    loadFiles()
}

const handleBreadcrumbClick = (index) => {
    currentPath.value = currentPath.value.slice(0, index + 1)
    pathIdStack.value = pathIdStack.value.slice(0, index + 1)
    currentParentId.value = pathIdStack.value[index] || store.state.userInfo.rootFolderId
    loadFiles()
}

onMounted(() => {
    const rootId = store.state.userInfo.rootFolderId
    if (!rootId) {
        ElMessage.error('根目录不存在')
        return
    }
    currentParentId.value = rootId
    currentPath.value = []
    pathIdStack.value = [rootId]
    loadFiles()
})

const getCurrentTimeStr = () => {
    const now = new Date()
    return `${now.getFullYear()}年${String(now.getMonth() + 1).padStart(2, '0')}月${String(
        now.getDate()
    ).padStart(2, '0')}日 ${String(now.getHours()).padStart(2, '0')}:${String(
        now.getMinutes()
    ).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`
}

const handleNewFolder = () => {
    newFolderDialogVisible.value = true
    newFolderName.value = ''
    newFolderTime.value = getCurrentTimeStr()
    tempFolderId = 'temp_' + Date.now()
    if (!Array.isArray(fileList.value)) {
        fileList.value = []
    }
    fileList.value.unshift({
        id: tempFolderId,
        name: '',
        type: 'folder',
        size: '-',
        modified: newFolderTime.value,
        isTemp: true,
        thumbnail: ''
    })
}

const cancelNewFolder = () => {
    newFolderDialogVisible.value = false
    fileList.value = fileList.value.filter((f) => f.id !== tempFolderId)
    tempFolderId = null
}

const confirmNewFolder = async () => {
    if (!newFolderName.value.trim()) {
        ElMessage.warning('请输入文件夹名称')
        return
    }
    creatingFolder.value = true
    try {
        await createFolder({
            name: newFolderName.value.trim(),
            parentId: currentParentId.value
        })
        ElMessage.success('新建文件夹成功')
        newFolderDialogVisible.value = false
        loadFiles()
    } catch (e) {
        ElMessage.error('新建失败')
    } finally {
        creatingFolder.value = false
        fileList.value = fileList.value.filter((f) => f.id !== tempFolderId)
        tempFolderId = null
    }
}

const beforeUpload = (file) => {
    const maxSize = 1024 * 1024 * 1024 // 1GB
    if (file.size > maxSize) {
        ElMessage.error('文件大小不能超过1GB')
        return false
    }
    return true
}

// 保存通知实例（用于更新/关闭）
let uploadNotify = null

// 上传进度
const handleUploadProgress = (event) => {
    const percent = Math.round(event.percent)
    // 第一次显示通知
    if (!uploadNotify) {
        uploadNotify = ElNotification({
            title: '上传中',
            message: `上传进度：${percent}%`,
            type: 'info',
            duration: 0
        })
    } else {
        // 关闭旧的，重新开一个（Element Plus 无法直接修改已有通知）
        uploadNotify.close()
        uploadNotify = ElNotification({
            title: '上传中',
            message: `上传进度：${percent}%`,
            type: 'info',
            duration: 0
        })
    }
}

const handleUploadSuccess = (response) => {
    if (uploadNotify) {
        uploadNotify.close()
        uploadNotify = null
    }
    if (response?.name) {
        ElMessage.success('文件上传成功')
        loadFiles()
    } else {
        ElMessage.error(response.message || '文件上传失败')
    }
}

const handleUploadError = (err) => {
    if (uploadNotify) {
        uploadNotify.close()
        uploadNotify = null
    }
    ElNotification.error({
        title: '上传失败',
        message: err?.message || '未知错误',
        duration: 3000
    })
}

const handleRename = (row) => {
    renameForm.value = { id: row.id, name: row.name }
    renameDialogVisible.value = true
}

const confirmRename = async () => {
    await renameFile(renameForm.value.id, renameForm.value.name)
    ElMessage.success('重命名成功')
    renameDialogVisible.value = false
    loadFiles()
}

// 网格视图删除 - 显示确认弹窗
const openDeleteDialog = (item) => {
    deleteTarget.value = item
    deleteDialogVisible.value = true
}

// 确认删除
const confirmDelete = async () => {
    deleting.value = true
    try {
        await deleteFile(deleteTarget.value.id)
        ElMessage.success('删除成功')
        deleteDialogVisible.value = false
        loadFiles()
    } catch (error) {
        ElMessage.error('删除失败')
    } finally {
        deleting.value = false
        deleteTarget.value = {}
    }
}

// 关闭弹窗时清理
const handleDeleteDialogClose = () => {
    deleteDialogVisible.value = false
    deleteTarget.value = {}
    deleting.value = false
}

const onFileItemEnter = (id) => {
    clearTimeout(hoverTimeout)
    hoveredId.value = id
}

const handleGridMenuCommand = (item, command) => {
    if (command === 'rename') handleRename(item)
    else if (command === 'delete') openDeleteDialog(item)
    else if (command === 'share') handleShare(item)
    else if (command === 'download') handleDownload(item)
}

const handleShare = (item) => {
    ElMessage.info(`分享文件: ${item.name}`)
    // TODO: 实现分享逻辑
}

const handleDownload = (item) => {
    ElMessage.info(`下载文件: ${item.name}`)
    // TODO: 实现下载逻辑
}

const handleMove = (item) => {
    ElMessage.info(`移动文件: ${item.name}`)
    // TODO: 实现移动逻辑
}
</script>

<style scoped>
/* 样式保持不变 */
.drive-container {
    padding: 20px;
}

.toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
}

.view-switch {
    margin-left: auto;
}

.file-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 20px;
    padding: 20px 0;
}

.file-item {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 12px;
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.2s;
}

.file-item-hover,
.file-item:hover {
    background: var(--el-color-primary-light-9);
}

.file-item-more {
    position: absolute;
    top: 8px;
    right: 8px;
    z-index: 2;
}

.file-item-more-btn {
    padding: 0;
    min-width: 0;
    background: transparent !important;
    border: none !important;
    box-shadow: none !important;
    outline: none !important;
}

.file-item-more-btn:focus,
.file-item-more-btn:active {
    background: transparent !important;
    border: none !important;
    box-shadow: none !important;
    outline: none !important;
}

.file-item-more-btn .el-icon {
    font-size: 20px;
}

.file-icon {
    margin-bottom: 8px;
}

.file-name {
    font-size: 13px;
    text-align: center;
    word-break: break-all;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
}

.file-meta {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
}

.file-name-cell {
    display: flex;
    align-items: center;
    gap: 8px;
}

.file-name-cell .el-icon {
    font-size: 18px;
}

.el-table {
    margin-top: 16px;
}

.el-table :deep(.el-table__cell) {
    .cell {
        display: flex;
        gap: 4px;
    }

    .el-button {
        padding: 0px;
    }
}

.delete-confirm-text {
    text-align: center; /* 文字居中 */
    font-size: 14px;
    line-height: 1.8; /* 行高，保证两行间距合适 */
    user-select: none;
}
</style>
