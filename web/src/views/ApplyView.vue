<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { ArrowLeft, ArrowRight, Check, Eye, EyeOff, LogOut, RotateCcw } from 'lucide-vue-next'
import SiteHeader from '../components/SiteHeader.vue'
import DynamicForm from '../components/DynamicForm.vue'
import CampaignSelect from '../components/CampaignSelect.vue'
import { ApiError, api, post } from '../api'
import type { ApplicationDetail, Campaign, FormField } from '../types'

type Phase='lookup'|'login'|'verify'|'form'|'status'
const campaigns=ref<Campaign[]>([]), selectedId=ref(0), phase=ref<Phase>('lookup'), loading=ref(false)
const studentId=ref(''), password=ref(''), email=ref(''), code=ref(''), verificationToken=ref('')
const fields=ref<FormField[]>([]), answers=ref<Record<string,unknown>>({}), files=ref<Record<string,File|undefined>>({})
const detail=ref<ApplicationDetail|null>(null), error=ref(''), notice=ref(''), showPassword=ref(false), resubmitting=ref(false), showWithdrawModal=ref(false)
const selected=computed(()=>campaigns.value.find(c=>c.id===selectedId.value))
function accepting(c?:Campaign){if(!c||c.status!=='open')return false;const now=Date.now();return(!c.startsAt||now>=new Date(c.startsAt).getTime())&&(!c.endsAt||now<=new Date(c.endsAt).getTime())}
const isOpen=computed(()=>accepting(selected.value))
watch(phase,async()=>{await nextTick();window.scrollTo({top:0,behavior:'auto'})})

onMounted(async()=>{
  campaigns.value=(await api<{campaigns:Campaign[]}>('/campaigns')).campaigns
  selectedId.value=(campaigns.value.find(c=>c.status==='open')||campaigns.value[0])?.id||0
  try{ detail.value=await api<ApplicationDetail>('/student/application');phase.value='status';selectedId.value=detail.value.campaign.id }catch{/* no active student session */}
})
function setError(e:unknown){error.value=e instanceof ApiError?e.message:'发生未知错误，请重试'}
async function lookup(){error.value='';notice.value='';if(!selectedId.value)return;loading.value=true;try{const r=await post<{exists:boolean}>('/applications/lookup',{campaignId:selectedId.value,studentId:studentId.value});phase.value=r.exists?'login':'verify';if(!r.exists&&!isOpen.value){error.value='该批次已结束，仅已有报名者可以查询';phase.value='lookup'}}catch(e){setError(e)}finally{loading.value=false}}
async function sendCode(){error.value='';loading.value=true;try{const r=await post<{sent:boolean;devCode?:string}>('/email/send-verification',{campaignId:selectedId.value,studentId:studentId.value,email:email.value});notice.value=r.devCode?`开发模式验证码：${r.devCode}`:'验证码已发送，请检查邮箱。'}catch(e){setError(e)}finally{loading.value=false}}
async function verify(){error.value='';loading.value=true;try{const r=await post<{verificationToken:string}>('/email/verify',{campaignId:selectedId.value,studentId:studentId.value,email:email.value,code:code.value});verificationToken.value=r.verificationToken;await loadForm();phase.value='form'}catch(e){setError(e)}finally{loading.value=false}}
async function login(){error.value='';loading.value=true;try{await post('/student/login',{campaignId:selectedId.value,studentId:studentId.value,password:password.value});await loadDetail()}catch(e){setError(e)}finally{loading.value=false}}
async function loadForm(){fields.value=(await api<{fields:FormField[]}>(`/campaigns/${selectedId.value}/form`)).fields}
async function loadDetail(){detail.value=await api<ApplicationDetail>('/student/application');phase.value='status';selectedId.value=detail.value.campaign.id}
async function submit(){error.value='';loading.value=true;try{const payload={campaignId:selectedId.value,studentId:studentId.value,email:email.value,password:password.value,verificationToken:verificationToken.value,answers:answers.value};const form=new FormData();form.set('payload',JSON.stringify(payload));Object.entries(files.value).forEach(([key,file])=>{if(file)form.set(`file_${key}`,file)});await api(resubmitting.value?'/student/resubmit':'/applications',{method:'POST',body:form});resubmitting.value=false;await loadDetail()}catch(e){setError(e)}finally{loading.value=false}}
async function confirmWithdraw(){showWithdrawModal.value=false;loading.value=true;try{await post('/student/withdraw');await loadDetail()}catch(e){setError(e)}finally{loading.value=false}}
async function startResubmit(){if(!detail.value)return;await loadForm();answers.value=Object.fromEntries(detail.value.answers.map(a=>[a.key,a.value]));files.value={};resubmitting.value=true;phase.value='form'}
async function resetRequest(){error.value='';loading.value=true;try{const r=await post<{message:string}>('/password/request-reset',{campaignId:selectedId.value,studentId:studentId.value});notice.value=r.message}catch(e){setError(e)}finally{loading.value=false}}
async function logout(){await post('/student/logout');detail.value=null;phase.value='lookup';password.value='';notice.value=''}
function back(){error.value='';notice.value='';phase.value=phase.value==='form'&&resubmitting.value?'status':'lookup';resubmitting.value=false}
</script>

