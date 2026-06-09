# web

Vue 3 + Vite + TypeScript + Tailwind v4 (CSS-first config in `src/style.css`). Dark mode is class-based — `@custom-variant dark (&:where(.dark, .dark *))` — with a no-FOUC inline script in `index.html` that reads `localStorage.inkwell-theme` and falls back to `prefers-color-scheme`.

## Dev

```sh
npm install
npm run dev
```

## Build

```sh
npm run build
# output: dist/
```
