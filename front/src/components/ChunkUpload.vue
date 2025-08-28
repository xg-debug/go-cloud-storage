<template>
    <div class="upload-container">
        <!-- 上传区域 -->
        <div class="upload-section">
            <el-upload
                ref="uploadRef"
                action=""
                :auto-upload="false"
                :show-file-list="false"
                :before-upload="beforeUpload"
                :on-remove="handleRemove"
                :on-change="handleChange"
                class="upload-dragger"
            >
                <template #trigger>
                    <div class="upload-trigger">
                        <div class="upload-icon">
                            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                            </svg>
                        </div>
                        <div class="upload-text">
                            <p class="upload-title">点击选择文件</p>
                            <p class="upload-desc">支持大文件上传，最大2G</p>
                        </div>
                    </div>
                </template>
            </el-upload>

            <div class="upload-actions">
                <el-button
                    type="primary"
                    size="large"
                    @click="submitUpload"
                    :disabled="!fileList.length"
                    class="upload-btn"
                >
                    <span class="btn-icon">↑</span>
                    开始上传
                </el-button>
            </div>
        </div>

        <!-- 待上传文件列表 -->
        <div v-if="fileList.length" class="file-queue">
            <div class="section-header">
                <h3>待上传文件</h3>
                <span class="file-count">{{ fileList.length }} 个文件</span>
            </div>
            <div class="file-list">
                <div v-for="item in fileList" :key="item.uid" class="file-item pending">
                    <div class="file-info">
                        <div class="file-icon">📄</div>
                        <div class="file-details">
                            <div class="file-name">{{ item.name }}</div>
                            <div class="file-size">{{ formatFileSize(item.size) }}</div>
                        </div>
                    </div>
                    <div class="file-status">
                        <span class="status-badge pending">等待上传</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- 正在上传文件列表 -->
        <div v-if="uploadingFiles.length" class="uploading-section">
            <div class="section-header">
                <h3>上传进度</h3>
                <span class="file-count">{{ uploadingFiles.length }} 个文件</span>
            </div>
            <div class="uploading-list">
                <div v-for="item in uploadingFiles" :key="item.uid" class="upload-item">
                    <div class="upload-header">
                        <div class="file-info">
                            <div class="file-icon">📄</div>
                            <div class="file-details">
                                <div class="file-name">{{ item.name }}</div>
                                <div class="file-meta">
                                    <span class="file-size">{{ formatFileSize(item.size) }}</span>
                                    <span v-if="item.status === 'uploading'" class="upload-speed">{{ item.speed }}</span>
                                    <span v-if="item.status === 'uploading'"  class="remaining-time">预计剩余 {{ item.remainingTime }}</span>
                                </div>
                            </div>
                        </div>
                        <div class="upload-controls">
                            <el-button
                                v-if="item.status === 'uploading' && item.status != 'mergeing'"
                                @click="pauseUpload(item)"
                                size="small"
                                type="warning"
                                class="control-btn"
                            >
                                ⏸ 暂停
                            </el-button>
                            <el-button
                                v-if="item.status === 'paused'"
                                @click="resumeUpload(item)"
                                size="small"
                                type="success"
                                class="control-btn"
                            >
                                ▶ 继续
                            </el-button>
                            <el-button
                                v-if="item.status === 'paused'"
                                @click="cancelUpload(item)"
                                size="small"
                                type="danger"
                                class="control-btn"
                            >
                                ✕ 取消
                            </el-button>
                            <el-button
                                v-if="item.status === 'success'"
                                @click="openFile(item)"
                                size="small"
                                type="danger"
                                class="control-btn open-btn"
                            >
                                📂 打开
                            </el-button>
                        </div>
                    </div>

                    <div class="progress-section">
                        <div class="progress-info">
                            <span class="progress-text">{{ item.percentage }}%</span>
                            <span class="status-badge" :class="item.status">
                {{ getStatusText(item.status) }}
              </span>
                        </div>
                        <el-progress
                            :status="getProgressStatus(item.status)"
                            :stroke-width="8"
                            :percentage="item.percentage"
                            :show-text="false"
                            class="custom-progress"
                        />
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { chunkFileCheck, chunkFileUpload, mergeChunkFile, getLoginInfo, cancelChunkUpload, pauseChunkUpload, resumeChunkUpload } from '@/api/upload'
import { FileUploadUtils } from '@/utils/fileUpload'
import { ElMessage } from 'element-plus'

const fileList = ref([])
const uploadingFiles = ref([])

// 文件上传工具类
const fileUploadUtils = new FileUploadUtils()

// 格式化文件大小
const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 获取状态文本
const getStatusText = (status) => {
    const statusMap = {
        uploading: '上传中',
        paused: '已暂停',
        success: '上传成功',
        exception: '上传失败',
        mergeing: '合并中'
    }
    return statusMap[status] || '未知状态'
}

