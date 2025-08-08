import request from '@/utils/request'

/**
 * 获取当前登录用户信息（在刷新时用）
 * GET /api/me
 */
export const getProfile = () => {
    return request({
        url: '/me',
        method: 'get'
    })
}

/**
 * 更新用户信息
 */
export const updateProfile = (data) => {
    return request({
        url: '/user/update',
        method: 'put',
        data
    })
}

// 更新头像
export const uploadAvatar = (formData) => {
    return request({
        url: '/user/avatar',
        method: 'post',
        data: formData,
        headers: {
            'Content-Type': 'multipart/form-data'
        }
    })
}

// 获取用户存储统计信息
export const getUserStats = () => {
    return request({
        url: '/user/stats',
        method: 'get'
    })
}

// 修改密码
export const updatePassword = (data) => {
    return request({
        url: '/user/password',
        method: 'put',
        data
    })
}

// 获取文件类型统计
export const getFileTypeStats = () => {
    return request({
        url: '/me/file-stats',
        method: 'get'
    })
}