<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Activity, Archive, ChevronRight, ClipboardList, FileText, Gauge, LogOut, Menu, Plus, Settings, Shield, Users, X } from 'lucide-vue-next'
import { ApiError, api, del, post, put } from '../../api'
import { useSessionStore } from '../../stores/session'
import type { ApplicationDetail, Application, Campaign, FormField, ReviewStatus, SiteContent } from '../../types'
import UiDialog from '../../components/admin/ui/UiDialog.vue'
import UiField from '../../components/admin/ui/UiField.vue'
import UiSelect from '../../components/admin/ui/UiSelect.vue'
import UiSwitch from '../../components/admin/ui/UiSwitch.vue'

type Tab='dashboard'|'applications'|'campaigns'|'content'|'admins'|'audit'
const router=useRouter(),session=useSessionStore(),tab=ref<Tab>('dashboard'),busy=ref(false),error=ref(''),menuOpen=ref(false)
const dashboard=ref<any>(null),campaigns=ref<Campaign[]>([]),applications=ref<Application[]>([]),content=ref<SiteContent|null>(null),admins=ref<any[]>([]),logs=ref<any[]>([])
const selectedCampaign=ref(0),statuses=ref<ReviewStatus[]>([]),fields=ref<FormField[]>([]),detail=ref<ApplicationDetail|null>(null),search=ref(''),note=ref('')
const fieldDraft=ref<FormField|null>(null),statusDraft=ref<ReviewStatus|null>(null)
const fieldOpen=ref(false),statusOpen=ref(false),campaignOpen=ref(false),userOpen=ref(false),confirmOpen=ref(false)
const campaignMode=ref<'create'|'edit'|'clone'>('create')
const campaignDraft=ref({name:'',slug:'',startsAt:'',endsAt:''})
const userDraft=ref({email:'',password:'',role:'owner'})
const confirmDraft=ref({title:'',description:''}),pendingAction=ref<null|(()=>Promise<void>)>(null)
const owner=computed(()=>session.isOwner), superAdmin=computed(()=>session.isSuperAdmin), selectedCampaignData=computed(()=>campaigns.value.find(c=>c.id===selectedCampaign.value))
const campaignOptions=computed(()=>campaigns.value.map(c=>({label:c.name,value:c.id,meta:c.status.toUpperCase()})))
const statusOptions=computed(()=>statuses.value.map(s=>({label:s.name,value:s.id,meta:s.isDefault?'默认状态':''})))
const fieldTypeOptions=[{label:'单行文本',value:'text'},{label:'多行文本',value:'textarea'},{label:'数字',value:'number'},{label:'日期',value:'date'},{label:'链接',value:'url'},{label:'单选',value:'radio'},{label:'多选',value:'checkbox'},{label:'下拉选择',value:'select'},{label:'图片',value:'image'}]
const roleOptions=[{label:'可编辑',value:'owner',meta:'可修改业务数据'},{label:'仅查看',value:'readonly',meta:'只读与导出'}]
const nav:[Tab,string,any][]=[['dashboard','总览',Gauge],['applications','报名管理',ClipboardList],['campaigns','批次与表单',Archive],['content','首页文案',FileText],['admins','用户管理',Users],['audit','审计日志',Activity]]

