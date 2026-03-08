<template>
  <div class="image-generation">
    <!-- Header -->
    <header class="page-header">
      <div class="header-left">
        <el-button text @click="$router.push('/')">
          <el-icon><ArrowLeft /></el-icon> Back
        </el-button>
        <h1>Image Generation</h1>
      </div>
      <div class="header-right">
        <el-button @click="$router.push('/video')">Videos</el-button>
        <el-button @click="$router.push('/ai-chat')">AI Chat</el-button>
      </div>
    </header>

    <main class="page-content">
      <!-- Left Panel: Generation Form -->
      <aside class="generation-panel">
        <div class="panel-header">
          <el-icon><Picture /></el-icon>
          <span>Create Image</span>
        </div>

        <el-form :model="form" label-position="top" class="generation-form">
          <!-- Provider Selection -->
          <el-form-item label="Provider">
            <el-select v-model="form.provider" @change="handleProviderChange">
              <el-option value="aliyun" label="Aliyun Wanx (通义万象)" />
              <el-option value="stability" label="Stability AI" />
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
              />
            </el-select>
          </el-form-item>

          <!-- Prompt -->
          <el-form-item label="Prompt">
            <el-input
              v-model="form.prompt"
              type="textarea"
              :rows="4"
              placeholder="Describe the image you want to create..."
            />
          </el-form-item>

          <!-- Size -->
          <el-form-item label="Size">
            <el-select v-model="form.size">
              <el-option value="1024*1024" label="Square (1024x1024)" />
              <el-option value="1280*720" label="Landscape (1280x720)" />
              <el-option value="720*1280" label="Portrait (720x1280)" />
            </el-select>
          </el-form-item>

          <!-- Style (for supported models) -->
          <el-form-item label="Style" v-if="form.provider === 'stability'">
            <el-select v-model="form.style">
              <el-option value="" label="Auto" />
              <el-option value="photographic" label="Photographic" />
              <el-option value="anime" label="Anime" />
              <el-option value="digital-art" label="Digital Art" />
              <el-option value="cinematic" label="Cinematic" />
            </el-select>
          </el-form-item>

          <!-- Cost Estimate -->
          <div class="cost-estimate">
            <span>Estimated: </span>
            <strong>¥{{ estimatedCost.toFixed(2) }}</strong>
          </div>

          <!-- Generate Button -->
          <el-button 
            type="primary" 
            size="large" 
            class="generate-btn"
            :loading="generating"
            :disabled="!form.prompt"
            @click="generateImage"
          >
            <el-icon><MagicStick /></el-icon>
            Generate Image
          </el-button>

          <!-- Variation / Edit Buttons -->
          <div class="secondary-actions" v-if="currentImage">
            <el-button @click="createVariation" :loading="creatingVariation">
              <el-icon><CopyDocument /></el-icon>
              Create Variation
            </el-button>
          </div>
        </el-form>
      </aside>

      <!-- Right Panel: Gallery -->
      <section class="gallery-panel">
        <div class="gallery-header">
          <div class="header-title">
            <el-icon><PictureFilled /></el-icon>
            <span>Image Gallery</span>
            <el-tag type="info">{{ pagination.total }} images</el-tag>
          </div>
          <div class="header-actions">
            <el-button @click="loadHistory" :icon="Refresh" circle />
            <el-radio-group v-model="viewMode" size="small">
              <el-radio-button value="grid"><el-icon><Grid /></el-icon></el-radio-button>
              <el-radio-button value="list"><el-icon><List /></el-icon></el-radio-button>
            </el-radio-group>
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="loadingHistory" class="loading-state">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>Loading images...</span>
        </div>

        <!-- Empty State -->
        <div v-else-if="history.length === 0" class="empty-state">
          <el-icon><Picture /></el-icon>
          <p>No images yet</p>
          <span>Generate your first image to see it here</span>
        </div>

        <!-- Grid View -->
        <div v-else class="gallery-grid">
          <div 
            v-for="img in history" 
            :key="img.id" 
            class="image-card"
            @click="openLightbox(img)"
          >
            <div class="image-thumbnail">
              <img :src="img.image_url" :alt="img.prompt" loading="lazy" />
              <div class="image-overlay">
                <div class="overlay-actions">
                  <el-button circle size="small" @click.stop="downloadImage(img)">
                    <el-icon><Download /></el-icon>
                  </el-button>
                  <el-button circle size="small" type="danger" @click.stop="deleteImage(img.id)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
            </div>
            <div class="image-info">
              <p class="image-prompt">{{ img.prompt }}</p>
              <div class="image-meta">
                <el-tag size="small" type="info">{{ img.model }}</el-tag>
                <span class="image-date">{{ formatDate(img.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Load More -->
        <div v-if="hasMore" class="load-more">
          <el-button @click="loadMore" :loading="loadingMore">
            Load More
          </el-button>
        </div>
      </section>
    </main>

    <!-- Lightbox Dialog -->
    <el-dialog 
      v-model="lightboxVisible" 
      :title="currentImage?.prompt || 'Image Preview'"
      width="80%"
      class="lightbox-dialog"
    >
      <div class="lightbox-content">
        <div class="lightbox-image">
          <img :src="currentImage?.image_url" :alt="currentImage?.prompt" />
        </div>
        <div class="lightbox-sidebar">
          <div class="sidebar-section">
            <h4>Details</h4>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Model">{{ currentImage?.model }}</el-descriptions-item>
              <el-descriptions-item label="Size">{{ currentImage?.size }}</el-descriptions-item>
              <el-descriptions-item label="Quality">{{ currentImage?.quality || 'standard' }}</el-descriptions-item>
              <el-descriptions-item label="Cost">¥{{ currentImage?.cost?.toFixed(2) }}</el-descriptions-item>
              <el-descriptions-item label="Created">{{ formatDate(currentImage?.created_at || '') }}</el-descriptions-item>
            </el-descriptions>
          </div>

          <div class="sidebar-section" v-if="currentImage?.revised_prompt">
            <h4>Revised Prompt</h4>
            <p class="revised-prompt">{{ currentImage.revised_prompt }}</p>
          </div>

          <div class="sidebar-actions">
            <el-button type="primary" @click="downloadImage(currentImage)">
              <el-icon><Download /></el-icon>
              Download
            </el-button>
            <el-button @click="useAsReference">
              <el-icon><Pointer /></el-icon>
              Use as Reference
            </el-button>
            <el-button @click="createVariation" :loading="creatingVariation">
              <el-icon><CopyDocument /></el-icon>
              Create Variation
            </el-button>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft, Picture, MagicStick, PictureFilled, Grid, List,
  Loading, Download, Delete, CopyDocument, Pointer, Refresh
} from '@element-plus/icons-vue'
import request from '../utils/request'

interface ImageModel {
  id: string
  name: string
  provider: string
}

interface GeneratedImage {
  id: string
  prompt: string
  revised_prompt?: string
  model: string
  size: string
  quality?: string
  style?: string
  image_url: string
  cost?: number
  created_at: string
}

const form = ref({
  provider: 'aliyun',
  model: 'wanx-v1',
  prompt: '',
  size: '1024*1024',
  quality: 'standard',
  style: ''
})

const availableModels = ref<ImageModel[]>([
  { id: 'wanx-v1', name: '通义万象 v1', provider: 'aliyun' },
  { id: 'wanx-sketch-to-image-v1', name: '通义万象草图生图', provider: 'aliyun' },
  { id: 'stable-diffusion-xl', name: 'Stable Diffusion XL', provider: 'stability' }
])

const history = ref<GeneratedImage[]>([])
const currentImage = ref<GeneratedImage | null>(null)
const generating = ref(false)
const creatingVariation = ref(false)
const loadingHistory = ref(true)
const loadingMore = ref(false)
const lightboxVisible = ref(false)
const viewMode = ref<'grid' | 'list'>('grid')
const hasMore = ref(true)
const offset = ref(0)

const pagination = ref({
  total: 0
})

const estimatedCost = computed(() => {
  // Aliyun Wanx pricing
  if (form.value.provider === 'aliyun') return 0.16
  // Stability AI pricing
  return 0.04
})

const handleProviderChange = () => {
  const filtered = availableModels.value.filter(m => m.provider === form.value.provider)
  if (filtered.length > 0) {
    form.value.model = filtered[0].id
  }
}

const generateImage = async () => {
  if (!form.value.prompt) return
  
  generating.value = true
  try {
    const res = await request.post('/image/generate', {
      prompt: form.value.prompt,
      model: form.value.model,
      size: form.value.size,
      quality: form.value.quality,
      style: form.value.style
    })
    const newImage = (res as any)
    currentImage.value = newImage
    ElMessage.success('Image generated successfully!')
    offset.value = 0
    await loadHistory()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Generation failed')
  } finally {
    generating.value = false
  }
}

const createVariation = async () => {
  if (!currentImage.value) return
  
  creatingVariation.value = true
  try {
    const res = await request.post('/image/variations', {
      source_image_id: currentImage.value.id,
      prompt: form.value.prompt || currentImage.value.prompt
    })
    const newImage = (res as any)
    currentImage.value = newImage
    ElMessage.success('Variation created!')
    offset.value = 0
    await loadHistory()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Failed to create variation')
  } finally {
    creatingVariation.value = false
  }
}

const loadHistory = async () => {
  loadingHistory.value = true
  try {
    const res = await request.get('/image/history', {
      params: { limit: 12, offset: 0 }
    })
    history.value = (res as any)?.images || []
    pagination.value.total = (res as any)?.total || history.value.length
    hasMore.value = history.value.length >= 12
    offset.value = 12
  } catch (error) {
    console.error('Failed to load history:', error)
  } finally {
    loadingHistory.value = false
  }
}

const loadMore = async () => {
  loadingMore.value = true
  try {
    const res = await request.get('/image/history', {
      params: { limit: 12, offset: offset.value }
    })
    const newImages = (res as any)?.images || []
    history.value.push(...newImages)
    hasMore.value = newImages.length >= 12
    offset.value += 12
  } catch (error) {
    console.error('Failed to load more:', error)
  } finally {
    loadingMore.value = false
  }
}

const openLightbox = (img: GeneratedImage) => {
  currentImage.value = img
  lightboxVisible.value = true
}

const useAsReference = () => {
  if (currentImage.value) {
    form.value.prompt = currentImage.value.prompt
    form.value.model = currentImage.value.model
    form.value.size = currentImage.value.size
    lightboxVisible.value = false
  }
}

const downloadImage = (img: GeneratedImage | null) => {
  if (!img?.image_url) return
  const link = document.createElement('a')
  link.href = img.image_url
  link.download = `generated-image-${Date.now()}.png`
  link.target = '_blank'
  link.click()
}

const deleteImage = async (id: string) => {
  try {
    await ElMessageBox.confirm('Delete this image?', 'Confirm')
    await request.delete(`/image/${id}`)
    history.value = history.value.filter(img => img.id !== id)
    if (currentImage.value?.id === id) {
      currentImage.value = null
      lightboxVisible.value = false
    }
    ElMessage.success('Image deleted')
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error('Failed to delete')
  }
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return 'N/A'
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}

watch(() => form.value.provider, handleProviderChange)

onMounted(loadHistory)
</script>

<style scoped>
.image-generation {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-primary);
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
  color: var(--text-primary);
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
  background: var(--bg-primary);
  border-right: 1px solid var(--border-primary);
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
  color: var(--text-primary);
}

.generation-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--text-secondary);
}

