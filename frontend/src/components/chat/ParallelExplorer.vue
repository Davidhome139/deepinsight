<template>
  <div class="parallel-explorer">
    <!-- Trigger Button -->
    <el-popover 
      placement="top" 
      :width="400" 
      trigger="click"
      v-model:visible="showPopover"
    >
      <template #reference>
        <el-button size="small">
          <el-icon><Connection /></el-icon>
          <span>Parallel</span>
        </el-button>
      </template>

      <div class="explore-panel">
        <h4>
          <el-icon><Connection /></el-icon>
          Multi-Model Parallel Exploration
        </h4>
        <p class="explore-desc">
          Send the same prompt to multiple models simultaneously and compare results.
        </p>

        <el-form :model="exploreForm" label-position="top">
          <el-form-item label="Select Models (2-5)">
            <el-checkbox-group v-model="exploreForm.selectedModels">
              <el-checkbox 
                v-for="model in availableModels" 
                :key="model.id" 
                :label="model.id"
              >
                {{ model.name }}
              </el-checkbox>
            </el-checkbox-group>
          </el-form-item>

          <el-form-item label="Prompt">
            <el-input
              v-model="exploreForm.prompt"
              type="textarea"
              :rows="3"
              placeholder="Enter the prompt to send to all selected models..."
            />
          </el-form-item>
        </el-form>

        <div class="explore-actions">
          <el-button @click="showPopover = false">Cancel</el-button>
          <el-button 
            type="primary" 
            :disabled="!canExplore"
            :loading="isExploring"
            @click="startExploration"
          >
            <el-icon><CaretRight /></el-icon>
            Start Exploration
          </el-button>
        </div>
      </div>
    </el-popover>

    <!-- Exploration Results Dialog -->
    <el-dialog 
      v-model="showResults" 
      title="Parallel Exploration Results" 
      width="900px"
      :close-on-click-modal="false"
    >
      <div v-if="currentExploration" class="exploration-results">
        <div class="results-header">
          <div class="results-status">
            <el-tag :type="statusType">{{ currentExploration.status }}</el-tag>
            <span class="prompt-preview">{{ currentExploration.prompt.substring(0, 50) }}...</span>
          </div>
          <div v-if="bestResult" class="best-badge">
            <el-icon><Trophy /></el-icon>
            Best: {{ bestResult.model }}
          </div>
        </div>

        <div class="results-grid">
          <div 
            v-for="result in parsedResults" 
            :key="result.branch_id"
            class="result-card"
            :class="{ best: result.branch_id === currentExploration.best_branch_id }"
          >
            <div class="card-header">
              <span class="model-name">{{ result.model }}</span>
              <div class="card-meta">
                <span v-if="result.score" class="score">
                  Score: {{ result.score.toFixed(2) }}
                </span>
                <span class="latency">{{ result.latency_ms }}ms</span>
              </div>
            </div>

            <div v-if="result.error" class="card-error">
              <el-icon><WarningFilled /></el-icon>
              {{ result.error }}
            </div>

            <div v-else class="card-content">
              <div class="response-preview" v-html="renderMarkdown(result.response_preview)"></div>
              <el-button 
                size="small" 
                type="text" 
                @click="showFullResponse(result)"
              >
                View Full Response
              </el-button>
            </div>

            <div class="card-actions">
              <el-button 
                size="small" 
                type="primary"
                @click="selectBranch(result.branch_id)"
              >
                <el-icon><Check /></el-icon>
                Use This
              </el-button>
              <el-button 
                size="small"
                @click="viewBranch(result.branch_id)"
              >
                <el-icon><View /></el-icon>
                View Branch
              </el-button>
            </div>
          </div>
        </div>

        <div v-if="isExploring" class="exploring-status">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>Exploring {{ exploreForm.selectedModels.length }} models in parallel...</span>
        </div>
      </div>
    </el-dialog>

    <!-- Full Response Dialog -->
    <el-dialog v-model="showFullResponseDialog" title="Full Response" width="600px">
      <div v-if="selectedResult" class="full-response">
        <div class="response-header">
          <span class="model-name">{{ selectedResult.model }}</span>
          <span class="token-count">~{{ selectedResult.token_count }} tokens</span>
        </div>
        <div class="response-content" v-html="renderMarkdown(selectedResult.full_response || selectedResult.response_preview)"></div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  Connection, CaretRight, Trophy, WarningFilled, 
  Check, View, Loading 
} from '@element-plus/icons-vue'
import { marked } from 'marked'

interface Model {
  id: string
  name: string
  provider: string
}

interface ParallelResult {
  model: string
  branch_id: string
  response_preview: string
  full_response?: string
  token_count: number
  latency_ms: number
  score?: number
  error?: string
}

interface Exploration {
  id: string
  conversation_id: number
  source_message_id: number
  prompt: string
  models: string
  status: string
  results: string
  best_branch_id?: string
  created_at: string
  completed_at?: string
}

const props = defineProps<{
  conversationId: number
  lastUserMessageId?: number
}>()

const emit = defineEmits(['branch-selected', 'refresh-messages'])

const showPopover = ref(false)
const showResults = ref(false)
const showFullResponseDialog = ref(false)
const isExploring = ref(false)
const availableModels = ref<Model[]>([])
const currentExploration = ref<Exploration | null>(null)
const selectedResult = ref<ParallelResult | null>(null)

