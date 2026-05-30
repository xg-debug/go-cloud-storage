import { createStore } from "vuex";
import file from "@/store/modules/file";



const store =  createStore({
    modules: {
        file
    },
    state: {
        userInfo: null,
        token: localStorage.getItem('token') || null,
    },
    mutations: {
        setToken(state, token) {
            state.token = token
            localStorage.setItem('token', token)
        },
        setUserInfo(state, userInfo) {
            state.userInfo = userInfo
        },
        clearAuth(state) {
            state.token = null
            state.userInfo = null
            localStorage.removeItem('token')
        }
    }
})

export default store