import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

let responseInterceptorRegistered = false

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('anox_token') || '')

  const isAuthenticated = computed(() => !!token.value)

  async function login(password: string): Promise<boolean> {
    try {
      const response = await axios.post('/api/login', { password })
      token.value = response.data.token
      localStorage.setItem('anox_token', token.value)
      // Set default header for all requests
      axios.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
      return true
    } catch (error) {
      return false
    }
  }

  function logout() {
    token.value = ''
    localStorage.removeItem('anox_token')
    delete axios.defaults.headers.common['Authorization']
  }

  function init() {
    if (token.value) {
      axios.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
    }

    if (!responseInterceptorRegistered) {
      axios.interceptors.response.use(
        response => response,
        error => {
          if (error.response?.status === 401) {
            logout()
          }
          return Promise.reject(error)
        },
      )
      responseInterceptorRegistered = true
    }
  }

  return {
    token,
    isAuthenticated,
    login,
    logout,
    init,
  }
})
