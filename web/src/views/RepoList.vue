<template>
  <div>
    <div class="filter-bar">
      <el-select v-model="query.platform" placeholder="Platform" clearable style="width: 130px">
        <el-option label="GitHub" value="github" />
        <el-option label="GitLab" value="gitlab" />
        <el-option label="Gitee" value="gitee" />
      </el-select>
      <el-select
        v-model="query.language"
        placeholder="Language"
        clearable
        filterable
        allow-create
        style="width: 150px"
      >
        <el-option v-for="lang in languages" :key="lang" :label="lang" :value="lang" />
      </el-select>
      <el-select v-model="query.source" placeholder="Source" clearable style="width: 140px">
        <el-option label="Manual" value="manual" />
        <el-option label="Trending" value="trending" />
      </el-select>
      <el-input v-model="query.keyword" placeholder="Keyword" clearable style="width: 180px" @keyup.enter="onSearch" />
      <el-button type="primary" :icon="Search" @click="onSearch">Search</el-button>
      <el-button type="success" :icon="Plus" @click="collectVisible = true">Collect Repo</el-button>
      <el-button :icon="Download" @click="exportVisible = true">Export</el-button>
      <el-button v-if="isAdmin" :icon="Upload" @click="importVisible = true">Import</el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe @sort-change="onSortChange">
      <el-table-column label="Platform" width="90">
        <template #default="{ row }">
          <el-tag :style="platformTagStyle(row.platform)">{{ row.platform }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Repository" min-width="200">
        <template #default="{ row }">
          <repo-hover-card :repo="row">
            <el-link type="primary" :href="row.url" target="_blank" underline="never">
              {{ row.owner }}/{{ row.name }}
            </el-link>
          </repo-hover-card>
        </template>
      </el-table-column>
      <el-table-column label="Description" min-width="180">
        <template #default="{ row }">
          <repo-hover-card :repo="row">
            <span class="desc-text">{{ row.description || '-' }}</span>
          </repo-hover-card>
        </template>
      </el-table-column>
      <el-table-column prop="language" label="Language" width="100" />
      <el-table-column prop="stars" label="Stars" width="90" sortable="custom" />
      <el-table-column prop="forks" label="Forks" width="90" sortable="custom" />
      <el-table-column label="License" width="120">
        <template #default="{ row }">{{ row.license || '-' }}</template>
      </el-table-column>
      <el-table-column label="Source" width="110">
        <template #default="{ row }">
          <el-tag :type="sourceTagType(row.source)" effect="plain">{{ sourceLabel(row.source) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Last Synced" width="170">
        <template #default="{ row }">{{ formatTime(row.refreshed_at) }}</template>
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

    <el-dialog v-model="exportVisible" title="Export Repos" width="480px">
      <el-checkbox-group v-model="exportPlatforms">
        <el-checkbox value="github">GitHub</el-checkbox>
        <el-checkbox value="gitlab">GitLab</el-checkbox>
        <el-checkbox value="gitee">Gitee</el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button @click="exportVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="exporting" :disabled="exportPlatforms.length === 0" @click="onExport">
          Export
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importVisible" title="Import Repos" width="520px" @closed="resetImport">
      <div class="import-section">
        <input ref="fileInput" type="file" accept=".json" @change="onFileChange" />
        <div v-if="importSummary.length" class="import-summary">
          <el-tag v-for="s in importSummary" :key="s.platform" effect="plain">{{ s.platform }}: {{ s.count }}</el-tag>
        </div>
      </div>
      <el-radio-group v-model="importMode" class="import-section">
        <el-radio value="incremental">Incremental (merge &amp; update)</el-radio>
        <el-radio value="overwrite">Overwrite (replace all)</el-radio>
      </el-radio-group>
      <el-alert
        v-if="importMode === 'overwrite'"
        type="error"
        title="All existing repos will be deleted and replaced."
        :closable="false"
        show-icon
      />
      <template #footer>
        <el-button @click="importVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="importing" :disabled="!importData" @click="onImport">Import</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus, Download, Upload } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { listRepos, collectRepo, refreshRepo, deleteRepo, exportRepos, importRepos } from '../api'
import RepoHoverCard from '../components/RepoHoverCard.vue'
import { useUser } from '../stores/user'

const { isAdmin } = useUser()

const query = reactive({ platform: '', keyword: '', language: '', source: '' })

const languages = [
  'JavaScript', 'TypeScript', 'Python', 'Go', 'Java', 'C', 'C++', 'C#', 'Rust', 'Ruby',
  'PHP', 'Swift', 'Kotlin', 'Dart', 'Scala', 'Lua', 'Shell', 'HTML', 'CSS', 'Vue'
]
const items = ref([])
const page = ref(1)
const size = ref(10)
const total = ref(0)
const loading = ref(false)
const sort = reactive({ by: '', order: '' })

const collectVisible = ref(false)
const collectUrl = ref('')
const collecting = ref(false)

const exportVisible = ref(false)
const exportPlatforms = ref(['github', 'gitlab', 'gitee'])
const exporting = ref(false)

const importVisible = ref(false)
const importMode = ref('incremental')
const importData = ref(null)
const importing = ref(false)
const fileInput = ref(null)
const importSummary = computed(() => {
  const platforms = importData.value?.platforms
  if (!platforms || typeof platforms !== 'object') return []
  return Object.entries(platforms)
    .filter(([, repos]) => Array.isArray(repos))
    .map(([platform, repos]) => ({ platform, count: repos.length }))
})

function formatTime(t) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}

// 平台品牌色：GitHub 深灰近黑、GitLab 橙、Gitee 橙红
function platformTagStyle(platform) {
  const color = { github: '#1b1f23', gitlab: '#fc6d26', gitee: '#c71d23' }[platform] || '#909399'
  return { backgroundColor: color, borderColor: color, color: '#fff' }
}

function sourceLabel(source) {
  return { manual: 'Manual', trending: 'Trending' }[source] || source
}

function sourceTagType(source) {
  return { manual: 'success', trending: 'primary' }[source] || 'info'
}

async function loadData() {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value }
    if (query.platform) params.platform = query.platform
    if (query.keyword) params.keyword = query.keyword
    if (query.language) params.language = query.language
    if (query.source) params.source = query.source
    if (sort.by && sort.order) {
      params.sort_by = sort.by
      params.sort_order = sort.order
    }
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

function onSortChange({ prop, order }) {
  sort.by = order ? prop : ''
  sort.order = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
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

async function onExport() {
  exporting.value = true
  try {
    const data = await exportRepos(exportPlatforms.value)
    const count = Object.values(data?.platforms || {}).reduce(
      (sum, repos) => sum + (Array.isArray(repos) ? repos.length : 0),
      0
    )
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `gitsune-repos-${dayjs().format('YYYYMMDD-HHmmss')}.json`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success(`Exported ${count} repos`)
    exportVisible.value = false
  } catch {
    // interceptor already shows the error
  } finally {
    exporting.value = false
  }
}

async function onFileChange(event) {
  const file = event.target.files?.[0]
  if (!file) return
  try {
    const parsed = JSON.parse(await file.text())
    if (!parsed || typeof parsed !== 'object' || !parsed.platforms) {
      throw new Error('missing platforms')
    }
    importData.value = parsed
  } catch {
    ElMessage.error('Invalid file')
    importData.value = null
    event.target.value = ''
  }
}

async function onImport() {
  if (!importData.value) return
  if (importMode.value === 'overwrite') {
    try {
      await ElMessageBox.confirm(
        `All ${total.value} existing repos will be deleted and replaced. Continue?`,
        'Overwrite Confirmation',
        { type: 'warning', confirmButtonText: 'Overwrite' }
      )
    } catch {
      return
    }
  }
  importing.value = true
  try {
    const result = await importRepos({ mode: importMode.value, platforms: importData.value.platforms })
    importVisible.value = false
    await showImportResult(result)
    loadData()
  } catch {
    // interceptor already shows the error
  } finally {
    importing.value = false
  }
}

function showImportResult(result) {
  const failed = result?.failed || []
  const escape = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)
  let html = `<p>Added ${result?.added ?? 0} / Updated ${result?.updated ?? 0} / Failed ${failed.length}</p>`
  if (failed.length) {
    const rows = failed
      .map((f) => `<div>${escape(f.platform)}: ${escape(f.owner)}/${escape(f.name)} - ${escape(f.reason)}</div>`)
      .join('')
    html += `<div style="max-height: 240px; overflow-y: auto; text-align: left;">${rows}</div>`
  }
  return ElMessageBox.alert(html, 'Import Result', {
    confirmButtonText: 'OK',
    dangerouslyUseHTMLString: true
  }).catch(() => {})
}

function resetImport() {
  importMode.value = 'incremental'
  importData.value = null
  importing.value = false
  if (fileInput.value) fileInput.value.value = ''
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
.desc-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.import-section {
  margin-bottom: 16px;
}
.import-summary {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.import-summary + .import-section {
  margin-top: 16px;
}
</style>
