<template>
  <div class="agent-builder">
    <!-- Header -->
    <div class="builder-header">
      <div class="header-left">
        <el-button :icon="ArrowLeft" text @click="$emit('back')">Back</el-button>
        <h2>{{ editingAgent ? 'Edit Agent' : 'Create Agent' }}</h2>
      </div>
      <div class="header-right">
        <el-button @click="handleSaveDraft" :loading="saving">Save Draft</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          {{ editingAgent ? 'Update' : 'Create' }} Agent
        </el-button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="builder-content">
      <!-- Left Panel - Configuration -->
      <div class="config-panel">
        <el-tabs v-model="activeTab">
          <!-- Basic Info Tab -->
          <el-tab-pane label="Basic Info" name="basic">
            <el-form :model="agent" label-position="top">
              <el-form-item label="Agent Name" required>
                <el-input v-model="agent.name" placeholder="Enter agent name" />
              </el-form-item>

              <el-form-item label="Description">
                <el-input 
                  v-model="agent.description" 
                  type="textarea" 
                  :rows="3"
                  placeholder="Describe what this agent does" 
                />
              </el-form-item>

              <el-form-item label="Icon">
                <div class="icon-selector">
                  <el-button 
                    v-for="icon in iconOptions" 
                    :key="icon"
                    :class="{ active: agent.icon === icon }"
                    @click="agent.icon = icon"
                  >
                    {{ icon }}
                  </el-button>
                </div>
              </el-form-item>

              <el-form-item label="Cognitive Architecture">
                <el-select v-model="agent.cognitiveArchitecture" style="width: 100%">
                  <el-option 
                    v-for="arch in architectures" 
                    :key="arch.id" 
                    :label="arch.name" 
                    :value="arch.id"
                  >
                    <div class="arch-option">
                      <span class="arch-name">{{ arch.name }}</span>
                      <span class="arch-desc">{{ arch.description }}</span>
                    </div>
                  </el-option>
                </el-select>
              </el-form-item>

              <el-form-item label="Tags">
                <el-select 
                  v-model="agent.tags" 
                  multiple 
                  filterable 
                  allow-create
                  placeholder="Add tags"
                  style="width: 100%"
                >
                  <el-option v-for="tag in suggestedTags" :key="tag" :label="tag" :value="tag" />
                </el-select>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <!-- Prompt Template Tab -->
          <el-tab-pane label="Prompt Template" name="prompt">
            <el-form :model="agent.promptTemplate" label-position="top">
              <el-form-item label="System Prompt">
                <el-input 
                  v-model="agent.promptTemplate.system" 
                  type="textarea" 
                  :rows="10"
                  placeholder="Define the agent's behavior, personality, and capabilities..."
                />
              </el-form-item>

              <el-form-item label="User Prompt Template">
                <el-input 
                  v-model="agent.promptTemplate.user" 
                  type="textarea" 
                  :rows="4"
                  placeholder="Optional template for user input. Use {{input}} for user's message."
                />
              </el-form-item>

              <el-form-item label="Few-Shot Examples">
                <div class="few-shot-examples">
                  <div 
                    v-for="(example, index) in agent.promptTemplate.fewShotExamples" 
                    :key="index"
                    class="example-item"
                  >
                    <el-input 
                      v-model="example.input" 
                      placeholder="Example input"
                      size="small"
                    />
                    <el-input 
                      v-model="example.output" 
                      type="textarea"
                      :rows="2"
                      placeholder="Expected output"
                      size="small"
                    />
                    <el-button :icon="Delete" circle size="small" @click="removeExample(index)" />
                  </div>
                  <el-button type="dashed" @click="addExample">+ Add Example</el-button>
                </div>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <!-- Persona Tab -->
          <el-tab-pane label="Persona" name="persona">
            <el-form :model="agent.persona" label-position="top">
              <el-form-item label="Expertise Areas">
                <el-select 
                  v-model="agent.persona.expertise" 
                  multiple 
                  filterable 
                  allow-create
                  placeholder="Add expertise areas"
                  style="width: 100%"
                >
                  <el-option v-for="exp in expertiseOptions" :key="exp" :label="exp" :value="exp" />
                </el-select>
              </el-form-item>

              <el-form-item label="Communication Style">
                <el-select v-model="agent.persona.style" style="width: 100%">
                  <el-option label="Professional" value="professional" />
                  <el-option label="Casual" value="casual" />
                  <el-option label="Technical" value="technical" />
                  <el-option label="Friendly" value="friendly" />
                  <el-option label="Analytical" value="analytical" />
                  <el-option label="Creative" value="creative" />
                </el-select>
              </el-form-item>

              <el-form-item label="Tone">
                <el-select v-model="agent.persona.tone" style="width: 100%">
                  <el-option label="Helpful" value="helpful" />
                  <el-option label="Direct" value="direct" />
                  <el-option label="Encouraging" value="encouraging" />
                  <el-option label="Objective" value="objective" />
                  <el-option label="Empathetic" value="empathetic" />
                </el-select>
              </el-form-item>

              <el-form-item label="Constraints">
                <el-input 
                  v-model="agent.persona.constraints" 
                  type="textarea" 
                  :rows="3"
                  placeholder="Define any behavioral constraints..."
                />
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <!-- Model & Tools Tab -->
          <el-tab-pane label="Model & Tools" name="model">
            <el-form :model="agent.modelPreferences" label-position="top">
              <el-form-item label="Preferred Model">
                <el-select v-model="agent.modelPreferences.preferredModel" style="width: 100%">
                  <el-option 
                    v-for="model in availableModels" 
                    :key="model.id" 
                    :label="model.name" 
                    :value="model.id"
                  />
                </el-select>
              </el-form-item>

              <el-form-item label="Fallback Models">
                <el-select 
                  v-model="agent.modelPreferences.fallbackModels" 
                  multiple
                  style="width: 100%"
                >
                  <el-option 
                    v-for="model in availableModels" 
                    :key="model.id" 
                    :label="model.name" 
                    :value="model.id"
                  />
                </el-select>
              </el-form-item>

              <el-divider>Tool Bindings</el-divider>

              <el-form-item label="Allowed Tools">
                <el-checkbox-group v-model="agent.toolBindings">
                  <div class="tool-list">
                    <el-checkbox 
                      v-for="tool in availableTools" 
                      :key="tool.name"
                      :label="tool.name"
                    >
                      <div class="tool-item">
                        <span class="tool-name">{{ tool.name }}</span>
                        <span class="tool-desc">{{ tool.description }}</span>
                      </div>
                    </el-checkbox>
                  </div>
                </el-checkbox-group>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <!-- Self-Improvement Tab -->
          <el-tab-pane label="Self-Improvement" name="improve">
            <el-form :model="agent.selfImproveConfig" label-position="top">
              <el-form-item label="Enable Self-Improvement">
                <el-switch v-model="agent.selfImproveConfig.enabled" />
              </el-form-item>

              <template v-if="agent.selfImproveConfig.enabled">
                <el-form-item label="Feedback Threshold">
                  <el-slider 
                    v-model="agent.selfImproveConfig.feedbackThreshold" 
                    :min="1" 
                    :max="5" 
                    :step="0.5"
                    show-stops
                  />
                  <div class="slider-hint">
                    Trigger improvement when average feedback drops below {{ agent.selfImproveConfig.feedbackThreshold }}
                  </div>
                </el-form-item>

                <el-form-item label="Max Auto-Revisions">
                  <el-input-number 
                    v-model="agent.selfImproveConfig.maxRevisions" 
                    :min="1" 
                    :max="10"
                  />
                </el-form-item>
              </template>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </div>

      <!-- Right Panel - Preview & Test -->
      <div class="preview-panel">
        <div class="preview-header">
          <h3>Test Agent</h3>
          <el-button size="small" @click="clearConversation">Clear</el-button>
        </div>

        <div class="test-conversation">
          <div 
            v-for="(msg, index) in testConversation" 
            :key="index"
            :class="['message', msg.role]"
          >
            <div class="message-content">{{ msg.content }}</div>
          </div>
          <div v-if="testing" class="message assistant loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            Thinking...
          </div>
        </div>

        <div class="test-input">
          <el-input 
            v-model="testInput" 
            placeholder="Test your agent..."
            @keyup.enter="handleTest"
            :disabled="testing"
          >
            <template #append>
              <el-button :icon="Position" @click="handleTest" :loading="testing" />
            </template>
          </el-input>
        </div>

        <!-- Template Selector -->
        <div class="template-section">
          <h4>Start from Template</h4>
          <div class="template-grid">
            <div 
              v-for="template in templates" 
              :key="template.id"
              class="template-card"
              @click="applyTemplate(template)"
            >
              <div class="template-icon">{{ template.id === 'coding' ? '💻' : template.id === 'analyst' ? '📊' : '🤖' }}</div>
              <div class="template-name">{{ template.name }}</div>
              <div class="template-desc">{{ template.description }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Delete, Position, Loading } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  editingAgent?: any
}>()

