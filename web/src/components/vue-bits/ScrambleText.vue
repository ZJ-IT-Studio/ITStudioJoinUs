<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';

interface Props {
  radius?: number;
  duration?: number;
  speed?: number;
  scrambleChars?: string;
  className?: string;
}

const props = withDefaults(defineProps<Props>(), {
  radius: 100,
  duration: 1.2,
  speed: 0.5,
  scrambleChars: '.:',
  className: ''
});

const containerRef = ref<HTMLElement>();

const CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?/';

let originalText = '';
let animationFrame: number | null = null;
let mouseX = -999;
let mouseY = -999;
let isHovering = false;

function getRandomChar() {
  const pool = props.scrambleChars.length > 0 ? props.scrambleChars : CHARS;
  return pool[Math.floor(Math.random() * pool.length)];
}

function scramble() {
  const el = containerRef.value;
  if (!el) return;

  const spans = el.querySelectorAll<HTMLElement>('.scramble-char');
  const rect = el.getBoundingClientRect();

  spans.forEach(span => {
    const charRect = span.getBoundingClientRect();
    const cx = charRect.left + charRect.width / 2;
    const cy = charRect.top + charRect.height / 2;
    const dx = mouseX - cx;
    const dy = mouseY - cy;
    const dist = Math.sqrt(dx * dx + dy * dy);

    if (isHovering && dist < props.radius) {
      if (span.dataset.original && Math.random() < props.speed) {
        span.textContent = getRandomChar();
      }
      span.style.transition = 'none';
    } else {
      span.textContent = span.dataset.original || span.textContent;
      span.style.transition = `color ${props.duration}s ease`;
    }
  });

  animationFrame = requestAnimationFrame(scramble);
}

function buildSpans() {
  const el = containerRef.value;
  if (!el) return;
  originalText = el.textContent || '';
  el.innerHTML = '';
  for (const char of originalText) {
    const span = document.createElement('span');
    span.className = 'scramble-char';
    span.textContent = char;
    span.dataset.original = char;
    span.style.display = 'inline';
    span.style.whiteSpace = 'pre';
    el.appendChild(span);
  }
}

function onMouseMove(e: MouseEvent) {
  mouseX = e.clientX;
  mouseY = e.clientY;
}

function onMouseEnter() {
  isHovering = true;
}

function onMouseLeave() {
  isHovering = false;
}

onMounted(() => {
  buildSpans();
  animationFrame = requestAnimationFrame(scramble);
  const el = containerRef.value;
  if (el) {
    el.addEventListener('mousemove', onMouseMove);
    el.addEventListener('mouseenter', onMouseEnter);
    el.addEventListener('mouseleave', onMouseLeave);
  }
});

onBeforeUnmount(() => {
  if (animationFrame) cancelAnimationFrame(animationFrame);
  const el = containerRef.value;
  if (el) {
    el.removeEventListener('mousemove', onMouseMove);
    el.removeEventListener('mouseenter', onMouseEnter);
    el.removeEventListener('mouseleave', onMouseLeave);
  }
});
</script>

<template>
  <span ref="containerRef" :class="['scramble-text', className]">
    <slot />
  </span>
</template>
