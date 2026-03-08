<template>
  <div class="ai-programming-layout">
    <!-- Sidebar -->
    <aside class="sidebar">
      <!-- Logo/Brand -->
      <div class="sidebar-brand">
        <div class="brand-icon">
          <el-icon :size="24"><Cpu /></el-icon>
        </div>
        <span class="brand-text">AI Code</span>
      </div>

      <!-- Navigation Icons -->
      <nav class="sidebar-nav">
        <el-tooltip content="Chat" placement="right">
          <div class="nav-item" @click="$router.push('/')">
            <el-icon :size="20"><ChatDotRound /></el-icon>
          </div>
        </el-tooltip>
        <el-tooltip content="AI-AI Chat" placement="right">
          <div class="nav-item" @click="$router.push('/ai-chat')">
            <el-icon :size="20"><Connection /></el-icon>
          </div>
        </el-tooltip>
        <el-tooltip content="Video" placement="right">
          <div class="nav-item" @click="$router.push('/video')">
            <el-icon :size="20"><VideoCamera /></el-icon>
          </div>
        </el-tooltip>
      </nav>

      <!-- Bottom Actions -->
      <div class="sidebar-bottom">
        <ThemeToggle />
        <el-tooltip content="Settings" placement="right">
          <div class="nav-item" @click="showAgentEditor = true">
            <el-icon :size="20"><Setting /></el-icon>
          </div>
        </el-tooltip>
        <el-tooltip content="Enhanced Features" placement="right">
          <div class="nav-item" :class="{ active: showEnhancedPanel }" @click="showEnhancedPanel = !showEnhancedPanel">
            <el-icon :size="20"><Operation /></el-icon>
          </div>
        </el-tooltip>
      </div>
    </aside>

    <!-- Control Panel -->
    <aside class="control-panel">
      <!-- Task Input Card -->
      <div class="task-card">
        <div class="task-header">
          <el-icon :size="16"><Edit /></el-icon>
          <span>Task Description</span>
        </div>
        <el-input
          v-model="taskDescription"
          type="textarea"
          :rows="4"
          placeholder="Describe what you want to build..."
          resize="none"
          class="task-input"
        />
        <div class="task-actions">
          <el-select v-model="selectedModel" size="small" class="model-select" @change="onModelChange">
            <el-option-group label="DeepSeek">
              <el-option label="Reasoner" value="deepseek-reasoner" />
              <el-option label="Chat" value="deepseek-chat" />
            </el-option-group>
            <el-option-group label="Hunyuan">
              <el-option label="Lite" value="hunyuan-lite" />
              <el-option label="Standard" value="hunyuan-standard" />
            </el-option-group>
            <el-option-group label="Qwen">
              <el-option label="Max" value="qwen-max" />
              <el-option label="Plus" value="qwen-plus" />
            </el-option-group>
          </el-select>
          <el-button 
            v-if="!isProcessing"
            type="primary" 
            @click="startTask"
            :disabled="!taskDescription.trim()"
            class="run-btn"
          >
            <el-icon><CaretRight /></el-icon>
            Run
          </el-button>
          <el-button 
            v-else
            type="danger" 
            @click="stopTask"
            class="stop-btn"
          >
            <el-icon><VideoPause /></el-icon>
            Stop
          </el-button>
        </div>
      </div>

      <!-- Active Agents -->
      <div class="section" v-if="activeAgents.length > 0">
        <div class="section-header">
          <el-icon :size="14"><User /></el-icon>
          <span>Active Agents</span>
          <span class="badge">{{ activeAgents.length }}</span>
        </div>
        <div class="agents-list">
          <div v-for="agent in activeAgents" :key="agent.id" class="agent-chip">
            <span class="agent-dot" :class="agent.role"></span>
            <span class="agent-name">{{ agent.name }}</span>
          </div>
        </div>
      </div>

      <!-- Execution Log -->
      <div class="section log-section">
        <div class="section-header">
          <el-icon :size="14"><Document /></el-icon>
          <span>Activity</span>
          <span class="badge" v-if="executionLogs.length">{{ executionLogs.length }}</span>
        </div>
        <div class="log-list" ref="logContainer">
          <div 
            v-for="(log, index) in executionLogs" 
            :key="index"
            :class="['log-item', log.type]"
          >
            <span class="log-indicator"></span>
            <span class="log-text">{{ log.message }}</span>
          </div>
          <div v-if="executionLogs.length === 0" class="log-empty">
            No activity yet
          </div>
        </div>
      </div>

      <!-- Tools Section -->
      <div class="section tools-section">
        <div class="section-header">
          <el-icon :size="14"><Box /></el-icon>
          <span>Tools</span>
        </div>
        <div class="tools-grid">
          <el-tooltip v-for="mcp in availableMCPs" :key="mcp.name" :content="mcp.name" placement="top">
            <div class="tool-chip" :class="{ connected: mcp.connected }">
              <el-icon :size="12"><Link /></el-icon>
            </div>
          </el-tooltip>
          <el-tooltip v-for="skill in availableSkills" :key="skill.name" :content="skill.name" placement="top">
            <div class="tool-chip skill">
              <el-icon :size="12"><MagicStick /></el-icon>
            </div>
          </el-tooltip>
        </div>
      </div>
    </aside>

    <!-- Main Workspace -->
    <main class="workspace">
      <!-- Workspace Header -->
      <header class="workspace-header">
        <div class="header-left">
          <el-button-group size="small">
            <el-button :type="showFileExplorer ? 'primary' : 'default'" @click="showFileExplorer = !showFileExplorer">
              <el-icon><FolderOpened /></el-icon>
            </el-button>
            <el-button @click="refreshFiles">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </el-button-group>
          <span class="path-display">{{ currentPath }}</span>
        </div>
        <div class="header-center">
          <div class="tab-bar" v-if="openTabs.length > 0">
            <div 
              v-for="tab in openTabs" 
              :key="tab.path"
              :class="['tab-item', { active: activeTab === tab.path }]"
              @click="activeTab = tab.path"
            >
              <el-icon :size="12"><Document /></el-icon>
              <span>{{ tab.name }}</span>
              <el-icon class="tab-close" :size="12" @click.stop="closeTab(tab.path)"><Close /></el-icon>
            </div>
          </div>
        </div>
        <div class="header-right">
          <div class="status-indicator" :class="{ processing: isProcessing }">
            <span class="status-dot"></span>
            <span>{{ isProcessing ? 'Running' : 'Ready' }}</span>
          </div>
        </div>
      </header>

      <!-- Editor Area -->
      <div class="editor-area">
        <!-- File Explorer -->
        <aside class="file-explorer" v-show="showFileExplorer">
          <div class="explorer-header">Explorer</div>
          <el-tree
            :data="fileTree"
            :props="defaultTreeProps"
            @node-click="handleFileSelect"
            highlight-current
            default-expand-all
            class="file-tree"
          >
            <template #default="{ node, data }">
              <span class="tree-node">
                <el-icon :size="14" v-if="data.isDirectory"><Folder /></el-icon>
                <el-icon :size="14" v-else><Document /></el-icon>
                <span>{{ node.label }}</span>
              </span>
            </template>
          </el-tree>
        </aside>

        <!-- Monaco Editor -->
        <div class="editor-main">
          <div ref="monacoContainer" class="monaco-container"></div>
        </div>
      </div>

      <!-- Terminal -->
      <div class="terminal-area" :style="{ height: terminalHeight + 'px' }">
        <div class="terminal-header" @mousedown="startResize">
          <div class="terminal-title">
            <el-icon :size="14"><Monitor /></el-icon>
            <span>Terminal</span>
          </div>
          <el-button size="small" text @click="clearTerminal">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
        <div ref="terminalContainer" class="terminal-body"></div>
      </div>
    </main>

    <!-- Agent Editor Dialog -->
    <el-dialog
      v-model="showAgentEditor"
      title="Agent Configuration"
      width="70%"
      :close-on-click-modal="false"
      class="config-dialog"
    >
      <el-tabs v-model="activeAgentTab" type="border-card">
        <el-tab-pane 
          v-for="agent in agentConfigs" 
          :key="agent.name"
          :label="agent.name"
          :name="agent.name"
        >
          <el-input
            v-model="agent.prompt"
            type="textarea"
            :rows="18"
            placeholder="Agent prompt configuration..."
          />
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="showAgentEditor = false">Cancel</el-button>
        <el-button type="primary" @click="saveAgentConfigs" :loading="savingAgents">
          Save
        </el-button>
      </template>
    </el-dialog>
    
    <!-- Enhanced Features Drawer -->
    <el-drawer
      v-model="showEnhancedPanel"
      title="Enhanced Features"
      direction="rtl"
      size="480px"
      class="enhanced-drawer"
    >
      <EnhancedPanel />
    </el-drawer>

    <!-- Task Results Modal -->
    <el-drawer
      v-model="showResults"
      title="Task Results"
      direction="rtl"
      size="500px"
      v-if="taskResults"
    >
      <div class="results-content">
        <div class="result-stat">
          <span class="stat-label">Duration</span>
          <span class="stat-value">{{ formatDuration(taskResults.duration) }}</span>
        </div>
        <div class="result-stat">
          <span class="stat-label">Files</span>
          <span class="stat-value">{{ taskResults.files_generated }}</span>
        </div>
        <div class="result-stat">
          <span class="stat-label">Iterations</span>
          <span class="stat-value">{{ taskResults.iterations }}</span>
        </div>
        <div class="result-files" v-if="Object.keys(taskResults.files || {}).length > 0">
          <h4>Generated Files</h4>
          <div class="file-chips">
            <span 
              v-for="(content, filePath) in taskResults.files" 
              :key="filePath"
              class="file-chip"
              @click="handleFileSelect({ path: filePath, isDirectory: false })"
            >
              {{ filePath }}
            </span>
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  CircleClose, CircleCheck, InfoFilled, User, Folder, FolderOpened, Document, 
  Refresh, Delete, Edit, Monitor, Operation, Cpu, ChatDotRound, Connection,
  VideoCamera, Setting, CaretRight, VideoPause, Box, Link, MagicStick, Close
} from '@element-plus/icons-vue'
import * as monaco from 'monaco-editor'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import EnhancedPanel from '@/components/ai-programming/EnhancedPanel.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'

