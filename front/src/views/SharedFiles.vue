<template>
    <div class="shared-files">
        <!-- 页面头部 -->
        <div class="page-header">
            <div class="header-content">
                <div class="header-info">
                    <div class="header-icon">
                        <el-icon :size="28" color="#10b981">
                            <Share/>
                        </el-icon>
                    </div>
                    <div class="header-text">
                        <h1 class="page-title">我的分享</h1>
                        <p class="page-description">管理您分享的文件和链接</p>
                    </div>
                </div>
                <div class="header-stats">
                    <div class="stat-item">
                        <span class="stat-number">{{ totalShares }}</span>
                        <span class="stat-label">总分享</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-number">{{ activeShares }}</span>
                        <span class="stat-label">有效分享</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- 工具栏 -->
        <div class="toolbar">
            <div class="toolbar-left">
                <el-input
                        v-model="searchKeyword"
                        placeholder="搜索分享文件..."
                        :prefix-icon="Search"
                        clearable
                        style="width: 300px;"
                        @input="handleSearch"
                />
                <el-select v-model="statusFilter" placeholder="状态筛选" style="width: 120px;" @change="handleFilter">
                    <el-option label="全部" value=""/>
                    <el-option label="有效" value="active"/>
                    <el-option label="已过期" value="expired"/>
                    <el-option label="已失效" value="invalid"/>
                </el-select>
            </div>
            <div class="toolbar-right">
                <el-button :icon="Refresh" @click="refreshData">
                    刷新
                </el-button>
            </div>
        </div>

        <!-- 分享文件列表 -->
        <div class="share-content">
            <!-- 加载状态 -->
            <div v-if="loading" class="loading-container">
                <el-icon class="is-loading" :size="40">
                    <Loading/>
                </el-icon>
                <p>正在加载分享文件...</p>
            </div>

            <!-- 分享列表 -->
            <div v-else-if="filteredShares.length > 0" class="share-list">
                <div class="share-item" v-for="share in filteredShares" :key="share.id">
                    <!-- 文件信息 -->
                    <div class="file-info">
                        <div class="file-icon">
                            <el-icon :size="32" :color="getFileIconColor(share.fileType)">
                                <component :is="getFileIcon(share.fileType)"/>
                            </el-icon>
                        </div>
                        <div class="file-details">
                            <div class="file-name" :title="share.fileName">{{ share.fileName }}</div>
                            <div class="file-meta">
                                <span class="file-size">{{ formatSize(share.fileSize) }}</span>
                                <span class="share-time">分享于 {{ formatDate(share.createdAt) }}</span>
                            </div>
                        </div>
                    </div>

                    <!-- 分享状态 -->
                    <div class="share-status">
                        <el-tag
                                :type="getStatusType(share.status)"
                                :effect="share.status === 'active' ? 'light' : 'plain'"
                        >
                            {{ getStatusText(share) }}
                        </el-tag>
                    </div>

                    <!-- 下载统计 -->
                    <div class="download-stats">
                        <div class="download-count">
                            <el-icon :size="16" color="#64748b">
                                <Download/>
                            </el-icon>
                            <span>{{ share.downloadCount }} 次下载</span>
                        </div>
                    </div>

                    <!-- 操作按钮 -->
                    <div class="share-actions">
                        <el-button
                                size="small"
                                type="primary"
                                link
                                @click="copyShareLink(share)"
                                :disabled="share.status !== 'active'"
                        >
                            <el-icon>
                                <Link/>
                            </el-icon>
                            复制链接
                        </el-button>
                        <el-dropdown @command="cmd => handleShareCommand(share, cmd)">
                            <el-button size="small" type="primary" link>
                                更多
                                <el-icon>
                                    <ArrowDown/>
                                </el-icon>
                            </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="detail">
                                        <el-icon>
                                            <View/>
                                        </el-icon>
                                        查看详情
                                    </el-dropdown-item>
                                    <el-dropdown-item command="edit" :disabled="share.status !== 'active'">
                                        <el-icon>
                                            <Edit/>
                                        </el-icon>
                                        编辑分享
                                    </el-dropdown-item>
                                    <el-dropdown-item command="qrcode">
                                        <el-icon>
                                            <QrCode/>
                                        </el-icon>
                                        二维码
                                    </el-dropdown-item>
                                    <el-dropdown-item divided command="cancel" :disabled="share.status !== 'active'">
                                        <el-icon>
                                            <Close/>
                                        </el-icon>
                                        取消分享
                                    </el-dropdown-item>
                                    <el-dropdown-item command="delete">
                                        <el-icon>
                                            <Delete/>
                                        </el-icon>
                                        删除记录
                                    </el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </div>
                </div>
            </div>

            <!-- 空状态 -->
            <div v-else class="empty-state">
                <div class="empty-icon">
                    <el-icon :size="80" color="#c0c4cc">
                        <Share/>
                    </el-icon>
                </div>
                <h3>暂无分享文件</h3>
                <p v-if="searchKeyword">没有找到包含 "{{ searchKeyword }}" 的分享文件</p>
                <p v-else>您还没有分享任何文件，可以在文件列表中点击分享按钮创建分享</p>
            </div>
        </div>

        <!-- 分享详情对话框 -->
        <el-dialog
                v-model="shareDetailVisible"
                title="分享详情"
                width="600px"
                :close-on-click-modal="false"
        >
            <div v-if="currentShare" class="share-detail">
                <div class="detail-section">
                    <h4>文件信息</h4>
                    <div class="detail-item">
                        <span class="label">文件名：</span>
                        <span class="value">{{ currentShare.fileName }}</span>
                    </div>
                    <div class="detail-item">
                        <span class="label">文件大小：</span>
                        <span class="value">{{ formatSize(currentShare.fileSize) }}</span>
                    </div>
                    <div class="detail-item">
                        <span class="label">文件类型：</span>
                        <span class="value">{{ currentShare.fileType }}</span>
                    </div>
                </div>

                <div class="detail-section">
                    <h4>分享信息</h4>
                    <div class="detail-item">
                        <span class="label">分享链接：</span>
                        <div class="link-container">
                            <el-input
                                    :model-value="currentShare.shareUrl"
                                    readonly
                                    class="link-input"
                            />
                            <el-button type="primary" @click="copyShareLink(currentShare)">
                                复制
                            </el-button>
                        </div>
                    </div>
                    <div class="detail-item">
                        <span class="label">提取码：</span>
                        <span class="value code">{{ currentShare.extractCode || '无需提取码' }}</span>
                    </div>
                    <div class="detail-item">
                        <span class="label">分享时间：</span>
                        <span class="value">{{ formatDate(currentShare.createdAt) }}</span>
                    </div>
                    <div class="detail-item">
                        <span class="label">过期时间：</span>
                        <span class="value">{{ getExpiryText(currentShare) }}</span>
                    </div>
                    <div class="detail-item">
                        <span class="label">分享状态：</span>
                        <el-tag :type="getStatusType(currentShare.status)">
                            {{ getStatusText(currentShare) }}
                        </el-tag>
                    </div>
                </div>

                <div class="detail-section">
                    <h4>访问统计</h4>
                    <div class="stats-grid">
                        <div class="stat-card">
                            <div class="stat-icon">
                                <el-icon :size="24" color="#10b981">
                                    <View/>
                                </el-icon>
                            </div>
                            <div class="stat-info">
                                <div class="stat-value">{{ currentShare.viewCount || 0 }}</div>
                                <div class="stat-name">查看次数</div>
                            </div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-icon">
                                <el-icon :size="24" color="#3b82f6">
                                    <Download/>
                                </el-icon>
                            </div>
                            <div class="stat-info">
                                <div class="stat-value">{{ currentShare.downloadCount || 0 }}</div>
                                <div class="stat-name">下载次数</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <template #footer>
                <el-button @click="shareDetailVisible = false">关闭</el-button>
                <el-button type="primary" @click="copyShareLink(currentShare)">
                    复制链接
                </el-button>
            </template>
        </el-dialog>

        <!-- 二维码对话框 -->
        <el-dialog
                v-model="qrcodeVisible"
                title="分享二维码"
                width="400px"
                :close-on-click-modal="false"
        >
            <div class="qrcode-container">
                <div class="qrcode-placeholder">
                    <el-icon :size="120" color="#e5e7eb">
                        <QrCode/>
                    </el-icon>
                    <p>二维码生成功能开发中...</p>
                </div>
                <div class="qrcode-info">
                    <p>扫描二维码即可访问分享文件</p>
                    <el-input :model-value="currentShare?.shareUrl" readonly/>
                </div>
            </div>
            <template #footer>
                <el-button @click="qrcodeVisible = false">关闭</el-button>
                <el-button type="primary">保存二维码</el-button>
            </template>
        </el-dialog>

        <!-- 取消分享对话框 -->
        <el-dialog
                v-model="cancelDialogVisible"
                title="确认取消分享"
                width="600px"
                :before-close="handleCancelDialogClose"
        >
            <div class="cancel-confirm-text">
                <p>确定要取消分享文件<strong>"{{currentShare?.fileName}}"</strong> 吗？</p>
                <p style="color: #f56c6c; font-weight: bold;">取消后分享链接将立即失效。</p>
            </div>
            <template #footer>
                <el-button @click="handleCancelDialogClose" :disabled="canceling">取消</el-button>
                <el-button type="warning" @click="confirmCancel" :loading="canceling">确定取消</el-button>
            </template>
        </el-dialog>
        <!-- 删除记录对话框 -->
        <el-dialog
            v-model="deleteDialogVisible"
            title="确认删除记录"
            width="600px"
            :before-close="handleDeleteDialogClose"
        >
            <div class="delete-confirm-text">
                <p>确定要删除分享记录 <strong>"{{ shareToDelete?.fileName }}"</strong> 吗？</p>
                <p style="color: #f56c6c; font-weight: bold;">此操作不可恢复。</p>
            </div>
            <template #footer>
                <el-button @click="handleDeleteDialogClose" :disabled="deleting">取消</el-button>
                <el-button type="danger" @click="confirmDelete" :loading="deleting">确定删除</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import {computed, onMounted, ref} from 'vue'
