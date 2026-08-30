const trimTrailingSlash = value => value.replace(/\/+$/, '')

export const API_BASE_URL = trimTrailingSlash(
  process.env.VUE_APP_API_BASE_URL || '/api'
)

export function buildApiUrl(path) {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return new URL(`${API_BASE_URL}${normalizedPath}`, window.location.origin).toString()
}