// Types
interface AgentInfo {
  id: string
  name: string
  role: string
  action: string
}

interface AgentExecution {
  agent_name: string
  input: string
  output: string
  error?: string
  start_time: number
  end_time: number
}

interface TaskResults {
  task_id: string
  description: string
  start_time: number
  end_time: number
  duration: number
  files_generated: number
  iterations: number
  files: Record<string, string>
  agent_history: AgentExecution[]
  execution_log: any[]
}

interface LogEntry {
  type: 'info' | 'error' | 'success' | 'command' | 'agent'
  message: string
  timestamp: Date
}

interface FileNode {
  label: string
  path: string
  isDirectory: boolean
  children?: FileNode[]
}

interface OpenTab {
  name: string
  path: string
  content: string
  language: string
}

interface MCPServer {
  name: string
  connected: boolean
}

interface Skill {
  name: string
  description: string
}

interface AgentConfig {
  name: string
  role: string
  prompt: string
}

// State
const taskDescription = ref('')
const isProcessing = ref(false)
const activeAgents = ref<AgentInfo[]>([])
const executionLogs = ref<LogEntry[]>([])
const fileTree = ref<FileNode[]>([])
const openTabs = ref<OpenTab[]>([])
const activeTab = ref('')
const showFileExplorer = ref(true)
const currentPath = ref('/workspace')
const terminalHeight = ref(200)
const availableMCPs = ref<MCPServer[]>([])
const availableSkills = ref<Skill[]>([])
const activeToolCategories = ref(['mcp', 'skills'])
const selectedModel = ref('deepseek-reasoner')
const showAgentEditor = ref(false)
const activeAgentTab = ref('')
const agentConfigs = ref<AgentConfig[]>([])
const savingAgents = ref(false)
const isSimpleTask = ref(false)
const taskResults = ref<TaskResults | null>(null)
const activeResultSections = ref(['summary'])
const needsWsConnection = ref(false) // 控制是否需要保持WebSocket连接
const showEnhancedPanel = ref(false)
const showResults = ref(false)