// 获取进度条状态
const getProgressStatus = (status) => {
    const statusMap = {
        uploading: '',
        paused: 'warning',
        success: 'success',
        exception: 'exception'
    }
    return statusMap[status] || ''
}

const beforeUpload = (file) => {
    console.log(file)
}

const handleChange = (file) => {
    let flag = fileUploadUtils.checkFileType(file, [])
    if (!flag) {
        return ElMessage.warning('文件格式不正确')
    }
    if (!fileUploadUtils.checkFileSize(file, 2000 * 1024 * 1024)) {
        return ElMessage.warning('文件大小不能超过2G')
    }
    fileList.value.push(file)
}

const handleRemove = (file) => {
    console.log(file)
}

const submitUpload = async () => {
    // 校验是否已登录
    await getLoginInfo()
    if (!fileList.value.length) {
        ElMessage.warning('请选择文件')
        return
    }
    console.log(fileList.value)
    const uploadTasks = fileList.value.map((file) => createUploadTask(file.raw))
    console.log(uploadTasks)

    await Promise.all(uploadTasks)
}

const openFile = (file) => {
    window.open(file.path)
}

// 创建上传任务
const createUploadTask = async (file) => {
    const uploadFile = reactive({
        uid: file.uid,
        name: file.name,
        size: file.size,
        percentage: 0,
        status: 'uploading',
        speed: '0 kb/s',
        remainingTime: '--',
        uploadedChunks: [], // 已上传的分片
        totalChunks: 0, // 总分片数
        fileHash: '', // 文件hash
        chunks: [], // 分片列表
        startTime: new Date(), // 开始时间
        endTime: null, // 结束时间
        uploadedBytes: 0, // 已上传的字节数
        isPaused: false, // 是否暂停
        abortController: new AbortController(), // 中止控制器
        path: '', // 文件上传成功的路径
        chunkSize:1024 * 1024, // 分片大小,默认1MB
    })
    uploadingFiles.value.push(uploadFile)
    fileList.value = fileList.value.filter((item) => item.uid !== file.uid)

    try {
        // 计算分片大小
        const chunkSize = fileUploadUtils.calculateDynamicChunkSize(file.size)
        console.log('chunkSize',chunkSize);
        uploadFile.chunkSize = chunkSize

        // 计算文件hash
        uploadFile.fileHash = await fileUploadUtils.calculateFileHash(file, chunkSize)

        // 创建分片
        uploadFile.chunks = fileUploadUtils.createFileChunks(file, chunkSize)
        uploadFile.totalChunks = uploadFile.chunks.length

        // 检查文件上传状态
        const res = await chunkFileCheck(uploadFile.fileHash)
        // console.log(res)
        // 文件已完全上传
        if (res.status === 'completed') {
            console.log('文件已完全上传', res.url)
            uploadFile.percentage = 100
            uploadFile.status = 'success'
            uploadFile.path = res.url
            console.log(uploadFile)
            return
        }
        if (res.status === 'uploading') {
            console.log('文件部分已上传')
            uploadFile.uploadedChunks = res.uploadedChunks || []
        }

        // 上传分片
        await uploadFileChunks(uploadFile)
        if(uploadFile.isPaused) return
        uploadFile.status = 'mergeing'
        // 合并分片文件
        const mergeRes = await mergeChunkFile({
            fileHash: uploadFile.fileHash,
            fileName: uploadFile.name,
            totalChunks: uploadFile.totalChunks,
            biz: 'big_file',
        })

        uploadFile.percentage = 100
        uploadFile.status = 'success'
        uploadFile.path = mergeRes
        uploadFile.endTime = new Date()
        console.log(mergeRes)
    } catch (error) {
        uploadFile.status = 'exception'
        console.log(error)
    }
}

// 上传分片
const uploadFileChunks = async (uploadFile) => {
    const concurrentLimit = 3 // 并发限制
    const pendingChunks = [] // 待上传的分片索引

    // 将未上传的分片索引加入待上传队列
    for (let i = 0; i < uploadFile.chunks.length; i++) {
        if (!uploadFile.uploadedChunks.includes(i)) {
            pendingChunks.push(i)
        }
    }

    // 并发上传
    const uploadPromises = []
    for (let i = 0; i < Math.min(concurrentLimit, pendingChunks.length); i++) {
        uploadPromises.push(uploadChunkWorker(uploadFile, pendingChunks))
    }
    await Promise.all(uploadPromises)
}

