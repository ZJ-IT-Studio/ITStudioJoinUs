<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { animate, stagger } from 'animejs'
import { ArrowDown, ArrowRight, ArrowUpRight } from 'lucide-vue-next'
import SiteHeader from '../components/SiteHeader.vue'
import DecryptedText from '../components/vue-bits/DecryptedText.vue'
import GridScan from '../components/vue-bits/GridScan.vue'
import Stack from '../components/vue-bits/Stack.vue'
import InfiniteScroll from '../components/vue-bits/InfiniteScroll.vue'
import CircularText from '../components/vue-bits/CircularText.vue'
import Game2048 from '../components/vue-bits/Game2048.vue'


import { api } from '../api'
import type { SiteContent } from '../types'

const content = ref<SiteContent | null>(null)
const splashReady = ref(!!(window as any).__splashDone)
const achievementItems = [
  { image: '/image/002/1.webp', title: '' },
  { image: '/image/002/1aa6f6d94dc0694075c7c851779949d5.webp', title: '' },
  { image: '/image/002/2.webp', title: '' },
  { image: '/image/002/ScreenShot_2026-07-04_153221_074.webp', title: '' },
  { image: '/image/002/微信图片_20260704151211_1899_38.webp', title: '' },
  { image: '/image/002/微信图片_20260704151219_1905_38.webp', title: '' }
]

