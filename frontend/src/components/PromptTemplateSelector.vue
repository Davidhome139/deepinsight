<template>
  <div class="prompt-template-selector">
    <!-- Trigger Button -->
    <el-popover
      v-model:visible="showPopover"
      placement="top-start"
      :width="600"
      trigger="click"
    >
      <template #reference>
        <el-button size="small" class="template-trigger-btn" title="Prompt Templates">
          <el-icon><Document /></el-icon>
          <span v-if="!compact">Templates</span>
        </el-button>
      </template>

      <div class="template-popover-content">
        <!-- Header with Search -->
        <div class="popover-header">
          <el-input
            v-model="searchQuery"
            placeholder="Search templates..."
            prefix-icon="Search"
            clearable
            size="small"
          />
          <el-select v-model="complexityFilter" placeholder="Complexity" size="small" clearable style="width: 120px; margin-left: 8px;" @click.stop :teleported="false">
            <el-option label="Simple" value="simple" />
            <el-option label="Medium" value="medium" />
            <el-option label="Complex" value="complex" />
          </el-select>
        </div>

        <!-- Category Tabs -->
        <div v-if="!selectedTemplate" class="category-section">
          <div class="category-tabs">
            <div
              v-for="cat in filteredCategories"
              :key="cat.id"
              class="category-tab"
              :class="{ active: selectedCategory === cat.id }"
              @click="selectCategory(cat.id)"
            >
              <span class="category-icon">{{ cat.icon }}</span>
              <span class="category-name">{{ cat.name }}</span>
            </div>
          </div>

          <!-- Topics and Templates -->
          <div v-if="selectedCategory" class="topics-section">
            <div v-for="topic in currentTopics" :key="topic.id" class="topic-group">
              <div class="topic-header">{{ topic.name }}</div>
              <div class="template-list">
                <div
                  v-for="tmpl in getFilteredTemplates(topic.templates)"
                  :key="tmpl.id"
                  class="template-item"
                  @click="selectTemplate(tmpl)"
                >
                  <div class="template-info">
                    <span class="template-name">{{ tmpl.name }}</span>
                    <el-tag :type="getComplexityType(tmpl.complexity)" size="small">
                      {{ tmpl.complexity }}
                    </el-tag>
                  </div>
                  <div class="template-desc">{{ tmpl.description }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Template Form -->
        <div v-else class="template-form-section">
          <div class="form-header">
            <el-button text @click="selectedTemplate = null">
              <el-icon><ArrowLeft /></el-icon> Back
            </el-button>
            <span class="form-title">{{ selectedTemplate.name }}</span>
            <el-tag :type="getComplexityType(selectedTemplate.complexity)" size="small">
              {{ selectedTemplate.complexity }}
            </el-tag>
          </div>

          <div class="form-body">
            <div
              v-for="(varDef, varKey) in selectedTemplate.customVariables"
              :key="varKey"
              class="form-field"
            >
              <label :class="{ required: varDef.required }">{{ varDef.label }}</label>
              
              <el-select
                v-if="varDef.type === 'select'"
                v-model="formValues[varKey]"
                :placeholder="`Select ${varDef.label}`"
                style="width: 100%"
              >
                <el-option
                  v-for="opt in varDef.options"
                  :key="opt"
                  :label="opt"
                  :value="opt"
                />
              </el-select>
              
              <el-input
                v-else-if="varDef.type === 'textarea'"
                v-model="formValues[varKey]"
                type="textarea"
                :rows="3"
                :placeholder="varDef.default || `Enter ${varDef.label}`"
              />
              
              <el-input
                v-else
                v-model="formValues[varKey]"
                :placeholder="varDef.default || `Enter ${varDef.label}`"
              />
            </div>

            <!-- Global Variables -->
            <div class="global-vars-section">
              <div class="section-title">Output Options</div>
              <div class="global-vars-grid">
                <div class="form-field">
                  <label>Output Format</label>
                  <el-select v-model="formValues['output_format']" style="width: 100%">
                    <el-option label="Markdown" value="markdown" />
                    <el-option label="Plain Text" value="plain text" />
                    <el-option label="Bullet Points" value="bullet points" />
                    <el-option label="Table" value="table" />
                  </el-select>
                </div>
                <div class="form-field">
                  <label>Detail Level</label>
                  <el-select v-model="formValues['detail_level']" style="width: 100%">
                    <el-option label="Brief" value="brief" />
                    <el-option label="Moderate" value="moderate" />
                    <el-option label="Detailed" value="detailed" />
                    <el-option label="Comprehensive" value="comprehensive" />
                  </el-select>
                </div>
              </div>
            </div>

            <!-- Insert Target -->
            <div class="insert-target-section">
              <div class="section-title">Insert To</div>
              <el-radio-group v-model="insertTarget" size="small">
                <el-radio-button value="user">
                  <el-icon><Edit /></el-icon> User Prompt
                </el-radio-button>
                <el-radio-button value="system">
                  <el-icon><Setting /></el-icon> System Prompt
                </el-radio-button>
              </el-radio-group>
              <div class="insert-target-hint">
                {{ insertTarget === 'user' ? 'Visible in chat, one-time use' : 'Hidden, applies to entire conversation' }}
              </div>
            </div>
          </div>

          <div class="form-actions">
            <el-button @click="previewPrompt" :disabled="!isFormValid">
              <el-icon><View /></el-icon> Preview
            </el-button>
            <el-button type="primary" @click="insertPrompt" :disabled="!isFormValid">
              <el-icon><Check /></el-icon> Insert Prompt
            </el-button>
          </div>

          <!-- Preview Panel -->
          <div v-if="previewText" class="preview-panel">
            <div class="preview-header">Preview</div>
            <div class="preview-content">{{ previewText }}</div>
          </div>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Document, Search, ArrowLeft, View, Check, Edit, Setting } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

