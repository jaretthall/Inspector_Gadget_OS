import axios from 'axios'
import useAuthStore from '@store/auth'

const api = axios.create({})

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (resp) => resp,
  async (error) => {
    const original = error.config
    if (error.response?.status === 401 && !original._retried) {
      original._retried = true
      try {
        const res = await api.post('/api/auth/refresh')
        const { token } = res.data
        useAuthStore.getState().setToken(token)
        original.headers = original.headers || {}
        original.headers.Authorization = `Bearer ${token}`
        return api(original)
      } catch (_) {
        useAuthStore.getState().logout()
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export default api


