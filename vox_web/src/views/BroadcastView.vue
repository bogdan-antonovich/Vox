<template>
  <nav>
    <span><strong>Vox</strong> / Broadcast</span>
    <RouterLink to="/host"><button class="secondary">← Dashboard</button></RouterLink>
  </nav>

  <div class="page">
    <div class="field">
      <label>Hub ID</label>
      <div class="row">
        <input :value="id" readonly />
        <button class="secondary" @click="copyId">{{ copied ? '✓' : 'Copy' }}</button>
      </div>
    </div>

    <div class="field">
      <label>Source language</label>
      <select v-model="lang" :disabled="recording">
        <option value="ru">Russian</option>
        <option value="uk">Ukrainian</option>
        <option value="de">German</option>
        <option value="fr">French</option>
        <option value="es">Spanish</option>
        <option value="zh">Chinese</option>
        <option value="ja">Japanese</option>
        <option value="ar">Arabic</option>
      </select>
    </div>

    <div class="field">
      <label>Voice sample (for TTS)</label>
      <p v-if="loadingVoice" class="muted">Loading…</p>
      <p v-else-if="!voiceRefs.length" class="muted">
        No samples —
        <RouterLink to="/profile" style="color: #6d5aff">upload one in Profile →</RouterLink>
      </p>
      <select v-else v-model="fileId" :disabled="recording">
        <option v-for="v in voiceRefs" :key="v.file_id" :value="v.file_id">
          {{ v.text ? v.text.slice(0, 70) : v.file_id.slice(0, 12) + '…' }}
        </option>
      </select>
    </div>

    <p v-if="error" class="error" style="margin-bottom: 12px">{{ error }}</p>

    <button
      v-if="!recording"
      style="width: 100%"
      :disabled="!fileId || loadingVoice"
      @click="start"
    >
      Start Broadcasting
    </button>
    <button v-else class="danger" style="width: 100%" @click="stop">
      Stop Broadcasting
    </button>

    <p class="muted" style="margin-top: 10px">
      {{ recording ? '● Live — translating to English' : 'Translates speech to English in real time' }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { voiceApi, hubApi } from '../api'
import type { VoiceRef } from '../api'

const route = useRoute()
const id = route.params.id as string

const lang = ref('ru')
const fileId = ref('')
const recording = ref(false)
const error = ref('')
const copied = ref(false)
const voiceRefs = ref<VoiceRef[]>([])
const loadingVoice = ref(true)

let ws: WebSocket | null = null
let mediaRecorder: MediaRecorder | null = null
let stream: MediaStream | null = null

onMounted(async () => {
  try {
    const { data } = await voiceApi.list()
    voiceRefs.value = data ?? []
    if (data?.length) fileId.value = data[0].file_id
  } catch { /* ignore */ } finally {
    loadingVoice.value = false
  }
})

onUnmounted(stop)

async function start() {
  error.value = ''
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  } catch {
    error.value = 'Microphone access denied'
    return
  }

  const url = hubApi.publishWsUrl(id, lang.value, fileId.value)
  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
      ? 'audio/webm;codecs=opus'
      : ''
    mediaRecorder = new MediaRecorder(stream!, mimeType ? { mimeType } : {})
    mediaRecorder.ondataavailable = (e) => {
      const sock = ws
      if (e.data.size > 0 && sock?.readyState === WebSocket.OPEN) {
        e.data.arrayBuffer().then(buf => sock.send(buf))
      }
    }
    mediaRecorder.start(250)
    recording.value = true
  }

  ws.onerror = () => { error.value = 'WebSocket connection failed'; stop() }
  ws.onclose = () => { recording.value = false }
}

function stop() {
  mediaRecorder?.stop()
  stream?.getTracks().forEach(t => t.stop())
  ws?.close()
  ws = null
  mediaRecorder = null
  stream = null
  recording.value = false
}

function copyId() {
  navigator.clipboard.writeText(id)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1500)
}
</script>
