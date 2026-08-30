<template>
  <div>
    <div class="filter-bar">
      <el-input v-model="query.keyword" placeholder="Search by username" clearable style="width: 200px" @keyup.enter="onSearch" />
      <el-button type="primary" :icon="Search" @click="onSearch">Search</el-button>
      <el-button type="success" :icon="Plus" @click="createVisible = true">New User</el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe>
      <el-table-column prop="username" label="Username" min-width="160" />
      <el-table-column label="Role" width="120">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'info'">
            {{ row.role === 'admin' ? 'Admin' : 'User' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Created At" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="Actions" width="320" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openReset(row)">Reset Password</el-button>
          <el-select
            :model-value="row.role"
            size="small"
            style="width: 100px; margin: 0 8px"
            :disabled="row.username === 'admin'"
            @change="(val) => onChangeRole(row, val)"
          >
            <el-option label="Admin" value="admin" />
            <el-option label="User" value="user" />
          </el-select>
          <el-tooltip :disabled="row.username !== 'admin'" content="The built-in admin cannot be deleted" placement="top">
            <span>
              <el-button link type="danger" :disabled="row.username === 'admin'" @click="onDelete(row)">Delete</el-button>
            </span>
          </el-tooltip>
        </template>
      </el-table-column>
      <template #empty>No data</template>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="size"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="loadData"
        @size-change="onSearch"
      />
    </div>

    <el-dialog v-model="createVisible" title="New User" width="420px">
      <el-form :model="createForm" label-width="90px">
        <el-form-item label="Username">
          <el-input v-model="createForm.username" />
        </el-form-item>
        <el-form-item label="Password">
          <el-input v-model="createForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="Role">
          <el-select v-model="createForm.role" style="width: 100%">
            <el-option label="Admin" value="admin" />
            <el-option label="User" value="user" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="creating" @click="onCreate">Create</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resetVisible" title="Reset Password" width="420px">
      <p style="margin-bottom: 12px">Set a new password for <b>{{ resetTarget?.username }}</b></p>
      <el-input v-model="resetForm.password" type="password" show-password placeholder="New password" />
      <template #footer>
        <el-button @click="resetVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="resetting" @click="onReset">Confirm</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { listUsers, createUser, updateUser, deleteUser, resetPassword } from '../api'

const query = reactive({ keyword: '' })
const items = ref([])
const page = ref(1)
const size = ref(10)
const total = ref(0)
const loading = ref(false)

const createVisible = ref(false)
const creating = ref(false)
const createForm = reactive({ username: '', password: '', role: 'user' })

const resetVisible = ref(false)
const resetting = ref(false)
const resetTarget = ref(null)
const resetForm = reactive({ password: '' })

function formatTime(t) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}

async function loadData() {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value }
    if (query.keyword) params.keyword = query.keyword
    const data = await listUsers(params)
    items.value = data.items || []
    total.value = data.total || 0
  } catch {
    // interceptor already shows the error
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  loadData()
}

async function onCreate() {
  if (!createForm.username || !createForm.password) {
    ElMessage.warning('Please enter username and password')
    return
  }
  creating.value = true
  try {
    await createUser(createForm.username, createForm.password, createForm.role)
    ElMessage.success('User created')
    createVisible.value = false
    createForm.username = ''
    createForm.password = ''
    createForm.role = 'user'
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    creating.value = false
  }
}

async function onChangeRole(row, role) {
  try {
    await updateUser(row.id, role)
    ElMessage.success('Role updated')
    loadData()
  } catch {
    loadData()
  }
}

function openReset(row) {
  resetTarget.value = row
  resetForm.password = ''
  resetVisible.value = true
}

async function onReset() {
  if (!resetForm.password) {
    ElMessage.warning('Please enter a new password')
    return
  }
  resetting.value = true
  try {
    await resetPassword(resetTarget.value.id, resetForm.password)
    ElMessage.success('Password reset')
    resetVisible.value = false
  } catch {
    // interceptor already shows the error
  } finally {
    resetting.value = false
  }
}

async function onDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete user ${row.username}?`, 'Delete Confirmation', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteUser(row.id)
    ElMessage.success('Deleted')
    loadData()
  } catch {
    // interceptor already shows the error (e.g. ADMIN_USER_IMMUTABLE)
  }
}

onMounted(loadData)
</script>

<style scoped>
.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
