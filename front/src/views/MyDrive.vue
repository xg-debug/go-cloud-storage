<template>
    <div class="my-drive">
        <!-- 页面头部 -->
        <div class="page-header">
            <div class="header-content">
                <div class="header-info">
                    <div class="header-icon">
                        <el-icon :size="28" class="header-folder-icon">
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
                        <span class="stat-number">{{ fileNumber }}</span>
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
                <el-button type="primary" :icon="Upload" @click="triggerUploadDialog">
                    上传文件
                </el-button>
                <input
                    type="file"
                    ref="fileInputRef"
                    style="display: none"
                    @change="handleFileInputChange" >

                <el-button :icon="FolderAdd" @click="handleNewFolder">
                    新建文件夹
                </el-button>
            </div>
            <div class="toolbar-right">
                <el-input
                    v-model="searchKeyword"
                    placeholder="搜索文件和文件夹..."
                    class="search-input"
                    clearable
                    @input="handleSearch"
                    @clear="clearSearch"
                    style="width: 280px; margin-right: 16px;"
                >
                    <template #prefix>
                        <el-icon>
                            <Search/>
                        </el-icon>
                    </template>
                </el-input>
                <el-button-group>
                    <el-button
                            :type="viewMode === 'grid' ? 'primary' : ''"
                            :icon="Grid"
                            @click="viewMode = 'grid'"
                    >
                    </el-button>
                    <el-button
                            :type="viewMode === 'list' ? 'primary' : ''"
                            :icon="List"
                            @click="viewMode = 'list'"
                    >
                    </el-button>
                </el-button-group>
            </div>
        </div>

        <!-- 搜索结果提示 -->
        <div v-if="isSearching" class="search-result-tip">
            <el-alert
                :title="`搜索 &quot;${searchKeyword}&quot; 找到 ${fileList.length} 个结果`"
                type="info"
                :closable="false"
                show-icon
            >
                <template #default>
                    <span>在当前目录中搜索到 {{ fileList.length }} 个匹配的文件和文件夹</span>
                    <el-button type="text" size="small" @click="clearSearch" style="margin-left: 10px;">
                        清除搜索
                    </el-button>
                </template>
            </el-alert>
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
                                    <el-dropdown-item command="move">
                                        <el-icon>
                                            <FolderAdd/>
                                        </el-icon>
                                        移动
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

        <!-- 文件上传对话框 -->
        <el-dialog
            v-model="uploadDialogVisible"
            title="文件上传"
            width="500px"
            :close-on-click-modal="false"
            @close="resetUploadState"
        >
            <div
                v-if="!pendingFile"
                class="drop-zone"
                :class="{ dragging: isDragging }"
                @dragover.prevent="isDragging = true"
                @dragleave.prevent="isDragging = false"
                @drop.prevent="onDrop"
            >
                <el-icon :size="48"><Upload /></el-icon>
                <p>将文件拖拽到此处 或</p>
                <el-button link @click="triggerSelect">选择本地文件</el-button>
                <input type="file" ref="uploadInputRef" style="display: none" @change="onSelectFile">
            </div>

            <!-- 上传进度 -->
            <div v-else class="upload-progress-info">
                <h4 style="margin-bottom: 15px;">正在上传：{{ pendingFile.name }}</h4>

                <el-progress :percentage="uploadProgress" />

                <p>{{ uploadStatusText }}</p>
            </div>

            <template #footer>
                <el-button @click="resetUploadState" :disabled="uploading">关闭</el-button>
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

        <!-- 移动文件对话框 -->
        <el-dialog
                v-model="moveDialogVisible"
                title="移动文件"
                width="500px"
                :close-on-click-modal="false"
                @close="cancelMove"
        >
            <div class="move-dialog-content">
                <div class="move-info">
                    <span>将文件 <strong>{{ moveTarget.name }}</strong> 移动到：</span>
                </div>
                <div class="folder-tree-container">
                    <el-tree
                            ref="folderTreeRef"
                            :data="folderTree"
                            :props="{ children: 'children', label: 'name', value: 'id' }"
                            node-key="id"
                            :default-expand-all="false"
                            :expand-on-click-node="false"
                            :highlight-current="true"
                            @node-click="handleFolderSelect"
                            class="folder-tree"
                    >
                        <template #default="{ node, data }">
                            <div class="folder-node" :class="{ 'selected': selectedTargetFolder && selectedTargetFolder.id === data.id }">
                                <el-icon><Folder /></el-icon>
                                <span>{{ data.name }}</span>
                            </div>
                        </template>
                    </el-tree>
                </div>
                <div class="selected-folder" v-if="selectedTargetFolder">
                    <el-icon><Folder /></el-icon>
                    <span>目标文件夹：{{ selectedTargetFolder.name }}</span>
                </div>
            </div>
            <template #footer>
                <el-button @click="cancelMove">取消</el-button>
                <el-button 
                    type="primary" 
                    @click="confirmMove" 
                    :loading="moving"
                    :disabled="!selectedTargetFolder"
                >
                    确定移动
                </el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import {onMounted, ref} from 'vue'
