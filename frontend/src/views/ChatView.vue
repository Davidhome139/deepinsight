<template>
  <div class="chat-layout">
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header" v-show="!sidebarCollapsed">
        <el-tooltip content="Collapse Sidebar" placement="right">
          <el-button @click="toggleSidebar" size="small" style="width: 100%; margin-bottom: 10px; background: transparent; border-color: transparent;" :icon="Fold">
          </el-button>
        </el-tooltip>
        <el-select v-model="selectedModel" placeholder="Select Model" style="margin-bottom: 10px; width: 100%;">
          <el-option 
            v-for="model in availableModels" 
            :key="model.id" 
            :label="model.name" 
            :value="model.id" 
          />
        </el-select>
        <el-button type="primary" @click="createNewChat" block>+ New Chat</el-button>
      </div>
      <div class="conversation-list" v-show="!sidebarCollapsed">
        <div 
          v-for="conv in conversations" 
          :key="conv.id"
          class="conversation-item"
          :class="{ active: currentConversation?.id === conv.id }"
          @click="selectConversation(conv)"
        >
          <div class="conv-title">{{ conv.title || 'Untitled Chat' }}</div>
          <div class="conv-meta">{{ conv.model_type }}</div>
        </div>
      </div>
      <div class="sidebar-footer" v-show="!sidebarCollapsed">
        <div class="feature-buttons">
          <el-button @click="$router.push('/programming')">AI Programming</el-button>
          <el-button @click="$router.push('/video')">Video Generation</el-button>
          <el-button @click="$router.push('/ai-chat')">AI-AI Chat</el-button>
        </div>
        <div class="footer-actions">
          <ThemeToggle />
          <el-dropdown @command="handleUserCommand">
            <span class="user-info">
              {{ user?.username }}
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="settings">Settings</el-dropdown-item>
                <el-dropdown-item command="logout">Logout</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <div class="collapsed-expand-btn" v-show="sidebarCollapsed" @click="toggleSidebar">
        <el-icon><Fold /></el-icon>
      </div>
    </aside>

    <main class="chat-main">
      <div class="messages" ref="messageBox">
        <div v-if="!currentConversation" class="empty-state">
          <h1>How can I help you today?</h1>
        </div>
        <template v-else>
          <div 
            v-for="msg in messages" 
            :key="msg.id" 
            class="message-wrapper"
            :class="msg.role"
          >
            <div class="message-container">
              <div v-if="msg.search_results" class="search-results-box">
                <div class="search-results-header">
                  <el-icon><Search /></el-icon>
                  <span>Web Search Results</span>
                </div>
                <div class="search-results-content">
                  <div v-for="(result, idx) in msg.search_results" :key="idx" class="search-result-item">
                    <a :href="result.url" target="_blank" class="result-title">{{ result.title }}</a>
                    <p class="result-snippet">{{ result.snippet }}</p>
                  </div>
                </div>
              </div>
              <div class="message-content" v-html="renderMarkdown(msg.content)"></div>
              <div v-if="msg.role === 'assistant'" class="message-actions">
                <el-tooltip content="Read aloud" placement="top">
                  <el-button 
                    size="small" 
                    circle
                    :loading="speakingMessageId === msg.id"
                    @click="speakMessage(msg)"
                  >
                    <el-icon><Microphone /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
              <!-- Message Editor for editing and regeneration -->
              <MessageEditor
                :message="msg"
                :conversation-id="currentConversation?.id || 0"
                :model="selectedModel"
                :is-streaming="isStreaming"
                @message-updated="handleMessageUpdate"
                @branch-created="handleBranchCreated"
                @refresh-messages="refreshMessages"
                @regenerate-stream="handleRegenerateStream"
              />
            </div>
          </div>
          <div v-if="currentSearchResults && currentSearchResults.length > 0" class="message-wrapper assistant">
            <div class="message-container">
              <div class="search-results-box">
                <div class="search-results-header">
                  <el-icon><Search /></el-icon>
                  <span>Web Search Results</span>
                </div>
                <div class="search-results-content">
                  <div v-for="(result, idx) in currentSearchResults" :key="idx" class="search-result-item">
                    <a :href="result.url" target="_blank" class="result-title">{{ result.title }}</a>
                    <p class="result-snippet">{{ result.snippet }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div v-if="streamingContent" class="message-wrapper assistant slide-up-enter-active">
            <div class="message-content" v-html="renderMarkdown(streamingContent)"></div>
          </div>
          <div v-else-if="isWaitingForResponse" class="message-wrapper assistant">
            <div class="message-content loading-container">
              <div class="typing-indicator">
                <span class="typing-dot"></span>
                <span class="typing-dot"></span>
                <span class="typing-dot"></span>
              </div>
              <span class="thinking-text">Thinking...</span>
            </div>
          </div>
        </template>
      </div>

      <div class="input-area">
        <!-- System Prompt Indicator -->
        <div v-if="systemPrompt" class="system-prompt-indicator">
          <el-tag type="info" size="small" closable @close="systemPrompt = ''">
            <el-icon><Setting /></el-icon>
            System Prompt Active
          </el-tag>
        </div>
        <div class="input-actions">
          <PromptTemplateSelector @insert="handleTemplateInsert" :compact="false" />
          <el-select v-model="selectedSearchProvider" placeholder="Web Search" size="small" class="toolbar-select">
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
            <el-option label="Search Disabled" value="" />
            <el-option 
              v-for="provider in availableSearchProviders" 
              :key="provider.id" 
              :label="provider.name" 
              :value="provider.id" 
            />
          </el-select>
          <!-- RAG Knowledge Base Toggle -->
          <el-popover placement="top" :width="280" trigger="click">
            <template #reference>
              <el-button 
                size="small" 
                class="toolbar-btn"
                :class="{ 'toolbar-btn-active': ragEnabled }"
              >
                <el-icon><Document /></el-icon>
                <span>{{ ragEnabled ? 'RAG On' : 'Knowledge' }}</span>
              </el-button>
            </template>
            <div class="rag-popover">
              <div class="rag-toggle">
                <span>Enable Knowledge Base</span>
                <el-switch v-model="ragEnabled" />
              </div>
              <div v-if="ragEnabled" class="rag-docs">
                <p class="rag-docs-label">Select Documents:</p>
                <el-checkbox-group v-model="selectedRagDocIds" v-if="ragDocuments.length > 0">
                  <el-checkbox 
                    v-for="doc in ragDocuments" 
                    :key="doc.id" 
                    :label="doc.id"
                    :disabled="doc.status !== 'ready'"
                  >
                    {{ doc.title }}
                    <el-tag v-if="doc.status !== 'ready'" size="small" type="warning">{{ doc.status }}</el-tag>
                  </el-checkbox>
                </el-checkbox-group>
                <el-empty v-else description="No documents uploaded" :image-size="60" />
                <el-button size="small" type="primary" text @click="$router.push('/rag')">
                  Manage Documents
                </el-button>
              </div>
            </div>
          </el-popover>
          <el-select v-model="selectedMCPTool" placeholder="MCP Tool" size="small" class="toolbar-select">
            <template #prefix>
              <el-icon><SetUp /></el-icon>
            </template>
            <el-option label="No MCP Tool" value="" />
            <el-option 
              v-for="tool in availableMCPTools" 
              :key="tool.id" 
              :label="tool.name" 
              :value="tool.id" 
            />
          </el-select>
          <ParallelExplorer
            v-if="currentConversation"
            :conversation-id="currentConversation.id"
            :last-user-message-id="lastUserMessageId"
            @branch-selected="handleBranchSwitch"
            @refresh-messages="refreshMessages"
          />
          <el-button 
            size="small" 
            class="toolbar-btn"
            :disabled="!currentConversation || messages.length === 0 || isStreaming"
            @click="generateSummary"
          >
            <el-icon><List /></el-icon>
            <span>Summary</span>
          </el-button>
        </div>
        <div class="input-row">
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="3"
            placeholder="Type a message... (Use /clear to clear context, /compact to summarize)"
            @keydown.enter.prevent="handleSend"
          />
          <el-tooltip :content="isListening ? 'Click to stop listening' : (speechSupported ? 'Voice input' : 'Voice input not supported in this browser')"
            placement="top">
            <el-button
              :type="isListening ? 'danger' : 'default'"
              circle
              :class="{ 'mic-btn-listening': isListening }"
              @click="startVoiceInput"
            >
              <el-icon><Microphone /></el-icon>
            </el-button>
          </el-tooltip>
          <el-button 
            type="primary" 
            :disabled="!inputText.trim() || isStreaming"
            @click="handleSend"
          >
            Send
          </el-button>
        </div>
      </div>
    </main>

    <aside class="context-sidebar" :class="{ collapsed: contextSidebarCollapsed }">
      <div class="context-sidebar-header" v-show="!contextSidebarCollapsed">
        <h3>Context & Branches</h3>
        <el-tooltip content="Collapse Context">
          <el-button @click="toggleContextSidebar" size="small" style="background: transparent; border-color: transparent;" :icon="Fold">
          </el-button>
        </el-tooltip>
      </div>
      <div class="context-content" v-show="!contextSidebarCollapsed">
        <!-- Branch Panel -->
        <BranchPanel 
          v-if="currentConversation"
          ref="branchPanelRef"
          :conversation-id="currentConversation.id"
          :messages="messages"
          @branch-switched="handleBranchSwitch"
          @refresh-messages="refreshMessages"
        />
        
        <el-divider v-if="currentConversation" />
        
        <div v-if="lastQuestion && lastAnswer" class="context-pair">
          <div class="context-question">
            <strong>Your Question:</strong>
            <div class="context-text" v-html="renderMarkdown(lastQuestion)"></div>
          </div>
          <div class="context-answer">
            <strong>AI Answer:</strong>
            <div class="context-text" v-html="renderMarkdown(lastAnswer)"></div>
          </div>
        </div>
        <div v-else class="empty-context">
          <p>No recent context available</p>
        </div>
      </div>
      <div class="collapsed-expand-btn" v-show="contextSidebarCollapsed" @click="toggleContextSidebar">
        <el-icon><Fold /></el-icon>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, Search, DArrowLeft, DArrowRight, Fold, Microphone, Document, List, SetUp, Setting } from '@element-plus/icons-vue'
