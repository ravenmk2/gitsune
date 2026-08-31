<template>
  <el-card v-loading="loading" class="settings-card">
    <template #header>Settings</template>
    <el-form label-width="160px" style="max-width: 720px">
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
      <el-divider content-position="left">Scheduled Tasks</el-divider>
      <el-form-item v-for="t in taskList" :key="t.type" :label="t.label">
        <div class="task-row">
          <el-switch v-model="form.tasks[t.type].enabled" />
          <el-input v-model="form.tasks[t.type].cron" placeholder="e.g. 0 3 * * *" class="cron-input" />
          <el-button type="primary" :loading="savingTask === t.type" @click="onSaveTask(t.type)">Save</el-button>
        </div>
        <div class="hint">{{ t.hint }}</div>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSetting, updateSetting } from '../api'

const taskList = [
  {
    type: 'github_trending',
    label: 'GitHub Trending',
    hint: 'Collects the GitHub Trending chart (new repos only, existing ones are never overwritten)'
  },
  {
    type: 'repo_refresh',
    label: 'Repo Refresh',
    hint: 'Re-fetches repos not refreshed in the last 7 days'
  }
]

const loading = ref(false)
const maskedToken = ref('')
const form = reactive({
  github_token: '',
  tasks: {
    github_trending: { enabled: true, cron: '' },
    repo_refresh: { enabled: true, cron: '' }
  }
})
const savingToken = ref(false)
const savingTask = ref('')

async function loadData() {
  loading.value = true
  try {
    const data = await getSetting()
    maskedToken.value = data.github_token || ''
    for (const t of taskList) {
      const cfg = (data.tasks || {})[t.type]
      if (cfg) {
        form.tasks[t.type].enabled = !!cfg.enabled
        form.tasks[t.type].cron = cfg.cron || ''
      }
    }
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

async function onSaveTask(type) {
  const cfg = form.tasks[type]
  if (!cfg.cron.trim()) {
    ElMessage.warning('Please enter a cron expression')
    return
  }
  savingTask.value = type
  try {
    await updateSetting({ tasks: { [type]: { enabled: cfg.enabled, cron: cfg.cron.trim() } } })
    ElMessage.success('Task schedule saved')
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    savingTask.value = ''
  }
}

onMounted(loadData)
</script>

<style scoped>
.settings-card {
  max-width: 800px;
}
.task-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.cron-input {
  width: 220px;
}
.hint {
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
</style>
