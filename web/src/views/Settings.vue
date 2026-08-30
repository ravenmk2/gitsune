<template>
  <el-card v-loading="loading" class="settings-card">
    <template #header>Settings</template>
    <el-form label-width="160px" style="max-width: 640px">
      <el-form-item label="GitHub Token">
        <el-input
          v-model="form.github_token"
          :placeholder="maskedToken ? `Currently set: ${maskedToken}. Enter a new value to override` : 'Not set. Enter a token to configure'"
          clearable
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="savingToken" @click="onSaveToken">Save Token</el-button>
        <el-button type="danger" plain :disabled="!maskedToken" @click="onClearToken">Clear Token</el-button>
      </el-form-item>
      <el-form-item label="Collection Cron">
        <el-input v-model="form.cron" placeholder="e.g. 0 3 * * *" />
      </el-form-item>
      <el-form-item label=" ">
        <span class="hint">Collection schedule, standard 5-field cron expression (minute hour day month weekday), default 0 */6 * * * runs every 6 hours</span>
      </el-form-item>
      <el-form-item label=" ">
        <el-button type="primary" :loading="savingCron" @click="onSaveCron">Save Cron</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSetting, updateSetting } from '../api'

const loading = ref(false)
const maskedToken = ref('')
const form = reactive({ github_token: '', cron: '' })
const savingToken = ref(false)
const savingCron = ref(false)

async function loadData() {
  loading.value = true
  try {
    const data = await getSetting()
    maskedToken.value = data.github_token || ''
    form.cron = data.cron || ''
  } catch {
    // interceptor already shows the error
  } finally {
    loading.value = false
  }
}

async function onSaveToken() {
  if (!form.github_token.trim()) {
    ElMessage.warning('Enter a new token, or click "Clear Token"')
    return
  }
  savingToken.value = true
  try {
    await updateSetting({ github_token: form.github_token.trim() })
    ElMessage.success('Token saved')
    form.github_token = ''
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    savingToken.value = false
  }
}

async function onClearToken() {
  try {
    await ElMessageBox.confirm('Clear the GitHub Token?', 'Confirm', { type: 'warning' })
  } catch {
    return
  }
  savingToken.value = true
  try {
    await updateSetting({ github_token: '' })
    ElMessage.success('Token cleared')
    form.github_token = ''
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    savingToken.value = false
  }
}

async function onSaveCron() {
  if (!form.cron.trim()) {
    ElMessage.warning('Please enter a cron expression')
    return
  }
  savingCron.value = true
  try {
    await updateSetting({ cron: form.cron.trim() })
    ElMessage.success('Cron saved')
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    savingCron.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.settings-card {
  max-width: 800px;
}
.hint {
  color: #909399;
  font-size: 12px;
}
</style>
