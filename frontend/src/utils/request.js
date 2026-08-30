import axios from 'axios'
import {ElMessage} from 'element-plus'
import router from '@/router'
import store from '@/store'
import { API_BASE_URL } from '@/config/runtime'
import { refreshSession } from '@/services/authSession'

const service = axios.create({
    baseURL: API_BASE_URL,
    timeout: 30 * 60 * 1000, // 30分钟超时，适合大文件上传
    withCredentials: true, // 允许携带 Cookie
})

const unsafeMethods = new Set(['post', 'put', 'patch', 'delete'])

function getCookie(name) {
    const prefix = `${name}=`
    return document.cookie
        .split(';')
        .map(v => v.trim())
        .find(v => v.startsWith(prefix))
        ?.slice(prefix.length) || ''
}

// 请求拦截器：认证走 HttpOnly Cookie；写请求附带 CSRF Token
service.interceptors.request.use(config => {
    const method = (config.method || 'get').toLowerCase()
    if (unsafeMethods.has(method)) {
        const csrfToken = getCookie('csrf_token')
        if (csrfToken) {
            config.headers = config.headers || {}
            config.headers['X-CSRF-Token'] = csrfToken
        }
    }
    return config
}, error => Promise.reject(error))

// 响应拦截器
service.interceptors.response.use(response => {
        const res = response.data
        const originalRequest = response.config

        if (response.config.responseType === 'blob') {
            return res
        }
        if (res.code !== 200) {
            // 业务错误一律 HTTP 200 + code 区分；silentError 请求不弹 toast（如分片上传）
            if (originalRequest.silentError) {
                return Promise.reject(new Error(res.message || 'Error'))
            }
            ElMessage.error(res.message || 'Error')
            return Promise.reject(new Error(res.message || 'Error'))
        }
        return res.data
    },
    async error => {
        const originalRequest = error.config || {}

        if (error.response && error.response.status === 401 && !originalRequest._retry) {
            // 只允许尝试一次刷新
            originalRequest._retry = true

            try {
                await refreshSession()
                store.commit('setAuthenticated', true)

                const csrfToken = getCookie('csrf_token')
                if (csrfToken && unsafeMethods.has((originalRequest.method || 'get').toLowerCase())) {
                    originalRequest.headers = originalRequest.headers || {}
                    originalRequest.headers['X-CSRF-Token'] = csrfToken
                }

                // 重试原请求
                return service(originalRequest)

            } catch (refreshError) {
                // 刷新失败：彻底清除
                store.commit('clearAuth')
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
