<template>
  <div>
    <div class="filter-bar">
      <el-select v-model="query.platform" placeholder="Platform" clearable style="width: 130px">
        <el-option label="GitHub" value="github" />
        <el-option label="GitLab" value="gitlab" />
        <el-option label="Gitee" value="gitee" />
      </el-select>
      <el-input v-model="query.keyword" placeholder="Keyword" clearable style="width: 180px" @keyup.enter="onSearch" />
      <el-input v-model="query.language" placeholder="Language" clearable style="width: 130px" @keyup.enter="onSearch" />
      <el-select v-model="query.source" placeholder="Source" clearable style="width: 140px">
        <el-option label="Manual" value="manual" />
        <el-option label="Trending" value="trending" />
        <el-option label="GVP" value="gvp" />
      </el-select>
      <el-button type="primary" :icon="Search" @click="onSearch">Search</el-button>
      <el-button type="success" :icon="Plus" @click="collectVisible = true">Collect Repo</el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe>
      <el-table-column label="Platform" width="90">
        <template #default="{ row }">
          <el-tag :type="platformTagType(row.platform)">{{ row.platform }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Repository" min-width="200">
        <template #default="{ row }">
          <el-link type="primary" :href="row.url" target="_blank" underline="never">
            {{ row.owner }}/{{ row.name }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="Description" min-width="180" show-overflow-tooltip />
      <el-table-column prop="language" label="Language" width="100" />
      <el-table-column prop="stars" label="Stars" width="90" sortable />
      <el-table-column prop="forks" label="Forks" width="90" />
      <el-table-column label="License" width="120">
        <template #default="{ row }">{{ row.license || '-' }}</template>
      </el-table-column>
      <el-table-column label="Source" width="110">
        <template #default="{ row }">
          <el-tag :type="sourceTagType(row.source)" effect="plain">{{ sourceLabel(row.source) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Last Synced" width="170">
        <template #default="{ row }">{{ formatTime(row.last_synced_at) }}</template>
      </el-table-column>
      <el-table-column v-if="isAdmin" label="Actions" width="140">
        <template #default="{ row }">
          <el-button link type="primary" @click="onRefresh(row)">Refresh</el-button>
          <el-button link type="danger" @click="onDelete(row)">Delete</el-button>
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

    <el-dialog v-model="collectVisible" title="Collect Repo" width="480px">
      <el-input v-model="collectUrl" placeholder="Enter a repo URL, e.g. https://github.com/owner/repo" />
      <template #footer>
        <el-button @click="collectVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="collecting" @click="onCollect">Submit</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { listRepos, collectRepo, refreshRepo, deleteRepo } from '../api'
import { useUser } from '../stores/user'

const { isAdmin } = useUser()

const query = reactive({ platform: '', keyword: '', language: '', source: '' })
const items = ref([])
const page = ref(1)
const size = ref(10)
const total = ref(0)
const loading = ref(false)

const collectVisible = ref(false)
const collectUrl = ref('')
const collecting = ref(false)

function formatTime(t) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}

function platformTagType(platform) {
  return { github: 'info', gitlab: 'danger', gitee: 'warning' }[platform] || 'info'
}

function sourceLabel(source) {
  return { manual: 'Manual', trending: 'Trending', gvp: 'GVP' }[source] || source
}

function sourceTagType(source) {
  return { manual: 'success', trending: 'primary', gvp: 'warning' }[source] || 'info'
}

async function loadData() {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value }
    if (query.platform) params.platform = query.platform
    if (query.keyword) params.keyword = query.keyword
    if (query.language) params.language = query.language
    if (query.source) params.source = query.source
    const data = await listRepos(params)
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

async function onCollect() {
  if (!collectUrl.value.trim()) {
    ElMessage.warning('Please enter a repo URL')
    return
  }
  collecting.value = true
  try {
    await collectRepo(collectUrl.value.trim())
    ElMessage.success('Repo collected')
    collectVisible.value = false
    collectUrl.value = ''
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    collecting.value = false
  }
}

async function onRefresh(row) {
  try {
    await refreshRepo(row.id)
    ElMessage.success('Refreshed')
    loadData()
  } catch {
    // interceptor already shows the error
  }
}

async function onDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete repo ${row.owner}/${row.name}?`, 'Delete Confirmation', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteRepo(row.id)
    ElMessage.success('Deleted')
    loadData()
  } catch {
    // interceptor already shows the error
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
