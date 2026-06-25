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
      <p style="margin-bottom: 16px; color: #6d5aff">
        {{ buffering ? '● Buffering…' : '● Listening' }}
      </p>
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
const buffering = ref(false)
const error = ref('')
const audioEl = ref<HTMLAudioElement | null>(null)

// The translation pipeline delivers audio in bursts with multi-second gaps
// between sentences (transcribe → translate → synthesize latency). A plain
// <audio src> has no jitter buffer, so it stalls audibly in every gap. We feed
// the chunked audio/mpeg stream through MediaSource instead and hold back
// BUFFER_TARGET_S of audio before starting playback, so the gaps are hidden at
// the cost of a constant ~3s delay — a fine trade for one-way broadcast.
const MIME = 'audio/mpeg'
const BUFFER_TARGET_S = 3 // seconds to accumulate before first playback
const EVICT_BEHIND_S = 10 // keep at most this much already-played audio buffered

let mediaSource: MediaSource | null = null
let sourceBuffer: SourceBuffer | null = null
let objectUrl = ''
let reader: ReadableStreamDefaultReader<Uint8Array> | null = null
let abort: AbortController | null = null
const queue: Uint8Array[] = []
let started = false

onMounted(() => {
  if (inputId.value) void connect()
})

onUnmounted(disconnect)

function useMediaSource(): boolean {
  return (
    typeof MediaSource !== 'undefined' &&
    typeof MediaSource.isTypeSupported === 'function' &&
    MediaSource.isTypeSupported(MIME)
  )
}

async function connect() {
  const hubId = inputId.value.trim()
  if (!hubId) return
  error.value = ''

  activeId.value = hubId
  listening.value = true
  await nextTick()

  const el = audioEl.value!

  // Fallback for browsers without MSE/mp3 support (e.g. Safari): stream straight
  // into the element, same as before — no jitter buffer, but it still plays.
  if (!useMediaSource()) {
    el.onerror = onPlaybackError
    el.src = hubApi.listenUrl(hubId)
    el.play().catch(() => {})
    return
  }

  buffering.value = true
  mediaSource = new MediaSource()
  objectUrl = URL.createObjectURL(mediaSource)
  el.src = objectUrl
  mediaSource.addEventListener('sourceopen', () => onSourceOpen(hubId), { once: true })
}

function onSourceOpen(hubId: string) {
  if (!mediaSource) return
  try {
    sourceBuffer = mediaSource.addSourceBuffer(MIME)
  } catch {
    onPlaybackError()
    return
  }
  sourceBuffer.addEventListener('updateend', pump)
  void streamInto(hubId)
}

async function streamInto(hubId: string) {
  abort = new AbortController()
  try {
    const res = await fetch(hubApi.listenUrl(hubId), { signal: abort.signal })
    if (!res.ok || !res.body) {
      onPlaybackError()
      return
    }
    reader = res.body.getReader()
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      if (value && value.byteLength > 0) {
        queue.push(value)
        pump()
      }
    }
  } catch (e) {
    // Aborted on disconnect is expected; anything else is a real failure.
    if (!(e instanceof DOMException && e.name === 'AbortError')) onPlaybackError()
  }
}

// pump moves queued chunks into the SourceBuffer one at a time (appendBuffer is
// async and can only run when the buffer is idle), then evicts already-played
// audio and kicks off playback once enough is buffered.
function pump() {
  const sb = sourceBuffer
  if (!sb || sb.updating) return

  evict()

  const chunk = queue.shift()
  if (chunk) {
    try {
      // Copy into a fresh ArrayBuffer-backed view (satisfies BufferSource typing
      // and avoids any detachment of the stream's buffer).
      sb.appendBuffer(new Uint8Array(chunk))
    } catch {
      // QuotaExceeded or buffer removed: put it back and let updateend retry.
      queue.unshift(chunk)
    }
    return
  }

  maybeStartPlayback()
}

function bufferedEnd(sb: SourceBuffer): number {
  return sb.buffered.length ? sb.buffered.end(sb.buffered.length - 1) : 0
}

function maybeStartPlayback() {
  const el = audioEl.value
  const sb = sourceBuffer
  if (!el || !sb || started) return
  if (bufferedEnd(sb) >= BUFFER_TARGET_S) {
    started = true
    buffering.value = false
    el.play().catch(() => {})
  }
}

// Drop audio well behind the playhead so a long broadcast doesn't grow the
// SourceBuffer without bound (and trigger QuotaExceededError).
function evict() {
  const el = audioEl.value
  const sb = sourceBuffer
  if (!el || !sb || sb.updating || !sb.buffered.length) return
  const start = sb.buffered.start(0)
  const cutoff = el.currentTime - EVICT_BEHIND_S
  if (cutoff > start + 1) {
    try {
      sb.remove(start, cutoff)
    } catch {
      /* remove can race with append; safe to skip this round */
    }
  }
}

function onPlaybackError() {
  error.value = 'Could not connect. Check the hub ID.'
  disconnect()
}

function disconnect() {
  abort?.abort()
  reader?.cancel().catch(() => {})
  if (sourceBuffer) {
    sourceBuffer.removeEventListener('updateend', pump)
    sourceBuffer = null
  }
  if (mediaSource && mediaSource.readyState === 'open') {
    try { mediaSource.endOfStream() } catch { /* already torn down */ }
  }
  if (objectUrl) {
    URL.revokeObjectURL(objectUrl)
    objectUrl = ''
  }
  if (audioEl.value) {
    audioEl.value.pause()
    audioEl.value.removeAttribute('src')
    audioEl.value.load()
  }
  queue.length = 0
  started = false
  mediaSource = null
  reader = null
  abort = null
  listening.value = false
  buffering.value = false
}
</script>
