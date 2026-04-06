<template>
  <el-form :model="form" label-width="120px" v-loading="loading">
    <el-form-item label="Enable">
      <el-switch v-model="form.enabled" />
    </el-form-item>
    <el-form-item label="API Key">
      <el-input v-model="form.api_key" type="password" show-password placeholder="Enter your API Key" />
    </el-form-item>
    <el-form-item v-if="provider === 'tencent'" label="Secret Id">
      <el-input v-model="form.secret_id" placeholder="Enter your Secret Id" />
    </el-form-item>
    <el-form-item v-if="provider === 'tencent'" label="Secret Key">
      <el-input v-model="form.secret_key" type="password" show-password placeholder="Enter your Secret Key" />
    </el-form-item>
    <el-form-item label="Base URL">
      <el-input v-model="form.base_url" :placeholder="defaultBaseURL" />
      <div class="tip">Leave empty to use system default</div>
    </el-form-item>
    <el-form-item>
      <el-button type="primary" @click="handleSave">Save Settings</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  provider: string
  type?: 'ai' | 'search' | 'video'
}>()

const form = ref({
  provider: props.provider,
  api_key: '',
  secret_key: '',
  secret_id: '',
  base_url: '',
  enabled: true,
  type: props.type || 'ai'
})

const loading = ref(false)

const defaultBaseURL = computed(() => {
  if (props.provider === 'aliyun') return 'https://dashscope.aliyuncs.com/compatible-mode/v1'
  if (props.provider === 'deepseek') return 'https://api.deepseek.com/v1'
  if (props.provider === 'tencent') return 'https://hunyuan.tencentcloudapi.com'
  if (props.provider === 'openai') return 'https://api.openai.com/v1'
  if (props.provider === 'baidu-air') return 'https://qianfan.baidubce.com/video/generations'
  if (props.provider === 'baidu') return 'https://qianfan.baidubce.com/v2/ai_search/web_search'
  if (props.provider === 'serper') return 'https://google.serper.dev/search'
  if (props.provider === 'brightdata') return 'https://brd.superproxy.io:22225'
  return ''
})

onMounted(async () => {
  await fetchSettings()
})

const fetchSettings = async () => {
  loading.value = true
  try {
    const response = await fetch(`/api/v1/settings/ai-providers?type=${form.value.type}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    if (response.ok) {
      const settings = await response.json()
      const current = settings.find((s: any) => s.provider === props.provider)
      if (current) {
        form.value = { ...form.value, ...current }
      }
    }
  } catch (error) {
    console.error('Failed to fetch settings', error)
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/v1/settings/ai-providers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify(form.value)
    })
    if (response.ok) {
      ElMessage.success('Settings saved successfully')
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
</style>
