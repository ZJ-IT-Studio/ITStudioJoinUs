<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';

type Cell = number | null;
type Grid = Cell[][];

const size = 4;
const grid = ref<Grid>([]);
const score = ref(0);
const gameOver = ref(false);

function initGrid() {
  grid.value = Array.from({ length: size }, () => Array(size).fill(null));
  addTile();
  addTile();
}

function addTile() {
  const empty: [number, number][] = [];
  for (let r = 0; r < size; r++)
    for (let c = 0; c < size; c++)
      if (!grid.value[r][c]) empty.push([r, c]);
  if (!empty.length) return;
  const [r, c] = empty[Math.floor(Math.random() * empty.length)];
  grid.value[r][c] = Math.random() < 0.9 ? 2 : 4;
}

function slideRow(row: Cell[]): Cell[] {
  let arr = row.filter((v): v is number => v !== null);
  for (let i = 0; i < arr.length - 1; i++) {
    if (arr[i] === arr[i + 1]) {
      arr[i] *= 2;
      score.value += arr[i];
      arr.splice(i + 1, 1);
    }
  }
  while (arr.length < size) arr.push(null);
  return arr;
}

function move(direction: 'up' | 'down' | 'left' | 'right') {
  const old = grid.value.map(r => [...r]);
  let g = grid.value.map(r => [...r]);

  if (direction === 'left') g = g.map(r => slideRow(r));
  if (direction === 'right') g = g.map(r => slideRow([...r].reverse()).reverse());

  if (direction === 'up' || direction === 'down') {
    let transposed: Grid = Array.from({ length: size }, (_, r) =>
      Array.from({ length: size }, (_, c) => g[c][r])
    );
    if (direction === 'up') transposed = transposed.map(r => slideRow(r));
    if (direction === 'down') transposed = transposed.map(r => slideRow([...r].reverse()).reverse());
    g = Array.from({ length: size }, (_, r) =>
      Array.from({ length: size }, (_, c) => transposed[c][r])
    );
  }

  grid.value = g;
  const changed = old.some((r, i) => r.some((v, j) => v !== g[i][j]));
  if (changed) addTile();
  checkGameOver();
}

function checkGameOver() {
  for (let r = 0; r < size; r++)
    for (let c = 0; c < size; c++) {
      if (!grid.value[r][c]) { gameOver.value = false; return; }
      if (c < size - 1 && grid.value[r][c] === grid.value[r][c + 1]) { gameOver.value = false; return; }
      if (r < size - 1 && grid.value[r][c] === grid.value[r + 1][c]) { gameOver.value = false; return; }
    }
  gameOver.value = true;
}

function cellStyle(v: Cell) {
  if (!v) return { background: 'rgba(255,255,255,.06)', color: 'transparent' };
  const colors: Record<number, string> = {
    2: '#eee4da', 4: '#ede0c8', 8: '#f2b179', 16: '#f59563',
    32: '#f67c5f', 64: '#f65e3b', 128: '#edcf72', 256: '#edcc61',
    512: '#edc850', 1024: '#edc53f', 2048: '#edc22e'
  };
  return {
    background: colors[v] || '#3c3a32',
    color: v <= 4 ? '#776e65' : '#f9f6f2',
    fontSize: v >= 1024 ? '14px' : v >= 128 ? '16px' : '18px'
  };
}

function reset() { score.value = 0; gameOver.value = false; initGrid(); }

function onKeydown(e: KeyboardEvent) {
  const map: Record<string, 'up' | 'down' | 'left' | 'right'> = {
    ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right',
    w: 'up', s: 'down', a: 'left', d: 'right'
  };
  const dir = map[e.key];
  if (dir) { e.preventDefault(); move(dir); }
}

onMounted(() => { initGrid(); window.addEventListener('keydown', onKeydown); });
onUnmounted(() => window.removeEventListener('keydown', onKeydown));
</script>

<template>
  <div class="game2048">
    <div class="game2048-header">
      <span class="game2048-score">{{ score }}</span>
      <button class="game2048-reset" @click="reset" title="重新开始">↺</button>
    </div>
    <div class="game2048-grid">
      <div v-for="(row, r) in grid" :key="r" class="game2048-row">
        <div
          v-for="(cell, c) in row" :key="c"
          class="game2048-cell"
          :style="cellStyle(cell)"
        >{{ cell || '' }}</div>
      </div>
    </div>
    <div v-if="gameOver" class="game2048-overlay" @click="reset">
      <span>Game Over</span>
      <small>点击重新开始</small>
    </div>
  </div>
</template>

<style scoped>
.game2048 { width: 200px; position: relative; }
.game2048-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.game2048-score { color: var(--acid); font-size: 18px; font-weight: 900; }
.game2048-reset { background: none; border: 1px solid rgba(255,255,255,.2); color: #fff; border-radius: 6px; padding: 2px 10px; font-size: 16px; cursor: pointer; }
.game2048-reset:hover { border-color: var(--acid); color: var(--acid); }
.game2048-grid { display: grid; grid-template-rows: repeat(4, 1fr); gap: 4px; background: rgba(255,255,255,.04); border-radius: 10px; padding: 6px; }
.game2048-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 4px; }
.game2048-cell { aspect-ratio: 1; display: grid; place-items: center; border-radius: 6px; font-weight: 800; transition: all .15s; }
.game2048-overlay { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; background: rgba(0,0,0,.75); border-radius: 10px; cursor: pointer; }
.game2048-overlay span { font-size: 22px; font-weight: 900; color: #fff; }
.game2048-overlay small { color: #999; margin-top: 6px; font-size: 11px; }
</style>
