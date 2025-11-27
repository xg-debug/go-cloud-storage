import axios from 'axios'
import {ElMessage} from 'element-plus'
import router from '@/router'
import store from '@/store'

const service = axios.create({
    baseURL: 'http://localhost:8081',
    timeout: 30 * 60 * 1000, // 30分钟超时，适合大文件上传
    withCredentials: true, // 允许携带 Cookie
})

// 请求拦截器：每个请求带上 access_token
service.interceptors.request.use(config => {
    // 免token接口列表
    const whiteList = ['/login', '/register', '/refresh-token']
    if (!whiteList.includes(config.url)) {
        const token = store.state.token || localStorage.getItem('token')
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }
    }
    return config
}, error => Promise.reject(error))

// 响应拦截器
service.interceptors.response.use(response => {
        const res = response.data

        if (response.request.responseType === 'blob') {
            return res
        }
        if (res.code !== 200) {
            ElMessage.error(res.message || 'Error')
            return Promise.reject(new Error(res.message || 'Error'))
        }
        return res.data
    },
    async error => {
        const originalRequest = error.config

        if (error.response && error.response.status === 401 && !originalRequest._retry) {
            // 只允许尝试一次刷新
            originalRequest._retry = true

            try {
                // 调用刷新接口
                const refreshResponse = await axios.post('http://localhost:8081/refresh-token', {}, {
                    withCredentials: true // 如果后端把 refresh_token 放 Cookie
                })

                const res = refreshResponse.data
                if (res.code !== 200) throw new Error(res.message || '刷新 Token 失败')
                const newAccessToken = res.data.token

                // 更新前端
                store.commit('setToken', newAccessToken)
                localStorage.setItem('token', newAccessToken)

                // 重新设置 Authorization
                originalRequest.headers.Authorization = `Bearer ${newAccessToken}`

                // 重试原请求
                return service(originalRequest)

            } catch (refreshError) {
                // 刷新失败：彻底清除
                store.commit('clearAuth')
                localStorage.removeItem('token')
                ElMessage.error('登录已过期, 请重新登录')
                router.push('/login')
                return Promise.reject(refreshError)
            }

        }

        // 检查是否设置了静默处理标志
        if (originalRequest.silentError) {
            // 静默处理错误，不显示错误消息
            return Promise.reject(error)
        }

        // 对于404错误，显示更友好的提示
        if (error.response && error.response.status === 404) {
            ElMessage.warning('暂无数据')
        } else {
            ElMessage.error(error.message || '请求错误')
        }

        return Promise.reject(error)
    })

// 自定义 request 函数，支持额外配置（如 onUploadProgress）
const request = ({url, method, data, params, headers, onUploadProgress, ...rest}) => {
    return service({
        url,
        method,
        data,
        params,
        headers,
        onUploadProgress, // 传递 onUploadProgress
        ...rest, // 支持其他 axios 配置（如 signal）
    });
};

export default service
