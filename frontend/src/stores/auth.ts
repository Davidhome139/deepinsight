import { defineStore } from 'pinia'
import { authApi } from '../api/auth'
import type { LoginRequest, User } from '../types'
import router from '../router'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: JSON.parse(localStorage.getItem('user') || 'null') as User | null,
    token: localStorage.getItem('token') || '',
  }),
  actions: {
    async login(data: LoginRequest) {
      const res = await authApi.login(data)
      this.user = res.user
      this.token = res.access_token
      localStorage.setItem('token', this.token)
      localStorage.setItem('user', JSON.stringify(this.user))
      router.push('/')
    },
    logout() {
      this.user = null
      this.token = ''
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      router.push('/login')
    }
  }
})
