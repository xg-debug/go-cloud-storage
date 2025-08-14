import request from '@/utils/request'

export const getRecentFiles = (limit = 20) => {
    return request({
        url: `/recent-file/${limit}`,
        method: 'get'
    })
}

