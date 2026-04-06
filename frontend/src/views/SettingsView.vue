<template>
  <div class="settings-container">
    <div class="settings-header">
      <el-button text @click="$router.push('/')">
        <el-icon><ArrowLeft /></el-icon> {{ t('settings.backToChat') }}
      </el-button>
      <h2>{{ t('settings.title') }}</h2>
    </div>

    <div class="settings-layout">
      <!-- Left Sidebar -->
      <div class="settings-sidebar">
        <el-menu
          :default-active="activeCategory"
          class="settings-menu"
          @select="handleMenuSelect"
        >
          <!-- AI Models Section -->
          <el-sub-menu index="models" v-if="settings.models && settings.models.length > 0">
            <template #title>
              <el-icon><Cpu /></el-icon>
              <span>{{ t('settings.aiModels') }}</span>
            </template>
            <el-menu-item
              v-for="item in settings.models"
              :key="item.key"
              :index="'model-' + item.key"
            >
              <span class="provider-name">{{ item.name }}</span>
              <el-tag v-if="item.enabled" type="success" size="small" class="status-tag">{{ t('settings.enabled') }}</el-tag>
              <el-tag v-else type="info" size="small" class="status-tag">{{ t('settings.disabled') }}</el-tag>
              <div class="item-actions">
                <el-button
                  class="action-btn delete"
                  text
                  size="small"
                  @click.stop="deleteItem('models', item.key)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </el-menu-item>
            <div class="add-item-row">
              <el-button text size="small" @click="showAddDialog('models')">
                <el-icon><Plus /></el-icon> {{ t('settings.addModel') }}
              </el-button>
            </div>
          </el-sub-menu>

          <!-- Search Providers Section -->
          <el-sub-menu index="searchs" v-if="settings.searchs && settings.searchs.length > 0">
            <template #title>
              <el-icon><Search /></el-icon>
              <span>{{ t('settings.searchEngines') }}</span>
            </template>
            <el-menu-item
              v-for="item in settings.searchs"
              :key="item.key"
              :index="'search-' + item.key"
            >
              <span class="provider-name">{{ item.name }}</span>
              <el-tag v-if="item.enabled" type="success" size="small" class="status-tag">{{ t('settings.enabled') }}</el-tag>
              <el-tag v-else type="info" size="small" class="status-tag">{{ t('settings.disabled') }}</el-tag>
              <div class="item-actions">
                <el-button
                  class="action-btn delete"
                  text
                  size="small"
                  @click.stop="deleteItem('searchs', item.key)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </el-menu-item>
            <div class="add-item-row">
              <el-button text size="small" @click="showAddDialog('searchs')">
                <el-icon><Plus /></el-icon> {{ t('settings.addSearch') }}
              </el-button>
            </div>
          </el-sub-menu>

          <!-- MCP Servers Section -->
          <el-sub-menu index="mcpservers">
            <template #title>
              <el-icon><Connection /></el-icon>
              <span>{{ t('settings.mcpServers') }}</span>
            </template>
            <el-menu-item
              v-for="item in settings.mcpservers"
              :key="item.key"
              :index="'mcp-' + item.key"
            >
              <span class="provider-name">{{ item.name }}</span>
              <el-tag v-if="item.enabled" type="success" size="small" class="status-tag">{{ t('settings.enabled') }}</el-tag>
              <el-tag v-else type="info" size="small" class="status-tag">{{ t('settings.disabled') }}</el-tag>
              <div class="item-actions">
                <el-button
                  class="action-btn delete"
                  text
                  size="small"
                  @click.stop="deleteItem('mcpservers', item.key)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </el-menu-item>
            <div class="add-item-row">
              <el-button text size="small" @click="showAddDialog('mcpservers')">
                <el-icon><Plus /></el-icon> {{ t('settings.addMcpServer') }}
              </el-button>
            </div>
          </el-sub-menu>

          <!-- Skills Section -->
          <el-sub-menu index="skills">
            <template #title>
              <el-icon><Tools /></el-icon>
              <span>{{ t('settings.skills') }}</span>
            </template>
            <el-menu-item
              v-for="item in settings.skills"
              :key="item.key"
              :index="'skill-' + item.key"
            >
              <span class="provider-name">{{ item.name }}</span>
              <el-tag v-if="item.enabled" type="success" size="small" class="status-tag">{{ t('settings.enabled') }}</el-tag>
              <el-tag v-else type="info" size="small" class="status-tag">{{ t('settings.disabled') }}</el-tag>
              <div class="item-actions">
                <el-button
                  class="action-btn delete"
                  text
                  size="small"
                  @click.stop="deleteItem('skills', item.key)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </el-menu-item>
            <div class="add-item-row">
              <el-button text size="small" @click="showAddDialog('skills')">
                <el-icon><Plus /></el-icon> {{ t('settings.addSkill') }}
              </el-button>
            </div>
          </el-sub-menu>

          <!-- Agents Section -->
          <el-sub-menu index="agents">
            <template #title>
              <el-icon><Avatar /></el-icon>
              <span>{{ t('settings.agents') }}</span>
            </template>
            <el-menu-item
              v-for="item in settings.agents"
              :key="item.key"
              :index="'agent-' + item.key"
            >
              <span class="provider-name">{{ item.name }}</span>
              <el-tag v-if="item.enabled" type="success" size="small" class="status-tag">{{ t('settings.enabled') }}</el-tag>
              <el-tag v-else type="info" size="small" class="status-tag">{{ t('settings.disabled') }}</el-tag>
              <div class="item-actions">
                <el-button
                  class="action-btn delete"
                  text
                  size="small"
                  @click.stop="deleteItem('agents', item.key)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </el-menu-item>
            <div class="add-item-row">
              <el-button text size="small" @click="showAddDialog('agents')">
                <el-icon><Plus /></el-icon> {{ t('settings.addAgent') }}
              </el-button>
            </div>
          </el-sub-menu>

          <!-- Loading state -->
          <div v-if="loading" class="loading-state">
            <el-icon class="is-loading"><Loading /></el-icon>
            <span>{{ t('common.loading') }}</span>
          </div>

          <!-- Empty state -->
          <div v-if="!loading && isEmpty" class="empty-state">
            <el-empty :description="t('settings.noConfig')" />
          </div>
        </el-menu>
      </div>

      <!-- Right Content -->
      <div class="settings-content">
        <!-- Connection Status Dashboard -->
        <div class="status-dashboard" v-if="!selectedItem">
          <h3>{{ t('settings.statusDashboard') }}</h3>
          <div class="status-grid">
            <!-- AI Models Status -->
            <div class="status-card">
              <div class="status-header">
                <el-icon class="status-icon"><Cpu /></el-icon>
                <span>{{ t('settings.aiModels') }}</span>
              </div>
              <div class="status-body">
                <div class="status-count">
                  <span class="count-number">{{ settings.models.filter(m => m.enabled).length }}</span>
                  <span class="count-label">{{ t('common.enabled') }}</span>
                </div>
                <div class="status-items">
                  <div 
                    v-for="item in settings.models.slice(0, 3)" 
                    :key="item.key"
                    class="status-item"
                    :class="{ 'connected': testResult[item.key]?.success, 'error': testResult[item.key]?.success === false }"
                  >
                    <span class="item-dot"></span>
                    <span class="item-name">{{ item.name }}</span>
                    <el-button 
                      size="small" 
                      text 
                      @click="testConnectivity(item.key, 'models')"
                      :loading="testingProvider === item.key"
                    >
                      {{ t('common.test') }}
                    </el-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Search Providers Status -->
            <div class="status-card">
              <div class="status-header">
                <el-icon class="status-icon"><Search /></el-icon>
                <span>{{ t('settings.searchEngines') }}</span>
              </div>
              <div class="status-body">
                <div class="status-count">
                  <span class="count-number">{{ settings.searchs.filter(s => s.enabled).length }}</span>
                  <span class="count-label">{{ t('common.enabled') }}</span>
                </div>
                <div class="status-items">
                  <div 
                    v-for="item in settings.searchs.slice(0, 3)" 
                    :key="item.key"
                    class="status-item"
                    :class="{ 'connected': testResult[item.key]?.success, 'error': testResult[item.key]?.success === false }"
                  >
                    <span class="item-dot"></span>
                    <span class="item-name">{{ item.name }}</span>
                    <el-button 
                      size="small" 
                      text 
                      @click="testConnectivity(item.key, 'searchs')"
                      :loading="testingProvider === item.key"
                    >
                      {{ t('common.test') }}
                    </el-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- MCP Servers Status -->
            <div class="status-card">
              <div class="status-header">
                <el-icon class="status-icon"><Connection /></el-icon>
                <span>{{ t('settings.mcpServers') }}</span>
              </div>
              <div class="status-body">
                <div class="status-count">
                  <span class="count-number">{{ settings.mcpservers.filter(m => m.enabled).length }}</span>
                  <span class="count-label">{{ t('common.enabled') }}</span>
                </div>
                <div class="status-items">
                  <div 
                    v-for="item in settings.mcpservers.slice(0, 3)" 
                    :key="item.key"
                    class="status-item"
                    :class="{ 'connected': testResult[item.key]?.success, 'error': testResult[item.key]?.success === false }"
                  >
                    <span class="item-dot"></span>
                    <span class="item-name">{{ item.name }}</span>
                    <el-button 
                      size="small" 
                      text 
                      @click="testConnectivity(item.key, 'mcpservers')"
                      :loading="testingProvider === item.key"
                    >
                      {{ t('common.test') }}
                    </el-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Agents Status -->
            <div class="status-card">
              <div class="status-header">
                <el-icon class="status-icon"><Avatar /></el-icon>
                <span>{{ t('settings.agents') }} &amp; {{ t('settings.skills') }}</span>
              </div>
              <div class="status-body">
                <div class="status-count">
                  <span class="count-number">{{ settings.agents.filter(a => a.enabled).length + settings.skills.filter(s => s.enabled).length }}</span>
                  <span class="count-label">{{ t('common.enabled') }}</span>
                </div>
                <div class="status-items">
                  <div class="status-item connected">
                    <span class="item-dot"></span>
                    <span class="item-name">{{ settings.agents.length }} {{ t('settings.agents') }}</span>
                  </div>
                  <div class="status-item connected">
                    <span class="item-dot"></span>
                    <span class="item-name">{{ settings.skills.length }} {{ t('settings.skills') }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Quick Actions -->
          <div class="quick-actions">
            <h4>{{ t('settings.quickActions') }}</h4>
            <div class="action-buttons">
              <el-button type="primary" @click="testAllConnections">
                <el-icon><Connection /></el-icon>
                {{ t('settings.testAllConnections') }}
              </el-button>
              <el-button @click="showAddDialog('models')">
                <el-icon><Plus /></el-icon>
                {{ t('settings.addAiModel') }}
              </el-button>
              <el-button @click="showAddDialog('mcpservers')">
                <el-icon><Plus /></el-icon>
                {{ t('settings.addMcpServer') }}
              </el-button>
            </div>
            <div class="language-selector">
              <h4>{{ t('settings.language') }}</h4>
              <el-select
                v-model="languageStore.currentLocale"
                :placeholder="t('settings.selectLanguage')"
                style="width: 200px"
                @change="handleLanguageChange"
              >
                <el-option
                  v-for="option in languageOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </div>
          </div>
        </div>

        <div v-if="selectedItem">
          <div class="content-header">
            <h3>{{ selectedItem.name }}</h3>
            <div class="actions">
              <el-button
                v-if="selectedItemType"
                :loading="selectedItemLoading"
                :type="selectedItemButtonType"
                size="small"
                @click="selectedItem && testConnectivity(selectedItem.key, selectedItemType)"
              >
                <el-icon><Connection /></el-icon>
                {{ selectedItemButtonText }}
              </el-button>
              <el-button v-if="!isEditing" type="primary" size="small" @click="startEdit">
                <el-icon><Edit /></el-icon> {{ t('common.edit') }}
              </el-button>
              <template v-else>
                <el-button size="small" @click="cancelEdit">{{ t('common.cancel') }}</el-button>
                <el-button type="primary" size="small" @click="saveEdit">
                  <el-icon><Check /></el-icon> {{ t('common.save') }}
                </el-button>
              </template>
            </div>
          </div>
          <div class="config-editor">
            <el-input
              v-model="editableConfig"
              type="textarea"
              :rows="20"
              :readonly="!isEditing"
              :placeholder="t('common.configContent')"
              class="config-textarea"
            />
          </div>
        </div>
        <div v-else>
          <el-empty :description="t('common.selectItem')" />
        </div>
      </div>
    </div>

    <!-- Add Provider Dialog -->
    <el-dialog v-model="addDialogVisible" :title="t('settings.addConfig')" width="400px">
      <el-form :model="newItem" label-width="100px">
        <el-form-item :label="t('settings.type')">
          <el-select v-model="newItem.type" :placeholder="t('common.pleaseSelect')">
            <el-option :label="t('settings.aiModels')" value="models" />
            <el-option :label="t('settings.searchEngines')" value="searchs" />
            <el-option :label="t('settings.mcpServers')" value="mcpservers" />
            <el-option :label="t('settings.skills')" value="skills" />
            <el-option :label="t('settings.agents')" value="agents" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('settings.identifier')">
          <el-input v-model="newItem.key" :placeholder="t('settings.identifierPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('settings.name')">
          <el-input v-model="newItem.name" :placeholder="t('settings.namePlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="addItem">{{ t('common.add') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Cpu, Search, Tools, Avatar, Plus, Close, ArrowLeft, Loading, Connection, Edit, Check
} from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import { useLanguageStore } from '../stores/language'

