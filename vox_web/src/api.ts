import axios from 'axios'
import type { UserInfo } from './store'

export const BASE = 'https://bogdanantonovich.com/vox/api'
const WS_BASE = 'wss://bogdanantonovich.com/vox/api'

export const client = axios.create({ baseURL: BASE, withCredentials: true })

export interface VoiceRef { file_id: string; text: string }

export const authApi = {
  login:  (login: string, password: string) => client.post('/auth/login', { login, password }),
  signUp: (login: string, password: string) => client.post('/auth/sign_up', { login, password }),
  logout: () => client.delete('/auth/logout'),
  oauth:  (provider: 'github' | 'google') => { window.location.href = `${BASE}/auth/${provider}/login` },
}

export const userApi = {
  info: () => client.get<UserInfo>('/user/info'),
  hubs: () => client.get<{ hub_ids: string[] }>('/user/hubs'),
}

export const hubApi = {
  create:   () => client.post<{ hub_id: string }>('/hub'),
  remove:   (id: string) => client.delete(`/hub/${id}`),
  listenUrl: (id: string) => `${BASE}/hub/${id}/listen`,
  publishWsUrl: (id: string, lang: string, fileId: string) =>
    `${WS_BASE}/hub/${id}/publish?lang=${lang}&file_id=${fileId}`,
}

export const voiceApi = {
  list:   () => client.get<VoiceRef[]>('/user/voice/meta'),
  upload: (blob: Blob, text: string) =>
    client.post('/user/voice', blob, {
      headers: { 'Content-Type': 'application/octet-stream' },
      params:  { text_ref: text },
    }),
  remove: (fileId: string) => client.delete('/user/voice', { params: { file_id: fileId } }),
}
