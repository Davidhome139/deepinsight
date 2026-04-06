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
      console.log('[Auth] Login request:', data)
      const res = await authApi.login(data)
      console.log('[Auth] Login response:', res)
      console.log('[Auth] Access token:', res.access_token)
      this.user = res.user
      this.token = res.access_token
      localStorage.setItem('token', this.token)
      localStorage.setItem('user', JSON.stringify(this.user))
      console.log('[Auth] Token stored:', this.token)
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
