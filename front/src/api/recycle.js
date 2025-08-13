import request from "@/utils/request";

// 加载回收站软删除的文件
export const loadSoftDeletedFiles = () => {
    return request.get('/recycle')
}

// 彻底删除回收站中的某个文件
export const deletePermanent = (fileId) => {
    return request.delete(`/recycle/${fileId}`)
}

// 批量删除
export const deleteSelected = (idList) => {
    return request({
        url: "/recycle/batch",
        method: "DELETE",
        data: idList
    })
}

// 清空回收站: 根据用户Id删除
export const clearRecycleBin = () => {
    return request.delete('/recycle')
}

// 恢复回收站中的某个文件
export const restore = (fileId) => {
    return request.put(`/recycle/${fileId}/restore`)
}

// 批量恢复文件
export const restoreBatch = (idList) => {
    return request({
        url: "/recycle/batch",
        method: "PUT",
        data: idList
    })
}

// 恢复所有文件：根据用户Id恢复
export const restoreAll = () => {
    return request.put("/recycle/restore/all")
}
