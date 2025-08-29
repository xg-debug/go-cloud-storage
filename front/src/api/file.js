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

// 上传文件(使用FormData)
export const uploadFile = (formData, onUploadProgress) => {
    return request({
        url: '/file/upload',
        method: 'post',
        data: formData,
        headers: {'Content-Type': 'multipart/form-data'},
        timeout: 30 * 60 * 1000, // 30分钟超时
        onUploadProgress
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