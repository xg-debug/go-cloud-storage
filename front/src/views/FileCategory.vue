<template>
    <div class="file-category">
        <!-- 页面头部 -->
        <div v-if="selectedCategory" class="category-header">
            <div class="header-content">
                <div class="category-info">
                    <div class="category-icon">
                        <el-icon :size="24" :color="getCurrentCategoryColor()">
                            <component :is="getCurrentCategoryIcon()"/>
                        </el-icon>
                    </div>
                    <div class="category-text">
                        <h1 class="category-title">{{ getCurrentCategoryName() }}</h1>
                        <span class="file-count">{{ getCurrentCategoryCount() }} 个文件</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- 工具栏 -->
        <div class="toolbar">
            <div class="toolbar-left">
                <el-input
                        v-model="searchKeyword"
                        placeholder="搜索文件..."
                        :prefix-icon="Search"
                        clearable
                        style="width: 300px;"
                        @input="handleSearch"
                />
            </div>
            <div class="toolbar-right">
                <div class="view-toggle">
                    <el-radio-group v-model="viewMode" size="small">
                        <el-radio-button label="grid">
                            <el-icon>
                                <Grid/>
                            </el-icon>
                        </el-radio-button>
                        <el-radio-button label="list">
                            <el-icon>
                                <List/>
                            </el-icon>
                        </el-radio-button>
                    </el-radio-group>
                </div>
                <el-select v-model="sortBy" @change="handleSortChange" size="small" style="width: 120px;">
                    <el-option label="按名称" value="name"/>
                    <el-option label="按大小" value="size"/>
                    <el-option label="按日期" value="created_at"/>
                </el-select>
                <el-select v-model="sortOrder" @change="handleSortChange" size="small" style="width: 80px;">
                    <el-option label="升序" value="asc"/>
                    <el-option label="降序" value="desc"/>
                </el-select>
            </div>
        </div>

        <!-- 默认欢迎状态 -->
        <div v-if="!selectedCategory" class="welcome-container">
            <div class="welcome-content">
                <div class="welcome-icon">
                    <el-icon :size="100" color="#c0c4cc">
                        <FolderOpened/>
                    </el-icon>
                </div>
                <h2>选择文件分类</h2>
                <p>请从左侧菜单选择一个文件分类来浏览对应的文件</p>
                <div class="category-preview">
                    <div class="preview-item" v-for="category in categories" :key="category.type">
                        <el-icon :size="24" :color="category.color">
                            <component :is="category.icon"/>
                        </el-icon>
                        <span>{{ category.name }}</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- 文件内容区域 -->
        <div v-else class="file-content">
            <!-- 加载状态 -->
            <div v-if="loading" class="loading-container">
                <el-icon class="is-loading" :size="40">
                    <Loading/>
                </el-icon>
                <p>正在加载{{ getCurrentCategoryName() }}文件...</p>
            </div>

            <!-- 网格视图 -->
            <div v-else-if="viewMode === 'grid'" class="file-grid">
                <div
                        class="file-item"
                        v-for="item in filteredFiles"
                        :key="item.id"
                        @dblclick="previewFileHandler(item)"
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
                            <el-icon>
                                <MoreFilled/>
                            </el-icon>
                        </el-button>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item command="star">
                                    <el-icon>
                                        <Star/>
                                    </el-icon>
                                    收藏
                                </el-dropdown-item>
                                <el-dropdown-item command="download">
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

                    <div class="file-thumbnail">
                        <img v-if="item.thumbnail_url" :src="item.thumbnail_url" :alt="item.name"/>
                        <div v-else class="file-icon-wrapper">
                            <el-icon :size="48" :color="getFileIconColor(getFileType(item.extension))">
                                <component :is="getFileIcon(getFileType(item.extension))"/>
                            </el-icon>
                        </div>
                    </div>
                    <div class="file-info">
                        <div class="file-name" :title="item.name">{{ item.name }}</div>
                        <div class="file-meta">
                            <span class="file-size">{{ item.size_str }}</span>
                            <span class="file-date">{{ item.created_at }}</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- 列表视图 -->
            <el-table v-else :data="filteredFiles" style="width: 100%" :fit="true" class="file-table">
                <el-table-column prop="name" label="名称" min-width="300" show-overflow-tooltip>
                    <template #default="{ row }">
                        <div class="file-name-cell" @dblclick="previewFileHandler(row)">
                            <div class="file-thumbnail-small">
                                <img v-if="row.thumbnail_url" :src="row.thumbnail_url" :alt="row.name"/>
                                <el-icon v-else :size="20" :color="getFileIconColor(getFileType(row.extension))">
                                    <component :is="getFileIcon(getFileType(row.extension))"/>
                                </el-icon>
                            </div>
                            <span class="file-name-text">{{ row.name }}</span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="size_str" label="大小" min-width="100">
                    <template #default="{ row }">
                        {{ row.size_str }}
                    </template>
                </el-table-column>
                <el-table-column prop="modified" label="创建日期" min-width="150">
                    <template #default="{ row }">
                        {{ row.created_at }}
                    </template>
                </el-table-column>
                <el-table-column label="操作" min-width="200" fixed="right">
                    <template #default="{ row }">
                        <el-button size="small" type="text" @click="toggleFavorite(row)">
                            <el-icon>
                                <Star/>
                            </el-icon>
                            收藏
                        </el-button>
                        <el-button size="small" type="text" @click="downloadFileHandler(row)">
                            <el-icon>
                                <Download/>
                            </el-icon>
                            下载
                        </el-button>
                        <el-dropdown @command="cmd => handleTableMenuCommand(row, cmd)">
                            <el-button size="small" type="text">
                                更多
                                <el-icon>
                                    <ArrowDown/>
                                </el-icon>
                            </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="share">分享</el-dropdown-item>
                                    <el-dropdown-item command="rename">重命名</el-dropdown-item>
                                    <el-dropdown-item divided command="delete">删除</el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </template>
                </el-table-column>
            </el-table>

            <!-- 空状态 -->
            <div v-if="!loading && filteredFiles.length === 0" class="empty-state">
                <div class="empty-icon">
                    <el-icon :size="80" color="#c0c4cc">
                        <component :is="getCurrentCategoryIcon()"/>
                    </el-icon>
                </div>
                <h3>暂无{{ getCurrentCategoryName() }}文件</h3>
                <p v-if="searchKeyword">没有找到包含 "{{ searchKeyword }}" 的文件</p>
                <p v-else>您还没有上传任何{{ getCurrentCategoryName() }}文件</p>
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
    </div>
