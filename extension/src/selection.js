// Injected into the page by executeScript. Must be entirely self-contained: no imports,
// no references to functions defined outside the function body. executeScript serializes
// only the function passed, not the module scope, so every helper must be nested inside.
export function readSelection() {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return null

  const range = selection.getRangeAt(0)
  if (range.collapsed) return null

  // -- All helpers defined inside so executeScript can serialize them. --

  function newState() {
    return { markdown: '', images: [], pending: '', lastWasImage: false }
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

  function walkNode(node, state, inBlock) {
    switch (node.nodeType) {
      case 3: // TEXT_NODE
        state.pending += node.textContent
        if (node.textContent.trim()) state.lastWasImage = false
        return

      case 1: { // ELEMENT_NODE
        const tag = (node.tagName || '').toLowerCase()

        if (tag === 'script' || tag === 'style' || tag === 'noscript') return

        if (tag === 'img' || tag === 'image') {
          flushPending(state)
          if (!inBlock && state.markdown.length > 0 && !state.markdown.endsWith('\n\n') && !state.lastWasImage) {
            state.markdown += '\n\n'
          }
          // SVG <image> uses href, HTML <img> uses currentSrc/src.
          let src = node.currentSrc || node.src
          if (!src && node.href && node.href.baseVal) src = node.href.baseVal
          if (!src && node.getAttribute) src = node.getAttribute('src') || node.getAttribute('href') || node.getAttribute('xlink:href')
          if (src) {
            try { src = new URL(src, document.baseURI).href } catch (_) {}
            const idx = state.images.length
            state.images.push({ src, alt: node.alt || '' })
            state.markdown += '![' + (node.alt || '') + '](@stash-image-' + idx + ')'
            state.lastWasImage = !inBlock
          }
          return
        }

        // <picture>: find the <img> inside and treat it as the image.
        if (tag === 'picture') {
          const innerImg = node.querySelector('img')
          if (innerImg) {
            walkNode(innerImg, state, inBlock)
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
          for (var i = 0; i < node.childNodes.length; i++) {
            walkNode(node.childNodes[i], state, true)
          }
          flushPending(state)
          return
        }

        for (var j = 0; j < node.childNodes.length; j++) {
          walkNode(node.childNodes[j], state, inBlock)
        }
        return
      }
    }
  }

  // Walk the cloned fragment.
  const state = newState()
  const fragment = range.cloneContents()
  for (var k = 0; k < fragment.childNodes.length; k++) {
    walkNode(fragment.childNodes[k], state, false)
  }
  if (state.pending) {
    state.markdown += state.pending
    state.pending = ''
  }

  const result = state.markdown.trim()
  if (!result && state.images.length === 0) return null
  return { markdown: result, images: state.images }
}