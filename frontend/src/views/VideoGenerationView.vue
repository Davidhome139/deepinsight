<template>
  <div class="video-generation">
    <!-- Header -->
    <header class="page-header">
      <div class="header-left">
        <el-button text @click="$router.push('/')">
          <el-icon><ArrowLeft /></el-icon> Back
        </el-button>
        <h1>Video Generation</h1>
      </div>
      <div class="header-right">
        <el-button @click="$router.push('/image')">Images</el-button>
        <el-button @click="$router.push('/ai-chat')">AI Chat</el-button>
      </div>
    </header>

    <main class="page-content">
      <!-- Left Panel: Generation Form -->
      <aside class="generation-panel">
        <div class="panel-header">
          <el-icon><VideoCamera /></el-icon>
          <span>Create Video</span>
        </div>

        <el-form :model="form" label-position="top" class="generation-form">
          <!-- Provider Selection -->
          <el-form-item label="Provider">
            <el-select v-model="form.provider" @change="handleProviderChange">
              <el-option 
                v-for="p in providers" 
                :key="p.id" 
                :value="p.id" 
                :label="p.name"
              />
            </el-select>
          </el-form-item>

          <!-- Model Selection -->
          <el-form-item label="Model">
            <el-select v-model="form.model">
              <el-option 
                v-for="m in availableModels" 
                :key="m.id" 
                :value="m.id" 
                :label="m.name"
              >
                <div class="model-option">
                  <span>{{ m.name }}</span>
                  <el-tag size="small" type="info">{{ m.max_duration }}s max</el-tag>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <!-- Prompt -->
          <el-form-item label="Prompt">
            <el-input
              v-model="form.prompt"
              type="textarea"
              :rows="4"
              placeholder="Describe the video you want to create..."
            />
          </el-form-item>

          <!-- Reference Image -->
          <el-form-item label="Reference Image">
            <div class="image-upload-area" @click="triggerUpload">
              <input 
                ref="fileInput" 
                type="file" 
                accept="image/*" 
                @change="handleImageUpload" 
                hidden 
              />
              <div v-if="!form.imageUrl" class="upload-placeholder">
                <el-icon class="upload-icon"><Upload /></el-icon>
                <span>Click to upload image</span>
              </div>
              <div v-else class="upload-preview">
                <img :src="form.imageUrl" alt="Reference" />
                <div class="preview-overlay" @click.stop="clearImage">
                  <el-icon><Delete /></el-icon>
                </div>
              </div>
            </div>
          </el-form-item>

          <!-- Resolution -->
          <el-form-item label="Resolution">
            <el-select v-model="form.resolution">
              <el-option value="1280x720" label="HD (1280x720)" />
              <el-option value="720x480" label="SD (720x480)" />
              <el-option value="1920x1080" label="Full HD (1920x1080)" />
            </el-select>
          </el-form-item>

          <!-- Generate Button -->
          <el-button 
            type="primary" 
            size="large" 
            class="generate-btn"
            :loading="generating"
            :disabled="!canGenerate"
            @click="generateVideo"
          >
            <el-icon><VideoPlay /></el-icon>
            Generate Video
          </el-button>
        </el-form>
      </aside>

      <!-- Right Panel: Gallery -->
      <section class="gallery-panel">
        <div class="gallery-header">
          <div class="header-title">
            <el-icon><Film /></el-icon>
            <span>Video Gallery</span>
            <el-tag type="info">{{ pagination.total }} videos</el-tag>
          </div>
          <div class="header-actions">
            <el-button @click="fetchTasks" :icon="Refresh" circle />
            <el-radio-group v-model="viewMode" size="small">
              <el-radio-button value="grid"><el-icon><Grid /></el-icon></el-radio-button>
              <el-radio-button value="list"><el-icon><List /></el-icon></el-radio-button>
            </el-radio-group>
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="loadingTasks" class="loading-state">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>Loading videos...</span>
        </div>

        <!-- Empty State -->
        <div v-else-if="tasks.length === 0" class="empty-state">
          <el-icon><VideoCamera /></el-icon>
          <p>No videos yet</p>
          <span>Generate your first video to see it here</span>
        </div>

        <!-- Grid View -->
        <div v-else-if="viewMode === 'grid'" class="gallery-grid">
          <div 
            v-for="task in tasks" 
            :key="task.id" 
            class="video-card"
            :class="{ 'is-processing': task.status === 'processing' }"
            @click="selectVideo(task)"
          >
            <div class="video-thumbnail">
              <img 
                v-if="task.thumbnail_url" 
                :src="task.thumbnail_url" 
                alt="Thumbnail"
              />
              <div v-else class="thumbnail-placeholder">
                <el-icon><VideoCamera /></el-icon>
              </div>
              <div class="video-overlay">
                <div class="video-duration" v-if="task.duration">
                  {{ formatDuration(task.duration) }}
                </div>
                <div class="video-status">
                  <el-tag :type="getStatusType(task.status)" size="small">
                    {{ task.status }}
                  </el-tag>
                </div>
              </div>
              <!-- Progress Bar for Processing -->
              <div v-if="task.status === 'processing'" class="progress-bar">
                <div class="progress-fill" :style="{ width: task.progress + '%' }"></div>
              </div>
            </div>
            <div class="video-info">
              <p class="video-prompt">{{ task.prompt }}</p>
              <span class="video-date">{{ formatDate(task.created_at) }}</span>
            </div>
          </div>
        </div>

        <!-- List View -->
        <el-table v-else :data="tasks" class="video-table">
          <el-table-column label="Video" width="120">
            <template #default="{ row }">
              <div class="table-thumbnail" @click="selectVideo(row)">
                <el-icon v-if="!row.thumbnail_url"><VideoCamera /></el-icon>
                <img v-else :src="row.thumbnail_url" alt="" />
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="prompt" label="Prompt">
            <template #default="{ row }">
              <el-text truncated>{{ row.prompt }}</el-text>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="Status" width="120">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
              <span v-if="row.status === 'processing'" class="progress-text">
                {{ row.progress }}%
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="Created" width="150">
            <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="Actions" width="100">
            <template #default="{ row }">
              <el-button 
                v-if="row.status === 'completed'" 
                @click="selectVideo(row)"
                type="primary" 
                size="small"
                text
              >
                <el-icon><VideoPlay /></el-icon>
              </el-button>
              <el-button 
                @click="deleteTask(row.id)"
                type="danger"
                size="small"
                text
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- Pagination -->
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          layout="prev, pager, next"
          @current-change="fetchTasks"
          class="pagination"
        />
      </section>
    </main>

    <!-- Video Player Dialog -->
    <el-dialog 
      v-model="playerVisible" 
      :title="selectedTask?.prompt || 'Video Preview'"
      width="70%"
      class="video-player-dialog"
    >
      <div class="player-content">
        <video 
          v-if="selectedTask?.video_url" 
          :src="selectedTask.video_url" 
          controls 
          autoplay
          class="video-player"
        ></video>
        <div v-else-if="selectedTask?.status === 'processing'" class="processing-state">
          <el-progress 
            type="circle" 
            :percentage="selectedTask.progress" 
            :width="120"
          />
          <p>Video is being generated...</p>
        </div>
      </div>
      <div class="player-info">
        <el-descriptions :column="3" border>
          <el-descriptions-item label="Provider">{{ selectedTask?.provider }}</el-descriptions-item>
          <el-descriptions-item label="Model">{{ selectedTask?.model }}</el-descriptions-item>
          <el-descriptions-item label="Resolution">{{ selectedTask?.resolution || 'N/A' }}</el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getStatusType(selectedTask?.status || '')">{{ selectedTask?.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Created">{{ formatDate(selectedTask?.created_at || '') }}</el-descriptions-item>
          <el-descriptions-item label="File Size">{{ formatFileSize(selectedTask?.file_size) }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft, VideoCamera, Upload, Delete, VideoPlay, Film,
  Refresh, Grid, List, Loading
} from '@element-plus/icons-vue'

