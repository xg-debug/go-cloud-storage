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


// 存储token
export const storeToken = (token, isRemember) => {
    if (isRemember) {
        // 长期存储（localStorage + Cookie双写）
        localStorage.setItem('token', token)
        document.cookie = `token=${token}; max-age=${7*24*60*60}; path=/; Secure`
    } else {
        // 会话存储（sessionStorage）
        sessionStorage.setItem('token', token)
    }
}

// 刷新token
export const refreshToken = () => {
    return request({
        url: '/refresh-token',
        method: 'post',
        withCredentials: true
    })
}

// 清除所有token
export const clearToken = () => {
    localStorage.removeItem('token')
    sessionStorage.removeItem('token')
    // 清除 cookie 中的 token
    document.cookie = 'token=; max-age=0; path=/'
}