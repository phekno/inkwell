# Web Terminal-Style Restyle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle the Vue web client to feel terminal-like (monospace, bordered panes, restrained accents), make the editor's writing surface a visible bounding box, and fix the bug where the whole page scrolls instead of just the entries pane.

**Architecture:** Lock the app to a definite-height viewport frame (`h-dvh` + `overflow-hidden`) so inner regions are bounded and scroll internally. Centralize terminal chrome (panes, buttons, inputs, prompts) in a handful of reusable CSS component classes in `style.css`, then apply them across the existing components. Keep the `marked`+DOMPurify pipeline and all data/auth logic untouched.

**Tech Stack:** Vue 3 SFC + Vite + TypeScript, Tailwind CSS v4 (`@import "tailwindcss"` in `src/style.css`, class-based dark mode via `@custom-variant dark`), Vitest + jsdom, self-hosted `@fontsource/jetbrains-mono`.

## Global Constraints

- Keep the existing `ink` palette (`--color-ink-50/100/800/900`) and class-based dark mode. No new theme switcher.
- Font is self-hosted via `@fontsource/jetbrains-mono` — **no external CDN request** (private journal).
- No changes to the TUI, API, or infra. No changes to auth flow, routing, data fetching, or the `marked`+DOMPurify markdown pipeline.
- No CRT effects (scanlines/glow) — the "phosphor" option was rejected.
- Accent (`--term-accent`) is used **sparingly**: prompt markers, block/caret cursor, focus ring, selected-row marker. Body text, borders, and button text stay monochrome ink.
- Tailwind v4: `Width()`-style gotchas don't apply here, but remember padding is inside an element's box; size panes with `min-h-0` where they must shrink inside fl/grid parents.
- After every task: `npm run build` (runs `vue-tsc --noEmit` then `vite build`) must be clean, and `npm run test` (vitest) must stay green.
- jsdom does no layout, so the scroll fix (#1) and the visible-box fix (#2) **cannot** be asserted in vitest. They are verified manually in a real browser (Task 7), which is a required deliverable, not optional.
- All commands below run from `web/` unless stated otherwise.

---

### Task 1: Theme foundation — font, tokens, terminal component classes

**Files:**
- Modify: `web/package.json` (add `@fontsource/jetbrains-mono` dependency)
- Modify: `web/src/main.ts` (import the font CSS)
- Modify: `web/src/style.css` (base mono font, accent/border CSS vars, reusable component classes)

**Interfaces:**
- Produces (CSS classes used by every later task):
  - `.pane` — a bordered, rounded terminal box (border uses `--term-border`).
  - `.btn-term` — bracketed/boxed monospace button; accent on hover.
  - `.input-term` — transparent bordered input/textarea/select; accent caret + focus ring.
  - `.prompt-accent` — `color: var(--term-accent)` for prompt markers `>` / `›` / `▸`.
- Produces CSS vars on `:root` / `.dark`: `--term-accent`, `--term-border`.

- [ ] **Step 1: Add the font dependency**

Run:
```bash
npm install @fontsource/jetbrains-mono@^5
```
Expected: `package.json` gains `"@fontsource/jetbrains-mono"` under `dependencies`; install succeeds.

- [ ] **Step 2: Import the font weights in the app entry**

In `web/src/main.ts`, add these imports above `import './style.css'`:
```ts
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import '@fontsource/jetbrains-mono/700.css'
```

- [ ] **Step 3: Add base font, tokens, and terminal component classes to `style.css`**

In `web/src/style.css`, after the existing `@theme { … }` block, append:
```css
/* Terminal theme tokens. Accent stays muted in both themes and is used
   sparingly (prompts, cursor, focus, selection marker). The border token is a
   touch stronger than ink-100/800 so panel boxes read clearly. */
:root {
  --term-accent: #2f8f5b;
  --term-border: #d6d6d2;
}
.dark {
  --term-accent: #5ec98a;
  --term-border: #3a3a37;
}

/* JetBrains Mono app-wide (self-hosted via @fontsource, imported in main.ts). */
html, body {
  font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

/* A bordered terminal panel. */
.pane {
  border: 1px solid var(--term-border);
  border-radius: 0.375rem;
  background: transparent;
}

/* Bracketed/boxed monospace button. */
.btn-term {
  font: inherit;
  border: 1px solid var(--term-border);
  border-radius: 0.25rem;
  padding: 0.3rem 0.7rem;
  background: transparent;
  transition: color 0.15s, border-color 0.15s, background-color 0.15s;
}
.btn-term:hover:not(:disabled) {
  color: var(--term-accent);
  border-color: var(--term-accent);
}
.btn-term:disabled { opacity: 0.5; }

/* Transparent bordered text controls with an accent caret + focus ring. */
.input-term {
  font: inherit;
  background: transparent;
  border: 1px solid var(--term-border);
  border-radius: 0.25rem;
  padding: 0.4rem 0.6rem;
  caret-color: var(--term-accent);
}
.input-term:focus {
  outline: none;
  border-color: var(--term-accent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--term-accent) 35%, transparent);
}

.prompt-accent { color: var(--term-accent); }
```

- [ ] **Step 4: Build and test**

Run:
```bash
npm run build && npm run test
```
Expected: build succeeds (no vue-tsc/vite errors), all vitest tests pass.

- [ ] **Step 5: Commit**

```bash
cd /Users/jfechner/personal/inkwell
git add web/package.json web/package-lock.json web/src/main.ts web/src/style.css
git commit -m "feat(web): terminal theme foundation — JetBrains Mono + tokens + component classes"
```

---

### Task 2: App shell — definite-height viewport frame (fixes #1 scroll)

**Files:**
- Modify: `web/src/App.vue`
- Modify: `web/src/views/Entries.vue` (section root classes only)
- Modify: `web/src/views/SignIn.vue`, `web/src/views/SignUp.vue`, `web/src/views/ConfirmSignUp.vue` (section root classes only — make the form scroll within the frame)

**Interfaces:**
- Produces: a bounded `flex-1 min-h-0 overflow-hidden` content region wrapping `<RouterView />`; every route renders into a definite-height box and must use `h-full` + its own internal scroll.

- [ ] **Step 1: Make `App.vue` a fixed terminal-window frame**

In `web/src/App.vue`, replace the `<template>` block with:
```vue
<template>
  <main class="h-dvh flex flex-col overflow-hidden">
    <header class="shrink-0 flex items-center justify-between px-6 py-4 border-b border-ink-100 dark:border-ink-800">
      <RouterLink to="/" class="text-lg font-bold tracking-tight">inkwell</RouterLink>
      <div class="flex items-center gap-3">
        <template v-if="auth.isSignedIn">
          <span class="text-sm opacity-70">{{ auth.email }}</span>
          <button class="btn-term text-sm" @click="signOut">[ sign out ]</button>
        </template>
        <DarkModeToggle />
      </div>
    </header>

    <div class="flex-1 min-h-0 overflow-hidden">
      <RouterView />
    </div>
  </main>
</template>
```

- [ ] **Step 2: Make the Entries section fill the bounded region**

In `web/src/views/Entries.vue`, change the `<section>` opening tag from:
```
  <section class="grid md:grid-cols-[20rem_1fr] grid-rows-1 flex-1 min-h-0 overflow-hidden">
```
to:
```
  <section class="h-full grid md:grid-cols-[20rem_1fr] grid-rows-1 overflow-hidden">
```
(The `aside` and `article` keep their existing `overflow-y-auto min-h-0`.)

- [ ] **Step 3: Make auth views scroll within the frame, not the page**

In each of `web/src/views/SignIn.vue`, `web/src/views/SignUp.vue`, `web/src/views/ConfirmSignUp.vue`, change the root `<section>` opening tag from:
```
  <section class="flex items-center justify-center px-6 py-12">
```
to:
```
  <section class="h-full overflow-y-auto flex items-center justify-center px-6 py-12">
```

- [ ] **Step 4: Build and test**

Run:
```bash
npm run build && npm run test
```
Expected: build clean, vitest green.

- [ ] **Step 5: Browser smoke check (manual)**

Run `npm run dev`, open the entries view with enough entries to overflow, scroll the entries list.
Expected: only the list pane scrolls; the page/header stays fixed; no document-level scrollbar. (Full verification with screenshots is Task 7; this is a quick confirm before committing.)

- [ ] **Step 6: Commit**

```bash
cd /Users/jfechner/personal/inkwell
git add web/src/App.vue web/src/views/Entries.vue web/src/views/SignIn.vue web/src/views/SignUp.vue web/src/views/ConfirmSignUp.vue
git commit -m "fix(web): lock app to viewport frame so only panes scroll, not the page"
```

---

### Task 3: Entries view + editor — bordered panes and visible writing box (fixes #2)

**Files:**
- Modify: `web/src/views/Entries.vue` (sidebar/article chrome, buttons, entry header)
- Modify: `web/src/components/EntryEditor.vue` (always-visible bordered body box + matching preview box)

**Interfaces:**
- Consumes: `.pane`, `.btn-term`, `.input-term`, `.prompt-accent` from Task 1.

- [ ] **Step 1: Restyle the sidebar header + buttons in `Entries.vue`**

In `web/src/views/Entries.vue`, in the sidebar header `div`, replace the `+ new` button with:
```vue
        <button class="btn-term text-sm" @click="startCompose('')">[ + new ]</button>
```
In the entry detail header, replace the three action buttons (`edit`, `move`, `delete`) with:
```vue
            <button class="btn-term text-sm" @click="startEdit">[ edit ]</button>
            <button class="btn-term text-sm" @click="showMove = true">[ move ]</button>
            <button
              class="btn-term text-sm text-red-700 dark:text-red-300 hover:!text-red-600 hover:!border-red-600"
              @click="remove(selected.id)"
            >[ delete ]</button>
```

- [ ] **Step 2: Give the article title a prompt marker**

In `web/src/views/Entries.vue`, replace the entry title `<h2>` line:
```vue
          <h2 class="text-2xl font-medium">{{ selected.title }}</h2>
```
with:
```vue
          <h2 class="text-xl font-bold"><span class="prompt-accent">#</span> {{ selected.title }}</h2>
```

- [ ] **Step 3: Make the editor body a visible bordered box with a label (fixes #2)**

In `web/src/components/EntryEditor.vue`, replace the `<template>` body grid (the `<div class="grid grid-cols-2 …">` block) with:
```vue
    <div class="grid grid-cols-2 gap-4 flex-1 min-h-0">
      <div class="flex flex-col min-h-0">
        <span class="text-xs opacity-60 mb-1 prompt-accent">› body</span>
        <div class="pane flex-1 min-h-0 p-3 focus-within:border-[var(--term-accent)] transition">
          <textarea
            v-model="body"
            placeholder="Write…"
            class="w-full h-full bg-transparent resize-none focus:outline-none leading-relaxed text-sm"
          ></textarea>
        </div>
      </div>
      <div class="flex flex-col min-h-0">
        <span class="text-xs opacity-60 mb-1">preview</span>
        <div class="pane prose-ink flex-1 min-h-0 overflow-y-auto p-4 leading-relaxed" v-html="preview"></div>
      </div>
    </div>
```

- [ ] **Step 4: Restyle the editor title input + action buttons**

In `web/src/components/EntryEditor.vue`, change the title `<input>` class to use the prompt look:
```vue
    <input
      v-model="title"
      placeholder="Title"
      class="input-term w-full text-lg font-bold mb-4"
    />
```
Replace the save/cancel buttons with:
```vue
    <div class="mt-4 flex gap-2">
      <button class="btn-term text-sm" :disabled="saving" @click="save">
        {{ saving ? '[ saving… ]' : '[ save ]' }}
      </button>
      <button class="btn-term text-sm" @click="emit('cancel')">[ cancel ]</button>
    </div>
```

- [ ] **Step 5: Build and test**

Run:
```bash
npm run build && npm run test
```
Expected: build clean, vitest green.

- [ ] **Step 6: Commit**

```bash
cd /Users/jfechner/personal/inkwell
git add web/src/views/Entries.vue web/src/components/EntryEditor.vue
git commit -m "feat(web): bordered entry panes + visible editor writing box"
```

---

### Task 4: Folder tree, list rows, move dialog, dark toggle — terminal accents

**Files:**
- Modify: `web/src/components/FolderTree.vue`
- Modify: `web/src/components/FolderTreeNode.vue`
- Modify: `web/src/components/MoveDialog.vue`
- Modify: `web/src/components/DarkModeToggle.vue`

**Interfaces:**
- Consumes: `.pane`, `.btn-term`, `.input-term`, `.prompt-accent` from Task 1.

- [ ] **Step 1: Selected list row shows an accent `>` prompt in `FolderTree.vue`**

In `web/src/components/FolderTree.vue`, replace the entry `<button>` with a version that renders a prompt marker when selected:
```vue
    <button
      v-for="e in tree.entries"
      :key="e.id"
      class="w-full text-left px-3 py-1.5 truncate hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
      :class="selectedId === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
      style="padding-left: 12px"
      @click="emit('select', e)"
    ><span class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
```

- [ ] **Step 2: Same prompt treatment for entries inside `FolderTreeNode.vue`**

In `web/src/components/FolderTreeNode.vue`, replace the nested entry `<button>` (the one bound to `select(e)`) with:
```vue
      <button
        v-for="e in node.entries"
        :key="e.id"
        class="w-full text-left px-3 py-1.5 truncate hover:bg-ink-100/50 dark:hover:bg-ink-800/50"
        :class="selectedId === e.id ? 'bg-ink-100 dark:bg-ink-800' : ''"
        :style="{ paddingLeft: pad(depth + 1) }"
        @click="select(e)"
      ><span class="prompt-accent" :class="selectedId === e.id ? '' : 'opacity-0'">&gt;</span> {{ e.title }}</button>
```
And change the `+ new here` button text to terminal style:
```vue
      >[ + new here ]</button>
```

- [ ] **Step 3: Terminal box + controls for `MoveDialog.vue`**

In `web/src/components/MoveDialog.vue`, change the dialog card and controls:
- Card `<div>` class from `bg-ink-50 dark:bg-ink-900 border border-ink-100 dark:border-ink-800 rounded-lg p-5 w-80` to:
  ```
  pane bg-ink-50 dark:bg-ink-900 p-5 w-80
  ```
- The `<select>` class: replace `w-full bg-transparent border border-ink-100 dark:border-ink-800 rounded-md px-2 py-1.5 text-sm mb-3` with `input-term w-full text-sm mb-3`.
- The `<input>` class: replace `w-full bg-transparent border border-ink-100 dark:border-ink-800 rounded-md px-2 py-1.5 text-sm` with `input-term w-full text-sm`.
- The two buttons: replace both with `class="btn-term text-sm"` and text `[ move ]` and `[ cancel ]`.
- The heading: prefix with a prompt marker — `<h3 class="text-sm font-bold mb-3"><span class="prompt-accent">›</span> move to folder</h3>`.

- [ ] **Step 4: Terminal style for `DarkModeToggle.vue`**

In `web/src/components/DarkModeToggle.vue`, change the button to:
```vue
  <button
    type="button"
    class="btn-term text-sm"
    :aria-pressed="isDark"
    @click="isDark = !isDark"
  >
    {{ isDark ? '[ ☾ dark ]' : '[ ☀ light ]' }}
  </button>
```

- [ ] **Step 5: Build and test**

Run:
```bash
npm run build && npm run test
```
Expected: build clean, vitest green.

- [ ] **Step 6: Commit**

```bash
cd /Users/jfechner/personal/inkwell
git add web/src/components/FolderTree.vue web/src/components/FolderTreeNode.vue web/src/components/MoveDialog.vue web/src/components/DarkModeToggle.vue
git commit -m "feat(web): terminal accents for folder tree, move dialog, dark toggle"
```

---

### Task 5: Auth views — terminal forms

**Files:**
- Modify: `web/src/views/SignIn.vue`
- Modify: `web/src/views/SignUp.vue`
- Modify: `web/src/views/ConfirmSignUp.vue`

**Interfaces:**
- Consumes: `.pane`, `.btn-term`, `.input-term`, `.prompt-accent` from Task 1.

All three forms share the same input class string and button pattern. In each file, every text `<input>` carries this exact class string:
```
mt-1 w-full rounded-md px-3 py-2 bg-transparent border border-ink-100 dark:border-ink-800 focus:outline-none focus:ring-2 focus:ring-ink-800 dark:focus:ring-ink-100
```
and every submit `<button>` carries:
```
w-full rounded-md px-3 py-2 bg-ink-800 dark:bg-ink-100 text-ink-50 dark:text-ink-900 hover:opacity-90 transition disabled:opacity-50
```

- [ ] **Step 1: `SignIn.vue`**

In `web/src/views/SignIn.vue`:
- `<form @submit.prevent="submit" class="w-full max-w-sm space-y-4">` → `<form @submit.prevent="submit" class="pane w-full max-w-sm space-y-4 p-6">`
- `<h2 class="text-2xl font-medium">Sign in</h2>` → `<h2 class="text-xl font-bold"><span class="prompt-accent">›</span> sign in</h2>`
- Replace **both** input class strings (shown above) with `input-term mt-1 w-full`.
- Replace the submit button class string (shown above) with `btn-term w-full`, and its text `{{ submitting ? 'signing in…' : 'sign in' }}` with `{{ submitting ? '[ signing in… ]' : '[ sign in ]' }}`.

- [ ] **Step 2: `SignUp.vue`**

In `web/src/views/SignUp.vue`:
- `<form @submit.prevent="submit" class="w-full max-w-sm space-y-4">` → `<form @submit.prevent="submit" class="pane w-full max-w-sm space-y-4 p-6">`
- `<h2 class="text-2xl font-medium">Create an account</h2>` → `<h2 class="text-xl font-bold"><span class="prompt-accent">›</span> create an account</h2>`
- Replace **both** input class strings with `input-term mt-1 w-full`.
- Replace the submit button class string with `btn-term w-full`, and its text `{{ submitting ? 'creating…' : 'create account' }}` with `{{ submitting ? '[ creating… ]' : '[ create account ]' }}`.

- [ ] **Step 3: `ConfirmSignUp.vue`**

In `web/src/views/ConfirmSignUp.vue`:
- `<form @submit.prevent="submit" class="w-full max-w-sm space-y-4">` → `<form @submit.prevent="submit" class="pane w-full max-w-sm space-y-4 p-6">`
- `<h2 class="text-2xl font-medium">Confirm your email</h2>` → `<h2 class="text-xl font-bold"><span class="prompt-accent">›</span> confirm your email</h2>`
- Replace **both** input class strings with `input-term mt-1 w-full`.
- Replace the submit button class string with `btn-term w-full`, and its text `{{ submitting ? 'confirming…' : 'confirm' }}` with `{{ submitting ? '[ confirming… ]' : '[ confirm ]' }}`.
- Leave the inline `resend` text-button (`class="underline"`) as-is.

- [ ] **Step 4: Build and test**

Run:
```bash
npm run build && npm run test
```
Expected: build clean, vitest green.

- [ ] **Step 5: Commit**

```bash
cd /Users/jfechner/personal/inkwell
git add web/src/views/SignIn.vue web/src/views/SignUp.vue web/src/views/ConfirmSignUp.vue
git commit -m "feat(web): terminal-style auth forms"
```

---

### Task 6: Markdown prose tuning for monospace

**Files:**
- Modify: `web/src/style.css` (`.prose-ink` rules)

**Interfaces:**
- Consumes: `--term-border`, `--term-accent` from Task 1.

- [ ] **Step 1: Tune `.prose-ink` heading scale + code/quote blocks for the mono context**

In `web/src/style.css`, update the `.prose-ink` rules so headings are tighter and code/quote blocks read terminal-ish. Replace the existing heading + `pre`/`code`/`blockquote`/`hr` rules with:
```css
.prose-ink h1 { font-size: 1.35rem; font-weight: 700; margin: 1.4rem 0 0.5rem; }
.prose-ink h2 { font-size: 1.15rem; font-weight: 700; margin: 1.2rem 0 0.5rem; }
.prose-ink h3 { font-size: 1rem;    font-weight: 700; margin: 1rem 0 0.4rem; }
.prose-ink blockquote {
  border-left: 2px solid var(--term-accent);
  padding-left: 0.85rem;
  margin: 0.75rem 0;
  opacity: 0.85;
}
.prose-ink pre {
  border: 1px solid var(--term-border);
  border-radius: 0.375rem;
  padding: 0.7rem 0.9rem;
  margin: 0.75rem 0;
  overflow-x: auto;
}
.prose-ink code { font-size: 0.9em; }
.prose-ink hr { margin: 1.4rem 0; border-color: var(--term-border); }
```
(Leave the `p`, `ul`, `ol`, `li`, `a`, and `> :first-child` rules as they are.)

- [ ] **Step 2: Build and test**

Run:
```bash
npm run build && npm run test
```
Expected: build clean, vitest green (the `markdown.test.ts` assertions are about HTML output, not CSS, so they remain valid).

- [ ] **Step 3: Commit**

```bash
cd /Users/jfechner/personal/inkwell
git add web/src/style.css
git commit -m "feat(web): tune rendered-markdown styling for the terminal theme"
```

---

### Task 7: Full browser verification (required deliverable)

**Files:** none (verification only)

**Interfaces:** none.

- [ ] **Step 1: Start the dev server**

Run:
```bash
npm run dev
```
Note the local URL (e.g. http://localhost:5173).

- [ ] **Step 2: Verify #1 — only the entries pane scrolls**

Sign in, open the entries view with enough entries to overflow the sidebar. Scroll the entries list.
Expected: the list pane scrolls internally; the header and overall page do **not** move; there is no document-level scrollbar. Confirm at a normal and a short window height. Capture a screenshot.

- [ ] **Step 3: Verify #2 — editor box is visible**

Click `[ + new ]` (and also `[ edit ]` on an existing entry). 
Expected: the body writing area is enclosed in a clearly visible bordered box labeled `› body`, with a matching `preview` box beside it; the border is visible at rest (not only on focus). Capture a screenshot.

- [ ] **Step 4: Verify #3 — terminal styling in both themes**

Toggle dark/light with the `[ ☾ dark ]` / `[ ☀ light ]` button. Open an entry, the editor, the move dialog, and an auth screen.
Expected: monospace throughout, bordered panes, bracketed buttons, accent prompt markers/cursor, readable in both themes. Capture light + dark screenshots.

- [ ] **Step 5: Record results**

Note pass/fail for each of #1, #2, #3 with the screenshots. If any fail, return to the relevant task before claiming completion. Stop the dev server.

---

## Notes for the implementer

- Tailwind v4 generates utilities from `src/style.css`'s `@import "tailwindcss"`. The plain CSS rules and component classes added here live *after* that import, so they override Preflight and are not purged (they're real CSS, not utility classes).
- `h-dvh` (dynamic viewport height) is preferred over `h-screen` so mobile browser chrome doesn't cause a 1px page scroll. Tailwind v4 ships `h-dvh`.
- Keep every `v-model`, `@click`, `:disabled`, prop, and emit exactly as-is — these tasks change classes and button label text only, never behavior.
