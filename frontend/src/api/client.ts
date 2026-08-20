import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')

  if (config.data instanceof FormData) {
    config.headers.delete('Content-Type')
  }

  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('access_token')
      localStorage.removeItem('user')

      const isSelfCarePath =
        window.location.pathname.startsWith('/selfcare')

      const loginPath = isSelfCarePath
        ? '/selfcare/login'
        : '/login'

      if (window.location.pathname !== loginPath) {
        window.location.replace(loginPath)
      }
    }

    return Promise.reject(error)
  },
)

export default apiClient
