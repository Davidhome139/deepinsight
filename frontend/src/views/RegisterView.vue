<template>
  <div class="auth-container">
    <el-card class="auth-card">
      <h2>Register</h2>
      <el-form :model="form" @submit.prevent="handleRegister">
        <el-form-item label="Username">
          <el-input v-model="form.username" required />
        </el-form-item>
        <el-form-item label="Email">
          <el-input v-model="form.email" type="email" required />
        </el-form-item>
        <el-form-item label="Password">
          <el-input v-model="form.password" type="password" required />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" block>Register</el-button>
        <div class="auth-footer">
          Already have an account? <router-link to="/login">Login</router-link>
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
    ElMessage.success('Registration successful, please login')
    router.push('/login')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Registration failed')
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