interface SettingItem {
  key: string
  name: string
  enabled: boolean
  type: string
  [key: string]: any
}

interface Settings {
  models: SettingItem[]
  searchs: SettingItem[]
  mcpservers: SettingItem[]
  skills: SettingItem[]
  agents: SettingItem[]
}

const activeCategory = ref('')
const loading = ref(false)
const settings = ref<Settings>({
  models: [],
  searchs: [],
  mcpservers: [],
  skills: [],
  agents: []
})
const selectedItem = ref<SettingItem | null>(null)
const isEditing = ref(false)
const editableConfig = ref('')
const originalConfig = ref('')

const addDialogVisible = ref(false)
const newItem = ref({ type: 'models', key: '', name: '' })

// 计算属性来处理selectedItem的逻辑
const selectedItemKey = computed(() => selectedItem.value?.key || '')
const selectedItemTestResult = computed(() => selectedItemKey.value ? testResult.value[selectedItemKey.value] : null)
const selectedItemLoading = computed(() => testingProvider.value === selectedItemKey.value)
const selectedItemButtonType = computed(() => {
  if (!selectedItemTestResult.value) return 'info'
  return selectedItemTestResult.value.success ? 'success' : selectedItemTestResult.value.success === false ? 'danger' : 'info'
})
const selectedItemButtonText = computed(() => {
  if (!selectedItemTestResult.value) return t('common.testConnectivity')
  return selectedItemTestResult.value.success ? t('common.connected') : selectedItemTestResult.value.success === false ? t('common.disconnected') : t('common.testConnectivity')
})
const selectedItemType = computed(() => {
  if (!selectedItem.value) return ''
  if (settings.value.models.find(m => m.key === selectedItem.value?.key)) return 'models'
  if (settings.value.searchs.find(s => s.key === selectedItem.value?.key)) return 'searchs'
  if (settings.value.mcpservers.find(m => m.key === selectedItem.value?.key)) return 'mcpservers'
  return ''
})

