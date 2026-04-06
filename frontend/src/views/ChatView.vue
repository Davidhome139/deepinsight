<template>
  <div class="chat-layout">
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header" v-show="!sidebarCollapsed">
        <el-tooltip :content="t('common.close')" placement="right">
          <el-button @click="toggleSidebar" size="small" style="width: 100%; margin-bottom: 10px; background: transparent; border-color: transparent;" :icon="Fold">
          </el-button>
        </el-tooltip>
        <el-select v-model="selectedModel" :placeholder="t('common.pleaseSelect')" style="margin-bottom: 10px; width: 100%;">
          <el-option 
            v-for="model in availableModels" 
            :key="model.id" 
            :label="model.name" 
            :value="model.id" 
          />
        </el-select>
        <el-button type="primary" @click="createNewChat" block>{{ t('chat.newChat') }}</el-button>
      </div>
      <div class="conversation-list" v-show="!sidebarCollapsed">
        <div 
          v-for="conv in conversations" 
          :key="conv.id"
          class="conversation-item"
          :class="{ active: currentConversation?.id === conv.id }"
        >
          <div class="conv-content" @click="selectConversation(conv)">
            <div class="conv-title">{{ conv.title || t('chat.newChat') }}</div>
            <div class="conv-meta">{{ conv.model_type }}</div>
          </div>
          <div class="conv-actions">
            <el-tooltip :content="t('promptEngineering.viewPrompts')" placement="right">
              <el-button 
                size="small" 
                circle
                @click.stop="viewConversationPrompts(conv.id.toString())"
              >
                <el-icon><Document /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>
      <div class="sidebar-footer" v-show="!sidebarCollapsed">
        <div class="feature-buttons">
          <el-button @click="$router.push('/programming')">{{ t('aiChat.aiProgramming') }}</el-button>
          <el-button @click="$router.push('/video')">{{ t('aiChat.videoGeneration') }}</el-button>
          <el-button @click="$router.push('/ai-chat')">{{ t('aiChat.aiAiChat') }}</el-button>
        </div>
        <div class="footer-actions">
          <ThemeToggle />
          <el-dropdown @command="handleUserCommand">
            <span class="user-info">
              {{ user?.username }}
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="settings">{{ t('settings.title') }}</el-dropdown-item>
                <el-dropdown-item command="logout">{{ t('auth.login') }}</el-dropdown-item>
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
          <h1>{{ t('chat.title') }}</h1>
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
                  <span>{{ t('common.search') }} Results</span>
                </div>
                <div class="search-results-content">
                  <div v-for="(result, idx) in msg.search_results" :key="idx" class="search-result-item">
                    <a :href="result.url" target="_blank" class="result-title">{{ result.title }}</a>
                    <p class="result-snippet">{{ result.snippet }}</p>
                  </div>
                </div>
              </div>
              <!-- Optimization process toggle for saved messages -->
              <div class="message-header" v-if="msg.content.includes('optimization-section')">
                <div class="optimization-progress">
                  <span>提示词优化过程</span>
                  <el-button 
                    size="small" 
                    type="text" 
                    @click="toggleMessageOptimizationCollapse(msg.id)"
                    class="toggle-optimization-btn"
                  >
                    {{ messageOptimizationCollapsed.get(msg.id) ? '展开' : '折叠' }}优化过程
                  </el-button>
                </div>
              </div>
              <div 
                class="message-content" 
                :class="{ 'hide-optimization': messageOptimizationCollapsed.get(msg.id) }" 
                v-html="renderMarkdown(msg.content)"
              ></div>
              <div v-if="msg.role === 'assistant'" class="message-actions">
                <el-tooltip :content="t('common.test')" placement="top">
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
                  <span>{{ t('common.search') }} Results</span>
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
            <div class="message-header" v-if="optimizationProgress > 0 || streamingContent.includes('optimization-section')">
              <div class="optimization-progress">
                <span>提示词优化进度: {{ optimizationProgress }}%</span>
                <el-button 
                  size="small" 
                  type="text" 
                  @click="showOptimizationDetails = !showOptimizationDetails"
                  class="toggle-optimization-btn"
                >
                  {{ showOptimizationDetails ? '折叠' : '展开' }}优化过程
                </el-button>
              </div>
            </div>
            <div class="message-content" :class="{ 'hide-optimization': !showOptimizationDetails }" v-html="renderMarkdown(streamingContent)"></div>
          </div>
          <div v-else-if="isWaitingForResponse" class="message-wrapper assistant">
            <div class="message-content loading-container">
              <div class="typing-indicator">
                <span class="typing-dot"></span>
                <span class="typing-dot"></span>
                <span class="typing-dot"></span>
              </div>
              <span class="thinking-text">{{ t('chat.thinking') }}</span>
            </div>
          </div>
        </template>
      </div>

      <div class="input-area">
        <!-- System Prompt Indicator -->
        <div v-if="systemPrompt" class="system-prompt-indicator">
          <el-tag type="info" size="small" closable @close="systemPrompt = ''">
            <el-icon><Setting /></el-icon>
            {{ t('settings.configuration') }} Active
          </el-tag>
        </div>
        <div class="input-actions">
          <PromptTemplateSelector @insert="handleTemplateInsert" :compact="false" />
          <el-select v-model="selectedSearchProvider" :placeholder="t('common.search')" size="small" class="toolbar-select">
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
            <el-option :label="t('common.disabled')" value="" />
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
                <span>{{ ragEnabled ? t('common.enabled') : 'RAG' }}</span>
              </el-button>
            </template>
            <div class="rag-popover">
              <div class="rag-toggle">
                <span>{{ t('common.enabled') }} Knowledge Base</span>
                <el-switch v-model="ragEnabled" />
              </div>
              <div v-if="ragEnabled" class="rag-docs">
                <p class="rag-docs-label">{{ t('common.selectItem') }}:</p>
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
                <el-empty v-else :description="t('settings.noConfig')" :image-size="60" />
                <el-button size="small" type="primary" text @click="$router.push('/rag')">
                  {{ t('common.configuration') }} Documents
                </el-button>
              </div>
            </div>
          </el-popover>
          <!-- Prompt Engineering Toggle -->
          <el-popover ref="promptEngineeringPopover" placement="top" :width="450" trigger="click" :hide-on-click="false">
            <template #reference>
              <el-button 
                size="small" 
                class="toolbar-btn"
                :class="{ 'toolbar-btn-active': promptEngineeringConfig.enabled }"
              >
                <el-icon><MagicStick /></el-icon>
                <span>{{ t('promptEngineering.title') }}</span>
              </el-button>
            </template>
            <PromptEngineeringPanel 
              :config="promptEngineeringConfig"
              @save="handlePromptEngineeringConfigSave"
              @close="handlePromptEngineeringClose"
            />
          </el-popover>
          <el-select v-model="selectedMCPTool" :placeholder="t('common.pleaseSelect')" size="small" class="toolbar-select">
            <template #prefix>
              <el-icon><SetUp /></el-icon>
            </template>
            <el-option :label="t('common.disabled')" value="" />
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
            <span>{{ t('common.export') }}</span>
          </el-button>
        </div>
        <div class="input-row">
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="3"
            :placeholder="t('chat.typeMessage')"
            @keydown.enter.prevent="handleSend"
          />
          <el-tooltip :content="isListening ? t('common.stop') : (speechSupported ? 'Voice input' : 'Voice input not supported in this browser')"
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
          <el-dropdown @command="handleVoiceSelection">
            <el-button circle>
              <el-icon><Bell /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <!-- 国语 -->
                <el-dropdown-item command="xiaoyun">小芸 (标准女声)</el-dropdown-item>
                <el-dropdown-item command="xiaogang">小刚 (标准男声)</el-dropdown-item>
                <el-dropdown-item command="zhixiaomei">知小梅 (通用女音)</el-dropdown-item>
                <el-dropdown-item command="zhishuo">知说 (通用男音)</el-dropdown-item>
                <el-dropdown-item command="ruoxi">若曦 (温柔女声)</el-dropdown-item>
                <el-dropdown-item command="sicheng">思成 (标准男声)</el-dropdown-item>
                <!-- 粤语 -->
                <el-dropdown-item command="abin">阿宾 (广西通用语)</el-dropdown-item>
                <el-dropdown-item command="shanshan">姗姗 (粤语女声)</el-dropdown-item>
                <!-- 英语 -->
                <el-dropdown-item command="cally">Cally (美式英语女音)</el-dropdown-item>
                <el-dropdown-item command="eric">Eric (英语男音)</el-dropdown-item>
                <el-dropdown-item command="lydia">Lydia (英语双语女声)</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <!-- Interrupt button - only shown during streaming -->
          <el-tooltip :content="t('chat.interrupt')" placement="top">
            <el-button
              v-if="isStreaming"
              type="danger"
              circle
              @click="handleInterrupt"
            >
              <el-icon><CircleClose /></el-icon>
            </el-button>
          </el-tooltip>
          <el-button 
            type="primary" 
            :disabled="!inputText.trim() || isStreaming"
            @click="handleSend"
          >
            {{ t('chat.sendMessage') }}
          </el-button>
        </div>
      </div>
    </main>

    <aside class="context-sidebar" :class="{ collapsed: contextSidebarCollapsed }">
      <div class="context-sidebar-header" v-show="!contextSidebarCollapsed">
        <h3>{{ t('settings.title') }}</h3>
        <el-tooltip :content="t('common.close')">
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
            <strong>{{ t('chat.title') }}:</strong>
            <div class="context-text" v-html="renderMarkdown(lastQuestion)"></div>
          </div>
          <div class="context-answer">
            <strong>AI {{ t('chat.title') }}:</strong>
            <div class="context-text" v-html="renderMarkdown(lastAnswer)"></div>
          </div>
        </div>
        <div v-else class="empty-context">
          <p>{{ t('settings.noConfig') }}</p>
        </div>
      </div>
      <div class="collapsed-expand-btn" v-show="contextSidebarCollapsed" @click="toggleContextSidebar">
        <el-icon><Fold /></el-icon>
      </div>
    </aside>

    <!-- Prompt Files Dialog -->
    <el-dialog
      v-model="showPromptDialog"
      :title="t('promptEngineering.viewPrompts')"
      width="800px"
      @close="closePromptDialog"
    >
      <div class="prompt-dialog-content">
        <div class="prompt-files-list">
          <h4>{{ t('promptEngineering.promptFiles') }}</h4>
          <el-tree
            :data="promptFiles.map(file => ({ label: file, value: file }))"
            :props="{ label: 'label', value: 'value' }"
            @node-click="loadPromptFile"
            :default-expanded-keys="[]"
          />
        </div>
        <div class="prompt-file-content">
          <h4 v-if="selectedPromptFile">{{ selectedPromptFile }}</h4>
          <div v-else class="empty-prompt">
            {{ t('promptEngineering.selectPromptFile') }}
          </div>
          <el-card v-if="promptFileContent" class="prompt-content-card">
            <div v-html="renderMarkdown(promptFileContent)"></div>
          </el-card>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, Search, DArrowLeft, DArrowRight, Fold, Microphone, Document, List, SetUp, Setting, Bell, MagicStick, CircleClose } from '@element-plus/icons-vue'
