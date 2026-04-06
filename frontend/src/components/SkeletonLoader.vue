<template>
  <div class="skeleton-loader" :style="containerStyle">
    <template v-if="type === 'text'">
      <div 
        v-for="n in lines" 
        :key="n" 
        class="skeleton skeleton-text"
        :style="{ width: n === lines ? lastLineWidth : '100%' }"
      />
    </template>
    
    <template v-else-if="type === 'avatar'">
      <div class="skeleton skeleton-avatar" :class="size" />
    </template>
    
    <template v-else-if="type === 'card'">
      <div class="skeleton-card">
        <div class="skeleton skeleton-image" />
        <div class="skeleton-card-body">
          <div class="skeleton skeleton-title" />
          <div class="skeleton skeleton-text" />
          <div class="skeleton skeleton-text" style="width: 60%" />
        </div>
      </div>
    </template>
    
    <template v-else-if="type === 'list'">
      <div v-for="n in count" :key="n" class="skeleton-list-item">
        <div class="skeleton skeleton-avatar small" />
        <div class="skeleton-list-content">
          <div class="skeleton skeleton-text" style="width: 40%" />
          <div class="skeleton skeleton-text" style="width: 70%" />
        </div>
      </div>
    </template>
    
    <template v-else>
      <div class="skeleton" :style="{ width, height, borderRadius }" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  type?: 'text' | 'avatar' | 'card' | 'list' | 'custom'
  lines?: number
  lastLineWidth?: string
  count?: number
  width?: string
  height?: string
  borderRadius?: string
  size?: 'small' | 'default' | 'large'
}>(), {
  type: 'custom',
  lines: 3,
  lastLineWidth: '60%',
  count: 3,
  width: '100%',
  height: '20px',
  borderRadius: '4px',
  size: 'default'
})

const containerStyle = computed(() => ({
  '--skeleton-lines': props.lines
}))
</script>

<style scoped>
.skeleton-loader {
  width: 100%;
}

.skeleton {
  background: linear-gradient(90deg, var(--bg-tertiary) 25%, var(--bg-secondary) 50%, var(--bg-tertiary) 75%);
  background-size: 200px 100%;
  animation: skeleton 1.5s ease-in-out infinite;
}

@keyframes skeleton {
  0% {
    background-position: -200px 0;
  }
  100% {
    background-position: calc(200px + 100%) 0;
  }
}

.skeleton-text {
  height: 16px;
  margin-bottom: 12px;
  border-radius: 4px;
}

.skeleton-text:last-child {
  margin-bottom: 0;
}

.skeleton-title {
  height: 24px;
  width: 50%;
  margin-bottom: 16px;
  border-radius: 4px;
}

.skeleton-avatar {
  border-radius: 50%;
}

.skeleton-avatar.small {
  width: 32px;
  height: 32px;
}

.skeleton-avatar.default {
  width: 48px;
  height: 48px;
}

.skeleton-avatar.large {
  width: 64px;
  height: 64px;
}

.skeleton-card {
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-primary);
}

.skeleton-image {
  width: 100%;
  height: 160px;
}

.skeleton-card-body {
  padding: 16px;
}

.skeleton-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-primary);
}

.skeleton-list-item:last-child {
  border-bottom: none;
}

.skeleton-list-content {
  flex: 1;
}
</style>
