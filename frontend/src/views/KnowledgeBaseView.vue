<template>
  <div class="knowledge-base-view">
    <div class="header">
      <div class="header-left">
        <button class="back-btn" @click="goBack" title="Back to Chat">
          ← Back
        </button>
        <h1>Knowledge Base</h1>
      </div>
      <button class="upload-btn" @click="showUpload = true">
        <span>📤</span> Upload Document
      </button>
    </div>

    <!-- Upload Modal -->
    <div v-if="showUpload" class="modal-overlay" @click.self="showUpload = false">
      <div class="modal">
        <h2>Upload Document</h2>
        <div 
          class="drop-zone"
          :class="{ dragging: isDragging }"
          @dragover.prevent="isDragging = true"
          @dragleave="isDragging = false"
          @drop.prevent="handleDrop"
          @click="triggerFileInput"
        >
          <input 
            ref="fileInputRef"
            type="file" 
            accept=".txt,.md,.pdf,.docx"
            @change="handleFileSelect"
            hidden
          />
          <div class="drop-content">
            <span class="drop-icon">📄</span>
            <p>Drop file here or click to select</p>
            <p class="file-types">Supported: TXT, MD, PDF, DOCX</p>
          </div>
        </div>
        <div v-if="selectedFile" class="selected-file">
          <span>{{ selectedFile.name }}</span>
          <span class="file-size">({{ formatFileSize(selectedFile.size) }})</span>
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showUpload = false">Cancel</button>
          <button 
            class="upload-confirm-btn" 
            @click="uploadFile"
            :disabled="!selectedFile || uploading"
          >
            {{ uploading ? 'Uploading...' : 'Upload' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Documents List -->
    <div class="documents-section">
      <h2>Documents</h2>
      <div v-if="loading" class="loading">Loading documents...</div>
      <div v-else-if="documents.length === 0" class="empty">
        <p>No documents uploaded yet</p>
        <p class="hint">Upload documents to build your knowledge base</p>
      </div>
      <div v-else class="documents-grid">
        <div v-for="doc in documents" :key="doc.id" class="document-card">
          <div class="doc-icon">{{ getFileIcon(doc.file_type) }}</div>
          <div class="doc-info">
            <h3>{{ doc.filename }}</h3>
            <div class="doc-meta">
              <span :class="'status-' + doc.status">{{ doc.status }}</span>
              <span>{{ doc.chunk_count }} chunks</span>
              <span>{{ formatFileSize(doc.file_size) }}</span>
            </div>
            <p class="doc-date">{{ formatDate(doc.created_at) }}</p>
          </div>
          <div class="doc-actions">
            <button @click="viewChunks(doc)" title="View Chunks">📋</button>
            <button @click="deleteDocument(doc.id)" title="Delete" class="delete-btn">🗑️</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Search Test Panel -->
    <div class="search-section">
      <h2>Search Knowledge Base</h2>
      <p class="search-hint">Test your knowledge base by entering a query. Results show the most relevant document chunks with similarity scores.</p>
      <div class="search-input-wrapper">
        <input 
          v-model="searchQuery"
          type="text"
          placeholder="e.g. How to configure the system?"
          @keyup.enter="performSearch"
        />
        <button @click="performSearch" :disabled="!searchQuery || searching || !hasReadyDocuments">
          {{ searching ? '🔄' : '🔍' }} Search
        </button>
      </div>
      <p v-if="!hasReadyDocuments" class="search-warning">⚠️ Upload and wait for documents to be ready before searching.</p>

      <div v-if="searchResults.length > 0" class="search-results">
        <h3>Results ({{ searchResults.length }})</h3>
        <div v-for="result in searchResults" :key="result.chunk_id" class="result-item">
          <div class="result-header">
            <span class="result-file">{{ result.filename }}</span>
            <span class="result-score">{{ (result.score * 100).toFixed(1) }}% match</span>
          </div>
          <p class="result-content">{{ result.content }}</p>
        </div>
      </div>
    </div>

    <!-- Chunks Modal -->
    <div v-if="showChunks" class="modal-overlay" @click.self="showChunks = false">
      <div class="modal chunks-modal">
        <h2>Document Chunks</h2>
        <div class="chunks-list">
          <div v-for="chunk in chunks" :key="chunk.id" class="chunk-item">
            <div class="chunk-header">
              <span>Chunk #{{ chunk.chunk_index + 1 }}</span>
              <span class="token-count">{{ chunk.token_count }} tokens</span>
            </div>
            <p class="chunk-content">{{ chunk.content }}</p>
          </div>
        </div>
        <button class="close-btn" @click="showChunks = false">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import request from '../utils/request'

const router = useRouter()

const loading = ref(true)
const documents = ref<any[]>([])
const showUpload = ref(false)
const isDragging = ref(false)
const selectedFile = ref<File | null>(null)
const uploading = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

const searchQuery = ref('')
const searching = ref(false)
const searchResults = ref<any[]>([])

const showChunks = ref(false)
const chunks = ref<any[]>([])

let pollingTimer: ReturnType<typeof setInterval> | null = null

const triggerFileInput = () => {
  fileInputRef.value?.click()
}

// Navigate back to Chat
const goBack = () => {
  router.push('/')
}

// Check if any documents are ready for search
const hasReadyDocuments = computed(() => {
  return documents.value.some(doc => doc.status === 'ready')
})

// Check if any documents are still processing
const hasProcessingDocuments = () => {
  return documents.value.some(doc => doc.status === 'processing')
}

// Start polling for status updates
const startPolling = () => {
  if (pollingTimer) return // Already polling
  
  pollingTimer = setInterval(async () => {
    if (hasProcessingDocuments()) {
      await loadDocuments(false) // Silent refresh
    } else {
      stopPolling()
    }
  }, 2000) // Poll every 2 seconds
}

// Stop polling
const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

const loadDocuments = async (showLoading = true) => {
  if (showLoading) loading.value = true
  try {
    const res = await request.get('/rag/documents')
    documents.value = (res as any)?.documents || []
    
    // Start polling if any documents are processing
    if (hasProcessingDocuments()) {
      startPolling()
    }
  } catch (error) {
    console.error('Failed to load documents:', error)
  } finally {
    loading.value = false
  }
}

const handleFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files[0]) {
    selectedFile.value = input.files[0]
  }
}

