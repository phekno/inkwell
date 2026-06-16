import { marked } from 'marked'
import DOMPurify from 'dompurify'

// Parse markdown to HTML and sanitize before it is bound via v-html. Content is
// the user's own private journal, but sanitizing is cheap insurance and keeps
// any pasted HTML from doing something surprising.
export function renderMarkdown(src: string): string {
  if (!src) return ''
  const raw = marked.parse(src, { async: false }) as string
  return DOMPurify.sanitize(raw)
}
