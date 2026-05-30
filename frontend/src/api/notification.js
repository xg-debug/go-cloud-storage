import request from '@/utils/request'

// 获取通知列表
export function getNotifications(params = {}) {
    return request({
        url: '/notification',
        method: 'get',
        params: {
            page: params.page || 1,
            page_size: params.pageSize || 20
        }
    })
}

// 获取未读通知数量
export function getUnreadCount() {
    return request({
        url: '/notification/unread-count',
        method: 'get'
    })
}

// 标记单个通知为已读
export function markAsRead(id) {
    return request({
        url: `/notification/${id}/read`,
        method: 'put'
    })
}

// 标记所有通知为已读
export function markAllAsRead() {
    return request({
        url: '/notification/read-all',
        method: 'put'
    })
}

// 删除单个通知
export function deleteNotification(id) {
    return request({
        url: `/notification/${id}`,
        method: 'delete'
    })
}

// 删除所有通知
export function deleteAllNotifications() {
    return request({
        url: '/notification/all',
        method: 'delete'
    })
}

// 创建通知（管理员接口）
export function createNotification(data) {
    return request({
        url: '/notification',
        method: 'post',
        data
    })
}