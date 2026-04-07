import request from '@/utils/request'

// 创建分享
export const createShare = (data) => {
    return request({
        url: '/share',
        method: 'post',
        data
    })
}

// 获取用户分享列表
export const getUserShares = () => {
    return request({
        url: '/share',
        method: 'get'
    })
}

// 获取分享详情
export const getShareDetail = (shareId) => {
    return request({
        url: `/share/${shareId}`,
        method: 'get'
    })
}

// 更新分享设置
export const updateShare = (shareId, data) => {
    return request({
        url: `/share/${shareId}`,
        method: 'put',
        data
    })
}

// 取消分享
export const cancelShare = (shareId) => {
    return request({
        url: `/share/${shareId}/cancel`,
        method: 'put'
    })
}

// 删除分享记录
export const deleteShare = (shareId) => {
    return request({
        url: `/share/${shareId}`,
        method: 'delete'
    })
}

// 访问分享（公开接口）
export const accessShare = (token, code) => {
    return request({
        url: `/s/${token}`,
        method: 'get',
        params: { code }
    })
}

// 下载分享文件（公开接口）
export const downloadSharedFile = (token, code) => {
    return request({
        url: `/s/${token}/download`,
        method: 'get',
        params: { code }
    })
}
