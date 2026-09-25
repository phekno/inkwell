import { describe, expect, it } from 'vitest'
import { EditorSelection, EditorState } from '@codemirror/state'
import { ensureSyntaxTree } from '@codemirror/language'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { buildDecorations } from './livePreview'

function stateOf(doc: string, cursor?: number): EditorState {
  const state = EditorState.create({
    doc,
    selection: cursor === undefined ? undefined : EditorSelection.cursor(cursor),
    extensions: [markdown({ base: markdownLanguage })],
  })
  ensureSyntaxTree(state, state.doc.length, 5000)
  return state
}

// visible renders the doc as the user would see it: hidden ranges removed and
// widget ranges swapped for their widget's text.
function visible(state: EditorState, focused: boolean): string {
  const decos = buildDecorations(state, focused)
  let out = ''
  let pos = 0
  decos.between(0, state.doc.length, (from, to, value) => {
    if (value.spec.class) return // line/mark styling, not replacement
    out += state.doc.sliceString(pos, from)
    out += (value.spec.widget as { text?: string } | undefined)?.text ?? ''
    pos = to
  })
  return out + state.doc.sliceString(pos)
}

function lineClasses(state: EditorState, focused = false): string[] {
  const out: string[] = []
  buildDecorations(state, focused).between(0, state.doc.length, (from, to, value) => {
    if (from === to && value.spec.class) out.push(value.spec.class)
  })
  return out
}

function markClasses(state: EditorState): string[] {
  const out: string[] = []
  buildDecorations(state, false).between(0, state.doc.length, (from, to, value) => {
    if (from < to && value.spec.class) out.push(state.doc.sliceString(from, to))
  })
  return out
}

describe('buildDecorations', () => {
  it('hides heading marks and styles the heading line', () => {
    const s = stateOf('## Title\nbody')
    expect(visible(s, false)).toBe('Title\nbody')
    expect(lineClasses(s)).toContain('cm-md-h2')
  })

  it('hides emphasis, strikethrough and inline-code marks', () => {
    const s = stateOf('a **b** _c_ ~~d~~ `e`')
    expect(visible(s, false)).toBe('a b c d e')
    expect(markClasses(s)).toEqual(['`e`'])
  })

  it('shows only link text', () => {
    const s = stateOf('see [docs](https://x.test "t") now')
    expect(visible(s, false)).toBe('see docs now')
  })

  it('renders bullets as widgets, draws rules on the line, and hides quote marks', () => {
    const s = stateOf('- one\n* two\n\n---\n\n> said')
    expect(visible(s, false)).toBe('• one\n• two\n\n\n\nsaid')
    expect(lineClasses(s)).toEqual(expect.arrayContaining(['cm-md-hr', 'cm-md-quote']))
  })

  it('reveals raw markdown on the line with the cursor when focused', () => {
    const doc = '## Title\n**bold**'
    const onHeading = stateOf(doc, 3)
    expect(visible(onHeading, true)).toBe('## Title\nbold')
    // Unfocused: everything renders, regardless of where the cursor sits.
    expect(visible(onHeading, false)).toBe('Title\nbold')
    // Heading styling stays while the marks are revealed.
    expect(lineClasses(onHeading, true)).toContain('cm-md-h2')
  })

  it('leaves fenced code untouched but styles its lines', () => {
    const s = stateOf('```\n**x**\n```')
    expect(visible(s, false)).toBe('```\n**x**\n```')
    expect(lineClasses(s).filter((c) => c === 'cm-md-codeblock')).toHaveLength(3)
  })
})
