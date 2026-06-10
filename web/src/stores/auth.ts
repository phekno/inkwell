import { defineStore } from 'pinia'
import { currentUser, signIn as cognitoSignIn, signOut as cognitoSignOut } from '../lib/cognito'

interface AuthState {
  email: string | null
  signedIn: boolean
  ready: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ email: null, signedIn: false, ready: false }),
  getters: {
    isSignedIn: (s) => s.signedIn,
  },
  actions: {
    async hydrate() {
      const u = await currentUser()
      if (u) {
        this.email = u.email
        this.signedIn = true
      }
      this.ready = true
    },
    async signIn(email: string, password: string) {
      await cognitoSignIn(email, password)
      const u = await currentUser()
      this.email = u?.email ?? email
      this.signedIn = true
    },
    async signOut() {
      await cognitoSignOut()
      this.email = null
      this.signedIn = false
    },
  },
})
