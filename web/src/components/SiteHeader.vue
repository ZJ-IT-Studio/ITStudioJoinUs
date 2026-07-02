<script setup lang="ts">
import { Menu, X, ArrowUpRight } from 'lucide-vue-next'
import { ref } from 'vue'
const open = ref(false)
</script>

<template>
  <header class="site-header">
    <RouterLink to="/" class="brand" aria-label="IT Studio 首页"><span>IT</span><strong>STUDIO</strong><small>校园招新</small></RouterLink>
    <button class="menu-toggle" :aria-expanded="open" aria-label="切换导航" @click="open=!open"><X v-if="open"/><Menu v-else/></button>
    <nav :class="{open}">
      <RouterLink to="/#about" @click="open=false">ABOUT</RouterLink>
      <RouterLink to="/#directions" @click="open=false">DIRECTIONS</RouterLink>
      <RouterLink to="/#process" @click="open=false">PROCESS</RouterLink>
      <RouterLink to="/apply" class="nav-apply" @click="open=false">报名 / 查询 <ArrowUpRight :size="16"/></RouterLink>
    </nav>
  </header>
</template>

