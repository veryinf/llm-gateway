import axios from 'axios'
import toast from 'react-hot-toast'

const client = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

client.interceptors.response.use(
  (res) => {
    const { code, msg, data } = res.data
    if (code !== 0) {
      toast.error(msg || 'error')
      return Promise.reject(new Error(msg))
    }
    return data
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/admin/login'
    }
    toast.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

export default client
