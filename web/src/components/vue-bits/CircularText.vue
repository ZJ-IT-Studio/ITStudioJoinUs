<script setup lang="ts">
import { animate, Motion, useMotionValue } from 'motion-v';
import type { MotionValue } from 'motion-v';
import { computed, onMounted, watch } from 'vue';

interface CircularTextProps {
  text: string;
  spinDuration?: number;
  onHover?: 'slowDown' | 'speedUp' | 'pause' | 'goBonkers';
  className?: string;
}

const props = withDefaults(defineProps<CircularTextProps>(), {
  spinDuration: 20,
  onHover: 'speedUp',
  className: ''
});

const letters = computed(() => Array.from(props.text));
const rotation: MotionValue<number> = useMotionValue(0);

let currentAnimation: ReturnType<typeof animate> | null = null;

const startRotation = (duration: number) => {
  currentAnimation?.stop();
  const start = rotation.get();
  currentAnimation = animate(rotation, start + 360, {
    duration,
    ease: 'linear',
    repeat: Infinity
  });
};

onMounted(() => {
  startRotation(props.spinDuration);
});

watch(
  () => [props.spinDuration, props.text],
  () => {
    startRotation(props.spinDuration);
  }
);

const handleHoverStart = () => {
  if (!props.onHover) return;
  switch (props.onHover) {
    case 'slowDown':
      startRotation(props.spinDuration * 2);
      break;
    case 'speedUp':
      startRotation(props.spinDuration / 4);
      break;
    case 'pause':
      currentAnimation?.stop();
      break;
    case 'goBonkers':
      startRotation(props.spinDuration / 20);
      break;
  }
};

const handleHoverEnd = () => {
  startRotation(props.spinDuration);
};

const getLetterTransform = (index: number) => {
  const allLetters = letters.value;
  // build cumulative angular positions: separator gets 3x space, regular gets 1x
  const weights = allLetters.map(l => l === '⭐' ? 3 : 1);
  const totalWeight = weights.reduce((s, w) => s + w, 0);
  let cumWeight = 0;
  for (let i = 0; i < index; i++) cumWeight += weights[i];
  cumWeight += weights[index] / 2; // center of this letter's angular slot
  const rotationDeg = (360 / totalWeight) * cumWeight;
  const factor = Math.PI / allLetters.length;
  const x = factor * index;
  const y = factor * index;
  return `rotateZ(${rotationDeg}deg) translate3d(${x}px, ${y}px, 0)`;
};
</script>

<template>
  <Motion
    tag="div"
    :class="[
      'm-0 mx-auto rounded-full w-[200px] h-[200px] relative font-black text-white text-center cursor-pointer origin-center',
      className
    ]"
    :style="{ rotate: rotation }"
    :initial="{ rotate: 0 }"
    @mouseenter="handleHoverStart"
    @mouseleave="handleHoverEnd"
  >
    <span
      v-for="(letter, i) in letters"
      :key="i"
      class="inline-block absolute inset-0 text-2xl transition-all duration-500 ease-[cubic-bezier(0,0,0,1)]"
      :style="{
        transform: getLetterTransform(i),
        WebkitTransform: getLetterTransform(i)
      }"
    >
      {{ letter }}
    </span>
  </Motion>
</template>