import {ElMessage} from 'element-plus'
import {
    ArrowDown,
    Close,
    Delete,
    Document,
    Download,
    Edit,
    Files,
    Headset,
    Link,
    Picture,
    QrCode,
    Refresh,
    Search,
    Share,
    VideoCamera,
    View
} from '@element-plus/icons-vue'
import {cancelShare, deleteShare, getUserShares} from '@/api/share'

// 响应式数据
const loading = ref(false)
const searchKeyword = ref('')
const statusFilter = ref('')
const shares = ref([])
const shareDetailVisible = ref(false)
const qrcodeVisible = ref(false)
const currentShare = ref(null)

const cancelDialogVisible = ref(false)
const canceling = ref(false) // 取消操作的加载状态

const deleteDialogVisible = ref(false)
const deleting = ref(false)  // 删除操作的加载状态
const shareToDelete = ref(null) // 存储待删除的分享对象

// 统计数据
const totalShares = computed(() => shares.value.length)
const activeShares = computed(() => shares.value.filter(s => s.status === 'active').length)
const totalDownloads = computed(() => shares.value.reduce((sum, s) => sum + (s.downloadCount || 0), 0))

// 过滤后的分享列表
const filteredShares = computed(() => {
    let result = [...shares.value]

    // 搜索过滤
    if (searchKeyword.value) {
        result = result.filter(share =>
            share.fileName.toLowerCase().includes(searchKeyword.value.toLowerCase())
        )
    }

    // 状态过滤
    if (statusFilter.value) {
        result = result.filter(share => share.status === statusFilter.value)
    }

    // 按创建时间倒序排列
    return result.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))
})