const defaultTreeProps = {
  children: 'children',
  label: 'label'
}

// Refs
const monacoContainer = ref<HTMLElement>()
const terminalContainer = ref<HTMLElement>()
const logContainer = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null
let terminal: XTerm | null = null
let fitAddon: FitAddon | null = null
let wsConnection: WebSocket | null = null
let isConnecting = false // Flag to prevent duplicate connections

// Initialize
onMounted(() => {
  initMonaco()
  initTerminal()
})

onUnmounted(() => {
  editor?.dispose()
  terminal?.dispose()
  wsConnection?.close()
  isConnecting = false
})

// Monaco Editor
const initMonaco = () => {
  if (!monacoContainer.value) return
  
  editor = monaco.editor.create(monacoContainer.value, {
    value: '// Welcome to AI Programming Agent\n// Describe your task and click "Start Task"',
    language: 'javascript',
    theme: 'vs-dark',
    automaticLayout: true,
    minimap: { enabled: true },
    scrollBeyondLastLine: false,
    fontSize: 14,
    fontFamily: 'Consolas, "Courier New", monospace'
  })
}

// Terminal
const initTerminal = () => {
  if (!terminalContainer.value) return
  
  terminal = new XTerm({
    cursorBlink: true,
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      selectionBackground: '#264f78'
    },
    fontFamily: 'Consolas, "Courier New", monospace',
    fontSize: 14,
    scrollback: 10000
  })
  
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())
  terminal.open(terminalContainer.value)
  fitAddon.fit()

  // Terminal input handling
  let commandBuffer = ''
  terminal?.onData((data: string) => {
    const code = data.charCodeAt(0)
    
    if (data === '\r') { // Enter
      terminal?.writeln('')
      if (commandBuffer.trim()) {
        executeTerminalCommand(commandBuffer.trim())
      }
      commandBuffer = ''
      terminal?.write('$ ')
    } else if (code === 127) { // Backspace
      if (commandBuffer.length > 0) {
        commandBuffer = commandBuffer.slice(0, -1)
        terminal?.write('\b \b')
      }
    } else if (code >= 32 && code < 127) { // Printable
      commandBuffer += data
      terminal?.write(data)
    }
  })
  
  terminal?.writeln('AI Programming Terminal')
  terminal?.writeln('Type commands to interact with the workspace')
  terminal?.write('$ ')
}