// 连通性测试状态
const testingProvider = ref<string | null>(null)
const testResult = ref<Record<string, { success: boolean; message: string }>>({})

const { t } = useI18n()
const languageStore = useLanguageStore()

const languageOptions = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'en', label: 'English' }
]

const isEmpty = computed(() => {
  return settings.value.models.length === 0 &&
         settings.value.searchs.length === 0 &&
         settings.value.mcpservers.length === 0 &&
         settings.value.skills.length === 0 &&
         settings.value.agents.length === 0
})

type Locale = 'zh-CN' | 'zh-TW' | 'en'

const handleLanguageChange = (value: Locale) => {
  languageStore.setLocale(value)
}

// Load all settings from API
const loadSettings = async () => {
  loading.value = true
  try {
    // 设置请求超时时间为10秒
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 10000)
    
    const response = await fetch('/api/v1/settings', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      signal: controller.signal
    })
    
    clearTimeout(timeoutId)
    
    if (response.ok) {
      const data = await response.json()
      settings.value = {
        models: data.models || [],
        searchs: data.searchs || [],
        mcpservers: data.mcpservers || [],
        skills: data.skills || [],
        agents: data.agents || []
      }
    } else {
      throw new Error(t('common.loadFailed'))
    }
  } catch (error: any) {
    if (error.name === 'AbortError') {
      ElMessage.error(t('common.requestTimeout'))
    } else {
      ElMessage.error(error.message || t('common.loadSettingsFailed'))
    }
  } finally {
    loading.value = false
  }
}

