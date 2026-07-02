import { createRouter, createWebHistory } from 'vue-router'
import HomeView from './views/HomeView.vue'
import ApplyView from './views/ApplyView.vue'
import ResetPasswordView from './views/ResetPasswordView.vue'
import AdminLoginView from './views/admin/AdminLoginView.vue'
import AdminView from './views/admin/AdminView.vue'

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (to) => to.hash ? ({ el:to.hash, behavior:'smooth' }) : ({ top: 0 }),
  routes: [
    { path:'/', component:HomeView },
    { path:'/apply', component:ApplyView },
    { path:'/reset-password', component:ResetPasswordView },
    { path:'/admin/login', component:AdminLoginView },
    { path:'/admin', component:AdminView },
  ],
})
