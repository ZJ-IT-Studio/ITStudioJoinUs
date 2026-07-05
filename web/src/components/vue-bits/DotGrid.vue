<script setup lang="ts">
// DotGrid — 物理点阵背景，鼠标推开 + 弹性回弹
import { ref, onMounted, onBeforeUnmount } from 'vue'
import gsap from 'gsap'

const props = withDefaults(defineProps<{
  dotSize?: number
  gap?: number
  baseColor?: string
  activeColor?: string
  proximity?: number
  speedTrigger?: number
  shockRadius?: number
  shockStrength?: number
  maxSpeed?: number
  resistance?: number
  returnDuration?: number
  className?: string
}>(), {
  dotSize: 2,
  gap: 28,
  baseColor: 'rgba(255,255,255,0.1)',
  activeColor: '#c5e801',
  proximity: 120,
  speedTrigger: 100,
  shockRadius: 200,
  shockStrength: 3,
  maxSpeed: 3000,
  resistance: 500,
  returnDuration: 1.2,
})

interface Dot {
  homeX: number; homeY: number // 网格原始位置
  x: number; y: number        // 当前位置
  vx: number; vy: number      // 速度
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
let dots: Dot[] = []
let animationId = 0
let mouseX = -9999
let mouseY = -9999
let prevMouseX = -9999
let prevMouseY = -9999
let mouseSpeed = 0
let returning = false

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const dpr = window.devicePixelRatio || 1
  let w = 0; let h = 0

  function resize() {
    w = window.innerWidth
    h = window.innerHeight
    canvas!.width = w * dpr
    canvas!.height = h * dpr
    canvas!.style.width = w + 'px'
    canvas!.style.height = h + 'px'
    ctx!.setTransform(dpr, 0, 0, dpr, 0, 0)
    initDots()
  }

  function initDots() {
    dots = []
    const g = props.gap
    for (let x = g; x < w; x += g) {
      for (let y = g; y < h; y += g) {
        dots.push({ homeX: x, homeY: y, x, y, vx: 0, vy: 0 })
      }
    }
  }

  function draw() {
    ctx!.clearRect(0, 0, w, h)

    // 鼠标速度
    const dx = mouseX - prevMouseX
    const dy = mouseY - prevMouseY
    mouseSpeed = Math.sqrt(dx * dx + dy * dy)
    prevMouseX = mouseX
    prevMouseY = mouseY

    const proximity = props.proximity
    const maxSpd = props.maxSpeed
    const resistance = props.resistance
    const dt = 1 / 60

    for (let i = 0; i < dots.length; i++) {
      const d = dots[i]
      const mdx = mouseX - d.x
      const mdy = mouseY - d.y
      const dist = Math.sqrt(mdx * mdx + mdy * mdy)

      if (!returning && dist < proximity && mouseSpeed > 0) {
        // 鼠标力：推开
        const force = (1 - dist / proximity) * props.shockStrength * 1000
        const nx = -mdx / dist
        const ny = -mdy / dist
        d.vx += nx * force * dt
        d.vy += ny * force * dt

        // 速度限制
        const spd = Math.sqrt(d.vx * d.vx + d.vy * d.vy)
        if (spd > maxSpd) {
          d.vx = (d.vx / spd) * maxSpd
          d.vy = (d.vy / spd) * maxSpd
        }
      } else if (!returning) {
        // 回弹力：向 home 位置加速
        const rx = d.homeX - d.x
        const ry = d.homeY - d.y
        d.vx += rx * 5 * dt
        d.vy += ry * 5 * dt
      }

      // 阻尼
      const damping = 1 - resistance * dt
      d.vx *= damping
      d.vy *= damping

      // 更新位置
      d.x += d.vx * dt
      d.y += d.vy * dt

      // 绘制
      const active = dist < proximity && mouseSpeed > 0
      const size = active ? props.dotSize * 1.8 : props.dotSize
      const color = active ? props.activeColor : props.baseColor

      ctx!.fillStyle = color
      ctx!.beginPath()
      ctx!.arc(d.x, d.y, size, 0, Math.PI * 2)
      ctx!.fill()
    }

    animationId = requestAnimationFrame(draw)
  }

  function onMove(e: MouseEvent) {
    if (returning) return
    mouseX = e.clientX
    mouseY = e.clientY
  }

  function onLeave() {
    returning = true
    // GSAP 弹性回弹
    for (const d of dots) {
      gsap.to(d, {
        x: d.homeX,
        y: d.homeY,
        vx: 0,
        vy: 0,
        duration: props.returnDuration,
        ease: 'elastic.out(1, 0.5)',
      })
    }
    setTimeout(() => { returning = false }, props.returnDuration * 1000 + 200)
  }

  function onEnter(e: MouseEvent) {
    returning = false
    mouseX = e.clientX
    mouseY = e.clientY
    prevMouseX = mouseX
    prevMouseY = mouseY
    // 清除所有 GSAP 动画
    for (const d of dots) {
      gsap.killTweensOf(d)
    }
    // 重新初始化位置
    initDots()
  }

  resize()
  draw()

  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseleave', onLeave)
  window.addEventListener('mouseenter', onEnter)
  window.addEventListener('resize', resize)

  onBeforeUnmount(() => {
    cancelAnimationFrame(animationId)
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseleave', onLeave)
    window.removeEventListener('mouseenter', onEnter)
    window.removeEventListener('resize', resize)
  })
})
</script>

<template>
  <canvas ref="canvasRef" :class="['dot-grid-canvas', className]" />
</template>

<style scoped>
.dot-grid-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}
</style>