const handleMenuSelect = (key: string) => {
  isEditing.value = false
  const [type, ...rest] = key.split('-')
  const itemKey = rest.join('-')

  let item: SettingItem | undefined
  switch (type) {
    case 'model':
      item = settings.value.models.find(i => i.key === itemKey)
      break
    case 'search':
      item = settings.value.searchs.find(i => i.key === itemKey)
      break
    case 'mcp':
      item = settings.value.mcpservers.find(i => i.key === itemKey)
      break
    case 'skill':
      item = settings.value.skills.find(i => i.key === itemKey)
      break
    case 'agent':
      item = settings.value.agents.find(i => i.key === itemKey)
      break
  }

  if (item) {
    selectedItem.value = item
  }
}

const startEdit = () => {
  isEditing.value = true
  originalConfig.value = editableConfig.value
}

const cancelEdit = () => {
  isEditing.value = false
  editableConfig.value = originalConfig.value
}

const saveEdit = async () => {
  try {
    // 解析编辑的 JSON
    const parsedData = JSON.parse(editableConfig.value)
    const key = selectedItem.value?.key

    // 根据菜单项确定类型
    let type = ''
    if (selectedItem.value) {
      // 根据菜单项的key前缀确定类型
      if (selectedItem.value.key.startsWith('model-') || settings.value.models.find(m => m.key === selectedItem.value?.key)) {
        type = 'models'
      } else if (selectedItem.value.key.startsWith('search-') || settings.value.searchs.find(s => s.key === selectedItem.value?.key)) {
        type = 'searchs'
      } else if (selectedItem.value.key.startsWith('mcp-') || settings.value.mcpservers.find(m => m.key === selectedItem.value?.key)) {
        type = 'mcpservers'
      } else if (selectedItem.value.key.startsWith('skill-') || settings.value.skills.find(s => s.key === selectedItem.value?.key)) {
        type = 'skills'
      } else if (selectedItem.value.key.startsWith('agent-') || settings.value.agents.find(a => a.key === selectedItem.value?.key)) {
        type = 'agents'
      }
    }

    console.log('Saving with type:', type, 'key:', key)
    console.log('Selected item:', selectedItem.value)
    console.log('Request URL:', `/api/v1/settings/${type}/${key}`)

    if (!type || !key) return

    const response = await fetch(`/api/v1/settings/${type}/${key}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify(parsedData)
    })

    if (response.ok) {
      ElMessage.success(t('common.saveSuccess'))
      await loadSettings()
      // 更新选中的项目
      if (selectedItem.value?.key === key) {
        const updated = settings.value[type as keyof Settings].find(i => i.key === key)
        if (updated) {
          selectedItem.value = updated
        }
      }
      // 最后设置isEditing为false
      isEditing.value = false
    } else {
      const errorData = await response.json().catch(() => null)
      console.error('Save failed:', response.status, errorData)
      const errorMsg = errorData?.error || errorData?.details?.join(', ') || `HTTP ${response.status}`
      throw new Error(t('common.saveFailed') + ': ' + errorMsg)
    }
  } catch (error: any) {
    if (error instanceof SyntaxError) {
      ElMessage.error(t('common.jsonFormatError'))
    } else {
      ElMessage.error(error.message || t('common.saveFailed'))
    }
  }
}

// Watch selectedItem changes and update editableConfig
watch(selectedItem, (newItem) => {
  if (newItem) {
    editableConfig.value = JSON.stringify(newItem, null, 2)
    originalConfig.value = editableConfig.value
  } else {
    editableConfig.value = ''
    originalConfig.value = ''
  }
}, { immediate: true })

const showAddDialog = (type: string) => {
  newItem.value = { type, key: '', name: '' }
  addDialogVisible.value = true
}

const addItem = async () => {
  if (!newItem.value.key || !newItem.value.name) {
    ElMessage.warning(t('common.pleaseFill'))
    return
  }

  try {
    let response
    
    // 根据不同类型处理不同的API请求
    if (newItem.value.key.startsWith('mcp-') || newItem.value.key.includes('context7') || newItem.value.key.includes('filesystem')) {
      // MCP服务器特殊处理
      response = await fetch('/api/v1/settings/mcpservers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          key: newItem.value.key,
          name: newItem.value.name,
          enabled: true,
          server_type: 'command', // 默认服务器类型
          command: '',
          args: [],
          env: {},
          allowed_paths: []
        })
      })
    } else if (newItem.value.key.startsWith('model-') || newItem.value.key.includes('openai') || newItem.value.key.includes('deepseek')) {
      // AI模型处理
      response = await fetch('/api/v1/settings/ai-providers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          key: newItem.value.key,
          name: newItem.value.name,
          type: 'ai',
          enabled: true
        })
      })
    } else if (newItem.value.key.startsWith('search-') || newItem.value.key.includes('brave') || newItem.value.key.includes('google')) {
      // 搜索引擎处理
      response = await fetch('/api/v1/settings/ai-providers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          key: newItem.value.key,
          name: newItem.value.name,
          type: 'search',
          enabled: true
        })
      })
    } else {
      // 默认处理
      response = await fetch('/api/v1/settings/ai-providers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          key: newItem.value.key,
          name: newItem.value.name,
          type: 'ai',
          enabled: true
        })
      })
    }

    if (response.ok) {
      ElMessage.success(t('common.addSuccess'))
      addDialogVisible.value = false
      await loadSettings()
    } else {
      throw new Error(t('common.addFailed'))
    }
  } catch (error: any) {
    ElMessage.error(error.message || t('common.addFailed'))
  }
}

const deleteItem = async (type: string, key: string) => {
  try {
    await ElMessageBox.confirm(
      t('common.deleteConfirm', { key }),
      t('common.deleteConfirmTitle'),
      {
        confirmButtonText: t('common.delete'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    const response = await fetch(`/api/v1/settings/${type}/${key}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })

    if (response.ok) {
      ElMessage.success(t('common.deleteSuccess'))
      selectedItem.value = null
      await loadSettings()
    } else {
      throw new Error(t('common.deleteFailed'))
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || t('common.deleteFailed'))
    }
  }
}