import PromptTemplateSelector from '../components/PromptTemplateSelector.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import BranchPanel from '../components/chat/BranchPanel.vue'
import MessageEditor from '../components/chat/MessageEditor.vue'
import ParallelExplorer from '../components/chat/ParallelExplorer.vue'

console.log('[ChatView] Script loaded - test log')

const authStore = useAuthStore()
const chatStore = useChatStore()
const router = useRouter()

const user = computed(() => authStore.user)
const conversations = computed(() => chatStore.conversations)
const currentConversation = computed(() => chatStore.currentConversation)
const messages = computed(() => chatStore.messages)

const selectedModel = ref('')
const selectedSearchProvider = ref('')
const selectedMCPTool = ref('')
const ragEnabled = ref(false)
const ragDocuments = ref<Array<{id: string, title: string, status: string}>>([])
const selectedRagDocIds = ref<string[]>([])
const webSearchEnabled = computed(() => selectedSearchProvider.value !== '')
const availableModels = ref<Array<{id: string, name: string, provider: string}>>([])
const availableSearchProviders = ref<Array<{id: string, name: string}>>([])
const availableMCPTools = ref<Array<{id: string, name: string, server: string}>>([])
const inputText = ref('')
const messageBox = ref<HTMLElement | null>(null)
const isStreaming = ref(false)
const isWaitingForResponse = ref(false)
const streamingContent = ref('')
const currentSearchResults = ref<Array<{title: string, snippet: string, url: string}>>([])
const sidebarCollapsed = ref(false)
const contextSidebarCollapsed = ref(false)
const branchPanelRef = ref<InstanceType<typeof BranchPanel> | null>(null)
const activeBranchId = ref<string | null>(null)
const systemPrompt = ref('')

