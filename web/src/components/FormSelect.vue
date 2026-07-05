<script setup lang="ts">
// 带过渡动画的下拉选择器，用于动态表单
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

const props = defineProps<{ options: string[]; modelValue: string; placeholder?: string; id?: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const open = ref(false)
const el = ref<HTMLElement | null>(null)

function select(v: string) { emit('update:modelValue', v); open.value = false }
function toggle() { open.value = !open.value }
function onOutside(e: MouseEvent) { if (el.value && !el.value.contains(e.target as Node)) open.value = false }
onMounted(() => document.addEventListener('mousedown', onOutside))
onBeforeUnmount(() => document.removeEventListener('mousedown', onOutside))
</script>

<template>
  <div ref="el" class="form-select" :id="id">
    <button class="fs-trigger" @click="toggle" type="button">
      <span :class="{ placeholder: !modelValue }">{{ modelValue || (placeholder || '请选择') }}</span>
      <ChevronDown :class="['fs-icon', { 'fs-icon-open': open }]" :size="16" />
    </button>
    <Transition name="fs-drop">
      <ul v-if="open" class="fs-list">
        <li v-for="opt in options" :key="opt" :class="{ active: modelValue === opt }" @click="select(opt)">{{ opt }}</li>
      </ul>
    </Transition>
  </div>
</template>

<style scoped>
.form-select { position: relative; }
.fs-trigger {
  width: 100%; height: 48px; padding: 0 14px;
  border: 0; border-bottom: 1px solid var(--ink);
  background: transparent; font-size: 15px;
  display: flex; align-items: center; justify-content: space-between;
  cursor: pointer; outline: none; text-align: left;
  transition: border-color .2s;
}
.fs-trigger:hover, .form-select:focus-within .fs-trigger { border-bottom: 3px solid var(--acid); }
.fs-trigger .placeholder { color: var(--muted); }
.fs-icon { transition: transform .25s; flex-shrink: 0; }
.fs-icon-open { transform: rotate(180deg); }

.fs-list {
  position: absolute; top: 100%; left: 0; right: 0; z-index: 50;
  background: #fff; border: 1px solid var(--ink);
  list-style: none; margin: 0; padding: 4px; max-height: 210px; overflow-y: auto;
}
.fs-list li {
  padding: 11px 14px; cursor: pointer; font-size: 14px;
  transition: background .15s;
}
.fs-list li:hover, .fs-list li.active { background: rgba(197,232,1,.2); }

.fs-drop-enter-active { transition: opacity .2s, transform .2s; }
.fs-drop-leave-active { transition: opacity .15s, transform .15s; }
.fs-drop-enter-from { opacity: 0; transform: translateY(-6px); }
.fs-drop-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