onMounted(() => {
  loadSettings()
})

// 测试提供商连通性
const testConnectivity = async (key: string, type: string) => {
  testingProvider.value = key
  try {
    const response = await fetch(`/api/v1/settings/${type}/${key}/test`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const result = await response.json()
    console.log('Test result:', result)
    
    testResult.value[key] = {
      success: result.success,
      message: result.message
    }

    if (result.success) {
      ElMessage.success(result.message)
    } else {
      ElMessage.error(result.message)
    }
  } catch (error: any) {
    console.error('Test connectivity error:', error)
    testResult.value[key] = {
      success: false,
      message: error.message || t('common.saveFailed')
    }
    ElMessage.error(error.message || t('common.saveFailed'))
  } finally {
    testingProvider.value = null
  }
}

// 测试所有连接
const testAllConnections = async () => {
  const allItems = [
    ...settings.value.models.map(m => ({ key: m.key, type: 'models' })),
    ...settings.value.searchs.map(s => ({ key: s.key, type: 'searchs' })),
    ...settings.value.mcpservers.map(m => ({ key: m.key, type: 'mcpservers' }))
  ]
  
  for (const item of allItems) {
    await testConnectivity(item.key, item.type)
  }
  
  ElMessage.success(t('common.allTestsCompleted'))
}
</script>

<style scoped>
.settings-container {
  height: 100vh;
  background: var(--bg-secondary);
  display: flex;
  flex-direction: column;
}

.settings-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 24px;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-primary);
}