.cost-estimate {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: var(--bg-tertiary);
  border-radius: 8px;
  margin-bottom: 16px;
  font-size: 14px;
  color: var(--text-secondary);
}

.cost-estimate strong {
  color: var(--accent-primary);
  font-size: 18px;
}

.generate-btn {
  width: 100%;
}

.secondary-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.secondary-actions .el-button {
  flex: 1;
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
  color: var(--text-primary);
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
  color: var(--text-tertiary);
}

.empty-state .el-icon {
  font-size: 64px;
}

.gallery-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  overflow-y: auto;
  padding-bottom: 60px;
}

.image-card {
  background: var(--card-bg);
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: var(--shadow-sm);
}

.image-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-md);
}

.image-thumbnail {
  position: relative;
  aspect-ratio: 1;
  background: var(--bg-tertiary);
}

.image-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.3s;
}

.image-card:hover .image-overlay {
  opacity: 1;
}

.overlay-actions {
  display: flex;
  gap: 8px;
}

.image-info {
  padding: 12px;
}

.image-prompt {
  margin: 0 0 8px 0;
  font-size: 13px;
  color: #303133;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.image-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.image-date {
  font-size: 12px;
  color: #909399;
}

.load-more {
  text-align: center;
  margin-top: 16px;
}

.lightbox-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.lightbox-content {
  display: flex;
  height: 70vh;
}

.lightbox-image {
  flex: 1;
  background: #000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.lightbox-image img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.lightbox-sidebar {
  width: 300px;
  background: #fafafa;
  padding: 20px;
  overflow-y: auto;
}

.sidebar-section {
  margin-bottom: 20px;
}

.sidebar-section h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.revised-prompt {
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
  padding: 12px;
  background: white;
  border-radius: 8px;
}

.sidebar-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sidebar-actions .el-button {
  width: 100%;
}
</style>
