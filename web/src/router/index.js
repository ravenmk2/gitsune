import { createRouter, createWebHistory } from 'vue-router'
import { TOKEN_KEY, USER_KEY } from '../api'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('../layouts/AdminLayout.vue'),
    redirect: '/repos',
    children: [
      { path: 'repos', name: 'repos', component: () => import('../views/RepoList.vue'), meta: { title: 'Repositories' } },
      { path: 'task-logs', name: 'task-logs', component: () => import('../views/TaskLog.vue'), meta: { title: 'Task Logs', requiresAdmin: true } },
      { path: 'users', name: 'users', component: () => import('../views/UserList.vue'), meta: { title: 'Users', requiresAdmin: true } },
      { path: 'settings', name: 'settings', component: () => import('../views/Settings.vue'), meta: { title: 'Settings', requiresAdmin: true } }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/repos' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (!to.meta.public && !token) {
    return { path: '/login' }
  }
  if (to.path === '/login' && token) {
    return { path: '/' }
  }
  if (to.meta.requiresAdmin) {
    try {
      const user = JSON.parse(localStorage.getItem(USER_KEY))
      if (user?.role !== 'admin') {
        return { path: '/repos' }
      }
    } catch {
      return { path: '/repos' }
    }
  }
  return true
})

export default router
