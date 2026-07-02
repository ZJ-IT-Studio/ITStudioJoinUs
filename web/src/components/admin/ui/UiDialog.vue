<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
withDefaults(defineProps<{ title:string; description?:string; size?:'sm'|'md'|'lg' }>(),{size:'md'})
const open = defineModel<boolean>('open',{default:false})
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="ui-dialog-overlay"/>
      <DialogContent :class="['ui-dialog-content',`is-${size}`]">
        <header class="ui-dialog-header">
          <div><span>IT STUDIO / SETTINGS</span><DialogTitle>{{ title }}</DialogTitle><DialogDescription v-if="description">{{ description }}</DialogDescription></div>
          <DialogClose class="ui-icon-button" aria-label="关闭弹窗"><X :size="18"/></DialogClose>
        </header>
        <div class="ui-dialog-body"><slot/></div>
        <footer v-if="$slots.footer" class="ui-dialog-footer"><slot name="footer"/></footer>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

