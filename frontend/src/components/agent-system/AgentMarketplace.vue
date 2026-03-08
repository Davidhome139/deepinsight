<template>
  <div class="agent-marketplace">
    <!-- Header -->
    <div class="marketplace-header">
      <div class="header-content">
        <h1>Agent Marketplace</h1>
        <p>Discover, share, and import community agents and workflows</p>
      </div>
      <el-input 
        v-model="searchQuery" 
        placeholder="Search agents and workflows..."
        class="search-input"
        @keyup.enter="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <!-- Featured Section -->
    <div v-if="!searchQuery && !selectedCategory" class="featured-section">
      <h2>Featured</h2>
      <div class="featured-grid">
        <div 
          v-for="item in featuredItems" 
          :key="item.id"
          class="featured-card"
          @click="viewItem(item)"
        >
          <div class="featured-icon">{{ item.icon || (item.type === 'agent' ? '🤖' : '📋') }}</div>
          <div class="featured-info">
            <div class="featured-name">{{ item.name }}</div>
            <div class="featured-author">by {{ item.author_name }}</div>
            <div class="featured-stats">
              <span>⬇️ {{ item.downloads }}</span>
              <span>⭐ {{ item.stars }}</span>
              <span>{{ item.avg_rating.toFixed(1) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Categories -->
    <div class="categories-section">
      <div class="categories-header">
        <h2>{{ selectedCategory ? selectedCategory : 'Categories' }}</h2>
        <el-button v-if="selectedCategory" text @click="selectedCategory = ''">Clear filter</el-button>
      </div>
      <div v-if="!selectedCategory" class="categories-grid">
        <div 
          v-for="cat in categories" 
          :key="cat.name"
          class="category-card"
          @click="selectedCategory = cat.name"
        >
          <div class="category-icon">{{ getCategoryIcon(cat.name) }}</div>
          <div class="category-name">{{ cat.name }}</div>
          <div class="category-count">{{ cat.count }} items</div>
        </div>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters-bar">
      <el-radio-group v-model="typeFilter" @change="loadItems">
        <el-radio-button label="">All</el-radio-button>
        <el-radio-button label="agent">Agents</el-radio-button>
        <el-radio-button label="workflow">Workflows</el-radio-button>
      </el-radio-group>
      
      <el-select v-model="sortBy" @change="loadItems" style="width: 150px">
        <el-option label="Most Downloads" value="downloads" />
        <el-option label="Highest Rated" value="rating" />
        <el-option label="Most Stars" value="stars" />
        <el-option label="Recently Added" value="recent" />
      </el-select>
    </div>

    <!-- Items Grid -->
    <div class="items-grid">
      <div v-if="loading" class="loading-state">
        <el-icon class="is-loading"><Loading /></el-icon>
        Loading...
      </div>

      <el-empty v-else-if="items.length === 0" description="No items found" />

      <div 
        v-else
        v-for="item in items" 
        :key="item.id"
        class="item-card"
        @click="viewItem(item)"
      >
        <div class="item-header">
          <div class="item-icon">{{ item.icon || (item.type === 'agent' ? '🤖' : '📋') }}</div>
          <el-tag :type="item.type === 'agent' ? 'primary' : 'success'" size="small">
            {{ item.type }}
          </el-tag>
        </div>
        <div class="item-body">
          <div class="item-name">{{ item.name }}</div>
          <div class="item-desc">{{ truncate(item.description, 100) }}</div>
          <div class="item-tags">
            <el-tag v-for="tag in parseTags(item.tags).slice(0, 3)" :key="tag" size="small" effect="plain">
              {{ tag }}
            </el-tag>
          </div>
        </div>
        <div class="item-footer">
          <div class="item-author">{{ item.author_name }}</div>
          <div class="item-stats">
            <span title="Downloads">⬇️ {{ item.downloads }}</span>
            <span title="Stars">⭐ {{ item.stars }}</span>
            <span title="Rating">{{ item.avg_rating.toFixed(1) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > limit" class="pagination">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="limit"
        :total="total"
        layout="prev, pager, next"
        @current-change="loadItems"
      />
    </div>

    <!-- Item Detail Dialog -->
    <el-dialog v-model="showDetailDialog" width="800px" :title="selectedItem?.name">
      <template v-if="selectedItem">
        <div class="detail-header">
          <div class="detail-icon">{{ selectedItem.icon || '🤖' }}</div>
          <div class="detail-info">
            <div class="detail-meta">
              <el-tag :type="selectedItem.type === 'agent' ? 'primary' : 'success'">
                {{ selectedItem.type }}
              </el-tag>
              <span>v{{ selectedItem.version }}</span>
              <span>by {{ selectedItem.author_name }}</span>
            </div>
            <div class="detail-stats">
              <span>⬇️ {{ selectedItem.downloads }} downloads</span>
              <span>⭐ {{ selectedItem.stars }} stars</span>
              <span>⭐ {{ selectedItem.avg_rating.toFixed(1) }} ({{ selectedItem.rating_count }} reviews)</span>
            </div>
          </div>
          <div class="detail-actions">
            <el-button @click="toggleStar" :type="isStarred ? 'warning' : ''">
              {{ isStarred ? '⭐ Starred' : '☆ Star' }}
            </el-button>
            <el-button type="primary" @click="downloadItem" :loading="downloading">
              Download
            </el-button>
          </div>
        </div>

        <el-divider />

        <div class="detail-description">
          <h4>Description</h4>
          <p>{{ selectedItem.description }}</p>
        </div>

        <div v-if="selectedItem.documentation" class="detail-docs">
          <h4>Documentation</h4>
          <div v-html="renderMarkdown(selectedItem.documentation)"></div>
        </div>

        <div class="detail-requirements">
          <h4>Requirements</h4>
          <div class="requirement-tags">
            <el-tag v-for="tool in parseRequiredTools(selectedItem.required_tools)" :key="tool" effect="plain">
              {{ tool }}
            </el-tag>
          </div>
          <div v-if="selectedItem.min_model_capability" class="min-model">
            Minimum model: {{ selectedItem.min_model_capability }}
          </div>
        </div>

        <el-divider />

        <div class="detail-reviews">
          <div class="reviews-header">
            <h4>Reviews</h4>
            <el-button @click="showReviewDialog = true">Write Review</el-button>
          </div>
          <div class="reviews-list">
            <div v-for="review in reviews" :key="review.id" class="review-item">
              <div class="review-header">
                <span class="review-author">{{ review.username }}</span>
                <span class="review-rating">{'⭐'.repeat(review.rating)}</span>
                <el-tag v-if="review.is_verified" size="small" type="success">Verified</el-tag>
              </div>
              <div class="review-title">{{ review.title }}</div>
              <div class="review-comment">{{ review.comment }}</div>
            </div>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- Review Dialog -->
    <el-dialog v-model="showReviewDialog" title="Write Review" width="500px">
      <el-form :model="reviewForm" label-position="top">
        <el-form-item label="Rating">
          <el-rate v-model="reviewForm.rating" />
        </el-form-item>
        <el-form-item label="Title">
          <el-input v-model="reviewForm.title" placeholder="Summary of your review" />
        </el-form-item>
        <el-form-item label="Review">
          <el-input v-model="reviewForm.comment" type="textarea" :rows="4" placeholder="Share your experience..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showReviewDialog = false">Cancel</el-button>
        <el-button type="primary" @click="submitReview" :loading="submittingReview">Submit</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Loading } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const searchQuery = ref('')
const selectedCategory = ref('')
const typeFilter = ref('')
const sortBy = ref('downloads')
const currentPage = ref(1)
const limit = 20
const total = ref(0)
const loading = ref(false)
const downloading = ref(false)

const items = ref<any[]>([])
const featuredItems = ref<any[]>([])
const categories = ref<any[]>([])

const showDetailDialog = ref(false)
const selectedItem = ref<any>(null)
const isStarred = ref(false)
const reviews = ref<any[]>([])

const showReviewDialog = ref(false)
const submittingReview = ref(false)
const reviewForm = reactive({
  rating: 5,
  title: '',
  comment: ''
})

onMounted(async () => {
  await Promise.all([
    loadFeatured(),
    loadCategories(),
    loadItems()
  ])
})

watch(selectedCategory, () => {
  currentPage.value = 1
  loadItems()
})

const loadFeatured = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/marketplace/featured?limit=4', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      featuredItems.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load featured:', error)
  }
}

