<script setup lang="ts">
import { Check, ChevronDown } from 'lucide-vue-next'
import { computed } from 'vue'
import { SelectContent, SelectItem, SelectItemIndicator, SelectItemText, SelectPortal, SelectRoot, SelectTrigger, SelectViewport } from 'reka-ui'

type Option = { label:string; value:string|number; meta?:string }
const props=defineProps<{ options:Option[]; placeholder?:string; disabled?:boolean; ariaLabel?:string }>()
const model = defineModel<string|number>()
const selectedLabel=computed(()=>props.options.find(option=>option.value===model.value)?.label || props.placeholder || '请选择')
</script>

<template>
  <SelectRoot v-model="model" :disabled="disabled">
    <SelectTrigger class="ui-select-trigger" :aria-label="ariaLabel">
      <span :class="{'is-placeholder':model===undefined||model===null||model===''}">{{ selectedLabel }}</span>
      <ChevronDown :size="15" aria-hidden="true"/>
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="ui-select-content" position="popper" :side-offset="6" :collision-padding="12">
        <SelectViewport class="ui-select-viewport">
          <SelectItem v-for="option in options" :key="option.value" class="ui-select-item" :value="option.value">
            <SelectItemText><span>{{ option.label }}</span><small v-if="option.meta">{{ option.meta }}</small></SelectItemText>
            <SelectItemIndicator class="ui-select-check"><Check :size="14"/></SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
