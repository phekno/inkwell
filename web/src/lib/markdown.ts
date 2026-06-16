import { marked } from 'marked'
import DOMPurify from 'dompurify'

// Parse markdown to HTML and sanitize before it is bound via v-html. Content is
// the user's own private journal, but sanitizing is cheap insurance and keeps
// any pasted HTML from doing something surprising.
export function renderMarkdown(src: string): string {
  if (!src) return ''
  // breaks: render a single newline as <br> — Notion exports (and the TUI) treat
  // a lone newline as a real line break, not CommonMark's soft break (a space).
  const raw = marked.parse(src, { async: false, gfm: true, breaks: true }) as string
  return DOMPurify.sanitize(raw)
}