interface VideoModel {
  id: string
  name: string
  provider: string
  max_duration: number
  resolutions: string[]
}

interface VideoTask {
  id: string
  provider: string
  model: string
  prompt: string
  status: string
  progress: number
  video_url?: string
  thumbnail_url?: string
  duration?: number
  resolution?: string
  file_size?: number
  created_at: string
}

interface Provider {
  id: string
  name: string
  models: VideoModel[]
}

const form = ref({
  provider: 'baidu-air',
  model: 'musesteamer-air-i2v',
  prompt: '',
  imageUrl: '',
  resolution: '1280x720'
})

const providers = ref<Provider[]>([
  { id: 'baidu-air', name: 'Baidu Air', models: [] },
  { id: 'stability', name: 'Stability AI', models: [] },
  { id: 'local', name: 'Local (FFmpeg)', models: [] }
])

const availableModels = ref<VideoModel[]>([
  { id: 'musesteamer-air-i2v', name: 'Muse Steamer Air I2V', provider: 'baidu-air', max_duration: 5, resolutions: ['1280x720'] },
  { id: 'musesteamer-2.0-i2v', name: 'Muse Steamer 2.0 I2V', provider: 'baidu-air', max_duration: 10, resolutions: ['1920x1080', '1280x720'] }
])

const tasks = ref<VideoTask[]>([])
const generating = ref(false)
const loadingTasks = ref(false)
const viewMode = ref<'grid' | 'list'>('grid')
const playerVisible = ref(false)
const selectedTask = ref<VideoTask | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const pagination = ref({
  page: 1,
  size: 12,
  total: 0
})

let pollInterval: number | null = null

const canGenerate = computed(() => form.value.prompt && form.value.imageUrl)

