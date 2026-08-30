import axios from 'axios'
import { ElMessage } from 'element-plus'

export const TOKEN_KEY = 'gitsune_token'
export const USER_KEY = 'gitsune_user'

const http = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' }
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body && body.success === false) {
      ElMessage.error(body.error?.message || 'Request failed')
      return Promise.reject(new Error(body.error?.message || 'Request failed'))
    }
    return body?.data ?? null
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    } else {
      const body = error.response?.data
      if (body && body.success === false) {
        ElMessage.error(body.error?.message || 'Request failed')
      } else {
        ElMessage.error(error.message || 'Network error')
      }
    }
    return Promise.reject(error)
  }
)

// Auth
export const login = (username, password) => http.post('/auth/login', { username, password })
export const logout = () => http.post('/auth/logout')
export const getMe = () => http.post('/me')

// User management
export const createUser = (username, password, role) => http.post('/user/create', { username, password, role })
export const listUsers = (params) => http.post('/user/list', params)
export const updateUser = (id, role) => http.post('/user/update', { id, role })
export const deleteUser = (id) => http.post('/user/delete', { id })
export const resetPassword = (id, password) => http.post('/user/reset-password', { id, password })
export const changePassword = (old_password, new_password) => http.post('/user/change-password', { old_password, new_password })

// Repos
export const collectRepo = (url) => http.post('/repo/collect', { url })
export const listRepos = (params) => http.post('/repo/list', params)
export const getRepo = (id) => http.post('/repo/get', { id })
export const refreshRepo = (id) => http.post('/repo/refresh', { id })
export const deleteRepo = (id) => http.post('/repo/delete', { id })

// Tasks
export const startTask = (type) => http.post('/task/start', { type })
export const listTaskLogs = (params) => http.post('/task-log/list', params)

// Settings
export const getSetting = () => http.post('/setting/get')
export const updateSetting = (payload) => http.post('/setting/update', payload)

export default http