const loadCategories = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/marketplace/categories', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      categories.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load categories:', error)
  }
}

const loadItems = async () => {
  loading.value = true
  try {
    let url = `/api/v1/agent-system/marketplace/search?sort=${sortBy.value}&limit=${limit}&offset=${(currentPage.value - 1) * limit}`
    if (searchQuery.value) url += `&q=${encodeURIComponent(searchQuery.value)}`
    if (selectedCategory.value) url += `&category=${encodeURIComponent(selectedCategory.value)}`
    if (typeFilter.value) url += `&type=${typeFilter.value}`

    const response = await fetch(url, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      const data = await response.json()
      items.value = data.items || []
      total.value = data.total || 0
    }
  } catch (error) {
    console.error('Failed to load items:', error)
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  selectedCategory.value = ''
  loadItems()
}

const viewItem = async (item: any) => {
  selectedItem.value = item
  showDetailDialog.value = true
  await loadReviews(item.id)
}

const loadReviews = async (itemId: string) => {
  try {
    const response = await fetch(`/api/v1/agent-system/marketplace/items/${itemId}/reviews`, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      reviews.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load reviews:', error)
  }
}

const downloadItem = async () => {
  if (!selectedItem.value) return
  
  downloading.value = true
  try {
    const response = await fetch(`/api/v1/agent-system/marketplace/items/${selectedItem.value.id}/download`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      ElMessage.success(`${selectedItem.value.type === 'agent' ? 'Agent' : 'Workflow'} imported successfully!`)
      showDetailDialog.value = false
    } else {
      const error = await response.json()
      ElMessage.error(error.error || 'Download failed')
    }
  } catch (error) {
    ElMessage.error('Download failed')
  } finally {
    downloading.value = false
  }
}

const toggleStar = async () => {
  if (!selectedItem.value) return
  
  try {
    const method = isStarred.value ? 'DELETE' : 'POST'
    const response = await fetch(`/api/v1/agent-system/marketplace/items/${selectedItem.value.id}/star`, {
      method,
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      isStarred.value = !isStarred.value
      selectedItem.value.stars += isStarred.value ? 1 : -1
    }
  } catch (error) {
    console.error('Failed to toggle star:', error)
  }
}

const submitReview = async () => {
  if (!selectedItem.value) return
  if (!reviewForm.title.trim() || !reviewForm.comment.trim()) {
    ElMessage.error('Please fill in all fields')
    return
  }

  submittingReview.value = true
  try {
    const response = await fetch(`/api/v1/agent-system/marketplace/items/${selectedItem.value.id}/reviews`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify(reviewForm)
    })
    if (response.ok) {
      ElMessage.success('Review submitted')
      showReviewDialog.value = false
      await loadReviews(selectedItem.value.id)
      reviewForm.rating = 5
      reviewForm.title = ''
      reviewForm.comment = ''
    }
  } catch (error) {
    ElMessage.error('Failed to submit review')
  } finally {
    submittingReview.value = false
  }
}

const getCategoryIcon = (name: string) => {
  const icons: Record<string, string> = {
    'coding': '💻',
    'writing': '✍️',
    'analysis': '📊',
    'automation': '⚙️',
    'creative': '🎨',
    'research': '🔬',
    'data': '📈',
    'integration': '🔗'
  }
  return icons[name.toLowerCase()] || '📦'
}

const parseTags = (tagsStr: string) => {
  if (!tagsStr) return []
  try {
    return JSON.parse(tagsStr)
  } catch (e) {
    return []
  }
}

const parseRequiredTools = (toolsStr: string) => {
  if (!toolsStr) return []
  try {
    return JSON.parse(toolsStr)
  } catch (e) {
    return []
  }
}

const truncate = (str: string, len: number) => {
  if (!str) return ''
  return str.length > len ? str.substring(0, len) + '...' : str
}

const renderMarkdown = (md: string) => {
  // Simple markdown rendering - in production use a library like marked
  return md
    .replace(/\n/g, '<br>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
}
</script>

<style scoped>
.agent-marketplace {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.marketplace-header {
  text-align: center;
  margin-bottom: 32px;
}

.marketplace-header h1 {
  margin: 0 0 8px 0;
}

.marketplace-header p {
  color: var(--el-text-color-secondary);
  margin: 0 0 16px 0;
}

.search-input {
  max-width: 500px;
}

.featured-section {
  margin-bottom: 32px;
}

.featured-section h2 {
  margin: 0 0 16px 0;
}

.featured-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.featured-card {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, var(--el-color-primary-light-9), var(--el-bg-color));
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.featured-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.featured-icon {
  font-size: 32px;
}

.featured-info {
  flex: 1;
}

.featured-name {
  font-weight: 600;
  margin-bottom: 4px;
}

.featured-author {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.featured-stats {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.categories-section {
  margin-bottom: 24px;
}

.categories-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.categories-header h2 {
  margin: 0;
}

.categories-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.category-card {
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
}

.category-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.category-icon {
  font-size: 28px;
  margin-bottom: 8px;
}

.category-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.category-count {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.filters-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.items-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.loading-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 48px;
  color: var(--el-text-color-secondary);
}

.item-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s;
}

.item-card:hover {
  border-color: var(--el-color-primary);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-fill-color-light);
}

.item-icon {
  font-size: 28px;
}

.item-body {
  padding: 16px;
}

.item-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}

.item-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
  line-height: 1.4;
}

.item-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.item-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
}

.item-author {
  color: var(--el-text-color-secondary);
}

.item-stats {
  display: flex;
  gap: 12px;
  color: var(--el-text-color-secondary);
}

.pagination {
  display: flex;
  justify-content: center;
}

/* Detail Dialog */
.detail-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.detail-icon {
  font-size: 48px;
}

.detail-info {
  flex: 1;
}

.detail-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.detail-stats {
  display: flex;
  gap: 16px;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.detail-actions {
  display: flex;
  gap: 8px;
}

.detail-description,
.detail-docs,
.detail-requirements {
  margin-bottom: 24px;
}

.detail-description h4,
.detail-docs h4,
.detail-requirements h4,
.reviews-header h4 {
  margin: 0 0 8px 0;
  color: var(--el-text-color-secondary);
}

.requirement-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.min-model {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.reviews-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.reviews-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.review-item {
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.review-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.review-author {
  font-weight: 500;
}

.review-rating {
  color: #f7ba2a;
}

.review-title {
  font-weight: 500;
  margin-bottom: 4px;
}

.review-comment {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>