// 更新上传进度
const updateUploadProgress = (uploadFile, progressEvent) => {
    // 计算上次进度
    const chunkProgress = progressEvent.loaded / progressEvent.total // 计算分片进度
    const completedChunks = uploadFile.uploadedChunks.length // 已完成的分片数
    const totalChunks = uploadFile.totalChunks // 总分片数
    const totalProgress = Math.round(((completedChunks + chunkProgress) / totalChunks) * 100)
    console.log('上传总进度：', totalProgress + '%')
    uploadFile.percentage = totalProgress // 更新文件的上传进度字段

    // 计算上传速度
    const currentTime = new Date()
    const elapsedTime = (currentTime - uploadFile.startTime) / 1000 // 计算已用时间(秒)
    const uploadedBytes = completedChunks * uploadFile.chunkSize + progressEvent.loaded // 计算已上传的总字节数
    const uploadSpeed = uploadedBytes / elapsedTime // 计算上传速度(bytes/s)

    console.log('上传速度：', fileUploadUtils.formatFileSize(uploadSpeed) + '/s')
    uploadFile.speed = fileUploadUtils.formatFileSize(uploadSpeed) + '/s'

    // 计算剩余时间
    if (uploadSpeed > 0) {
        const remainingBytes = uploadFile.size - uploadedBytes // 计算剩余字节数
        const remainingTime = Math.round(remainingBytes / uploadSpeed) // 计算剩余时间(秒)
        console.log('预计剩余时间：', formatTime(remainingTime))
        uploadFile.remainingTime = formatTime(remainingTime)
    }
}

// 格式化时间
const formatTime = (seconds) => {
    if (seconds < 60) {
        return Math.round(seconds) + '秒'
    } else if (seconds < 3600) {
        return Math.round(seconds / 60) + '分钟'
    } else {
        return Math.round(seconds / 3600) + '小时'
    }
}

// 上传分片工作器
const uploadChunkWorker = async (uploadFile, pendingChunks) => {
    while (pendingChunks.length > 0 && !uploadFile.isPaused) {
        const chunkIndex = pendingChunks.shift()
        if (chunkIndex === undefined) {
            break
        }

        const chunk = uploadFile.chunks[chunkIndex]
        const formData = new FormData()
        formData.append('fileHash', uploadFile.fileHash)
        formData.append('fileName', uploadFile.name)
        formData.append('chunkIndex', chunkIndex.toString())
        formData.append('chunk', chunk.chunk)
        formData.append('totalChunks', uploadFile.totalChunks.toString())
        // TODO
        formData.append('biz', 'big_file')

        try {
            await chunkFileUpload(formData, (e) => {
                updateUploadProgress(uploadFile, e)
            })
            uploadFile.uploadedChunks.push(chunkIndex)
            // console.log( '上传成功的分片索引：', chunkIndex)
        } catch (error) {
            if (error.name === 'AbortError') {
                console.log('上传中止')
                break
            }
            pendingChunks.unshift(chunkIndex)
            console.log(error)
        }
    }
}

// 暂停上传
const pauseUpload = async (uploadFile) => {
    console.log('暂停上传');
    try {
        uploadFile.isPaused = true
        uploadFile.abortController.abort()
        uploadFile.status = 'paused'
        
        // 调用后端暂停API
        await pauseChunkUpload({
            fileHash: uploadFile.fileHash
        })
        console.log('后端暂停状态已更新')
    } catch (error) {
        console.error('暂停上传失败:', error)
    }
}

// 继续上传
const resumeUpload = async (uploadFile) => {
    console.log('继续上传');
    try {
        uploadFile.isPaused = false
        uploadFile.status = 'uploading'
        uploadFile.abortController = new AbortController()

        // 调用后端继续API
        await resumeChunkUpload({
            fileHash: uploadFile.fileHash
        })
        console.log('后端继续状态已更新')

        // 上传分片
        await uploadFileChunks(uploadFile)
        if(uploadFile.isPaused) return
        uploadFile.status = 'mergeing'
        // 合并分片文件
        const mergeRes = await mergeChunkFile({
            fileHash: uploadFile.fileHash,
            fileName: uploadFile.name,
            totalChunks: uploadFile.totalChunks,
            biz: 'big_file',
        })
        uploadFile.percentage = 100
        uploadFile.status = 'success'
        uploadFile.path = mergeRes
        uploadFile.endTime = new Date()
        console.log(mergeRes)
    } catch (error) {
        uploadFile.status = 'exception'
        console.log(error)
    }
}

// 取消上传
const cancelUpload = async (uploadFile) => {
    try {
        uploadFile.isPaused = true
        uploadFile.abortController.abort()
        
        // 调用后端取消API
        await cancelChunkUpload({
            fileHash: uploadFile.fileHash
        })
        console.log('后端取消状态已更新')
        
        // 从上传列表中移除
        const index = uploadingFiles.value.findIndex((item) => item.uid === uploadFile.uid)
        if(index > -1){
            uploadingFiles.value.splice(index, 1)
        }
    } catch (error) {
        console.error('取消上传失败:', error)
        // 即使后端调用失败，也要从前端列表中移除
        const index = uploadingFiles.value.findIndex((item) => item.uid === uploadFile.uid)
        if(index > -1){
            uploadingFiles.value.splice(index, 1)
        }
    }
}
</script>

