import { createStore } from "vuex";
import file from "@/store/modules/file";
import upload from "@/store/modules/upload";


const store =  createStore({
    modules: {
        file,
        upload
    },
    state: {
        userInfo: null,
        isAuthenticated: false,
        authChecked: false,
    },
    mutations: {
        setAuthenticated(state, value) {
            state.isAuthenticated = value
        },
        setAuthChecked(state, value) {
            state.authChecked = value
        },
        setUserInfo(state, userInfo) {
            state.userInfo = userInfo
            state.isAuthenticated = !!userInfo
        },
        clearAuth(state) {
            state.isAuthenticated = false
            state.authChecked = true
            state.userInfo = null
        }
    }
})

export default store