</template>

<script setup>
import {computed, onMounted, ref, watch} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {deleteFile, getFilesByCategory, previewFile} from '@/api/file'
import {addFavorite} from '@/api/favorite'
import {ElMessage, ElMessageBox} from 'element-plus'
import {
    ArrowDown,
    Delete,
    Document,
    Download,
    FolderOpened,
    Grid,
    Headset,
    List,
    MoreFilled,
    Picture,
    Search,
    Share,
    Star,
    VideoCamera
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()

const selectedCategory = ref(route.params.type || null)
const viewMode = ref('grid')
const sortBy = ref('created_at')
const sortOrder = ref('desc')
const loading = ref(false)
const files = ref([])
const hoveredId = ref(null)
const searchKeyword = ref('')
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const deleteDialogVisible = ref(false)
const deleteTarget = ref({})
const deleting = ref(false)

const categories = ref([
    {
        type: 'image',
        name: '相册',
        icon: Picture,
        color: '#f56565',
        description: '图片和照片文件',
        count: 0,
        totalSize: 0,
        extensions: ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg']
    },
    {
        type: 'video',
        name: '视频',
        icon: VideoCamera,
        color: '#9f7aea',
        description: '视频和影片文件',
        count: 0,
        totalSize: 0,
        extensions: ['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv', 'webm']
    },
    {
        type: 'audio',
        name: '音频',
        icon: Headset,
        color: '#38b2ac',
        description: '音乐和音频文件',
        count: 0,
        totalSize: 0,
        extensions: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma']
    },
    {
        type: 'document',
        name: '文档',
        icon: Document,
        color: '#4299e1',
        description: '文档和办公文件',
        count: 0,
        totalSize: 0,
        extensions: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt']
    }
])

const filteredFiles = computed(() => {
    let result = [...files.value]

    // 搜索过滤
    if (searchKeyword.value) {
        result = result.filter(file =>
            file.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
        )
    }

    return result
})

const loadCategoryFiles = async (type) => {
    loading.value = true
    try {
        // 调用后端API获取分类文件
        const response = await getFilesByCategory({
            fileType: type,
            sortBy: sortBy.value,
            sortOrder: sortOrder.value,
            page: page.value,
            pageSize: pageSize.value
        })

        if (response) {
            files.value = response.list || []
            total.value = response.total || 0

            // 更新分类统计
            const categoryIndex = categories.value.findIndex(c => c.type === type)
            if (categoryIndex !== -1) {
                categories.value[categoryIndex].count = total.value
            }
        }
    } catch (error) {
        console.error('加载文件失败:', error)
        ElMessage.error('加载文件失败')
    } finally {
        loading.value = false
    }
}

// 监听路由变化
watch(() => route.params.type, (newType) => {
    selectedCategory.value = newType
    if (newType) {
        loadCategoryFiles(newType)
    }
}, {immediate: true})

// 监听排序变化
const handleSortChange = () => {
    if (selectedCategory.value) {
        loadCategoryFiles(selectedCategory.value)
    }
}

onMounted(() => {
    if (selectedCategory.value) {
        loadCategoryFiles(selectedCategory.value)
    }
})

// 获取文件类型
const getFileType = (extension) => {
    if (!extension) return 'document'

    const ext = extension.toLowerCase().replace('.', '')

    const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg']
    if (imageExts.includes(ext)) return 'image'

    const videoExts = ['mp4', 'avi', 'mov', 'wmv', 'flv', 'webm', 'mkv']
    if (videoExts.includes(ext)) return 'video'

    const audioExts = ['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a']
    if (audioExts.includes(ext)) return 'audio'

    return 'document'
}

const getCurrentCategory = () => {
    return categories.value.find(c => c.type === selectedCategory.value) || {}
}

const getCurrentCategoryName = () => {
    const category = getCurrentCategory()
    return category.name || '文件分类'
}

const getCurrentCategoryIcon = () => {
    const category = getCurrentCategory()
    return category.icon || Document
}

const getCurrentCategoryColor = () => {
    const colorMap = {
        image: '#f59e0b',
        video: '#ef4444',
        audio: '#8b5cf6',
        document: '#06b6d4'
    }
    return colorMap[selectedCategory.value] || '#6b7280'
}

const getCurrentCategoryCount = () => {
    return total.value || 0
}

const getFileIcon = (type) => {
    const iconMap = {
        image: Picture,
        video: VideoCamera,
        audio: Headset,
        document: Document
    }
    return iconMap[type] || Document
}

const getFileIconColor = (type) => {
    const colorMap = {
        image: '#f56565',
        video: '#9f7aea',
        audio: '#38b2ac',
        document: '#4299e1'
    }
    return colorMap[type] || '#718096'
}

const formatDate = (dateStr) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    })
}

