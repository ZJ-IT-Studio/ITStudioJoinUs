<script setup lang="ts">
import { SwitchRoot, SwitchThumb } from 'reka-ui'
defineProps<{ disabled?:boolean; label:string }>()
const model = defineModel<boolean>({ default:false })
</script>

<template>
  <SwitchRoot v-model="model" class="ui-switch" :disabled="disabled" :aria-label="label">
    <SwitchThumb class="ui-switch-thumb"/>
  </SwitchRoot>
</template>