const emit = defineEmits(['back', 'saved'])
const authStore = useAuthStore()

const activeTab = ref('basic')
const saving = ref(false)
const testing = ref(false)
const testInput = ref('')
const testConversation = ref<Array<{role: string, content: string}>>([])
const templates = ref<any[]>([])
const availableModels = ref<any[]>([])
const availableTools = ref<any[]>([])

const iconOptions = ['🤖', '💡', '🎯', '🔧', '📊', '💻', '🎨', '📝', '🔍', '⚡']
const suggestedTags = ['coding', 'writing', 'analysis', 'creative', 'research', 'automation', 'debugging', 'optimization']
const expertiseOptions = ['programming', 'data analysis', 'writing', 'research', 'design', 'debugging', 'optimization', 'architecture']

const architectures = [
  { id: 'react', name: 'ReAct', description: 'Reasoning and Acting - thinks step by step' },
  { id: 'reflexion', name: 'Reflexion', description: 'Self-reflecting agent that learns' },
  { id: 'tot', name: 'Tree of Thought', description: 'Explores multiple reasoning paths' },
  { id: 'cot', name: 'Chain of Thought', description: 'Step-by-step logical reasoning' },
  { id: 'custom', name: 'Custom', description: 'Define your own architecture' }
]

const agent = reactive({
  name: '',
  description: '',
  icon: '🤖',
  cognitiveArchitecture: 'react',
  status: 'draft',
  tags: [] as string[],
  promptTemplate: {
    system: '',
    user: '',
    fewShotExamples: [] as Array<{input: string, output: string}>
  },
  persona: {
    expertise: [] as string[],
    style: 'professional',
    tone: 'helpful',
    constraints: ''
  },
  modelPreferences: {
    preferredModel: 'deepseek-chat',
    fallbackModels: [] as string[]
  },
  toolBindings: [] as string[],
  selfImproveConfig: {
    enabled: false,
    feedbackThreshold: 3,
    maxRevisions: 3
  }
})

