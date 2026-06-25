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

    <template v-if="!recording">
      <button style="width: 100%" @click="start">
        Start Broadcasting
      </button>
      <button class="secondary" style="width: 100%; margin-top: 8px" @click="startTest">
        🎧 Test (play sample Russian audio)
      </button>
    </template>
    <button v-else class="danger" style="width: 100%" @click="stop">
      Stop {{ testing ? 'Test' : 'Broadcasting' }}
    </button>

    <p class="muted" style="margin-top: 10px">
      <template v-if="!recording">Ready to broadcast</template>
      <template v-else-if="testing">● Testing — sample audio → {{ langName(outputLang) }} (you should hear it below)</template>
      <template v-else>● Live — translating to {{ langName(outputLang) }}</template>
    </p>

    <!-- Embedded listener so the host hears the translated output during a test -->
    <div v-if="testing" class="field" style="margin-top: 16px">
      <label>Translated output (live)</label>
      <!-- eslint-disable-next-line vue/html-self-closing -->
      <audio ref="listenEl" controls autoplay style="width: 100%"></audio>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { hubApi } from '../api'

const route = useRoute()
const id = route.params.id as string

const TEST_AUDIO_URL = '/test-audio-ru.mp3'

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
const testing = ref(false)
const error = ref('')
const copied = ref(false)
const listenEl = ref<HTMLAudioElement | null>(null)

let ws: WebSocket | null = null
let mediaRecorder: MediaRecorder | null = null
let stream: MediaStream | null = null
let testAudio: HTMLAudioElement | null = null

onUnmounted(stop)

// Live broadcast from the microphone.
async function start() {
  error.value = ''
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  } catch {
    error.value = 'Microphone access denied'
    return
  }
  beginStream(stream)
}

// Test broadcast: stream a bundled Russian audio file through the exact same
// publish pipeline as the mic, and listen to the translated output in-page.
async function startTest() {
  error.value = ''
  lang.value = 'ru' // the sample file is Russian speech
  testing.value = true

  testAudio = new Audio(TEST_AUDIO_URL)
  testAudio.muted = true // don't blast the source language at the host; the capture is unaffected
  testAudio.onended = stop
  testAudio.onerror = () => { error.value = 'Could not load test audio'; stop() }

  try {
    await testAudio.play()
  } catch {
    error.value = 'Could not start test playback'
    stop()
    return
  }

  // captureStream() yields a MediaStream from the playing element, which we feed
  // to MediaRecorder just like the mic. Firefox exposes it as mozCaptureStream().
  const el = testAudio as HTMLAudioElement & {
    captureStream?: () => MediaStream
    mozCaptureStream?: () => MediaStream
  }
  const captured = el.captureStream?.() ?? el.mozCaptureStream?.()
  if (!captured) {
    error.value = 'Audio capture not supported in this browser'
    stop()
    return
  }

  beginStream(captured)
  startTestListener()
}

// Shared publish path: open the WS and pump MediaRecorder chunks into it.
function beginStream(src: MediaStream) {
  stream = src
  const url = hubApi.publishWsUrl(id, lang.value, outputLang.value)
  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
      ? 'audio/webm;codecs=opus'
      : ''
    mediaRecorder = new MediaRecorder(src, mimeType ? { mimeType } : {})
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

// Point the embedded <audio> at this hub's listen stream so the host hears output.
async function startTestListener() {
  await nextTick()
  const el = listenEl.value
  if (!el) return
  el.src = hubApi.listenUrl(id)
  el.play().catch(() => {})
}

function stop() {
  mediaRecorder?.stop()
  stream?.getTracks().forEach(t => t.stop())
  ws?.close()
  if (testAudio) {
    testAudio.onended = null
    testAudio.pause()
  }
  if (listenEl.value) {
    listenEl.value.pause()
    listenEl.value.src = ''
  }
  ws = null
  mediaRecorder = null
  stream = null
  testAudio = null
  recording.value = false
  testing.value = false
}

function copyId() {
  navigator.clipboard.writeText(id)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1500)
}
</script>
