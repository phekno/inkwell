import { describe, expect, it } from 'vitest'
import { renderMarkdown } from './markdown'

describe('renderMarkdown', () => {
  it('renders basic markdown to HTML', () => {
    const html = renderMarkdown('# Hello\n\nsome **bold** text')
    expect(html).toContain('<h1')
    expect(html).toContain('Hello')
    expect(html).toContain('<strong>bold</strong>')
  })

  it('strips dangerous markup', () => {
    const html = renderMarkdown('ok <img src=x onerror="alert(1)"> <script>alert(2)<\/script>')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('<script')
  })

  it('handles empty input', () => {
    expect(renderMarkdown('')).toBe('')
  })
})