import {createFolder, deleteFile, getFolderTree, listFiles, moveFile, previewFile, renameFile, searchFiles, uploadFile,
    chunkUploadInit, chunkUploadPart, chunkUploadMerge, chunkUploadCancel} from '@/api/file'
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
    Search,
    Share,
    Star,
    Upload
} from '@element-plus/icons-vue'
import {useStore} from 'vuex'
import {addFavorite} from "@/api/favorite";
import CreateShareDialog from '@/components/CreateShareDialog.vue'

const store = useStore()
const viewMode = ref('grid')
const currentPath = ref([])
const currentParentId = ref('')
const pathIdStack = ref([])
const fileList = ref([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchKeyword = ref('')
const isSearching = ref(false)
const originalFileList = ref([])
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

let tempFolderId = null
let hoverTimeout = null

const fileNumber = ref(0)

// 移动文件相关状态
const moveDialogVisible = ref(false)
const moveTarget = ref({})
const folderTree = ref([])
const selectedTargetFolder = ref('')
const moving = ref(false)
const folderTreeRef = ref()

// 上传相关状态
const uploadDialogVisible = ref(false)
const uploadInputRef = ref(null)

const pendingFile = ref(null)            // 当前上传的文件
const uploadProgress = ref(0)            // 总进度（大文件/小文件统一）
const uploading = ref(false)             // 上传中
const isDragging = ref(false)

const CHUNK_SIZE = 10 * 1024 * 1024      // 10MB
const CHUNK_THRESHOLD = 50 * 1024 * 1024 // 判定大文件

// 打开上传对话框
const triggerUploadDialog = () => {
    uploadDialogVisible.value = true
}

/* 打开本地选择文件 */
const triggerSelect = () => {
    uploadInputRef.value.click()
}

/* 选择文件 */
const onSelectFile = (e) => {
    const file = e.target.files[0]
    e.target.value = ''
    if (file) prepareUpload(file)
}

/* 拖拽上传 */
const onDrop = (e) => {
    const file = e.dataTransfer.files[0]
    isDragging.value = false
    if (file) prepareUpload(file)
}

/* 准备上传（区分大小文件） */
const prepareUpload = (file) => {
    pendingFile.value = file
    uploadProgress.value = 0

    if (file.size >= CHUNK_THRESHOLD) {
        uploadLargeFile(file)
    } else {
        uploadSmallFile(file)
    }
}

const uploadSmallFile = async (file) => {
    uploading.value = true

    const form = new FormData()
    form.append('file', file)
    form.append('parentId', currentParentId.value)

    try {
        await uploadFile(form, (e) => {
            uploadProgress.value = Math.round((e.loaded * 100) / e.total)
        })

        ElMessage.success('上传成功')
        resetUploadState()
        loadFiles()
    } catch (err) {
        ElMessage.error('上传失败')
    } finally {
        uploading.value = false
    }
}

const uploadLargeFile = async (file) => {
    uploading.value = true
    uploadProgress.value = 0

    try {
        const fileHash = await calcSHA256(file)

        // 初始化任务
        const initRes = await chunkUploadInit({
            fileHash,
            parentId: currentParentId.value,
            fileName: file.name,
            fileSize: file.size,
        })

        // 秒传成功
        if (initRes.finished) {
            uploadProgress.value = 100
            ElMessage.success('秒传成功')
            resetUploadState()
            loadFiles()
            return
        }

        const uploaded = new Set(initRes.uploadedChunks)
        const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
        let finished = uploaded.size

        // 上传每个分片
        for (let index = 0; index < totalChunks; index++) {
            if (uploaded.has(index)) {
                updateProgress(uploaded.size, totalChunks)
                continue
            }

            const start = index * CHUNK_SIZE
            const end = Math.min(file.size, start + CHUNK_SIZE)
            const chunk = file.slice(start, end)

            const form = new FormData()
            form.append('fileHash', fileHash)
            form.append('chunkIndex', index)
            form.append('chunk', chunk)

            await chunkUploadPart(form, () => {})

            finished++
            updateProgress(finished, totalChunks)
        }

        // 合并分片
        uploadProgress.value = 98
        await chunkUploadMerge({
            fileHash,
            fileName: file.name,
            fileSize: file.size,
            parentId: currentParentId.value
        })

        uploadProgress.value = 100
        ElMessage.success('上传成功')

        resetUploadState()
        loadFiles()

    } catch (err) {
        console.error(err)
        ElMessage.error('大文件上传失败')
    } finally {
        uploading.value = false
    }
}
/* 更新进度条（95% 用于上传部分） */
const updateProgress = (finished, total) => {
    uploadProgress.value = Math.round((finished / total) * 95)
}
/* -----------------------------------------------------
   计算 SHA-256（浏览器原生）
------------------------------------------------------ */
const calcSHA256 = async (file) => {
    const buffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
    return Array.from(new Uint8Array(hashBuffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('')
}
/* -----------------------------------------------------
   上传对话框关闭时重置
------------------------------------------------------ */
const resetUploadState = () => {
    if (uploading.value) return ElMessage.warning('上传中请勿关闭')
    pendingFile.value = null
    uploadProgress.value = 0
    uploading.value = false
    uploadDialogVisible.value = false
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
        // 文件数量
        fileNumber.value = res.list.filter(file => file.is_dir === false).length
        
        // 如果当前正在搜索，清除搜索状态并重新应用搜索
        if (isSearching.value) {
            const keyword = searchKeyword.value
            isSearching.value = false
            originalFileList.value = []
            if (keyword.trim()) {
                searchKeyword.value = keyword
                performSearch()
            }
        }
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
    
    // 清除搜索状态
    clearSearch()
    
    currentParentId.value = item.id
    // 进入文件夹时，追加路径，同时将当前目录 id 入栈
    currentPath.value = [...currentPath.value, item.name]
    pathIdStack.value = [...pathIdStack.value, item.id]
    loadFiles()
}

const goRoot = () => {
    // 清除搜索状态
    clearSearch()
    
    const rootId = store.state.userInfo.rootFolderId
    currentParentId.value = rootId
    currentPath.value = []       // 根目录不显示名字
    pathIdStack.value = [rootId] // 只保存 rootId
    loadFiles()
}

const handleBreadcrumbClick = (index) => {
    // 清除搜索状态
    clearSearch()
    
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

// 处理文件选择变化（核心分流逻辑）
const handleFileChange = (e) => {
    const file = e.target.files[0]
    if (!file) return

    // 重置 input value，允许重复选择同一文件
    e.target.value = ''

    if (file.size > CHUNK_THRESHOLD) {
        // --- 走大文件分片逻辑 ---
        console.log('文件较大，使用分片上传')
        pendingLargeFile.value = file
        chunkUploadDialogVisible.value = true
    } else {
        // --- 走小文件直接上传逻辑 ---
        console.log('文件较小，使用直接上传')
        handleSmallFileUpload(file)
    }
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
    else if (command === 'move') handleMove(item)
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

const handleMove = async (item) => {
    moveTarget.value = item
    moveDialogVisible.value = true
    selectedTargetFolder.value = ''
    
    // 加载文件夹树
    await loadFolderTree()
}

// 加载文件夹树结构
const loadFolderTree = async () => {
    try {
        const res = await getFolderTree()
        folderTree.value = buildFolderTree(res.data || res.list || [])
    } catch (error) {
        console.error('加载文件夹失败:', error)
        ElMessage.error('加载文件夹失败')
        folderTree.value = []
    }
}

// 构建文件夹树结构
const buildFolderTree = (folders) => {
    const tree = []
    const map = new Map()
    
    // 添加根目录
    const rootFolder = {
        id: store.state.userInfo.rootFolderId,
        name: '根目录',
        parentId: null,
        children: []
    }
    tree.push(rootFolder)
    map.set(rootFolder.id, rootFolder)
    
    // 构建树结构
    folders.forEach(folder => {
        const node = {
            id: folder.id,
            name: folder.name,
            parentId: folder.parent_id || folder.parentId,
            children: []
        }
        map.set(folder.id, node)
        
        if (folder.parent_id || folder.parentId) {
            const parentId = folder.parent_id || folder.parentId
            if (map.has(parentId)) {
                map.get(parentId).children.push(node)
            }
        } else {
            rootFolder.children.push(node)
        }
    })
    
    return tree
}

// 选择目标文件夹
const handleFolderSelect = (data) => {
    // 不能移动到当前文件夹
    if (data.id === currentParentId.value) {
        ElMessage.warning('不能移动到当前文件夹')
        return
    }
    
    // 不能移动到自己或子文件夹（如果移动的是文件夹）
    if (moveTarget.value.is_dir && isSubFolder(data.id, moveTarget.value.id)) {
        ElMessage.warning('不能移动到自己或子文件夹')
        return
    }
    
    selectedTargetFolder.value = data
}

// 检查是否为子文件夹
const isSubFolder = (targetId, sourceId) => {
    // 递归检查文件夹树，防止将文件夹移动到自己的子文件夹中
    const checkRecursive = (folders, parentId) => {
        for (const folder of folders) {
            if (folder.id === parentId) {
                return true
            }
            if (folder.children && folder.children.length > 0) {
                if (checkRecursive(folder.children, parentId)) {
                    return true
                }
            }
        }
        return false
    }
    
    // 找到源文件夹节点
    const findFolder = (folders, folderId) => {
        for (const folder of folders) {
            if (folder.id === folderId) {
                return folder
            }
            if (folder.children && folder.children.length > 0) {
                const found = findFolder(folder.children, folderId)
                if (found) return found
            }
        }
        return null
    }
    
    const sourceFolder = findFolder(folderTree.value, sourceId)
    if (sourceFolder && sourceFolder.children) {
        return checkRecursive(sourceFolder.children, targetId)
    }
    
    return false
}

// 确认移动
const confirmMove = async () => {
    if (!selectedTargetFolder.value) {
        ElMessage.warning('请选择目标文件夹')
        return
    }
    
    moving.value = true
    try {
        await moveFile({
            fileId: moveTarget.value.id,
            targetFolderId: selectedTargetFolder.value.id
        })
        
        ElMessage.success('移动成功')
        moveDialogVisible.value = false
        loadFiles() // 刷新当前文件列表
    } catch (error) {
        console.error('移动失败:', error)
        ElMessage.error('移动失败：' + (error.message || '未知错误'))
    } finally {
        moving.value = false
    }
}

// 取消移动
const cancelMove = () => {
    moveDialogVisible.value = false
    moveTarget.value = {}
    selectedTargetFolder.value = ''
}

// 搜索处理函数
let searchTimeout = null
const handleSearch = () => {
    // 防抖处理，避免频繁搜索
    clearTimeout(searchTimeout)
    searchTimeout = setTimeout(() => {
        performSearch()
    }, 300)
}

// 执行搜索
const performSearch = async () => {
    const keyword = searchKeyword.value.trim()
    
    if (!keyword) {
        clearSearch()
        return
    }
    
    // 如果不是搜索状态，保存原始文件列表
    if (!isSearching.value) {
        originalFileList.value = [...fileList.value]
        isSearching.value = true
    }
    
    try {
        // 调用后端搜索API
        const res = await searchFiles({
            keyword: keyword,
            parentId: currentParentId.value, // 在当前目录下搜索
            page: 1,
            pageSize: 100 // 搜索结果较多时可以分页
        })
        
        if (res.list) {
            fileList.value = res.list
            total.value = res.total || res.list.length
        } else {
            fileList.value = []
            total.value = 0
        }
    } catch (error) {
        console.error('搜索失败:', error)
        ElMessage.error('搜索失败，请重试')
        // 搜索失败时恢复原始列表
        clearSearch()
    }
}

// 清除搜索
const clearSearch = () => {
    if (isSearching.value) {
        fileList.value = [...originalFileList.value]
        total.value = originalFileList.value.length
        isSearching.value = false
        originalFileList.value = []
    }
    searchKeyword.value = ''
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
    background: #ffffff;
    padding: 24px 32px;
    border-bottom: 1px solid #e2e8f0;
}

.header-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.header-info {
    display: flex;
    align-items: center;
    gap: 20px;
}

.header-icon {
    width: 56px;
    height: 56px;
    background: #eff6ff;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 6px -1px rgba(59, 130, 246, 0.1), 0 2px 4px -1px rgba(59, 130, 246, 0.06);
}

.header-folder-icon {
    color: #3b82f6;
}

.page-title {
    font-size: 24px;
    font-weight: 700;
    color: #1e293b;
    margin: 0 0 4px 0;
    letter-spacing: -0.5px;
}

.page-description {
    font-size: 14px;
    color: #64748b;
    margin: 0;
    font-weight: 500;
}

.header-stats {
    display: flex;
    gap: 48px;
}

.stat-item {
    text-align: center;
    position: relative;
}

.stat-item:not(:last-child)::after {
    content: '';
    position: absolute;
    right: -24px;
    top: 50%;
    transform: translateY(-50%);
    width: 1px;
    height: 24px;
    background: #e2e8f0;
}

.stat-number {
    display: block;
    font-size: 28px;
    font-weight: 700;
    color: #1e293b;
    line-height: 1.2;
}

.stat-label {
    font-size: 13px;
    color: #64748b;
    font-weight: 500;
}

/* 面包屑导航 */
.breadcrumb-container {
    background: white;
    padding: 16px 32px;
    border-bottom: 1px solid #f1f5f9;
}

/* 工具栏 */
.toolbar {
    background: white;
    padding: 16px 32px;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.02);
    z-index: 10;
}

.toolbar-left {
    display: flex;
    align-items: center;
    gap: 16px;
}

.toolbar-right {
    display: flex;
    align-items: center;
    gap: 16px;
}

/* 搜索结果提示 */
.search-result-tip {
    padding: 16px 32px;
    background: #fffbeb;
    border-bottom: 1px solid #fcd34d;
}

/* 文件内容 */
.file-content {
    flex: 1;
    background: #f8fafc;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}

/* 网格视图 */
.grid-view {
    flex: 1;
    padding: 32px;
    overflow: auto;
}

.file-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 24px;
}

.file-card {
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 16px;
    padding: 20px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    cursor: pointer;
    position: relative;
    display: flex;
    flex-direction: column;
}

.file-card:hover,
.file-card-hover {
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.05);
    border-color: #3b82f6;
    transform: translateY(-4px);
}

.file-actions-dropdown {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 2;
    opacity: 0;
    transition: opacity 0.2s ease;
}

.file-card:hover .file-actions-dropdown,
.file-card-hover .file-actions-dropdown {
    opacity: 1;
}

.more-btn {
    padding: 6px;
    min-width: 0;
    background: white !important;
    border-radius: 8px;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
    color: #64748b;
}

.more-btn:hover {
    color: #3b82f6;
    background: #f8fafc !important;
}

.file-thumbnail {
    width: 100%;
    height: 140px;
    border-radius: 12px;
    overflow: hidden;
    background: #f8fafc;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 16px;
    transition: transform 0.3s ease;
}

.file-card:hover .file-thumbnail {
    transform: scale(1.02);
}

.thumbnail-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.file-info {
    text-align: center;
}

.file-name {
    font-size: 15px;
    font-weight: 600;
    color: #1e293b;
    margin-bottom: 6px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.4;
}

.temp-folder {
    color: #9ca3af;
    font-style: italic;
}

.file-meta {
    font-size: 13px;
    color: #94a3b8;
}

/* 列表视图 */
.list-view {
    flex: 1;
    padding: 24px 32px;
    overflow: auto;
}

.file-table {
    border-radius: 12px;
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
    border: 1px solid #e2e8f0;
}

.file-table :deep(.el-table__header) {
    background: #f8fafc;
}

.file-table :deep(.el-table__header th) {
    background: #f8fafc;
    color: #64748b;
    font-weight: 600;
    height: 56px;
}

.file-table :deep(.el-table__row) {
    height: 64px;
}

.file-table :deep(.el-table__row:hover) {
    background-color: #f8fafc;
}

.file-name-text {
    font-weight: 500;
    color: #1e293b;
}

.list-el-dropdown {
    padding-top: 4px;
}

/* 对话框样式 */
.delete-confirm-text {
    text-align: center;
    font-size: 15px;
    line-height: 1.8;
    user-select: none;
    color: #475569;
    padding: 20px 0;
}

.delete-confirm-text strong {
    color: #ef4444;
    font-weight: 600;
}

/* 移动文件对话框样式 */
.move-dialog-content {
    padding: 10px 0;
}

.move-info {
    margin-bottom: 20px;
    font-size: 15px;
    color: #475569;
}

.move-info strong {
    color: #3b82f6;
    font-weight: 600;
}

.folder-tree-container {
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    max-height: 320px;
    overflow-y: auto;
    background: #fff;
    padding: 12px;
}

.folder-tree {
    background: transparent;
}

.folder-tree :deep(.el-tree-node__content) {
    height: 40px;
    border-radius: 8px;
    margin-bottom: 2px;
}

.folder-tree :deep(.el-tree-node__content:hover) {
    background-color: #f1f5f9;
}

.folder-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: #eff6ff;
    color: #3b82f6;
}

.folder-node {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 0 8px;
}

.folder-node .el-icon {
    color: #f59e0b;
    font-size: 18px;
}

.folder-node.selected .el-icon {
    color: #3b82f6;
}

.selected-folder {
    margin-top: 20px;
    padding: 16px;
    background: #eff6ff;
    border: 1px solid #bfdbfe;
    border-radius: 12px;
    font-size: 14px;
    color: #3b82f6;
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 500;
}

.selected-folder .el-icon {
    color: #3b82f6;
    font-size: 20px;
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
        gap: 32px;
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
        flex-wrap: wrap;
    }

    .search-input {
        width: 100% !important;
        margin-right: 0 !important;
        margin-bottom: 12px;
    }

    .search-result-tip {
        padding: 12px 16px;
    }

    .file-grid {
        grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
        gap: 16px;
        padding: 0;
    }

    .grid-view,
    .list-view {
        padding: 16px;
    }
}

.drop-zone {
    border: 2px dashed #e2e8f0;
    padding: 40px 20px;
    text-align: center;
    cursor: pointer;
    transition: all 0.3s ease;
    border-radius: 16px;
    background: #f8fafc;
}

.drop-zone:hover {
    border-color: #3b82f6;
    background: #eff6ff;
}

.drop-zone.dragging {
    border-color: #3b82f6;
    background: #eff6ff;
    transform: scale(1.02);
}

.drop-zone p {
    margin: 16px 0;
    color: #64748b;
    font-size: 15px;
}
</style>
