import {
  Editor,
  defaultValueCtx,
  editorViewOptionsCtx,
  pasteRulesCtx,
  rootCtx,
} from '@milkdown/kit/core'
import { commonmark } from '@milkdown/kit/preset/commonmark'
import { gfm } from '@milkdown/kit/preset/gfm'
import { clipboard } from '@milkdown/kit/plugin/clipboard'
import { history } from '@milkdown/kit/plugin/history'
import { listener, listenerCtx } from '@milkdown/kit/plugin/listener'
import { upload, uploadConfig, type Uploader } from '@milkdown/kit/plugin/upload'
import { $inputRule, getMarkdown, insert, replaceAll } from '@milkdown/kit/utils'
import { InputRule } from '@milkdown/kit/prose/inputrules'
import { Fragment, Slice } from '@milkdown/kit/prose/model'
import type { EditorView } from '@milkdown/kit/prose/view'

export interface NoteUpload {
  markdown: string
  images: File[]
}

export interface NoteEditorOptions {
  initialMarkdown?: string
  fileUrl?: (path: string) => string
}

export interface NoteEditor {
  upload(): NoteUpload
  clear(): void
  insertImage(file: File): void
  destroy(): Promise<void>
}

const imageUrlRe = /^https?:\/\/\S+\.(?:png|jpe?g|gif|webp|bmp)(?:\?\S*)?$/i

const unsafeProtocol = /^(javascript|data|vbscript):/i

const storedImageMarkdownRe = /!\[([^\]]*)\]\(([^\s)]+)\)/g

function resolveStoredImages(markdown: string, fileUrlFn?: (path: string) => string): { markdown: string; keyByUrl: Map<string, string> } {
  const keyByUrl = new Map<string, string>()
  if (!fileUrlFn) return { markdown, keyByUrl }

  const result = markdown.replace(storedImageMarkdownRe, (full, alt, url) => {
    if (!url.includes('/images/')) return full
    const displayUrl = fileUrlFn(url)
    keyByUrl.set(displayUrl, url)
    return `![${alt}](${displayUrl})`
  })
  return { markdown: result, keyByUrl }
}

// Turns the markdown link syntax [text](url) into a real link as the closing parenthesis is typed.
const linkInputRule = $inputRule(
  () =>
    new InputRule(/\[([^\]]+)\]\(([^)\s]+)\)$/, (state, match, start, end) => {
      const [, text, href] = match
      const link = state.schema.marks.link
      if (!link || !text) return null
      if (unsafeProtocol.test(href)) return null
      const tr = state.tr
      tr.replaceWith(start, end, state.schema.text(text))
      tr.addMark(start, start + text.length, link.create({ href }))
      return tr
    }),
)

export async function createNoteEditor(
  container: HTMLElement,
  onChange: (markdown: string) => void = () => {},
  options: NoteEditorOptions = {},
): Promise<NoteEditor> {
  const filesByUrl = new Map<string, File>()
  const { initialMarkdown, fileUrl: fileUrlFn } = options

  const resolved = resolveStoredImages(initialMarkdown ?? '', fileUrlFn)

  const uploader: Uploader = async (files, schema) => {
    const nodes = []
    for (let i = 0; i < files.length; i++) {
      const file = files.item(i)
      if (!file || !file.type.startsWith('image/')) continue
      const url = URL.createObjectURL(file)
      filesByUrl.set(url, file)
      const node = schema.nodes.image?.createAndFill({ src: url, alt: file.name })
      if (node) nodes.push(node)
    }
    return nodes
  }

  const imageUrlPasteRule = {
    priority: 100,
    run: (slice: Slice, view: EditorView) => {
      const text = slice.content.textBetween(0, slice.content.size, '\n').trim()
      if (!imageUrlRe.test(text)) return slice
      const image = view.state.schema.nodes.image
      if (!image) return slice
      return new Slice(Fragment.from(image.create({ src: text, alt: '' })), 0, 0)
    },
  }

  const editor = await Editor.make()
    .config((ctx) => {
      ctx.set(rootCtx, container)
      ctx.set(defaultValueCtx, resolved.markdown)
      ctx.update(editorViewOptionsCtx, (prev) => ({
        ...prev,
        attributes: { class: 'capture-editor', spellcheck: 'false' },
      }))
      ctx.set(uploadConfig.key, {
        ...ctx.get(uploadConfig.key),
        uploader,
        enableHtmlFileUploader: false,
      })
      ctx.update(pasteRulesCtx, (rules) => [...rules, imageUrlPasteRule])
      ctx.get(listenerCtx).markdownUpdated((_ctx, markdown) => onChange(markdown))
    })
    .use(commonmark)
    .use(gfm)
    .use(history)
    .use(listener)
    .use(clipboard)
    .use(upload)
    .use(linkInputRule)
    .create()

  return {
    upload() {
      let raw = editor.action(getMarkdown())
      const images: File[] = []
      let index = 0
      raw = raw.replace(/blob:https?:\/\/[^\s)]+/g, (url) => {
        const file = filesByUrl.get(url)
        if (!file) return url
        images.push(file)
        return `@stash-image-${index++}`
      })
      // Reverse stored-image display URLs back to keys so the token is never saved.
      for (const [displayUrl, storedKey] of resolved.keyByUrl) {
        raw = raw.replaceAll(displayUrl, storedKey)
      }
      return { markdown: raw, images }
    },
    clear() {
      editor.action(replaceAll(''))
    },
    insertImage(file: File) {
      const url = URL.createObjectURL(file)
      filesByUrl.set(url, file)
      editor.action(insert(`![](${url})`))
    },
    async destroy() {
      for (const url of filesByUrl.keys()) URL.revokeObjectURL(url)
      filesByUrl.clear()
      await editor.destroy()
    },
  }
}
