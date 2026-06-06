<template>
  <nav>
    <strong>Smart Translator</strong>
    <RouterLink to="/"><button class="secondary">← Home</button></RouterLink>
  </nav>

  <div class="page" style="max-width: 480px">
    <div v-if="!listening">
      <h1>Join a Hub</h1>
      <div class="field">
        <label>Hub ID</label>
        <input
          v-model="inputId"
          placeholder="Paste the hub ID here"
          @keydown.enter="connect"
        />
      </div>
      <p v-if="error" class="error" style="margin-bottom: 12px">{{ error }}</p>
      <button :disabled="!inputId.trim()" style="width: 100%" @click="connect">
        Start Listening
      </button>
      <p class="muted" style="margin-top: 12px">No account needed.</p>
    </div>

    <div v-else>
      <p style="margin-bottom: 16px; color: #6d5aff">● Listening</p>
      <p class="muted" style="margin-bottom: 16px; word-break: break-all">
        Hub: <code>{{ activeId }}</code>
      </p>
      <!-- eslint-disable-next-line vue/html-self-closing -->
      <audio ref="audioEl" controls autoplay style="width: 100%"></audio>
      <button class="secondary" style="width: 100%; margin-top: 16px" @click="disconnect">
        Stop
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { hubApi } from '../api'

const route = useRoute()
const inputId = ref((route.params.id as string | undefined) ?? '')
const activeId = ref('')
const listening = ref(false)
const error = ref('')
const audioEl = ref<HTMLAudioElement | null>(null)

onMounted(() => {
  if (inputId.value) void connect()
})

onUnmounted(disconnect)

async function connect() {
  const hubId = inputId.value.trim()
  if (!hubId) return
  error.value = ''

  activeId.value = hubId
  listening.value = true
  await nextTick()

  const el = audioEl.value!
  el.onerror = () => {
    error.value = 'Could not connect. Check the hub ID.'
    listening.value = false
    activeId.value = ''
  }
  el.src = hubApi.listenUrl(hubId)
  el.play().catch(() => {})
}

function disconnect() {
  if (audioEl.value) {
    audioEl.value.pause()
    audioEl.value.src = ''
  }
  listening.value = false
}
</script>
