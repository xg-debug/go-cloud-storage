import request from '@/utils/request'

// 登录接口
export const login = (data) => {
    return request({
        url: '/login',
        method: 'post',
        data
    })
}

// 注册接口
export const register = (data) => {
    return request({
        url: '/register',
        method: 'post',
        data
    })
}

// 退出接口
export const logout = () => {
    return request({
        url: '/logout',
        method: 'post'
    })
}


// 刷新token
export const refreshToken = () => {
    return request({
        url: '/refresh-token',
        method: 'post',
        withCredentials: true
    })
}

// 忘记密码 - 发送重置邮件
export const forgotPassword = (data) => {
    return request({
        url: '/forgot-password',
        method: 'post',
        data
    })
}

// 重置密码
export const resetPassword = (data) => {
    return request({
        url: '/reset-password',
        method: 'post',
        data
    })
}

// 清除所有token
export const clearToken = () => {
    document.cookie = 'csrf_token=; max-age=0; path=/'
}
