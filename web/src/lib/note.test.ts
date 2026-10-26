import { describe, expect, it } from 'vitest'
import { buildNoteUpload, readSegments } from './note'

function editor(html: string): HTMLDivElement {
  const element = document.createElement('div')
  element.innerHTML = html
  return element
}

function imageFile(name: string): File {
  return new File([name], name, { type: 'image/png' })
}

describe('note', () => {
  describe('readSegments', () => {
    it('reads text and images in order', () => {
      const segments = readSegments(editor('before<img data-key="a">after'))

      expect(segments).toEqual([
        { type: 'text', text: 'before' },
        { type: 'image', key: 'a' },
        { type: 'text', text: 'after' },
      ])
    })

    it('turns line breaks into newlines', () => {
      expect(readSegments(editor('one<br>two'))).toEqual([{ type: 'text', text: 'one\ntwo' }])
    })

    it('turns block elements into newlines', () => {
      expect(readSegments(editor('<div>one</div><div>two</div>'))).toEqual([
        { type: 'text', text: 'one\ntwo' },
      ])
    })

    it('drops empty text between images', () => {
      expect(readSegments(editor('<img data-key="a"><img data-key="b">'))).toEqual([
        { type: 'image', key: 'a' },
        { type: 'image', key: 'b' },
      ])
    })

    it('ignores images without a key', () => {
      expect(readSegments(editor('<img src="blob:missing">text'))).toEqual([
        { type: 'text', text: 'text' },
      ])
    })

    it('returns no segments for an empty editor', () => {
      expect(readSegments(editor(''))).toEqual([])
    })
  })

  describe('buildNoteUpload', () => {
    it('indexes images and keeps their order', () => {
      const first = imageFile('first.png')
      const second = imageFile('second.png')

      const upload = buildNoteUpload(
        [
          { type: 'text', text: 'before' },
          { type: 'image', key: 'a' },
          { type: 'image', key: 'b' },
          { type: 'text', text: 'after' },
        ],
        new Map([
          ['a', first],
          ['b', second],
        ]),
      )

      expect(upload.text).toBe('before\nafter')
      expect(upload.document).toEqual({
        blocks: [
          { type: 'text', text: 'before' },
          { type: 'image', index: 0 },
          { type: 'image', index: 1 },
          { type: 'text', text: 'after' },
        ],
      })
      expect(upload.images).toEqual([first, second])
    })

    it('reuses the index when the same image appears twice', () => {
      const file = imageFile('photo.png')

      const upload = buildNoteUpload(
        [
          { type: 'image', key: 'a' },
          { type: 'image', key: 'a' },
        ],
        new Map([['a', file]]),
      )

      expect(upload.document.blocks).toEqual([
        { type: 'image', index: 0 },
        { type: 'image', index: 0 },
      ])
      expect(upload.images).toEqual([file])
    })

    it('drops images that have no file', () => {
      const upload = buildNoteUpload([{ type: 'image', key: 'missing' }], new Map())

      expect(upload.document.blocks).toEqual([])
      expect(upload.images).toEqual([])
    })
  })
})
