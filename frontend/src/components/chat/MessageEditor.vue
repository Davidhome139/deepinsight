<template>
  <div class="message-editor">
    <!-- Edit Mode -->
    <div v-if="isEditing" class="edit-container">
      <el-input
        v-model="editedContent"
        type="textarea"
        :autosize="{ minRows: 2, maxRows: 10 }"
        placeholder="Edit message..."
      />
      <div class="edit-actions">
        <el-checkbox v-model="createBranch" label="Create new branch" />
        <div class="edit-buttons">
          <el-button size="small" @click="cancelEdit">Cancel</el-button>
          <el-button size="small" type="primary" @click="saveEdit">
            {{ createBranch ? 'Save & Branch' : 'Save' }}
          </el-button>
        </div>
      </div>
    </div>

    <!-- View Mode - Message Actions -->
    <div v-else class="message-actions-bar">
      <el-tooltip content="Edit message" placement="top">
        <el-button size="small" circle @click="startEdit">
          <el-icon><Edit /></el-icon>
        </el-button>
      </el-tooltip>
      
      <el-tooltip v-if="message.role === 'user'" content="Regenerate response" placement="top">
        <el-button size="small" circle @click="regenerateResponse" :disabled="isStreaming">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </el-tooltip>
      
      <el-tooltip v-if="message.role === 'user'" content="Multi-regenerate (3 alternatives)" placement="top">
        <el-button size="small" circle @click="multiRegenerate" :disabled="isStreaming">
          <el-icon><CopyDocument /></el-icon>
        </el-button>
      </el-tooltip>
      
      <el-tooltip content="Create branch from here" placement="top">
        <el-button size="small" circle @click="forkFromHere">
          <el-icon><Share /></el-icon>
        </el-button>
      </el-tooltip>
      
      <el-tooltip v-if="message.is_edited" content="View edit history" placement="top">
        <el-button size="small" circle @click="showVersionHistory = true">
          <el-icon><Clock /></el-icon>
        </el-button>
      </el-tooltip>
      
      <span v-if="message.is_edited" class="edited-badge">
        Edited (v{{ message.version }})
      </span>
    </div>

    <!-- Version History Dialog -->
    <el-dialog v-model="showVersionHistory" title="Edit History" width="500px">
      <div class="version-list">
        <div v-if="versions.length === 0" class="no-versions">
          No previous versions
        </div>
        <div 
          v-for="version in versions" 
          :key="version.id" 
          class="version-item"
        >
          <div class="version-header">
            <span class="version-number">Version {{ version.version }}</span>
            <span class="version-date">{{ formatDate(version.created_at) }}</span>
          </div>
          <div class="version-content">{{ version.content }}</div>
          <el-button size="small" @click="revertToVersion(version.version)">
            Revert to this version
          </el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Multi-Regenerate Results Dialog -->
    <el-dialog v-model="showMultiResults" title="Alternative Responses" width="700px">
      <div class="multi-results">
        <div 
          v-for="(result, idx) in multiResults" 
          :key="result.id" 
          class="result-item"
        >
          <div class="result-header">
            <span class="result-number">Option {{ idx + 1 }}</span>
            <el-button size="small" type="primary" @click="selectResult(result)">
              Use This
            </el-button>
          </div>
          <div class="result-content" v-html="renderMarkdown(result.content)"></div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Edit, Refresh, CopyDocument, Share, Clock } from '@element-plus/icons-vue'
import { marked } from 'marked'

interface Message {
  id: number
  role: string
  content: string
  is_edited?: boolean
  version?: number
}

interface Version {
  id: number
  message_id: number
  version: number
  content: string
  created_at: string
}

const props = defineProps<{
  message: Message
  conversationId: number
  model: string
  isStreaming?: boolean
}>()

const emit = defineEmits(['message-updated', 'branch-created', 'refresh-messages', 'regenerate-stream'])

const isEditing = ref(false)
const editedContent = ref('')
const createBranch = ref(false)
const showVersionHistory = ref(false)
const showMultiResults = ref(false)
const versions = ref<Version[]>([])
const multiResults = ref<Message[]>([])

