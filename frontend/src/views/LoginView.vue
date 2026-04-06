<template>
  <div class="auth-container">
    <el-card class="auth-card">
      <h2>{{ t('auth.login') }}</h2>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item :label="t('auth.email')">
          <el-input v-model="form.email" type="email" required />
        </el-form-item>
        <el-form-item :label="t('auth.password')">
          <el-input v-model="form.password" type="password" required />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" block>{{ t('auth.login') }}</el-button>
        <div class="auth-footer">
          {{ t('auth.dontHaveAccount') }} <router-link to="/register">{{ t('auth.register') }}</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const authStore = useAuthStore()
const loading = ref(false)
const form = reactive({
  email: '',
  password: '',
})

const handleLogin = async () => {
  loading.value = true
  try {
    await authStore.login(form)
    ElMessage.success(t('auth.loginSuccessful'))
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || t('auth.loginFailed'))
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
