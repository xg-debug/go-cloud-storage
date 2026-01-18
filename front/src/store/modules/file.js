export default {
    namespaced: true,
    state: {
        needRefresh: false
    },
    mutations: {
        setNeedRefresh(state, val) {
            state.needRefresh = val
        }
    }
}