<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight } from 'lucide-vue-next'
import { ApiError, post } from '../../api'
import { useSessionStore } from '../../stores/session'
const router=useRouter(),session=useSessionStore(),email=ref(''),password=ref(''),error=ref(''),loading=ref(false)
async function login(){error.value='';loading.value=true;try{await post('/admin/login',{email:email.value,password:password.value});await session.checkAdmin();router.push('/admin')}catch(e){error.value=e instanceof ApiError?e.message:'登录失败'}finally{loading.value=false}}
</script>
<template><main class="admin-login"><section><div class="brand"><span>IT</span><strong>STUDIO</strong></div><p>RECRUITMENT OPERATING SYSTEM</p><h1>CONTROL<br/>ROOM<span>.</span></h1><small>AUTHORIZED PERSONNEL ONLY / 2026</small></section><form @submit.prevent="login"><p class="step-no">ADMIN / AUTHENTICATION</p><h2>管理后台</h2><label>管理员邮箱<input v-model="email" type="email" required autocomplete="username"/></label><label>密码<input v-model="password" type="password" required autocomplete="current-password"/></label><p v-if="error" class="form-message error">{{ error }}</p><button class="primary-btn" :disabled="loading">{{ loading?'验证中…':'进入后台' }} <ArrowRight/></button><RouterLink to="/">← 返回公开站点</RouterLink></form></main></template>