const stackCards = [
  { id: 1, img: '/image/IMG_20260419_182152..webp' },
  { id: 2, img: '/image/IMG_20260420_114312..webp' },
  { id: 3, img: '/image/IMG_20260420_114456..webp' },
  { id: 4, img: '/image/IMG_20260420_131022..webp' },
  { id: 5, img: '/image/IMG_20260420_192203..webp' },
  { id: 6, img: '/image/IMG_20260420_192226..webp' }
]
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

  if (!splashReady.value) {
    await new Promise<void>(resolve => {
      window.addEventListener('splash-done', () => { splashReady.value = true; resolve() }, { once: true })
    })
  }

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
          <h1 class="hero-title hero-line">
                <span class="hero-word"><DecryptedText :text="content.heroTitle.split(' ')[0]" :speed="45" :max-iterations="8" :sequential="true" reveal-direction="start" :start="splashReady" /></span>
                <span class="hero-word"><DecryptedText :text="content.heroTitle.split(' ')[1]" :speed="45" :max-iterations="8" :sequential="true" reveal-direction="start" :start="splashReady" /></span>
                <span class="hero-word"><DecryptedText :text="content.heroTitle.split(' ')[2]" :speed="45" :max-iterations="8" :sequential="true" reveal-direction="start" :start="splashReady" /></span>
              </h1>
          <p class="hero-cn hero-line"><DecryptedText :text="content.heroSubtitle" :speed="45" :max-iterations="2" :sequential="true" reveal-direction="start" :start="splashReady" /></p>
        </div>
        <div class="hero-footer hero-line">
          <a href="#about" class="scroll-cue"><ArrowDown/> SCROLL TO EXPLORE</a>
          <RouterLink to="/apply" class="hero-cta"><span>加入我们</span><ArrowUpRight/></RouterLink>
        </div>
      </section>

      <section id="about" class="manifesto section-pad">
        <div class="section-kicker reveal"><span>001</span><span>OUR MANIFESTO</span></div>
        <div class="manifesto-layout reveal">
          <div class="manifesto-text">
            <h2 class="manifesto-title">工作环境一览</h2>
            <div class="manifesto-quote">
              <p>一个人可以走很快，<br/>一群人才能走很远。</p>
              <p>每一行代码都是成长的印记，<br/>每一个项目都是团队的勋章。</p>
              <p class="manifesto-tagline">在这里，把想法编译成现实。</p>
            </div>
          </div>
          <div class="manifesto-arrow" aria-hidden="true">
            <svg viewBox="0 0 120 80" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M8 40 Q22 34 36 40 Q52 46 66 38 Q78 32 90 40 Q100 46 108 40"
                    stroke="var(--acid)" stroke-width="3" stroke-linecap="round" fill="none"
                    stroke-dasharray="70 3 8 2 12 4" />
              <path d="M84 22 Q90 12 96 18"
                    stroke="var(--acid)" stroke-width="3" stroke-linecap="round" fill="none" />
              <path d="M100 36 Q108 30 112 34"
                    stroke="var(--acid)" stroke-width="3" stroke-linecap="round" fill="none" />
              <path d="M94 42 Q100 48 96 56"
                    stroke="var(--acid)" stroke-width="3" stroke-linecap="round" fill="none" />
              <path d="M84 20 Q80 26 78 32"
                    stroke="var(--acid)" stroke-width="2.5" stroke-linecap="round" fill="none" />
              <path d="M104 32 Q106 26 108 28"
                    stroke="var(--acid)" stroke-width="2.5" stroke-linecap="round" fill="none" />
            </svg>
          </div>
          <div class="manifesto-stack">
            <Stack
              :cards-data="stackCards"
              :random-rotation="true"
              :sensitivity="180"
              :send-to-back-on-click="true"
              :autoplay="true"
              :autoplay-delay="2500"
              :pause-on-hover="true"
              :max-rotation="25"
            >
              <template #default="{ card, index }">
                <img v-if="card.img" :src="card.img" class="stack-card-img" draggable="false" />
                <div
                  v-else
                  class="stack-solid-card"
                  :style="{
                    background: ['#c5e801','#1a1a2e','#16213e','#0f3460','#e94560'][index % 5],
                    width: '100%',
                    height: '100%'
                  }"
                />
              </template>
            </Stack>
          </div>
        </div>
      </section>

      <section id="achievements" class="achievements section-pad dark-panel">
        <div class="section-kicker reveal"><span>002</span><span>ACHIEVEMENTS</span></div>
        <div class="achievements-layout reveal">
          <div class="achievements-text">
            <h2 class="achievements-title">参赛经历与部分获奖</h2>
            <div class="achievements-quote">
              <p>每一份荣誉的背后，<br/>都是无数个日夜的打磨与坚持。</p>
              <p>你也可以站在这里，<br/>从一行代码开始，写出属于自己的故事。</p>
              <p class="achievements-tagline">下一个获奖的，就是你。</p>
            </div>
          </div>
          <div class="achievements-scroll">
            <InfiniteScroll
              :items="achievementItems"
              width="clamp(280px, 34vw, 440px)"
              max-height="100%"
              :item-min-height="220"
              :is-tilted="true"
              tilt-direction="left"
              :autoplay="true"
              :autoplay-speed="1.0"
              :pause-on-hover="true"
              :drag-sensitivity="10"
              negative-margin="-1.2em"
              border-color="#ffffff"
            />
          </div>
        </div>
      </section>

      <section id="directions" class="directions section-pad">
        <div class="section-kicker reveal"><span>003</span><span>{{ content.directionsTitle }}</span></div>
        <div class="direction-list">
          <article v-for="(item,index) in content.directions" :key="item.label" class="direction-card reveal">
            <div class="card-no">0{{ index+1 }}</div><div><span class="label-en">{{ item.label }}</span><h3>{{ item.title }}</h3><p>{{ item.body }}</p></div><ArrowUpRight class="card-arrow"/>
          </article>
        </div>
      </section>

      <section class="values section-pad dark-panel">
        <div class="section-kicker reveal"><span>004</span><span>HOW WE BUILD</span></div>
        <div class="values-intro reveal"><h2>MAKE.<br/>SHARE.<br/><em>SHIP.</em></h2><div class="values-cta"><CircularText text="⭐HAVE⭐A⭐TRY" :spin-duration="12" on-hover="speedUp" /><p>不是简历上的一行字，<br/>是你亲手完成的一件事。</p></div></div>
        <div class="values-bottom reveal">
          <div class="values-motto">
            <p class="values-motto-main">好玩的工具，都是自己一行一行写出来的。</p>
            <p class="values-motto-sub">别只玩游戏 — 去写一个属于你自己的。</p>
          </div>
          <div class="values-game-area">
            <Game2048 />
            <span class="values-game-hint">⌨ 方向键 / WASD 操作</span>
          </div>
        </div>
      </section>

      <section id="process" class="process section-pad">
        <div class="section-kicker reveal"><span>005</span><span>RECRUITMENT FLOW</span></div>
        <h2 class="display-heading reveal">从一次点击，<br/>到一起创造。</h2>
        <ol class="process-list">
          <li v-for="item in content.process" :key="item.label" class="reveal"><span>{{ item.label }}</span><h3>{{ item.title }}</h3><p>{{ item.body }}</p></li>
        </ol>
      </section>

      <section class="faq section-pad">
        <div class="section-kicker reveal"><span>006</span><span>BEFORE YOU ASK</span></div>
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