.settings-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: var(--text-primary);
}

.settings-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.settings-sidebar {
  width: 260px;
  background: var(--bg-primary);
  border-right: 1px solid var(--border-primary);
  overflow-y: auto;
}

.settings-menu {
  border-right: none;
}

.settings-menu :deep(.el-sub-menu__title) {
  height: 44px;
  line-height: 44px;
  font-weight: 500;
  color: var(--text-secondary);
}

.settings-menu :deep(.el-menu-item) {
  height: 40px;
  line-height: 40px;
  display: flex;
  align-items: center;
  padding-right: 12px;
  color: var(--text-secondary);
}

.provider-name {
  flex: 1;
  font-size: 14px;
}

.status-tag {
  margin-right: 8px;
  font-size: 11px;
  height: 20px;
  line-height: 18px;
}

.item-actions {
  display: flex;
  gap: 4px;
}

.action-btn {
  opacity: 0;
  transition: opacity 0.2s;
  color: var(--text-tertiary);
  padding: 4px;
  height: 24px;
  width: 24px;
}

.action-btn:hover {
  color: var(--accent-primary);
}

.action-btn.delete:hover {
  color: var(--accent-danger);
}

.settings-menu :deep(.el-menu-item:hover) .action-btn {
  opacity: 1;
}

.add-item-row {
  padding: 8px 20px;
}

