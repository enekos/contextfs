<script setup>
import { ref, computed } from 'vue'

function useCounter(initial) {
  const count = ref(initial)
  const doubled = computed(() => count.value * 2)

  function increment() {
    count.value++
  }

  function decrement() {
    if (count.value > 0) {
      count.value--
    }
  }

  function reset() {
    count.value = initial
  }

  return { count, doubled, increment, decrement, reset }
}
</script>

<template>
  <div class="counter">
    <button @click="decrement">-</button>
    <span>{{ count }} (doubled: {{ doubled }})</span>
    <button @click="increment">+</button>
    <button @click="reset">Reset</button>
  </div>
</template>