// WebSocket
const connectWebSocket = () => {
  // Prevent duplicate connections
  if (wsConnection !== null && wsConnection.readyState === WebSocket.OPEN) {
    return
  }
  
  if (isConnecting) {
    return
  }
  
  isConnecting = true
  
  const token = localStorage.getItem('token')
  if (!token) {
    // No token, redirect to login
    window.location.href = '/login'
    isConnecting = false
    return
  }
  
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/ws/agent?token=${token}`
  
  wsConnection = new WebSocket(wsUrl)
  
  wsConnection.onopen = () => {
    addLog('info', 'Connected to AI Agent')
    isConnecting = false
    // Load available MCPs and Skills
    loadAvailableTools()
    // Load agent configurations
    loadAgentConfigs()
  }
  
  wsConnection.onmessage = (event) => {
    const data = JSON.parse(event.data)
    handleAgentMessage(data)
  }
  
  wsConnection.onerror = (error) => {
    console.error('WebSocket error:', error)
    addLog('error', 'Connection error')
    isConnecting = false
  }
  
  wsConnection.onclose = (event) => {
    addLog('info', 'Disconnected from agent')
    isConnecting = false
    
    // Check if close reason indicates token expiration
    if (event.code === 1008 || event.code === 1011) {
      // These codes can indicate policy violations or server errors (might be auth related)
      // Clear token and redirect to login
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
      return
    }
    
    // Only reconnect if token exists AND we still need the connection (e.g., task is running)
    if (localStorage.getItem('token') && needsWsConnection.value) {
      // Reconnect after 3 seconds
      setTimeout(connectWebSocket, 3000)
    }
  }
}

// Handle agent messages
const handleAgentMessage = (msg: any) => {
  // Backend wraps event data in a "data" field, but some messages (like mcp_discovered)
  // send data directly. Handle both cases.
  const eventType = msg.type
  const data = msg.data || msg // Use msg.data if available, otherwise use msg directly
  
  switch (eventType) {
    case 'task_started':
      // Handle both formats - some events have description, some have message
      const taskDesc = data.description || data.message || 'New task'
      addLog('info', `Task started: ${taskDesc}`)
      break
    case 'agent_status':
      updateAgentStatus(data)
      break
    case 'agent_started':
      activeAgents.value.push({
        id: data.id,
        name: data.name,
        role: data.role,
        action: data.action
      })
      addLog('agent', `${data.name} started: ${data.action}`)
      break
    case 'agent_completed':
      removeAgent(data.agentId)
      addLog('success', `${data.agentName} completed`)
      break
    case 'log':
      addLog(data.logType, data.message)
      break
    case 'terminal_output':
      terminal?.writeln(data.output)
      terminal?.write('$ ')
      break
    case 'code_update':
      updateCodeEditor(data.filePath, data.content, data.language)
      break
    case 'file_tree_update':
      fileTree.value = data.files
      break
    case 'task_complete':
      isProcessing.value = false
      activeAgents.value = []
      taskResults.value = data
      needsWsConnection.value = false // 任务完成，不需要保持连接
      addLog('success', 'Task completed successfully!')
      ElMessage.success('Task completed!')
      
      // 如果是简单任务，完成后重定向到聊天页面
      if (isSimpleTask.value) {
        setTimeout(() => {
          window.location.href = '/'
        }, 1500)
      } else {
        // 5秒后关闭WebSocket连接，让用户有时间查看结果
        setTimeout(() => {
          if (!needsWsConnection.value && wsConnection?.readyState === WebSocket.OPEN) {
            wsConnection.close()
          }
        }, 5000)
      }
      break
    case 'task_error':
      isProcessing.value = false
      needsWsConnection.value = false // 任务出错，不需要保持连接
      addLog('error', data.message)
      ElMessage.error(data.message)
      
      // 5秒后关闭WebSocket连接
      setTimeout(() => {
        if (!needsWsConnection.value && wsConnection?.readyState === WebSocket.OPEN) {
          wsConnection.close()
        }
      }, 5000)
      break
    case 'task_stopped':
      isProcessing.value = false
      activeAgents.value = []
      needsWsConnection.value = false // 任务停止，不需要保持连接
      addLog('info', data.message)
      ElMessage.info(data.message)
      
      // 5秒后关闭WebSocket连接
      setTimeout(() => {
        if (!needsWsConnection.value && wsConnection?.readyState === WebSocket.OPEN) {
          wsConnection.close()
        }
      }, 5000)
      break
    case 'mcp_discovered':
      availableMCPs.value = data.mcps
      break
    case 'skills_discovered':
      availableSkills.value = data.skills
      break
    case 'simple_task_detected':
      // 处理简单任务，记录信息但不立即重定向
      addLog('info', data.message)
      ElMessage.info(data.message)
      // 标记当前任务为简单任务
      isSimpleTask.value = true
      break
    case 'agent_configs':
      agentConfigs.value = data.configs
      if (data.configs?.length > 0) {
        activeAgentTab.value = data.configs[0].name
      }
      break
  }
}

// Model management
const onModelChange = (model: string) => {
  // 如果没有WebSocket连接，先建立连接
  if (wsConnection === null || wsConnection.readyState !== WebSocket.OPEN) {
    connectWebSocket()
    // 等待连接建立后发送请求
    setTimeout(() => {
      wsConnection?.send(JSON.stringify({
        type: 'set_model',
        model: model
      }))
      ElMessage.success(`Switched to ${model}`)
    }, 500)
  } else {
    wsConnection.send(JSON.stringify({
      type: 'set_model',
      model: model
    }))
    ElMessage.success(`Switched to ${model}`)
  }
}

// Task management
const startTask = () => {
  if (!taskDescription.value.trim()) {
    ElMessage.warning('Please describe your task')
    return
  }
  
  isProcessing.value = true
  executionLogs.value = []
  activeAgents.value = []
  taskResults.value = null
  isSimpleTask.value = false
  
  // 需要WebSocket连接来执行任务
  needsWsConnection.value = true
  
  // 确保WebSocket连接已建立
  if (wsConnection === null || wsConnection.readyState !== WebSocket.OPEN) {
    connectWebSocket()
    // 等待连接建立后发送任务
    setTimeout(() => {
      wsConnection?.send(JSON.stringify({
        type: 'start_task',
        task: taskDescription.value,
        model: selectedModel.value,
        context: {
          currentFiles: openTabs.value.map(t => ({ path: t.path, content: t.content })),
          os: detectOS()
        }
      }))
    }, 500)
  } else {
    wsConnection.send(JSON.stringify({
      type: 'start_task',
      task: taskDescription.value,
      model: selectedModel.value,
      context: {
        currentFiles: openTabs.value.map(t => ({ path: t.path, content: t.content })),
        os: detectOS()
      }
    }))
  }
}

const stopTask = () => {
  wsConnection?.send(JSON.stringify({
    type: 'stop_task'
  }))
  
  isProcessing.value = false
  activeAgents.value = []
  needsWsConnection.value = false // 任务停止，不需要保持连接
  addLog('info', 'Task stopped by user')
  ElMessage.info('Task stopped')
  
  // 5秒后关闭WebSocket连接，允许取消操作
  setTimeout(() => {
    if (!needsWsConnection.value && wsConnection?.readyState === WebSocket.OPEN) {
      wsConnection.close()
    }
  }, 5000)
}

const updateAgentStatus = (agent: AgentInfo) => {
  const idx = activeAgents.value.findIndex(a => a.id === agent.id)
  if (idx >= 0) {
    activeAgents.value[idx] = agent
  }
}

const removeAgent = (agentId: string) => {
  const idx = activeAgents.value.findIndex(a => a.id === agentId)
  if (idx >= 0) {
    activeAgents.value.splice(idx, 1)
  }
}

// Terminal commands
const executeTerminalCommand = (command: string) => {
  // 如果没有WebSocket连接，先建立连接
  if (wsConnection === null || wsConnection.readyState !== WebSocket.OPEN) {
    connectWebSocket()
    // 等待连接建立后发送请求
    setTimeout(() => {
      wsConnection?.send(JSON.stringify({
        type: 'execute_command',
        command: command,
        cwd: currentPath.value
      }))
    }, 500)
  } else {
    wsConnection.send(JSON.stringify({
      type: 'execute_command',
      command: command,
      cwd: currentPath.value
    }))
  }
}

const clearTerminal = () => {
  terminal?.clear()
  terminal?.write('$ ')
}

// Code editor
const updateCodeEditor = (filePath: string, content: string, language?: string) => {
  const fileName = filePath.split('/').pop() || filePath
  const ext = filePath.split('.').pop() || ''
  const langMap: Record<string, string> = {
    'js': 'javascript', 'ts': 'typescript', 'py': 'python',
    'go': 'go', 'java': 'java', 'json': 'json', 'md': 'markdown',
    'vue': 'html', 'html': 'html', 'css': 'css', 'scss': 'scss'
  }
  const detectedLang = language || langMap[ext] || 'plaintext'
  
  const existingTab = openTabs.value.find(t => t.path === filePath)
  
  if (existingTab) {
    existingTab.content = content
    if (activeTab.value === filePath && editor) {
      editor.setValue(content)
    }
  } else {
    openTabs.value.push({
      name: fileName,
      path: filePath,
      content,
      language: detectedLang
    })
    activeTab.value = filePath
    
    if (editor) {
      editor.setValue(content)
      monaco.editor.setModelLanguage(editor.getModel()!, detectedLang)
    }
  }
}

const handleFileSelect = (data: { path: string; isDirectory: boolean }) => {
  if (!data.isDirectory) {
    // 如果没有WebSocket连接，先建立连接
    if (wsConnection === null || wsConnection.readyState !== WebSocket.OPEN) {
      connectWebSocket()
      // 等待连接建立后发送请求
      setTimeout(() => {
        wsConnection?.send(JSON.stringify({
          type: 'read_file',
          path: data.path
        }))
      }, 500)
    } else {
      wsConnection.send(JSON.stringify({
        type: 'read_file',
        path: data.path
      }))
    }
  }
}

const closeTab = (path: string) => {
  const idx = openTabs.value.findIndex(t => t.path === path)
  if (idx >= 0) {
    openTabs.value.splice(idx, 1)
    if (activeTab.value === path && openTabs.value.length > 0) {
      activeTab.value = openTabs.value[0].path
      editor?.setValue(openTabs.value[0].content)
    }
  }
}

const refreshFiles = () => {
  // 如果没有WebSocket连接，先建立连接
  if (wsConnection === null || wsConnection.readyState !== WebSocket.OPEN) {
    connectWebSocket()
    // 等待连接建立后发送请求
    setTimeout(() => {
      wsConnection?.send(JSON.stringify({ type: 'refresh_files' }))
    }, 500)
  } else {
    wsConnection.send(JSON.stringify({ type: 'refresh_files' }))
  }
}

// Agent configs
const loadAgentConfigs = () => {
  wsConnection?.send(JSON.stringify({ type: 'get_agent_configs' }))
}

const saveAgentConfigs = () => {
  savingAgents.value = true
  
  // 如果没有WebSocket连接，先建立连接
  if (wsConnection === null || wsConnection.readyState !== WebSocket.OPEN) {
    connectWebSocket()
    // 等待连接建立后发送请求
    setTimeout(() => {
      wsConnection?.send(JSON.stringify({
        type: 'save_agent_configs',
        configs: agentConfigs.value
      }))
      setTimeout(() => {
        savingAgents.value = false
        showAgentEditor.value = false
        ElMessage.success('Agent configurations saved')
      }, 500)
    }, 500)
  } else {
    wsConnection.send(JSON.stringify({
      type: 'save_agent_configs',
      configs: agentConfigs.value
    }))
    setTimeout(() => {
      savingAgents.value = false
      showAgentEditor.value = false
      ElMessage.success('Agent configurations saved')
    }, 500)
  }
}

// Tools
const loadAvailableTools = () => {
  wsConnection?.send(JSON.stringify({ type: 'discover_tools' }))
}

// Helpers
const addLog = (type: LogEntry['type'], message: string) => {
  executionLogs.value.push({ type, message, timestamp: new Date() })
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

const detectOS = () => {
  const platform = window.navigator.platform.toLowerCase()
  if (platform.includes('win')) return 'windows'
  if (platform.includes('linux')) return 'linux'
  if (platform.includes('mac')) return 'darwin'
  return 'linux'
}

const formatTime = (date: Date) => {
  return date.toLocaleTimeString('en-US', { 
    hour12: false, 
    hour: '2-digit', 
    minute: '2-digit', 
    second: '2-digit' 
  })
}

const formatDuration = (duration: any) => {
  if (!duration) return '0s'
  
  let milliseconds: number
  
  // Handle different duration formats
  if (typeof duration === 'number') {
    milliseconds = duration
  } else if (typeof duration === 'object' && duration.seconds !== undefined) {
    // Handle Go duration format
    milliseconds = duration.seconds * 1000
    if (duration.nanoseconds) {
      milliseconds += duration.nanoseconds / 1000000
    }
  } else if (typeof duration === 'string') {
    // Try to parse ISO duration string
    try {
      const start = new Date(duration)
      const end = new Date()
      milliseconds = end.getTime() - start.getTime()
    } catch {
      milliseconds = 0
    }
  } else {
    milliseconds = 0
  }
  
  const seconds = Math.floor(milliseconds / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  
  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  } else {
    return `${seconds}s`
  }
}

const getAgentType = (role: string) => {
  const types: Record<string, any> = {
    'planner': 'primary',
    'coder': 'success',
    'debugger': 'warning',
    'executor': 'info',
    'reviewer': 'danger'
  }
  return types[role] || 'info'
}

const startResize = (e: MouseEvent) => {
  const startY = e.clientY
  const startHeight = terminalHeight.value
  
  const onMouseMove = (e: MouseEvent) => {
    const delta = startY - e.clientY
    terminalHeight.value = Math.max(100, Math.min(400, startHeight + delta))
    fitAddon?.fit()
  }
  
  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }
  
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// Watch for tab changes
watch(activeTab, (newPath) => {
  const tab = openTabs.value.find(t => t.path === newPath)
  if (tab && editor) {
    editor.setValue(tab.content)
    monaco.editor.setModelLanguage(editor.getModel()!, tab.language)
  }
})
</script>

<style>
/* Global reset */
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body { height: 100%; overflow: hidden; }
body { background: #0d1117; }
#app { height: 100%; }
</style>

<style scoped>
/* Layout */
.ai-programming-layout {
  display: flex;
  height: 100vh;
  background: #0d1117;
  color: #c9d1d9;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* Sidebar */
.sidebar {
  width: 56px;
  background: #010409;
  border-right: 1px solid #21262d;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 0;
}

.sidebar-brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  margin-bottom: 24px;
  color: #58a6ff;
}

.brand-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #238636 0%, #2ea043 100%);
  border-radius: 8px;
}

.brand-text {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-item {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  cursor: pointer;
  color: #8b949e;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background: #21262d;
  color: #c9d1d9;
}

.nav-item.active {
  background: #388bfd26;
  color: #58a6ff;
}

/* Control Panel */
.control-panel {
  width: 300px;
  background: #0d1117;
  border-right: 1px solid #21262d;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.task-card {
  padding: 16px;
  border-bottom: 1px solid #21262d;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #8b949e;
}

.task-input :deep(.el-textarea__inner) {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  color: #c9d1d9;
  font-size: 13px;
  resize: none;
}

.task-input :deep(.el-textarea__inner:focus) {
  border-color: #388bfd;
  box-shadow: 0 0 0 3px rgba(56, 139, 253, 0.15);
}

.task-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.model-select {
  flex: 1;
}

.model-select :deep(.el-input__wrapper) {
  background: #161b22;
  border-color: #30363d;
  box-shadow: none;
}

.run-btn, .stop-btn {
  min-width: 80px;
}

.run-btn {
  background: #238636;
  border-color: #238636;
}

.run-btn:hover {
  background: #2ea043;
  border-color: #2ea043;
}

/* Sections */
.section {
  padding: 12px 16px;
  border-bottom: 1px solid #21262d;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #8b949e;
}

.badge {
  background: #30363d;
  color: #c9d1d9;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 10px;
  margin-left: auto;
}

/* Agents */
.agents-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.agent-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 12px;
  font-size: 11px;
}

.agent-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #8b949e;
}

.agent-dot.planner { background: #58a6ff; }
.agent-dot.coder { background: #3fb950; }
.agent-dot.debugger { background: #d29922; }
.agent-dot.reviewer { background: #f85149; }

/* Log */
.log-section {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.log-list {
  flex: 1;
  overflow-y: auto;
  font-size: 12px;
  font-family: 'SF Mono', Monaco, 'Consolas', monospace;
}

.log-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid #21262d;
}

.log-indicator {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #8b949e;
  margin-top: 6px;
  flex-shrink: 0;
}

.log-item.error .log-indicator { background: #f85149; }
.log-item.success .log-indicator { background: #3fb950; }
.log-item.command .log-indicator { background: #58a6ff; }
.log-item.agent .log-indicator { background: #d29922; }

.log-text {
  color: #8b949e;
  line-height: 1.5;
  word-break: break-word;
}

.log-item.error .log-text { color: #f85149; }
.log-item.success .log-text { color: #3fb950; }

.log-empty {
  color: #484f58;
  font-style: italic;
  text-align: center;
  padding: 20px;
}

/* Tools */
.tools-section {
  padding: 12px 16px;
}

.tools-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tool-chip {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  color: #8b949e;
  cursor: default;
}

.tool-chip.connected {
  border-color: #238636;
  color: #3fb950;
}

.tool-chip.skill {
  border-color: #9e6a03;
  color: #d29922;
}

/* Workspace */
.workspace {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #0d1117;
}

.workspace-header {
  display: flex;
  align-items: center;
  height: 44px;
  padding: 0 12px;
  background: #161b22;
  border-bottom: 1px solid #21262d;
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-left :deep(.el-button) {
  background: transparent;
  border-color: #30363d;
  color: #8b949e;
}

.header-left :deep(.el-button:hover) {
  background: #21262d;
  color: #c9d1d9;
}

.header-left :deep(.el-button--primary) {
  background: #388bfd26;
  border-color: #388bfd;
  color: #58a6ff;
}

.path-display {
  font-size: 12px;
  color: #8b949e;
  font-family: 'SF Mono', Monaco, monospace;
}

.header-center {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.tab-bar {
  display: flex;
  gap: 2px;
  overflow-x: auto;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: transparent;
  border-radius: 6px 6px 0 0;
  font-size: 12px;
  color: #8b949e;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.tab-item:hover {
  background: #21262d;
}

.tab-item.active {
  background: #0d1117;
  color: #c9d1d9;
}

.tab-close {
  opacity: 0;
  transition: opacity 0.15s;
}

.tab-item:hover .tab-close {
  opacity: 0.6;
}

.tab-close:hover {
  opacity: 1;
  color: #f85149;
}

.header-right {
  display: flex;
  align-items: center;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: #8b949e;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #3fb950;
}

.status-indicator.processing .status-dot {
  background: #58a6ff;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* Editor Area */
.editor-area {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.file-explorer {
  width: 220px;
  background: #010409;
  border-right: 1px solid #21262d;
  overflow: auto;
}

.explorer-header {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #8b949e;
  border-bottom: 1px solid #21262d;
}

.file-tree {
  background: transparent;
}

.file-tree :deep(.el-tree-node__content) {
  height: 28px;
  padding-left: 8px !important;
}

.file-tree :deep(.el-tree-node__content:hover) {
  background: #161b22;
}

.file-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: #388bfd26;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #c9d1d9;
}

.editor-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.monaco-container {
  flex: 1;
  min-height: 0;
}

/* Terminal */
.terminal-area {
  background: #010409;
  border-top: 1px solid #21262d;
  display: flex;
  flex-direction: column;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: #161b22;
  cursor: ns-resize;
  user-select: none;
}

.terminal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #8b949e;
}

.terminal-body {
  flex: 1;
  padding: 8px;
  overflow: hidden;
}

/* Dialogs & Drawers */
.config-dialog :deep(.el-dialog) {
  background: #161b22;
  border: 1px solid #30363d;
}

.config-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid #21262d;
}

.config-dialog :deep(.el-dialog__title) {
  color: #c9d1d9;
}

.enhanced-drawer :deep(.el-drawer) {
  background: #0d1117;
}

.enhanced-drawer :deep(.el-drawer__header) {
  border-bottom: 1px solid #21262d;
  color: #c9d1d9;
}

/* Results */
.results-content {
  padding: 16px;
}

.result-stat {
  display: flex;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #21262d;
}

.stat-label {
  color: #8b949e;
  font-size: 13px;
}

.stat-value {
  color: #c9d1d9;
  font-weight: 500;
}

.result-files {
  margin-top: 20px;
}

.result-files h4 {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #8b949e;
  margin-bottom: 12px;
}

.file-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.file-chip {
  padding: 6px 12px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  font-size: 12px;
  color: #58a6ff;
  cursor: pointer;
  transition: all 0.15s;
}

.file-chip:hover {
  background: #21262d;
  border-color: #58a6ff;
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: #30363d;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #484f58;
}
</style>