// 加载分享数据
const loadShares = async () => {
    loading.value = true
    try {
        const response = await getUserShares()
        console.log(response)

        // 处理正常响应
        if (response && response.code === 200) {
            const shareData = Array.isArray(response.data) ? response.data : []
            shares.value = shareData.map(share => ({
                ...share,
                createdAt: new Date(share.createdAt),
                expiresAt: share.expiresAt ? new Date(share.expiresAt) : null
            }))
        }
        // 处理直接返回的数组（404情况下的空数组）
        else if (Array.isArray(response)) {
            shares.value = response.map(share => ({
                ...share,
                createdAt: new Date(share.createdAt),
                expiresAt: share.expiresAt ? new Date(share.expiresAt) : null
            }))
        }
        // 其他情况设为空数组
        else {
            shares.value = []
        }
    } catch (error) {
        console.error('加载分享数据失败:', error)
        // 404错误已经在API层面处理，这里只处理其他错误
        if (error.response && error.response.status !== 404) {
            ElMessage.error('网络错误，请稍后重试')
        }
        shares.value = []
    } finally {
        loading.value = false
    }
}

// 获取文件图标
const getFileIcon = (type) => {
    const iconMap = {
        image: Picture,
        video: VideoCamera,
        audio: Headset,
        document: Document
    }
    return iconMap[type] || Files
}