onMounted(async () => {
  await loadTemplates()
  await loadModels()
  await loadTools()
  
  if (props.editingAgent) {
    Object.assign(agent, props.editingAgent)
    if (typeof agent.promptTemplate === 'string') {
      try {
        agent.promptTemplate = JSON.parse(agent.promptTemplate)
      } catch (e) {
        agent.promptTemplate = { system: '', user: '', fewShotExamples: [] }
      }
    }
    if (typeof agent.persona === 'string') {
      try {
        agent.persona = JSON.parse(agent.persona)
      } catch (e) {
        agent.persona = { expertise: [], style: 'professional', tone: 'helpful', constraints: '' }
      }
    }
  }
})

const loadTemplates = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/agents/templates', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      templates.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load templates:', error)
  }
}

const loadModels = async () => {
  try {
    const response = await fetch('/api/v1/chat/models', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      const data = await response.json()
      availableModels.value = data.models || []
    }
  } catch (error) {
    console.error('Failed to load models:', error)
  }
}

const loadTools = async () => {
  try {
    const response = await fetch('/api/v1/chat/mcp-tools', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      availableTools.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load tools:', error)
  }
}

const applyTemplate = (template: any) => {
  agent.cognitiveArchitecture = template.architecture
  if (template.promptTemplate) {
    agent.promptTemplate.system = template.promptTemplate.system || ''
  }
  if (template.defaultPersona) {
    Object.assign(agent.persona, template.defaultPersona)
  }
  ElMessage.success(`Applied ${template.name} template`)
}

const addExample = () => {
  agent.promptTemplate.fewShotExamples.push({ input: '', output: '' })
}

const removeExample = (index: number) => {
  agent.promptTemplate.fewShotExamples.splice(index, 1)
}

const handleSaveDraft = async () => {
  agent.status = 'draft'
  await saveAgent()
}

const handleSave = async () => {
  agent.status = 'active'
  await saveAgent()
}

const saveAgent = async () => {
  if (!agent.name.trim()) {
    ElMessage.error('Agent name is required')
    return
  }

  saving.value = true
  try {
    const url = props.editingAgent 
      ? `/api/v1/agent-system/agents/${props.editingAgent.id}`
      : '/api/v1/agent-system/agents'
    
    const method = props.editingAgent ? 'PUT' : 'POST'
    
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({
        ...agent,
        prompt_template: JSON.stringify(agent.promptTemplate),
        persona: JSON.stringify(agent.persona),
        model_preferences: JSON.stringify(agent.modelPreferences),
        tool_bindings: JSON.stringify(agent.toolBindings),
        self_improve_config: JSON.stringify(agent.selfImproveConfig),
        tags: JSON.stringify(agent.tags)
      })
    })

    if (response.ok) {
      ElMessage.success(props.editingAgent ? 'Agent updated' : 'Agent created')
      emit('saved')
    } else {
      const error = await response.json()
      ElMessage.error(error.error || 'Failed to save agent')
    }
  } catch (error) {
    ElMessage.error('Failed to save agent')
  } finally {
    saving.value = false
  }
}

