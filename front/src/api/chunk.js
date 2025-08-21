import request from '@/utils/request'

// 初始化分片上传
export function initUpload(data) {
  return request({
    url: '/api/upload/init',
    method: 'post',
    data
  })
}

// 上传分片
export function uploadChunk(data, onUploadProgress) {
  return request({
    url: '/api/upload/chunk',
    method: 'post',
    data,
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    onUploadProgress
  })
}

// 合并分片
export function mergeChunks(data) {
  return request({
    url: '/api/upload/merge',
    method: 'post',
    data
  })
}

// 获取已上传的分片
export function getUploadedChunks(fileId) {
  return request({
    url: '/api/upload/chunks',
    method: 'get',
    params: { fileId }
  })
}

// 检查文件是否存在
export function checkFileExists(params) {
  return request({
    url: '/api/upload/check',
    method: 'get',
    params
  })
}