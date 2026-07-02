<script setup lang="ts">
// Local copy-in component following Vue Bits' animated-text composition pattern.
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
const props = withDefaults(defineProps<{ text:string; speed?:number; characters?:string; animateOn?:'mount'|'hover' }>(), { speed:28, characters:'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#@%&', animateOn:'mount' })
const shown = ref(props.text)
let timer:number|undefined
const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
function play(){
  if (reduce) { shown.value=props.text; return }
  window.clearInterval(timer); let index=0
  timer=window.setInterval(()=>{ shown.value=props.text.split('').map((ch,i)=> ch===' ' ? ' ' : i<index ? ch : props.characters[Math.floor(Math.random()*props.characters.length)]).join(''); index+=0.7; if(index>=props.text.length){shown.value=props.text;window.clearInterval(timer)} },props.speed)
}
onMounted(()=>{ if(props.animateOn==='mount') play() })
onBeforeUnmount(()=>window.clearInterval(timer))
watch(()=>props.text,play)
</script>

<template><span class="decrypted" @mouseenter="animateOn==='hover' && play()">{{ shown }}</span></template>

