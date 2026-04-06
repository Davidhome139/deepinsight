import request from '../utils/request'
import type { Conversation, Message } from '../types'

export const chatApi = {
  getConversations: () => request.get<any, Conversation[]>('/chat/conversations'),
  createConversation: (data: { title: string; model: string }) => 
    request.post<any, Conversation>('/chat/conversations', data),
  getMessages: (convId: number) => 
    request.get<any, Message[]>(`/chat/conversations/${convId}/messages`),
  getBranchMessages: (branchId: string) =>
    request.get<any, Message[]>(`/chat/branches/${branchId}/messages`),
  generateSummary: (conversationId: string, model: string) => request.post<any, { summary: string }>(`/chat/conversations/${conversationId}/summary`, { model })
}
