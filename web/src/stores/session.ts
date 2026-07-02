import { defineStore } from 'pinia'
import { api } from '../api'

type AdminSession = { id:number; email:string; role:'owner'|'readonly'; isSuperAdmin:boolean }

export const useSessionStore = defineStore('session', {
  state: () => ({ admin: null as null | AdminSession, checked: false }),
  getters: {
    isOwner: state => state.admin?.role === 'owner',
    isSuperAdmin: state => state.admin?.isSuperAdmin === true,
  },
  actions: {
    async checkAdmin() {
      try { this.admin = (await api<{admin:AdminSession}>('/admin/me')).admin }
      catch { this.admin = null }
      this.checked = true
    },
    async logout() { await api('/admin/logout', { method:'POST', body:'{}' }); this.admin = null },
  },
})
