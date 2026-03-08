<template>
  <div class="auth-container">
    <el-card class="auth-card">
      <h2>Login</h2>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item label="Email">
          <el-input v-model="form.email" type="email" required />
        </el-form-item>
        <el-form-item label="Password">
          <el-input v-model="form.password" type="password" required />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" block>Login</el-button>
        <div class="auth-footer">
          Don't have an account? <router-link to="/register">Register</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

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
    ElMessage.success('Login successful')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Login failed')
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
