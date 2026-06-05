import { reactive } from 'vue'

export interface UserInfo {
  id: string
  name: string
  email: string
  picture: string
}

export const auth = reactive({
  user: null as UserInfo | null,
  loading: true,
})
