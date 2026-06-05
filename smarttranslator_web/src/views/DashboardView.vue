<template>
  <nav>
    <strong>Smart Translator</strong>
    <div class="row">
      <span>{{ auth.user?.name }}</span>
      <RouterLink to="/profile"><button class="secondary">Profile</button></RouterLink>
      <button class="secondary" @click="logout">Log out</button>
    </div>
  </nav>

  <div class="page">
    <div class="row" style="margin-bottom: 24px">
      <h1 style="margin: 0">My Hubs</h1>
      <button :disabled="creating" @click="createHub">
        {{ creating ? '…' : '+ New Hub' }}
      </button>
    </div>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="!hubs.length" class="muted">No hubs yet — create one to start broadcasting.</p>

    <div
      v-for="id in hubs"
      :key="id"
      class="row"
      style="padding: 12px 0; border-bottom: 1px solid #1a1a1a"
    >
      <code style="flex: 1; color: #666; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
        {{ id }}
      </code>
      <RouterLink :to="`/host/${id}/broadcast`">
        <button>Broadcast</button>
      </RouterLink>
      <button class="danger" :disabled="deleting === id" @click="deleteHub(id)">
        {{ deleting === id ? '…' : 'Delete' }}
      </button>
    </div>

    <hr class="divider" />
    <p class="muted">
      Share a hub ID with listeners — they can join at
      <RouterLink to="/listen" style="color: #6d5aff">/listen</RouterLink>
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { userApi, hubApi, authApi } from '../api'
import { auth } from '../store'

const router = useRouter()
const hubs = ref<string[]>([])
const loading = ref(true)
const creating = ref(false)
const deleting = ref('')

onMounted(async () => {
  try {
    const { data } = await userApi.hubs()
    hubs.value = data.hub_ids ?? []
  } finally {
    loading.value = false
  }
})

async function createHub() {
  creating.value = true
  try {
    const { data } = await hubApi.create()
    hubs.value.push(data.hub_id)
  } finally {
    creating.value = false
  }
}

async function deleteHub(id: string) {
  deleting.value = id
  try {
    await hubApi.remove(id)
    hubs.value = hubs.value.filter(h => h !== id)
  } finally {
    deleting.value = ''
  }
}

async function logout() {
  await authApi.logout().catch(() => {})
  auth.user = null
  router.push('/')
}
</script>
