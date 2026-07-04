<template>
  <div class="w-full">
    <div
      class="relative flex justify-center items-center w-full overflow-hidden infinite-scroll-wrapper"
      ref="wrapperRef"
      :style="{
        maxHeight: maxHeight,
        overscrollBehavior: 'none'
      }"
    >
      <div
        class="flex flex-col px-4 infinite-scroll-container cursor-grab"
        ref="containerRef"
        :style="{
          transform: getTiltTransform(),
          width: width,
          overscrollBehavior: 'contain',
          transformOrigin: 'center center',
          transformStyle: 'preserve-3d'
        }"
      >
        <div
          v-for="(item, index) in items"
          :key="index"
          class="box-border relative flex justify-center items-center border-2 rounded-2xl font-semibold text-xl text-center infinite-scroll-item select-none overflow-hidden"
          :style="{
            height: itemMinHeight + 'px',
            marginTop: negativeMargin,
            borderColor: borderColor
          }"
        >
          <img v-if="item.image" :src="item.image" class="w-full h-full object-cover" :alt="item.title || ''" />
          <component v-else-if="typeof item.content === 'object'" :is="item.content" />
          <template v-else>{{ item.content }}</template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { gsap } from 'gsap';
import { Observer } from 'gsap/all';
import { onMounted, onUnmounted, ref, watch } from 'vue';

gsap.registerPlugin(Observer);

interface InfiniteScrollItem {
  content?: string | object;
  image?: string;
  title?: string;
}

interface Props {
  width?: string;
  maxHeight?: string;
  negativeMargin?: string;
  items?: InfiniteScrollItem[];
  itemMinHeight?: number;
  isTilted?: boolean;
  tiltDirection?: 'left' | 'right';
  autoplay?: boolean;
  autoplaySpeed?: number;
  autoplayDirection?: 'down' | 'up';
  pauseOnHover?: boolean;
  borderColor?: string;
  dragSensitivity?: number;
}

const props = withDefaults(defineProps<Props>(), {
  width: '30rem',
  maxHeight: '100%',
  negativeMargin: '-0.5em',
  items: () => [],
  itemMinHeight: 150,
  isTilted: false,
  tiltDirection: 'left',
  autoplay: false,
  autoplaySpeed: 0.5,
  autoplayDirection: 'down',
  pauseOnHover: false,
  borderColor: '#ffffff',
  dragSensitivity: 5
});

const wrapperRef = ref<HTMLDivElement>();
const containerRef = ref<HTMLDivElement>();
let observer: Observer | null = null;
let rafId: number | null = null;
let velocity = 0;
let stopTicker: (() => void) | null = null;
let startTicker: (() => void) | null = null;

const getTiltTransform = (): string => {
  if (!props.isTilted) return 'none';
  return props.tiltDirection === 'left'
    ? 'rotateX(20deg) rotateZ(-20deg) skewX(20deg)'
    : 'rotateX(20deg) rotateZ(20deg) skewX(-20deg)';
};

const initializeScroll = () => {
  const container = containerRef.value;
  if (!container) return;
  if (props.items.length === 0) return;

  const divItems = gsap.utils.toArray<HTMLDivElement>(container.children);
  if (!divItems.length) return;

  const firstItem = divItems[0];
  const itemHeight = firstItem.offsetHeight;
  const itemMarginTop = parseFloat(getComputedStyle(firstItem).marginTop) || 0;
  const totalItemHeight = itemHeight + itemMarginTop;
  const totalHeight = itemHeight * props.items.length + itemMarginTop * (props.items.length - 1);

  const wrapFn = gsap.utils.wrap(-totalHeight, totalHeight);

  divItems.forEach((child, i) => {
    gsap.set(child, { y: i * totalItemHeight });
  });

  observer = Observer.create({
    target: container,
    type: 'wheel,touch,pointer',
    preventDefault: true,
    onPress: ({ target }) => {
      (target as HTMLElement).style.cursor = 'grabbing';
    },
    onRelease: ({ target }) => {
      (target as HTMLElement).style.cursor = 'grab';
      if (Math.abs(velocity) > 0.1) {
        const momentum = velocity * 0.8;
        divItems.forEach(child => {
          gsap.to(child, {
            duration: 1.5,
            ease: 'power2.out',
            y: `+=${momentum}`,
            modifiers: { y: gsap.utils.unitize(wrapFn) }
          });
        });
      }
      velocity = 0;
    },
    onChange: ({ deltaY, isDragging, event }) => {
      const d = event.type === 'wheel' ? -deltaY : deltaY;
      const distance = isDragging ? d * props.dragSensitivity : d * 1.5;
      velocity = distance * 0.5;
      divItems.forEach(child => {
        gsap.to(child, {
          duration: isDragging ? 0.3 : 1.2,
          ease: isDragging ? 'power1.out' : 'power3.out',
          y: `+=${distance}`,
          modifiers: { y: gsap.utils.unitize(wrapFn) }
        });
      });
    }
  });

  if (props.autoplay) {
    const directionFactor = props.autoplayDirection === 'down' ? 1 : -1;
    const speedPerFrame = props.autoplaySpeed * directionFactor;

    const tick = () => {
      divItems.forEach(child => {
        gsap.set(child, {
          y: `+=${speedPerFrame}`,
          modifiers: { y: gsap.utils.unitize(wrapFn) }
        });
      });
      rafId = requestAnimationFrame(tick);
    };
    rafId = requestAnimationFrame(tick);

    if (props.pauseOnHover) {
      stopTicker = () => { if (rafId) { cancelAnimationFrame(rafId); rafId = null; } };
      startTicker = () => { rafId = requestAnimationFrame(tick); };
      container.addEventListener('mouseenter', stopTicker);
      container.addEventListener('mouseleave', startTicker);
    }
  }
};

const cleanup = () => {
  if (observer) { observer.kill(); observer = null; }
  if (rafId) { cancelAnimationFrame(rafId); rafId = null; }
  velocity = 0;
  const container = containerRef.value;
  if (container && props.pauseOnHover && stopTicker && startTicker) {
    container.removeEventListener('mouseenter', stopTicker);
    container.removeEventListener('mouseleave', startTicker);
  }
  stopTicker = null;
  startTicker = null;
};

onMounted(() => { initializeScroll(); });
onUnmounted(() => { cleanup(); });

watch(
  [() => props.items, () => props.autoplay, () => props.autoplaySpeed, () => props.autoplayDirection, () => props.pauseOnHover, () => props.isTilted, () => props.tiltDirection, () => props.negativeMargin],
  () => { cleanup(); setTimeout(() => initializeScroll(), 0); }
);
</script>

<style scoped>
.infinite-scroll-wrapper::before,
.infinite-scroll-wrapper::after {
  content: '';
  position: absolute;
  background: linear-gradient(var(--dir, to bottom), var(--dark, #202220), transparent);
  height: 25%;
  width: 100%;
  z-index: 1;
  pointer-events: none;
}
.infinite-scroll-wrapper::before { top: 0; }
.infinite-scroll-wrapper::after { --dir: to top; bottom: 0; }
.infinite-scroll-container {
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
}
.infinite-scroll-item {
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
  transform: translateZ(0);
}
</style>
