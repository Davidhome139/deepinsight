<template>
  <nav class="breadcrumb" aria-label="Breadcrumb">
    <ol class="breadcrumb-list">
      <li class="breadcrumb-item">
        <router-link to="/" class="breadcrumb-link home">
          <el-icon><HomeFilled /></el-icon>
          <span v-if="!compact">Home</span>
        </router-link>
      </li>
      <li 
        v-for="(crumb, index) in crumbs" 
        :key="crumb.path"
        class="breadcrumb-item"
      >
        <el-icon class="separator"><ArrowRight /></el-icon>
        <router-link 
          v-if="index < crumbs.length - 1" 
          :to="crumb.path" 
          class="breadcrumb-link"
        >
          <el-icon v-if="crumb.icon"><component :is="crumb.icon" /></el-icon>
          <span>{{ crumb.name }}</span>
        </router-link>
        <span v-else class="breadcrumb-current">
          <el-icon v-if="crumb.icon"><component :is="crumb.icon" /></el-icon>
          <span>{{ crumb.name }}</span>
        </span>
      </li>
    </ol>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { HomeFilled, ArrowRight, ChatLineRound, Monitor, VideoCamera, Picture, Setting, DataAnalysis, User, FolderOpened, Document } from '@element-plus/icons-vue'

defineProps<{
  compact?: boolean
}>()

const route = useRoute()

interface Breadcrumb {
  name: string
  path: string
  icon?: any
}

const routeConfig: Record<string, { name: string, icon?: any }> = {
  '/': { name: 'Chat', icon: ChatLineRound },
  '/chat': { name: 'Chat', icon: ChatLineRound },
  '/programming': { name: 'AI Programming', icon: Monitor },
  '/video': { name: 'Video Generation', icon: VideoCamera },
  '/image': { name: 'Image Generation', icon: Picture },
  '/ai-chat': { name: 'AI-AI Chat', icon: ChatLineRound },
  '/settings': { name: 'Settings', icon: Setting },
  '/analytics': { name: 'Analytics', icon: DataAnalysis },
  '/agents': { name: 'Agent Studio', icon: User },
  '/rag': { name: 'Knowledge Base', icon: FolderOpened },
}

const crumbs = computed<Breadcrumb[]>(() => {
  const path = route.path
  const config = routeConfig[path]
  
  if (!config) {
    // Try to match partial paths
    const segments = path.split('/').filter(Boolean)
    return segments.map((seg, i) => ({
      name: seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, ' '),
      path: '/' + segments.slice(0, i + 1).join('/'),
      icon: Document
    }))
  }
  
  return [{
    name: config.name,
    path: path,
    icon: config.icon
  }]
})
</script>

<style scoped>
.breadcrumb {
  padding: 12px 16px;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-primary);
}

.breadcrumb-list {
  display: flex;
  align-items: center;
  list-style: none;
  margin: 0;
  padding: 0;
  gap: 4px;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.breadcrumb-link {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 14px;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.breadcrumb-link:hover {
  color: var(--accent-primary);
  background: var(--bg-hover);
}

.breadcrumb-link.home {
  color: var(--text-primary);
}

.breadcrumb-current {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 500;
  padding: 4px 8px;
}

.separator {
  color: var(--text-tertiary);
  font-size: 12px;
}
</style>