const props = defineProps<{
  compact?: boolean
}>()

const emit = defineEmits<{
  (e: 'insert', prompt: string, target: 'user' | 'system'): void
}>()

const showPopover = ref(false)
const searchQuery = ref('')
const complexityFilter = ref('')
const selectedCategory = ref('')
const selectedTemplate = ref<any>(null)
const formValues = ref<Record<string, string>>({})
const previewText = ref('')
const insertTarget = ref<'user' | 'system'>('user')

const templates = ref<any>({
  categories: []
})

// Load templates on mount
onMounted(async () => {
  try {
    const res = await request.get('/prompt-templates')
    templates.value = res as any
    if (templates.value.categories?.length > 0) {
      selectedCategory.value = templates.value.categories[0].id
    }
  } catch (error) {
    console.error('Failed to load templates:', error)
  }
})

// Filter categories based on search
const filteredCategories = computed(() => {
  if (!searchQuery.value) return templates.value.categories || []
  
  const query = searchQuery.value.toLowerCase()
  return (templates.value.categories || []).filter((cat: any) => {
    // Check if category name matches
    if (cat.name.toLowerCase().includes(query)) return true
    
    // Check if any topic or template matches
    return cat.topics?.some((topic: any) => {
      if (topic.name.toLowerCase().includes(query)) return true
      return topic.templates?.some((tmpl: any) => 
        tmpl.name.toLowerCase().includes(query) ||
        tmpl.description.toLowerCase().includes(query)
      )
    })
  })
})

// Get current category's topics
const currentTopics = computed(() => {
  const cat = (templates.value.categories || []).find((c: any) => c.id === selectedCategory.value)
  return cat?.topics || []
})

// Filter templates by search and complexity
const getFilteredTemplates = (templateList: any[]) => {
  return templateList.filter(tmpl => {
    if (complexityFilter.value && tmpl.complexity !== complexityFilter.value) {
      return false
    }
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      return tmpl.name.toLowerCase().includes(query) ||
             tmpl.description.toLowerCase().includes(query)
    }
    return true
  })
}

// Check if form has all required fields
const isFormValid = computed(() => {
  if (!selectedTemplate.value) return false
  
  const customVars = selectedTemplate.value.customVariables || {}
  for (const [key, varDef] of Object.entries(customVars) as [string, any][]) {
    if (varDef.required && !formValues.value[key]) {
      return false
    }
  }
  return true
})

// Select a category
const selectCategory = (catId: string) => {
  selectedCategory.value = catId
}