<style scoped>
.upload-container {
    margin: 0 auto;
    padding: 16px;
    background: #f8fafc;
    max-height: 500px;
    overflow-y: auto;
}

/* 上传区域 */
.upload-section {
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 16px;
}

.upload-dragger :deep(.el-upload) {
    width: 100%;
}

.upload-trigger {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 20px;
    border: 2px dashed #d1d5db;
    border-radius: 8px;
    background: #fafafa;
    cursor: pointer;
}

.upload-trigger:hover {
    border-color: #3b82f6;
    background: #f0f9ff;
}

.upload-icon {
    width: 40px;
    height: 40px;
    background: #3b82f6;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.upload-icon svg {
    width: 24px;
    height: 24px;
    color: white;
}

.upload-text {
    text-align: left;
}

.upload-title {
    font-size: 16px;
    font-weight: 600;
    color: #1f2937;
    margin: 0 0 4px 0;
}

.upload-desc {
    font-size: 14px;
    color: #6b7280;
    margin: 0;
}

.upload-actions {
    margin-top: 16px;
    text-align: center;
}

.upload-btn {
    padding: 8px 24px;
    font-size: 14px;
    font-weight: 500;
    border-radius: 6px;
    background: #3b82f6;
    border: none;
    color: white;
}

.upload-btn:hover {
    background: #2563eb;
}

.upload-btn:disabled {
    opacity: 0.5;
    background: #9ca3af;
}

.btn-icon {
    margin-right: 6px;
    font-size: 16px;
}

/* 文件列表区域 */
.file-queue, .uploading-section {
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 16px;
    max-height: 200px;
    overflow-y: auto;
}

.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #f1f5f9;
}

.section-header h3 {
    font-size: 16px;
    font-weight: 600;
    color: #1f2937;
    margin: 0;
}

.file-count {
    background: #3b82f6;
    color: white;
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
}

/* 文件项样式 */
.file-item, .upload-item {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    padding: 12px;
    margin-bottom: 8px;
}

.file-item:hover, .upload-item:hover {
    background: #f3f4f6;
    border-color: #d1d5db;
}

.file-info {
    display: flex;
    align-items: center;
    gap: 12px;
}

.file-icon {
    font-size: 20px;
    width: 32px;
    height: 32px;
    background: #3b82f6;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.file-details {
    flex: 1;
    min-width: 0;
}

.file-name {
    font-weight: 500;
    color: #1f2937;
    margin-bottom: 2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.file-size {
    font-size: 12px;
    color: #6b7280;
}

.file-meta {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: #6b7280;
    flex-wrap: wrap;
}

.upload-speed {
    color: #059669;
    font-weight: 500;
}

.remaining-time {
    color: #dc2626;
}

/* 状态标签 */
.status-badge {
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.status-badge.pending {
    background: #fef3c7;
    color: #d97706;
}

.status-badge.uploading {
    background: #dbeafe;
    color: #2563eb;
}

.status-badge.paused {
    background: #fed7aa;
    color: #ea580c;
}

.status-badge.success {
    background: #d1fae5;
    color: #059669;
}

.status-badge.exception {
    background: #fee2e2;
    color: #dc2626;
}

/* 上传控制 */
.upload-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}

.upload-controls {
    display: flex;
    gap: 6px;
}

.control-btn {
    padding: 4px 8px !important;
    font-size: 12px !important;
    border-radius: 4px !important;
    min-width: auto !important;
}

/* 进度区域 */
.progress-section {
    margin-top: 8px;
}

.progress-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
}

.progress-text {
    font-weight: 600;
    color: #1f2937;
    font-size: 12px;
}

.custom-progress :deep(.el-progress-bar__outer) {
    background-color: #f3f4f6;
    border-radius: 6px;
    overflow: hidden;
}

.custom-progress :deep(.el-progress-bar__inner) {
    background: #3b82f6;
    border-radius: 6px;
}

.open-btn {
    background: #10b981 !important;
    border: none !important;
    color: white !important;
}

.open-btn:hover {
    background: #059669 !important;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .upload-container {
        padding: 12px;
    }

    .upload-section {
        padding: 16px;
    }

    .upload-trigger {
        flex-direction: column;
        gap: 12px;
        padding: 16px;
    }

    .upload-text {
        text-align: center;
    }

    .upload-header {
        flex-direction: column;
        gap: 8px;
        align-items: stretch;
    }

    .upload-controls {
        justify-content: center;
    }

    .file-meta {
        flex-direction: column;
        gap: 4px;
    }
}
</style>