.add-item-row .el-button {
  color: var(--text-tertiary);
  font-size: 13px;
}

.add-item-row .el-button:hover {
  color: var(--accent-primary);
}

.loading-state, .empty-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-tertiary);
}

.settings-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background: var(--bg-primary);
  margin: 16px;
  border-radius: 8px;
  box-shadow: var(--shadow-sm);
}

.settings-content h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-primary);
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-primary);
  margin-bottom: 16px;
}

.content-header .actions {
  display: flex;
  gap: 8px;
}

.config-editor {
  margin-top: 16px;
}

.config-textarea :deep(textarea) {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-secondary);
  border-radius: 4px;
  padding: 12px;
}

.config-textarea :deep(textarea[readonly]) {
  background: var(--bg-tertiary);
  cursor: default;
}

.config-preview {
  background: var(--bg-tertiary);
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  overflow-x: auto;
}

/* Status Dashboard Styles */
.status-dashboard {
  padding: 0;
}

.status-dashboard h3 {
  margin: 0 0 20px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.status-card {
  background: linear-gradient(135deg, #f8fafc 0%, #ffffff 100%);
  border: 1px solid #e4e7ed;
  border-radius: 12px;
  padding: 20px;
  transition: all 0.3s ease;
}

.status-card:hover {
  border-color: #409eff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}

.status-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.status-icon {
  font-size: 20px;
  color: #409eff;
}

.status-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.status-count {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.count-number {
  font-size: 32px;
  font-weight: 700;
  color: #409eff;
}

.count-label {
  font-size: 13px;
  color: #909399;
}

.status-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 8px;
  font-size: 13px;
  color: #606266;
}

.item-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #909399;
}

.status-item.connected .item-dot {
  background: #67c23a;
}

.status-item.error .item-dot {
  background: #f56c6c;
}

.item-name {
  flex: 1;
}

.quick-actions {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #e4e7ed;
}

.quick-actions h4 {
  margin: 0 0 16px 0;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 24px;
}

.language-selector {
  padding-top: 24px;
  border-top: 1px solid #e4e7ed;
}

.language-selector h4 {
  margin: 0 0 12px 0;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
</style>
