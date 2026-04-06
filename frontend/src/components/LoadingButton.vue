<template>
  <el-button 
    :type="type" 
    :size="size"
    :disabled="disabled || loading"
    :loading="loading"
    :icon="loading ? Loading : icon"
    class="loading-button"
    :class="{ 'is-loading': loading }"
    v-bind="$attrs"
  >
    <span v-if="loading && loadingText" class="loading-text">{{ loadingText }}</span>
    <slot v-else />
  </el-button>
</template>

<script setup lang="ts">
import { Loading } from '@element-plus/icons-vue'
import type { Component } from 'vue'

defineProps<{
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'default'
  size?: 'large' | 'default' | 'small'
  loading?: boolean
  loadingText?: string
  disabled?: boolean
  icon?: Component
}>()
</script>

<style scoped>
.loading-button {
  transition: all 0.3s ease;
  position: relative;
}

.loading-button.is-loading {
  pointer-events: none;
}

.loading-text {
  margin-left: 4px;
}
</style>
