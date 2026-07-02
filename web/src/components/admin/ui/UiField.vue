<script setup lang="ts">
import { Label } from 'reka-ui'
defineProps<{ id:string; label:string; hint?:string; required?:boolean; wide?:boolean }>()
</script>
<template><div :class="['ui-field',{wide}]"><Label :for="id">{{ label }}<b v-if="required">*</b></Label><slot/><small v-if="hint">{{ hint }}</small></div></template>

