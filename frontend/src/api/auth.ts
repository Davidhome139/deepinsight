import request from '../utils/request'
import type { LoginRequest, RegisterRequest, AuthResponse } from '../types'

export const authApi = {
  login: (data: LoginRequest) => request.post<any, AuthResponse>('/auth/login', data),
  register: (data: RegisterRequest) => request.post('/auth/register', data),
}