import PromptTemplateSelector from '../components/PromptTemplateSelector.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import BranchPanel from '../components/chat/BranchPanel.vue'
import MessageEditor from '../components/chat/MessageEditor.vue'
import ParallelExplorer from '../components/chat/ParallelExplorer.vue'
import PromptEngineeringPanel from '../components/prompt-engineering/PromptEngineeringPanel.vue'

const { t } = useI18n()

console.log('[ChatView] Script loaded - test log')

const authStore = useAuthStore()
const chatStore = useChatStore()
const router = useRouter()

console.log('[ChatView] Auth store state:', {
  user: authStore.user,
  token: authStore.token,
  tokenLength: authStore.token.length
})

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
const showOptimizationDetails = ref(true)
const abortController = ref<AbortController | null>(null)
const optimizationProgress = ref(0)
const streamingContent = ref('')
// Map to store collapse state for each saved message with optimization
const messageOptimizationCollapsed = ref<Map<number, boolean>>(new Map())
const currentSearchResults = ref<Array<{title: string, snippet: string, url: string}>>([])
const sidebarCollapsed = ref(false)
const contextSidebarCollapsed = ref(false)
const branchPanelRef = ref<InstanceType<typeof BranchPanel> | null>(null)
const promptEngineeringPopover = ref<any>(null)
const activeBranchId = ref<string | null>(null)
const systemPrompt = ref('')

