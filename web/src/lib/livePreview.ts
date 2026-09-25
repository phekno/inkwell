import { type EditorState, type Range } from '@codemirror/state'
import {
  Decoration, type DecorationSet, EditorView, ViewPlugin, type ViewUpdate, WidgetType,
} from '@codemirror/view'
import { syntaxTree } from '@codemirror/language'

// Obsidian-style live preview for CodeMirror's markdown mode: the document
// stays plain markdown, but syntax marks (##, **, [](url), > …) are hidden
// and headings/quotes/code get line styling. Marks are revealed on whichever
// lines the cursor/selection touches while the editor is focused, so they can
// still be edited.

class TextWidget extends WidgetType {
  constructor(readonly text: string, readonly className: string) {
    super()
  }
  eq(other: TextWidget) {
    return other.text === this.text && other.className === this.className
  }
  toDOM() {
    const span = document.createElement('span')
    span.className = this.className
    span.textContent = this.text
    return span
  }
}

const hide = Decoration.replace({})
const bullet = Decoration.replace({ widget: new TextWidget('•', 'cm-md-bullet') })
const inlineCode = Decoration.mark({ class: 'cm-md-inline-code' })
const lineClass = (cls: string) => Decoration.line({ class: cls })

function activeLines(state: EditorState): Set<number> {
  const lines = new Set<number>()
  for (const r of state.selection.ranges) {
    const last = state.doc.lineAt(r.to).number
    for (let n = state.doc.lineAt(r.from).number; n <= last; n++) lines.add(n)
  }
  return lines
}

export function buildDecorations(
  state: EditorState,
  focused: boolean,
  ranges: readonly { from: number; to: number }[] = [{ from: 0, to: state.doc.length }],
): DecorationSet {
  const doc = state.doc
  const active = focused ? activeLines(state) : new Set<number>()
  const out: Range<Decoration>[] = []

  // Replace decorations may not cross line breaks, and are skipped on active
  // lines so the raw markdown shows where the user is typing.
  const replace = (from: number, to: number, deco = hide) => {
    if (from >= to) return
    const line = doc.lineAt(from)
    if (to > line.to || active.has(line.number)) return
    out.push(deco.range(from, to))
  }
  const styleLines = (from: number, to: number, cls: string) => {
    const last = doc.lineAt(to).number
    for (let n = doc.lineAt(from).number; n <= last; n++) {
      out.push(lineClass(cls).range(doc.line(n).from))
    }
  }
  // Hide a mark plus one following space ("## ", "> ").
  const hideWithSpace = (from: number, to: number) =>
    replace(from, doc.sliceString(to, to + 1) === ' ' ? to + 1 : to)

  for (const { from, to } of ranges) {
    syntaxTree(state).iterate({
      from,
      to,
      enter: (node) => {
        const name = node.name
        const heading = /^ATXHeading(\d)$/.exec(name)
        if (heading) {
          styleLines(node.from, node.to, `cm-md-h${heading[1]}`)
          return
        }
        switch (name) {
          case 'HeaderMark':
            // Leading "## " — a closing "##" just gets hidden on its own.
            if (node.from === doc.lineAt(node.from).from) hideWithSpace(node.from, node.to)
            else replace(node.from, node.to)
            break
          case 'EmphasisMark':
          case 'StrikethroughMark':
            replace(node.from, node.to)
            break
          case 'InlineCode':
            out.push(inlineCode.range(node.from, node.to))
            break
          case 'CodeMark':
            if (node.node.parent?.name === 'InlineCode') replace(node.from, node.to)
            break
          case 'QuoteMark':
            hideWithSpace(node.from, node.to)
            break
          case 'Blockquote':
            styleLines(node.from, node.to, 'cm-md-quote')
            break
          case 'FencedCode':
            styleLines(node.from, node.to, 'cm-md-codeblock')
            return false // leave code contents alone
          case 'ListMark':
            if (node.node.parent?.parent?.name === 'BulletList') replace(node.from, node.to, bullet)
            break
          case 'HorizontalRule':
            // The line itself draws the rule (CSS); hide the dashes.
            styleLines(node.from, node.to, 'cm-md-hr')
            replace(node.from, node.to)
            break
          case 'Link': {
            // "[text](url "title")" → "text": hide the opening "[" and
            // everything from the closing "]" on.
            const marks = node.node.getChildren('LinkMark')
            if (marks.length >= 2) {
              replace(marks[0].from, marks[0].to)
              replace(marks[1].from, node.to)
            }
            break
          }
        }
      },
    })
  }

  return Decoration.set(out, true)
}

export const livePreview = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    constructor(view: EditorView) {
      this.decorations = buildDecorations(view.state, view.hasFocus, view.visibleRanges)
    }
    update(u: ViewUpdate) {
      if (
        u.docChanged || u.selectionSet || u.viewportChanged || u.focusChanged ||
        syntaxTree(u.startState) !== syntaxTree(u.state)
      ) {
        this.decorations = buildDecorations(u.state, u.view.hasFocus, u.view.visibleRanges)
      }
    }
  },
  { decorations: (v) => v.decorations },
)
