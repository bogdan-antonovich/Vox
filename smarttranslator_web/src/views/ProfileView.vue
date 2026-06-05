<template>
  <nav>
    <span><strong>Smart Translator</strong> / Profile</span>
    <RouterLink to="/host"><button class="secondary">← Dashboard</button></RouterLink>
  </nav>

  <div class="page">
    <div class="section">
      <h2>Upload Voice Sample</h2>
      <div class="field">
        <label>Audio file</label>
        <input type="file" accept="audio/*" @change="onFile" />
      </div>
      <div class="field">
        <label>Reference text (what you say in the recording)</label>
        <input v-model="refText" type="text" placeholder="e.g. The quick brown fox jumps over the lazy dog" />
      </div>
      <p v-if="uploadError" class="error" style="margin-bottom: 10px">{{ uploadError }}</p>
      <button :disabled="!selectedFile || !refText.trim() || uploading" @click="upload">
        {{ uploading ? 'Uploading…' : 'Upload' }}
      </button>
    </div>

    <hr class="divider" />

    <div class="section">
      <h2>My Samples</h2>
      <p v-if="loadingRefs" class="muted">Loading…</p>
      <p v-else-if="!voiceRefs.length" class="muted">No samples yet.</p>
      <div
        v-for="v in voiceRefs"
        :key="v.file_id"
        class="row"
        style="padding: 10px 0; border-bottom: 1px solid #1a1a1a"
      >
        <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
          {{ v.text || v.file_id }}
        </span>
        <button class="danger" :disabled="deleting === v.file_id" @click="deleteRef(v.file_id)">
          {{ deleting === v.file_id ? '…' : 'Delete' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { voiceApi } from '../api'
import type { VoiceRef } from '../api'

const voiceRefs = ref<VoiceRef[]>([])
const loadingRefs = ref(true)
const selectedFile = ref<File | null>(null)
const refText = ref('')
const uploading = ref(false)
const uploadError = ref('')
const deleting = ref('')

onMounted(async () => {
  try {
    const { data } = await voiceApi.list()
    voiceRefs.value = data ?? []
  } finally {
    loadingRefs.value = false
  }
})

function onFile(e: Event) {
  selectedFile.value = (e.target as HTMLInputElement).files?.[0] ?? null
}

async function upload() {
  if (!selectedFile.value || !refText.value.trim()) return
  uploadError.value = ''
  uploading.value = true
  try {
    await voiceApi.upload(selectedFile.value, refText.value.trim())
    const { data } = await voiceApi.list()
    voiceRefs.value = data ?? []
    refText.value = ''
    selectedFile.value = null
  } catch {
    uploadError.value = 'Upload failed'
  } finally {
    uploading.value = false
  }
}

async function deleteRef(fileId: string) {
  deleting.value = fileId
  try {
    await voiceApi.remove(fileId)
    voiceRefs.value = voiceRefs.value.filter(v => v.file_id !== fileId)
  } finally {
    deleting.value = ''
  }
}
</script>
