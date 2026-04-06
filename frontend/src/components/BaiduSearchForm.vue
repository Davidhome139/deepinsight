<template>
  <el-form :model="form" label-width="140px" v-loading="loading">
    <el-form-item label="Enable">
      <el-switch v-model="form.enabled" />
    </el-form-item>
    <el-form-item label="Access Token">
      <el-input v-model="form.api_key" type="password" show-password placeholder="Enter your Baidu Access Token" />
      <div class="tip">Get from: Baidu Cloud Console → Application List → Get Access Token</div>
    </el-form-item>
    
    <el-divider content-position="left">Search Endpoints Configuration</el-divider>
    
    <el-form-item label="Web Search">
      <el-input v-model="endpoints.web" placeholder="Enter Web Search Endpoint" />
    </el-form-item>
    
    <el-form-item label="Image Search">
      <el-input v-model="endpoints.image" placeholder="Enter Image Search Endpoint" />
    </el-form-item>
    
    <el-form-item label="AI Search">
      <el-input v-model="endpoints.ai" placeholder="Enter AI Search Endpoint" />
    </el-form-item>
    
    <el-form-item label="Performance Search">
      <el-input v-model="endpoints.performance" placeholder="Enter Performance Search Endpoint" />
    </el-form-item>
    
    <el-divider content-position="left">Active Search Mode</el-divider>
    
    <el-form-item label="Current Mode">
      <el-select v-model="activeMode" placeholder="Select Active Search Mode" @change="handleModeChange">
        <el-option label="Web Search" value="web" />
        <el-option label="Image Search" value="image" />
        <el-option label="AI Search" value="ai" />
        <el-option label="Performance Search" value="performance" />
      </el-select>
      <div class="tip mode-desc">{{ getModeDescription() }}</div>
    </el-form-item>
    
    <el-form-item>
      <el-button type="primary" @click="handleSave">Save All Settings</el-button>
      <el-button @click="resetToDefaults">Reset to Defaults</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'

const form = ref({
  provider: 'baidu',
  api_key: '',
  base_url: 'https://qianfan.baidubce.com/v2/ai_search/web_search',
  enabled: true,
  type: 'search'
})

const endpoints = ref({
  web: 'https://qianfan.baidubce.com/v2/ai_search/web_search',
  image: 'https://qianfan.baidubce.com/v2/tools/image_similar_info',
  ai: 'https://qianfan.baidubce.com/v2/ai_search/chat/completions',
  performance: 'https://qianfan.baidubce.com/v2/ai_search/web_summary'
})

const activeMode = ref('web')
const loading = ref(false)

// Watch for activeMode changes to update base_url
watch(activeMode, (newMode) => {
  form.value.base_url = endpoints.value[newMode as keyof typeof endpoints.value]
})

onMounted(async () => {
  await fetchSettings()
})

const fetchSettings = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/v1/settings/ai-providers?type=search', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    if (response.ok) {
      const settings = await response.json()
      const current = settings.find((s: any) => s.provider === 'baidu')
      if (current) {
        form.value = { ...form.value, ...current }
        
        // Try to restore endpoints configuration from metadata
        try {
          const metadata = JSON.parse(localStorage.getItem('baidu_endpoints') || '{}')
          if (Object.keys(metadata).length > 0) {
            endpoints.value = { ...endpoints.value, ...metadata }
          }
        } catch (e) {
          console.log('No saved endpoints metadata')
        }
        
        // Infer current mode from base_url
        const currentUrl = current.base_url
        if (currentUrl.includes('web_search')) activeMode.value = 'web'
        else if (currentUrl.includes('image_similar_info') || currentUrl.includes('image')) activeMode.value = 'image'
        else if (currentUrl.includes('chat/completions') || currentUrl.includes('ai_search')) activeMode.value = 'ai'
        else if (currentUrl.includes('web_summary') || currentUrl.includes('performance')) activeMode.value = 'performance'
      }
    }
  } catch (error) {
    console.error('Failed to fetch settings', error)
  } finally {
    loading.value = false
  }
}

const handleModeChange = () => {
  form.value.base_url = endpoints.value[activeMode.value as keyof typeof endpoints.value]
}

const getModeDescription = () => {
  const descriptions = {
    web: 'Standard web search for general queries',
    image: 'Image similarity search for finding similar images',
    ai: 'AI-powered semantic search with better understanding',
    performance: 'High-performance search engine with faster response'
  }
  return descriptions[activeMode.value as keyof typeof descriptions] || ''
}

const resetToDefaults = () => {
  endpoints.value = {
    web: 'https://qianfan.baidubce.com/v2/ai_search/web_search',
    image: 'https://qianfan.baidubce.com/v2/tools/image_similar_info',
    ai: 'https://qianfan.baidubce.com/v2/ai_search/chat/completions',
    performance: 'https://qianfan.baidubce.com/v2/ai_search/web_summary'
  }
  form.value.base_url = endpoints.value[activeMode.value as keyof typeof endpoints.value]
  ElMessage.success('Endpoints reset to defaults')
}

const handleSave = async () => {
  loading.value = true
  try {
    // Save endpoints configuration to localStorage (since backend only stores one base_url)
    localStorage.setItem('baidu_endpoints', JSON.stringify(endpoints.value))
    
    const response = await fetch('/api/v1/settings/ai-providers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify(form.value)
    })
    if (response.ok) {
      ElMessage.success('Baidu Search settings saved successfully')
    } else {
      throw new Error('Failed to save settings')
    }
  } catch (error: any) {
    ElMessage.error(error.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
.mode-desc {
  color: #409EFF;
  font-weight: 500;
}
</style>
