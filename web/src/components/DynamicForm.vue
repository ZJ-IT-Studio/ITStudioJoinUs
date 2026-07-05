<script setup lang="ts">
import type { FormField } from '../types'
import FormSelect from './FormSelect.vue'
defineProps<{ fields:FormField[]; model:Record<string,unknown>; files:Record<string,File|undefined> }>()
const emit = defineEmits<{ 'update:model':[value:Record<string,unknown>]; 'update:files':[value:Record<string,File|undefined>] }>()
function setValue(model:Record<string,unknown>, key:string, value:unknown){ emit('update:model',{...model,[key]:value}) }
function setFile(files:Record<string,File|undefined>, key:string, event:Event){ emit('update:files',{...files,[key]:(event.target as HTMLInputElement).files?.[0]}) }
function toggle(model:Record<string,unknown>, key:string, option:string, checked:boolean){ const list=Array.isArray(model[key])?[...(model[key] as string[])]:[]; const next=checked?[...new Set([...list,option])]:list.filter(v=>v!==option);setValue(model,key,next) }
</script>

<template>
  <div class="dynamic-form">
    <div v-for="(field,index) in fields" :key="field.id" class="field-block">
      <label class="field-label" :for="`field-${field.id}`"><span>{{ String(index+1).padStart(2,'0') }}</span>{{ field.label }}<b v-if="field.required">*</b></label>
      <p v-if="field.helpText" class="field-help">{{ field.helpText }}</p>
      <textarea v-if="field.type==='textarea'" :id="`field-${field.id}`" :required="field.required" :placeholder="field.placeholder" :value="model[field.key] as string" @input="setValue(model,field.key,($event.target as HTMLTextAreaElement).value)"/>
      <FormSelect v-else-if="field.type==='select'" :id="`field-${field.id}`" :options="field.options" :model-value="(model[field.key] as string) || ''" :placeholder="field.placeholder || '请选择'" @update:model-value="setValue(model, field.key, $event)" />
      <div v-else-if="field.type==='radio'" class="choice-grid"><label v-for="option in field.options" :key="option"><input type="radio" :name="field.key" :value="option" :checked="model[field.key]===option" :required="field.required" @change="setValue(model,field.key,option)"/><span>{{ option }}</span></label></div>
      <div v-else-if="field.type==='checkbox'" class="choice-grid"><label v-for="option in field.options" :key="option"><input type="checkbox" :value="option" :checked="Array.isArray(model[field.key])&&(model[field.key] as string[]).includes(option)" @change="toggle(model,field.key,option,($event.target as HTMLInputElement).checked)"/><span>{{ option }}</span></label></div>
      <label v-else-if="field.type==='image'" class="file-field"><input :id="`field-${field.id}`" type="file" accept="image/jpeg,image/png,image/webp" :required="field.required&&!files[field.key]" @change="setFile(files,field.key,$event)"/><span>{{ files[field.key]?.name || '选择 JPEG / PNG / WebP（最大 5 MB）' }}</span></label>
      <input v-else :id="`field-${field.id}`" :type="field.type==='text'?'text':field.type" :required="field.required" :placeholder="field.placeholder" :value="model[field.key] as string|number" @input="setValue(model,field.key,($event.target as HTMLInputElement).value)"/>
    </div>
  </div>
</template>