const handleDrop = (event: DragEvent) => {
  isDragging.value = false
  if (event.dataTransfer?.files && event.dataTransfer.files[0]) {
    selectedFile.value = event.dataTransfer.files[0]
  }
}

const uploadFile = async () => {
  if (!selectedFile.value) return
  
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    
    await request.post('/rag/documents', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    
    showUpload.value = false
    selectedFile.value = null
    await loadDocuments()
    // Start polling for status updates after upload
    startPolling()
  } catch (error) {
    console.error('Failed to upload:', error)
    alert('Upload failed. Please try again.')
  } finally {
    uploading.value = false
  }
}

const deleteDocument = async (id: string) => {
  if (!confirm('Delete this document? This cannot be undone.')) return
  
  try {
    await request.delete(`/rag/documents/${id}`)
    await loadDocuments()
  } catch (error) {
    console.error('Failed to delete:', error)
  }
}

const viewChunks = async (doc: any) => {
  try {
    const res = await request.get(`/rag/documents/${doc.id}/chunks`)
    chunks.value = (res as any)?.chunks || []
    showChunks.value = true
  } catch (error) {
    console.error('Failed to load chunks:', error)
  }
}

const performSearch = async () => {
  if (!searchQuery.value) return
  
  searching.value = true
  try {
    const res = await request.post('/rag/query', {
      query: searchQuery.value,
      top_k: 5,
      threshold: 0.5
    })
    searchResults.value = (res as any)?.results || []
  } catch (error) {
    console.error('Search failed:', error)
  } finally {
    searching.value = false
  }
}

const formatFileSize = (bytes: number) => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric'
  })
}

const getFileIcon = (type: string) => {
  const icons: Record<string, string> = {
    'txt': '📄',
    'md': '📝',
    'pdf': '📕',
    'docx': '📘'
  }
  return icons[type] || '📄'
}