const startEdit = () => {
  editedContent.value = props.message.content
  isEditing.value = true
}

const cancelEdit = () => {
  isEditing.value = false
  editedContent.value = ''
  createBranch.value = false
}

const saveEdit = async () => {
  if (!editedContent.value.trim()) {
    ElMessage.warning('Message cannot be empty')
    return
  }

  try {
    if (createBranch.value) {
      // Create branch with edit
      const response = await fetch(`/api/v1/chat/messages/${props.message.id}/branch`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          content: editedContent.value,
          branch_name: `Edit: ${editedContent.value.substring(0, 20)}...`
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        emit('branch-created', data.branch)
        ElMessage.success('Branch created with edited message')
      }
    } else {
      // Simple edit
      const response = await fetch(`/api/v1/chat/messages/${props.message.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          content: editedContent.value,
          regenerate: false
        })
      })
      
      if (response.ok) {
        emit('message-updated', await response.json())
        ElMessage.success('Message updated')
      }
    }
    
    cancelEdit()
    emit('refresh-messages')
  } catch (error) {
    ElMessage.error('Failed to save edit')
  }
}

const regenerateResponse = async () => {
  // Emit event to parent for streaming regeneration
  emit('regenerate-stream', { messageId: props.message.id, model: props.model })
}

const multiRegenerate = async () => {
  try {
    ElMessage.info('Generating 3 alternative responses...')
    
    const response = await fetch(`/api/v1/chat/messages/${props.message.id}/multi-regenerate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ model: props.model, count: 3 })
    })
    
    if (response.ok) {
      multiResults.value = await response.json()
      showMultiResults.value = true
    }
  } catch (error) {
    ElMessage.error('Failed to generate alternatives')
  }
}

const selectResult = (result: Message) => {
  showMultiResults.value = false
  emit('refresh-messages')
  ElMessage.success('Response selected')
}

const forkFromHere = async () => {
  emit('branch-created', { fork_point_message_id: props.message.id })
}

const fetchVersions = async () => {
  if (!props.message.is_edited) return
  
  try {
    const response = await fetch(`/api/v1/chat/messages/${props.message.id}/versions`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    if (response.ok) {
      versions.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to fetch versions:', error)
  }
}

const revertToVersion = async (version: number) => {
  try {
    const response = await fetch(`/api/v1/chat/messages/${props.message.id}/revert`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ version })
    })
    
    if (response.ok) {
      showVersionHistory.value = false
      emit('refresh-messages')
      ElMessage.success('Reverted to version ' + version)
    }
  } catch (error) {
    ElMessage.error('Failed to revert')
  }
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const renderMarkdown = (content: string) => {
  return marked.parse(content, { async: false }) as string
}

watch(showVersionHistory, (show) => {
  if (show) fetchVersions()
})
</script>

<style scoped>
.message-editor {
  margin-top: 8px;
}

.edit-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.edit-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.edit-buttons {
  display: flex;
  gap: 8px;
}

.message-actions-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.message-wrapper:hover .message-actions-bar {
  opacity: 1;
}

.edited-badge {
  font-size: 11px;
  color: #8c8c8c;
  margin-left: 8px;
  padding: 2px 6px;
  background: #f0f0f0;
  border-radius: 4px;
}

.version-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 400px;
  overflow-y: auto;
}

.version-item {
  padding: 12px;
  background: #f9f9f9;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.version-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.version-number {
  font-weight: 600;
  color: #1890ff;
}

.version-date {
  font-size: 12px;
  color: #8c8c8c;
}

.version-content {
  font-size: 13px;
  line-height: 1.5;
  margin-bottom: 8px;
  max-height: 100px;
  overflow-y: auto;
}

.no-versions {
  text-align: center;
  color: #8c8c8c;
  padding: 20px;
}

.multi-results {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 500px;
  overflow-y: auto;
}

.result-item {
  padding: 12px;
  background: #f9f9f9;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.result-number {
  font-weight: 600;
  color: #52c41a;
}

.result-content {
  font-size: 13px;
  line-height: 1.6;
  max-height: 150px;
  overflow-y: auto;
}
</style>
