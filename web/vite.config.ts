import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// amazon-cognito-identity-js (and a couple of its deps) reference Node's
// `global` — define it to globalThis so the browser bundle doesn't throw at
// startup with a blank page.
export default defineConfig({
  plugins: [vue()],
  server: { port: 5173 },
  define: {
    global: 'globalThis',
  },
})
