<template>
  <transition name="slide-down">
    <div 
      v-if="visible" 
      class="form-feedback"
      :class="type"
    >
      <el-icon class="feedback-icon">
        <SuccessFilled v-if="type === 'success'" />
        <WarningFilled v-else-if="type === 'warning'" />
        <CircleCloseFilled v-else-if="type === 'error'" />
        <InfoFilled v-else />
      </el-icon>
      <span class="feedback-message">{{ message }}</span>
      <el-button 
        v-if="closable" 
        :icon="Close" 
        size="small" 
        text 
        @click="$emit('close')"
        class="close-btn"
      />
    </div>
  </transition>
</template>

<script setup lang="ts">
import { SuccessFilled, WarningFilled, CircleCloseFilled, InfoFilled, Close } from '@element-plus/icons-vue'

defineProps<{
  visible?: boolean
  type?: 'success' | 'warning' | 'error' | 'info'
  message: string
  closable?: boolean
}>()

defineEmits(['close'])
</script>

<style scoped>
.form-feedback {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 8px;
  margin: 12px 0;
  font-size: 14px;
}

.form-feedback.success {
  background: rgba(103, 194, 58, 0.1);
  color: #67c23a;
  border: 1px solid rgba(103, 194, 58, 0.3);
}

.form-feedback.warning {
  background: rgba(230, 162, 60, 0.1);
  color: #e6a23c;
  border: 1px solid rgba(230, 162, 60, 0.3);
}

.form-feedback.error {
  background: rgba(245, 108, 108, 0.1);
  color: #f56c6c;
  border: 1px solid rgba(245, 108, 108, 0.3);
}

.form-feedback.info {
  background: rgba(64, 158, 255, 0.1);
  color: #409eff;
  border: 1px solid rgba(64, 158, 255, 0.3);
}

.feedback-icon {
  font-size: 18px;
}

.feedback-message {
  flex: 1;
}

.close-btn {
  opacity: 0.6;
}

.close-btn:hover {
  opacity: 1;
}
</style>
