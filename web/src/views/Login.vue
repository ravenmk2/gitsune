<template>
  <div class="login-page">
    <el-card class="login-card">
      <div class="login-title">Gitsune</div>
      <el-form :model="form" @submit.prevent>
        <el-form-item>
          <el-input v-model="form.username" placeholder="Username" :prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="form.password"
            type="password"
            placeholder="Password"
            show-password
            :prefix-icon="Lock"
            @keyup.enter="onSubmit"
          />
        </el-form-item>
        <el-button type="primary" class="login-btn" :loading="loading" @click="onSubmit">Log In</el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { login, TOKEN_KEY } from '../api'
import { useUser } from '../stores/user'

const router = useRouter()
const { setUser } = useUser()

const form = reactive({ username: '', password: '' })
const loading = ref(false)

async function onSubmit() {
  if (!form.username || !form.password) {
    ElMessage.warning('Please enter username and password')
    return
  }
  loading.value = true
  try {
    const data = await login(form.username, form.password)
    localStorage.setItem(TOKEN_KEY, data.token)
    setUser(data.user)
    ElMessage.success('Login successful')
    router.replace('/')
  } catch {
    // interceptor already shows the error
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}
.login-card {
  width: 360px;
}
.login-title {
  font-size: 24px;
  font-weight: 600;
  text-align: center;
  margin-bottom: 24px;
}
.login-btn {
  width: 100%;
}
</style>
