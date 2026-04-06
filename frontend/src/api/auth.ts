import request from '../utils/request'
import type { LoginRequest, RegisterRequest, AuthResponse } from '../types'

export const authApi = {
  login: async (data: LoginRequest): Promise<AuthResponse> => {
    console.log('[Auth API] Login request:', data)
    const response = await request.post('/auth/login', data)
    console.log('[Auth API] Login response:', response)
    return response as unknown as AuthResponse
  },
  register: (data: RegisterRequest) => request.post('/auth/register', data),
}
