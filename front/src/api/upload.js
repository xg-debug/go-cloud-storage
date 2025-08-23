import request from '@/utils/request'

// 创建上传任务
export function createUploadTask(data) {
    return request({
        url: '/upload-tasks',
        method: 'post',
        data
    })
}

// 获取上传任务详情（断点续传用）
export function getUploadTask(taskId) {
    return request({
        url: `/upload-tasks/${taskId}`,
        method: 'get'
    })
}

// 获取分片预签名 URL
export function getChunkURL(taskId, partNumber) {
    return request({
        url: `/upload-tasks/${taskId}/chunks/${partNumber}/url`,
        method: 'get'
    })
}

// 标记分片已上传
export function markChunkUploaded(taskId, partNumber, data) {
    return request({
        url: `/upload-tasks/${taskId}/chunks/${partNumber}`,
        method: 'patch',
        data
    })
}

// 完成上传
export function completeUpload(taskId) {
    return request({
        url: `/upload-tasks/${taskId}/complete`,
        method: 'post'
    })
}

// 获取未完成的上传任务
export function getIncompleteTasks() {
    return request({
        url: '/upload-tasks/incomplete',
        method: 'get'
    })
}

// 删除上传任务
export function deleteUploadTask(taskId) {
    return request({
        url: `/upload-tasks/${taskId}`,
        method: 'delete'
    })
}

// 检查文件是否存在（秒传）
export function checkFileExists(fileHash) {
    return request({
        url: `/files/check-exists`,
        method: 'post',
        data: { fileHash }
    })
}
