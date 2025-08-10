import request from '@/utils/request'


// 获取文件分类（按类型/标签/时间）
export const getFileCategories = (categoryType) => {
    return request.get('/file/categories', {params: {type: categoryType}})
}

// 删除分类
export const delCategory = (id) => {
    return request.delete('/file/categories/${id}')
}

// 获取分类下的文件列表
export const getFilesByCategory = (categoryId) => {
    return request.get("/file/list/${categoryId}")
}

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

// 最近文件(按时间段)
export const listRecentFiles = (range = 'week') => {
    return request({
        url: '/recent',
        method: 'get',
        params: {range}
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
        onUploadProgress
    })
}