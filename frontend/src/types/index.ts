export interface User {
  id: number
  username: string
  email: string
  avatar?: string
}

export interface AuthResponse {
  user: User
  access_token: string
  refresh_token: string
}

export interface Conversation {
  id: number
  title: string
  model_type: string
  last_message?: string
  updated_at: string
}

export interface Message {
  id: number
  conversation_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  search_results?: Array<{
    title: string
    snippet: string
    url: string
  }>
  created_at: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}
