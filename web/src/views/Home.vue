<template>
  <div v-loading="loading" class="home">
    <div class="stat-cards">
      <el-card shadow="hover" class="stat-card">
        <div class="stat-value">{{ formatCount(overview.total_repos) }}</div>
        <div class="stat-label">Total Repos</div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-value">{{ formatCount(overview.total_stars) }}</div>
        <div class="stat-label">Total Stars</div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-value">+{{ formatCount(overview.recent_added) }}</div>
        <div class="stat-label">Added Last 7 Days</div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="platform-counts">
          <div v-for="p in overview.platforms" :key="p.name" class="platform-count">
            <el-tag size="small" :style="platformTagStyle(p.name)">{{ p.name }}</el-tag>
            <span class="platform-num">{{ p.count }}</span>
          </div>
          <span v-if="!overview.platforms?.length" class="empty-hint">No data</span>
        </div>
        <div class="stat-label">Platforms</div>
      </el-card>
    </div>

    <div class="card-row">
      <el-card shadow="never" class="panel">
        <template #header>Top Stars</template>
        <div v-if="overview.top_repos?.length" class="repo-list">
          <div v-for="(r, i) in overview.top_repos" :key="r.id" class="repo-row">
            <span class="repo-rank">{{ i + 1 }}</span>
            <repo-hover-card :repo="r" class="repo-name">
              <el-link type="primary" :href="r.url" target="_blank" underline="never">
                {{ r.owner }}/{{ r.name }}
              </el-link>
            </repo-hover-card>
            <span class="repo-stars">★ {{ formatCount(r.stars) }}</span>
          </div>
        </div>
        <el-empty v-else description="No data" :image-size="60" />
      </el-card>

      <el-card shadow="never" class="panel">
        <template #header>Recently Added</template>
        <div v-if="overview.latest_repos?.length" class="repo-list">
          <div v-for="r in overview.latest_repos" :key="r.id" class="repo-row">
            <el-tag size="small" :style="platformTagStyle(r.platform)" class="repo-platform">{{ r.platform }}</el-tag>
            <repo-hover-card :repo="r" class="repo-name">
              <el-link type="primary" :href="r.url" target="_blank" underline="never">
                {{ r.owner }}/{{ r.name }}
              </el-link>
            </repo-hover-card>
            <span class="repo-time">{{ formatDate(r.created_at) }}</span>
          </div>
        </div>
        <el-empty v-else description="No data" :image-size="60" />
      </el-card>
    </div>

    <div class="card-row">
      <el-card shadow="never" class="panel" :class="{ 'panel-full': !isAdmin }">
        <template #header>Top Languages</template>
        <div v-if="overview.languages?.length" class="lang-list">
          <div v-for="lang in overview.languages" :key="lang.name" class="lang-row">
            <span class="lang-name" :title="lang.name">{{ lang.name }}</span>
            <div class="lang-bar-track">
              <div class="lang-bar" :style="{ width: barWidth(lang.count, maxLangCount) }" />
            </div>
            <span class="lang-count">{{ lang.count }}</span>
          </div>
        </div>
        <el-empty v-else description="No data" :image-size="60" />
      </el-card>

      <el-card v-if="isAdmin" v-loading="tasksLoading" shadow="never" class="panel">
        <template #header>
          <div class="panel-header">
            <span>Task Status</span>
            <el-link type="primary" underline="never" @click="router.push('/task-logs')">View All</el-link>
          </div>
        </template>
        <div v-if="taskRows.length" class="task-list">
          <div v-for="t in taskRows" :key="t.type" class="task-row">
            <span class="task-name">{{ typeLabel(t.type) }}</span>
            <template v-if="t.log">
              <el-tag :type="statusTagType(t.log.status)" size="small">{{ t.log.status }}</el-tag>
              <span class="task-meta">+{{ t.log.added_count }} added</span>
              <span class="task-meta">{{ formatTime(t.log.finished_at || t.log.started_at) }}</span>
            </template>
            <span v-else class="empty-hint">Never run</span>
          </div>
        </div>
        <el-empty v-else-if="!tasksLoading" description="No data" :image-size="60" />
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { getStatsOverview, listTaskLogs } from '../api'
import { useUser } from '../stores/user'
import RepoHoverCard from '../components/RepoHoverCard.vue'