// 获取文件图标颜色
const getFileIconColor = (type) => {
    const colorMap = {
        image: '#f59e0b',
        video: '#ef4444',
        audio: '#8b5cf6',
        document: '#06b6d4'
    }
    return colorMap[type] || '#6b7280'
}

// 获取状态类型
const getStatusType = (status) => {
    const typeMap = {
        active: 'success',
        expired: 'warning',
        invalid: 'danger'
    }
    return typeMap[status] || 'info'
}

// 获取状态文本
const getStatusText = (share) => {
    if (share.status === 'active') {
        if (!share.expiresAt) {
            return '永久有效'
        }
        const now = new Date()
        const expires = new Date(share.expiresAt)
        const diffDays = Math.ceil((expires - now) / (1000 * 60 * 60 * 24))
        if (diffDays > 0) {
            return `${diffDays}天后过期`
        } else {
            return '已过期'
        }
    } else if (share.status === 'expired') {
        return '已过期'
    } else if (share.status === 'invalid') {
        return '已失效'
    }
    return '未知状态'
}

// 获取过期时间文本
const getExpiryText = (share) => {
    if (!share.expiresAt) {
        return '永久有效'
    }
    return formatDate(share.expiresAt)
}

// 格式化文件大小
const formatSize = (bytes) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化日期
const formatDate = (date) => {
    return new Date(date).toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    })
}

// 复制分享链接
const copyShareLink = async (share) => {
    try {
        const text = share.extractCode
            ? `${share.shareUrl} 提取码: ${share.extractCode}`
            : share.shareUrl

        await navigator.clipboard.writeText(text)
        ElMessage.success('分享链接已复制到剪贴板')
    } catch (error) {
        console.error('复制失败:', error)
        ElMessage.error('复制失败，请手动复制')
    }
}

// 处理分享操作
const handleShareCommand = (share, command) => {
    currentShare.value = share // 设置当前操作的分享对象

    switch (command) {
        case 'detail':
            shareDetailVisible.value = true
            break
        case 'edit':
            ElMessage.info('编辑分享功能开发中...')
            break
        case 'qrcode':
            qrcodeVisible.value = true
            break
        case 'cancel':
            cancelDialogVisible.value = true
            break
        case 'delete':
            shareToDelete.value = share
            deleteDialogVisible.value = true
            break
    }
}

// 取消分享
const confirmCancel = async () => {
    if (!currentShare.value) return

    canceling.value = true
    try {
        await cancelShare(currentShare.value.id)

        // 更新状态
        currentShare.value.status = 'invalid'
        ElMessage.success('分享已取消')
        cancelDialogVisible.value = false
    } catch (error) {
        console.error('取消分享失败:', error)
    } finally {
        canceling.value = false
    }
}

const handleCancelDialogClose = () => {
    // 确保关闭时重置 currentShare
    currentShare.value = null
    cancelDialogVisible.value = false
}

// 删除分享记录
const confirmDelete = async () => {
    if (!shareToDelete.value) return

    deleting.value = true
    try {
        await deleteShare(shareToDelete.value.id)

        // 从前端列表移除
        shares.value = shares.value.filter(s => s.id !== shareToDelete.value.id)
        ElMessage.success('分享记录已删除')
        deleteDialogVisible.value = false
    } catch (error) {
        // 拦截器已经显示了错误消息，这里无需再次调用 ElMessage.error
        console.error('删除失败:', error)
    } finally {
        deleting.value = false
        // 无论成功失败，重置待删除对象
        shareToDelete.value = null
    }
}

