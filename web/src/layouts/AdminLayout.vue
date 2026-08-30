<template>
  <el-container class="admin-layout">
    <el-aside width="210px" class="aside">
      <div class="brand">Gitsune</div>
      <el-menu :default-active="route.path" router background-color="#001529" text-color="#bfcbd9" active-text-color="#409eff">
        <el-menu-item index="/repos">
          <el-icon><Folder /></el-icon>
          <span>Repositories</span>
        </el-menu-item>
        <el-menu-item v-if="isAdmin" index="/task-logs">
          <el-icon><Timer /></el-icon>
          <span>Tasks</span>
        </el-menu-item>
        <el-menu-item v-if="isAdmin" index="/users">
          <el-icon><User /></el-icon>
          <span>Users</span>
        </el-menu-item>
        <el-menu-item v-if="isAdmin" index="/settings">
          <el-icon><Setting /></el-icon>
          <span>Settings</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span class="header-title">Git Repository Collection</span>
        <div class="header-right">
          <span class="username">{{ user?.username }}</span>
          <el-button link type="primary" @click="pwdDialogVisible = true">Change Password</el-button>
          <el-button link type="danger" @click="onLogout">Log Out</el-button>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>

    <el-dialog v-model="pwdDialogVisible" title="Change Password" width="400px">
      <el-form :model="pwdForm" label-width="110px">
        <el-form-item label="Old Password">
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="New Password">
          <el-input v-model="pwdForm.new_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="pwdLoading" @click="onChangePassword">Confirm</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { logout, changePassword, TOKEN_KEY } from '../api'
import { useUser } from '../stores/user'

const route = useRoute()
const router = useRouter()
const { user, isAdmin, clearUser } = useUser()

const pwdDialogVisible = ref(false)
const pwdLoading = ref(false)
const pwdForm = reactive({ old_password: '', new_password: '' })

async function onChangePassword() {
  if (!pwdForm.old_password || !pwdForm.new_password) {
    ElMessage.warning('Please enter both old and new password')
    return
  }
  pwdLoading.value = true
  try {
    await changePassword(pwdForm.old_password, pwdForm.new_password)
    ElMessage.success('Password changed')
    pwdDialogVisible.value = false
    pwdForm.old_password = ''
    pwdForm.new_password = ''
  } catch {
    // interceptor already shows the error
  } finally {
    pwdLoading.value = false
  }
}

async function onLogout() {
  try {
    await ElMessageBox.confirm('Log out now?', 'Confirm', { type: 'warning' })
  } catch {
    return
  }
  try {
    await logout()
  } catch {
    // ignore logout API errors
  }
  localStorage.removeItem(TOKEN_KEY)
  clearUser()
  router.replace('/login')
}
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
}
.aside {
  background: #001529;
}
.brand {
  color: #fff;
  font-size: 20px;
  font-weight: 600;
  text-align: center;
  line-height: 60px;
}
.aside :deep(.el-menu) {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;
}
.header-title {
  font-size: 16px;
  font-weight: 500;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.username {
  color: #606266;
}
.main {
  background: #f5f7fa;
}
</style>
