<template>
  <div class="chain-of-thought-display">
    <div class="chain-header">
      <h4>{{ t('promptEngineering.chainOfThought') }}</h4>
      <el-button type="text" @click="$emit('close')">
        <el-icon><Close /></el-icon>
      </el-button>
    </div>
    
    <div class="chain-content">
      <div v-if="chains.length === 0" class="empty-state">
        {{ t('promptEngineering.noChains') }}
      </div>
      <div v-else class="chains-list">
        <div 
          v-for="(chain, index) in chains" 
          :key="chain.id"
          class="chain-item"
          :class="{ active: selectedChainId === chain.id }"
          @click="selectChain(chain.id)"
        >
          <div class="chain-header-row">
            <div class="chain-index">
              {{ index + 1 }}
            </div>
            <div class="chain-score">
              {{ t('promptEngineering.score') }}: {{ chain.score.toFixed(2) }}
            </div>
          </div>
          <div class="chain-content-text">
            {{ chain.content }}
          </div>
        </div>
      </div>
    </div>
    
    <div class="chain-footer">
      <el-button @click="$emit('close')">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" @click="confirmSelection" :disabled="!selectedChainId">
        {{ t('common.confirm') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Close } from '@element-plus/icons-vue'

const { t } = useI18n()

const props = defineProps<{
  chains: Array<{
    id: string
    content: string
    score: number
  }>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select', chainId: string): void
}>()

const selectedChainId = ref(props.chains.length > 0 ? props.chains[0].id : '')

const selectChain = (chainId: string) => {
  selectedChainId.value = chainId
}

const confirmSelection = () => {
  if (selectedChainId.value) {
    emit('select', selectedChainId.value)
    emit('close')
  }
}
</script>

<style scoped>
.chain-of-thought-display {
  width: 500px;
  padding: 20px;
  background-color: var(--bg-primary);
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.chain-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-primary);
}

.chain-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.chain-content {
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 20px;
}

.empty-state {
  text-align: center;
  color: var(--text-tertiary);
  padding: 40px 0;
}

.chains-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.chain-item {
  padding: 15px;
  border: 1px solid var(--border-primary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.chain-item:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.chain-item.active {
  border-color: var(--color-primary);
  background-color: rgba(64, 158, 255, 0.05);
}

.chain-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.chain-index {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
  background-color: rgba(64, 158, 255, 0.1);
  padding: 2px 8px;
  border-radius: 10px;
}

.chain-score {
  font-size: 12px;
  color: var(--text-secondary);
}

.chain-content-text {
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-primary);
  white-space: pre-wrap;
}

.chain-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-primary);
}
</style>