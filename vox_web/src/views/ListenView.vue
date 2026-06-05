<template>
  <nav>
    <strong>Vox</strong>
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

let reader: ReadableStreamDefaultReader<Uint8Array> | null = null
let objectUrl = ''

onMounted(() => {
  if (inputId.value) void connect()
})

onUnmounted(disconnect)

async function connect() {
  const hubId = inputId.value.trim()
  if (!hubId) return
  error.value = ''

  let response: Response
  try {
    response = await fetch(hubApi.listenUrl(hubId))
    if (!response.ok || !response.body) throw new Error('bad response')
  } catch {
    error.value = 'Could not connect. Check the hub ID and try again.'
    return
  }

  activeId.value = hubId
  listening.value = true
  await nextTick()

  const ms = new MediaSource()
  objectUrl = URL.createObjectURL(ms)
  audioEl.value!.src = objectUrl

  ms.addEventListener('sourceopen', () => {
    const sb = ms.addSourceBuffer('audio/mpeg')
    reader = response.body!.getReader()

    const pump = async () => {
      for (;;) {
        const { done, value } = await reader!.read()
        if (done) break
        if (sb.updating) {
          await new Promise<void>(r => sb.addEventListener('updateend', () => r(), { once: true }))
        }
        sb.appendBuffer(value.buffer as ArrayBuffer)
      }
    }

    pump().catch(() => {})
  })

  audioEl.value!.play().catch(() => {})
}

function disconnect() {
  reader?.cancel()
  if (audioEl.value) {
    audioEl.value.pause()
    audioEl.value.src = ''
  }
  if (objectUrl) {
    URL.revokeObjectURL(objectUrl)
    objectUrl = ''
  }
  reader = null
  listening.value = false
}
</script>
