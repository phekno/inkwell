# Web UI: terminal-style restyle + layout fixes

Date: 2026-06-17
Status: approved (direction)

## Problem

Three issues with the Vue web client:

1. **The whole page scrolls** when only the entries pane should. A prior fix
   (`flex-1 min-h-0` on the entries section) did not work.
2. **No visible bounding box** around the editor text area, so it's unclear
   where the writing surface is.
3. The UI should feel **more terminal-like** — not a 1:1 copy of the TUI, just
   terminal-flavored styling.

## Root cause of #1

`App.vue`'s root is `<main class="min-h-screen flex flex-col">`. `min-height`
is a floor, not a cap: when entries content is tall, the flex column grows past
the viewport and the **document body** scrolls. Adding `flex-1 min-h-0` to the
entries `<section>` cannot help, because the *container* (`main`) still has no
definite height for flex-grow to distribute against or for `min-h-0` to cap.

Fix: give the app shell a **definite viewport height** and clip overflow at the
top, so inner regions are bounded and scroll internally.

## Decisions

- **Aesthetic:** "Modern terminal / muted." Keep the existing `ink` palette and
  class-based dark mode. Add monospace type, visibly bordered panels, and
  restrained terminal accents. Not a green-screen costume.
- **Font:** Self-host **JetBrains Mono** via `@fontsource/jetbrains-mono`
  (no external CDN request — fits a private journal). Applied app-wide as the
  base font.
- **Accent:** A single muted terminal-green token (`--color-term-accent`),
  desaturated for both themes, used *sparingly*: prompt markers (`▸`/`>`),
  block cursor, focus ring, and selected-row marker. Body text, borders, and
  buttons stay monochrome ink. Selection highlight stays inverse-ink (like the
  TUI), with the accent only as the prompt marker/edge.

## Design

### 1. Layout shell (fixes #1)

`App.vue` becomes a fixed terminal-window frame:

```
<main class="h-dvh flex flex-col overflow-hidden">   <!-- definite height, no page scroll -->
  <header class="shrink-0 ...">                       <!-- title bar -->
  <div class="flex-1 min-h-0 overflow-hidden">        <!-- content region, bounded -->
    <RouterView />
  </div>
</main>
```

`RouterView` is wrapped in a bounded `flex-1 min-h-0` div rather than relying on
`class` fall-through, because `Entries.vue` has a fragment root (section +
dialog) and Vue does not forward attributes to multi-root components.

`Entries.vue` `<section>` fills that region:

```
<section class="h-full grid md:grid-cols-[20rem_1fr] grid-rows-1 overflow-hidden">
```

`grid-rows-1` resolves to `minmax(0, 1fr)`, so the single row fills the
definite height and its children are bounded; the sidebar and article keep
`overflow-y-auto min-h-0` and scroll independently. No magic numbers; the page
cannot scroll.

Auth views (`SignIn`, `SignUp`, `ConfirmSignUp`) render in the same bounded
region; they get `h-full overflow-y-auto` so a short viewport scrolls the form
*within* the frame, never the page.

### 2. Theme foundation

In `style.css`:
- `@import "@fontsource/jetbrains-mono"` (weights 400/500/700) and set the
  base `font-family` to JetBrains Mono + monospace fallbacks on `body`.
- Add theme tokens: keep `ink-*`; add `--color-term-accent` (light + dark) and
  a `--color-term-border` that's a touch stronger than `ink-100/800` so panel
  boxes read clearly.
- Keep `.prose-ink`; retune heading sizes slightly smaller/tighter so rendered
  markdown reads terminal-ish, and ensure `code`/`pre`/`blockquote` use the
  mono + border treatment.

### 3. Components → bordered panes + terminal accents

- **App frame & panels:** the shell, sidebar, editor, and preview are each a
  visibly bordered box using `--color-term-border`.
- **Editor (fixes #2):** the body textarea sits in an **always-visible**
  bordered, padded box, introduced by a `body` label (mirrors the TUI). The
  preview is a matching box. The focus state brightens the border + shows the
  accent ring — but the box is visible at rest.
- **Buttons:** terminal-style `[ label ]` — monospace, bracketed/boxed, ink
  foreground, accent on hover/active where it reads well.
- **Inputs:** prompt marker (`›`) and a block-cursor accent; focus ring uses
  the accent.
- **List rows / folder tree:** selected row shows a `>` prompt marker in accent
  plus inverse-ink highlight; folder rows keep the `▸/▾` disclosure.
- **Move dialog:** bordered terminal box, mono controls.
- **Dark-mode toggle:** mono, bracketed, consistent with buttons.

### 4. Markdown rendering

Keep the `.prose-ink` block spacing from the previous fix; adjust heading scale
for the monospace context and confirm `pre`/`code`/`blockquote`/`hr` render as
bordered/tinted terminal blocks. No change to the `marked` + DOMPurify pipeline.

## Out of scope

- No changes to the TUI, API, or infra.
- No change to auth flow, routing, or data fetching.
- No new color-theme switcher (single ink palette + dark mode only).
- No CRT effects (scanlines/glow) — that was the rejected "phosphor" option.

## Testing & verification

- Existing `vitest` suite stays green; `npm run build` (incl. `vue-tsc`) clean.
- Add a small unit assertion where it adds value (e.g. the layout classes are
  present) only if it's meaningful; this work is primarily visual.
- **Manual browser verification is required, not optional:** run the dev
  server and confirm in a browser that
  (a) scrolling the entries list does **not** scroll the page (only the pane),
  (b) the editor bounding box is visible at rest, and
  (c) the terminal styling renders in both light and dark mode.
  Capture this before claiming done.
