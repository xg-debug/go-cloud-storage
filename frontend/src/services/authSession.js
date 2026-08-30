import axios from 'axios'
import { API_BASE_URL } from '@/config/runtime'

const authClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
  withCredentials: true
})

let refreshPromise = null

function unwrap(response, fallbackMessage) {
  if (response.data?.code !== 200) {
    throw new Error(response.data?.message || fallbackMessage)
  }
  return response.data.data
}

export async function fetchProfile() {
  return unwrap(await authClient.get('/me'), '未登录')
}

// All callers share one refresh request, preventing a burst of parallel 401s
// from rotating the refresh token multiple times.
export function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = authClient.post('/refresh-token')
      .then(response => unwrap(response, '刷新登录状态失败'))
      .finally(() => { refreshPromise = null })
  }
  return refreshPromise
}
