<template>
  <el-tooltip :content="isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'" placement="bottom">
    <el-button 
      :icon="isDark ? Sunny : Moon" 
      circle 
      size="small"
      @click="toggleTheme"
      class="theme-toggle-btn"
    />
  </el-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Sunny, Moon } from '@element-plus/icons-vue'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

const isDark = computed(() => themeStore.theme === 'dark')

const toggleTheme = () => {
  themeStore.toggleTheme()
}
</script>

<style scoped>
.theme-toggle-btn {
  transition: transform 0.3s ease, background-color 0.3s ease;
}

.theme-toggle-btn:hover {
  transform: rotate(15deg);
}
</style>
