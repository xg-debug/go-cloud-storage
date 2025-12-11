import request from '@/utils/request'

// Check file status (Instant upload / Resume)
export const chunkFileCheck = (fileHash) => {
    return request({
        url: '/file/chunk/check',
        method: 'post',
        data: { fileHash }
    })
}

// Upload a chunk
export const chunkFileUpload = (data, onUploadProgress) => {
    return request({
        url: '/file/chunk/upload',
        method: 'post',
        data,
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress
    })
}

// Merge chunks
export const mergeChunkFile = (data) => {
    return request({
        url: '/file/chunk/merge',
        method: 'post',
        data
    })
}

// Pause upload (Just to notify backend if needed, or client side only)
// The backend might not need an explicit pause if we just stop sending chunks, 
// but maybe we want to update status.
export const pauseChunkUpload = (data) => {
   // Client side pause doesn't necessarily need a backend call unless we track status in DB.
   // But if the frontend calls it, we should provide it.
   return Promise.resolve()
}

export const resumeChunkUpload = (data) => {
    // Re-check is enough usually
    return chunkFileCheck(data.fileHash)
}

export const cancelChunkUpload = (data) => {
    return request({
        url: '/file/chunk/cancel',
        method: 'post',
        data
    })
}

export const getLoginInfo = () => {
    // Helper to check login status, maybe calls /me or just checks token
    return Promise.resolve() 
}
