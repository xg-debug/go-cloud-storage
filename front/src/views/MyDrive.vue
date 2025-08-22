<template>
    <div class="my-drive">
        <!-- 页面头部 -->
        <div class="page-header">
            <div class="header-content">
                <div class="header-info">
                    <div class="header-icon">
                        <el-icon :size="28" color="#ffffff">
                            <Folder/>
                        </el-icon>
                    </div>
                    <div class="header-text">
                        <h1 class="page-title">我的网盘</h1>
                        <p class="page-description">{{
                            currentPath.length > 0 ? currentPath.join(' / ') : '根目录'
                            }}</p>
                    </div>
                </div>
                <div class="header-stats">
                    <div class="stat-item">
                        <span class="stat-number">{{ fileList.length }}</span>
                        <span class="stat-label">文件数量</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-number">{{ total }}</span>
                        <span class="stat-label">总数量</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- 面包屑导航 -->
        <div class="breadcrumb-container">
            <el-breadcrumb separator="/">
                <el-breadcrumb-item @click="goRoot" style="cursor: pointer">
                    <el-icon>
                        <HomeFilled/>
                    </el-icon>
                    全部文件
                </el-breadcrumb-item>
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

        <!-- 工具栏 -->
        <div class="toolbar">
            <div class="toolbar-left">
                <el-dropdown @command="handleUploadCommand">
                    <el-button type="primary" :icon="Upload">
                        上传文件
                        <el-icon class="el-icon--right">
                            <arrow-down/>
                        </el-icon>
                    </el-button>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item command="normal">
                                <el-icon>
                                    <Upload/>
                                </el-icon>
                                普通上传
                            </el-dropdown-item>
                            <el-dropdown-item command="chunk">
                                <el-icon>
                                    <Upload/>
                                </el-icon>
                                大文件上传
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>

                <!-- 隐藏的普通上传组件 -->
                <el-upload
                        ref="normalUploadRef"
                        :http-request="uploadRequest"
                        :data="{ parentId: currentParentId }"
                        :multiple="true"
                        :show-file-list="false"
                        :before-upload="beforeUpload"
                        :on-success="handleUploadSuccess"
                        :on-error="handleUploadError"
                        style="display: none"
                >
                </el-upload>

                <el-button :icon="FolderAdd" @click="handleNewFolder">
                    新建文件夹
                </el-button>
            </div>
            <div class="toolbar-right">
                <el-button-group>
                    <el-button
                            :type="viewMode === 'grid' ? 'primary' : ''"
                            :icon="Grid"
                            @click="viewMode = 'grid'"
                    >
                        网格
                    </el-button>
                    <el-button
                            :type="viewMode === 'list' ? 'primary' : ''"
                            :icon="List"
                            @click="viewMode = 'list'"
                    >
                        列表
                    </el-button>
                </el-button-group>
            </div>
        </div>

        <!-- 文件内容区域 -->
        <div class="file-content">
            <!-- 网格视图 -->
            <div v-if="viewMode === 'grid'" class="grid-view">
                <div class="file-grid">
                    <div
                            class="file-card"
                            v-for="item in fileList"
                            :key="item.id"
                            @dblclick="handleOpenFolder(item)"
                            @mouseenter="onFileItemEnter(item.id)"
                            :class="{ 'file-card-hover': hoveredId === item.id }"
                    >
                        <!-- 操作菜单 -->
                        <el-dropdown
                                v-if="hoveredId === item.id"
                                class="file-actions-dropdown"
                                trigger="hover"
                                @command="cmd => handleGridMenuCommand(item, cmd)"
                        >
                            <el-button link size="small" class="more-btn">
                                <el-icon>
                                    <MoreFilled/>
                                </el-icon>
                            </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="rename">
                                        <el-icon>
                                            <Edit/>
                                        </el-icon>
                                        重命名
                                    </el-dropdown-item>
                                    <el-dropdown-item command="star">
                                        <el-icon>
                                            <Star/>
                                        </el-icon>
                                        收藏
                                    </el-dropdown-item>
                                    <el-dropdown-item command="download" v-if="!item.is_dir">
                                        <el-icon>
                                            <Download/>
                                        </el-icon>
                                        下载
                                    </el-dropdown-item>
                                    <el-dropdown-item command="share">
                                        <el-icon>
                                            <Share/>
                                        </el-icon>
                                        分享
                                    </el-dropdown-item>
                                    <el-dropdown-item divided command="delete">
                                        <el-icon>
                                            <Delete/>
                                        </el-icon>
                                        删除
                                    </el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>

                        <!-- 文件缩略图 -->
                        <div class="file-thumbnail">
                            <el-icon v-if="item.is_dir === true" :size="48" color="#FFB800">
                                <Folder/>
                            </el-icon>
                            <el-image
                                    v-else-if="['jpg','png','gif','jpeg','webp',].includes(item.extension)"
                                    :src="item.thumbnail_url"
                                    fit="cover"
                                    class="thumbnail-image"
                            />
                            <el-icon v-else :size="48" color="#3a86ff">
                                <Document/>
                            </el-icon>
                        </div>

                        <!-- 文件信息 -->
                        <div class="file-info">
                            <div class="file-name" :title="item.name || newFolderName">
                                <template v-if="item.isTemp">
                                    <span class="temp-folder">（新建文件夹）</span>
                                </template>
                                {{ item.name || newFolderName }}
                            </div>
                            <div class="file-meta">{{ item.modified }}</div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- 列表视图 -->
            <div v-else class="list-view">
                <el-table :data="fileList" class="file-table" @row-dblclick="handleOpenFolder">
                    <el-table-column width="60">
                        <template #default="{ row }">
                            <el-icon :size="20" :color="row.is_dir === true ? '#FFB800' : '#3a86ff'">
                                <Folder v-if="row.is_dir === true"/>
                                <Document v-else/>
                            </el-icon>
                        </template>
                    </el-table-column>

                    <el-table-column prop="name" label="名称" min-width="300" show-overflow-tooltip>
                        <template #default="{ row }">
                            <span class="file-name-text">{{ row.name }}</span>
                        </template>
                    </el-table-column>

                    <el-table-column prop="modified" label="修改日期" width="180"/>
                    <el-table-column prop="size_str" label="大小" width="120"/>

                    <el-table-column label="操作" width="200" fixed="right">
                        <template #default="{ row }">
                            <el-button size="small" type="primary" link @click="handleRename(row)">
                                <el-icon>
                                    <Edit/>
                                </el-icon>
                                重命名
                            </el-button>
                            <el-button size="small" type="danger" link @click="openDeleteDialog(row)">
                                <el-icon>
                                    <Delete/>
                                </el-icon>
                                删除
                            </el-button>
                            <el-dropdown class="list-el-dropdown">
                                <el-button size="small" type="primary" link>
                                    更多
                                    <el-icon>
                                        <ArrowDown/>
                                    </el-icon>
                                </el-button>
                                <template #dropdown>
                                    <el-dropdown-menu>
                                        <el-dropdown-item @click="handleDownload(row)" v-if="!row.is_dir">下载
                                        </el-dropdown-item>
                                        <el-dropdown-item @click="handleShare(row)">分享</el-dropdown-item>
                                        <el-dropdown-item @click="handleMove(row)">移动</el-dropdown-item>
                                    </el-dropdown-menu>
                                </template>
                            </el-dropdown>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </div>

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
            <el-input v-model="renameForm.name"/>
            <template #footer>
                <el-button @click="renameDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmRename">确定</el-button>
            </template>
        </el-dialog>

        <!-- 创建分享对话框 -->
        <CreateShareDialog
                v-model="shareDialogVisible"
                :file-info="shareFileInfo"
                @success="handleShareSuccess"
        />

        <!-- 大文件上传对话框 -->
        <el-dialog
                v-model="chunkUploadDialogVisible"
                title="大文件上传"
                width="600px"
                :close-on-click-modal="false"
        >
            <ChunkUpload
                    :folder-id="currentParentId"
                    :chunk-size="2 * 1024 * 1024"
                    :max-file-size="5 * 1024 * 1024 * 1024"
                    @upload-success="handleChunkUploadSuccess"
                    @upload-error="handleChunkUploadError"
            />
            <template #footer>
                <el-button @click="chunkUploadDialogVisible = false">关闭</el-button>
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
import {onMounted, ref} from 'vue'
import {createFolder, deleteFile, listFiles, previewFile, renameFile, uploadFile} from '@/api/file'
import {ElMessage} from 'element-plus'
import {
    ArrowDown,
    Delete,
    Document,
    Download,
    Edit,
    Folder,
    FolderAdd,
    Grid,
    HomeFilled,
    List,
    MoreFilled,
    Share,
    Star,
    Upload
} from '@element-plus/icons-vue'
import {useStore} from 'vuex'
import {addFavorite} from "@/api/favorite";
import CreateShareDialog from '@/components/CreateShareDialog.vue'
import ChunkUpload from '@/components/ChunkUpload.vue'

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
const renameForm = ref({id: null, name: ''})
const hoveredId = ref(null)

const deleteDialogVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)
const shareDialogVisible = ref(false)
const shareFileInfo = ref({})
const chunkUploadDialogVisible = ref(false)
const normalUploadRef = ref()
let tempFolderId = null
let hoverTimeout = null


const uploadRequest = (options) => {
    const {file, onProgress, onSuccess, onError, data} = options
    const formData = new FormData()
    formData.append('file', file)
    for (const key in data) {
        formData.append(key, data[key])
    }
    uploadFile(formData, (event) => {
        const percent = Math.floor((event.loaded / event.total) * 100)
        onProgress({percent})
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
    if (res.list) {
        fileList.value = res.list
        total.value = res.total
    } else {
        fileList.value = []
        total.value = 0
    }
}

const handleOpenFolder = (item) => {
    if (!item.is_dir) {
        previewFile(item.id)
        return
    }
    currentParentId.value = item.id
    // 进入文件夹时，追加路径，同时将当前目录 id 入栈
    currentPath.value = [...currentPath.value, item.name]
    pathIdStack.value = [...pathIdStack.value, item.id]
    loadFiles()
}

const goRoot = () => {
    const rootId = store.state.userInfo.rootFolderId
    currentParentId.value = rootId
    currentPath.value = []       // 根目录不显示名字
    pathIdStack.value = [rootId] // 只保存 rootId
    loadFiles()
}

const handleBreadcrumbClick = (index) => {
    currentPath.value = currentPath.value.slice(0, index + 1)
    pathIdStack.value = pathIdStack.value.slice(0, index + 2)
    currentParentId.value = pathIdStack.value[pathIdStack.value.length - 1]
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

const handleUploadSuccess = (response) => {
    if (response?.name) {
        ElMessage.success('上传成功')
        loadFiles()
    } else {
        ElMessage.error(response.message || '上传失败')
    }
}

const handleUploadError = (err) => {
    ElMessage.error(err?.message || '上传失败')
}

const handleRename = (row) => {
    renameForm.value = {id: row.id, name: row.name}
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
    else if (command === 'star') handleStar(item)
    else if (command === 'share') handleShare(item)
    else if (command === 'download') handleDownload(item)
}

const handleStar = (item) => {
    addFavorite(item.id)
    ElMessage.success('收藏成功')
}

const handleShare = (item) => {
    if (item.is_dir) {
        ElMessage.warning('暂不支持分享文件夹')
        return
    }

    shareFileInfo.value = {
        id: item.id,
        name: item.name,
        size: item.size,
        fileType: getFileTypeFromExtension(item.file_extension || item.extension)
    }
    shareDialogVisible.value = true
}

// 分享成功回调
const handleShareSuccess = (shareData) => {
    ElMessage.success('分享创建成功')
    // 可以在这里添加其他逻辑，比如跳转到分享列表
}

// 根据文件扩展名获取文件类型
const getFileTypeFromExtension = (extension) => {
    if (!extension) return 'other'

    const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp']
    const videoExts = ['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv']
    const audioExts = ['mp3', 'wav', 'flac', 'aac', 'ogg']
    const docExts = ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt']

    const ext = extension.toLowerCase().replace('.', '')

    if (imageExts.includes(ext)) return 'image'
    if (videoExts.includes(ext)) return 'video'
    if (audioExts.includes(ext)) return 'audio'
    if (docExts.includes(ext)) return 'document'

    return 'other'
}

const handleDownload = (item) => {
    ElMessage.info(`下载文件: ${item.name}`)
    // TODO: 实现下载逻辑
}

const handleMove = (item) => {
    ElMessage.info(`移动文件: ${item.name}`)
    // TODO: 实现移动逻辑
}

// 处理上传命令
const handleUploadCommand = (command) => {
    if (command === 'normal') {
        // 触发普通上传
        normalUploadRef.value.$el.querySelector('input').click()
    } else if (command === 'chunk') {
        // 打开大文件上传对话框
        chunkUploadDialogVisible.value = true
    }
}

// 分片上传成功回调
const handleChunkUploadSuccess = (fileInfo) => {
    ElMessage.success(`文件 ${fileInfo.fileName} 上传成功！`)
    loadFiles() // 刷新文件列表
}

// 分片上传错误回调
const handleChunkUploadError = (error) => {
    ElMessage.error(`上传失败: ${error.message}`)
}
</script>

<style scoped>
.my-drive {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #f8fafc;
}

/* 页面头部 */
.page-header {
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    color: white;
    padding: 14px 24px;
}

.header-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.header-info {
    display: flex;
    align-items: center;
    gap: 16px;
}

.header-icon {
    width: 48px;
    height: 48px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
}

.page-title {
    font-size: 24px;
    font-weight: 600;
    margin: 0 0 4px 0;
}

.page-description {
    font-size: 14px;
    opacity: 0.9;
    margin: 0;
}

.header-stats {
    display: flex;
    gap: 32px;
}

.stat-item {
    text-align: center;
}

.stat-number {
    display: block;
    font-size: 24px;
    font-weight: 600;
    line-height: 1;
}

.stat-label {
    font-size: 12px;
    opacity: 0.8;
}

/* 面包屑导航 */
.breadcrumb-container {
    background: white;
    padding: 16px 24px;
    border-bottom: 1px solid #e2e8f0;
}

/* 工具栏 */
.toolbar {
    background: white;
    padding: 20px 24px;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;
}

.toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
}

/* 文件内容 */
.file-content {
    flex: 1;
    background: white;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}

/* 网格视图 */
.grid-view {
    flex: 1;
    padding: 24px;
    overflow: auto;
}

.file-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 20px;
}

.file-card {
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    padding: 16px;
    transition: all 0.3s ease;
    cursor: pointer;
    position: relative;
}

.file-card:hover,
.file-card-hover {
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
    border-color: #3b82f6;
    transform: translateY(-2px);
}

.file-actions-dropdown {
    position: absolute;
    top: 8px;
    right: 8px;
    z-index: 2;
}

.more-btn {
    padding: 4px;
    min-width: 0;
    background: rgba(255, 255, 255, 0.9) !important;
    border-radius: 50%;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.file-thumbnail {
    width: 100%;
    height: 120px;
    border-radius: 8px;
    overflow: hidden;
    background: #f9fafb;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 12px;
}

.thumbnail-image {
    width: 100%;
    height: 100%;
}

.file-info {
    text-align: center;
}

.file-name {
    font-size: 14px;
    font-weight: 500;
    color: #1f2937;
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.temp-folder {
    color: #9ca3af;
    font-style: italic;
}

.file-meta {
    font-size: 12px;
    color: #6b7280;
}

/* 列表视图 */
.list-view {
    flex: 1;
    padding: 24px;
    overflow: auto;
}

.file-table {
    border-radius: 8px;
    overflow: visible;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.file-table :deep(.el-table__header) {
    background: #f8fafc;
}

.file-table :deep(.el-table__row:hover) {
    background: #f0f9ff;
}

.file-name-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.list-el-dropdown {
    padding-top: 4px;
}

/* 对话框样式 */
.delete-confirm-text {
    text-align: center;
    font-size: 14px;
    line-height: 1.8;
    user-select: none;
}

.delete-confirm-text strong {
    color: #ef4444;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .page-header {
        padding: 20px 16px;
    }

    .header-content {
        flex-direction: column;
        gap: 20px;
        text-align: center;
    }

    .header-stats {
        gap: 24px;
    }

    .breadcrumb-container {
        padding: 12px 16px;
    }

    .toolbar {
        padding: 16px;
        flex-direction: column;
        gap: 16px;
        align-items: stretch;
    }

    .toolbar-left {
        justify-content: center;
    }

    .toolbar-right {
        justify-content: center;
    }

    .file-grid {
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 16px;
        padding: 16px;
    }

    .grid-view,
    .list-view {
        padding: 16px;
    }
}
</style>
