<script setup lang="ts">
// Vue Bits DecryptedText — 文字解密动画组件
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  text: string
  speed?: number
  maxIterations?: number
  characters?: string
  sequential?: boolean
  revealDirection?: 'start' | 'end' | 'center'
  useOriginalCharsOnly?: boolean
  animateOn?: 'mount' | 'hover'
  start?: boolean
  className?: string
  encryptedClassName?: string
}>(), {
  speed: 50,
  maxIterations: 10,
  characters: 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#@%&',
  sequential: false,
  revealDirection: 'start',
  useOriginalCharsOnly: false,
  animateOn: 'mount',
  start: true,
})

const shown = ref(props.text)
let timer: number | undefined
const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches

// 计算每个非空格字符的解密顺序（空格直接跳过，不占位）
function buildOrder(): number[] {
  const nonSpaceIndices: number[] = []
  for (let i = 0; i < props.text.length; i++) {
    if (props.text[i] !== ' ') nonSpaceIndices.push(i)
  }
  if (props.revealDirection === 'end') nonSpaceIndices.reverse()
  if (props.revealDirection === 'center') {
    const half = Math.ceil(nonSpaceIndices.length / 2)
    const result: number[] = []
    for (let i = 0; i < half; i++) {
      if (half - 1 - i >= 0) result.push(nonSpaceIndices[half - 1 - i])
      if (half + i < nonSpaceIndices.length) result.push(nonSpaceIndices[half + i])
    }
    return result
  }
  return nonSpaceIndices
}

// 获取可用于加密的字符集
function getChars(): string {
  if (props.useOriginalCharsOnly) {
    return [...new Set(props.text.replace(/\s/g, '').split(''))].join('') || props.characters
  }
  return props.characters
}

function play() {
  if (reduced) { shown.value = props.text; return }
  window.clearInterval(timer)

  const chars = getChars()
  const order = props.sequential ? buildOrder() : null
  let cursor = 0  // sequential 模式下的扫光位置
  const iterations = Array(props.text.length).fill(0)

  timer = window.setInterval(() => {
    const arr = props.text.split('')

    if (props.sequential && order && cursor < order.length) {
      // 扫光模式：cursor 位置之前的字符揭示原文，cursor 及之后的字符显示密文
      for (let i = 0; i < arr.length; i++) {
        if (arr[i] === ' ') continue
        const posInOrder = order.indexOf(i)
        if (posInOrder < cursor) {
          // 已扫过的位置 → 显示原文
          // nothing, arr already has the original char
        } else if (posInOrder === cursor) {
          // 当前扫光位置 → 跳动 iterations，然后揭示
          if (iterations[i] < props.maxIterations) {
            arr[i] = chars[Math.floor(Math.random() * chars.length)]
            iterations[i]++
          } else {
            arr[i] = props.text[i]
          }
        } else {
          // 未扫到的位置 → 随机密文
          arr[i] = chars[Math.floor(Math.random() * chars.length)]
        }
      }
      // 当前字符揭示完毕，扫光前进
      const currentIdx = order[cursor]
      if (iterations[currentIdx] >= props.maxIterations) {
        cursor++
      }
    } else {
      // 非顺序模式：每个字符独立跳动
      for (let i = 0; i < arr.length; i++) {
        if (arr[i] === ' ') continue
        if (iterations[i] < props.maxIterations) {
          arr[i] = chars[Math.floor(Math.random() * chars.length)]
          iterations[i]++
        }
      }
    }

    shown.value = arr.join('')

    // 全部完成
    if (!props.sequential) {
      const allDone = iterations.every((n, i) => props.text[i] === ' ' || n >= props.maxIterations)
      if (allDone) {
        shown.value = props.text
        window.clearInterval(timer)
      }
    } else if (cursor >= (order?.length || 0)) {
      shown.value = props.text
      window.clearInterval(timer)
    }
  }, props.speed)
}

onMounted(() => { if (props.animateOn === 'mount' && props.start) play() })
onBeforeUnmount(() => window.clearInterval(timer))
watch(() => props.text, play)
watch(() => props.start, (v) => { if (v && props.animateOn === 'mount') play() })
</script>

<template>
  <span
    :class="['decrypted', className]"
    @mouseenter="animateOn === 'hover' && play()"
  >{{ shown }}</span>
</template>
