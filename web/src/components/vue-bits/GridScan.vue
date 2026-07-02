<script setup lang="ts">
// A lightweight Vue Bits-style animated background, kept CSS-native for mobile performance.
withDefaults(defineProps<{ intensity?:number }>(), { intensity:.6 })
</script>
<template><div class="grid-scan" aria-hidden="true" :style="{'--intensity':intensity}"><i/><i/><i/></div></template>

