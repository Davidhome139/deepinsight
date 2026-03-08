import { defineStore } from 'pinia'
import { chatApi } from '../api/chat'
import type { Conversation, Message } from '../types'

export const useChatStore = defineStore('chat', {
  state: () => ({
    conversations: [] as Conversation[],
    currentConversation: null as Conversation | null,
    messages: [] as Message[],
  }),
  actions: {
    async fetchConversations() {
      this.conversations = await chatApi.getConversations()
    },
    async selectConversation(conv: Conversation) {
      this.currentConversation = conv
      this.messages = await chatApi.getMessages(conv.id)
    },
    async selectBranch(branchId: string) {
      // Fetch messages for a specific branch
      this.messages = await chatApi.getBranchMessages(branchId)
    },
    async createNewConversation(title: string, model: string) {
      const conv = await chatApi.createConversation({ title, model })
      this.conversations.unshift(conv)
      this.currentConversation = conv
      this.messages = []
      return conv
    },
    async generateConversationSummary(conversationId: string, model: string) {
      return await chatApi.generateSummary(conversationId, model)
    }
  }
})