const handleDeleteDialogClose = () => {
    // 确保关闭时重置待删除对象
    shareToDelete.value = null
    deleteDialogVisible.value = false
}

// 刷新数据
const refreshData = () => {
    loadShares()
}

onMounted(() => {
    loadShares()
})
</script>

<style scoped>
.shared-files {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: #f8fafc;
}

/* 页面头部 */
.page-header {
    background: #f8fafc;
    padding: 6px 32px;
    border-bottom: 1px solid var(--border-light);
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
    width: 40px;
    height: 40px;
    background: #e2e8f0;
    border-radius: var(--radius-md);
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
    gap: 12px;
}

/* 分享内容 */
.share-content {
    flex: 1;
    background: white;
    overflow: auto;
}

.loading-container {
    padding: 80px;
    text-align: center;
    color: #909399;
}

.loading-container p {
    margin-top: 16px;
    font-size: 16px;
}

/* 分享列表 */
.share-list {
    padding: 24px;
}

.share-item {
    display: flex;
    align-items: center;
    padding: 20px;
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    margin-bottom: 16px;
    transition: all 0.3s ease;
}

.share-item:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    border-color: #d1d5db;
}

.file-info {
    display: flex;
    align-items: center;
    gap: 16px;
    flex: 1;
    min-width: 0;
}

.file-icon {
    flex-shrink: 0;
}

.file-details {
    min-width: 0;
    flex: 1;
}

.file-name {
    font-size: 16px;
    font-weight: 500;
    color: #1f2937;
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.file-meta {
    display: flex;
    gap: 16px;
    font-size: 14px;
    color: #6b7280;
}

.share-status {
    margin: 0 24px;
}

.download-stats {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0 24px;
    min-width: 120px;
}

.download-count,
.view-count {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
    color: #6b7280;
}

.share-actions {
    display: flex;
    align-items: center;
    gap: 8px;
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

/* 分享详情对话框 */
.share-detail {
    max-height: 500px;
    overflow-y: auto;
}

.detail-section {
    margin-bottom: 24px;
}

.detail-section h4 {
    font-size: 16px;
    font-weight: 600;
    color: #1f2937;
    margin: 0 0 16px 0;
    padding-bottom: 8px;
    border-bottom: 1px solid #e5e7eb;
}

.detail-item {
    display: flex;
    align-items: center;
    margin-bottom: 12px;
}

.detail-item .label {
    width: 80px;
    font-size: 14px;
    color: #6b7280;
    flex-shrink: 0;
}

.detail-item .value {
    font-size: 14px;
    color: #1f2937;
}

.detail-item .value.code {
    font-family: 'Courier New', monospace;
    background: #f3f4f6;
    padding: 2px 6px;
    border-radius: 4px;
}

.link-container {
    display: flex;
    gap: 8px;
    flex: 1;
}

.link-input {
    flex: 1;
}

.stats-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
}

.stat-card {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    background: #f9fafb;
    border-radius: 8px;
}

.stat-info {
    flex: 1;
}

.stat-value {
    font-size: 20px;
    font-weight: 600;
    color: #1f2937;
    line-height: 1;
}

.stat-name {
    font-size: 12px;
    color: #6b7280;
    margin-top: 2px;
}

/* 二维码对话框 */
.qrcode-container {
    text-align: center;
}

.qrcode-placeholder {
    padding: 40px;
    background: #f9fafb;
    border-radius: 8px;
    margin-bottom: 20px;
}

.qrcode-placeholder p {
    margin: 16px 0 0 0;
    color: #6b7280;
}

.qrcode-info p {
    margin: 0 0 12px 0;
    font-size: 14px;
    color: #6b7280;
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

    .toolbar {
        padding: 16px;
        flex-direction: column;
        gap: 16px;
        align-items: stretch;
    }

    .toolbar-right {
        justify-content: space-between;
    }

    .share-list {
        padding: 16px;
    }

    .share-item {
        flex-direction: column;
        align-items: stretch;
        gap: 16px;
        padding: 16px;
    }

    .file-info {
        gap: 12px;
    }

    .download-stats {
        flex-direction: row;
        margin: 0;
        min-width: auto;
    }

    .share-actions {
        justify-content: flex-end;
    }

    .stats-grid {
        grid-template-columns: 1fr;
    }
}
</style>