const handleProviderChange = async () => {
  try {
    const response = await fetch(`/api/v1/video/models`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    if (response.ok) {
      const data = await response.json()
      availableModels.value = data.models.filter((m: VideoModel) => m.provider === form.value.provider)
      if (availableModels.value.length > 0) {
        form.value.model = availableModels.value[0].id
      }
    }
  } catch (e) {
    console.error('Failed to fetch models', e)
  }
}

const triggerUpload = () => fileInput.value?.click()

const handleImageUpload = (e: Event) => {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    const reader = new FileReader()
    reader.onload = (e) => form.value.imageUrl = e.target?.result as string
    reader.readAsDataURL(file)
  }
}

const clearImage = () => {
  form.value.imageUrl = ''
  if (fileInput.value) fileInput.value.value = ''
}

const generateVideo = async () => {
  if (!canGenerate.value) return
  generating.value = true
  try {
    const response = await fetch('/api/v1/video/generate', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        provider: form.value.provider,
        model: form.value.model,
        prompt: form.value.prompt,
        image_url: form.value.imageUrl,
        resolution: form.value.resolution
      })
    })

    if (!response.ok) throw new Error('Generation failed')

    const result = await response.json()
    ElMessage.success('Video generation started!')
    form.value.prompt = ''
    clearImage()
    await fetchTasks()
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to generate video')
  } finally {
    generating.value = false
  }
}

const fetchTasks = async () => {
  loadingTasks.value = true
  try {
    const response = await fetch(
      `/api/v1/video/tasks?page=${pagination.value.page}&size=${pagination.value.size}`,
      { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` } }
    )
    if (response.ok) {
      const result = await response.json()
      tasks.value = result.items || []
      pagination.value.total = result.total || 0
    }
  } catch (e) {
    console.error('Failed to fetch tasks', e)
  } finally {
    loadingTasks.value = false
  }
}

const selectVideo = (task: VideoTask) => {
  selectedTask.value = task
  playerVisible.value = true
}

const deleteTask = async (id: string) => {
  try {
    await ElMessageBox.confirm('Delete this video?', 'Confirm')
    const response = await fetch(`/api/v1/video/tasks/${id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    if (response.ok) {
      ElMessage.success('Video deleted')
      await fetchTasks()
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('Failed to delete')
  }
}

const getStatusType = (status: string) => {
  const map: Record<string, any> = {
    completed: 'success', failed: 'danger', processing: 'warning', pending: 'info'
  }
  return map[status] || 'info'
}

const formatDate = (date: string) => {
  if (!date) return 'N/A'
  return new Date(date).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}

const formatDuration = (seconds: number) => {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

const formatFileSize = (bytes?: number) => {
  if (!bytes) return 'N/A'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const pollTasks = () => {
  const hasProcessing = tasks.value.some(t => t.status === 'processing')
  if (hasProcessing && !pollInterval) {
    pollInterval = window.setInterval(fetchTasks, 5000)
  } else if (!hasProcessing && pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

onMounted(() => {
  fetchTasks()
  handleProviderChange()
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
.video-generation {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f0f2f5;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  background: white;
  border-bottom: 1px solid #e4e7ed;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-left h1 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.header-right {
  display: flex;
  gap: 8px;
}

.page-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.generation-panel {
  width: 360px;
  background: white;
  border-right: 1px solid #e4e7ed;
  padding: 20px;
  overflow-y: auto;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.generation-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

.model-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.image-upload-area {
  width: 100%;
  height: 200px;
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
  overflow: hidden;
}

.image-upload-area:hover {
  border-color: #409eff;
}

.upload-placeholder {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #909399;
}

.upload-icon {
  font-size: 48px;
}

.upload-preview {
  position: relative;
  height: 100%;
}

.upload-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.preview-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.3s;
  color: white;
  font-size: 24px;
}

.upload-preview:hover .preview-overlay {
  opacity: 1;
}

.generate-btn {
  width: 100%;
  margin-top: 16px;
}

.gallery-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.gallery-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.loading-state, .empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #909399;
}

.empty-state .el-icon {
  font-size: 64px;
}

.gallery-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
  overflow-y: auto;
  padding-bottom: 60px;
}

.video-card {
  background: white;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

.video-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.12);
}

.video-thumbnail {
  position: relative;
  aspect-ratio: 16/9;
  background: #f5f7fa;
}

.video-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumbnail-placeholder {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
  color: #dcdfe6;
}

.video-overlay {
  position: absolute;
  inset: 0;
  padding: 8px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.video-duration {
  align-self: flex-end;
  background: rgba(0,0,0,0.7);
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.video-status {
  align-self: flex-start;
}

.progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: rgba(255,255,255,0.3);
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #409eff, #67c23a);
  transition: width 0.3s;
}

.video-info {
  padding: 12px;
}

.video-prompt {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #303133;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.video-date {
  font-size: 12px;
  color: #909399;
}

.table-thumbnail {
  width: 80px;
  height: 45px;
  background: #f5f7fa;
  border-radius: 4px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.table-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.progress-text {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
}

.pagination {
  margin-top: 16px;
  justify-content: center;
}

.video-player-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.player-content {
  background: black;
}

.video-player {
  width: 100%;
  max-height: 60vh;
}

.processing-state {
  height: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  color: white;
}

.player-info {
  padding: 16px;
}
</style>
