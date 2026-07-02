<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { animate, stagger } from 'animejs'
import { ArrowDown, ArrowRight, ArrowUpRight } from 'lucide-vue-next'
import SiteHeader from '../components/SiteHeader.vue'
import DecryptedText from '../components/vue-bits/DecryptedText.vue'
import GridScan from '../components/vue-bits/GridScan.vue'
import { api } from '../api'
import type { SiteContent } from '../types'

const content = ref<SiteContent | null>(null)
let observer: IntersectionObserver | undefined
let wheelLocked = false
let wheelTimer: number | undefined
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches

function pageSections() { return [...document.querySelectorAll<HTMLElement>('.public-shell main > section')] }
function onWheel(event: WheelEvent) {
  if (Math.abs(event.deltaY) < 8) return
  const sections = pageSections()
  if (!sections.length) return
  const current = sections.reduce((best, section, index) => Math.abs(section.getBoundingClientRect().top) < Math.abs(sections[best].getBoundingClientRect().top) ? index : best, 0)
  const target = Math.max(0, Math.min(sections.length - 1, current + (event.deltaY > 0 ? 1 : -1)))
  event.preventDefault()
  if (target === current || wheelLocked) return
  wheelLocked = true
  sections[target].scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'start' })
  window.clearTimeout(wheelTimer)
  wheelTimer = window.setTimeout(() => { wheelLocked = false }, 850)
}

onMounted(async () => {
  content.value = (await api<{content:SiteContent}>('/content')).content
  document.documentElement.classList.add('page-snap')
  window.addEventListener('wheel', onWheel, { passive:false })
  if (!reducedMotion) {
    requestAnimationFrame(() => {
      animate('.hero-line', { opacity:[0,1], y:[70,0], duration:1100, delay:stagger(100), ease:'out(4)' })
      observer = new IntersectionObserver(entries => entries.forEach(entry => {
        if(entry.isIntersecting){ animate(entry.target, { opacity:[0,1], y:[45,0], duration:850, ease:'out(3)' }); observer?.unobserve(entry.target) }
      }), { threshold:.12 })
      document.querySelectorAll('.reveal').forEach(el => observer?.observe(el))
    })
  }
})
onBeforeUnmount(()=>{
  observer?.disconnect()
  window.removeEventListener('wheel', onWheel)
  window.clearTimeout(wheelTimer)
  document.documentElement.classList.remove('page-snap')
})
</script>

<template>
  <div v-if="content" class="public-shell">
    <SiteHeader />
    <main>
      <section class="hero section-grid">
        <GridScan :intensity=".55" />
        <div class="hero-meta hero-line"><span>{{ content.heroEyebrow }}</span><span>31.2304° N / 121.4737° E</span></div>
        <div class="hero-title-wrap">
          <p class="hero-index hero-line">/ RECRUIT<br/>2026</p>
          <h1 class="hero-title hero-line"><DecryptedText :text="content.heroTitle"/></h1>
          <p class="hero-cn hero-line">{{ content.heroSubtitle }}</p>
        </div>
        <div class="hero-footer hero-line">
          <a href="#about" class="scroll-cue"><ArrowDown/> SCROLL TO EXPLORE</a>
          <RouterLink to="/apply" class="hero-cta"><span>加入我们</span><ArrowUpRight/></RouterLink>
        </div>
      </section>

      <section id="about" class="manifesto section-pad">
        <div class="section-kicker reveal"><span>001</span><span>OUR MANIFESTO</span></div>
        <div class="manifesto-grid">
          <h2 class="display-heading reveal">{{ content.manifestoTitle }}</h2>
          <div class="manifesto-copy reveal"><p>{{ content.manifestoBody }}</p><span class="vertical-rule"/><small>IDEA → PROTOTYPE → SHIP</small></div>
        </div>
      </section>

      <section id="directions" class="directions section-pad">
        <div class="section-kicker reveal"><span>002</span><span>{{ content.directionsTitle }}</span></div>
        <div class="direction-list">
          <article v-for="(item,index) in content.directions" :key="item.label" class="direction-card reveal">
            <div class="card-no">0{{ index+1 }}</div><div><span class="label-en">{{ item.label }}</span><h3>{{ item.title }}</h3><p>{{ item.body }}</p></div><ArrowUpRight class="card-arrow"/>
          </article>
        </div>
      </section>

      <section class="values section-pad dark-panel">
        <div class="section-kicker reveal"><span>003</span><span>HOW WE BUILD</span></div>
        <div class="values-intro reveal"><h2>MAKE.<br/>SHARE.<br/><em>SHIP.</em></h2><p>不是简历上的一行字，<br/>是你亲手完成的一件事。</p></div>
        <div class="value-grid">
          <article v-for="item in content.values" :key="item.label" class="value-card reveal"><span>{{ item.label }}</span><h3>{{ item.title }}</h3><p>{{ item.body }}</p></article>
        </div>
      </section>

      <section id="process" class="process section-pad">
        <div class="section-kicker reveal"><span>004</span><span>RECRUITMENT FLOW</span></div>
        <h2 class="display-heading reveal">从一次点击，<br/>到一起创造。</h2>
        <ol class="process-list">
          <li v-for="item in content.process" :key="item.label" class="reveal"><span>{{ item.label }}</span><h3>{{ item.title }}</h3><p>{{ item.body }}</p></li>
        </ol>
      </section>

      <section class="faq section-pad">
        <div class="section-kicker reveal"><span>005</span><span>BEFORE YOU ASK</span></div>
        <details v-for="item in content.faqs" :key="item.label" class="faq-item reveal"><summary><span>{{ item.label }}</span>{{ item.title }}<b>+</b></summary><p>{{ item.body }}</p></details>
      </section>

      <section class="closing section-pad">
        <GridScan :intensity=".3" />
        <p class="reveal">END OF WAITING / START OF BUILDING</p>
        <h2 class="reveal">{{ content.contactTitle }}</h2>
        <div class="closing-bottom reveal"><p>{{ content.contactBody }}</p><RouterLink to="/apply" class="giant-link">报名 / 查询 <ArrowRight/></RouterLink></div>
        <div class="closing-footer"><div class="brand"><span>IT</span><strong>STUDIO</strong></div><a :href="content.contactLink">CONTACT <ArrowUpRight :size="15"/></a><span>© 2026 / CAMPUS RECRUITMENT</span></div>
      </section>
    </main>
  </div>
  <div v-else class="loading-screen"><span>IT STUDIO</span><i/></div>
</template>
