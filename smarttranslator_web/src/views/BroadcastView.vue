<template>
  <nav>
    <span><strong>Smart Translator</strong> / Broadcast</span>
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
        <option value="pt">Portuguese</option>
        <option value="it">Italian</option>
        <option value="ko">Korean</option>
        <option value="pl">Polish</option>
        <option value="nl">Dutch</option>
        <option value="tr">Turkish</option>
      </select>
    </div>

    <div class="field">
      <label>Translate to</label>
      <select v-model="outputLang" :disabled="recording">
        <option value="en">English</option>
        <option value="ru">Russian</option>
        <option value="uk">Ukrainian</option>
        <option value="de">German</option>
        <option value="fr">French</option>
        <option value="es">Spanish</option>
        <option value="zh">Chinese</option>
        <option value="ja">Japanese</option>
        <option value="ar">Arabic</option>
        <option value="pt">Portuguese</option>
        <option value="it">Italian</option>
        <option value="ko">Korean</option>
        <option value="pl">Polish</option>
        <option value="nl">Dutch</option>
        <option value="tr">Turkish</option>
      </select>
    </div>

    <p v-if="error" class="error" style="margin-bottom: 12px">{{ error }}</p>

    <button v-if="!recording" style="width: 100%" @click="start">
      Start Broadcasting
    </button>
    <button v-else class="danger" style="width: 100%" @click="stop">
      Stop Broadcasting
    </button>

    <p class="muted" style="margin-top: 10px">
      {{ recording ? `● Live — translating to ${langName(outputLang)}` : 'Ready to broadcast' }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { hubApi } from '../api'

const route = useRoute()
const id = route.params.id as string

const LANG_NAMES: Record<string, string> = {
  en: 'English', ru: 'Russian', uk: 'Ukrainian', de: 'German',
  fr: 'French', es: 'Spanish', zh: 'Chinese', ja: 'Japanese',
  ar: 'Arabic', pt: 'Portuguese', it: 'Italian', ko: 'Korean',
  pl: 'Polish', nl: 'Dutch', tr: 'Turkish',
}
const langName = (code: string) => LANG_NAMES[code] ?? code.toUpperCase()

const lang = ref('ru')
const outputLang = ref('en')
const recording = ref(false)
const error = ref('')
const copied = ref(false)

let ws: WebSocket | null = null
let mediaRecorder: MediaRecorder | null = null
let stream: MediaStream | null = null

onUnmounted(stop)

async function start() {
  error.value = ''
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  } catch {
    error.value = 'Microphone access denied'
    return
  }

  const url = hubApi.publishWsUrl(id, lang.value, outputLang.value)
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