// Handle prompt template insertion
const handleTemplateInsert = (prompt: string, target: 'user' | 'system' = 'user') => {
  if (target === 'system') {
    systemPrompt.value = prompt
  } else {
    inputText.value = prompt
  }
}

// Handle branch switching
const handleBranchSwitch = async (branchId: string | { compare?: boolean, branchId?: string }) => {
  if (typeof branchId === 'object') {
    // Handle compare mode
    if (branchId.compare) {
      ElMessage.info('Compare mode coming soon')
    }
    return
  }
  activeBranchId.value = branchId
  // Fetch messages for the selected branch
  await chatStore.selectBranch(branchId)
  scrollToBottom()
}

// Refresh messages (re-fetch from server)
const refreshMessages = async () => {
  if (currentConversation.value) {
    // If a branch is active, fetch branch-specific messages
    if (activeBranchId.value) {
      await chatStore.selectBranch(activeBranchId.value)
    } else {
      await chatStore.selectConversation(currentConversation.value)
    }
    scrollToBottom()
  }
}

// Handle streaming regenerate from MessageEditor
const handleRegenerateStream = async (data: { messageId: number, model: string }) => {
  isStreaming.value = true
  isWaitingForResponse.value = true
  streamingContent.value = ''
  
  try {
    const response = await fetch(`/api/v1/chat/messages/${data.messageId}/regenerate/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ model: data.model })
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(errorData.error || `Server error: ${response.status}`)
    }

    const reader = response.body?.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader!.read()
      if (done) break

      const chunk = decoder.decode(value, { stream: true })
      buffer += chunk
      
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      
      for (const line of lines) {
        const trimmedLine = line.trim()
        if (!trimmedLine) continue
        
        const dataPrefix = 'data: '
        if (!trimmedLine.startsWith(dataPrefix)) continue
        
        const jsonStr = trimmedLine.slice(dataPrefix.length).trim()
        if (!jsonStr) continue
        
        try {
          const chunkData = JSON.parse(jsonStr)
          if (chunkData.done) break
          if (chunkData.content) {
            isWaitingForResponse.value = false
            streamingContent.value += chunkData.content
            scrollToBottom()
          }
        } catch (e) {
          console.error('Failed to parse SSE data:', trimmedLine, e)
        }
      }
    }
    
    // Refresh messages to show the new assistant message
    streamingContent.value = ''
    await refreshMessages()
    ElMessage.success('Response regenerated')
    
  } catch (error: any) {
    console.error('Streaming regenerate error:', error)
    ElMessage.error(error.message || 'Failed to regenerate response')
  } finally {
    isStreaming.value = false
    isWaitingForResponse.value = false
  }
}

// Handle message update from editor
const handleMessageUpdate = async () => {
  await refreshMessages()
}

// Handle branch creation from message editor
const handleBranchCreated = async (data: any) => {
  if (data.fork_point_message_id) {
    // Open branch creation dialog
    ElMessage.info('Creating branch from message...')
  }
  if (branchPanelRef.value) {
    branchPanelRef.value.fetchBranches()
  }
  await refreshMessages()
}

// Get last user message ID for parallel explorer
const lastUserMessageId = computed(() => {
  const userMessages = messages.value.filter(m => m.role === 'user')
  return userMessages.length > 0 ? userMessages[userMessages.length - 1].id : undefined
})

// Voice input (STT)
const isListening = ref(false)
const speechSupported = ref(
  typeof window !== 'undefined' &&
  ('SpeechRecognition' in window || 'webkitSpeechRecognition' in window)
)
let recognizer: any = null

// TTS (Text-to-Speech)
const speakingMessageId = ref<number | null>(null)
let currentAudio: HTMLAudioElement | null = null

const speakMessage = async (msg: any) => {
  // If currently speaking this message, stop it
  if (speakingMessageId.value === msg.id) {
    currentAudio?.pause()
    currentAudio = null
    speakingMessageId.value = null
    return
  }
  
  // Stop any current audio
  currentAudio?.pause()
  
  speakingMessageId.value = msg.id
  try {
    const response = await fetch('/api/v1/tts/speak', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        text: msg.content,
        voice: 'longxiaochun',
        model: 'cosyvoice-v1'
      })
    })
    
    if (!response.ok) throw new Error('TTS request failed')
    
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    currentAudio = new Audio(url)
    
    currentAudio.onended = () => {
      speakingMessageId.value = null
      URL.revokeObjectURL(url)
    }
    
    currentAudio.onerror = () => {
      speakingMessageId.value = null
      ElMessage.error('Failed to play audio')
    }
    
    await currentAudio.play()
  } catch (error) {
    console.error('TTS error:', error)
    speakingMessageId.value = null
    ElMessage.error('Failed to generate speech')
  }
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const toggleContextSidebar = () => {
  contextSidebarCollapsed.value = !contextSidebarCollapsed.value
}

const startVoiceInput = () => {
  if (!speechSupported.value) {
    ElMessage.warning('Voice input is not supported in this browser. Please use Chrome or Edge.')
    return
  }
  if (isListening.value) {
    recognizer?.stop()
    isListening.value = false
    return
  }
  const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
  recognizer = new SR()
  recognizer.lang = 'zh-CN'
  recognizer.interimResults = false
  recognizer.maxAlternatives = 1
  recognizer.onresult = (e: any) => {
    const transcript = e.results[0][0].transcript
    inputText.value += (inputText.value ? ' ' : '') + transcript
  }
  recognizer.onend = () => { isListening.value = false }
  recognizer.onerror = () => { isListening.value = false }
  recognizer.start()
  isListening.value = true
}

// Get the last question and answer from messages
const lastQuestion = computed(() => {
  const userMessages = messages.value.filter(msg => msg.role === 'user')
  return userMessages.length > 0 ? userMessages[userMessages.length - 1].content : ''
})

const lastAnswer = computed(() => {
  const assistantMessages = messages.value.filter(msg => msg.role === 'assistant')
  return assistantMessages.length > 0 ? assistantMessages[assistantMessages.length - 1].content : ''
})

onMounted(async () => {
  await chatStore.fetchConversations()
  await fetchAvailableModels()
  await fetchAvailableSearchProviders()
  await fetchAvailableMCPTools()
  await fetchRagDocuments()
})

const fetchRagDocuments = async () => {
  try {
    const response = await fetch('/api/v1/rag/documents', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      ragDocuments.value = data.documents || []
    }
  } catch (error) {
    console.error('Failed to fetch RAG documents:', error)
  }
}

const fetchAvailableModels = async () => {
  try {
    const response = await fetch('/api/v1/chat/models', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const models = await response.json()
      availableModels.value = models
      // Set default model if available
      if (models.length > 0 && !selectedModel.value) {
        selectedModel.value = models[0].id
      }
    }
  } catch (error) {
    console.error('Failed to fetch models:', error)
  }
}

const fetchAvailableSearchProviders = async () => {
  try {
    const response = await fetch('/api/v1/chat/search-providers', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const providers = await response.json()
      availableSearchProviders.value = providers
    }
  } catch (error) {
    console.error('Failed to fetch search providers:', error)
  }
}

const fetchAvailableMCPTools = async () => {
  try {
    const response = await fetch('/api/v1/chat/mcp-tools', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const tools = await response.json()
      availableMCPTools.value = tools
    }
  } catch (error) {
    console.error('Failed to fetch MCP tools:', error)
  }
}

const renderMarkdown = (content: string) => {
  console.log('[ChatView] renderMarkdown called, content length:', content?.length || 0, 'contains 延续探讨:', content?.includes('延续探讨'))
  
  const renderer = new marked.Renderer()
  renderer.code = (code: string, lang: string | undefined) => {
    const language = lang && hljs.getLanguage(lang) ? lang : 'plaintext'
    const highlighted = hljs.highlight(code, { language }).value
    return `<pre><code class="hljs ${language}">${highlighted}</code></pre>\n`
  }
  
  // Parse the markdown first
  let html = marked.parse(content, { renderer, async: false }) as string
  
  // Check if content contains follow-up markers
  const followUpKeywords = ['Related Questions', 'Follow-up Questions', '相关问题', '建议问题', 'Suggested Questions', '延续探讨']
  let hasFollowUpMarker = false
  
  for (const keyword of followUpKeywords) {
    if (content.includes(keyword)) {
      hasFollowUpMarker = true
      console.log('[ChatView] Detected follow-up keyword:', keyword)
      break
    }
  }
  
  // If we detect follow-up marker, wrap everything after it
  if (hasFollowUpMarker) {
    // Match <p> tag that directly contains the keyword (not across multiple paragraphs)
    // Pattern: <p...>...<strong>...keyword...</strong>...</p> or <p...>...keyword...</p>
    // Use [^<]* to ensure we don't cross into other tags
    const patterns = followUpKeywords.map(keyword => 
      `<p[^>]*>[^<]*(?:<strong>[^<]*)?${keyword}(?:[^<]*</strong>)?[^<]*</p>`
    )
    const combinedPattern = new RegExp(`(${patterns.join('|')})`, 'i')
    
    console.log('[ChatView] Searching for pattern in HTML length:', html.length)
    const match = html.match(combinedPattern)
    
    if (match && match.index !== undefined) {
      console.log('[ChatView] Match found at index:', match.index, 'matched text:', match[0].substring(0, 100))
      const beforeContent = html.substring(0, match.index)
      const afterContent = html.substring(match.index)
      html = beforeContent + '<div class="follow-up-questions">' + afterContent + '</div>'
      console.log('[ChatView] Wrapped follow-up section')
    } else {
      console.log('[ChatView] No match found in HTML')
    }
  }
  
  return html
}

const createNewChat = async () => {
  await chatStore.createNewConversation('New Chat', selectedModel.value)
}

const selectConversation = async (conv: any) => {
  await chatStore.selectConversation(conv)
  scrollToBottom()
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messageBox.value) {
      messageBox.value.scrollTop = messageBox.value.scrollHeight
    }
  })
}

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    authStore.logout()
  } else if (command === 'settings') {
    router.push('/settings')
  }
}

const handleSend = async () => {
  if (!inputText.value.trim() || isStreaming.value) return
  
  const content = inputText.value.trim()
    
  // Check for slash commands
  if (content === '/clear') {
    inputText.value = ''
    clearContext()
    return
  }
  
  if (content === '/compact') {
    inputText.value = ''
    compactContext()
    return
  }
  
  if (!currentConversation.value) {
    await createNewChat()
  }

  inputText.value = ''
  
  // Push user message locally for immediate feedback
  chatStore.messages.push({
    id: Date.now(),
    conversation_id: currentConversation.value!.id,
    role: 'user',
    content: content,
    created_at: new Date().toISOString()
  })

  scrollToBottom()
  await startStreaming(content)
}

const startStreaming = async (content: string) => {
  isStreaming.value = true
  isWaitingForResponse.value = true
  streamingContent.value = ''
  
  try {
    // Use RAG endpoint if RAG is enabled
    const endpoint = ragEnabled.value ? '/api/v1/chat/stream/rag' : '/api/v1/chat/stream'
    const requestBody: any = {
      conversation_id: currentConversation.value!.id,
      content: content,
      model: selectedModel.value,
      web_search: webSearchEnabled.value,
      search_provider: selectedSearchProvider.value,
      mcp_tool: selectedMCPTool.value,
      system_prompt: systemPrompt.value || undefined
    }
    
    // Add RAG parameters if enabled
    if (ragEnabled.value) {
      requestBody.rag_enabled = true
      requestBody.rag_document_ids = selectedRagDocIds.value
    }
    
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify(requestBody)
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(errorData.error || `Server error: ${response.status}`)
    }

    const reader = response.body?.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    let hasContent = false

    while (true) {
      const { done, value } = await reader!.read()
      if (done) break

      const chunk = decoder.decode(value, { stream: true })
      buffer += chunk
      
      const lines = buffer.split('\n')
      buffer = lines.pop() || '' 
      
      for (const line of lines) {
        const trimmedLine = line.trim()
        if (!trimmedLine) continue
        
        // Handle lines that might not start exactly with "data: " (resilience)
        const dataPrefix = 'data: '
        if (!trimmedLine.startsWith(dataPrefix)) continue
        
        const jsonStr = trimmedLine.slice(dataPrefix.length).trim()
        if (!jsonStr) continue
        
        try {
          const data = JSON.parse(jsonStr)
          
          // Check if this chunk contains search results
          if (data.search_results && data.search_results.length > 0) {
            currentSearchResults.value = data.search_results
            scrollToBottom()
            continue
          }
          
          if (data.done) {
            hasContent = true
            break
          }
          if (data.content) {
            isWaitingForResponse.value = false
            streamingContent.value += data.content
            hasContent = true
            scrollToBottom()
          }
        } catch (e) {
          console.error('Failed to parse SSE data:', trimmedLine, e)
        }
      }
    }
    
    // Always save the assistant message if there was any content
    if (hasContent && streamingContent.value) {
      chatStore.messages.push({
        id: Date.now(),
        conversation_id: currentConversation.value!.id,
        role: 'assistant',
        content: streamingContent.value,
        search_results: currentSearchResults.value.length > 0 ? currentSearchResults.value : undefined,
        created_at: new Date().toISOString()
      } as any)
    }
    streamingContent.value = ''
    currentSearchResults.value = []
    
  } catch (error: any) {
    console.error('Streaming error:', error)
    ElMessage.error(error.message || 'Failed to get response from AI')
  } finally {
    isStreaming.value = false
    isWaitingForResponse.value = false
    // Refresh messages to get actual database IDs for regenerate feature
    await refreshMessages()
    // Refresh conversation list to get potential auto-generated title
    if (messages.value.length <= 2) {
      setTimeout(() => {
        chatStore.fetchConversations()
      }, 2000)
    }
  }
}

const clearContext = () => {
  if (!currentConversation.value) return
  
  ElMessageBox.confirm(
    'This will clear all messages in this conversation. Continue?',
    'Clear Context',
    {
      confirmButtonText: 'Clear',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(() => {
    chatStore.messages = []
    ElMessage.success('Context cleared')
  }).catch(() => {
    // User cancelled
  })
}

const compactContext = async () => {
  if (!currentConversation.value || messages.value.length === 0) return
  
  try {
    isStreaming.value = true
    isWaitingForResponse.value = true
    
    // Build a prompt to summarize the conversation
    const conversationText = messages.value
      .map(m => `${m.role}: ${m.content}`)
      .join('\n\n')
    
    const summaryPrompt = `Please provide a concise summary of the following conversation, capturing the key points and context:\n\n${conversationText}`
    
    const response = await fetch('/api/v1/chat/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        conversation_id: currentConversation.value.id,
        content: summaryPrompt,
        model: selectedModel.value,
        stream: true
      })
    })
    
    if (!response.ok) {
      throw new Error('Failed to generate summary')
    }
    
    const reader = response.body?.getReader()
    const decoder = new TextDecoder()
    let summary = ''
    
    while (true) {
      const { done, value } = await reader!.read()
      if (done) break
      
      const chunk = decoder.decode(value, { stream: true })
      const lines = chunk.split('\n')
      
      for (const line of lines) {
        const trimmedLine = line.trim()
        if (!trimmedLine || !trimmedLine.startsWith('data: ')) continue
        
        const jsonStr = trimmedLine.slice(6).trim()
        if (!jsonStr) continue
        
        try {
          const data = JSON.parse(jsonStr)
          if (data.content) {
            summary += data.content
          }
        } catch (e) {
          // Ignore parse errors
        }
      }
    }
    
    // Replace messages with a single summary message
    chatStore.messages = [{
      id: Date.now(),
      conversation_id: currentConversation.value.id,
      role: 'system',
      content: `**Conversation Summary:**\n\n${summary}`,
      created_at: new Date().toISOString()
    }]
    
    ElMessage.success('Context compacted successfully')
  } catch (error: any) {
    console.error('Compact error:', error)
    ElMessage.error(error.message || 'Failed to compact context')
  } finally {
    isStreaming.value = false
    isWaitingForResponse.value = false
  }
}

const generateSummary = async () => {
  if (!currentConversation.value || messages.value.length === 0) return
  
  try {
    isStreaming.value = true
    isWaitingForResponse.value = true
    
    const response = await chatStore.generateConversationSummary(currentConversation.value.id.toString(), selectedModel.value)
    
    // Add summary as a new system message
    chatStore.messages.push({
      id: Date.now(),
      conversation_id: currentConversation.value.id,
      role: 'system',
      content: `**Conversation Summary:**\n\n${response.summary}`,
      created_at: new Date().toISOString()
    } as any)
    
    ElMessage.success('Conversation summary generated successfully')
  } catch (error: any) {
    console.error('Generate summary error:', error)
    ElMessage.error(error.message || 'Failed to generate conversation summary')
  } finally {
    isStreaming.value = false
    isWaitingForResponse.value = false
  }
}
</script>

<style scoped>
.chat-layout {
  display: flex;
  height: 100vh;
}
.sidebar {
  width: 260px;
  background-color: var(--sidebar-bg);
  color: var(--text-primary);
  display: flex;
  flex-direction: column;
  position: relative;
  transition: width 0.3s ease;
  border-right: 1px solid var(--border-primary);
}
.sidebar.collapsed {
  width: 50px;
}
.context-sidebar {
  width: 300px;
  background-color: var(--bg-secondary);
  border-left: 1px solid var(--border-primary);
  display: flex;
  flex-direction: column;
  position: relative;
  transition: width 0.3s ease;
}
.context-sidebar.collapsed {
  width: 50px;
}
.collapsed-expand-btn {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 36px;
  height: 36px;
  background-color: transparent;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.2s;
}
.sidebar .collapsed-expand-btn:hover {
  background-color: var(--bg-hover);
}
.context-sidebar .collapsed-expand-btn:hover {
  background-color: var(--bg-hover);
}
.collapsed-expand-btn .el-icon {
  color: var(--text-primary);
  font-size: 18px;
}
.context-sidebar .collapsed-expand-btn .el-icon {
  color: var(--text-secondary);
}
.context-sidebar-header {
  padding: 15px;
  border-bottom: 1px solid var(--border-primary);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.context-sidebar-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-primary);
}
.context-content {
  flex: 1;
  overflow-y: auto;
  padding: 15px;
}
.context-pair {
  display: flex;
  flex-direction: column;
  gap: 15px;
}
.context-question, .context-answer {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.context-text {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
  background-color: var(--card-bg);
  padding: 10px;
  border-radius: 4px;
  border: 1px solid var(--border-primary);
  max-height: 300px;
  overflow-y: auto;
}
.empty-context {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: var(--text-tertiary);
  font-style: italic;
}
.sidebar-header {
  padding: 10px;
}
.conversation-list {
  flex: 1;
  overflow-y: auto;
}
.conversation-item {
  padding: 10px 15px;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.conversation-item:hover, .conversation-item.active {
  background-color: var(--bg-hover);
}
.conv-meta {
  font-size: 12px;
  color: var(--text-tertiary);
}
.sidebar-footer {
  padding: 15px;
  border-top: 1px solid var(--border-primary);
}
.feature-buttons {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 15px;
}
.feature-buttons .el-button {
  width: 100%;
  margin: 0;
}
.user-info {
  color: var(--text-primary);
  cursor: pointer;
}
.footer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: var(--bg-primary);
}
.messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}
.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: var(--text-tertiary);
}
.message-wrapper {
  margin-bottom: 20px;
  display: flex;
}
.message-wrapper.user {
  justify-content: flex-end;
}
.message-container {
  display: flex;
  flex-direction: column;
  max-width: 80%;
  gap: 10px;
}
.message-content {
  padding: 10px 15px;
  border-radius: 8px;
}
.user .message-content {
  background-color: var(--accent-primary);
  color: white;
}
.assistant .message-content {
  background-color: var(--bg-secondary);
  color: var(--text-primary);
}
.message-actions {
  display: flex;
  gap: 4px;
  margin-top: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}
.message-container:hover .message-actions {
  opacity: 1;
}
.loading-container {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-tertiary);
}
.loading-container .el-icon {
  font-size: 20px;
}
.search-results-box {
  border: 1px solid var(--border-primary);
  border-radius: 6px;
  background-color: var(--bg-tertiary);
  padding: 12px;
  width: 100%;
}
.search-results-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 10px;
}
.search-results-header .el-icon {
  font-size: 16px;
  color: var(--accent-primary);
}
.search-results-content {
  font-size: 12px;
  color: var(--text-secondary);
}
.search-result-item {
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-primary);
}
.search-result-item:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}
.result-title {
  color: var(--accent-primary);
  text-decoration: none;
  font-weight: 500;
  display: block;
  margin-bottom: 4px;
}
.result-title:hover {
  text-decoration: underline;
}
.result-snippet {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.4;
}
.input-area {
  padding: 20px;
  border-top: 1px solid var(--border-primary);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.input-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 5px;
  align-items: center;
  flex-wrap: wrap;
}

.system-prompt-indicator {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.system-prompt-indicator .el-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Unified Toolbar Styling */
.toolbar-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px !important;
  border: 1px solid var(--border-primary) !important;
  border-radius: 20px !important;
  background: var(--bg-primary) !important;
  color: var(--text-secondary) !important;
  font-size: 13px !important;
  font-weight: 400 !important;
  transition: all 0.2s ease;
  height: 32px !important;
}

.toolbar-btn:hover:not(:disabled) {
  border-color: var(--color-primary) !important;
  color: var(--color-primary) !important;
  background: var(--bg-secondary) !important;
}

.toolbar-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toolbar-btn-active {
  border-color: var(--color-success) !important;
  color: var(--color-success) !important;
  background: rgba(103, 194, 58, 0.1) !important;
}

.toolbar-btn .el-icon {
  font-size: 14px;
}

.toolbar-select {
  width: 150px;
}

.toolbar-select .el-input__wrapper {
  border-radius: 20px !important;
  padding: 0 12px !important;
  height: 32px !important;
  box-shadow: none !important;
  border: 1px solid var(--border-primary) !important;
  background: var(--bg-primary) !important;
}

.toolbar-select .el-input__wrapper:hover {
  border-color: var(--color-primary) !important;
}

.toolbar-select .el-input__inner {
  font-size: 13px !important;
  color: var(--text-secondary) !important;
}

.toolbar-select .el-input__prefix {
  color: var(--text-secondary);
}

:deep(.prompt-template-selector .template-trigger-btn) {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px !important;
  border: 1px solid var(--border-primary) !important;
  border-radius: 20px !important;
  background: var(--bg-primary) !important;
  color: var(--text-secondary) !important;
  font-size: 13px !important;
  font-weight: 400 !important;
  transition: all 0.2s ease;
  height: 32px !important;
}

:deep(.prompt-template-selector .template-trigger-btn:hover) {
  border-color: var(--color-primary) !important;
  color: var(--color-primary) !important;
  background: var(--bg-secondary) !important;
}

:deep(.parallel-explorer .el-button) {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px !important;
  border: 1px solid var(--border-primary) !important;
  border-radius: 20px !important;
  background: var(--bg-primary) !important;
  color: var(--text-secondary) !important;
  font-size: 13px !important;
  font-weight: 400 !important;
  transition: all 0.2s ease;
  height: 32px !important;
}

:deep(.parallel-explorer .el-button:hover) {
  border-color: var(--color-primary) !important;
  color: var(--color-primary) !important;
  background: var(--bg-secondary) !important;
}
.input-row {
  display: flex;
  gap: 10px;
}

/* Override markdown styling for context sidebar */
.context-text pre {
  background-color: var(--code-bg);
  border-radius: 3px;
  padding: 8px;
  overflow-x: auto;
  font-size: 12px;
  margin: 5px 0;
}

.context-text code {
  background-color: var(--code-bg);
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 12px;
}

.context-text p {
  margin: 5px 0;
}

.context-text h1, .context-text h2, .context-text h3 {
  font-size: 14px;
  margin: 8px 0;
}

/* Follow-up questions styling - smaller font, transparent background, FangSong font */
/* Use :deep() to target elements inside v-html content (scoped CSS) */
:deep(.follow-up-questions) {
  font-size: 12px;
  background-color: transparent;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--border-primary);
  font-family: 'FangSong', '仿宋', 'STFangSong', serif;
}

:deep(.follow-up-questions p),
:deep(.follow-up-questions li) {
  font-size: 12px !important;
  color: var(--text-tertiary) !important;
  font-family: 'FangSong', '仿宋', 'STFangSong', serif !important;
}

:deep(.follow-up-questions strong) {
  font-size: 12px !important;
  color: var(--text-tertiary) !important;
  font-family: 'FangSong', '仿宋', 'STFangSong', serif !important;
}

:deep(.follow-up-questions ol),
:deep(.follow-up-questions ul) {
  margin: 8px 0;
  padding-left: 20px;
  font-size: 12px;
}

:deep(.follow-up-questions h1),
:deep(.follow-up-questions h2),
:deep(.follow-up-questions h3),
:deep(.follow-up-questions h4),
:deep(.follow-up-questions h5),
:deep(.follow-up-questions h6) {
  font-size: 13px !important;
  color: var(--text-tertiary) !important;
  margin: 8px 0;
  font-family: 'FangSong', '仿宋', 'STFangSong', serif !important;
}

.mic-btn-listening {
  animation: mic-pulse 1s infinite;
}

@keyframes mic-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(245, 108, 108, 0.4); }
  50%       { box-shadow: 0 0 0 8px rgba(245, 108, 108, 0); }
}

/* RAG Popover Styles */
.rag-popover {
  padding: 8px;
}

.rag-toggle {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #ebeef5;
  margin-bottom: 12px;
}

.rag-docs {
  max-height: 200px;
  overflow-y: auto;
}

.rag-docs-label {
  margin: 0 0 8px 0;
  font-size: 12px;
  color: #909399;
}

.rag-docs :deep(.el-checkbox) {
  display: block;
  margin-bottom: 8px;
}

.rag-docs :deep(.el-checkbox__label) {
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}

/* Responsive styles */
@media (max-width: 991.98px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 100;
    transform: translateX(-100%);
  }
  
  .sidebar:not(.collapsed) {
    transform: translateX(0);
  }
  
  .context-sidebar {
    display: none;
  }
  
  .chat-layout {
    flex-direction: column;
  }
}

@media (max-width: 767.98px) {
  .input-actions {
    flex-wrap: wrap;
    gap: 8px;
  }
  
  .input-actions .el-select {
    width: 100% !important;
    margin-left: 0 !important;
  }
  
  .message-content {
    padding: 12px;
    font-size: 14px;
  }
  
  .feature-buttons .el-button {
    font-size: 12px;
    padding: 8px;
  }
}
</style>
