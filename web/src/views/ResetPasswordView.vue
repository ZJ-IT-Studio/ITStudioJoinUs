<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { Check } from 'lucide-vue-next'
import SiteHeader from '../components/SiteHeader.vue'
import { ApiError, post } from '../api'
const route=useRoute(),password=ref(''),confirmPassword=ref(''),done=ref(false),error=ref(''),loading=ref(false)
async function submit(){error.value='';if(password.value!==confirmPassword.value){error.value='两次输入的密码不一致';return};loading.value=true;try{await post('/password/reset',{token:String(route.query.token||''),password:password.value});done.value=true}catch(e){error.value=e instanceof ApiError?e.message:'重置失败'}finally{loading.value=false}}
</script>
<template><div class="portal-page"><SiteHeader/><main class="standalone-card"><div v-if="done" class="success-state"><Check/><h1>查询密码已更新</h1><p>现在可以返回报名中心查看进度。</p><RouterLink to="/apply" class="primary-btn">返回报名中心</RouterLink></div><form v-else @submit.prevent="submit"><p class="step-no">SECURE / RESET</p><h1>重置查询密码</h1><label>新密码<input v-model="password" type="password" minlength="8" required autocomplete="new-password"/></label><label>再次输入<input v-model="confirmPassword" type="password" minlength="8" required autocomplete="new-password"/></label><p v-if="error" class="form-message error">{{ error }}</p><button class="primary-btn" :disabled="loading">{{ loading?'正在更新…':'确认更新' }}</button></form></main></div></template>