const handleSearch = () => {
    // 搜索逻辑已在计算属性中处理
}

const previewFileHandler = async (file) => {
    try {
        const response = await previewFile(file.id)
        if (response && response.data) {
            const previewData = response.data

            // 根据文件类型处理预览
            if (previewData.can_preview) {
                // 这里可以根据不同的预览类型打开不同的预览组件
                window.open(previewData.file_url, '_blank')
            } else {
                ElMessage.info('该文件类型暂不支持预览')
            }
        }
    } catch (error) {
        console.error('预览文件失败:', error)
        ElMessage.error('预览文件失败')
    }
}

const toggleFavorite = async (file) => {
    try {
        await addFavorite(file.id)
        ElMessage.success('已添加到收藏夹')
    } catch (error) {
        console.error('收藏失败:', error)
        ElMessage.error('收藏失败')
    }
}

const downloadFileHandler = (file) => {
    if (file.file_url) {
        const link = document.createElement('a')
        link.href = file.file_url
        link.download = file.name
        link.click()
        ElMessage.success('开始下载')
    } else {
        ElMessage.warning('文件下载链接不可用')
    }
}

// 显示确认删除弹窗
const openDeleteDialog = (item) => {
    deleteTarget.value = item
    deleteDialogVisible.value = true
}

// 关闭弹窗时清理
const handleDeleteDialogClose = () => {
    deleteDialogVisible.value = false
    deleteTarget.value = {}
    deleting.value = false
}

// 确认删除
const confirmDelete = async () => {
    deleting.value = true
    try {
        await deleteFile(deleteTarget.value.id)
        ElMessage.success('删除成功')
        deleteDialogVisible.value = false
        loadCategoryFiles(selectedCategory.value)
    } catch (error) {
        ElMessage.error('删除失败')
    } finally {
        deleting.value = false
        deleteTarget.value = {}
    }
}

const onFileItemEnter = (id) => {
    hoveredId.value = id
}

const onFileItemLeave = () => {
    hoveredId.value = null
}

const handleGridMenuCommand = (item, command) => {
    switch (command) {
        case 'delete':
            openDeleteDialog(item)
            break
        case 'star':
            toggleFavorite(item)
            break
        case 'download':
            downloadFileHandler(item)
            break
        case 'share':
            ElMessage.info('分享功能开发中...')
            break
    }
}

const handleTableMenuCommand = (item, command) => {
    switch (command) {
        case 'delete':
            openDeleteDialog(item)
            break
        case 'share':
            ElMessage.info('分享功能开发中...')
            break
        case 'rename':
            ElMessage.info('重命名功能开发中...')
            break
    }
}