// Prompt Engineering Configuration
const promptEngineeringConfig = ref({
  enabled: true,
  role: 'default',
  customRoleInfo: '',
  outputFormat: 'default',
  schema: '',
  chainOfThought: false,
  maxChains: 3,
  toolCalls: false,
  selfEvaluation: false,
  promptOptimizationEnabled: true,
  promptOptimizationLevel: 'medium',
  enableContextEnhancement: true,
  enableIntentUnderstanding: true
})

// Prompt files dialog
const showPromptDialog = ref(false)
const promptFiles = ref([])
const selectedPromptFile = ref('')
const promptFileContent = ref('')
const currentConversationId = ref('')

// Handle prompt template insertion
const handleTemplateInsert = (prompt: string, target: 'user' | 'system' = 'user') => {
  if (target === 'system') {
    systemPrompt.value = prompt
  } else {
    inputText.value = prompt
  }
}

// Handle prompt engineering config save
const handlePromptEngineeringConfigSave = (config: typeof promptEngineeringConfig.value) => {
  promptEngineeringConfig.value = config
  ElMessage.success('Prompt engineering configuration saved')
  // Close the popover after saving
  promptEngineeringPopover.value?.hide()
}

// Handle prompt engineering close
const handlePromptEngineeringClose = () => {
  promptEngineeringPopover.value?.hide()
}

