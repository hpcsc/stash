import { describe, expect, it } from 'vitest'

const TEXT_NODE = 3
const ELEMENT_NODE = 1

function textNode(content) {
  return { nodeType: TEXT_NODE, textContent: content }
}

function element(tag, ...children) {
  return {
    nodeType: ELEMENT_NODE,
    tagName: tag.toUpperCase(),
    childNodes: children.flat(),
  }
}

function img(src, alt = '') {
  const el = element('img')
  el.currentSrc = src
  el.src = src
  el.alt = alt
  return el
}

function fragment(...children) {
  return { childNodes: children.flat() }
}

// Inlined walker matching extension/src/selection.js

function newState() {
  return { markdown: '', images: [], pending: '', lastWasImage: false }
}

function walkFragment(fragment) {
  const state = newState()
  for (const node of fragment.childNodes) {
    walkNode(node, state)
  }
  if (state.pending) {
    state.markdown += state.pending
    state.pending = ''
  }
  const result = state.markdown.trim()
  return { markdown: result, images: state.images }
}

function walkNode(node, state, inBlock = false) {
  switch (node.nodeType) {
    case TEXT_NODE:
      state.pending += node.textContent
      if (node.textContent.trim()) state.lastWasImage = false
      return

    case ELEMENT_NODE: {
      const tag = node.tagName?.toLowerCase()

      if (tag === 'script' || tag === 'style' || tag === 'noscript') return

      if (tag === 'img') {
        flushPending(state)
        if (!inBlock && state.markdown.length > 0 && !state.markdown.endsWith('\n\n') && !state.lastWasImage) {
          state.markdown += '\n\n'
        }
        const src = node.currentSrc || node.src
        if (src) {
          const idx = state.images.length
          state.images.push({ src, alt: node.alt || '' })
          state.markdown += `![${node.alt || ''}](@stash-image-${idx})`
          state.lastWasImage = !inBlock
        }
        return
      }

      if (tag === 'br') {
        flushPending(state)
        state.markdown += '\n'
        state.lastWasImage = false
        return
      }

      if (isBlock(tag)) {
        flushPending(state)
        if (state.markdown.length > 0 && !state.markdown.endsWith('\n\n')) {
          state.markdown += '\n\n'
        }
        state.lastWasImage = false
        for (const child of node.childNodes) {
          walkNode(child, state, true)
        }
        flushPending(state)
        return
      }

      for (const child of node.childNodes) {
        walkNode(child, state, inBlock)
      }
      return
    }
  }
}

function flushPending(state) {
  if (state.pending) {
    state.markdown += state.pending
    state.pending = ''
  }
}

function isBlock(tag) {
  const blocks = new Set(['p', 'div', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'blockquote', 'pre', 'tr', 'section', 'article', 'header', 'footer', 'nav', 'main', 'aside',
    'ul', 'ol', 'dl', 'table', 'figure', 'figcaption', 'hr'])
  return blocks.has(tag)
}

describe('readSelection', () => {
  describe('walkFragment', () => {
    it('converts text nodes to markdown', () => {
      const result = walkFragment(fragment(textNode('hello world')))
      expect(result.markdown).toBe('hello world')
      expect(result.images).toEqual([])
    })

    it('converts an image to a markdown image reference', () => {
      const result = walkFragment(fragment(img('https://example.com/photo.png', 'A photo')))
      expect(result.markdown).toBe('![A photo](@stash-image-0)')
      expect(result.images).toEqual([{ src: 'https://example.com/photo.png', alt: 'A photo' }])
    })

    it('places an image between paragraphs', () => {
      const result = walkFragment(fragment(
        element('p', textNode('before')),
        img('https://example.com/mid.png'),
        element('p', textNode('after')),
      ))
      expect(result.markdown).toBe('before\n\n![](@stash-image-0)\n\nafter')
      expect(result.images).toHaveLength(1)
    })

    it('handles an image-only selection', () => {
      const result = walkFragment(fragment(img('https://example.com/solo.png')))
      expect(result.markdown).toBe('![](@stash-image-0)')
      expect(result.images).toEqual([{ src: 'https://example.com/solo.png', alt: '' }])
    })

    it('numbers two images in selection order', () => {
      const result = walkFragment(fragment(
        img('https://example.com/first.png'),
        textNode(' '),
        img('https://example.com/second.png'),
      ))
      expect(result.markdown).toBe('![](@stash-image-0) ![](@stash-image-1)')
      expect(result.images).toEqual([
        { src: 'https://example.com/first.png', alt: '' },
        { src: 'https://example.com/second.png', alt: '' },
      ])
    })

    it('skips script and style elements', () => {
      const result = walkFragment(fragment(
        textNode('visible'),
        element('script', textNode('console.log(1)')),
        element('style', textNode('.x { color: red; }')),
        textNode(' more visible'),
      ))
      expect(result.markdown).toBe('visible more visible')
    })

    it('skips noscript elements', () => {
      const result = walkFragment(fragment(
        textNode('a'),
        element('noscript', textNode('hidden')),
        textNode('b'),
      ))
      expect(result.markdown).toBe('ab')
    })

    it('separates block elements with blank lines', () => {
      const result = walkFragment(fragment(
        element('p', textNode('first paragraph')),
        element('p', textNode('second paragraph')),
      ))
      expect(result.markdown).toBe('first paragraph\n\nsecond paragraph')
    })

    it('handles a heading followed by a paragraph', () => {
      const result = walkFragment(fragment(
        element('h2', textNode('Title')),
        element('p', textNode('Content here.')),
      ))
      expect(result.markdown).toBe('Title\n\nContent here.')
    })

    it('handles a br inside a paragraph', () => {
      const result = walkFragment(fragment(
        element('p', textNode('line1'), element('br'), textNode('line2')),
      ))
      expect(result.markdown).toBe('line1\nline2')
    })

    it('handles nested inline elements (flattens to text)', () => {
      const result = walkFragment(fragment(
        element('p',
          textNode('start '),
          element('strong', textNode('bold')),
          textNode(' and '),
          element('em', textNode('italic')),
        ),
      ))
      expect(result.markdown).toBe('start bold and italic')
    })

    it('returns empty string for an empty fragment', () => {
      const result = walkFragment(fragment())
      expect(result.markdown).toBe('')
      expect(result.images).toEqual([])
    })
  })
})