onMounted(loadDocuments)

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.knowledge-base-view {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  background: var(--bg-primary);
  min-height: 100vh;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-btn {
  padding: 8px 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-primary);
  transition: all 0.2s;
}

.back-btn:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-primary);
}

.header h1 {
  margin: 0;
  font-size: 24px;
  color: var(--text-primary);
}

.upload-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
}

.upload-btn:hover {
  background: #1a8ceb;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: var(--card-bg);
  border-radius: 16px;
  padding: 24px;
  width: 90%;
  max-width: 500px;
  border: 1px solid var(--border-primary);
}

.modal h2 {
  margin: 0 0 20px 0;
  font-size: 18px;
  color: var(--text-primary);
}

.drop-zone {
  border: 2px dashed var(--border-secondary);
  border-radius: 12px;
  padding: 48px 24px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--bg-secondary);
}

.drop-zone:hover, .drop-zone.dragging {
  border-color: var(--accent-primary);
  background: var(--bg-tertiary);
}

.drop-icon {
  font-size: 48px;
  display: block;
  margin-bottom: 12px;
}

.file-types {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 8px;
}

.drop-content p {
  color: var(--text-secondary);
}

.selected-file {
  margin-top: 16px;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: 8px;
  color: var(--text-primary);
}

.file-size {
  color: var(--text-secondary);
  font-size: 13px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
}

.cancel-btn, .upload-confirm-btn, .close-btn {
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
}

.cancel-btn {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-primary);
  color: var(--text-primary);
}

.upload-confirm-btn {
  background: var(--accent-primary);
  color: white;
  border: none;
}

.upload-confirm-btn:disabled {
  background: var(--text-placeholder);
  cursor: not-allowed;
}

.documents-section, .search-section {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-primary);
}

.documents-section h2, .search-section h2 {
  margin: 0 0 8px 0;
  font-size: 16px;
  color: var(--text-primary);
}

.search-hint {
  margin: 0 0 16px 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.search-warning {
  margin: 12px 0 0 0;
  font-size: 13px;
  color: #f59e0b;
}

.loading, .empty {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
}

.hint {
  font-size: 13px;
  color: var(--text-tertiary);
}

.documents-grid {
  display: grid;
  gap: 12px;
}

.document-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 10px;
  border: 1px solid var(--border-primary);
}

.doc-icon {
  font-size: 32px;
}

.doc-info {
  flex: 1;
}

.doc-info h3 {
  margin: 0 0 6px 0;
  font-size: 15px;
  color: var(--text-primary);
}

.doc-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}

.status-processing { color: #f59e0b; }
.status-ready { color: #10b981; }
.status-failed { color: #ef4444; }

.doc-date {
  margin: 4px 0 0 0;
  font-size: 11px;
  color: var(--text-tertiary);
}

.doc-actions {
  display: flex;
  gap: 8px;
}

.doc-actions button {
  padding: 8px;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 16px;
  opacity: 0.7;
}

.doc-actions button:hover {
  opacity: 1;
}

.delete-btn:hover {
  color: var(--accent-danger);
}

.search-input-wrapper {
  display: flex;
  gap: 8px;
}

.search-input-wrapper input {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-primary);
}

.search-input-wrapper input::placeholder {
  color: var(--text-placeholder);
}

.search-input-wrapper button {
  padding: 12px 16px;
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 16px;
}

.search-results {
  margin-top: 20px;
}

.search-results h3 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.result-item {
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
  margin-bottom: 12px;
  border: 1px solid var(--border-primary);
}

.result-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.result-file {
  font-weight: 500;
  color: var(--text-primary);
}

.result-score {
  color: var(--accent-success);
  font-size: 13px;
}

.result-content {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-secondary);
}

.chunks-modal {
  max-width: 700px;
  max-height: 80vh;
  overflow-y: auto;
}

.chunks-list {
  max-height: 60vh;
  overflow-y: auto;
}

.chunk-item {
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
  margin-bottom: 12px;
  border: 1px solid var(--border-primary);
}

.chunk-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.token-count {
  color: var(--text-tertiary);
}

.chunk-content {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  color: var(--text-primary);
}

.close-btn {
  width: 100%;
  margin-top: 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-primary);
  color: var(--text-primary);
}
</style>
