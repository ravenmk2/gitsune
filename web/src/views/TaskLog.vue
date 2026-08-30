<template>
  <div>
    <div class="toolbar">
      <el-button type="primary" :loading="starting === 'github_trending'" @click="onStart('github_trending')">
        Collect GitHub Trending
      </el-button>
      <el-button type="warning" :loading="starting === 'gitee_gvp'" @click="onStart('gitee_gvp')">
        Collect Gitee GVP
      </el-button>
    </div>

    <div class="filter-bar">
      <el-select v-model="query.type" placeholder="Type" clearable style="width: 170px">
        <el-option label="GitHub Trending" value="github_trending" />
        <el-option label="Gitee GVP" value="gitee_gvp" />
      </el-select>
      <el-select v-model="query.status" placeholder="Status" clearable style="width: 130px">
        <el-option label="Running" value="running" />
        <el-option label="Success" value="success" />
        <el-option label="Failed" value="failed" />
      </el-select>
      <el-select v-model="query.trigger_mode" placeholder="Trigger" clearable style="width: 130px">
        <el-option label="Manual" value="manual" />
        <el-option label="Auto" value="auto" />
      </el-select>
      <el-button type="primary" :icon="Search" @click="onSearch">Search</el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe>
      <el-table-column label="Type" width="160">
        <template #default="{ row }">{{ typeLabel(row.type) }}</template>
      </el-table-column>
      <el-table-column label="Status" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Trigger" width="100">
        <template #default="{ row }">{{ triggerLabel(row.trigger_mode) }}</template>
      </el-table-column>
      <el-table-column prop="added_count" label="Added" width="90" />
      <el-table-column label="Duration" width="110">
        <template #default="{ row }">{{ duration(row) }}</template>
      </el-table-column>
      <el-table-column prop="message" label="Message" min-width="200" show-overflow-tooltip />
      <el-table-column label="Started At" width="170">
        <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
      </el-table-column>
      <el-table-column label="Finished At" width="170">
        <template #default="{ row }">{{ formatTime(row.finished_at) }}</template>
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
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { listTaskLogs, startTask } from '../api'

const query = reactive({ type: '', status: '', trigger_mode: '' })
const items = ref([])
const page = ref(1)
const size = ref(10)
const total = ref(0)
const loading = ref(false)
const starting = ref('')

let pollTimer = null

function formatTime(t) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}

function duration(row) {
  if (!row.started_at || !row.finished_at) return '-'
  const ms = dayjs(row.finished_at).diff(dayjs(row.started_at))
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function typeLabel(type) {
  return { github_trending: 'GitHub Trending', gitee_gvp: 'Gitee GVP' }[type] || type
}

function statusLabel(status) {
  return { running: 'Running', success: 'Success', failed: 'Failed' }[status] || status
}

function statusTagType(status) {
  return { running: 'warning', success: 'success', failed: 'danger' }[status] || 'info'
}

function triggerLabel(mode) {
  return { manual: 'Manual', auto: 'Auto' }[mode] || mode
}

async function loadData() {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value }
    if (query.type) params.type = query.type
    if (query.status) params.status = query.status
    if (query.trigger_mode) params.trigger_mode = query.trigger_mode
    const data = await listTaskLogs(params)
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

async function onStart(type) {
  starting.value = type
  try {
    await startTask(type)
    ElMessage.success('Task started')
    setTimeout(loadData, 1500)
    setupPolling()
  } catch {
    // interceptor already shows the error (e.g. TASK_ALREADY_RUNNING)
  } finally {
    starting.value = ''
  }
}

function setupPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    const hasRunning = items.value.some((it) => it.status === 'running')
    if (!hasRunning) {
      clearInterval(pollTimer)
      pollTimer = null
      return
    }
    await loadData()
  }, 10000)
}

onMounted(loadData)
onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
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
