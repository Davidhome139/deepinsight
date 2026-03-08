<template>
  <el-form :model="formData" label-width="120px" class="config-form">
    <!-- 通用字段 -->
    <el-form-item label="名称">
      <el-input v-model="formData.name" disabled />
    </el-form-item>

    <el-form-item label="启用状态">
      <el-switch v-model="formData.enabled" />
    </el-form-item>

    <!-- AI 模型和搜索的 API Key -->
    <template v-if="type === 'models' || type === 'searchs'">
      <el-form-item label="API Key">
        <el-input
          v-model="formData.api_key"
          type="password"
          show-password
          placeholder="输入 API Key"
        />
      </el-form-item>

      <el-form-item label="Secret Key" v-if="type === 'models'">
        <el-input
          v-model="formData.secret_key"
          type="password"
          show-password
          placeholder="输入 Secret Key（可选）"
        />
      </el-form-item>

      <el-form-item label="Secret ID" v-if="type === 'models'">
        <el-input
          v-model="formData.secret_id"
          placeholder="输入 Secret ID（可选）"
        />
      </el-form-item>

      <el-form-item label="Base URL">
        <el-input
          v-model="formData.base_url"
          placeholder="输入 API 基础 URL"
        />
      </el-form-item>
    </template>

    <!-- MCP 服务器配置 -->
    <template v-if="type === 'mcpservers'">
      <el-form-item label="服务器类型">
        <el-select v-model="formData.server_type" placeholder="选择类型">
          <el-option label="内置" value="builtin" />
          <el-option label="外部命令" value="command" />
          <el-option label="SSE" value="sse" />
        </el-select>
      </el-form-item>

      <el-form-item label="命令" v-if="formData.server_type === 'command'">
        <el-input v-model="formData.command" placeholder="输入命令" />
      </el-form-item>

      <el-form-item label="参数" v-if="formData.server_type === 'command'">
        <el-input
          v-model="formData.argsText"
          type="textarea"
          :rows="2"
          placeholder="每行一个参数"
        />
      </el-form-item>
    </template>

    <!-- 技能配置 -->
    <template v-if="type === 'skills'">
      <el-form-item label="描述">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="技能描述"
        />
      </el-form-item>

      <el-form-item label="分类">
        <el-select v-model="formData.category" placeholder="选择分类">
          <el-option label="开发" value="development" />
          <el-option label="分析" value="analysis" />
          <el-option label="工具" value="tool" />
        </el-select>
      </el-form-item>
    </template>

    <!-- 智能体配置 -->
    <template v-if="type === 'agents'">
      <el-form-item label="角色">
        <el-input v-model="formData.role" placeholder="角色标识" />
      </el-form-item>

      <el-form-item label="使用模型">
        <el-select v-model="formData.model" placeholder="选择模型">
          <el-option label="阿里云 (Qwen)" value="aliyun" />
          <el-option label="DeepSeek" value="deepseek" />
          <el-option label="腾讯混元" value="tencent" />
          <el-option label="OpenAI" value="openai" />
        </el-select>
      </el-form-item>

      <el-form-item label="系统提示词">
        <el-input
          v-model="formData.system_prompt"
          type="textarea"
          :rows="6"
          placeholder="输入系统提示词"
        />
      </el-form-item>
    </template>

    <!-- 操作按钮 -->
    <el-form-item>
      <el-button type="primary" @click="handleSave" :loading="saving">
        保存
      </el-button>
      <el-button @click="$emit('cancel')">取消</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

interface Props {
  type: string
  data: Record<string, any>
}

const props = defineProps<Props>()
const emit = defineEmits<{
  save: [data: Record<string, any>]
  cancel: []
}>()

const formData = ref<Record<string, any>>({})
const saving = ref(false)

// 初始化表单数据
watch(() => props.data, (newData) => {
  formData.value = { ...newData }

  // 处理数组转文本
  if (formData.value.args && Array.isArray(formData.value.args)) {
    formData.value.argsText = formData.value.args.join('\n')
  }
}, { immediate: true, deep: true })

const handleSave = async () => {
  saving.value = true
  try {
    // 处理文本转数组
    const saveData = { ...formData.value }
    if (saveData.argsText) {
      saveData.args = saveData.argsText.split('\n').filter((s: string) => s.trim())
      delete saveData.argsText
    }

    emit('save', saveData)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.config-form {
  max-width: 600px;
}
</style>
