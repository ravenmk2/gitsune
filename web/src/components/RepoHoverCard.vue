<template>
  <el-popover trigger="hover" placement="right" :width="360" popper-class="repo-hover-card">
    <template #reference>
      <span class="repo-hover-trigger"><slot /></span>
    </template>
    <div class="card">
      <div class="card-header">
        <el-tag size="small" :style="platformTagStyle">{{ repo.platform }}</el-tag>
        <el-link type="primary" :href="repo.url" target="_blank" underline="hover" class="card-title">
          {{ repo.owner }}/{{ repo.name }}
        </el-link>
      </div>
      <div class="card-desc" :class="{ empty: !repo.description }">
        {{ repo.description || 'No description' }}
      </div>
      <div class="card-meta">
        <span v-if="repo.language" class="meta-item">
          <span class="lang-dot" :style="{ backgroundColor: langColor }" />{{ repo.language }}
        </span>
        <span class="meta-item" title="Stars">★ {{ formatCount(repo.stars) }}</span>
        <span class="meta-item" title="Forks">⑂ {{ formatCount(repo.forks) }}</span>
        <span v-if="repo.license" class="meta-item" title="License">{{ repo.license }}</span>
      </div>
      <div class="card-footer">Last synced {{ syncedAt }}</div>
    </div>
  </el-popover>
</template>

<script setup>
import { computed } from 'vue'
import dayjs from 'dayjs'

const props = defineProps({
  repo: { type: Object, required: true }
})

const LANG_COLORS = {
  JavaScript: '#f1e05a',
  TypeScript: '#3178c6',
  Python: '#3572a5',
  Go: '#00add8',
  Java: '#b07219',
  'C++': '#f34b7d',
  C: '#555555',
  'C#': '#178600',
  Rust: '#dea584',
  Ruby: '#701516',
  PHP: '#4f5d95',
  Vue: '#41b883',
  Shell: '#89e051',
  HTML: '#e34c26',
  CSS: '#563d7c',
  Kotlin: '#a97bff',
  Swift: '#f05138'
}

// 平台品牌色：GitHub 深灰近黑、GitLab 橙、Gitee 橙红
const platformTagStyle = computed(() => {
  const color = { github: '#1b1f23', gitlab: '#fc6d26', gitee: '#c71d23' }[props.repo.platform] || '#909399'
  return { backgroundColor: color, borderColor: color, color: '#fff' }
})

const langColor = computed(() => LANG_COLORS[props.repo.language] || '#909399')

const syncedAt = computed(() =>
  props.repo.last_synced_at ? dayjs(props.repo.last_synced_at).format('YYYY-MM-DD HH:mm:ss') : 'never'
)

function formatCount(n) {
  const num = Number(n) || 0
  if (num >= 1000) {
    const k = (num / 1000).toFixed(1)
    return (k.endsWith('.0') ? k.slice(0, -2) : k) + 'k'
  }
  return String(num)
}
</script>

<style scoped>
.repo-hover-trigger {
  display: inline-block;
  max-width: 100%;
  vertical-align: middle;
}
.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.card-title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-desc {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: #606266;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.card-desc.empty {
  color: #c0c4cc;
  font-style: italic;
}
.card-meta {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  font-size: 12px;
  color: #606266;
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.lang-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}
.card-footer {
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px solid #ebeef5;
  font-size: 12px;
  color: #909399;
}
</style>
