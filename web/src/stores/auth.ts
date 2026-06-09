import { defineStore } from 'pinia'
import { currentSession, signIn as cognitoSignIn, signOut as cognitoSignOut } from '../lib/cognito'

interface AuthState {
  email: string | null
  idToken: string | null
  ready: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ email: null, idToken: null, ready: false }),
  getters: {
    isSignedIn: (s) => !!s.idToken,
  },
  actions: {
    async hydrate() {
      const session = await currentSession()
      if (session) {
        this.email = session.getIdToken().payload.email ?? null
        this.idToken = session.getIdToken().getJwtToken()
      }
      this.ready = true
    },
    async signIn(email: string, password: string) {
      const session = await cognitoSignIn(email, password)
      this.email = email
      this.idToken = session.getIdToken().getJwtToken()
    },
    signOut() {
      cognitoSignOut()
      this.email = null
      this.idToken = null
    },
  },
})
