import request from "@/utils/request";

// 获取收藏列表
export const getFavorites = (page, pageSize) => {
    return request ({
        url: "/favorite",
        method: "get",
        params: { page, pageSize }
    })
}

// 收藏
export const addFavorite = (fileId) => {
    return request({
        url: `/favorite/${fileId}`,
        method: "post"
    })
}

// 取消收藏
export const cancelFavorite = (fileId) => {
    return request({
        url: `/favorite/${fileId}`,
        method: "delete"
    })
}