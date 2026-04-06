<template>
  <div class="auth-container">
    <el-card class="auth-card">
      <h2>{{ t('auth.register') }}</h2>
      <el-form :model="form" @submit.prevent="handleRegister">
        <el-form-item :label="t('auth.username')">
          <el-input v-model="form.username" required />
        </el-form-item>
        <el-form-item :label="t('auth.email')">
          <el-input v-model="form.email" type="email" required />
        </el-form-item>
        <el-form-item :label="t('auth.password')">
          <el-input v-model="form.password" type="password" required />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" block>{{ t('auth.register') }}</el-button>
        <div class="auth-footer">
          {{ t('auth.alreadyHaveAccount') }} <router-link to="/login">{{ t('auth.login') }}</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { authApi } from '../api/auth'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const form = reactive({
  username: '',
  email: '',
  password: '',
})

const handleRegister = async () => {
  loading.value = true
  try {
    await authApi.register(form)
    ElMessage.success(t('auth.registrationSuccessful'))
    router.push('/login')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || t('auth.registrationFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
}
.auth-card {
  width: 400px;
}
.auth-footer {
  margin-top: 15px;
  text-align: center;
}
</style>
