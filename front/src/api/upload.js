import request from '@/utils/request'

export const getLoginInfo = () =>{
    return request({
        url: '/api/user/get/login',
        method: 'get'
    })
}

export const upload = (data) =>{
    return request({
        url: '/api/upload',
        method: 'post',
        data
    })
}

/**
 * 检查文件上传状态
 * @param {Object} data
 * @returns
 */
export const chunkFileCheck  = (fileHash) =>{
    return request({
        url: '/api/file/check/'+fileHash,
        method: 'post',
    })
}

/**
 * 分片文件上传
 * @param {Object} data
 * @param {Function} onUploadProgress
 * @returns
 */
export const chunkFileUpload = (data,onUploadProgress) =>{
    return request({
        url: '/api/chunk/upload',
        method: 'post',
        data,
        headers: {
            'Content-Type': 'multipart/form-data'
        },
        onUploadProgress
    })
}

/**
 * 合并分片文件
 * @param {Object} data
 * @returns
 */
export const mergeChunkFile = (data) =>{
    return request({
        url: '/api/chunk/merge',
        method: 'post',
        data
    })
}

/**
 * 取消分片上传
 * @param {Object} data
 * @returns
 */
export const cancelChunkUpload = (data) =>{
    return request({
        url: '/api/chunk/cancel',
        method: 'post',
        data
    })
}

/**
 * 暂停分片上传
 * @param {Object} data
 * @returns
 */
export const pauseChunkUpload = (data) =>{
    return request({
        url: '/api/chunk/pause',
        method: 'post',
        data
    })
}

/**
 * 继续分片上传
 * @param {Object} data
 * @returns
 */
export const resumeChunkUpload = (data) =>{
    return request({
        url: '/api/chunk/resume',
        method: 'post',
        data
    })
}
