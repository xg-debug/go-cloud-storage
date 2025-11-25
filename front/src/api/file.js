import request from '@/utils/request'

// 文件列表：获取指定父目录下的文件/文件夹
export const listFiles = (data) => {
    return request({
        url: '/file/list',
        method: 'post',
        data
    })
}

// 新建文件夹
export const createFolder = (data) => {
    return request({
        url: '/file/create-folder',
        method: 'post',
        data
    })
}

// 收藏列表
export const lisStarred = () => {
    return request({
        url: '/star',
        method: 'get'
    })
}

// 添加收藏
export const addStarred = (fileId) => {
    return request({
        url: '/star',
        method: 'post',
        data: {fileId}
    })
}

// 取消收藏
export const removeStar = (starId) => {
    return request({
        url: '/star/${starId}',
        method: 'delete'
    })
}

// 文件预览：获取文件可预览链接：
export const previewFile = (fileId) => {
    return request({
        url: 'file/${fileId}/preview',
        method: 'get'
    })
}

// 文件下载：获取下载链接或者直接触发下载
export const downloadFile = (fileId) => {
    return request({
        url: '/file/${fileId}/download',
        method: 'get',
        responseType: 'blob'
    })
}

// 删除文件
export const deleteFile = (fileId) => {
    return request({
        url: `/file/${fileId}`,
        method: 'delete'
    })
}

// 重命名文件
export const renameFile = (fileId, newName) => {
    return request({
        url: '/file/rename',
        method: 'post',
        data: {fileId: fileId, newName: newName}
    })
}

// 获取最近文件
export const getRecentFiles = (timeRange) => {
    return request({
        url: '/file/recent',
        method: 'get',
        params: { timeRange: timeRange }
    })
}

// 获取分类下的文件列表（按文件类型：图片、视频、音频、文档）
export const getFilesByCategory = (data) => {
    return request({
        url: '/category/files',
        method: 'post',
        data
    })
}

// 搜索文件
export const searchFiles = (data) => {
    return request({
        url: '/file/search',
        method: 'post',
        data
    })
}

// 获取文件夹树结构
export const getFolderTree = () => {
    return request({
        url: '/file/folders/tree',
        method: 'get'
    })
}

// 移动文件
export const moveFile = (data) => {
    return request({
        url: '/file/move',
        method: 'post',
        data
    })
}


/**
 * 标准上传文件(使用FormData)
 * @param {FormData} formData - 包含文件和元数据的 FormData 对象
 * @param {function} onUploadProgress - 进度回调函数
 */
export const uploadFile = (formData, onUploadProgress) => {
    // formData 内部结构应该是:
    // formData.append('file', fileObject);
    // formData.append('parentId', currentParentId);
    return request({
        url: '/file/upload',
        method: 'post',
        data: formData,
        headers: {'Content-Type': 'multipart/form-data'},
        timeout: 2 * 60 * 1000, // 2分钟超时
        onUploadProgress
    })
}

/**
 * 初始化分片上传任务
 * @param {object} data - {fileHash: string, fileName: string, parentId: string, fileSize: int64}
 */
export const chunkUploadInit = (data) => request({
    url: '/file/chunk/init',
    method: 'post',
    data
})

/**
 * 上传单个分片
 * @param {FormData} formData - 包含 chunkIndex, fileHash, file Blob 的 FormData
 * @param {function} onUploadProgress - 进度回调函数
 */
export const chunkUploadPart = (formData, onUploadProgress) => {
    return request({
        url: '/file/chunk/upload',
        method: 'post',
        data: formData,
        headers: {'Content-Type': 'multipart/form-data'},
        onUploadProgress
    })
}

/**
 * 完成分片合并
 * @param {object} data - {fileHash: string, fileName: string, parentId: string, fileSize: int64}
 */
export const chunkUploadMerge = (data) => {
    return request({
        url: '/file/chunk/merge',
        method: 'post',
        data
    })
}

/**
 * 取消分片上传任务
 * @param {object} data - {fileHash: string}
 */
export const chunkUploadCancel = (fileHash) => request({
    url: '/file/chunk/cancel',
    method: 'post',
    data: { fileHash }
})