// Select a template and initialize form
const selectTemplate = (tmpl: any) => {
  selectedTemplate.value = tmpl
  formValues.value = {}
  previewText.value = ''
  
  // Initialize default values
  const customVars = tmpl.customVariables || {}
  for (const [key, varDef] of Object.entries(customVars) as [string, any][]) {
    if (varDef.default) {
      formValues.value[key] = varDef.default
    }
  }
  
  // Initialize global vars
  formValues.value['output_format'] = 'markdown'
  formValues.value['detail_level'] = 'moderate'
}

// Get tag type for complexity
const getComplexityType = (complexity: string) => {
  switch (complexity) {
    case 'simple': return 'success'
    case 'medium': return 'warning'
    case 'complex': return 'danger'
    default: return 'info'
  }
}

// Generate preview
const previewPrompt = () => {
  if (!selectedTemplate.value) return
  
  let rendered = selectedTemplate.value.template
  for (const [key, value] of Object.entries(formValues.value)) {
    rendered = rendered.replace(new RegExp(`\\{${key}\\}`, 'g'), value || `{${key}}`)
  }
  previewText.value = rendered
}

// Insert the prompt
const insertPrompt = () => {
  if (!selectedTemplate.value) return
  
  let rendered = selectedTemplate.value.template
  for (const [key, value] of Object.entries(formValues.value)) {
    rendered = rendered.replace(new RegExp(`\\{${key}\\}`, 'g'), value || '')
  }
  
  const target = insertTarget.value
  emit('insert', rendered, target)
  showPopover.value = false
  
  // Show message before reset
  ElMessage.success(target === 'system' ? 'System prompt set!' : 'Prompt inserted!')
  
  // Reset state
  selectedTemplate.value = null
  formValues.value = {}
  previewText.value = ''
  insertTarget.value = 'user'
}

// Reset when closing popover
watch(showPopover, (visible) => {
  if (!visible) {
    selectedTemplate.value = null
    formValues.value = {}
    previewText.value = ''
  }
})
</script>

<style scoped>
.template-trigger-btn {
  display: flex;
  align-items: center;
  gap: 4px;
}

.template-popover-content {
  max-height: 500px;
  overflow-y: auto;
}

.popover-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #eee;
}

.category-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.category-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  background: #f5f5f5;
  transition: all 0.2s;
}

.category-tab:hover {
  background: #e8e8e8;
}

.category-tab.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.category-icon {
  font-size: 18px;
}

.category-name {
  font-size: 13px;
  font-weight: 500;
}

.topics-section {
  max-height: 350px;
  overflow-y: auto;
}

.topic-group {
  margin-bottom: 16px;
}

.topic-header {
  font-weight: 600;
  font-size: 13px;
  color: #666;
  margin-bottom: 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid #eee;
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.template-item {
  padding: 10px 12px;
  border-radius: 8px;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.2s;
}

.template-item:hover {
  background: #f0f0f0;
  transform: translateX(4px);
}

.template-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.template-name {
  font-weight: 500;
  font-size: 14px;
}

.template-desc {
  font-size: 12px;
  color: #888;
}

/* Form Section */
.template-form-section {
  min-height: 300px;
}

.form-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #eee;
}

.form-title {
  font-weight: 600;
  font-size: 16px;
  flex: 1;
}

.form-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 280px;
  overflow-y: auto;
  padding-right: 8px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-field label {
  font-size: 13px;
  font-weight: 500;
  color: #666;
}

.form-field label.required::after {
  content: ' *';
  color: #f56c6c;
}

.global-vars-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #eee;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: #666;
  margin-bottom: 8px;
}

.global-vars-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #eee;
}

.preview-panel {
  margin-top: 12px;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 8px;
  border: 1px solid #eee;
}

.preview-header {
  font-size: 12px;
  font-weight: 600;
  color: #888;
  margin-bottom: 8px;
}

.preview-content {
  font-size: 13px;
  white-space: pre-wrap;
  max-height: 150px;
  overflow-y: auto;
  line-height: 1.5;
}

.insert-target-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #eee;
}

.insert-target-hint {
  font-size: 11px;
  color: #999;
  margin-top: 6px;
  font-style: italic;
}
</style>