onMounted(async()=>{await session.checkAdmin();if(!session.admin){router.replace('/admin/login');return}await loadBase();await loadTab()})
watch(tab,()=>{menuOpen.value=false;error.value='';detail.value=null;loadTab()})
async function loadBase(){campaigns.value=(await api<{campaigns:Campaign[]}>('/admin/campaigns')).campaigns;selectedCampaign.value=selectedCampaign.value||campaigns.value[0]?.id||0}
async function loadTab(){try{if(tab.value==='dashboard')dashboard.value=await api('/admin/dashboard');if(tab.value==='applications')await loadApplications();if(tab.value==='campaigns')await loadCampaignConfig();if(tab.value==='content')content.value=(await api<{content:SiteContent}>('/admin/content')).content;if(tab.value==='admins'&&superAdmin.value)admins.value=(await api<{admins:any[]}>('/admin/admins')).admins;if(tab.value==='audit')logs.value=(await api<{logs:any[]}>('/admin/audit')).logs}catch(e){setError(e)}}
function setError(e:unknown){error.value=e instanceof ApiError?e.message:'操作失败，请重试'}
async function act(fn:()=>Promise<void>){error.value='';busy.value=true;try{await fn()}catch(e){setError(e)}finally{busy.value=false}}
async function logout(){await session.logout();router.replace('/admin/login')}
async function loadApplications(){applications.value=(await api<{applications:Application[]}>(`/admin/applications?campaignId=${selectedCampaign.value}&q=${encodeURIComponent(search.value)}`)).applications;statuses.value=(await api<{statuses:ReviewStatus[]}>(`/admin/campaigns/${selectedCampaign.value}/statuses`)).statuses}
async function loadCampaignConfig(){if(!selectedCampaign.value)return;[fields.value,statuses.value]=await Promise.all([api<{fields:FormField[]}>(`/admin/campaigns/${selectedCampaign.value}/fields`).then(r=>r.fields),api<{statuses:ReviewStatus[]}>(`/admin/campaigns/${selectedCampaign.value}/statuses`).then(r=>r.statuses)])}
async function openApplication(id:number){detail.value=await api<ApplicationDetail>(`/admin/applications/${id}`)}
async function changeStatus(statusId:number){if(!detail.value)return;await act(async()=>{await post(`/admin/applications/${detail.value!.application.id}/status`,{statusId});await openApplication(detail.value!.application.id);await loadApplications()})}
async function addNote(){if(!detail.value||!note.value.trim())return;await act(async()=>{await post(`/admin/applications/${detail.value!.application.id}/notes`,{content:note.value});note.value='';await openApplication(detail.value!.application.id)})}
function exportCsv(){window.location.href=`/api/v1/admin/export?campaignId=${selectedCampaign.value}`}
function localDate(value?:string){if(!value)return '';const d=new Date(value),offset=d.getTimezoneOffset()*60000;return new Date(d.getTime()-offset).toISOString().slice(0,16)}
function openCampaignDialog(mode:'create'|'edit'|'clone'){campaignMode.value=mode;const c=selectedCampaignData.value;campaignDraft.value=mode==='edit'&&c?{name:c.name,slug:c.slug,startsAt:localDate(c.startsAt),endsAt:localDate(c.endsAt)}:{name:mode==='clone'&&c?`${c.name} 副本`:'',slug:'',startsAt:'',endsAt:''};campaignOpen.value=true}
async function saveCampaign(){const d=campaignDraft.value;if(!d.name.trim()||!d.slug.trim())return;await act(async()=>{if(campaignMode.value==='create')await post('/admin/campaigns',{name:d.name,slug:d.slug});else if(campaignMode.value==='clone')await post(`/admin/campaigns/${selectedCampaign.value}/clone`,{name:d.name,slug:d.slug});else await put(`/admin/campaigns/${selectedCampaign.value}`,{name:d.name,slug:d.slug,startsAt:d.startsAt?new Date(d.startsAt).toISOString():null,endsAt:d.endsAt?new Date(d.endsAt).toISOString():null});campaignOpen.value=false;await loadBase();await loadCampaignConfig()})}
async function toggleCampaign(){const c=selectedCampaignData.value;if(!c)return;await act(async()=>{if(c.status==='open')await post(`/admin/campaigns/${c.id}/close`);else await post(`/admin/campaigns/${c.id}/open`);await loadBase();await loadCampaignConfig()})}
function askConfirm(title:string,description:string,action:()=>Promise<void>){confirmDraft.value={title,description};pendingAction.value=action;confirmOpen.value=true}
async function runConfirmed(){if(!pendingAction.value)return;await act(pendingAction.value);confirmOpen.value=false;pendingAction.value=null}
function archiveCampaign(){const c=selectedCampaignData.value;if(!c)return;askConfirm('归档招新批次',`“${c.name}”归档后仍可查询历史报名，但不能再次开放。`,async()=>{await post(`/admin/campaigns/${c.id}/archive`);await loadBase();await loadCampaignConfig()})}
function editField(field?:FormField){fieldDraft.value=field?JSON.parse(JSON.stringify(field)):{id:0,key:'',label:'',type:'text',required:false,placeholder:'',helpText:'',options:[],position:(fields.value.length+1)*10,validation:{}};fieldOpen.value=true}
async function saveField(){if(!fieldDraft.value)return;await act(async()=>{await post(`/admin/campaigns/${selectedCampaign.value}/fields`,fieldDraft.value);fieldOpen.value=false;await loadCampaignConfig()})}
function removeField(id:number){askConfirm('删除表单字段','该操作会立即修改尚未开放的报名表单。',async()=>{await del(`/admin/campaigns/${selectedCampaign.value}/fields/${id}`);await loadCampaignConfig()})}
function editStatus(status?:ReviewStatus){statusDraft.value=status?JSON.parse(JSON.stringify(status)):{id:0,name:'',color:'#c5e801',description:'',position:(statuses.value.length+1)*10,isDefault:false};statusOpen.value=true}
async function saveStatus(){if(!statusDraft.value)return;await act(async()=>{await post(`/admin/campaigns/${selectedCampaign.value}/statuses`,statusDraft.value);statusOpen.value=false;await loadCampaignConfig()})}
function removeStatus(id:number){askConfirm('删除审核状态','已被报名引用的状态需要先迁移，系统会阻止误删。',async()=>{await del(`/admin/campaigns/${selectedCampaign.value}/statuses/${id}`);await loadCampaignConfig()})}
async function saveContent(){if(!content.value)return;await act(async()=>{await put('/admin/content',content.value!);alert('首页文案已保存')})}
function createAdmin(){userDraft.value={email:'',password:'',role:'owner'};userOpen.value=true}
async function saveAdmin(){if(!userDraft.value.email||userDraft.value.password.length<8)return;await act(async()=>{await post('/admin/admins',userDraft.value);userOpen.value=false;admins.value=(await api<{admins:any[]}>('/admin/admins')).admins})}
async function updateAdmin(item:any){await act(async()=>{await put(`/admin/admins/${item.id}`,{role:item.role,active:item.active});admins.value=(await api<{admins:any[]}>('/admin/admins')).admins})}
</script>

