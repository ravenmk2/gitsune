import { reactive, computed } from 'vue'
import { USER_KEY } from '../api'

function loadUser() {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY)) || null
  } catch {
    return null
  }
}

const state = reactive({
  user: loadUser()
})

export function setUser(user) {
  state.user = user
  if (user) {
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  } else {
    localStorage.removeItem(USER_KEY)
  }
}

export function clearUser() {
  setUser(null)
}

export function useUser() {
  return {
    user: computed(() => state.user),
    isAdmin: computed(() => state.user?.role === 'admin'),
    setUser,
    clearUser
  }
}
