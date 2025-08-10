import { create } from 'zustand'

type AuthState = {
  token: string | null
  user: string | null
  roles: string[]
  setAuth: (data: { token: string; user: string; roles: string[] }) => void
  setToken: (token: string | null) => void
  logout: () => void
}

const storageKey = 'igos_auth'

const persisted = (() => {
  try { return JSON.parse(localStorage.getItem(storageKey) || 'null') } catch { return null }
})()

const useAuthStore = create<AuthState>((set) => ({
  token: persisted?.token || null,
  user: persisted?.user || null,
  roles: persisted?.roles || [],
  setAuth: ({ token, user, roles }) => {
    localStorage.setItem(storageKey, JSON.stringify({ token, user, roles }))
    set({ token, user, roles })
  },
  setToken: (token) => {
    const current = (() => { try { return JSON.parse(localStorage.getItem(storageKey) || 'null') } catch { return null } })() || {}
    localStorage.setItem(storageKey, JSON.stringify({ ...current, token }))
    set({ token })
  },
  logout: () => {
    localStorage.removeItem(storageKey)
    set({ token: null, user: null, roles: [] })
  }
}))

export default useAuthStore


