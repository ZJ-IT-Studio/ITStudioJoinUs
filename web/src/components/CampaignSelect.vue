<script setup lang="ts">
// 招新批次下拉选择器，带展开/收起过渡动画
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import type { Campaign } from '../types'

defineProps<{ campaigns: Campaign[]; modelValue: number }>()
const emit = defineEmits<{ 'update:modelValue': [id: number] }>()

const open = ref(false)
const el = ref<HTMLElement | null>(null)

function select(id: number) {
  emit('update:modelValue', id)
  open.value = false
}

function accepting(c: Campaign) {
  if (!c || c.status !== 'open') return false
  const now = Date.now()
  return (!c.startsAt || now >= new Date(c.startsAt).getTime()) && (!c.endsAt || now <= new Date(c.endsAt).getTime())
}

function toggle() { open.value = !open.value }

// 点击组件外部时关闭下拉
function onOutsideClick(e: MouseEvent) {
  if (el.value && !el.value.contains(e.target as Node)) {
    open.value = false
  }
}
onMounted(() => document.addEventListener('mousedown', onOutsideClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onOutsideClick))
</script>

<template>
  <div ref="el" class="campaign-select">
    <button class="campaign-select-trigger" @click="toggle" type="button">
      <span>{{ campaigns.find(c => c.id === modelValue)?.name || '请选择批次' }}</span>
      <ChevronDown :class="['cs-icon', { 'cs-icon-open': open }]" :size="16" />
    </button>
    <Transition name="cs-drop">
      <ul v-if="open" class="campaign-select-drop">
        <li v-for="c in campaigns" :key="c.id" :class="{ active: c.id === modelValue }" @click="select(c.id)">
          <span>{{ c.name }}</span>
          <small :class="accepting(c) ? 'text-acid' : ''">{{ accepting(c) ? '● 开放中' : '已结束' }}</small>
        </li>
      </ul>
    </Transition>
  </div>
</template>

<style scoped>
.campaign-select { position: relative; outline: none; }
.campaign-select-trigger {
  width: 100%; height: 52px; padding: 0 16px;
  border: 1px solid var(--ink); background: transparent;
  display: flex; align-items: center; justify-content: space-between;
  font-size: 15px; cursor: pointer;
  transition: border-color .2s;
}
.campaign-select-trigger:hover { border-color: var(--acid); }
.cs-icon { transition: transform .25s; }
.cs-icon-open { transform: rotate(180deg); }

.campaign-select-drop {
  position: absolute; top: 100%; left: 0; right: 0; z-index: 50;
  background: #fff; border: 1px solid var(--ink); border-top: 0;
  list-style: none; margin: 0; padding: 4px; max-height: 220px; overflow-y: auto;
}
.campaign-select-drop li {
  padding: 12px 14px; cursor: pointer; display: flex; justify-content: space-between; align-items: center;
  font-size: 14px; transition: background .15s;
}
.campaign-select-drop li:hover, .campaign-select-drop li.active {
  background: rgba(197,232,1,.2);
}
.campaign-select-drop small { font-size: 11px; color: var(--muted); }
.campaign-select-drop .text-acid { color: var(--acid); font-weight: 700; }

/* 下拉过渡动画 */
.cs-drop-enter-active { transition: opacity .2s, transform .2s; }
.cs-drop-leave-active { transition: opacity .15s, transform .15s; }
.cs-drop-enter-from { opacity: 0; transform: translateY(-8px); }
.cs-drop-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