// View conversation prompts
const viewConversationPrompts = async (conversationId: string) => {
  currentConversationId.value = conversationId
  try {
    const response = await fetch(`/api/v1/chat/conversations/${conversationId}/prompts`, {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      promptFiles.value = data.files
      showPromptDialog.value = true
    } else {
      ElMessage.error('Failed to load prompt files')
    }
  } catch (error) {
    console.error('Error loading prompt files:', error)
    ElMessage.error('Failed to load prompt files')
  }
}

// Load prompt file content
const loadPromptFile = async (node: any) => {
  const fileName = typeof node === 'string' ? node : node.value
  selectedPromptFile.value = fileName
  try {
    const response = await fetch(`/api/v1/chat/conversations/${currentConversationId.value}/prompts/${fileName}`, {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      promptFileContent.value = data.content
    } else {
      ElMessage.error('Failed to load prompt file content')
    }
  } catch (error) {
    console.error('Error loading prompt file content:', error)
    ElMessage.error('Failed to load prompt file content')
  }
}

// Close prompt dialog
const closePromptDialog = () => {
  showPromptDialog.value = false
  promptFiles.value = []
  selectedPromptFile.value = ''
  promptFileContent.value = ''
  currentConversationId.value = ''
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
  
  // Create new AbortController for this request
  abortController.value = new AbortController()
  
  try {
    const response = await fetch(`/api/v1/chat/messages/${data.messageId}/regenerate/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ model: data.model }),
      signal: abortController.value.signal // Pass the abort signal
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
    // Clean up abort controller
    abortController.value = null
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

// 检查 Electron 环境
const isElectron = typeof window !== 'undefined' && (window as any).electron
console.log('[Voice Input] Electron environment:', isElectron)

// TTS (Text-to-Speech)
const speakingMessageId = ref<number | null>(null)
let currentAudio: HTMLAudioElement | null = null
const selectedVoice = ref('xiaoyun') // 默认声音

const handleVoiceSelection = (voice: string) => {
  selectedVoice.value = voice
  ElMessage.success(`已选择声音: ${voice}`)
}

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
    console.log('[TTS] Auth token:', authStore.token)
    console.log('[TTS] Requesting speech for message:', msg.id)
    console.log('[TTS] Selected voice:', selectedVoice.value)
    const response = await fetch('/api/v1/tts/speak', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({
        text: msg.content,
        voice: selectedVoice.value,
        model: 'cosyvoice-v1'
      })
    })
    
    console.log('[TTS] Response status:', response.status)
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
  console.log('[Voice Input] Starting voice input...')
  console.log('[Voice Input] speechSupported:', speechSupported.value)
  console.log('[Voice Input] isListening:', isListening.value)
  
  if (!speechSupported.value) {
    console.log('[Voice Input] Speech not supported')
    ElMessage.warning('Voice input is not supported in this browser. Please use Chrome or Edge.')
    return
  }
  if (isListening.value) {
    console.log('[Voice Input] Stopping current recognition')
    recognizer?.stop()
    isListening.value = false
    return
  }
  
  try {
    const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
    console.log('[Voice Input] SpeechRecognition API:', SR)
    
    recognizer = new SR()
    console.log('[Voice Input] Recognizer created:', recognizer)
    
    recognizer.lang = 'zh-CN'
    recognizer.interimResults = false
    recognizer.maxAlternatives = 1
    
    recognizer.onresult = (e: any) => {
      console.log('[Voice Input] Recognition result:', e)
      const transcript = e.results[0][0].transcript
      console.log('[Voice Input] Transcript:', transcript)
      inputText.value += (inputText.value ? ' ' : '') + transcript
    }
    
    recognizer.onend = () => {
      console.log('[Voice Input] Recognition ended')
      isListening.value = false
    }
    
    recognizer.onerror = (error: any) => {
      console.error('[Voice Input] Recognition error:', error)
      isListening.value = false
      ElMessage.error('Voice recognition error: ' + error.message)
    }
    
    console.log('[Voice Input] Starting recognition')
    recognizer.start()
    isListening.value = true
    console.log('[Voice Input] Recognition started')
  } catch (error) {
    console.error('[Voice Input] Error starting voice input:', error)
    isListening.value = false
    ElMessage.error('Failed to start voice input: ' + (error as any).message)
  }
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

// 处理点击后续问题
const handleFollowUpQuestionClick = (question: string) => {
  console.log('[ChatView] Follow-up question clicked:', question)
  // 将问题填充到输入框
  inputText.value = question
  // 自动聚焦到输入框
  setTimeout(() => {
    const textarea = document.querySelector('.input-row textarea') as HTMLTextAreaElement
    if (textarea) {
      textarea.focus()
      // 将光标移动到文本末尾
      textarea.setSelectionRange(textarea.value.length, textarea.value.length)
    }
  }, 50)
}

onMounted(async () => {
  await chatStore.fetchConversations()
  await fetchAvailableModels()
  await fetchAvailableSearchProviders()
  await fetchAvailableMCPTools()
  await fetchRagDocuments()
  
  // 添加点击事件监听器用于处理后续问题点击
  setTimeout(() => {
    if (messageBox.value) {
      messageBox.value.addEventListener('click', (event) => {
        const target = event.target as HTMLElement
        // 检查是否点击了可点击的后续问题
        const clickableElement = target.closest('.follow-up-question-clickable')
        if (clickableElement) {
          event.preventDefault()
          const question = clickableElement.getAttribute('data-question')
          if (question) {
            handleFollowUpQuestionClick(question)
          }
        }
      })
    }
  }, 500) // 稍微延迟以确保DOM完全加载
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
    return `<pre style="max-width: calc(100vw - 140px); word-break: break-all; white-space: pre-wrap;"><code class="hljs ${language}">${highlighted}</code></pre>\n`
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
      
      // 进一步处理，给后续问题的列表项添加点击功能
      // 查找 <ol> 或 <ul> 中的 <li> 元素，并添加 clickable 类
      // 匹配格式：1. [问题] 或 1. 问题 或 - 问题 或 * 问题
      const liPattern = /(<li[^>]*>)(\s*)(\d+\.\s*|[-•*]\s*)?(\s*\[?)([^<\[\]]+?)(\]?\s*)(<\/li>)/gi
      html = html.replace(liPattern, (match, openingTag, space1, prefix, bracketOpen, questionText, bracketClose, closingTag) => {
        // 清理问题文本（去掉可能的方括号和多余空格）
        const cleanedQuestion = questionText.trim()
        return `${openingTag}${space1}<span class="follow-up-question-clickable" data-question="${escapeHtml(cleanedQuestion)}">${prefix || ''}${bracketOpen || ''}${questionText}${bracketClose || ''}</span>${closingTag}`
      })
    } else {
      console.log('[ChatView] No match found in HTML')
    }
  }
  
  return html
}

// HTML转义辅助函数
const escapeHtml = (text: string): string => {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

// 获取步骤名称的辅助函数
const getStepName = (step: string): string => {
  const stepNames: Record<string, string> = {
    'intent_understanding': '意图理解',
    'context_enhancement': '上下文增强',
    'prompt_refinement': '提示词精炼'
  }
  return stepNames[step] || step
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
  
  // Create new AbortController for this request
  abortController.value = new AbortController()
  
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
      system_prompt: systemPrompt.value || undefined,
      promptEngineeringConfig: promptEngineeringConfig.value.enabled ? promptEngineeringConfig.value : undefined
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
      body: JSON.stringify(requestBody),
      signal: abortController.value.signal // Pass the abort signal
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
          // 处理提示词优化事件
          if (data.type && data.type.startsWith('prompt_optimization_')) {
            isWaitingForResponse.value = false
            
            // 根据事件类型处理
            if (data.type === 'prompt_optimization_start') {
              // 重置优化状态
              showOptimizationDetails.value = true
              optimizationProgress.value = 0
              // 添加开始标记
              streamingContent.value += '<div class="optimization-section">\n'
            }
            
            // 更新进度信息
            if (data.metadata && data.metadata.progress) {
              optimizationProgress.value = data.metadata.progress
            }
            
            // 特殊处理优化内容事件
            if (data.type === 'prompt_optimization_content') {
              // 显示优化内容（使用代码块格式）
              const stepName = data.metadata && data.metadata.step ? getStepName(data.metadata.step) : '优化'
              streamingContent.value += `<div class="optimization-content" data-step="${data.metadata?.step || ''}">
  <div class="optimization-content-header">${stepName}结果：</div>
  <pre class="optimized-prompt">${escapeHtml(data.content)}</pre>
</div>\n`
            } else {
              // 添加优化事件内容（带有步骤标记）
              const stepClass = data.type === 'prompt_optimization_step' ? 'optimization-step' : 'optimization-event'
              const stepInfo = data.metadata && data.metadata.step ? ` data-step="${data.metadata.step}"` : ''
              streamingContent.value += `<div class="${stepClass}"${stepInfo}>${data.content}</div>\n`
            }
            
            if (data.type === 'prompt_optimization_complete') {
              // 添加结束标记
              streamingContent.value += '</div>\n'
              // 优化完成，准备折叠
              showOptimizationDetails.value = false
            }
            
            hasContent = true
            scrollToBottom()
            continue
          }
          // 处理错误事件
          if (data.type === 'error') {
            isWaitingForResponse.value = false
            streamingContent.value += data.content + '\n'
            hasContent = true
            scrollToBottom()
            continue
          }
          if (data.content) {
            isWaitingForResponse.value = false
            // AI响应开始，折叠优化过程
            showOptimizationDetails.value = false
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
    // Clean up abort controller
    abortController.value = null
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

// Handle interrupting model execution
const handleInterrupt = () => {
  if (abortController.value) {
    console.log('[ChatView] Interrupting model execution...')
    abortController.value.abort()
    abortController.value = null
    
    // Update UI state
    isStreaming.value = false
    isWaitingForResponse.value = false
    
    // Show interrupt message
    ElMessage.success('Model execution interrupted')
  }
}

// Toggle optimization process collapse for saved messages
const toggleMessageOptimizationCollapse = (messageId: number) => {
  // Get current state or default to false (expanded)
  const currentState = messageOptimizationCollapsed.value.get(messageId) || false
  // Toggle the state
  messageOptimizationCollapsed.value.set(messageId, !currentState)
  console.log(`[ChatView] Toggle optimization collapse for message ${messageId}: ${!currentState}`)
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
  width: 100%;
  max-width: 100vw;
  overflow: hidden;
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
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.conv-content {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.conv-actions {
  margin-left: 10px;
  opacity: 0;
  transition: opacity 0.2s;
}

.conversation-item:hover .conv-actions {
  opacity: 1;
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
  min-width: 0; /* 允许flex项目收缩 */
  width: 100%;
}
.messages {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 20px;
  max-width: 100%;
  box-sizing: border-box;
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
  max-width: 90%;
  gap: 10px;
  width: fit-content;
  box-sizing: border-box;
}
.message-content {
  padding: 10px 15px;
  border-radius: 8px;
  max-width: calc(100vw - 100px); /* 基于视口的硬限制 */
  overflow-wrap: anywhere;
  word-break: break-all; /* 强制断行，包括单词内部 */
  white-space: normal;
  box-sizing: border-box;
  display: block;
  position: relative;
  overflow: hidden;
}

/* 确保所有子元素都不会溢出 */
.message-content * {
  max-width: 100% !important;
  box-sizing: border-box !important;
  overflow: hidden !important;
  word-break: break-all !important;
  white-space: normal !important;
}

/* 特别处理图像和媒体 */
.message-content img,
.message-content video,
.message-content iframe {
  max-width: 100% !important;
  height: auto !important;
  display: block !important;
  margin: 0 auto !important;
}

/* 处理表格溢出 */
.message-content table {
  display: block !important;
  overflow-x: auto !important;
  white-space: nowrap !important;
  max-width: 100% !important;
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
  opacity: 1;
  transition: opacity 0.2s;
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
  width: 100%;
  box-sizing: border-box;
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
  width: 100%;
  align-items: center;
}

.input-row textarea {
  flex: 1;
  min-width: 0; /* 允许textarea收缩 */
  resize: vertical;
}

/* Override markdown styling for context sidebar */
.context-text pre {
  background-color: var(--code-bg);
  border-radius: 3px;
  padding: 8px;
  overflow-x: hidden;
  overflow-y: auto;
  font-size: 12px;
  margin: 5px 0;
  max-width: calc(100vw - 140px);
  word-break: break-all;
  white-space: pre-wrap;
  box-sizing: border-box;
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

/* Prompt Files Dialog */
.prompt-dialog-content {
  display: flex;
  gap: 20px;
  height: 500px;
}

.prompt-files-list {
  flex: 1;
  max-width: 300px;
  border-right: 1px solid var(--border-primary);
  padding-right: 20px;
  overflow-y: auto;
}

.prompt-file-content {
  flex: 2;
  overflow-y: auto;
}

.empty-prompt {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
  color: var(--text-tertiary);
  font-style: italic;
}

.prompt-content-card {
  margin-top: 10px;
  max-height: 400px;
  overflow-y: auto;
}

.prompt-content-card :deep(.el-card__body) {
  padding: 15px;
}

.prompt-content-card pre {
  background-color: var(--code-bg);
  border-radius: 3px;
  padding: 8px;
  overflow-x: hidden;
  overflow-y: auto;
  font-size: 12px;
  margin: 5px 0;
  max-width: calc(100vw - 140px);
  word-break: break-all;
  white-space: pre-wrap;
  box-sizing: border-box;
}

.prompt-content-card code {
  background-color: var(--code-bg);
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 13px;
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
    height: auto;
    min-height: 100vh;
  }
  
  .chat-main {
    width: 100%;
  }
  
  .message-container {
    max-width: 95%;
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
  
  /* 移动端优化代码块显示 */
  .optimized-prompt {
    font-size: 11px;
    max-height: 150px;
    padding: 8px;
  }
  
  .optimization-step,
  .optimization-event {
    padding-left: 15px;
    font-size: 13px;
  }
  
  /* 移动端消息容器优化 */
  .message-container {
    max-width: 100% !important;
  }
  
  .message-wrapper.user .message-container {
    max-width: 100% !important;
  }
  
  .input-row {
    flex-wrap: wrap;
  }
  
  .input-row .el-button:not(:first-child) {
    margin-top: 8px;
  }
  
  /* 移动端工具栏按钮优化 */
  .toolbar-btn {
    padding: 6px 10px !important;
    font-size: 12px !important;
  }
}
/* 提示词优化样式 */
.optimization-progress {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: #f0f9ff;
  border-radius: 6px;
  margin-bottom: 8px;
  font-size: 14px;
  color: #409eff;
}

.toggle-optimization-btn {
  padding: 2px 8px;
  font-size: 12px;
}

.hide-optimization .optimization-section {
  max-height: 30px; /* 保留最小高度，显示摘要 */
  opacity: 0.8; /* 保持可见但区分状态 */
  overflow: hidden;
  transition: max-height 0.3s cubic-bezier(0.4, 0, 0.2, 1), 
              opacity 0.3s ease;
  margin: 2px 0;
  padding: 2px 0 8px 20px; /* 左侧缩进对齐 */
  pointer-events: auto; /* 保持可点击性 */
  border-left: 3px solid #409eff;
  background: linear-gradient(to bottom, rgba(240, 249, 255, 0.6), rgba(240, 249, 255, 0));
}

/* 折叠状态下的优化内容隐藏 */
.hide-optimization .optimization-content {
  display: none;
}

/* 折叠状态下的优化步骤仅显示第一个 */
.hide-optimization .optimization-step:not(:first-child),
.hide-optimization .optimization-event {
  display: none;
}

/* 折叠状态下添加省略号提示 */
.hide-optimization .optimization-section::after {
  content: '...';
  position: absolute;
  bottom: 0;
  right: 20px;
  color: #409eff;
  font-weight: bold;
  background: var(--bg-primary);
  padding-left: 4px;
}

.optimization-section {
  max-height: 5000px;
  opacity: 1;
  transform: translateY(0);
  transition: max-height 0.4s cubic-bezier(0.4, 0, 0.2, 1), 
              opacity 0.3s ease;
  overflow: hidden;
  position: relative; /* 为伪元素定位提供基准 */
  margin: 8px 0;
}

.optimization-step {
  padding-left: 20px;
  margin: 4px 0;
  color: #67c23a;
}

.optimization-event {
  padding-left: 20px;
  margin: 4px 0;
  color: #909399;
}

/* 优化内容样式 */
.optimization-content {
  margin: 8px 0 12px 20px;
  border-left: 3px solid #409eff;
  padding-left: 12px;
}

.optimization-content-header {
  font-size: 13px;
  font-weight: 500;
  color: #409eff;
  margin-bottom: 6px;
}

.optimized-prompt {
  background-color: #f6f8fa;
  border-radius: 6px;
  padding: 10px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow-wrap: anywhere;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
  overflow-x: hidden;
  border: 1px solid #e1e4e8;
  margin: 0;
  max-width: calc(100vw - 140px);
  box-sizing: border-box;
}

/* 可点击的后续问题样式 */
:deep(.follow-up-question-clickable) {
  cursor: pointer;
  color: #409eff;
  text-decoration: none;
  transition: color 0.2s ease;
}

:deep(.follow-up-question-clickable:hover) {
  color: #66b1ff;
  text-decoration: underline;
}

</style>
