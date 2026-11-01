import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { fileUrl } from './api/client'

const remoteRe = /^https?:\/\//i

export function renderMarkdown(markdown: string): string {
  const raw = marked.parse(markdown, { gfm: true, breaks: true }) as string
  const fragment = DOMPurify.sanitize(raw, { RETURN_DOM_FRAGMENT: true }) as DocumentFragment

  fragment.querySelectorAll('img').forEach((img) => {
    const src = img.getAttribute('src') ?? ''
    if (remoteRe.test(src)) {
      img.setAttribute('referrerpolicy', 'no-referrer')
      img.setAttribute('loading', 'lazy')
    } else if (src && !src.startsWith('data:')) {
      img.setAttribute('src', fileUrl(src))
      img.setAttribute('loading', 'lazy')
    }
  })

  const container = document.createElement('div')
  container.appendChild(fragment)
  return container.innerHTML
}

export function markdownToPlainText(markdown: string): string {
  const raw = marked.parse(markdown, { gfm: true, breaks: true }) as string
  const container = document.createElement('div')
  container.innerHTML = DOMPurify.sanitize(raw)

  for (const img of Array.from(container.querySelectorAll('img'))) {
    img.replaceWith(document.createTextNode(` ${img.getAttribute('alt') || '📷'} `))
  }
  for (const br of Array.from(container.querySelectorAll('br'))) {
    br.replaceWith(document.createTextNode('\n'))
  }
  for (const block of Array.from(container.querySelectorAll('p, h1, h2, h3, h4, h5, h6, li, blockquote, pre'))) {
    block.appendChild(document.createTextNode('\n'))
  }

  return (container.textContent ?? '')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n{2,}/g, '\n')
    .trim()
}
