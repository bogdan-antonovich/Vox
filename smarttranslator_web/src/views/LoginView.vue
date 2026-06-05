<template>
  <div class="page" style="max-width: 380px; padding-top: 80px">
    <h1>Smart Translator</h1>
    <p class="muted" style="margin-bottom: 28px">Real-time speech translation</p>

    <div class="row" style="margin-bottom: 20px">
      <button :class="mode === 'login' ? '' : 'secondary'" @click="mode = 'login'">Log in</button>
      <button :class="mode === 'signup' ? '' : 'secondary'" @click="mode = 'signup'">Sign up</button>
    </div>

    <form @submit.prevent="submit">
      <div class="field">
        <label>Login</label>
        <input v-model="login" type="text" autocomplete="username" required />
      </div>
      <div class="field">
        <label>Password</label>
        <input v-model="password" type="password" autocomplete="current-password" required />
      </div>
      <p v-if="error" class="error" style="margin-bottom: 10px">{{ error }}</p>
      <button type="submit" :disabled="busy" style="width: 100%">
        {{ busy ? '…' : mode === 'login' ? 'Log in' : 'Create account' }}
      </button>
    </form>

    <hr class="divider" />

    <div class="row">
      <button class="secondary" style="flex: 1" @click="oauth('github')">GitHub</button>
      <button class="secondary" style="flex: 1" @click="oauth('google')">Google</button>
    </div>

    <hr class="divider" />

    <p class="muted" style="text-align: center">
      Want to listen? <RouterLink to="/listen" style="color: #6d5aff">Join a hub →</RouterLink>
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi, userApi } from '../api'
import { auth } from '../store'

type Mode = 'login' | 'signup'

const router = useRouter()
const mode = ref<Mode>('login')
const login = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    if (mode.value === 'login') {
      await authApi.login(login.value, password.value)
    } else {
      await authApi.signUp(login.value, password.value)
    }
    const { data } = await userApi.info()
    auth.user = data
    router.push('/host')
  } catch (e) {
    const msg = (e as { response?: { data?: { error?: { message?: string } } } })
      ?.response?.data?.error?.message
    error.value = msg ?? 'Something went wrong'
  } finally {
    busy.value = false
  }
}

function oauth(provider: 'github' | 'google') {
  authApi.oauth(provider)
}
</script>