const exploreForm = ref({
  selectedModels: [] as string[],
  prompt: ''
})

const canExplore = computed(() => {
  return exploreForm.value.selectedModels.length >= 2 && 
         exploreForm.value.selectedModels.length <= 5 &&
         exploreForm.value.prompt.trim().length > 0
})

const parsedResults = computed(() => {
  if (!currentExploration.value?.results) return []
  try {
    return JSON.parse(currentExploration.value.results) as ParallelResult[]
  } catch {
    return []
  }
})

const bestResult = computed(() => {
  return parsedResults.value.find(r => r.branch_id === currentExploration.value?.best_branch_id)
})

const statusType = computed(() => {
  switch (currentExploration.value?.status) {
    case 'completed': return 'success'
    case 'running': return 'warning'
    case 'failed': return 'danger'
    default: return 'info'
  }
})

const fetchModels = async () => {
  try {
    const response = await fetch('/api/v1/chat/models', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    if (response.ok) {
      availableModels.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to fetch models:', error)
  }
}

const startExploration = async () => {
  if (!canExplore.value) return
  
  isExploring.value = true
  showPopover.value = false
  
  try {
    const response = await fetch(
      `/api/v1/chat/conversations/${props.conversationId}/parallel-explore`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          source_message_id: props.lastUserMessageId || 0,
          prompt: exploreForm.value.prompt,
          models: exploreForm.value.selectedModels
        })
      }
    )
    
    if (response.ok) {
      currentExploration.value = await response.json()
      showResults.value = true
      
      // Poll for completion
      pollExploration(currentExploration.value!.id)
    } else {
      throw new Error('Failed to start exploration')
    }
  } catch (error) {
    ElMessage.error('Failed to start parallel exploration')
    isExploring.value = false
  }
}

const pollExploration = async (explorationId: string) => {
  const poll = async () => {
    try {
      const response = await fetch(
        `/api/v1/chat/parallel-explorations/${explorationId}`,
        {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`
          }
        }
      )
      
      if (response.ok) {
        currentExploration.value = await response.json()
        
        if (currentExploration.value?.status === 'completed' || 
            currentExploration.value?.status === 'failed') {
          isExploring.value = false
          return
        }
        
        // Continue polling
        setTimeout(poll, 1000)
      }
    } catch (error) {
      console.error('Poll error:', error)
      isExploring.value = false
    }
  }
  
  poll()
}

const selectBranch = async (branchId: string) => {
  try {
    await fetch(
      `/api/v1/chat/conversations/${props.conversationId}/parallel-explorations/${currentExploration.value?.id}/select`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({ branch_id: branchId })
      }
    )
    
    showResults.value = false
    emit('branch-selected', branchId)
    emit('refresh-messages')
    ElMessage.success('Branch selected as active')
  } catch (error) {
    ElMessage.error('Failed to select branch')
  }
}

const viewBranch = (branchId: string) => {
  emit('branch-selected', branchId)
  showResults.value = false
}

const showFullResponse = (result: ParallelResult) => {
  selectedResult.value = result
  showFullResponseDialog.value = true
}

const renderMarkdown = (content: string) => {
  return marked.parse(content, { async: false }) as string
}

onMounted(fetchModels)

// Auto-fill prompt from last message
watch(() => props.lastUserMessageId, async (msgId) => {
  if (msgId) {
    // Could fetch the message content here
  }
})
</script>

<style scoped>
.parallel-explorer {
  display: inline-block;
}

.explore-panel {
  padding: 8px;
}

.explore-panel h4 {
  margin: 0 0 8px 0;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 15px;
}

.explore-desc {
  font-size: 12px;
  color: #8c8c8c;
  margin-bottom: 12px;
}

.explore-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.exploration-results {
  padding: 8px;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e0e0e0;
}

.results-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.prompt-preview {
  font-size: 13px;
  color: #606266;
}

.best-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  background: #f6ffed;
  border: 1px solid #b7eb8f;
  border-radius: 4px;
  color: #52c41a;
  font-weight: 500;
}

.results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.result-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 12px;
  background: #fafafa;
  transition: all 0.2s;
}

.result-card:hover {
  border-color: #1890ff;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
}

.result-card.best {
  border-color: #52c41a;
  background: #f6ffed;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.model-name {
  font-weight: 600;
  font-size: 14px;
}

.card-meta {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: #8c8c8c;
}

.score {
  color: #1890ff;
}

.card-error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #ff4d4f;
  font-size: 13px;
  padding: 8px;
  background: #fff2f0;
  border-radius: 4px;
}

.card-content {
  margin-bottom: 12px;
}

.response-preview {
  font-size: 13px;
  line-height: 1.5;
  max-height: 120px;
  overflow-y: auto;
  margin-bottom: 8px;
}

.card-actions {
  display: flex;
  gap: 8px;
}

.exploring-status {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  margin-top: 16px;
  background: #e6f7ff;
  border-radius: 8px;
  color: #1890ff;
}

.full-response {
  padding: 8px;
}

.response-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e0e0e0;
}

.token-count {
  font-size: 12px;
  color: #8c8c8c;
}

.response-content {
  font-size: 14px;
  line-height: 1.6;
  max-height: 400px;
  overflow-y: auto;
}
</style>