</script>

<style scoped>
.file-category {
    height: 100%;
    min-height: calc(100vh - 60px); /* 减去可能的顶部导航高度 */
    display: flex;
    flex-direction: column;
    background: #f8fafc;
    overflow: hidden;
}

/* 页面头部 */
.category-header {
    background: #f8fafc;
    color: white;
    border-bottom: 1px solid #e2e8f0;
    padding: 10px 24px;
}

.header-content {
    display: flex;
    align-items: center;
}

.category-info {
    display: flex;
    align-items: center;
    gap: 16px;
}

.category-icon {
    width: 40px;
    height: 40px;
    background: #e2e8f0;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
}

.category-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.category-title {
    font-size: 20px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
}

.file-count {
    font-size: 14px;
    color: #64748b;
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
    gap: 16px;
}

.toolbar-right {
    display: flex;
    align-items: center;
    gap: 16px;
}

/* 文件内容区域 */
.file-content {
    flex: 1;
    background: white;
    overflow: auto;
}

/* 加载状态 */
.loading-container {
    padding: 80px;
    text-align: center;
    color: #909399;
}

.loading-container p {
    margin-top: 16px;
    font-size: 16px;
}

/* 文件网格视图 */
.file-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 20px;
    padding: 24px;
}

.file-item {
    position: relative;
    background: white;
    border-radius: 12px;
    padding: 16px;
    cursor: pointer;
    transition: all 0.3s ease;
    border: 2px solid transparent;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.file-item:hover,
.file-item-hover {
    /* transform: translateY(-4px);
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
    border-color: #e2e8f0; */
    background: var(--el-color-primary-light-9);
}

.file-item-more {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 2;
}

.file-item-more-btn {
    padding: 4px;
    min-width: 0;
    background: rgba(255, 255, 255, 0.9) !important;
    border: 1px solid #e2e8f0 !important;
    border-radius: 6px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.file-thumbnail {
    width: 100%;
    height: 120px;
    border-radius: 8px;
    overflow: hidden;
    margin-bottom: 12px;
    background: #f8fafc;
    display: flex;
    align-items: center;
    justify-content: center;
}

.file-thumbnail img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.file-icon-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
}

.file-info {
    text-align: center;
}

.file-name {
    font-size: 14px;
    font-weight: 500;
    color: #2d3748;
    margin-bottom: 8px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.file-meta {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #718096;
}

/* 文件列表视图 */
.file-table {
    margin: 0;
}

.file-name-cell {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
}

.file-thumbnail-small {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    overflow: hidden;
    background: #f8fafc;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.file-thumbnail-small img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.file-name-text {
    font-weight: 500;
    color: #2d3748;
}

/* 空状态 */
.empty-state {
    padding: 80px 24px;
    text-align: center;
    color: #909399;
}

.empty-icon {
    margin-bottom: 24px;
}

.empty-state h3 {
    font-size: 20px;
    color: #4a5568;
    margin: 0 0 12px 0;
}

.empty-state p {
    margin: 0 0 24px 0;
    font-size: 14px;
}

/* 欢迎页面 */
.welcome-container {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background: white;
}

.welcome-content {
    text-align: center;
    max-width: 500px;
    padding: 40px;
}

.welcome-icon {
    margin-bottom: 32px;
}

.welcome-content h2 {
    font-size: 24px;
    font-weight: 600;
    color: #2d3748;
    margin: 0 0 16px 0;
}

.welcome-content p {
    font-size: 16px;
    color: #718096;
    margin: 0 0 32px 0;
    line-height: 1.6;
}

.category-preview {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
    margin-top: 32px;
}

.preview-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    background: #f8fafc;
    border-radius: 12px;
    transition: all 0.3s ease;
    cursor: pointer;
}

.preview-item:hover {
    background: #edf2f7;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.preview-item span {
    font-size: 14px;
    font-weight: 500;
    color: #4a5568;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .category-header {
        padding: 24px 16px;
    }

    .header-content {
        flex-direction: column;
        gap: 20px;
        text-align: center;
    }

    .category-info {
        flex-direction: column;
        text-align: center;
    }

    .category-stats {
        justify-content: center;
    }

    .toolbar {
        padding: 16px;
        flex-direction: column;
        gap: 16px;
        align-items: stretch;
    }

    .toolbar-right {
        justify-content: space-between;
    }

    .file-grid {
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 16px;
        padding: 16px;
    }

    .file-thumbnail {
        height: 100px;
    }

    .welcome-content {
        padding: 20px;
    }

    .category-preview {
        grid-template-columns: 1fr;
    }
}
</style>