const router = useRouter()
const { isAdmin } = useUser()

const overview = ref({})
const loading = ref(false)
const taskRows = ref([])
const tasksLoading = ref(false)

const TASK_TYPES = ['github_trending', 'repo_refresh']

const maxLangCount = computed(() => Math.max(1, ...(overview.value.languages || []).map((l) => l.count)))

function platformTagStyle(platform) {
  const color = { github: '#1b1f23', gitlab: '#fc6d26', gitee: '#c71d23' }[platform] || '#909399'
  return { backgroundColor: color, borderColor: color, color: '#fff' }
}

function formatCount(n) {
  const num = Number(n) || 0
  if (num >= 1000) {
    const k = (num / 1000).toFixed(1)
    return (k.endsWith('.0') ? k.slice(0, -2) : k) + 'k'
  }
  return String(num)
}

function barWidth(count, max) {
  return `${Math.max(2, Math.round((count / max) * 100))}%`
}

function formatTime(t) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}

function formatDate(t) {
  return t ? dayjs(t).format('MM-DD HH:mm') : '-'
}

function typeLabel(type) {
  return { github_trending: 'GitHub Trending', repo_refresh: 'Repo Refresh' }[type] || type
}

function statusTagType(status) {
  return { running: 'warning', success: 'success', failed: 'danger' }[status] || 'info'
}

async function loadOverview() {
  loading.value = true
  try {
    overview.value = (await getStatsOverview()) || {}
  } catch {
    // interceptor already shows the error
  } finally {
    loading.value = false
  }
}

async function loadTasks() {
  tasksLoading.value = true
  try {
    const results = await Promise.all(
      TASK_TYPES.map((type) => listTaskLogs({ type, page: 1, size: 1 }))
    )
    taskRows.value = TASK_TYPES.map((type, i) => ({ type, log: results[i]?.items?.[0] || null }))
  } catch {
    // interceptor already shows the error
  } finally {
    tasksLoading.value = false
  }
}

onMounted(() => {
  loadOverview()
  if (isAdmin.value) loadTasks()
})
</script>

<style scoped>
.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}
.stat-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 20px;
}
.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}
.stat-label {
  font-size: 13px;
  color: #909399;
}
.platform-counts {
  display: flex;
  gap: 12px;
  align-items: center;
  min-height: 32px;
}
.platform-count {
  display: flex;
  align-items: center;
  gap: 4px;
}
.platform-num {
  font-weight: 600;
  color: #303133;
}
.card-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}
.panel-full {
  grid-column: 1 / -1;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.empty-hint {
  color: #c0c4cc;
  font-size: 13px;
}
.lang-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.lang-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.lang-name {
  width: 130px;
  flex-shrink: 0;
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lang-bar-track {
  flex: 1;
  height: 12px;
  background: #f0f2f5;
  border-radius: 6px;
  overflow: hidden;
}
.lang-bar {
  height: 100%;
  background: #409eff;
  border-radius: 6px;
}
.lang-count {
  width: 40px;
  flex-shrink: 0;
  text-align: right;
  font-size: 13px;
  color: #606266;
}
.task-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.task-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}
.task-name {
  width: 130px;
  flex-shrink: 0;
  color: #303133;
  font-weight: 500;
}
.task-meta {
  color: #909399;
}
.repo-list {
  display: flex;
  flex-direction: column;
}
.repo-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 0;
  border-bottom: 1px solid #f0f2f5;
}
.repo-row:last-child {
  border-bottom: none;
}
.repo-platform {
  flex-shrink: 0;
}
.repo-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.repo-rank {
  width: 22px;
  flex-shrink: 0;
  text-align: center;
  font-weight: 600;
  color: #909399;
}
.repo-stars,
.repo-time {
  flex-shrink: 0;
  font-size: 13px;
  color: #909399;
}
@media (max-width: 1100px) {
  .stat-cards,
  .card-row {
    grid-template-columns: 1fr;
  }
}
</style>