<template>
  <div v-if="session.admin" class="admin-shell">
    <aside :class="['admin-sidebar',{open:menuOpen}]"><div class="admin-brand"><div class="brand"><span>IT</span><strong>STUDIO</strong></div><button @click="menuOpen=false"><X/></button></div><p class="admin-role"><Shield/> {{ superAdmin?'SUPER ADMIN':session.admin.role==='owner'?'EDIT ACCESS':'VIEW ONLY' }}</p><nav><button v-for="[key,label,Icon] in nav" v-show="key!=='admins'||superAdmin" :key="key" :class="{active:tab===key}" @click="tab=key"><component :is="Icon"/>{{ label }}<ChevronRight/></button></nav><div class="admin-user"><span>{{ session.admin.email }}</span><button @click="logout"><LogOut/> 退出</button></div></aside>
    <main class="admin-main">
      <header class="admin-topbar"><button class="admin-menu" @click="menuOpen=true"><Menu/></button><div><span>IT STUDIO / OPERATIONS</span><h1>{{ nav.find(n=>n[0]===tab)?.[1] }}</h1></div><time>{{ new Date().toLocaleDateString('zh-CN') }}</time></header>
      <p v-if="error" class="form-message error admin-error">{{ error }}</p>

      <section v-if="tab==='dashboard'&&dashboard" class="admin-section">
        <div class="metric-grid"><article><span>CAMPAIGNS</span><b>{{ dashboard.totals.campaigns }}</b><p>累计招新批次</p></article><article><span>APPLICATIONS</span><b>{{ dashboard.totals.applications }}</b><p>累计提交版本</p></article><article class="accent"><span>ACTIVE</span><b>{{ dashboard.totals.active }}</b><p>当前有效报名</p></article></div>
        <div class="admin-card"><div class="card-heading"><h2>批次概览</h2><span>LIVE DATABASE</span></div><table><thead><tr><th>批次</th><th>全部版本</th><th>有效报名</th></tr></thead><tbody><tr v-for="c in dashboard.campaigns" :key="c.id"><td>{{ c.name }}</td><td>{{ c.total }}</td><td>{{ c.active }}</td></tr></tbody></table></div>
      </section>

      <section v-if="tab==='applications'" class="admin-section">
        <div class="toolbar modern-toolbar"><UiSelect v-model="selectedCampaign" :options="campaignOptions" aria-label="选择招新批次" @update:model-value="loadApplications"/><input v-model="search" class="ui-input toolbar-search" placeholder="搜索学号或邮箱" @keyup.enter="loadApplications"/><button @click="loadApplications">查询</button><button @click="exportCsv">导出 CSV</button></div>
        <div class="split-admin"><div class="admin-card list-card"><div class="card-heading"><h2>报名列表</h2><span>{{ applications.length }} RECORDS</span></div><button v-for="a in applications" :key="a.id" class="application-row" :class="{selected:detail?.application.id===a.id}" @click="openApplication(a.id)"><div><b>{{ a.studentId }}</b><span>{{ a.email }}</span></div><i :style="{background:a.reviewStatus?.color}"/><span>{{ a.reviewStatus?.name }}</span><small>REV.{{ a.revision }}</small></button><p v-if="!applications.length" class="empty-hint">暂无匹配报名。</p></div>
          <div class="admin-card detail-card"><template v-if="detail"><div class="card-heading"><div><span>APPLICATION DETAIL</span><h2>{{ detail.application.studentId }}</h2></div><b>{{ detail.application.systemStatus==='submitted'?'有效报名':'已撤回' }}</b></div><UiField id="review-status" label="审核状态"><UiSelect :model-value="detail.application.reviewStatus?.id || 0" :options="statusOptions" :disabled="!owner" aria-label="审核状态" @update:model-value="changeStatus(Number($event))"/></UiField><dl class="admin-answers"><template v-for="a in detail.answers" :key="a.key"><dt>{{ a.label }}</dt><dd>{{ Array.isArray(a.value)?a.value.join('、'):a.value }}</dd></template><template v-for="u in detail.uploads" :key="u.id"><dt>{{ u.label }}</dt><dd><a :href="`/api/v1/admin/applications/${detail.application.id}/uploads/${u.id}`" target="_blank">查看 {{ u.name }}</a></dd></template></dl><div class="notes"><h3>内部备注</h3><article v-for="n in detail.notes as any[]" :key="n.id"><p>{{ n.content }}</p><small>{{ n.admin }} · {{ new Date(n.createdAt).toLocaleString('zh-CN') }}</small></article><form v-if="owner" @submit.prevent="addNote"><textarea v-model="note" class="ui-textarea" placeholder="添加仅管理员可见的备注"/><button>添加备注</button></form></div></template><div v-else class="detail-placeholder"><ClipboardList/><p>选择一条报名查看完整内容</p></div></div>
        </div>
      </section>

      <section v-if="tab==='campaigns'" class="admin-section">
        <div class="toolbar modern-toolbar"><UiSelect v-model="selectedCampaign" :options="campaignOptions" aria-label="选择招新批次" @update:model-value="loadCampaignConfig"/><button v-if="owner" @click="openCampaignDialog('create')"><Plus/> 新建批次</button><button v-if="owner" @click="openCampaignDialog('edit')">编辑设置</button><button v-if="owner" @click="openCampaignDialog('clone')">复制批次</button><button v-if="owner&&selectedCampaignData?.status!=='archived'" :class="{danger:selectedCampaignData?.status==='open'}" @click="toggleCampaign">{{ selectedCampaignData?.status==='open'?'关闭招新':'开放招新' }}</button><button v-if="owner&&['draft','closed'].includes(selectedCampaignData?.status||'')" @click="archiveCampaign">归档</button></div>
        <div v-if="selectedCampaignData" class="campaign-banner"><div><span>CAMPAIGN / {{ selectedCampaignData.id }}</span><h2>{{ selectedCampaignData.name }}</h2></div><div><b>{{ selectedCampaignData.status.toUpperCase() }}</b><small>{{ selectedCampaignData.formLocked?'表单结构已永久锁定':'表单仍可编辑' }}</small></div></div>
        <div class="config-grid"><div class="admin-card"><div class="card-heading"><h2>动态表单</h2><button v-if="owner&&!selectedCampaignData?.formLocked" @click="editField()"><Plus/> 字段</button></div><div v-for="f in fields" :key="f.id" class="config-row"><span>{{ String(f.position).padStart(2,'0') }}</span><div><b>{{ f.label }}</b><small>{{ f.key }} / {{ f.type }} {{ f.required?'· 必填':'' }}</small></div><button v-if="owner&&!selectedCampaignData?.formLocked" @click="editField(f)">编辑</button><button v-if="owner&&!selectedCampaignData?.formLocked" @click="removeField(f.id)">删除</button></div></div>
          <div class="admin-card"><div class="card-heading"><h2>审核状态</h2><button v-if="owner" @click="editStatus()"><Plus/> 状态</button></div><div v-for="s in statuses" :key="s.id" class="config-row"><i :style="{background:s.color}"/><div><b>{{ s.name }} <em v-if="s.isDefault">默认</em></b><small>{{ s.description }}</small></div><button v-if="owner" @click="editStatus(s)">编辑</button><button v-if="owner&&!s.isDefault" @click="removeStatus(s.id)">删除</button></div></div></div>
      </section>

      <section v-if="tab==='content'&&content" class="admin-section"><div class="content-editor admin-card"><div class="card-heading"><div><span>FIXED CONTENT SLOTS</span><h2>首页文案</h2></div><button v-if="owner" class="accent-button" :disabled="busy" @click="saveContent">保存并发布</button></div><div class="editor-grid"><label>首屏眉题<input v-model="content.heroEyebrow" :disabled="!owner"/></label><label>首屏主标题<input v-model="content.heroTitle" :disabled="!owner"/></label><label class="wide">首屏副标题<textarea v-model="content.heroSubtitle" :disabled="!owner"/></label><label class="wide">宣言标题<textarea v-model="content.manifestoTitle" :disabled="!owner"/></label><label class="wide">宣言正文<textarea v-model="content.manifestoBody" :disabled="!owner"/></label></div><h3>方向卡片 / 4 SLOTS</h3><div class="repeater-grid"><article v-for="(item,i) in content.directions" :key="i"><input v-model="item.label" :disabled="!owner"/><input v-model="item.title" :disabled="!owner"/><textarea v-model="item.body" :disabled="!owner"/></article></div><h3>价值主张 / 3 SLOTS</h3><div class="repeater-grid"><article v-for="(item,i) in content.values" :key="i"><input v-model="item.label" :disabled="!owner"/><input v-model="item.title" :disabled="!owner"/><textarea v-model="item.body" :disabled="!owner"/></article></div><h3>招新流程 / 4 SLOTS</h3><div class="repeater-grid"><article v-for="(item,i) in content.process" :key="i"><input v-model="item.label" :disabled="!owner"/><input v-model="item.title" :disabled="!owner"/><textarea v-model="item.body" :disabled="!owner"/></article></div><h3>FAQ / 3 SLOTS</h3><div class="repeater-grid"><article v-for="(item,i) in content.faqs" :key="i"><input v-model="item.label" :disabled="!owner"/><input v-model="item.title" :disabled="!owner"/><textarea v-model="item.body" :disabled="!owner"/></article></div><div class="editor-grid"><label>结尾标题<input v-model="content.contactTitle" :disabled="!owner"/></label><label>联系链接<input v-model="content.contactLink" :disabled="!owner"/></label><label class="wide">结尾文案<textarea v-model="content.contactBody" :disabled="!owner"/></label></div></div></section>

      <section v-if="tab==='admins'&&superAdmin" class="admin-section"><div class="admin-card"><div class="card-heading"><div><span>SUPERADMIN ONLY / ACCESS CONTROL</span><h2>后台用户管理</h2></div><button @click="createAdmin"><Plus/> 添加用户</button></div><p class="access-hint">超级管理员由环境变量初始化且不可降级；可编辑用户能处理报名、批次和文案，但不能管理后台用户；仅查看用户不能执行写操作。</p><table><thead><tr><th>邮箱</th><th>权限</th><th>状态</th><th>创建时间</th><th/></tr></thead><tbody><tr v-for="a in admins" :key="a.id"><td>{{ a.email }} <em v-if="a.isSuperAdmin" class="super-badge">超级管理员</em></td><td><UiSelect v-model="a.role" :options="roleOptions" :disabled="a.isSuperAdmin" :aria-label="`${a.email} 的权限`"/></td><td><UiSwitch v-model="a.active" :disabled="a.isSuperAdmin" :label="`${a.email} 的启用状态`"/></td><td>{{ new Date(a.createdAt).toLocaleDateString('zh-CN') }}</td><td><button v-if="!a.isSuperAdmin" @click="updateAdmin(a)">保存</button><span v-else>ENV LOCKED</span></td></tr></tbody></table></div></section>
      <section v-if="tab==='audit'" class="admin-section"><div class="admin-card"><div class="card-heading"><div><span>IMMUTABLE TRAIL</span><h2>最近 300 条操作</h2></div></div><table><thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>对象</th></tr></thead><tbody><tr v-for="l in logs" :key="l.id"><td>{{ new Date(l.createdAt).toLocaleString('zh-CN') }}</td><td>{{ l.actorType }} #{{ l.actorId }}</td><td>{{ l.action }}</td><td>{{ l.entityType }} #{{ l.entityId }}</td></tr></tbody></table></div></section>
    </main>

    <UiDialog v-model:open="campaignOpen" :title="campaignMode==='create'?'新建招新批次':campaignMode==='clone'?'复制招新批次':'批次设置'" description="批次首次开放后，报名表单结构将永久锁定。">
      <div class="ui-form-grid">
        <UiField id="campaign-name" label="批次名称" required><input id="campaign-name" v-model="campaignDraft.name" class="ui-input" placeholder="例如：2027 春季招新"/></UiField>
        <UiField id="campaign-slug" label="英文标识" hint="仅使用小写字母、数字和连字符" required><input id="campaign-slug" v-model="campaignDraft.slug" class="ui-input" placeholder="2027-spring"/></UiField>
        <template v-if="campaignMode==='edit'">
          <UiField id="campaign-start" label="开放时间"><input id="campaign-start" v-model="campaignDraft.startsAt" class="ui-input" type="datetime-local"/></UiField>
          <UiField id="campaign-end" label="截止时间"><input id="campaign-end" v-model="campaignDraft.endsAt" class="ui-input" type="datetime-local"/></UiField>
        </template>
      </div>
      <template #footer><button class="ui-button ghost" @click="campaignOpen=false">取消</button><button class="ui-button primary" :disabled="busy||!campaignDraft.name||!campaignDraft.slug" @click="saveCampaign">保存批次</button></template>
    </UiDialog>

    <UiDialog v-model:open="fieldOpen" :title="fieldDraft?.id?'编辑表单字段':'新增表单字段'" description="字段类型、选项与必填规则会直接影响学生报名表。" size="lg">
      <div v-if="fieldDraft" class="ui-form-grid">
        <UiField id="field-key" label="字段标识" hint="保存后用于答案映射" required><input id="field-key" v-model="fieldDraft.key" class="ui-input" pattern="[a-z0-9-]+" placeholder="portfolio-url"/></UiField>
        <UiField id="field-label" label="显示名称" required><input id="field-label" v-model="fieldDraft.label" class="ui-input" placeholder="例如：作品链接"/></UiField>
        <UiField id="field-type" label="字段类型"><UiSelect v-model="fieldDraft.type" :options="fieldTypeOptions" aria-label="字段类型"/></UiField>
        <UiField id="field-position" label="排序值"><input id="field-position" v-model.number="fieldDraft.position" class="ui-input" type="number"/></UiField>
        <UiField id="field-placeholder" label="占位提示" wide><input id="field-placeholder" v-model="fieldDraft.placeholder" class="ui-input"/></UiField>
        <UiField id="field-help" label="帮助文字" wide><input id="field-help" v-model="fieldDraft.helpText" class="ui-input"/></UiField>
        <UiField v-if="['radio','checkbox','select'].includes(fieldDraft.type)" id="field-options" label="选项" hint="每行一个选项" wide><textarea id="field-options" class="ui-textarea" :value="fieldDraft.options.join('\n')" @input="fieldDraft!.options=($event.target as HTMLTextAreaElement).value.split('\n').filter(Boolean)"/></UiField>
        <div class="ui-setting-row wide"><div><b>必填字段</b><small>学生提交前必须完成此项</small></div><UiSwitch v-model="fieldDraft.required" label="是否必填"/></div>
      </div>
      <template #footer><button class="ui-button ghost" @click="fieldOpen=false">取消</button><button class="ui-button primary" :disabled="busy||!fieldDraft?.key||!fieldDraft?.label" @click="saveField">保存字段</button></template>
    </UiDialog>

    <UiDialog v-model:open="statusOpen" :title="statusDraft?.id?'编辑审核状态':'新增审核状态'" description="状态名称与说明会展示给学生。">
      <div v-if="statusDraft" class="ui-form-grid">
        <UiField id="status-name" label="状态名称" required><input id="status-name" v-model="statusDraft.name" class="ui-input" placeholder="例如：待面试"/></UiField>
        <UiField id="status-color" label="标识颜色"><input id="status-color" v-model="statusDraft.color" class="ui-color" type="color"/></UiField>
        <UiField id="status-description" label="学生可见说明" wide><textarea id="status-description" v-model="statusDraft.description" class="ui-textarea"/></UiField>
        <UiField id="status-position" label="排序值"><input id="status-position" v-model.number="statusDraft.position" class="ui-input" type="number"/></UiField>
        <div class="ui-setting-row"><div><b>默认状态</b><small>新报名自动进入此状态</small></div><UiSwitch v-model="statusDraft.isDefault" label="设为默认状态"/></div>
      </div>
      <template #footer><button class="ui-button ghost" @click="statusOpen=false">取消</button><button class="ui-button primary" :disabled="busy||!statusDraft?.name" @click="saveStatus">保存状态</button></template>
    </UiDialog>

    <UiDialog v-model:open="userOpen" title="添加后台用户" description="超级管理员可以创建可编辑或仅查看账号。">
      <div class="ui-form-grid">
        <UiField id="user-email" label="登录邮箱" wide required><input id="user-email" v-model="userDraft.email" class="ui-input" type="email" autocomplete="off" placeholder="member@example.com"/></UiField>
        <UiField id="user-password" label="初始密码" hint="至少 8 位" wide required><input id="user-password" v-model="userDraft.password" class="ui-input" type="password" autocomplete="new-password"/></UiField>
        <UiField id="user-role" label="账号权限" wide><UiSelect v-model="userDraft.role" :options="roleOptions" aria-label="账号权限"/></UiField>
      </div>
      <template #footer><button class="ui-button ghost" @click="userOpen=false">取消</button><button class="ui-button primary" :disabled="busy||!userDraft.email||userDraft.password.length<8" @click="saveAdmin">创建用户</button></template>
    </UiDialog>

    <UiDialog v-model:open="confirmOpen" :title="confirmDraft.title" :description="confirmDraft.description" size="sm">
      <div class="ui-warning-block">此操作会立即写入系统，并记录到审计日志。</div>
      <template #footer><button class="ui-button ghost" @click="confirmOpen=false">取消</button><button class="ui-button danger" :disabled="busy" @click="runConfirmed">确认操作</button></template>
    </UiDialog>
  </div>
</template>