const handleTest = async () => {
  if (!testInput.value.trim() || !agent.name) {
    ElMessage.warning('Please enter a test message and ensure agent has a name')
    return
  }

  testConversation.value.push({ role: 'user', content: testInput.value })
  const input = testInput.value
  testInput.value = ''
  testing.value = true

  try {
    // If agent is saved, use the execute endpoint
    if (props.editingAgent?.id) {
      const response = await fetch(`/api/v1/agent-system/agents/${props.editingAgent.id}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authStore.token}`
        },
        body: JSON.stringify({ input })
      })

      if (response.ok) {
        const result = await response.json()
        testConversation.value.push({ role: 'assistant', content: result.output })
      } else {
        testConversation.value.push({ role: 'assistant', content: 'Error executing agent' })
      }
    } else {
      // Preview mode - just show the prompt that would be sent
      testConversation.value.push({ 
        role: 'assistant', 
        content: `[Preview] Agent would process: "${input}" with system prompt starting with: "${agent.promptTemplate.system.substring(0, 100)}..."`
      })
    }
  } catch (error) {
    testConversation.value.push({ role: 'assistant', content: 'Error testing agent' })
  } finally {
    testing.value = false
  }
}

const clearConversation = () => {
  testConversation.value = []
}
</script>

<style scoped>
.agent-builder {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
}

.builder-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid var(--el-border-color);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-left h2 {
  margin: 0;
}

.header-right {
  display: flex;
  gap: 8px;
}

.builder-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.config-panel {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  border-right: 1px solid var(--el-border-color);
}

.preview-panel {
  width: 400px;
  display: flex;
  flex-direction: column;
  padding: 16px;
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.preview-header h3 {
  margin: 0;
}

.test-conversation {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  margin-bottom: 16px;
}

.message {
  padding: 8px 12px;
  margin-bottom: 8px;
  border-radius: 8px;
  max-width: 90%;
}

.message.user {
  background: var(--el-color-primary-light-9);
  margin-left: auto;
}

.message.assistant {
  background: var(--el-bg-color);
}

.message.loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
}

.test-input {
  margin-bottom: 16px;
}

.icon-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.icon-selector .el-button.active {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary);
}

.arch-option {
  display: flex;
  flex-direction: column;
}

.arch-name {
  font-weight: 500;
}

.arch-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.few-shot-examples {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.example-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  position: relative;
}

.example-item .el-button {
  position: absolute;
  top: 8px;
  right: 8px;
}

.tool-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tool-item {
  display: flex;
  flex-direction: column;
}

.tool-name {
  font-weight: 500;
}

.tool-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.slider-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.template-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color);
}

.template-section h4 {
  margin: 0 0 12px 0;
  color: var(--el-text-color-secondary);
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.template-card {
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.template-icon {
  font-size: 24px;
  margin-bottom: 8px;
}

.template-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.template-desc {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}
</style>
