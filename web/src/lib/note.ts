import type { NoteDocument, NoteDocumentBlock } from './api/client'

export type NoteSegment =
  | { type: 'text'; text: string }
  | { type: 'image'; key: string }

export interface NoteUpload {
  text: string
  document: NoteDocument
  images: File[]
}

export function readSegments(root: Node): NoteSegment[] {
  const segments: NoteSegment[] = []
  let text = ''

  function flush() {
    const value = normalizeText(text)
    if (value) segments.push({ type: 'text', text: value })
    text = ''
  }

  function lineBreak() {
    if (text.length > 0 && !text.endsWith('\n')) text += '\n'
  }

  function walk(node: Node) {
    if (node.nodeType === Node.TEXT_NODE) {
      text += node.textContent ?? ''
      return
    }
    if (node.nodeType !== Node.ELEMENT_NODE) return

    const element = node as HTMLElement
    if (element.tagName === 'IMG') {
      const key = element.dataset.key
      if (key) {
        flush()
        segments.push({ type: 'image', key })
      }
      return
    }
    if (element.tagName === 'BR') {
      text += '\n'
      return
    }

    const block = element.tagName === 'DIV' || element.tagName === 'P'
    if (block) lineBreak()
    for (const child of Array.from(element.childNodes)) walk(child)
    if (block) lineBreak()
  }

  for (const child of Array.from(root.childNodes)) walk(child)
  flush()
  return segments
}

export function buildNoteUpload(
  segments: NoteSegment[],
  files: ReadonlyMap<string, File>,
): NoteUpload {
  const blocks: NoteDocumentBlock[] = []
  const images: File[] = []
  const indexByKey = new Map<string, number>()
  const textParts: string[] = []

  for (const segment of segments) {
    if (segment.type === 'text') {
      blocks.push({ type: 'text', text: segment.text })
      textParts.push(segment.text)
      continue
    }

    const file = files.get(segment.key)
    if (!file) continue

    let index = indexByKey.get(segment.key)
    if (index === undefined) {
      index = images.length
      images.push(file)
      indexByKey.set(segment.key, index)
    }
    blocks.push({ type: 'image', index })
  }

  return {
    text: textParts.join('\n'),
    document: { blocks },
    images,
  }
}

function normalizeText(value: string): string {
  return value
    .replace(/\r\n?/g, '\n')
    .split('\n')
    .map((line) => line.trimEnd())
    .join('\n')
    .trim()
}