<template>
  <div class="portal-page">
    <SiteHeader />
    <main class="portal-main">
      <aside class="portal-aside"><p>APPLICATION<br/>TERMINAL</p><div><span>SECURE CHANNEL</span><i/><span>SESSION / 2026</span></div><h1>加入<br/>创造者<br/>序列。</h1><small>IT STUDIO / CAMPUS RECRUITMENT</small></aside>
      <section class="portal-panel">
        <div class="portal-top"><span>{{ phase.toUpperCase() }} / {{ selected?.name || 'NO CAMPAIGN' }}</span><button v-if="phase!=='lookup'&&phase!=='status'" @click="back"><ArrowLeft/> 返回</button><button v-if="phase==='status'" @click="logout"><LogOut/> 退出查询</button></div>

        <div v-if="phase==='lookup'" class="portal-content compact">
          <p class="step-no">STEP 00 / ENTRY</p><h2>报名，或查询进度。</h2><p class="lead">选择招新批次，输入学号。系统会带你去正确的下一步。</p>
          <form @submit.prevent="lookup">
            <label>招新批次<CampaignSelect v-model="selectedId" :campaigns="campaigns" /></label>
            <label>学号<input v-model.trim="studentId" required minlength="4" maxlength="32" autocomplete="username" placeholder="输入你的学号"/></label>
            <button class="primary-btn" :disabled="loading||!campaigns.length">{{ loading?'查询中…':'继续' }} <ArrowRight/></button>
          </form><p v-if="!campaigns.length" class="empty-hint">暂无公开招新批次，请稍后再来。</p>
        </div>

        <div v-else-if="phase==='login'" class="portal-content compact">
          <p class="step-no">WELCOME BACK / QUERY</p><h2>找到你的报名记录。</h2><p class="lead">使用提交报名时设置的查询密码继续。</p>
          <form @submit.prevent="login"><label>学号<input :value="studentId" disabled/></label><label>查询密码<div class="password-field"><input v-model="password" :type="showPassword?'text':'password'" required autocomplete="current-password"/><button type="button" @click="showPassword=!showPassword"><EyeOff v-if="showPassword"/><Eye v-else/></button></div></label><button class="primary-btn" :disabled="loading">登录并查看 <ArrowRight/></button></form>
          <button class="text-btn" @click="resetRequest">忘记查询密码？发送重置邮件</button>
        </div>

        <div v-else-if="phase==='verify'" class="portal-content compact">
          <p class="step-no">STEP 01 / VERIFY</p><h2>先确认你的邮箱。</h2><p class="lead">它只用于身份验证与找回查询密码，不发送营销邮件。</p>
          <form @submit.prevent="verify"><label>学号<input :value="studentId" disabled/></label><label>邮箱<div class="input-action"><input v-model.trim="email" type="email" required placeholder="name@example.com"/><button type="button" :disabled="loading||!email" @click="sendCode">发送验证码</button></div></label><label>6 位验证码<input v-model.trim="code" inputmode="numeric" maxlength="6" required placeholder="000000"/></label><label>设置查询密码<div class="password-field"><input v-model="password" :type="showPassword?'text':'password'" minlength="8" required autocomplete="new-password" placeholder="至少 8 位"/><button type="button" @click="showPassword=!showPassword"><EyeOff v-if="showPassword"/><Eye v-else/></button></div></label><button class="primary-btn" :disabled="loading">验证并填写报名表 <ArrowRight/></button></form>
        </div>

        <div v-else-if="phase==='form'" class="portal-content form-content">
          <p class="step-no">{{ resubmitting?'REVISION / RESUBMIT':'STEP 02 / PROFILE' }}</p><h2>{{ resubmitting?'重新整理你的表达。':'让我们认识你。' }}</h2><p class="lead">表单提交后将锁定；开放期内可以撤回并重新提交。</p>
          <form @submit.prevent="submit"><DynamicForm :fields="fields" :model="answers" :files="files" @update:model="answers=$event" @update:files="files=$event"/><label class="agreement"><input type="checkbox" required/> 我确认所填内容真实，并理解提交后的查询与撤回规则。</label><button class="primary-btn submit-btn" :disabled="loading">{{ loading?'正在安全提交…':resubmitting?'重新提交':'提交报名' }} <Check/></button></form>
        </div>

        <div v-else-if="phase==='status'&&detail" class="portal-content status-content">
          <p class="step-no">APPLICATION / REV.{{ String(detail.application.revision).padStart(2,'0') }}</p><div class="status-hero"><div><span>当前进度</span><h2>{{ detail.application.reviewStatus?.name || '待处理' }}</h2><p>{{ detail.application.reviewStatus?.description }}</p></div><i :style="{background:detail.application.reviewStatus?.color}"/></div>
          <div class="status-meta"><div><span>学号</span><b>{{ detail.application.studentId }}</b></div><div><span>提交时间</span><b>{{ new Date(detail.application.submittedAt).toLocaleString('zh-CN') }}</b></div><div><span>系统状态</span><b>{{ detail.application.systemStatus==='submitted'?'已提交':'已撤回' }}</b></div></div>
          <div class="answer-review"><h3>提交内容</h3><dl><template v-for="a in detail.answers" :key="a.key"><dt>{{ a.label }}</dt><dd>{{ Array.isArray(a.value)?a.value.join('、'):a.value }}</dd></template><template v-for="u in detail.uploads" :key="u.id"><dt>{{ u.label }}</dt><dd><a :href="`/api/v1/student/uploads/${u.id}`" target="_blank">{{ u.name }}</a></dd></template></dl></div>
          <div class="status-actions"><button v-if="detail.canWithdraw" class="danger-btn" :disabled="loading" @click="showWithdrawModal=true">撤回报名</button><button v-if="detail.canResubmit" class="primary-btn" @click="startResubmit"><RotateCcw/> 修改并重新提交</button></div>
        </div>

        <Teleport to="body">
          <Transition name="modal">
            <div v-if="showWithdrawModal" class="modal-overlay" @click.self="showWithdrawModal=false">
              <div class="modal-dialog">
                <h3>确认撤回</h3>
                <p>确定撤回当前报名吗？开放期内可以重新填写并提交。</p>
                <div class="modal-actions">
                  <button class="modal-btn-cancel" @click="showWithdrawModal=false">取消</button>
                  <button class="modal-btn-danger" @click="confirmWithdraw">确认撤回</button>
                </div>
              </div>
            </div>
          </Transition>
        </Teleport>

        <p v-if="error" class="form-message error" role="alert">{{ error }}</p><p v-if="notice" class="form-message notice" role="status">{{ notice }}</p>
      </section>
    </main>
  </div>
</template>
