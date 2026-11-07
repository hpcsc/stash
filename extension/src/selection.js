// Injected into the page by executeScript. Must be self-contained: no imports, no closure
// over module state. Exported only for unit tests; the injection uses the function value, not
// the module.
export function readSelection() {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return null

  const range = selection.getRangeAt(0)
  if (range.collapsed) return null

  return walkFragment(range.cloneContents())
}

// Used by walkFragment to track state across top-level nodes.
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
  if (!result && state.images.length === 0) return null
  return { markdown: result, images: state.images }
}

function walkNode(node, state, inBlock = false) {
  switch (node.nodeType) {
    case Node.TEXT_NODE:
      state.pending += node.textContent
      // Whitespace-only text between images does not count as content separation.
      if (node.textContent.trim()) state.lastWasImage = false
      return

    case Node.ELEMENT_NODE: {
      const tag = node.tagName?.toLowerCase()

      // Skip hidden content.
      if (tag === 'script' || tag === 'style' || tag === 'noscript') return

      // Images become markdown image references.
      if (tag === 'img') {
        flushPending(state)
        // When the image is not inside a block, separate it from preceding content,
        // but only when the preceding content was text (not another image).
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

      // Line breaks.
      if (tag === 'br') {
        flushPending(state)
        state.markdown += '\n'
        state.lastWasImage = false
        return
      }

      // Block-level elements get blank-line separation on entry.
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

      // Inline elements: just recurse into children.
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

// Async wrapper: try to read blob: images as data URLs. Callable when the browser supports
// async injected functions.
export async function readSelectionAsync() {
  const result = readSelection()
  if (!result) return null

  for (const img of result.images) {
    if (img.src.startsWith('blob:')) {
      try {
        const response = await fetch(img.src)
        if (response.ok) {
          const blob = await response.blob()
          img._blobBytes = Array.from(new Uint8Array(await blob.arrayBuffer()))
          img._blobType = blob.type
          img._blobName = img.alt || 'image'
        }
      } catch {
        // Fall back to the blob URL; the background will try to download.
      }
    }
  }

  return result
}