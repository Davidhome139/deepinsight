<template>
  <div class="role-selector" @click.stop>
    <el-select 
      v-model="localRole" 
      :placeholder="t('promptEngineering.selectRole')" 
      style="width: 100%; margin-bottom: 10px;" 
      :append-to-body="false"
      :popper-append-to-body="false"
      @click.stop
    >
      <el-option :label="t('promptEngineering.defaultRole')" value="default" />
      <el-option :label="t('promptEngineering.doctor')" value="doctor" />
      <el-option :label="t('promptEngineering.engineer')" value="engineer" />
      <el-option :label="t('promptEngineering.lawyer')" value="lawyer" />
      <el-option :label="t('promptEngineering.teacher')" value="teacher" />
      <el-option :label="t('promptEngineering.writer')" value="writer" />
      <el-option :label="t('promptEngineering.customRole')" value="custom" />
    </el-select>
    
    <div v-if="localRole === 'custom'" class="custom-role-section">
      <el-input
        v-model="localCustomRoleInfo"
        type="textarea"
        :rows="3"
        placeholder="{{ t('promptEngineering.enterCustomRoleInfo') }}"
        style="width: 100%;"
        @input="handleCustomRoleInfoChanged"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  selectedRole: string
  customRoleInfo: string
}>()

const emit = defineEmits<{
  (e: 'role-changed', role: string): void
  (e: 'custom-role-info-changed', info: string): void
}>()

const localRole = ref(props.selectedRole)
const localCustomRoleInfo = ref(props.customRoleInfo)

watch(localRole, (newRole) => {
  emit('role-changed', newRole)
})

const handleCustomRoleInfoChanged = () => {
  emit('custom-role-info-changed', localCustomRoleInfo.value)
}
</script>

<style scoped>
.role-selector {
  width: 100%;
}

.custom-role-section {
  margin-top: 10px;
}
</style>