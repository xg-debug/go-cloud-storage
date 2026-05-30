export default {
    namespaced: true,
    state: {
        needRefresh: false,
        needRefreshStorage: false,
        currentParentId: ''
    },
    mutations: {
        setNeedRefresh(state, val) {
            state.needRefresh = val
        },
        setNeedRefreshStorage(state, val) {
            state.needRefreshStorage = val
        },
        setCurrentParentId(state, id) {
            state.currentParentId = id
        }
    }
}