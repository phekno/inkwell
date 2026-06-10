// Thin wrappers around @aws-amplify/auth (v6). Amplify owns the session
// store; we just expose the verbs our views use and a fetchIdToken() helper
// the api client calls before each request (Amplify auto-refreshes).

import { Amplify } from 'aws-amplify'
import {
  confirmSignUp as amplifyConfirmSignUp,
  fetchAuthSession,
  resendSignUpCode,
  signIn as amplifySignIn,
  signOut as amplifySignOut,
  signUp as amplifySignUp,
} from 'aws-amplify/auth'

Amplify.configure({
  Auth: {
    Cognito: {
      userPoolId: import.meta.env.VITE_COGNITO_USER_POOL_ID,
      userPoolClientId: import.meta.env.VITE_COGNITO_CLIENT_ID,
    },
  },
})

export async function signUp(email: string, password: string): Promise<void> {
  await amplifySignUp({
    username: email,
    password,
    options: { userAttributes: { email } },
  })
}

export async function confirmSignUp(email: string, code: string): Promise<void> {
  await amplifyConfirmSignUp({ username: email, confirmationCode: code })
}

export async function resendConfirmation(email: string): Promise<void> {
  await resendSignUpCode({ username: email })
}

export async function signIn(email: string, password: string): Promise<void> {
  const res = await amplifySignIn({ username: email, password })
  if (!res.isSignedIn) {
    throw new Error(`sign-in incomplete: ${res.nextStep.signInStep}`)
  }
}

export async function signOut(): Promise<void> {
  await amplifySignOut()
}

export interface CurrentUser {
  idToken: string
  email: string | null
}

export async function currentUser(): Promise<CurrentUser | null> {
  try {
    const session = await fetchAuthSession()
    const idToken = session.tokens?.idToken
    if (!idToken) return null
    return {
      idToken: idToken.toString(),
      email: (idToken.payload.email as string | undefined) ?? null,
    }
  } catch {
    return null
  }
}

// Always-fresh id token for outgoing API calls. Amplify refreshes
// transparently when the cached token is near expiry.
export async function fetchIdToken(): Promise<string | null> {
  const session = await fetchAuthSession()
  return session.tokens?.idToken?.toString() ?? null
}
