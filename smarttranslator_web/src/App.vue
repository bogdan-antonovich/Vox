<template>
  <div v-if="auth.loading" class="splash">Loading…</div>
  <RouterView v-else />
</template>

<script setup lang="ts">
import { auth } from './store'
</script>

<style>
.splash {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  color: #333;
}
</style>
