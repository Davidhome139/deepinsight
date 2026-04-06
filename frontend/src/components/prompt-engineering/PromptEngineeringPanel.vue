<template>
  <div class="prompt-engineering-panel" @click.stop>
    <div class="panel-header">
      <h3>{{ t('promptEngineering.title') }}</h3>
      <el-button type="text" @click="$emit('close')">
        <el-icon><Close /></el-icon>
      </el-button>
    </div>
    
    <div class="panel-content">
      <!-- 启用/禁用提示词工程 -->
      <div class="setting-item">
        <div class="setting-label">{{ t('promptEngineering.enabled') }}</div>
        <el-switch v-model="config.enabled" @click.stop />
      </div>
      
      <el-divider v-if="config.enabled" />
      
      <!-- 角色选择 -->
      <div v-if="config.enabled" class="setting-section">
        <h4>{{ t('promptEngineering.roleSelection') }}</h4>
        <RoleSelector 
          :selected-role="config.role"
          :custom-role-info="config.customRoleInfo"
          @role-changed="handleRoleChanged"
          @custom-role-info-changed="handleCustomRoleInfoChanged"
        />
      </div>
      
      <!-- 输出格式设置 -->
      <div v-if="config.enabled" class="setting-section">
        <h4>{{ t('promptEngineering.outputFormat') }}</h4>
        <OutputFormatSelector 
          :selected-format="config.outputFormat"
          :schema="config.schema"
          @format-changed="handleFormatChanged"
          @schema-changed="handleSchemaChanged"
        />
      </div>
      
      <!-- 思维链设置 -->
      <div v-if="config.enabled" class="setting-section">
        <h4>{{ t('promptEngineering.chainOfThought') }}</h4>
        <div class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.enableChainOfThought') }}</div>
          <el-switch v-model="config.chainOfThought" @click.stop />
        </div>
        <div v-if="config.chainOfThought" class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.maxChains') }}</div>
          <el-input-number v-model="config.maxChains" :min="1" :max="5" :step="1" @click.stop />
        </div>
      </div>
      
      <!-- 工具调用设置 -->
      <div v-if="config.enabled" class="setting-section">
        <h4>{{ t('promptEngineering.toolCalls') }}</h4>
        <div class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.enableToolCalls') }}</div>
          <el-switch v-model="config.toolCalls" @click.stop />
        </div>
      </div>
      
      <!-- 自我评估设置 -->
      <div v-if="config.enabled" class="setting-section">
        <h4>{{ t('promptEngineering.selfEvaluation') }}</h4>
        <div class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.enableSelfEvaluation') }}</div>
          <el-switch v-model="config.selfEvaluation" @click.stop />
        </div>
      </div>
      
      <!-- 智能提示词优化设置 -->
      <div v-if="config.enabled" class="setting-section">
        <h4>{{ t('promptEngineering.promptOptimization') }}</h4>
        <div class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.enablePromptOptimization') }}</div>
          <el-switch v-model="config.promptOptimizationEnabled" @click.stop />
        </div>
        <div v-if="config.promptOptimizationEnabled" class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.promptOptimizationLevel') }}</div>
          <el-select 
            v-model="config.promptOptimizationLevel" 
            style="width: 150px;"
            :append-to-body="false"
            :popper-append-to-body="false"
            @click.stop
          >
            <el-option label="低" value="low" />
            <el-option label="中" value="medium" />
            <el-option label="高" value="high" />
          </el-select>
        </div>
        <div v-if="config.promptOptimizationEnabled" class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.enableContextEnhancement') }}</div>
          <el-switch v-model="config.enableContextEnhancement" @click.stop />
        </div>
        <div v-if="config.promptOptimizationEnabled" class="setting-item">
          <div class="setting-label">{{ t('promptEngineering.enableIntentUnderstanding') }}</div>
          <el-switch v-model="config.enableIntentUnderstanding" @click.stop />
        </div>
      </div>
    </div>
    
    <div class="panel-footer">
      <el-button @click="$emit('close')" @click.stop>{{ t('common.cancel') }}</el-button>
      <el-button type="primary" @click="saveConfig" @click.stop>{{ t('common.save') }}</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Close } from '@element-plus/icons-vue'
import RoleSelector from './RoleSelector.vue'
import OutputFormatSelector from './OutputFormatSelector.vue'

const { t } = useI18n()

const props = defineProps<{
  config: {
    enabled: boolean
    role: string
    customRoleInfo: string
    outputFormat: string
    schema: string
    chainOfThought: boolean
    maxChains: number
    toolCalls: boolean
    selfEvaluation: boolean
    promptOptimizationEnabled: boolean
    promptOptimizationLevel: string
    enableContextEnhancement: boolean
    enableIntentUnderstanding: boolean
  }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', config: typeof props.config): void
}>()

const config = reactive({ ...props.config })

// Watch for changes in props
watch(() => props.config, (newConfig) => {
  Object.assign(config, newConfig)
}, { deep: true })

const handleRoleChanged = (role: string) => {
  config.role = role
}

const handleCustomRoleInfoChanged = (info: string) => {
  config.customRoleInfo = info
}

const handleFormatChanged = (format: string) => {
  config.outputFormat = format
}

const handleSchemaChanged = (schema: string) => {
  config.schema = schema
}

const saveConfig = () => {
  emit('save', { ...config })
  emit('close')
}
</script>

<style scoped>
.prompt-engineering-panel {
  width: 400px;
  max-height: 500px;
  overflow-y: auto;
  padding: 20px;
  background-color: var(--bg-primary);
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-primary);
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.panel-content {
  margin-bottom: 20px;
}

.setting-section {
  margin-bottom: 20px;
}

.setting-section h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.setting-label {
  font-size: 13px;
  color: var(--text-primary);
}

.panel-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-primary);
}
</style>